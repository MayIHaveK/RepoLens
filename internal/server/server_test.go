package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/repolens/repolens/internal/storage"
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
