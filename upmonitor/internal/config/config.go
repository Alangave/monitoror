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
	TypeGroup  = "GROUP"
	TypeEmpty  = "EMPTY"

	DefaultTimeoutMs = 2000
	DefaultIntervalS = 30
	HistorySize      = 100
	MinIntervalS     = 5
	MaxTimeoutMs     = 30000
	MaxColumns       = 12
	MaxHeight        = 4
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
// GROUP tiles embed their sub-tiles in Tiles; EMPTY tiles are grid placeholders.
// Width and Height are the tile size in grid cells (defaults: 1x1, 2x1 for groups).
type TileConfig struct {
	Type   string       `json:"type"`
	Label  string       `json:"label,omitempty"`
	Params Params       `json:"params,omitempty"`
	Tiles  []TileConfig `json:"tiles,omitempty"`
	Width  int          `json:"width,omitempty"`
	Height int          `json:"height,omitempty"`

	// ID is the computed state key, filled by Validate/Load.
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

	for i := range c.Tiles {
		if err := c.Tiles[i].validate(i, c.Columns); err != nil {
			return fmt.Errorf("tile %d: %w", i, err)
		}
	}

	seen := make(map[string]bool)
	var walk func(tiles []TileConfig) error
	walk = func(tiles []TileConfig) error {
		for i := range tiles {
			tile := &tiles[i]
			if tile.Type == TypeEmpty {
				continue
			}
			if seen[tile.ID] {
				return fmt.Errorf("duplicate tile definition %q", tile.ID)
			}
			seen[tile.ID] = true
			if len(tile.Tiles) > 0 {
				if err := walk(tile.Tiles); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(c.Tiles); err != nil {
		return err
	}
	return nil
}

func (t *TileConfig) validate(index, columns int) error {
	switch {
	case strings.EqualFold(t.Type, TypeUptime):
		t.Type = TypeUptime
		if err := t.validateUptimeParams(); err != nil {
			return err
		}
	case strings.EqualFold(t.Type, TypeGroup):
		t.Type = TypeGroup
		if t.Label == "" {
			return fmt.Errorf("group tile requires a label")
		}
		if len(t.Tiles) == 0 {
			return fmt.Errorf("group tile requires at least one sub-tile")
		}
		for i := range t.Tiles {
			child := &t.Tiles[i]
			if !strings.EqualFold(child.Type, TypeUptime) {
				return fmt.Errorf("sub-tile %d: only %q tiles are supported inside groups", i, TypeUptime)
			}
			if err := child.validate(i, columns); err != nil {
				return fmt.Errorf("sub-tile %d: %w", i, err)
			}
		}
	case strings.EqualFold(t.Type, TypeEmpty):
		t.Type = TypeEmpty
		t.ID = fmt.Sprintf("empty-%d", index)
		return nil
	default:
		return fmt.Errorf("unknown tile type %q, supported types: %q, %q, %q", t.Type, TypeUptime, TypeGroup, TypeEmpty)
	}

	if err := t.validateSize(columns); err != nil {
		return err
	}

	t.ID = t.computeID()
	return nil
}

// validateSize normalizes the tile size in grid cells.
// GROUP tiles default to 2 cells wide, everything else to 1x1.
func (t *TileConfig) validateSize(columns int) error {
	if t.Type == TypeGroup && t.Width <= 0 {
		t.Width = 2
	}
	if t.Width <= 0 {
		t.Width = 1
	}
	if t.Width > columns {
		return fmt.Errorf("tile width %d exceeds columns count (%d)", t.Width, columns)
	}
	if t.Height <= 0 {
		t.Height = 1
	}
	if t.Height > MaxHeight {
		return fmt.Errorf("tile height must be lower or equal to %d", MaxHeight)
	}
	return nil
}

func (t *TileConfig) validateUptimeParams() error {
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
