package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// The JSON file structure
type Config struct {
	DBUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

// File name constant
const configFileName = ".gatorconfig.json"

// Returns: /home/you/.gatorconfig.json
func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}
	return filepath.Join(home, configFileName), nil
}

// Reads ~/.gatorconfig.json and unmarshals JSON → Config
func Read() (Config, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(fileBytes, &cfg); err != nil {
		return Config{}, fmt.Errorf("error unmarshaling config json: %w", err)
	}

	return cfg, nil
}

// Writes Config back to ~/.gatorconfig.json
func write(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config json: %w", err)
	}

	return os.WriteFile(path, jsonBytes, 0644)
}

// Sets the user and persists the change
func (c *Config) SetUser(name string) error {
	c.CurrentUserName = name
	return write(*c)
}
