package runner

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const defaultPort = 39281

type Manager struct {
	dir    string
	client *http.Client
}

type RunningServer struct {
	Endpoint string
	Model    string
	LogPath  string
	Cmd      *exec.Cmd
	logFile  *os.File
	done     chan error
}

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func DefaultDir() string {
	return filepath.Join(".gofaux", "runners", "llama.cpp")
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

func (m *Manager) Dir() string {
	return m.dir
}

func (m *Manager) ServerPath() string {
	exe := "llama-server"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	var found string
	_ = filepath.WalkDir(m.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || found != "" {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), exe) {
			found = path
		}
		return nil
	})
	return found
}

func (m *Manager) Installed() bool {
	path := m.ServerPath()
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (m *Manager) InstallLatest(ctx context.Context) error {
	if m.Installed() {
		return nil
	}
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return err
	}

	rel, err := m.latestRelease(ctx)
	if err != nil {
		return err
	}
	selected, ok := selectAsset(rel.Assets)
	if !ok {
		return errors.New("no compatible llama.cpp server release asset found")
	}

	archivePath := filepath.Join(m.dir, selected.Name)
	if err := m.download(ctx, selected.BrowserDownloadURL, archivePath); err != nil {
		return err
	}
	if err := extractArchive(archivePath, m.dir); err != nil {
		return err
	}
	_ = os.Remove(archivePath)
	if !m.Installed() {
		return errors.New("llama-server was not found after extracting runner archive")
	}
	return nil
}

