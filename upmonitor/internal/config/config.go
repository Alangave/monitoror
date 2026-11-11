// Package config loads and validates the upmonitor wall configuration.
// The format is inspired by Monitoror: a version, a column count and a list of tiles.
package config

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	TypeUptime = "UPTIME"

	DefaultTimeoutMs = 2000
	DefaultIntervalS = 30
	HistorySize      = 100
	MinIntervalS     = 5
	MaxTimeoutMs     = 30000
	MaxColumns       = 12
)

// Params holds the UPTIME tile parameters.
type Params struct {
	URL            string `json:"url"`
	Method         string `json:"method,omitempty"`
	Keyword        string `json:"keyword,omitempty"`
	ExpectedStatus []int  `json:"expectedStatus,omitempty"`
	TimeoutMs      int    `json:"timeout,omitempty"`
	IntervalS      int    `json:"interval,omitempty"`
	InsecureTLS    bool   `json:"insecureTLS,omitempty"`
}

// TileConfig is a single tile declaration from the configuration file.
type TileConfig struct {
	Type   string `json:"type"`
	Label  string `json:"label,omitempty"`
	Params Params `json:"params"`

	// ID is the computed state key, filled by Load.
	ID string `json:"-"`
}

// Config is the whole wall configuration.
type Config struct {
	Version string       `json:"version"`
	Columns int          `json:"columns"`
	Tiles   []TileConfig `json:"tiles"`
}

// Load reads, validates and normalizes the configuration file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file: %w", err)
	}

	cfg := &Config{}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks the configuration and fills defaults and tile IDs.
// It is called by Load and must be called on programmatically built configs.
func (c *Config) Validate() error {
	if c.Version == "" {
		c.Version = "1.0"
	}
	if c.Columns <= 0 {
		c.Columns = 4
	}
	if c.Columns > MaxColumns {
		return fmt.Errorf("columns must be lower or equal to %d", MaxColumns)
	}
	if len(c.Tiles) == 0 {
		return fmt.Errorf("at least one tile is required")
	}

	seen := make(map[string]bool, len(c.Tiles))
	for i := range c.Tiles {
		tile := &c.Tiles[i]
		if err := tile.validate(); err != nil {
			return fmt.Errorf("tile %d: %w", i, err)
		}
		if seen[tile.ID] {
			return fmt.Errorf("tile %d: duplicate tile definition", i)
		}
		seen[tile.ID] = true
	}
	return nil
}

func (t *TileConfig) validate() error {
	if !strings.EqualFold(t.Type, TypeUptime) {
		return fmt.Errorf("unknown tile type %q, only %q is supported", t.Type, TypeUptime)
	}
	t.Type = TypeUptime

	p := &t.Params
	if p.URL == "" {
		return fmt.Errorf("missing params.url")
	}
	if !strings.HasPrefix(p.URL, "http://") && !strings.HasPrefix(p.URL, "https://") {
		return fmt.Errorf("params.url must start with http:// or https://")
	}

	p.Method = strings.ToUpper(strings.TrimSpace(p.Method))
	if p.Method == "" {
		p.Method = http.MethodGet
	}
	switch p.Method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
	default:
		return fmt.Errorf("unsupported params.method %q", p.Method)
	}

	if p.TimeoutMs <= 0 {
		p.TimeoutMs = DefaultTimeoutMs
	}
	if p.TimeoutMs > MaxTimeoutMs {
		return fmt.Errorf("params.timeout must be lower or equal to %dms", MaxTimeoutMs)
	}
	if p.IntervalS <= 0 {
		p.IntervalS = DefaultIntervalS
	}
	if p.IntervalS < MinIntervalS {
		return fmt.Errorf("params.interval must be greater or equal to %ds", MinIntervalS)
	}
	if len(p.ExpectedStatus) == 0 {
		p.ExpectedStatus = []int{200}
	}
	for _, status := range p.ExpectedStatus {
		if status < 100 || status > 599 {
			return fmt.Errorf("invalid expected status code %d", status)
		}
	}

	t.ID = t.computeID()
	return nil
}

// computeID builds a stable state key from the tile type and params, like Monitoror stateKeys.
func (t *TileConfig) computeID() string {
	normalized := *t
	normalized.ID = ""
	data, _ := json.Marshal(normalized)
	sum := md5.Sum(data)
	return strings.ToLower(t.Type) + "-" + hex.EncodeToString(sum[:8])
}
