package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadValidConfig(t *testing.T) {
	path := writeTempConfig(t, `{
		"version": "1.0",
		"columns": 3,
		"tiles": [
			{"type": "UPTIME", "label": "Blog", "params": {"url": "https://example.com"}},
			{"type": "uptime", "params": {"url": "http://example.org", "method": "HEAD", "interval": 10, "timeout": 500, "expectedStatus": [200, 301], "keyword": "hello"}}
		]
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Columns != 3 {
		t.Errorf("Columns = %d, want 3", cfg.Columns)
	}
	if len(cfg.Tiles) != 2 {
		t.Fatalf("len(Tiles) = %d, want 2", len(cfg.Tiles))
	}

	first := cfg.Tiles[0]
	if first.ID == "" {
		t.Error("first tile ID should be computed")
	}
	if first.Params.TimeoutMs != DefaultTimeoutMs {
		t.Errorf("TimeoutMs = %d, want default %d", first.Params.TimeoutMs, DefaultTimeoutMs)
	}
	if first.Params.IntervalS != DefaultIntervalS {
		t.Errorf("IntervalS = %d, want default %d", first.Params.IntervalS, DefaultIntervalS)
	}

	second := cfg.Tiles[1]
	if second.Type != TypeUptime {
		t.Errorf("Type = %q, want %q", second.Type, TypeUptime)
	}
	if second.Params.Method != "HEAD" {
		t.Errorf("Method = %q, want HEAD", second.Params.Method)
	}
	if second.ID == first.ID {
		t.Error("distinct tiles must have distinct IDs")
	}
}

func TestLoadDefaults(t *testing.T) {
	path := writeTempConfig(t, `{"tiles": [{"type": "UPTIME", "params": {"url": "https://example.com"}}]}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != "1.0" {
		t.Errorf("Version = %q, want 1.0", cfg.Version)
	}
	if cfg.Columns != 4 {
		t.Errorf("Columns = %d, want 4", cfg.Columns)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"invalid json", `{"tiles": [`},
		{"no tiles", `{"tiles": []}`},
		{"unknown type", `{"tiles": [{"type": "PING", "params": {"url": "https://example.com"}}]}`},
		{"missing url", `{"tiles": [{"type": "UPTIME", "params": {}}]}`},
		{"bad scheme", `{"tiles": [{"type": "UPTIME", "params": {"url": "ftp://example.com"}}]}`},
		{"bad method", `{"tiles": [{"type": "UPTIME", "params": {"url": "https://example.com", "method": "TELEPORT"}}]}`},
		{"interval too low", `{"tiles": [{"type": "UPTIME", "params": {"url": "https://example.com", "interval": 1}}]}`},
		{"timeout too high", `{"tiles": [{"type": "UPTIME", "params": {"url": "https://example.com", "timeout": 60000}}]}`},
		{"bad status", `{"tiles": [{"type": "UPTIME", "params": {"url": "https://example.com", "expectedStatus": [42]}}]}`},
		{"too many columns", `{"columns": 50, "tiles": [{"type": "UPTIME", "params": {"url": "https://example.com"}}]}`},
		{"duplicate tiles", `{"tiles": [
			{"type": "UPTIME", "params": {"url": "https://example.com"}},
			{"type": "UPTIME", "params": {"url": "https://example.com"}}
		]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempConfig(t, tt.content)
			if _, err := Load(path); err == nil {
				t.Errorf("Load() should fail for %s", tt.name)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Error("Load() should fail for a missing file")
	}
}
