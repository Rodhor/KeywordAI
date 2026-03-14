package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

type Config struct {
	GeminiAPIKey     string
	GoogleAuthAPIKey string
}

func ConfigPath() (string, error) {
	// Create config path if it does not exist and return the full path to the config file.
	configBasePath, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve user config directory: %w", err)
	}
	configPath := path.Join(configBasePath, "keywordai", "config.json")
	if err := os.MkdirAll(path.Dir(configPath), 0755); err != nil {
		return "", fmt.Errorf("Failed to create config directory: %w", err)
	}
	return configPath, nil
}

func Load() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file does not exist, return empty config (not an error).
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	// Load config from file.
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %w", err)
	}

	cfg := &Config{}
	err = json.Unmarshal(configFile, &cfg)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal config file: %w", err)
	}

	if cfg.GeminiAPIKey == "" {
		fmt.Println("Warning: Gemini API key is missing in config. AI features will not work.")
		cfg.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	}

	if cfg.GoogleAuthAPIKey == "" {
		fmt.Println("Warning: Google Auth API key is missing in config. Google Sheets integration will not work.")
		cfg.GoogleAuthAPIKey = os.Getenv("GOOGLE_AUTH_API_KEY")
	}

	return cfg, nil
}

func (cfg *Config) Save() error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}

	configData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to marshal config: %w", err)
	}

	err = os.WriteFile(configPath, configData, 0600)
	if err != nil {
		return fmt.Errorf("Failed to write config file: %w", err)
	}

	return nil
}