func (m *Manager) Start(ctx context.Context, modelPath string) (RunningServer, error) {
	if strings.TrimSpace(modelPath) == "" {
		return RunningServer{}, errors.New("model path is required")
	}
	modelPath, err := filepath.Abs(modelPath)
	if err != nil {
		return RunningServer{}, fmt.Errorf("resolve model path: %w", err)
	}
	if _, err := os.Stat(modelPath); err != nil {
		return RunningServer{}, fmt.Errorf("model file is not available: %w", err)
	}
	if err := m.InstallLatest(ctx); err != nil {
		return RunningServer{}, err
	}

	serverPath := m.ServerPath()
	if serverPath == "" {
		return RunningServer{}, errors.New("llama-server executable is missing")
	}
	serverPath, err = filepath.Abs(serverPath)
	if err != nil {
		return RunningServer{}, fmt.Errorf("resolve runner path: %w", err)
	}
	if _, err := os.Stat(serverPath); err != nil {
		return RunningServer{}, fmt.Errorf("llama-server executable is not available: %w", err)
	}

	port, err := freePort(defaultPort)
	if err != nil {
		return RunningServer{}, err
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port)
	cmd := exec.Command(serverPath,
		"--model", modelPath,
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", port),
		"--ctx-size", "2048",
		"--parallel", "1",
		"--batch-size", "512",
		"--ubatch-size", "128",
		"--cache-ram", "0",
		"--no-cache-prompt",
		"--flash-attn", "off",
		"--gpu-layers", "0",
		"--no-kv-offload",
		"--no-op-offload",
	)
	cmd.Dir = filepath.Dir(serverPath)
	logPath := filepath.Join(m.dir, "llama-server.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		_, _ = fmt.Fprintf(logFile, "\n--- GoFaux managed runner start %s ---\nmodel=%s\nport=%d\n", time.Now().Format(time.RFC3339), modelPath, port)
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return RunningServer{}, err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	if err := waitForServer(ctx, healthURL, 90*time.Second, done); err != nil {
		_ = cmd.Process.Kill()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
		if logFile != nil {
			_ = logFile.Close()
		}
		return RunningServer{}, fmt.Errorf("%w; see %s", err, logPath)
	}

	return RunningServer{
		Endpoint: endpoint,
		Model:    filepath.Base(modelPath),
		LogPath:  logPath,
		Cmd:      cmd,
		logFile:  logFile,
		done:     done,
	}, nil
}

func (s RunningServer) Stop() {
	if s.Cmd == nil || s.Cmd.Process == nil {
		return
	}
	_ = s.Cmd.Process.Kill()
	if s.done != nil {
		select {
		case <-s.done:
		case <-time.After(3 * time.Second):
		}
	} else {
		_, _ = s.Cmd.Process.Wait()
	}
	if s.logFile != nil {
		_ = s.logFile.Close()
	}
}

func (m *Manager) latestRelease(ctx context.Context) (release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/ggml-org/llama.cpp/releases/latest", nil)
	if err != nil {
		return release{}, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return release{}, fmt.Errorf("GitHub release lookup failed with %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return release{}, err
	}
	return rel, nil
}

func (m *Manager) download(ctx context.Context, url, target string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("runner download failed with %s", resp.Status)
	}
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func selectAsset(assets []asset) (asset, bool) {
	var required []string
	switch runtime.GOOS {
	case "windows":
		required = []string{"win", "cpu", "x64", ".zip"}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			required = []string{"macos", "arm64", ".tar.gz"}
		} else {
			required = []string{"macos", "x64", ".tar.gz"}
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			required = []string{"ubuntu", "arm64"}
		} else {
			required = []string{"ubuntu", "x64"}
		}
	default:
		return asset{}, false
	}

	for _, item := range assets {
		name := strings.ToLower(item.Name)
		if containsAll(name, required) && !strings.Contains(name, "cuda") && !strings.Contains(name, "vulkan") {
			return item, true
		}
	}
	for _, item := range assets {
		name := strings.ToLower(item.Name)
		if containsAll(name, required[:len(required)-1]) && !strings.Contains(name, "cuda") && !strings.Contains(name, "vulkan") {
			return item, true
		}
	}
	if runtime.GOOS == "windows" {
		for _, item := range assets {
			name := strings.ToLower(item.Name)
			if containsAll(name, []string{"win", "x64", ".zip"}) &&
				!strings.Contains(name, "cuda") &&
				!strings.Contains(name, "vulkan") {
				return item, true
			}
		}
	}
	return asset{}, false
}

func containsAll(value string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}

func extractArchive(path, targetDir string) error {
	switch {
	case strings.HasSuffix(strings.ToLower(path), ".zip"):
		return extractZip(path, targetDir)
	case strings.HasSuffix(strings.ToLower(path), ".tar.gz"):
		return extractTarGz(path, targetDir)
	default:
		return fmt.Errorf("unsupported runner archive: %s", filepath.Base(path))
	}
}

func extractZip(path, targetDir string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		target, ok := safeJoin(targetDir, file.Name)
		if !ok {
			continue
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		if err := writeFile(target, src, file.FileInfo().Mode()); err != nil {
			_ = src.Close()
			return err
		}
		_ = src.Close()
	}
	return nil
}

func extractTarGz(path, targetDir string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target, ok := safeJoin(targetDir, header.Name)
		if !ok {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := writeFile(target, tr, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
}

func safeJoin(base, name string) (string, bool) {
	target := filepath.Clean(filepath.Join(base, name))
	baseClean := filepath.Clean(base)
	rel, err := filepath.Rel(baseClean, target)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", false
	}
	return target, true
}

func writeFile(path string, src io.Reader, mode os.FileMode) error {
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

func freePort(start int) (int, error) {
	for port := start; port < start+100; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			_ = listener.Close()
			return port, nil
		}
	}
	return 0, errors.New("no free local port found")
}

func waitForServer(ctx context.Context, url string, timeout time.Duration, done <-chan error) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: time.Second}
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err != nil {
				return fmt.Errorf("local inference runner exited during startup: %w", err)
			}
			return errors.New("local inference runner exited during startup")
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			if err != nil {
				return fmt.Errorf("local inference runner exited during startup: %w", err)
			}
			return errors.New("local inference runner exited during startup")
		case <-time.After(500 * time.Millisecond):
		}
	}
	return errors.New("local inference runner did not become ready in time")
}
