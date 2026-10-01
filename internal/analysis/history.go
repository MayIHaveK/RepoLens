package analysis

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/filter"
	"github.com/MayIHaveK/RepoLens/internal/model"
)

const commitMarker = "@@REPOLENS@@"

func (a *Analyzer) scanHistory(
	ctx context.Context,
	repository, ref string,
	cfg config.Config,
	classifier filter.Classifier,
	agg *aggregate,
	progress ProgressFunc,
) error {
	format := "%x00" + commitMarker + "%H%x00%aN%x00%aE%x00%at%x00%s%x00%(trailers:key=Co-authored-by,valueonly,separator=%x1c)%x00"
	args := []string{"log", ref, "--date-order", "--raw", "--numstat", "-z", "--no-abbrev", "--no-ext-diff", "--no-textconv", "--format=" + format}
	if !cfg.IncludeMerges {
		args = append(args, "--no-merges")
	} else {
		args = append(args, "--diff-merges=first-parent")
	}
	if cfg.DetectRenames {
		args = append(args, "--find-renames=50%")
	} else {
		args = append(args, "--no-renames")
	}
	batch, err := a.git.OpenBatch(ctx, repository, true)
	if err != nil {
		return err
	}
	defer batch.Close()
	sizes := map[string]int64{}
	sizeOf := func(id string) (int64, error) {
		if strings.Trim(id, "0") == "" {
			return 0, nil
		}
		if size, ok := sizes[id]; ok {
			return size, nil
		}
		_, size, err := batch.Read(id, 0)
		if err == nil {
			if len(sizes) >= 16384 {
				clear(sizes)
			}
			sizes[id] = size
		}
		return size, err
	}

	var current *commitRecord
	commitCount := int64(0)
	return a.git.Stream(ctx, repository, args, func(reader io.Reader) error {
		buffer := bufio.NewReaderSize(reader, 128*1024)
		objects := map[string][4]string{}
		readToken := func() (string, error) {
			value, err := buffer.ReadString(0)
			return strings.TrimSuffix(value, "\x00"), err
		}
		for {
			line, err := readToken()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Git places record separators before metadata; never trim path tokens.
			line = strings.TrimLeft(line, "\r\n")
			if strings.HasPrefix(line, commitMarker) {
				if current != nil {
					a.commit(current, cfg, agg)
					commitCount++
					if commitCount%250 == 0 {
						progress(model.Progress{Phase: "history", Message: fmt.Sprintf("已读取 %d 个提交", commitCount), Current: commitCount, Percent: 8 + minFloat(float64(commitCount)/50000*32, 32)})
					}
				}
				fields := []string{strings.TrimPrefix(line, commitMarker)}
				for i := 0; i < 5; i++ {
					field, err := readToken()
					if err != nil {
						return err
					}
					fields = append(fields, field)
				}
				record, err := parseCommitFields(fields, cfg, a)
				if err != nil {
					return err
				}
				current = record
				objects = map[string][4]string{}
				continue
			}
			if current == nil || strings.TrimSpace(line) == "" {
				continue
			}
			if strings.HasPrefix(line, ":") {
				fields := strings.Fields(strings.TrimPrefix(line, ":"))
				if len(fields) != 5 {
					return fmt.Errorf("invalid raw Git record")
				}
				filePath, err := readToken()
				if err != nil {
					return err
				}
				if strings.HasPrefix(fields[4], "R") || strings.HasPrefix(fields[4], "C") {
					filePath, err = readToken()
					if err != nil {
						return err
					}
				}
				objects[filePath] = [4]string{fields[2], fields[3], fields[0], fields[1]}
				continue
			}
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) != 3 {
				return fmt.Errorf("invalid Git numstat record")
			}
			filePath := parts[2]
			if filePath == "" {
				if _, err := readToken(); err != nil {
					return err
				}
				filePath, err = readToken()
				if err != nil {
					return err
				}
			}
			if !inDateRange(current.timestamp, cfg) {
				continue
			}
			additions, okA := parseInt64(parts[0])
			deletions, okD := parseInt64(parts[1])
			if !okA || !okD {
				agg.historySkipped["binary"]++
				continue
			}
			classification := classifier.Classify(filePath)
			if classification.Ignored {
				agg.historySkipped[classification.Reason]++
				continue
			}
			object, ok := objects[filePath]
			if !ok {
				return fmt.Errorf("missing raw Git record")
			}
			if object[2] == "120000" || object[3] == "120000" || object[2] == "160000" || object[3] == "160000" {
				agg.historySkipped["non-regular"]++
				continue
			}
			oldSize, err := sizeOf(object[0])
			if err != nil {
				return err
			}
			newSize, err := sizeOf(object[1])
			if err != nil {
				return err
			}
			if oldSize > cfg.MaxFileSizeBytes || newSize > cfg.MaxFileSizeBytes {
				agg.historySkipped["oversized"]++
				continue
			}
			current.files = append(current.files, fileChange{
				path:     filePath,
				category: classification.Category, directory: directoryName(filePath),
				additions: additions, deletions: deletions,
			})
		}
		if current != nil {
			a.commit(current, cfg, agg)
			commitCount++
		}
		return nil
	})
}

