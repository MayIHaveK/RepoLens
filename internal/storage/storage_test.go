package storage

import (
	"testing"
	"time"

	"github.com/repolens/repolens/internal/model"
)

func TestSaveAndLoad(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	original := &model.Analysis{SchemaVersion: model.SchemaVersion, ID: "aabbccddeeff0011", GeneratedAt: time.Now().UTC()}
	if err := store.Save(original); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != original.ID {
		t.Fatalf("expected %q, got %q", original.ID, loaded.ID)
	}
}
