// Package config handles configuration management for PaddleOCR CLI.
//
// Config file search order:
//  1. Current directory (./.paddleocr_cli.yaml)
//  2. Project root (alongside .claude/ directory)
//  3. User config directory (~/.config/paddleocr_cli/config.yaml)
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	ConfigFilename = ".paddleocr_cli.yaml"
	UserConfigDir  = ".config/paddleocr_cli"
	UserConfigFile = "config.yaml"
)

// PaddleOCRConfig holds the PaddleOCR API configuration.
type PaddleOCRConfig struct {
	ServerURL   string `yaml:"server_url"`
	AccessToken string `yaml:"access_token"`
}

// Config is the main configuration structure.
type Config struct {
	PaddleOCR PaddleOCRConfig `yaml:"paddleocr"`
}

// New creates a new empty Config.
func New() *Config {
	return &Config{}
}

// IsConfigured checks if the configuration has required fields set.
func (c *Config) IsConfigured() bool {
	return c.PaddleOCR.ServerURL != "" && c.PaddleOCR.AccessToken != ""
}

// GetScriptDir returns the directory of the current executable.
func GetScriptDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// GetProjectRoot finds the project root by looking for .claude/ directory.
func GetProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	home, _ := os.UserHomeDir()
	current := cwd

	for {
		claudeDir := filepath.Join(current, ".claude")
		if info, err := os.Stat(claudeDir); err == nil && info.IsDir() {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current || current == home {
			break
		}
		current = parent
	}

	return ""
}

// FindConfig searches for configuration file in standard locations.
func FindConfig() string {
	searchPaths := []string{}

	// 1. Current directory
	if cwd, err := os.Getwd(); err == nil {
		searchPaths = append(searchPaths, filepath.Join(cwd, ConfigFilename))
	}

	// 2. Project root
	if projectRoot := GetProjectRoot(); projectRoot != "" {
		searchPaths = append(searchPaths, filepath.Join(projectRoot, ConfigFilename))
	}

	// 3. User config directory
	if home, err := os.UserHomeDir(); err == nil {
		searchPaths = append(searchPaths, filepath.Join(home, UserConfigDir, UserConfigFile))
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// Load loads configuration from a file or searches default locations.
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = FindConfig()
	}

	if configPath == "" {
		return New(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return nil, err
	}

	config := New()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// Save saves configuration to a file.
func Save(config *Config, configPath string) error {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir := filepath.Join(home, UserConfigDir)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}
		configPath = filepath.Join(configDir, UserConfigFile)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// GetConfigLocations returns all possible config locations with their status.
func GetConfigLocations() []struct {
	Description string
	Path        string
	Exists      bool
} {
	var locations []struct {
		Description string
		Path        string
		Exists      bool
	}

	// 1. Current directory
	if cwd, err := os.Getwd(); err == nil {
		path := filepath.Join(cwd, ConfigFilename)
		_, err := os.Stat(path)
		locations = append(locations, struct {
			Description string
			Path        string
			Exists      bool
		}{"Current directory", path, err == nil})
	}

	// 2. Project root
	if projectRoot := GetProjectRoot(); projectRoot != "" {
		path := filepath.Join(projectRoot, ConfigFilename)
		_, err := os.Stat(path)
		locations = append(locations, struct {
			Description string
			Path        string
			Exists      bool
		}{"Project root", path, err == nil})
	} else {
		locations = append(locations, struct {
			Description string
			Path        string
			Exists      bool
		}{"Project root", "(not found)", false})
	}

	// 3. User config
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, UserConfigDir, UserConfigFile)
		_, err := os.Stat(path)
		locations = append(locations, struct {
			Description string
			Path        string
			Exists      bool
		}{"User config", path, err == nil})
	}

	return locations
}

// GetSavePath returns the save path based on scope.
func GetSavePath(scope string) (string, error) {
	switch scope {
	case "local":
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ConfigFilename), nil
	case "project":
		projectRoot := GetProjectRoot()
		if projectRoot == "" {
			return "", os.ErrNotExist
		}
		return filepath.Join(projectRoot, ConfigFilename), nil
	default: // "user"
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, UserConfigDir, UserConfigFile), nil
	}
}
