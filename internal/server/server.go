package server

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/repolens/repolens/internal/analysis"
	"github.com/repolens/repolens/internal/config"
	"github.com/repolens/repolens/internal/exporthtml"
	"github.com/repolens/repolens/internal/model"
	"github.com/repolens/repolens/internal/storage"
)

//go:embed all:web
var webFiles embed.FS

type Server struct {
	analyzer *analysis.Analyzer
	store    *storage.Store
	logger   *slog.Logger
	jobs     map[string]*Job
	mu       sync.RWMutex
}

type Job struct {
	ID         string         `json:"id"`
	Status     string         `json:"status"`
	Progress   model.Progress `json:"progress"`
	AnalysisID string         `json:"analysisId,omitempty"`
	Error      string         `json:"error,omitempty"`
	Cached     bool           `json:"cached"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type analyzeRequest struct {
	RepositoryPath string         `json:"repositoryPath"`
	Config         *config.Config `json:"config"`
}

type exportRequest struct {
	AnalysisID string        `json:"analysisId"`
	Privacy    model.Privacy `json:"privacy"`
}

func New(store *storage.Store, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		analyzer: analysis.New(), store: store, logger: logger,
		jobs: map[string]*Job{},
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/config/default", s.handleDefaultConfig)
	mux.HandleFunc("POST /api/jobs", s.handleCreateJob)
	mux.HandleFunc("GET /api/jobs/{id}", s.handleJob)
	mux.HandleFunc("GET /api/analyses/{id}", s.handleAnalysis)
	mux.HandleFunc("GET /api/analyses", s.handleRecent)
	mux.HandleFunc("POST /api/export", s.handleExport)
	mux.HandleFunc("GET /api/github/status", s.handleGitHubStatus)
	mux.Handle("/", s.staticHandler())
	return s.middleware(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "schemaVersion": model.SchemaVersion})
}

func (s *Server) handleDefaultConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, config.Default())
}

func (s *Server) handleCreateJob(w http.ResponseWriter, request *http.Request) {
	var input analyzeRequest
	if err := decodeJSON(w, request, &input); err != nil {
		return
	}
	if strings.TrimSpace(input.RepositoryPath) == "" {
		writeError(w, http.StatusBadRequest, "repositoryPath is required")
		return
	}
	cfg := config.Default()
	if input.Config != nil {
		cfg = *input.Config
		cfg.Normalize()
	}
	job := &Job{
		ID: newID(), Status: "queued", Progress: model.Progress{Phase: "queued", Message: "等待分析", Percent: 0},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()
	go s.runJob(context.Background(), job.ID, input.RepositoryPath, cfg)
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) runJob(ctx context.Context, jobID, repository string, cfg config.Config) {
	s.updateJob(jobID, func(job *Job) {
		job.Status = "running"
		job.Progress = model.Progress{Phase: "cache", Message: "正在检查分析缓存", Percent: 1}
	})
	cacheKey, err := s.analyzer.CacheKey(ctx, repository, cfg)
	if err == nil {
		if cached, loadErr := s.store.Load(cacheKey); loadErr == nil {
			s.updateJob(jobID, func(job *Job) {
				job.Status = "complete"
				job.Cached = true
				job.AnalysisID = cached.ID
				job.Progress = model.Progress{Phase: "complete", Message: "已载入缓存结果", Current: 1, Total: 1, Percent: 100}
			})
			return
		}
	}

	result, err := s.analyzer.Analyze(ctx, repository, cfg, func(progress model.Progress) {
		s.updateJob(jobID, func(job *Job) { job.Progress = progress })
	})
	if err != nil {
		s.logger.Error("analysis failed", "job", jobID, "error", err)
		s.updateJob(jobID, func(job *Job) {
			job.Status = "failed"
			job.Error = err.Error()
		})
		return
	}
	if err := s.store.Save(result); err != nil {
		s.logger.Error("save analysis", "job", jobID, "error", err)
	}
	s.updateJob(jobID, func(job *Job) {
		job.Status = "complete"
		job.AnalysisID = result.ID
		job.Progress = model.Progress{Phase: "complete", Message: "分析完成", Current: 1, Total: 1, Percent: 100}
	})
}

func (s *Server) updateJob(id string, update func(*Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job := s.jobs[id]; job != nil {
		update(job)
		job.UpdatedAt = time.Now().UTC()
	}
}

func (s *Server) handleJob(w http.ResponseWriter, request *http.Request) {
	s.mu.RLock()
	job := s.jobs[request.PathValue("id")]
	if job != nil {
		copy := *job
		job = &copy
	}
	s.mu.RUnlock()
	if job == nil {
		writeError(w, http.StatusNotFound, "analysis job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleAnalysis(w http.ResponseWriter, request *http.Request) {
	result, err := s.store.Load(request.PathValue("id"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			writeError(w, http.StatusNotFound, "analysis not found")
		} else {
			writeError(w, http.StatusInternalServerError, "cannot read analysis")
		}
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRecent(w http.ResponseWriter, _ *http.Request) {
	results, err := s.store.List(8)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot list analyses")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleExport(w http.ResponseWriter, request *http.Request) {
	var input exportRequest
	if err := decodeJSON(w, request, &input); err != nil {
		return
	}
	result, err := s.store.Load(input.AnalysisID)
	if err != nil {
		writeError(w, http.StatusNotFound, "analysis not found")
		return
	}
	document, err := exporthtml.Render(result, input.Privacy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot render report")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="repolens-report.html"`)
	w.Header().Set("Content-Length", fmt.Sprint(len(document)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
}

func (s *Server) handleGitHubStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"connected": false,
		"available": false,
		"message":   "GitHub 增强分析将在下一个里程碑开放",
	})
}

func (s *Server) staticHandler() http.Handler {
	sub, _ := fs.Sub(webFiles, "web")
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requested := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if requested == "." || requested == "" {
			requested = "index.html"
		}
		data, err := fs.ReadFile(sub, requested)
		if err != nil && !strings.Contains(path.Base(requested), ".") {
			data, err = fs.ReadFile(sub, "index.html")
		}
		if err != nil {
			http.Error(w, "RepoLens frontend has not been built. Run npm run build in web/.", http.StatusServiceUnavailable)
			return
		}
		if contentType := mime.TypeByExtension(path.Ext(requested)); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		_, _ = w.Write(data)
	})
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'")
		if origin := request.Header.Get("Origin"); origin == "http://127.0.0.1:5173" || origin == "http://localhost:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if request.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, request)
	})
}

func decodeJSON(w http.ResponseWriter, request *http.Request, target any) error {
	request.Body = http.MaxBytesReader(w, request.Body, 2<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func newID() string {
	buffer := make([]byte, 12)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}