func parseCommitFields(parts []string, cfg config.Config, analyzer *Analyzer) (*commitRecord, error) {
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid Git log record")
	}
	timestamp, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid commit timestamp: %w", err)
	}
	record := &commitRecord{
		sha: parts[0], name: parts[1], email: parts[2],
		timestamp: time.Unix(timestamp, 0).UTC(), subject: parts[4],
	}
	if len(parts) == 6 && parts[5] != "" {
		for _, raw := range strings.Split(parts[5], "\x1c") {
			name, email := parseCoauthor(raw)
			if email == "" {
				continue
			}
			record.coauthors = append(record.coauthors, analyzer.identity(name, email, cfg))
		}
	}
	return record, nil
}

func parseCoauthor(value string) (string, string) {
	value = strings.TrimSpace(value)
	start := strings.LastIndex(value, "<")
	end := strings.LastIndex(value, ">")
	if start < 0 || end <= start {
		return "", ""
	}
	return strings.TrimSpace(value[:start]), strings.TrimSpace(value[start+1 : end])
}

func (a *Analyzer) commit(record *commitRecord, cfg config.Config, agg *aggregate) {
	if !inDateRange(record.timestamp, cfg) {
		return
	}
	authors := []identity{a.identity(record.name, record.email, cfg)}
	seen := map[string]bool{authors[0].ID: true}
	for _, author := range record.coauthors {
		if !seen[author.ID] {
			authors = append(authors, author)
			seen[author.ID] = true
		}
	}
	if !cfg.IncludeBots {
		filtered := authors[:0]
		for _, author := range authors {
			if !isBot(author, cfg) {
				filtered = append(filtered, author)
			}
		}
		authors = filtered
	}
	if len(authors) == 0 {
		return
	}

	var totalAdd, totalDelete int64
	for _, change := range record.files {
		totalAdd += change.additions
		totalDelete += change.deletions
		if metric, current := agg.fileChanges[change.path]; current {
			metric.Commits++
			metric.Additions += change.additions
			metric.Deletions += change.deletions
			date := record.timestamp.Format(time.RFC3339)
			if date > metric.LastChangedAt {
				metric.LastChangedAt = date
			}
			agg.fileChanges[change.path] = metric
		}
	}
	period := record.timestamp.Format("2006-01")
	bucket := agg.timeline[period]
	if bucket == nil {
		bucket = &model.TimelineBucket{Period: period, Contributors: map[string]model.Change{}}
		agg.timeline[period] = bucket
	}
	bucket.Commits++
	bucket.Additions += totalAdd
	bucket.Deletions += totalDelete

	for authorIndex, author := range authors {
		contributor := ensureContributor(agg, author)
		contributor.Contributor.Commits++
		if contributor.Contributor.FirstCommitAt == "" || record.timestamp.Format(time.RFC3339) < contributor.Contributor.FirstCommitAt {
			contributor.Contributor.FirstCommitAt = record.timestamp.Format(time.RFC3339)
		}
		if record.timestamp.Format(time.RFC3339) > contributor.Contributor.LastCommitAt {
			contributor.Contributor.LastCommitAt = record.timestamp.Format(time.RFC3339)
		}

		var commitAdd, commitDelete, addOffset, deleteOffset int64
		for _, change := range record.files {
			additions := splitCreditOffset(change.additions, len(authors), authorIndex, addOffset)
			deletions := splitCreditOffset(change.deletions, len(authors), authorIndex, deleteOffset)
			addOffset += change.additions
			deleteOffset += change.deletions
			commitAdd += additions
			commitDelete += deletions
			category := contributor.Contributor.Categories[change.category]
			category.Additions += additions
			category.Deletions += deletions
			contributor.Contributor.Categories[change.category] = category
			contributor.Contributor.Additions += additions
			contributor.Contributor.HistoricalAdded += additions
			contributor.Contributor.Deletions += deletions
			weight := cfg.CategoryWeights[change.category]
			contributor.Workload += (float64(additions) + float64(deletions)*0.5) * weight
			directory := contributor.Directories[change.directory]
			directory.Additions += additions
			directory.Deletions += deletions
			contributor.Directories[change.directory] = directory
		}
		contributor.Contributor.RecentCommits = append(contributor.Contributor.RecentCommits, model.CommitSummary{
			SHA: record.sha, Message: record.subject, Timestamp: record.timestamp.Format(time.RFC3339), Additions: commitAdd, Deletions: commitDelete,
		})
		sort.Slice(contributor.Contributor.RecentCommits, func(i, j int) bool {
			left, right := contributor.Contributor.RecentCommits[i], contributor.Contributor.RecentCommits[j]
			if left.Timestamp == right.Timestamp {
				return left.SHA < right.SHA
			}
			return left.Timestamp > right.Timestamp
		})
		if len(contributor.Contributor.RecentCommits) > 8 {
			contributor.Contributor.RecentCommits = contributor.Contributor.RecentCommits[:8]
		}
		bucketChange := bucket.Contributors[author.ID]
		bucketChange.Additions += commitAdd
		bucketChange.Deletions += commitDelete
		bucket.Contributors[author.ID] = bucketChange
	}

	agg.commits++
	agg.additions += totalAdd
	agg.deletions += totalDelete
}

