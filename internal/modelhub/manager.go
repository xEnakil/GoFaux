package modelhub

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Progress struct {
	WrittenBytes int64
	TotalBytes   int64
	Done         bool
}

type Manager struct {
	dir    string
	client *http.Client
}

func NewManager(dir string) *Manager {
	if strings.TrimSpace(dir) == "" {
		dir = DefaultDir()
	}
	return &Manager{
		dir: dir,
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

func DefaultDir() string {
	return filepath.Join(".gofaux", "models")
}

func (m *Manager) Dir() string {
	return m.dir
}

func (m *Manager) Catalog() []ModelSpec {
	return Catalog()
}

func (m *Manager) ListDownloaded() ([]DownloadedModel, error) {
	var downloaded []DownloadedModel
	seen := map[string]bool{}
	for _, spec := range Catalog() {
		path := m.ModelPath(spec)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		downloaded = append(downloaded, DownloadedModel{
			Spec:      spec,
			Path:      path,
			SizeBytes: info.Size(),
		})
		seen[filepath.Clean(path)] = true
	}

	matches, err := filepath.Glob(filepath.Join(m.dir, "*.gguf"))
	if err != nil {
		return nil, err
	}
	for _, path := range matches {
		clean := filepath.Clean(path)
		if seen[clean] {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || info.Size() == 0 {
			continue
		}
		filename := filepath.Base(path)
		downloaded = append(downloaded, DownloadedModel{
			Spec: ModelSpec{
				ID:       "custom-" + strings.TrimSuffix(strings.ToLower(filename), ".gguf"),
				Name:     filename,
				Family:   "custom",
				Quant:    "unknown",
				Filename: filename,
				Notes:    "Custom or manually placed GGUF model.",
			},
			Path:      path,
			SizeBytes: info.Size(),
		})
	}
	return downloaded, nil
}

func (m *Manager) ModelPath(spec ModelSpec) string {
	filename := spec.Filename
	if filename == "" {
		filename = filepath.Base(spec.URL)
	}
	return filepath.Join(m.dir, filename)
}

func (m *Manager) IsDownloaded(spec ModelSpec) bool {
	info, err := os.Stat(m.ModelPath(spec))
	return err == nil && info.Size() > 0
}

func (m *Manager) DownloadByID(ctx context.Context, id string, onProgress func(Progress)) (DownloadedModel, error) {
	spec, ok := Find(id)
	if !ok {
		return DownloadedModel{}, fmt.Errorf("unknown model id %q", id)
	}
	return m.Download(ctx, spec, onProgress)
}

func (m *Manager) Download(ctx context.Context, spec ModelSpec, onProgress func(Progress)) (DownloadedModel, error) {
	if strings.TrimSpace(spec.URL) == "" {
		return DownloadedModel{}, errors.New("model URL is empty")
	}
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return DownloadedModel{}, err
	}

	targetPath := m.ModelPath(spec)
	if info, err := os.Stat(targetPath); err == nil && info.Size() > 0 {
		return DownloadedModel{Spec: spec, Path: targetPath, SizeBytes: info.Size()}, nil
	}

	tempPath := targetPath + ".part"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.URL, nil)
	if err != nil {
		return DownloadedModel{}, err
	}

	client := m.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return DownloadedModel{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DownloadedModel{}, fmt.Errorf("download failed with %s", resp.Status)
	}

	out, err := os.Create(tempPath)
	if err != nil {
		return DownloadedModel{}, err
	}
	defer out.Close()

	counter := &progressWriter{
		writer:     out,
		totalBytes: resp.ContentLength,
		onProgress: onProgress,
	}
	if _, err := io.Copy(counter, resp.Body); err != nil {
		_ = os.Remove(tempPath)
		return DownloadedModel{}, err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tempPath)
		return DownloadedModel{}, err
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return DownloadedModel{}, err
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		return DownloadedModel{}, err
	}
	if onProgress != nil {
		onProgress(Progress{WrittenBytes: info.Size(), TotalBytes: resp.ContentLength, Done: true})
	}
	return DownloadedModel{Spec: spec, Path: targetPath, SizeBytes: info.Size()}, nil
}

type progressWriter struct {
	writer       io.Writer
	writtenBytes int64
	totalBytes   int64
	onProgress   func(Progress)
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.writtenBytes += int64(n)
	if w.onProgress != nil {
		w.onProgress(Progress{WrittenBytes: w.writtenBytes, TotalBytes: w.totalBytes})
	}
	return n, err
}
