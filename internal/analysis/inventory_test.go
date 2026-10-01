package analysis

import (
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/model"
)

func testRepository(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git unavailable")
	}
	root := t.TempDir()
	testGit(t, root, "init", "--initial-branch=main")
	return root
}
func testGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	args = append([]string{"-C", root, "-c", "core.hooksPath=" + filepath.Join(root, "no-hooks"), "-c", "commit.gpgsign=false", "-c", "core.autocrlf=false"}, args...)
	command := exec.Command("git", args...)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_AUTHOR_NAME=Alice", "GIT_AUTHOR_EMAIL=alice@example.test", "GIT_COMMITTER_NAME=Alice", "GIT_COMMITTER_EMAIL=alice@example.test", "GIT_AUTHOR_DATE=2026-01-15T12:00:00Z", "GIT_COMMITTER_DATE=2026-01-15T12:00:00Z")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}
func testWrite(t *testing.T, root, file, content string) {
	t.Helper()
	destination := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
func testCommit(t *testing.T, root, message string) {
	t.Helper()
	testGit(t, root, "add", "--all")
	testGit(t, root, "commit", "-m", message)
}

func TestLineCountingCommentsStringsAndFinalNewlines(t *testing.T) {
	tests := []struct {
		name, source, language string
		expected               model.LineCounts
	}{
		{"Go", "package main\n\n// comment\n/* block\n end */\nconst url = \"https://example.test\" // mixed\n", "Go", model.LineCounts{Total: 6, Code: 2, Comment: 3, Blank: 1}},
		{"empty", "", "Go", model.LineCounts{}},
		{"no final newline", "const s = `/* literal */`", "Go", model.LineCounts{Total: 1, Code: 1}},
		{"raw multiline", "const s = `hello\n// literal\n`\n", "Go", model.LineCounts{Total: 3, Code: 3}},
		{"CRLF", "# comment\r\n\r\nx = '# literal'\r\n", "Python", model.LineCounts{Total: 3, Code: 1, Comment: 1, Blank: 1}},
		{"HTML", "<!-- comment\ncontinued -->\n<div>text</div>\n", "HTML", model.LineCounts{Total: 3, Code: 1, Comment: 2}},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			if actual := countLines([]byte(item.source), item.language); actual != item.expected {
				t.Fatalf("got %+v want %+v", actual, item.expected)
			}
		})
	}
}

