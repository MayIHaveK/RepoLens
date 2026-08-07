package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/repolens/repolens/internal/config"
	"github.com/repolens/repolens/internal/filter"
	"github.com/repolens/repolens/internal/gitutil"
	"github.com/repolens/repolens/internal/model"
)

type ProgressFunc func(model.Progress)

type Analyzer struct {
	git gitutil.Client
}

func New() *Analyzer {
	return &Analyzer{git: gitutil.New()}
}

type contributorAggregate struct {
	Contributor model.Contributor
	Email       string
	Workload    float64
	Directories map[string]model.Change
}

type aggregate struct {
	contributors map[string]*contributorAggregate
	timeline     map[string]*model.TimelineBucket
	categories   map[model.Category]*model.CategorySummary
	warnings     []string
	commits      int64
	files        int64
	additions    int64
	deletions    int64
	ownedLines   int64
	analyzed     int64
	skipped      int64
	eligible     int64
	mu           sync.Mutex
}

type identity struct {
	ID    string
	Name  string
	Email string
}

type fileChange struct {
	category  model.Category
	directory string
	additions int64
	deletions int64
}

type commitRecord struct {
	sha       string
	name      string
	email     string
	timestamp time.Time
	subject   string
	coauthors []identity
	files     []fileChange
}

type treeFile struct {
	path     string
	objectID string
	size     int64
	category model.Category
}

func (a *Analyzer) CacheKey(ctx context.Context, repository string, cfg config.Config) (string, error) {
	cfg.Normalize()
	root, sha, _, _, err := a.git.ResolveRepository(ctx, repository, cfg.Ref)
	if err != nil {
		return "", err
	}
	return shortHash(root + "\x00" + sha + "\x00" + cfg.Fingerprint()), nil
}

func (a *Analyzer) Analyze(ctx context.Context, repository string, cfg config.Config, progress ProgressFunc) (*model.Analysis, error) {
	started := time.Now()
	cfg.Normalize()
	applyProfile(&cfg)
	if progress == nil {
		progress = func(model.Progress) {}
	}

	progress(model.Progress{Phase: "repository", Message: "正在验证仓库和目标版本", Percent: 2})
	root, sha, name, headDate, err := a.git.ResolveRepository(ctx, repository, cfg.Ref)
	if err != nil {
		return nil, err
	}

	agg := newAggregate()
	classifier := filter.NewClassifier(cfg)
	progress(model.Progress{Phase: "history", Message: "正在流式读取提交历史", Percent: 8})
	if err := a.scanHistory(ctx, root, sha, cfg, classifier, agg, progress); err != nil {
		return nil, fmt.Errorf("scan history: %w", err)
	}

	progress(model.Progress{Phase: "tree", Message: "正在索引当前代码树", Percent: 42})
	files, err := a.scanTree(ctx, root, sha, cfg, classifier, agg)
	if err != nil {
		return nil, fmt.Errorf("scan tree: %w", err)
	}

	if cfg.EnableOwnership {
		progress(model.Progress{Phase: "ownership", Message: "正在并行计算当前代码归属", Total: int64(len(files)), Percent: 48})
		a.scanOwnership(ctx, root, sha, cfg, files, agg, progress)
	} else {
		agg.warnings = append(agg.warnings, "当前性能配置未启用逐行代码归属分析。")
	}

	progress(model.Progress{Phase: "scoring", Message: "正在计算多维贡献占比", Percent: 91})
	contributors := finalizeContributors(agg, cfg)
	timeline := finalizeTimeline(agg)
	categories := finalizeCategories(agg)

	coverage := 100.0
	if agg.eligible > 0 {
		coverage = float64(agg.analyzed) / float64(agg.eligible) * 100
	}
	analysisID := shortHash(root + "\x00" + sha + "\x00" + cfg.Fingerprint())
	result := &model.Analysis{
		SchemaVersion: model.SchemaVersion,
		ID:            analysisID,
		Repository: model.Repository{
			Name:        name,
			Ref:         cfg.Ref,
			CommitSHA:   sha,
			HeadDate:    headDate,
			CommitCount: agg.commits,
			FileCount:   agg.files,
		},
		GeneratedAt: time.Now().UTC(),
		DurationMS:  time.Since(started).Milliseconds(),
		Mode:        analysisMode(cfg),
		Config:      cfg.Snapshot(),
		Summary: model.Summary{
			Contributors:    len(contributors),
			Commits:         agg.commits,
			Files:           agg.files,
			Additions:       agg.additions,
			Deletions:       agg.deletions,
			OwnedLines:      agg.ownedLines,
			AnalyzedFiles:   agg.analyzed,
			SkippedFiles:    agg.skipped,
			CoveragePercent: round(coverage, 1),
		},
		Contributors: contributors,
		Timeline:     timeline,
		Categories:   categories,
		Warnings:     agg.warnings,
	}
	progress(model.Progress{Phase: "complete", Message: "分析完成", Current: 1, Total: 1, Percent: 100})
	return result, nil
}

func newAggregate() *aggregate {
	return &aggregate{
		contributors: map[string]*contributorAggregate{},
		timeline:     map[string]*model.TimelineBucket{},
		categories:   map[model.Category]*model.CategorySummary{},
	}
}

func applyProfile(cfg *config.Config) {
	switch strings.ToLower(cfg.Profile) {
	case "fast":
		cfg.EnableOwnership = false
		cfg.EnableRetention = false
		cfg.DetectRenames = false
	case "thorough":
		cfg.MaxOwnershipFiles = math.MaxInt
		if cfg.Parallelism < 2 {
			cfg.Parallelism = 2
		}
	default:
		cfg.Profile = "balanced"
	}
}

func analysisMode(cfg config.Config) string {
	if cfg.EnableCollaboration {
		return "git-only"
	}
	return "git-only"
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:10])
}

func round(value float64, places int) float64 {
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}

func (a *Analyzer) identity(name, email string, cfg config.Config) identity {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.Trim(strings.TrimSpace(email), "<>"))
	canonical := email
	if alias, ok := cfg.Aliases[email]; ok {
		canonical = strings.ToLower(strings.TrimSpace(alias))
	} else if alias, ok := cfg.Aliases[strings.ToLower(name)]; ok {
		canonical = strings.ToLower(strings.TrimSpace(alias))
	}
	if canonical == "" {
		canonical = strings.ToLower(name)
	}
	if name == "" {
		name = "未知贡献者"
	}
	return identity{ID: shortHash(canonical), Name: name, Email: email}
}

func isBot(author identity, cfg config.Config) bool {
	value := strings.ToLower(author.Name + " " + author.Email)
	for _, pattern := range cfg.BotPatterns {
		if strings.Contains(value, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func directoryName(filePath string) string {
	clean := strings.Trim(strings.ReplaceAll(filePath, "\\", "/"), "/")
	if index := strings.IndexByte(clean, '/'); index >= 0 {
		return clean[:index]
	}
	return "根目录"
}

func parseInt64(value string) (int64, bool) {
	if value == "-" {
		return 0, false
	}
	number, err := strconv.ParseInt(value, 10, 64)
	return number, err == nil
}

func cleanRenamePath(value string) string {
	if index := strings.LastIndex(value, " => "); index >= 0 {
		value = value[index+4:]
		value = strings.ReplaceAll(value, "{", "")
		value = strings.ReplaceAll(value, "}", "")
	}
	return path.Clean(value)
}
