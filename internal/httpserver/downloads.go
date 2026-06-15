package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"GoFaux/internal/modelhub"
)

type modelDownloadJob struct {
	ID           string                    `json:"id"`
	ModelID      string                    `json:"model_id,omitempty"`
	Name         string                    `json:"name"`
	Status       string                    `json:"status"`
	Error        string                    `json:"error,omitempty"`
	Path         string                    `json:"path,omitempty"`
	Progress     modelhub.Progress         `json:"progress"`
	Percent      float64                   `json:"percent"`
	StartedAt    time.Time                 `json:"started_at"`
	FinishedAt   time.Time                 `json:"finished_at,omitempty"`
	Downloaded   *modelhub.DownloadedModel `json:"downloaded,omitempty"`
	CustomSource bool                      `json:"custom_source,omitempty"`
}

type modelDownloadRequest struct {
	ID       string `json:"id,omitempty"`
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	Name     string `json:"name,omitempty"`
}

func (s *Server) startModelDownload(input modelDownloadRequest) (modelDownloadJob, error) {
	cfg := s.configSnapshot()
	manager := modelhub.NewManager(cfg.ModelDir)

	spec, custom, err := downloadSpec(input)
	if err != nil {
		return modelDownloadJob{}, err
	}
	if manager.IsDownloaded(spec) {
		path := manager.ModelPath(spec)
		return modelDownloadJob{
			ID:           "download_existing_" + safeJobID(spec.ID),
			ModelID:      spec.ID,
			Name:         spec.Name,
			Status:       "completed",
			Path:         path,
			Progress:     modelhub.Progress{WrittenBytes: spec.SizeBytes, TotalBytes: spec.SizeBytes, Done: true},
			Percent:      100,
			StartedAt:    time.Now(),
			FinishedAt:   time.Now(),
			CustomSource: custom,
		}, nil
	}

	job := &modelDownloadJob{
		ID:           fmt.Sprintf("download_%d", time.Now().UnixNano()),
		ModelID:      spec.ID,
		Name:         spec.Name,
		Status:       "running",
		StartedAt:    time.Now(),
		CustomSource: custom,
	}
	s.storeDownloadJob(job)

	go func() {
		downloaded, err := manager.Download(context.Background(), spec, func(progress modelhub.Progress) {
			s.updateDownloadJob(job.ID, func(current *modelDownloadJob) {
				current.Progress = progress
				current.Percent = downloadPercent(progress)
				current.Status = "running"
			})
		})
		s.updateDownloadJob(job.ID, func(current *modelDownloadJob) {
			current.FinishedAt = time.Now()
			if err != nil {
				current.Status = "failed"
				current.Error = err.Error()
				return
			}
			current.Status = "completed"
			current.Path = downloaded.Path
			current.Progress.Done = true
			current.Progress.WrittenBytes = downloaded.SizeBytes
			if current.Progress.TotalBytes <= 0 {
				current.Progress.TotalBytes = downloaded.SizeBytes
			}
			current.Percent = 100
			current.Downloaded = &downloaded
		})
	}()

	return s.downloadJobSnapshot(job.ID), nil
}

func downloadSpec(input modelDownloadRequest) (modelhub.ModelSpec, bool, error) {
	if strings.TrimSpace(input.ID) != "" {
		spec, ok := modelhub.Find(strings.TrimSpace(input.ID))
		if !ok {
			return modelhub.ModelSpec{}, false, fmt.Errorf("unknown model id %q", input.ID)
		}
		return spec, false, nil
	}
	if strings.TrimSpace(input.URL) == "" {
		return modelhub.ModelSpec{}, false, fmt.Errorf("model id or URL is required")
	}
	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		filename = filepath.Base(strings.TrimSpace(input.URL))
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".gguf") {
		return modelhub.ModelSpec{}, false, fmt.Errorf("filename should end with .gguf")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = filename
	}
	id := "custom-" + strings.TrimSuffix(strings.ToLower(strings.ReplaceAll(filename, " ", "-")), ".gguf")
	return modelhub.ModelSpec{
		ID:       id,
		Name:     name,
		Family:   "custom",
		Quant:    "unknown",
		URL:      strings.TrimSpace(input.URL),
		Filename: filename,
		Notes:    "Custom user-provided GGUF URL.",
	}, true, nil
}

func (s *Server) storeDownloadJob(job *modelDownloadJob) {
	s.downloadMu.Lock()
	defer s.downloadMu.Unlock()
	if s.downloads == nil {
		s.downloads = map[string]*modelDownloadJob{}
	}
	s.downloads[job.ID] = job
}

func (s *Server) updateDownloadJob(id string, update func(*modelDownloadJob)) {
	s.downloadMu.Lock()
	defer s.downloadMu.Unlock()
	if s.downloads == nil {
		return
	}
	if job, ok := s.downloads[id]; ok {
		update(job)
	}
}

func (s *Server) downloadJobSnapshot(id string) modelDownloadJob {
	s.downloadMu.RLock()
	defer s.downloadMu.RUnlock()
	if s.downloads == nil {
		return modelDownloadJob{}
	}
	if job, ok := s.downloads[id]; ok {
		return *job
	}
	return modelDownloadJob{}
}

func (s *Server) downloadJobSnapshots() []modelDownloadJob {
	s.downloadMu.RLock()
	defer s.downloadMu.RUnlock()
	jobs := make([]modelDownloadJob, 0, len(s.downloads))
	for _, job := range s.downloads {
		jobs = append(jobs, *job)
	}
	return jobs
}

func downloadPercent(progress modelhub.Progress) float64 {
	if progress.TotalBytes <= 0 {
		if progress.Done {
			return 100
		}
		return 0
	}
	percent := float64(progress.WrittenBytes) / float64(progress.TotalBytes) * 100
	if percent > 100 {
		return 100
	}
	return percent
}

func safeJobID(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "model"
	}
	return b.String()
}

func (s *Server) handleAPIModelDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var input modelDownloadRequest
	if err := readJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	job, err := s.startModelDownload(input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) handleAPIModelDownloadJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/_gofaux/api/models/downloads/")
	job := s.downloadJobSnapshot(id)
	if job.ID == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "download job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}
