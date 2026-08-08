package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
