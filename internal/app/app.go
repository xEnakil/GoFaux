package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"GoFaux/internal/cli"
	"GoFaux/internal/config"
	"GoFaux/internal/httpserver"
	"GoFaux/internal/mock"
)

func Run() error {
	cfg := config.FromEnv()

	store, err := mock.NewStore(cfg.MockConfigPath)
	if err != nil {
		return err
	}

	server := httpserver.New(store, cfg)
	if hasArg("--cli") {
		menu := cli.NewMenu(cfg, store, server)
		return menu.Run()
	}
	return runUI(server)
}

func runUI(server *httpserver.Server) error {
	if err := server.Start(); err != nil {
		return err
	}

	uiURL := server.UIURL()
	baseURL := strings.TrimSuffix(uiURL, "/_gofaux/ui/")
	fmt.Println("GoFaux 2.0 - local mock API studio")
	fmt.Println("Dashboard: " + uiURL)
	fmt.Println("Health: " + baseURL + "/_gofaux/health")
	fmt.Println("Use Ctrl+C to stop. Use --cli to open the terminal menu.")
	if !hasArg("--no-open") {
		openBrowser(uiURL)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return server.Stop(ctx)
}

func hasArg(flag string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return false
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
