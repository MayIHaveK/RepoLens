package analysis

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/filter"
	"github.com/MayIHaveK/RepoLens/internal/model"
)

func (a *Analyzer) scanTree(ctx context.Context, repository, ref string, cfg config.Config, classifier filter.Classifier, agg *aggregate) ([]treeFile, error) {
	files := make([]treeFile, 0, 4096)
	err := a.git.Stream(ctx, repository, []string{"ls-tree", "-r", "-l", "-z", ref}, func(reader io.Reader) error {
		buffer := bufio.NewReaderSize(reader, 128*1024)
		for {
			record, err := buffer.ReadString(0)
			if err != nil && err != io.EOF {
				return err
			}
			if len(record) > 1 {
				file, ok := parseTreeRecord(strings.TrimSuffix(record, "\x00"))
				if ok {
					agg.files++
					classification := classifier.Classify(file.path)
					category := agg.categories[classification.Category]
					if category == nil {
						category = &model.CategorySummary{Category: classification.Category}
						agg.categories[classification.Category] = category
					}
					category.Files++
					if classification.Ignored || file.size > cfg.MaxFileSizeBytes {
						agg.skipped++
					} else {
						file.category = classification.Category
						agg.eligible++
						if len(files) < cfg.MaxOwnershipFiles {
							files = append(files, file)
						} else {
							agg.skipped++
						}
					}
				}
			}
			if err == io.EOF {
				break
			}
		}
		return nil
	})
	return files, err
}

func parseTreeRecord(record string) (treeFile, bool) {
	metadata, filePath, ok := strings.Cut(record, "\t")
	if !ok {
		return treeFile{}, false
	}
	parts := strings.Fields(metadata)
	if len(parts) != 4 || parts[1] != "blob" {
		return treeFile{}, false
	}
	size, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return treeFile{}, false
	}
	return treeFile{path: filePath, objectID: parts[2], size: size}, true
}

func (a *Analyzer) scanOwnership(ctx context.Context, repository, ref string, cfg config.Config, files []treeFile, agg *aggregate, progress ProgressFunc) {
	if len(files) == 0 {
		return
	}
	jobs := make(chan treeFile)
	results := make(chan blameResult, cfg.Parallelism)
	var workers sync.WaitGroup
	for i := 0; i < cfg.Parallelism; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for file := range jobs {
				ownership, err := a.blameFile(ctx, repository, ref, file.path, cfg)
				results <- blameResult{file: file, ownership: ownership, err: err}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, file := range files {
			select {
			case jobs <- file:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()

	var completed int64
	var ownershipErrors int64
	for result := range results {
		completed++
		if result.err != nil {
			agg.skipped++
			ownershipErrors++
		} else {
			agg.analyzed++
			for _, owned := range result.ownership {
				contributor := ensureContributor(agg, owned.author)
				contributor.Contributor.OwnedLines += owned.lines
				agg.ownedLines += owned.lines
			}
		}
		if completed%25 == 0 || completed == int64(len(files)) {
			percent := 48 + float64(completed)/float64(len(files))*40
			progress(model.Progress{Phase: "ownership", Message: fmt.Sprintf("已分析 %d / %d 个文件", completed, len(files)), Current: completed, Total: int64(len(files)), Percent: percent})
		}
	}
	if ownershipErrors > 0 {
		agg.warnings = append(agg.warnings, fmt.Sprintf("有 %d 个文件无法完成代码归属分析，已从归属指标中排除。", ownershipErrors))
	}
}
