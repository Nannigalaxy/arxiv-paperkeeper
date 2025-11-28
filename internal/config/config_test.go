package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaultConfig(t *testing.T) {
	// Load config with non-existent file (should use defaults)
	cfg, err := Load("nonexistent.yaml")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test defaults
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", cfg.Server.Host)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Database.Path != "./data/arxiv.db" {
		t.Errorf("Expected default DB path './data/arxiv.db', got '%s'", cfg.Database.Path)
	}

	if cfg.ArXiv.MaxResults != 100 {
		t.Errorf("Expected default max results 100, got %d", cfg.ArXiv.MaxResults)
	}

	if cfg.ArXiv.FetchInterval != 24*time.Hour {
		t.Errorf("Expected default fetch interval 24h, got %v", cfg.ArXiv.FetchInterval)
	}

	if cfg.UI.PageSize != 20 {
		t.Errorf("Expected default page size 20, got %d", cfg.UI.PageSize)
	}
}

func TestLoadFromYAML(t *testing.T) {
	// Create temporary YAML file
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	yamlContent := `
server:
  host: "127.0.0.1"
  port: 9090

database:
  path: "/tmp/test.db"

arxiv:
  categories:
    - "cs.AI"
    - "cs.CV"
  max_results: 50
  fetch_interval: 12h

ui:
  page_size: 10
`

	if _, err := tmpfile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpfile.Close()

	// Load config
	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test loaded values
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host '127.0.0.1', got '%s'", cfg.Server.Host)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}

	if cfg.Database.Path != "/tmp/test.db" {
		t.Errorf("Expected DB path '/tmp/test.db', got '%s'", cfg.Database.Path)
	}

	if cfg.ArXiv.MaxResults != 50 {
		t.Errorf("Expected max results 50, got %d", cfg.ArXiv.MaxResults)
	}

	if len(cfg.ArXiv.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(cfg.ArXiv.Categories))
	}

	if cfg.UI.PageSize != 10 {
		t.Errorf("Expected page size 10, got %d", cfg.UI.PageSize)
	}
}

func TestEnvironmentVariableOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("SERVER_HOST", "192.168.1.1")
	os.Setenv("SERVER_PORT", "3000")
	os.Setenv("DB_PATH", "/custom/path.db")
	os.Setenv("ARXIV_MAX_RESULTS", "200")
	os.Setenv("UI_PAGE_SIZE", "50")

	defer func() {
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DB_PATH")
		os.Unsetenv("ARXIV_MAX_RESULTS")
		os.Unsetenv("UI_PAGE_SIZE")
	}()

	// Load config (should use env vars)
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test env var overrides
	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("Expected host from env '192.168.1.1', got '%s'", cfg.Server.Host)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Expected port from env 3000, got %d", cfg.Server.Port)
	}

	if cfg.Database.Path != "/custom/path.db" {
		t.Errorf("Expected DB path from env '/custom/path.db', got '%s'", cfg.Database.Path)
	}

	if cfg.ArXiv.MaxResults != 200 {
		t.Errorf("Expected max results from env 200, got %d", cfg.ArXiv.MaxResults)
	}

	if cfg.UI.PageSize != 50 {
		t.Errorf("Expected page size from env 50, got %d", cfg.UI.PageSize)
	}
}

func TestAddress(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	addr := cfg.Address()
	expected := "localhost:8080"

	if addr != expected {
		t.Errorf("Expected address '%s', got '%s'", expected, addr)
	}
}
