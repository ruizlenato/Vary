package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"vary/internal/storage"
)

type Mode string

const (
	ModeStable Mode = "stable"
	ModeDev    Mode = "dev"
)

type Config struct {
	Mode               Mode   `json:"mode"`
	CustomKeystorePath string `json:"customKeystorePath,omitempty"`
}

func Default() *Config {
	return &Config{
		Mode: ModeStable,
	}
}

func Load() (*Config, error) {
	appDir, err := storage.AppDataDir("vary")
	if err != nil {
		return nil, fmt.Errorf("failed to get app data dir: %w", err)
	}

	configPath := filepath.Join(appDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), nil
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	appDir, err := storage.AppDataDir("vary")
	if err != nil {
		return fmt.Errorf("failed to get app data dir: %w", err)
	}

	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to create app dir: %w", err)
	}

	configPath := filepath.Join(appDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (c *Config) IsDev() bool {
	return c.Mode == ModeDev
}
