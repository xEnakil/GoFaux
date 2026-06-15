package httpserver

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"GoFaux/internal/config"
	"GoFaux/internal/generator"
	"GoFaux/internal/mock"
	"GoFaux/internal/modelhub"
)

//go:embed ui/*
var uiFiles embed.FS

type Server struct {
	mu            sync.Mutex
	cfgMu         sync.RWMutex
	trafficMu     sync.RWMutex
	downloadMu    sync.RWMutex
	evalMu        sync.RWMutex
	store         *mock.Store
	cfg           config.Config
	addr          string
	server        *http.Server
	running       bool
	traffic       []TrafficEvent
	nextTrafficID int64
	downloads     map[string]*modelDownloadJob
	evaluations   map[string]*evaluationJob
}

func New(store *mock.Store, cfg config.Config) *Server {
	if store == nil {
		store = mock.NewMemoryStore()
	}
	addr := cfg.ServerAddr
	if addr == "" {
		addr = ":8080"
	}
	cfg.ServerAddr = addr
	return &Server{store: store, cfg: cfg, addr: addr, downloads: map[string]*modelDownloadJob{}, evaluations: map[string]*evaluationJob{}}
}

func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addr
}

func (s *Server) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}

	httpServer := &http.Server{
		Addr:              s.addr,
		Handler:           requestLogger(s.routes()),
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("listen on %s: %w", s.addr, err)
	}

	s.server = httpServer
	s.running = true
	s.mu.Unlock()

	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("mock server stopped with error: %v\n", err)
		}
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running || s.server == nil {
		s.mu.Unlock()
		return nil
	}
	httpServer := s.server
	s.mu.Unlock()

	if err := httpServer.Shutdown(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	return nil
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/_gofaux", s.handleUIRedirect)
	mux.Handle("/_gofaux/ui/", s.uiHandler())
	mux.HandleFunc("/_gofaux/health", s.handleHealth)
	mux.HandleFunc("/_gofaux/mocks", s.handleMocks)
	mux.HandleFunc("/_gofaux/api/state", s.handleAPIState)
	mux.HandleFunc("/_gofaux/api/generate", s.handleAPIGenerate)
	mux.HandleFunc("/_gofaux/api/mocks", s.handleAPIMocks)
	mux.HandleFunc("/_gofaux/api/mocks/", s.handleAPIMockByID)
	mux.HandleFunc("/_gofaux/api/settings", s.handleAPISettings)
	mux.HandleFunc("/_gofaux/api/models", s.handleAPIModels)
	mux.HandleFunc("/_gofaux/api/models/download", s.handleAPIModelDownload)
	mux.HandleFunc("/_gofaux/api/models/downloads/", s.handleAPIModelDownloadJob)
	mux.HandleFunc("/_gofaux/api/openapi/preview", s.handleAPIOpenAPIPreview)
	mux.HandleFunc("/_gofaux/api/openapi/import", s.handleAPIOpenAPIImport)
	mux.HandleFunc("/_gofaux/api/project-scan/preview", s.handleAPIProjectScanPreview)
	mux.HandleFunc("/_gofaux/api/project-scan/import", s.handleAPIProjectScanImport)
	mux.HandleFunc("/_gofaux/api/evaluations", s.handleAPIEvaluations)
	mux.HandleFunc("/_gofaux/api/evaluations/", s.handleAPIEvaluationJob)
	mux.HandleFunc("/_gofaux/api/traffic", s.handleAPITraffic)
	mux.HandleFunc("/favicon.ico", s.handleFavicon)
	mux.HandleFunc("/", s.handleMock)
	return mux
}

func (s *Server) UIURL() string {
	return displayAddr(s.Addr()) + "/_gofaux/ui/"
}

func (s *Server) handleUIRedirect(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/_gofaux" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/_gofaux/ui/", http.StatusFound)
}

func (s *Server) uiHandler() http.Handler {
	sub, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.StripPrefix("/_gofaux/ui/", http.FileServer(http.FS(sub)))
}

func (s *Server) handleFavicon(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"mocks":  s.store.Count(),
	})
}

