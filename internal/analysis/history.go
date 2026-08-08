package analysis

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/filter"
	"github.com/MayIHaveK/RepoLens/internal/gitutil"
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
	format := commitMarker + "%H%x09%aN%x09%aE%x09%at%x09%s%x09%(trailers:key=Co-authored-by,valueonly,separator=%x1c)"
	args := []string{"log", ref, "--date-order", "--numstat", "--format=" + format}
	if !cfg.IncludeMerges {
		args = append(args, "--no-merges")
	}
	if cfg.DetectRenames {
		args = append(args, "--find-renames=50%")
	} else {
		args = append(args, "--no-renames")
	}
	if cfg.Since != "" {
		args = append(args, "--since="+cfg.Since)
	}
	if cfg.Until != "" {
		args = append(args, "--until="+cfg.Until)
	}

	var current *commitRecord
	commitCount := int64(0)
	return a.git.Stream(ctx, repository, args, func(reader io.Reader) error {
		err := gitutil.ScanLines(reader, func(line string) error {
			if strings.HasPrefix(line, commitMarker) {
				if current != nil {
					a.commit(current, cfg, agg)
					commitCount++
					if commitCount%250 == 0 {
						progress(model.Progress{Phase: "history", Message: fmt.Sprintf("已读取 %d 个提交", commitCount), Current: commitCount, Percent: 8 + minFloat(float64(commitCount)/50000*32, 32)})
					}
				}
				record, err := parseCommitHeader(strings.TrimPrefix(line, commitMarker), cfg, a)
				if err != nil {
					return err
				}
				current = record
				return nil
			}
			if current == nil || strings.TrimSpace(line) == "" {
				return nil
			}
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) != 3 {
				return nil
			}
			additions, okA := parseInt64(parts[0])
			deletions, okD := parseInt64(parts[1])
			if !okA || !okD {
				return nil
			}
			filePath := cleanRenamePath(parts[2])
			classification := classifier.Classify(filePath)
			if classification.Ignored {
				return nil
			}
			current.files = append(current.files, fileChange{
				category: classification.Category, directory: directoryName(filePath),
				additions: additions, deletions: deletions,
			})
			return nil
		})
		if err != nil {
			return err
		}
		if current != nil {
			a.commit(current, cfg, agg)
			commitCount++
		}
		return nil
	})
}

func parseCommitHeader(value string, cfg config.Config, analyzer *Analyzer) (*commitRecord, error) {
	parts := strings.SplitN(value, "\t", 6)
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

		commitAdd := splitCredit(totalAdd, len(authors), authorIndex)
		commitDelete := splitCredit(totalDelete, len(authors), authorIndex)
		contributor.Contributor.RecentCommits = append(contributor.Contributor.RecentCommits, model.CommitSummary{
			SHA: record.sha, Message: record.subject, Timestamp: record.timestamp.Format(time.RFC3339),
			Additions: commitAdd, Deletions: commitDelete,
		})
		for _, change := range record.files {
			additions := splitCredit(change.additions, len(authors), authorIndex)
			deletions := splitCredit(change.deletions, len(authors), authorIndex)
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
		bucketChange := bucket.Contributors[author.ID]
		bucketChange.Additions += commitAdd
		bucketChange.Deletions += commitDelete
		bucket.Contributors[author.ID] = bucketChange
	}

	agg.commits++
	agg.additions += totalAdd
	agg.deletions += totalDelete
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
