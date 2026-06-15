package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"GoFaux/internal/assistant"
	"GoFaux/internal/mock"
	"GoFaux/internal/modelhub"
	"GoFaux/internal/runner"
)

type Config struct {
	MockConfigPath string
	ServerAddr     string
	SettingsPath   string
	ModelDir       string
	RunnerDir      string
	Assistant      assistant.Config
}

type FileSettings struct {
	MockConfigPath string           `json:"mock_config_path,omitempty"`
	ServerAddr     string           `json:"server_addr,omitempty"`
	ModelDir       string           `json:"model_dir,omitempty"`
	RunnerDir      string           `json:"runner_dir,omitempty"`
	Assistant      assistant.Config `json:"assistant,omitempty"`
}

func FromEnv() Config {
	cfg := Config{
		MockConfigPath: envOrDefault("GOFAUX_CONFIG", mock.DefaultConfigPath()),
		ServerAddr:     envOrDefault("GOFAUX_ADDR", ":8080"),
		SettingsPath:   envOrDefault("GOFAUX_SETTINGS", DefaultSettingsPath()),
		ModelDir:       envOrDefault("GOFAUX_MODEL_DIR", modelhub.DefaultDir()),
		RunnerDir:      envOrDefault("GOFAUX_RUNNER_DIR", runner.DefaultDir()),
		Assistant: assistant.Config{
			Provider: os.Getenv("GOFAUX_AI_PROVIDER"),
			Model:    os.Getenv("GOFAUX_AI_MODEL"),
			Endpoint: os.Getenv("GOFAUX_AI_ENDPOINT"),
			APIKey:   os.Getenv("GOFAUX_AI_API_KEY"),
		},
	}

	cfg = mergeSettingsFile(cfg)
	cfg = applyEnvOverrides(cfg)
	return cfg
}

func DefaultSettingsPath() string {
	return ".gofaux.settings.json"
}

func SaveSettings(cfg Config) error {
	settings := FileSettings{
		MockConfigPath: cfg.MockConfigPath,
		ServerAddr:     cfg.ServerAddr,
		ModelDir:       cfg.ModelDir,
		RunnerDir:      cfg.RunnerDir,
		Assistant: assistant.Config{
			Provider: cfg.Assistant.Provider,
			Model:    cfg.Assistant.Model,
			Endpoint: cfg.Assistant.Endpoint,
		},
	}

	if cfg.SettingsPath == "" {
		cfg.SettingsPath = DefaultSettingsPath()
	}
	dir := filepath.Dir(cfg.SettingsPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(cfg.SettingsPath, content, 0o644)
}

func mergeSettingsFile(cfg Config) Config {
	content, err := os.ReadFile(cfg.SettingsPath)
	if err != nil || len(content) == 0 {
		return cfg
	}

	var settings FileSettings
	if err := json.Unmarshal(content, &settings); err != nil {
		return cfg
	}
	if settings.MockConfigPath != "" {
		cfg.MockConfigPath = settings.MockConfigPath
	}
	if settings.ServerAddr != "" {
		cfg.ServerAddr = settings.ServerAddr
	}
	if settings.ModelDir != "" {
		cfg.ModelDir = settings.ModelDir
	}
	if settings.RunnerDir != "" {
		cfg.RunnerDir = settings.RunnerDir
	}
	if settings.Assistant.Provider != "" {
		cfg.Assistant.Provider = settings.Assistant.Provider
	}
	if settings.Assistant.Model != "" {
		cfg.Assistant.Model = settings.Assistant.Model
	}
	if settings.Assistant.Endpoint != "" {
		cfg.Assistant.Endpoint = settings.Assistant.Endpoint
	}
	return cfg
}

func applyEnvOverrides(cfg Config) Config {
	if value, ok := os.LookupEnv("GOFAUX_CONFIG"); ok && strings.TrimSpace(value) != "" {
		cfg.MockConfigPath = strings.TrimSpace(value)
	}
	if value, ok := os.LookupEnv("GOFAUX_ADDR"); ok && strings.TrimSpace(value) != "" {
		cfg.ServerAddr = strings.TrimSpace(value)
	}
	if value, ok := os.LookupEnv("GOFAUX_MODEL_DIR"); ok && strings.TrimSpace(value) != "" {
		cfg.ModelDir = strings.TrimSpace(value)
	}
	if value, ok := os.LookupEnv("GOFAUX_RUNNER_DIR"); ok && strings.TrimSpace(value) != "" {
		cfg.RunnerDir = strings.TrimSpace(value)
	}
	if value, ok := os.LookupEnv("GOFAUX_AI_PROVIDER"); ok {
		cfg.Assistant.Provider = value
	}
	if value, ok := os.LookupEnv("GOFAUX_AI_MODEL"); ok {
		cfg.Assistant.Model = value
	}
	if value, ok := os.LookupEnv("GOFAUX_AI_ENDPOINT"); ok {
		cfg.Assistant.Endpoint = value
	}
	if value, ok := os.LookupEnv("GOFAUX_AI_API_KEY"); ok {
		cfg.Assistant.APIKey = value
	}
	return cfg
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
