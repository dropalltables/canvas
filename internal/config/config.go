package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
}

var ErrNotConfigured = errors.New("canvas not configured: run 'canvas auth login'")

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "canvas", "config.json"), nil
}

func Load() (*Config, error) {
	baseURL := os.Getenv("CANVAS_BASE_URL")
	token := os.Getenv("CANVAS_TOKEN")

	if baseURL != "" && token != "" {
		return &Config{BaseURL: baseURL, Token: token}, nil
	}

	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotConfigured
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.BaseURL == "" || cfg.Token == "" {
		return nil, ErrNotConfigured
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func Path() string {
	p, _ := configPath()
	return p
}
