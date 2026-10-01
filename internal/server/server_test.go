package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/storage"
)

func TestHealthAndDefaultConfig(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, nil).Handler()
	for _, endpoint := range []string{"/api/health", "/api/config/default"} {
		request := httptest.NewRequest(http.MethodGet, endpoint, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s returned %d: %s", endpoint, response.Code, response.Body.String())
		}
	}
}

func TestConfigValidationAndTrailingJSON(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, nil).Handler()
	cases := []struct {
		body   string
		status int
	}{
		{`{"enableOwnership":false}`, 200},
		{`{"weights":{"workload":-1}}`, 400},
		{`{"since":"2026-02-03","until":"2026-02-01"}`, 400},
		{`{"unknown":true}`, 400},
		{`{} {}`, 400},
	}
	for _, item := range cases {
		request := httptest.NewRequest(http.MethodPost, "/api/config/validate", strings.NewReader(item.body))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != item.status {
			t.Fatalf("%s: %d %s", item.body, response.Code, response.Body.String())
		}
	}
}

func TestBoundedQueueAndCancellation(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	application := New(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	application.slots <- struct{}{}
	defer func() { <-application.slots }()
	handler := application.Handler()
	for index := 0; index < 9; index++ {
		request := httptest.NewRequest(http.MethodPost, "/api/jobs", strings.NewReader(`{"repositoryPath":"unused-repository"}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		expected := http.StatusAccepted
		if index == 8 {
			expected = http.StatusTooManyRequests
		}
		if response.Code != expected {
			t.Fatalf("request %d: %d %s", index, response.Code, response.Body.String())
		}
	}
	application.mu.RLock()
	ids := []string{}
	for id := range application.jobs {
		ids = append(ids, id)
	}
	application.mu.RUnlock()
	for _, id := range ids {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/jobs/"+id, nil))
		if response.Code != 200 {
			t.Fatal(response.Code)
		}
	}
}

func TestSaveFailureDoesNotReportCompletion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git unavailable")
	}
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", root, "-c", "core.hooksPath=" + filepath.Join(root, "no-hooks"), "-c", "commit.gpgsign=false"}, args...)...)
		command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.test", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.test")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, output)
		}
	}
	run("init")
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "main.go")
	run("commit", "-m", "initial")
	cache := filepath.Join(t.TempDir(), "cache")
	store, err := storage.New(cache)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(cache); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	application := New(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	application.jobs["test"] = &Job{ID: "test", Status: "queued"}
	cfg := config.Default()
	cfg.Profile = "fast"
	application.runJob(context.Background(), "test", root, cfg)
	if application.jobs["test"].Status != "failed" || application.jobs["test"].AnalysisID != "" {
		t.Fatalf("save failure marked complete: %+v", application.jobs["test"])
	}
}

func TestCancelJob(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	application := New(store, nil)
	ctx, cancel := context.WithCancel(context.Background())
	application.jobs["active-job"] = &Job{ID: "active-job", Status: "running"}
	application.cancels["active-job"] = cancel

	request := httptest.NewRequest(http.MethodDelete, "/api/jobs/active-job", nil)
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("cancel returned %d: %s", response.Code, response.Body.String())
	}
	if application.jobs["active-job"].Status != "cancelled" {
		t.Fatalf("expected cancelled status, got %q", application.jobs["active-job"].Status)
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("job context was not cancelled")
	}
}