func (s *Server) handleAPIState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	cfg := s.configSnapshot()
	mocks := s.store.All()
	traffic := s.trafficSummary()

	writeJSON(w, http.StatusOK, map[string]any{
		"server": map[string]any{
			"running":       s.Running(),
			"address":       displayAddr(s.Addr()),
			"ui_url":        s.UIURL(),
			"mock_config":   s.store.ConfigPath(),
			"settings_path": cfg.SettingsPath,
		},
		"assistant": cfg.Assistant,
		"models": map[string]any{
			"model_dir":  cfg.ModelDir,
			"runner_dir": cfg.RunnerDir,
		},
		"stats": map[string]any{
			"mock_count":       len(mocks),
			"total_requests":   traffic.Total,
			"matched_requests": traffic.Matched,
			"missed_requests":  traffic.Missed,
			"average_ms":       traffic.AverageMS,
		},
		"traffic": traffic,
		"mocks":   mocks,
	})
}

func (s *Server) handleAPIGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var input generator.Request
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request JSON: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	result, err := generator.Generate(ctx, s.configSnapshot(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAPIMocks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"count": s.store.Count(),
			"mocks": s.store.All(),
		})
	case http.MethodPost:
		var input mock.Definition
		if err := readJSON(r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid mock JSON: " + err.Error()})
			return
		}
		if input.Headers == nil {
			input.Headers = map[string]string{"Content-Type": "application/json"}
		}
		def, err := s.store.Add(input)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, def)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleAPIMockByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/_gofaux/api/mocks/")
	if id == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "mock id is required"})
		return
	}
	switch r.Method {
	case http.MethodDelete:
		removed, ok, err := s.store.DeleteByID(id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "mock not found"})
			return
		}
		writeJSON(w, http.StatusOK, removed)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleAPISettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.configSnapshot().Assistant)
	case http.MethodPost:
		var input struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Endpoint string `json:"endpoint"`
		}
		if err := readJSON(r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid settings JSON: " + err.Error()})
			return
		}
		cfg := s.configSnapshot()
		cfg.Assistant.Provider = strings.TrimSpace(input.Provider)
		cfg.Assistant.Model = strings.TrimSpace(input.Model)
		cfg.Assistant.Endpoint = strings.TrimSpace(input.Endpoint)
		if err := config.SaveSettings(cfg); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		s.setConfig(cfg)
		writeJSON(w, http.StatusOK, cfg.Assistant)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleAPIModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	cfg := s.configSnapshot()
	manager := modelhub.NewManager(cfg.ModelDir)
	downloaded, err := manager.ListDownloaded()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"model_dir":  manager.Dir(),
		"catalog":    manager.Catalog(),
		"downloaded": downloaded,
		"downloads":  s.downloadJobSnapshots(),
	})
}

func (s *Server) handleAPITraffic(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.trafficSummary())
	case http.MethodDelete:
		s.clearTraffic()
		writeJSON(w, http.StatusOK, map[string]any{"status": "cleared"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleMocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": s.store.Count(),
		"mocks": s.store.All(),
	})
}

func (s *Server) handleMock(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	requestBody, requestBytes, bodyTruncated := captureRequestBody(r)

	def, match, ok := s.store.Find(r)
	if !ok {
		status := http.StatusNotFound
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":  "no mock matched the request",
			"method": r.Method,
			"path":   r.URL.Path,
		})
		s.recordTraffic(trafficEventFromRequest(r, status, false, mock.Definition{}, mock.Match{}, started, requestBody, requestBytes, bodyTruncated, 0))
		return
	}

	if def.DelayMS > 0 {
		time.Sleep(time.Duration(def.DelayMS) * time.Millisecond)
	}
	for key, value := range def.Headers {
		w.Header().Set(key, value)
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.Header().Set("X-GoFaux-Mock-ID", def.ID)
	w.WriteHeader(def.Status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(def.Body)
	}
	s.store.RecordRequest(def.ID)
	responseBytes := 0
	if r.Method != http.MethodHead {
		responseBytes = len(def.Body)
	}
	s.recordTraffic(trafficEventFromRequest(r, def.Status, true, def, match, started, requestBody, requestBytes, bodyTruncated, responseBytes))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func readJSON(r *http.Request, out any) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(out)
}

const maxLoggedBodyBytes = 64 * 1024

func captureRequestBody(r *http.Request) (string, int, bool) {
	if r.Body == nil {
		return "", 0, false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxLoggedBodyBytes+1))
	_ = r.Body.Close()
	if err != nil {
		r.Body = io.NopCloser(bytes.NewReader(nil))
		return "", 0, false
	}
	truncated := len(data) > maxLoggedBodyBytes
	if truncated {
		data = data[:maxLoggedBodyBytes]
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	return string(data), len(data), truncated
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (s *Server) configSnapshot() config.Config {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg
}

func (s *Server) setConfig(cfg config.Config) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.cfg = cfg
}

func displayAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
}
