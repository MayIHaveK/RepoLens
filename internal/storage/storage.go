package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/repolens/repolens/internal/model"
)

type Store struct {
	directory string
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
	path := s.path(analysis.ID)
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
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
	if analysis.SchemaVersion != model.SchemaVersion {
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
		analysis, err := s.Load(strings.TrimSuffix(entry.Name(), ".json"))
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
