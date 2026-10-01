package storage

import (
	"sync"
	"testing"
	"time"

	"github.com/MayIHaveK/RepoLens/internal/model"
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

func TestConcurrentSavesAndLightweightRecentList(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	analysis := &model.Analysis{ID: "aabbccddeeff0011", SchemaVersion: model.SchemaVersion, GeneratedAt: time.Now().UTC(), Project: &model.Project{Files: []model.FileMetric{{Path: "sample.go"}}}}
	var group sync.WaitGroup
	failures := make(chan error, 8)
	for index := 0; index < 8; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := store.Save(analysis); err != nil {
				failures <- err
			}
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	full, err := store.Load(analysis.ID)
	if err != nil {
		t.Fatal(err)
	}
	if full.Project == nil {
		t.Fatal("full inventory lost")
	}
	recent, err := store.List(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 1 || recent[0].Project != nil {
		t.Fatal("recent list loaded full inventory")
	}
}
