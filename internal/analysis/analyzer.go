package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/filter"
	"github.com/MayIHaveK/RepoLens/internal/gitutil"
	"github.com/MayIHaveK/RepoLens/internal/model"
)

type ProgressFunc func(model.Progress)

type Analyzer struct {
	git     gitutil.Client
	Version string
}

func New() *Analyzer {
	return &Analyzer{git: gitutil.New(), Version: "0.2.0-dev"}
}

type contributorAggregate struct {
	Contributor model.Contributor
	Email       string
	Workload    float64
	Directories map[string]model.Change
}

type aggregate struct {
	contributors   map[string]*contributorAggregate
	timeline       map[string]*model.TimelineBucket
	categories     map[model.Category]*model.CategorySummary
	warnings       []string
	commits        int64
	files          int64
	additions      int64
	deletions      int64
	ownedLines     int64
	analyzed       int64
	skipped        int64
	eligible       int64
	project        *model.Project
	fileChanges    map[string]model.FileMetric
	historySkipped map[string]int64
}

type identity struct {
	ID    string
	Name  string
	Email string
}

type fileChange struct {
	path      string
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
	mode     string
}

func (a *Analyzer) CacheKey(ctx context.Context, repository string, cfg config.Config) (string, error) {
	var err error
	cfg, err = cfg.Prepared()
	if err != nil {
		return "", err
	}
	root, sha, _, _, err := a.git.ResolveRepository(ctx, repository, cfg.Ref)
	if err != nil {
		return "", err
	}
	return a.cacheID(ctx, root, sha, cfg)
}

func (a *Analyzer) Analyze(ctx context.Context, repository string, cfg config.Config, progress ProgressFunc) (*model.Analysis, error) {
	started := time.Now()
	requestedRetention := cfg.EnableRetention
	var err error
	cfg, err = cfg.Prepared()
	if err != nil {
		return nil, err
	}
	if progress == nil {
		progress = func(model.Progress) {}
	}

	progress(model.Progress{Phase: "repository", Message: "正在验证仓库和目标版本", Percent: 2})
	root, sha, name, headDate, err := a.git.ResolveRepository(ctx, repository, cfg.Ref)
	if err != nil {
		return nil, err
	}
	analysisID, err := a.cacheID(ctx, root, sha, cfg)
	if err != nil {
		return nil, err
	}

	agg := newAggregate()
	shallow, err := a.git.Output(ctx, root, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(shallow)) == "true" {
		agg.warnings = append(agg.warnings, "当前为浅克隆，历史工作量与近似留存只反映本机已有提交，不能代表完整历史。")
	}
	if requestedRetention && (cfg.Since != "" || cfg.Until != "") {
		agg.warnings = append(agg.warnings, "时间筛选只作用于历史工作量；当前文件树仍取目标版本，留存指标已停用以避免混用范围。")
	}
	if cfg.EnableCollaboration {
		agg.warnings = append(agg.warnings, "当前未连接协作平台，协作维度不参与综合评分。")
	}
	classifier := filter.NewClassifier(cfg)
	progress(model.Progress{Phase: "tree", Message: "正在索引目标版本文件树", Percent: 4})
	files, err := a.scanTree(ctx, root, sha, cfg, classifier, agg)
	if err != nil {
		return nil, fmt.Errorf("scan tree: %w", err)
	}
	progress(model.Progress{Phase: "history", Message: "正在流式读取提交历史", Percent: 8})
	if err := a.scanHistory(ctx, root, sha, cfg, classifier, agg, progress); err != nil {
		return nil, fmt.Errorf("scan history: %w", err)
	}

	progress(model.Progress{Phase: "inventory", Message: "正在统计代码、配置和文件关系", Percent: 44})
	files, err = a.scanInventory(ctx, root, files, cfg, agg)
	if err != nil {
		return nil, fmt.Errorf("scan inventory: %w", err)
	}

	if cfg.EnableOwnership {
		progress(model.Progress{Phase: "ownership", Message: "正在并行计算当前代码归属", Total: int64(len(files)), Percent: 48})
		a.scanOwnership(ctx, root, sha, cfg, files, agg, progress)
	} else {
		agg.warnings = append(agg.warnings, "当前性能配置未启用逐行代码归属分析。")
	}

	progress(model.Progress{Phase: "scoring", Message: "正在计算多维贡献占比", Percent: 91})
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	contributors := finalizeContributors(agg, cfg)
	timeline := finalizeTimeline(agg)
	categories := finalizeCategories(agg)

	coverage := 0.0
	if agg.eligible > 0 {
		coverage = float64(agg.analyzed) / float64(agg.eligible) * 100
	}
	finalID, err := a.cacheID(ctx, root, sha, cfg)
	if err != nil {
		return nil, err
	}
	if finalID != analysisID {
		return nil, fmt.Errorf("分析期间 Git 配置或身份映射发生变化，请重新分析")
	}
	snapshot := cfg.Snapshot()
	snapshot.EffectiveWeights = effectiveWeights(agg, cfg)
	result := &model.Analysis{
		ProgramVersion:   a.Version,
		AlgorithmVersion: model.AlgorithmVersion,
		Project:          agg.project,
		SchemaVersion:    model.SchemaVersion,
		ID:               analysisID,
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
		Config:      snapshot,
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
		contributors:   map[string]*contributorAggregate{},
		timeline:       map[string]*model.TimelineBucket{},
		categories:     map[model.Category]*model.CategorySummary{},
		fileChanges:    map[string]model.FileMetric{},
		historySkipped: map[string]int64{},
		project:        &model.Project{Scope: "目标提交中的受支持文本文件，不含工作区未提交内容", Skipped: map[string]int64{}, Files: []model.FileMetric{}, Languages: []model.LanguageSummary{}, Directories: []model.DirectorySummary{}, Dependencies: []model.Dependency{}, Configurations: []model.ConfigurationInfo{}, Cycles: [][]string{}},
	}
}

