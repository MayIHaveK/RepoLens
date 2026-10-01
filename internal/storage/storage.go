package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/MayIHaveK/RepoLens/internal/model"
)

type Store struct {
	directory string
	mu        sync.Mutex
}

func New(directory string) (*Store, error) {
	if directory == "" {
		if configured := os.Getenv("REPOLENS_CACHE_DIR"); configured != "" {
			directory = configured
		} else {
			base, err := os.UserCacheDir()
			if err != nil {
				return nil, err
			}
			directory = filepath.Join(base, "RepoLens", "analyses")
		}
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	return &Store{directory: directory}, nil
}

func (s *Store) Save(analysis *model.Analysis) error {
	if analysis == nil || !validID(analysis.ID) {
		return errors.New("invalid analysis")
	}
	data, err := json.Marshal(analysis)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.writeAtomic(s.path(analysis.ID), data); err != nil {
		return err
	}
	// A small sidecar keeps recent lists independent of large file inventories.
	summary := *analysis
	summary.Project = nil
	summary.Contributors = nil
	summary.Timeline = nil
	summary.Categories = nil
	summary.Config.Effective = nil
	data, err = json.Marshal(summary)
	if err != nil {
		return err
	}
	return s.writeAtomic(filepath.Join(s.directory, analysis.ID+".summary"), data)
}

func (s *Store) writeAtomic(destination string, data []byte) error {
	file, err := os.CreateTemp(s.directory, "repolens-*.tmp")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(name, destination)
}

func (s *Store) Load(id string) (*model.Analysis, error) {
	if !validID(id) {
		return nil, os.ErrNotExist
	}
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var analysis model.Analysis
	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, err
	}
	if analysis.SchemaVersion != 1 && analysis.SchemaVersion != model.SchemaVersion {
		return nil, fmt.Errorf("unsupported analysis schema %d", analysis.SchemaVersion)
	}
	return &analysis, nil
}

func (s *Store) List(limit int) ([]model.Analysis, error) {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, err
	}
	results := make([]model.Analysis, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(s.directory, id+".summary"))
		var analysis *model.Analysis
		if err == nil {
			analysis = &model.Analysis{}
			err = json.Unmarshal(data, analysis)
		} else {
			analysis, err = s.Load(id)
			if err == nil {
				analysis.Project = nil
				analysis.Contributors = nil
				analysis.Timeline = nil
				analysis.Categories = nil
				analysis.Config.Effective = nil
			}
		}
		if err == nil {
			results = append(results, *analysis)
		}
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].GeneratedAt.After(results[j].GeneratedAt) })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.directory, id+".json")
}

func validID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, char := range id {
		if (char < 'a' || char > 'f') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}