func splitCreditOffset(total int64, count, index int, offset int64) int64 {
	return splitCredit(total+offset, count, index) - splitCredit(offset, count, index)
}

func inDateRange(timestamp time.Time, cfg config.Config) bool {
	if cfg.Since != "" {
		since, _ := config.ParseDate(cfg.Since)
		if timestamp.Before(since) {
			return false
		}
	}
	if cfg.Until != "" {
		until, _ := config.ParseDate(cfg.Until)
		if len(cfg.Until) == 10 {
			if !timestamp.Before(until.AddDate(0, 0, 1)) {
				return false
			}
		} else if timestamp.After(until) {
			return false
		}
	}
	return true
}

func ensureContributor(agg *aggregate, author identity) *contributorAggregate {
	contributor := agg.contributors[author.ID]
	if contributor == nil {
		contributor = &contributorAggregate{
			Contributor: model.Contributor{ID: author.ID, Name: author.Name, Categories: map[model.Category]model.Change{}},
			Email:       author.Email, Directories: map[string]model.Change{},
		}
		agg.contributors[author.ID] = contributor
	}
	return contributor
}

func splitCredit(total int64, count, index int) int64 {
	if count <= 1 {
		return total
	}
	base := total / int64(count)
	if int64(index) < total%int64(count) {
		return base + 1
	}
	return base
}

func finalizeTimeline(agg *aggregate) []model.TimelineBucket {
	periods := make([]string, 0, len(agg.timeline))
	for period := range agg.timeline {
		periods = append(periods, period)
	}
	sort.Strings(periods)
	result := make([]model.TimelineBucket, 0, len(periods))
	for _, period := range periods {
		result = append(result, *agg.timeline[period])
	}
	return result
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
