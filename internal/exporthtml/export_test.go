package exporthtml

import (
	"bytes"
	"testing"

	"github.com/MayIHaveK/RepoLens/internal/model"
)

func TestPrivacyRemovesSensitiveDisplayFields(t *testing.T) {
	analysis := &model.Analysis{
		SchemaVersion: model.SchemaVersion,
		Repository:    model.Repository{Name: "secret-repository", CommitSHA: "1234567890"},
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
	for _, secret := range [][]byte{[]byte("secret-repository"), []byte("1234567890"), []byte("private-email-derived-id"), []byte("Private Name"), []byte("secret-module"), []byte("secret feature")} {
		if bytes.Contains(document, secret) {
			t.Fatalf("export leaked %q", secret)
		}
	}
}
