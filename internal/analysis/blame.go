package analysis

import (
	"context"
	"io"
	"strings"

	"github.com/repolens/repolens/internal/config"
	"github.com/repolens/repolens/internal/gitutil"
)

type ownedLines struct {
	author identity
	lines  int64
}

type blameResult struct {
	file      treeFile
	ownership []ownedLines
	err       error
}

func (a *Analyzer) blameFile(ctx context.Context, repository, ref, filePath string, cfg config.Config) ([]ownedLines, error) {
	counts := map[string]*ownedLines{}
	var currentName, currentEmail string
	err := a.git.Stream(ctx, repository, []string{"blame", "--line-porcelain", ref, "--", filePath}, func(reader io.Reader) error {
		return gitutil.ScanLines(reader, func(line string) error {
			switch {
			case strings.HasPrefix(line, "author "):
				currentName = strings.TrimPrefix(line, "author ")
			case strings.HasPrefix(line, "author-mail "):
				currentEmail = strings.Trim(strings.TrimPrefix(line, "author-mail "), "<>")
			case strings.HasPrefix(line, "\t"):
				author := a.identity(currentName, currentEmail, cfg)
				if !cfg.IncludeBots && isBot(author, cfg) {
					return nil
				}
				entry := counts[author.ID]
				if entry == nil {
					entry = &ownedLines{author: author}
					counts[author.ID] = entry
				}
				entry.lines++
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	result := make([]ownedLines, 0, len(counts))
	for _, entry := range counts {
		result = append(result, *entry)
	}
	return result, nil
}
