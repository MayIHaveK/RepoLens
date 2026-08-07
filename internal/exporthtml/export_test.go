package exporthtml

import (
	"bytes"
	"testing"

	"github.com/repolens/repolens/internal/model"
)

func TestPrivacyRemovesSensitiveDisplayFields(t *testing.T) {
	analysis := &model.Analysis{
		SchemaVersion: model.SchemaVersion,
		Repository:    model.Repository{Name: "secret-repository", CommitSHA: "1234567890"},
		Contributors: []model.Contributor{{
			Name: "Private Name", TopDirectories: []model.DirectoryShare{{Name: "secret-module"}},
			RecentCommits: []model.CommitSummary{{SHA: "abcdef", Message: "secret feature"}},
		}},
	}
	document, err := Render(analysis, model.Privacy{})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range [][]byte{[]byte("secret-repository"), []byte("1234567890"), []byte("Private Name"), []byte("secret-module"), []byte("secret feature")} {
		if bytes.Contains(document, secret) {
			t.Fatalf("export leaked %q", secret)
		}
	}
}
