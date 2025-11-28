package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	ArXiv    ArXivConfig    `yaml:"arxiv"`
	UI       UIConfig       `yaml:"ui"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host string `yaml:"host" env:"SERVER_HOST"`
	Port int    `yaml:"port" env:"SERVER_PORT"`
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Path string `yaml:"path" env:"DB_PATH"`
}

// ArXivConfig holds arXiv fetching settings
type ArXivConfig struct {
	Categories     []string      `yaml:"categories"`
	Keywords       []string      `yaml:"keywords"`
	MaxResults     int           `yaml:"max_results" env:"ARXIV_MAX_RESULTS"`
	FetchInterval  time.Duration `yaml:"fetch_interval" env:"ARXIV_FETCH_INTERVAL"`
	RateLimitDelay time.Duration `yaml:"rate_limit_delay"`
}

// UIConfig holds UI-related settings
type UIConfig struct {
	PageSize int `yaml:"page_size" env:"UI_PAGE_SIZE"`
}

// Load reads configuration from YAML file and environment variables
// Environment variables take precedence over YAML values
func Load(configPath string) (*Config, error) {
	// Default configuration
	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Path: "./data/arxiv.db",
		},
		ArXiv: ArXivConfig{
			Categories:     []string{"cs.AI", "cs.LG", "cs.CL"},
			Keywords:       []string{},
			MaxResults:     100,
			FetchInterval:  24 * time.Hour,
			RateLimitDelay: 3 * time.Second,
		},
		UI: UIConfig{
			PageSize: 20,
		},
	}

	// Load from YAML file if it exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// File doesn't exist, use defaults
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with environment variables
	if host := os.Getenv("SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			cfg.Server.Port = p
		}
	}
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}
	if maxResults := os.Getenv("ARXIV_MAX_RESULTS"); maxResults != "" {
		var m int
		if _, err := fmt.Sscanf(maxResults, "%d", &m); err == nil {
			cfg.ArXiv.MaxResults = m
		}
	}
	if pageSize := os.Getenv("UI_PAGE_SIZE"); pageSize != "" {
		var p int
		if _, err := fmt.Sscanf(pageSize, "%d", &p); err == nil {
			cfg.UI.PageSize = p
		}
	}

	return cfg, nil
}

// Address returns the server address in host:port format
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
