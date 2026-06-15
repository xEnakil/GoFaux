package modelhub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestDownloadModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("fake-model"))
	}))
	defer server.Close()

	manager := NewManager(t.TempDir())
	spec := ModelSpec{
		ID:       "fake",
		Name:     "Fake model",
		URL:      server.URL + "/model.gguf",
		Filename: "model.gguf",
	}

	model, err := manager.Download(context.Background(), spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if model.SizeBytes != int64(len("fake-model")) {
		t.Fatalf("unexpected size: %d", model.SizeBytes)
	}
	if model.Path != filepath.Join(manager.Dir(), "model.gguf") {
		t.Fatalf("unexpected path: %s", model.Path)
	}
	if !manager.IsDownloaded(spec) {
		t.Fatal("expected model to be marked as downloaded")
	}
}
