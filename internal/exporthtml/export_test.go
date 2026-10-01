package exporthtml

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/MayIHaveK/RepoLens/internal/model"
)

func TestPrivacyRemovesSensitiveDisplayFields(t *testing.T) {
	analysis := &model.Analysis{
		SchemaVersion: model.SchemaVersion,
		Config:        model.ConfigSnapshot{Effective: json.RawMessage(`{"aliases":{"private@example.test":"canonical@example.test"},"ignoredPatterns":["private-filename.go"]}`)},
		Project: &model.Project{
			Files: []model.FileMetric{{Path: "private-filename.go"}}, Dependencies: []model.Dependency{{From: "private-source.go", To: "private-target.go"}},
			Configurations: []model.ConfigurationInfo{{Path: "private-config.json", Keys: []string{"private-key"}}},
			Directories:    []model.DirectorySummary{{Path: "private-directory"}}, Cycles: [][]string{{"private-cycle.go"}},
			Totals: model.LineCounts{Total: 42},
		},
		Repository: model.Repository{Name: "secret-repository", CommitSHA: "1234567890"},
		Contributors: []model.Contributor{{
			ID: "private-email-derived-id", Name: "Private Name", TopDirectories: []model.DirectoryShare{{Name: "secret-module"}},
			RecentCommits: []model.CommitSummary{{SHA: "abcdef", Message: "secret feature"}},
		}},
		Timeline: []model.TimelineBucket{{Period: "2026-08", Contributors: map[string]model.Change{"private-email-derived-id": {Additions: 1}}}},
	}
	document, err := Render(analysis, model.Privacy{})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range [][]byte{[]byte("secret-repository"), []byte("1234567890"), []byte("private-email-derived-id"), []byte("Private Name"), []byte("secret-module"), []byte("secret feature"), []byte("private@example.test"), []byte("private-filename.go"), []byte("private-source.go"), []byte("private-target.go"), []byte("private-config.json"), []byte("private-key"), []byte("private-directory"), []byte("private-cycle.go")} {
		if bytes.Contains(document, secret) {
			t.Fatalf("export leaked %q", secret)
		}
	}
	if analysis.Project.Files == nil || analysis.Config.Effective == nil {
		t.Fatal("export mutated original analysis")
	}
}