func (a *Analyzer) cacheID(ctx context.Context, root, sha string, cfg config.Config) (string, error) {
	inputs, err := a.git.Output(ctx, root, "config", "--null", "--list")
	if err != nil {
		return "", err
	}
	mappings := []string{filepath.Join(root, ".mailmap")}
	for _, entry := range strings.Split(string(inputs), "\x00") {
		key, value, ok := strings.Cut(entry, "\n")
		if ok && key == "mailmap.blob" && value != "" {
			blob, err := a.git.Output(ctx, root, "rev-parse", "--verify", "--end-of-options", value+"^{blob}")
			if err != nil {
				return "", err
			}
			inputs = append(inputs, blob...)
		}
		if ok && key == "mailmap.file" && value != "" {
			if strings.HasPrefix(value, "~/") || strings.HasPrefix(value, "~\\") {
				userHome, err := os.UserHomeDir()
				if err != nil {
					return "", err
				}
				value = filepath.Join(userHome, value[2:])
			}
			if !filepath.IsAbs(value) {
				value = filepath.Join(root, value)
			}
			mappings = append(mappings, value)
		}
	}
	for _, file := range mappings {
		digest, err := fingerprintFile(ctx, file)
		if err != nil {
			return "", err
		}
		inputs = append(inputs, []byte(file+"\x00"+digest)...)
	}
	// Deepening a shallow clone or changing replacement refs can change history
	// without moving HEAD, so these inputs also invalidate cached analyses.
	for _, name := range []string{"shallow", "info/grafts"} {
		location, err := a.git.Output(ctx, root, "rev-parse", "--git-path", name)
		if err != nil {
			return "", err
		}
		file := strings.TrimSpace(string(location))
		if !filepath.IsAbs(file) {
			file = filepath.Join(root, file)
		}
		digest, err := fingerprintFile(ctx, file)
		if err != nil {
			return "", err
		}
		inputs = append(inputs, []byte(digest)...)
	}
	refs, err := a.git.Output(ctx, root, "for-each-ref", "--format=%(refname) %(objectname)", "refs/replace")
	if err != nil {
		return "", err
	}
	inputs = append(inputs, refs...)
	gitVersion, err := a.git.Output(ctx, root, "--version")
	if err != nil {
		return "", err
	}
	return shortHash(root + "\x00" + sha + "\x00" + cfg.Fingerprint() + "\x00" + model.AlgorithmVersion + "\x00" + a.Version + "\x00" + string(inputs) + string(gitVersion)), nil
}

func fingerprintFile(ctx context.Context, filename string) (string, error) {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return "missing", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("Git 分析输入必须是普通文件")
	}
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, contextReader{ctx, file}); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(buffer []byte) (int, error) {
	if r.ctx.Err() != nil {
		return 0, r.ctx.Err()
	}
	return r.reader.Read(buffer)
}

func analysisMode(_ config.Config) string { return "git-only" }

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
