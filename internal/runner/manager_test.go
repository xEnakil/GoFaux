package runner

import "testing"

func TestSelectAssetWindowsLikeName(t *testing.T) {
	assets := []asset{
		{Name: "llama-b9000-bin-win-cuda-12.4-x64.zip"},
		{Name: "llama-b9000-bin-win-cpu-x64.zip"},
		{Name: "llama-b9000-bin-ubuntu-x64.zip"},
		{Name: "llama-b9000-bin-ubuntu-arm64.zip"},
		{Name: "llama-b9000-bin-macos-arm64.tar.gz"},
		{Name: "llama-b9000-bin-macos-x64.tar.gz"},
	}
	selected, ok := selectAsset(assets)
	if !ok {
		t.Fatal("expected compatible asset")
	}
	if selected.Name == "llama-b9000-bin-win-cuda-12.4-x64.zip" {
		t.Fatalf("unexpected asset: %s", selected.Name)
	}
}

func TestSafeJoinRejectsTraversal(t *testing.T) {
	if _, ok := safeJoin("/tmp/base", "../evil"); ok {
		t.Fatal("expected traversal to be rejected")
	}
}
