package analysis

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"sort"
	"unicode/utf8"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/model"
)

func (a *Analyzer) scanInventory(ctx context.Context, repository string, files []treeFile, cfg config.Config, agg *aggregate) ([]treeFile, error) {
	project := agg.project
	project.TotalFiles = agg.files
	project.HistorySkipped = agg.historySkipped
	project.Notes = []string{
		"行数仅统计目标提交中通过过滤与大小限制的 UTF-8 文本文件；尾部换行不额外计一行。",
		"混合代码与注释行归入代码；注释按词法规则估算。文档和未支持格式的非空行计为内容行。",
		"有效源代码行只汇总 source 与 test 类别，不含文档、配置、资源和其他文件。",
		"文件变更次数与新增/删除采用历史日期范围；文件内容与依赖采用目标提交。已删除文件不在当前文件表中。",
		"Go 导入使用语法解析；JS/TS、CSS 与 HTML 引用使用静态字符串规则。动态导入、路径别名和运行时调用关系不作推断。",
		"配置分析只记录键名、结构和引用，不保存配置值或执行脚本。JSON 支持语法检查，YAML/TOML 等只检查顶层结构。",
	}
	batch, err := a.git.OpenBatch(ctx, repository, false)
	if err != nil {
		return nil, err
	}
	defer batch.Close()
	indexes := make(map[string]int, len(project.Files))
	for index, metric := range project.Files {
		indexes[metric.Path] = index
	}
	languages := map[string]*model.LanguageSummary{}
	directories := map[string]*model.DirectorySummary{}
	var eligible []treeFile
	var references []model.Dependency
	var goModule string
	for index, file := range files {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		metric := &project.Files[indexes[file.path]]
		*metric = agg.fileChanges[file.path]
		if index >= cfg.MaxInventoryFiles {
			metric.Status = "skipped"
			metric.Reason = "inventory-limit"
			project.Skipped[metric.Reason]++
			agg.skipped++
			continue
		}
		data, _, err := batch.Read(file.objectID, cfg.MaxFileSizeBytes)
		if err != nil {
			return nil, err
		}
		if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
			metric.Status = "skipped"
			metric.Reason = "binary-or-encoding"
			project.Skipped[metric.Reason]++
			agg.skipped++
			agg.eligible--
			continue
		}
		metric.Status = "analyzed"
		metric.Lines = countLines(data, metric.Language)
		project.TextFiles++
		addLines(&project.Totals, metric.Lines)
		if file.category == model.CategorySource || file.category == model.CategoryTest {
			project.SourceCodeLines += metric.Lines.Code
		}
		category := agg.categories[file.category]
		if category == nil {
			category = &model.CategorySummary{Category: file.category}
			agg.categories[file.category] = category
		}
		category.Files++
		language := languages[metric.Language]
		if language == nil {
			language = &model.LanguageSummary{Language: metric.Language}
			languages[metric.Language] = language
		}
		language.Files++
		addLines(&language.Lines, metric.Lines)
		for directory := path.Dir(file.path); ; directory = path.Dir(directory) {
			item := directories[directory]
			if item == nil {
				item = &model.DirectorySummary{Path: directory}
				directories[directory] = item
			}
			item.Files++
			addLines(&item.Lines, metric.Lines)
			if directory == "." {
				break
			}
		}
		edges, info, module := inspectReferences(file.path, data, file.category)
		references = append(references, edges...)
		if info != nil {
			project.Configurations = append(project.Configurations, *info)
		}
		if file.path == "go.mod" {
			goModule = module
		}
		eligible = append(eligible, file)
	}
	for _, item := range languages {
		project.Languages = append(project.Languages, *item)
	}
	sort.Slice(project.Languages, func(i, j int) bool {
		if project.Languages[i].Lines.Total == project.Languages[j].Lines.Total {
			return project.Languages[i].Language < project.Languages[j].Language
		}
		return project.Languages[i].Lines.Total > project.Languages[j].Lines.Total
	})
	for _, item := range directories {
		project.Directories = append(project.Directories, *item)
	}
	sort.Slice(project.Directories, func(i, j int) bool { return project.Directories[i].Path < project.Directories[j].Path })
	project.Dependencies, project.Cycles = resolveReferences(references, project.Files, goModule)
	sort.Slice(project.Configurations, func(i, j int) bool { return project.Configurations[i].Path < project.Configurations[j].Path })
	if project.Skipped["inventory-limit"] > 0 {
		agg.warnings = append(agg.warnings, "项目画像达到文件数量上限，代码总量和关系图仅覆盖已读取文件。")
	}
	// Sample ownership deterministically by path hash rather than directory order.
	sort.Slice(eligible, func(i, j int) bool {
		left, right := shortHash(eligible[i].path), shortHash(eligible[j].path)
		if left == right {
			return eligible[i].path < eligible[j].path
		}
		return left < right
	})
	if cfg.EnableOwnership && len(eligible) > cfg.MaxOwnershipFiles {
		agg.skipped += int64(len(eligible) - cfg.MaxOwnershipFiles)
		agg.warnings = append(agg.warnings, fmt.Sprintf("归属分析按路径哈希选取 %d / %d 个文件，结果是部分覆盖，不代表完整仓库归属。", cfg.MaxOwnershipFiles, len(eligible)))
		eligible = eligible[:cfg.MaxOwnershipFiles]
	}
	return eligible, nil
}