func TestAnalyzeInventoryUsesCommitAndMatchesCacheAcrossProfiles(t *testing.T) {
	root := testRepository(t)
	testWrite(t, root, "go.mod", "module example.test/sample\n")
	testWrite(t, root, "src/main.go", "package main\n\n// comment\nimport \"example.test/sample/lib\"\n")
	testWrite(t, root, "lib/value.go", "package lib\nconst Value = 1\n")
	testWrite(t, root, "web/one.ts", "import { two } from './two'\nexport const one = two\n")
	testWrite(t, root, "web/two.ts", "import { one } from './one'\nexport const two = one\n")
	testWrite(t, root, "web/tsconfig.json", "{ // comment\n\"references\":[{\"path\":\"./tsconfig.app.json\"}],\n}\n")
	testWrite(t, root, "web/tsconfig.app.json", "{\"compilerOptions\":{\"strict\":true}}\n")
	testWrite(t, root, "web/package-lock.json", "{\"ignored\":true}\n")
	testWrite(t, root, "huge.txt", strings.Repeat("large\n", 100))
	testCommit(t, root, "subject\twith tab\n\nCo-authored-by: Bob <bob@example.test>")
	// Uncommitted changes must not affect inventory or source line totals.
	testWrite(t, root, "src/main.go", "uncommitted content\n")
	testWrite(t, root, "untracked.go", "untracked content\n")
	for _, profile := range []string{"fast", "balanced", "thorough"} {
		t.Run(profile, func(t *testing.T) {
			cfg := config.Default()
			cfg.Profile = profile
			cfg.MaxFileSizeBytes = 200
			analyzer := New()
			key, err := analyzer.CacheKey(context.Background(), root, cfg)
			if err != nil {
				t.Fatal(err)
			}
			result, err := analyzer.Analyze(context.Background(), root, cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			if key != result.ID {
				t.Fatalf("cache mismatch %s / %s", key, result.ID)
			}
			if result.Project.TotalFiles != 9 || result.Project.TextFiles != 7 {
				t.Fatalf("unexpected coverage: %+v", result.Project)
			}
			if result.Project.Skipped["ignored"] != 1 || result.Project.Skipped["oversized"] != 1 || result.Project.HistorySkipped["oversized"] != 1 {
				t.Fatalf("missing exclusions: %+v", result.Project)
			}
			var main model.FileMetric
			for _, file := range result.Project.Files {
				if file.Path == "src/main.go" {
					main = file
				}
				if file.Path == "untracked.go" {
					t.Fatal("included worktree file")
				}
			}
			if main.Lines.Total != 4 || main.Lines.Comment != 1 {
				t.Fatalf("did not read target blob: %+v", main)
			}
			if len(result.Project.Cycles) != 1 || strings.Join(result.Project.Cycles[0], ",") != "web/one.ts,web/two.ts" {
				t.Fatalf("unexpected cycles: %v", result.Project.Cycles)
			}
			foundGo, foundConfig := false, false
			for _, edge := range result.Project.Dependencies {
				if edge.Kind == "go-package" && edge.From == "src" && edge.To == "lib" && edge.Resolved {
					foundGo = true
				}
				if edge.Kind == "config-reference" && edge.To == "web/tsconfig.app.json" && edge.Resolved {
					foundConfig = true
				}
			}
			if !foundGo || !foundConfig {
				t.Fatalf("missing relationships: %+v", result.Project.Dependencies)
			}
			var additions, composite float64
			for _, person := range result.Contributors {
				additions += float64(person.Additions)
				composite += person.CompositeShare
				if person.RecentCommits[0].Additions != person.Additions {
					t.Fatal("coauthor commit detail differs")
				}
			}
			if additions != float64(result.Summary.Additions) || math.Abs(composite-100) > 0.03 {
				t.Fatalf("credits not conserved: %+v", result.Contributors)
			}
			if result.Config.Effective == nil || result.ProgramVersion == "" {
				t.Fatal("missing reproducibility data")
			}
		})
	}
}

func TestRenameAndSpecialPathsStayInCorrectDirectory(t *testing.T) {
	root := testRepository(t)
	testWrite(t, root, "src/old/main.go", "package main\nconst value = 1\n")
	testCommit(t, root, "initial")
	testGit(t, root, "mv", "src/old", "src/new")
	testCommit(t, root, "rename")
	testWrite(t, root, "src/new/main.go", "package main\nconst value = 2\n")
	testWrite(t, root, "src/中文 file.go", "package sample\n")
	if runtime.GOOS != "windows" {
		testWrite(t, root, "src/tab\tnewline\n.go", "package sample\n")
	}
	testCommit(t, root, "edit")
	cfg := config.Default()
	cfg.Profile = "fast"
	result, err := New().Analyze(context.Background(), root, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range result.Contributors[0].TopDirectories {
		if item.Name != "src" {
			t.Fatalf("lost rename prefix: %+v", item)
		}
	}
	for _, item := range result.Project.Files {
		if item.Path == "src/new/main.go" && item.Commits != 2 {
			t.Fatalf("rename/edit history missing: %+v", item)
		}
	}
}

func TestDateRangeDisablesRetentionAndNormalizesOwnershipOnly(t *testing.T) {
	root := testRepository(t)
	testWrite(t, root, "main.go", "package main\n")
	testCommit(t, root, "initial")
	cfg := config.Default()
	cfg.Since = "2027-01-01"
	result, err := New().Analyze(context.Background(), root, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Commits != 0 || result.Summary.OwnedLines != 1 || result.Contributors[0].CompositeShare != 100 || result.Config.EffectiveWeights.Ownership != 1 {
		t.Fatalf("empty history weights: %+v", result)
	}
	if result.Contributors[0].RetentionRate != 0 {
		t.Fatal("mixed retention ranges")
	}
	cfg.Since = "2026-01-15"
	cfg.Until = "2026-01-15"
	result, err = New().Analyze(context.Background(), root, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Commits != 1 {
		t.Fatal("end date did not include full day")
	}
}

func TestCancellationStopsAnalysis(t *testing.T) {
	root := testRepository(t)
	testWrite(t, root, "main.go", "package main\n")
	testCommit(t, root, "initial")
	ctx, cancel := context.WithCancel(context.Background())
	_, err := New().Analyze(ctx, root, config.Default(), func(progress model.Progress) {
		if progress.Phase == "ownership" {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancelled analysis, got %v", err)
	}
}

func TestCoauthorCreditsAgreeAcrossFilesAndTimeline(t *testing.T) {
	agg := newAggregate()
	cfg := config.Default()
	analyzer := New()
	record := &commitRecord{sha: "abc", name: "Alice", email: "alice@example.test", timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), coauthors: []identity{analyzer.identity("Bob", "bob@example.test", cfg)}, files: []fileChange{{path: "a.go", category: model.CategorySource, directory: "root", additions: 1}, {path: "b.go", category: model.CategorySource, directory: "root", additions: 1}}}
	analyzer.commit(record, cfg, agg)
	for id, item := range agg.contributors {
		if item.Contributor.Additions != 1 || item.Contributor.RecentCommits[0].Additions != 1 || agg.timeline["2026-01"].Contributors[id].Additions != 1 {
			t.Fatalf("credit discrepancy: %+v", item)
		}
	}
}

func TestJSONConfigStructureAndNoValuesStored(t *testing.T) {
	_, info, _ := inspectReferences("package.json", []byte(`{"token":"do-not-store","scripts":{"test":"never execute"},"dependencies":{"x":"1"},"scripts":{"build":"secret command"}}`), model.CategoryConfig)
	if info.ParseStatus != "valid" || len(info.Issues) != 1 || info.Dependencies != 1 || info.Scripts != 1 {
		t.Fatalf("unexpected config check: %+v", info)
	}
	_, invalid, _ := inspectReferences("broken.json", []byte(`{"x":`), model.CategoryConfig)
	if invalid.ParseStatus != "invalid" {
		t.Fatal("invalid JSON accepted")
	}
}

func TestAppRootResourceAndConfigSections(t *testing.T) {
	edges, _ := resolveReferences([]model.Dependency{{From: "web/index.html", To: "/src/main.tsx", Kind: "asset-reference"}}, []model.FileMetric{{Path: "web/package.json"}, {Path: "web/src/main.tsx"}}, "")
	if len(edges) != 1 || !edges[0].Resolved || edges[0].To != "web/src/main.tsx" {
		t.Fatalf("wrong app resource root: %+v", edges)
	}
	_, info, _ := inspectReferences("settings.toml", []byte("[first]\nvalue = 1\n[second]\nvalue = 2\n"), model.CategoryConfig)
	if len(info.Issues) != 0 || strings.Join(info.Keys, ",") != "first.value,second.value" {
		t.Fatalf("valid config sections marked duplicate: %+v", info)
	}
}

func TestCacheInvalidatesIdentityInputsWithoutMovingHead(t *testing.T) {
	root := testRepository(t)
	testWrite(t, root, "main.go", "package main\n")
	testCommit(t, root, "initial")
	analyzer := New()
	key := func() string {
		value, err := analyzer.CacheKey(context.Background(), root, config.Default())
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := key()
	testWrite(t, root, ".mailmap", "Canonical <canonical@example.test> <alice@example.test>\n")
	if key() == before {
		t.Fatal("mailmap change reused cache")
	}
	before = key()
	testGit(t, root, "config", "mailmap.file", "custom-mailmap")
	testWrite(t, root, "custom-mailmap", "Alias <alias@example.test> <alice@example.test>\n")
	if key() == before {
		t.Fatal("custom mailmap setting reused cache")
	}
	before = key()
	testWrite(t, root, "custom-mailmap", "Another <another@example.test> <alice@example.test>\n")
	if key() == before {
		t.Fatal("custom mailmap content reused cache")
	}
	before = key()
	analyzer.Version = "test-new-version"
	if key() == before {
		t.Fatal("program upgrade reused cache")
	}
}
