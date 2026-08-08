package filter

import (
	"testing"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/model"
)

func TestGlobMatcher(t *testing.T) {
	m := NewMatcher([]string{"node_modules/**", "**/*.generated.*", "*.lock"})
	for _, candidate := range []string{"node_modules/react/index.js", "src/api.generated.ts", "go.lock"} {
		if !m.Match(candidate) {
			t.Errorf("expected %q to match", candidate)
		}
	}
	if m.Match("src/main.go") {
		t.Fatal("source file should not match ignore rules")
	}
}

func TestClassifier(t *testing.T) {
	c := NewClassifier(config.Default())
	if got := c.Classify("internal/server/server_test.go"); got.Category != model.CategoryTest || got.Ignored {
		t.Fatalf("unexpected test classification: %#v", got)
	}
	if got := c.Classify("web/node_modules/react/index.js"); !got.Ignored {
		t.Fatal("vendored dependency should be ignored")
	}
}
