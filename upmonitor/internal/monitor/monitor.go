// Package monitor runs periodic uptime checks for every configured tile,
// keeps a bounded history per tile and broadcasts state updates to subscribers.
package monitor

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Alangave/monitoror/upmonitor/internal/config"
)

// CheckStatus is the outcome of a single check.
type CheckStatus string

const (
	StatusSuccess CheckStatus = "SUCCESS"
	StatusFailed  CheckStatus = "FAILED"
	StatusPending CheckStatus = "PENDING"
)

// Check is a single check result stored in the history ring buffer.
type Check struct {
	At         int64       `json:"at"` // unix milliseconds
	Status     CheckStatus `json:"status"`
	DurationMs int64       `json:"durationMs"`
	Message    string      `json:"message,omitempty"`
}

// TileState is the public state of a tile, sent to the UI.
type TileState struct {
	ID             string      `json:"id"`
	Type           string      `json:"type"`
	Label          string      `json:"label"`
	Status         CheckStatus `json:"status"`
	Message        string      `json:"message,omitempty"`
	ResponseTimeMs int64       `json:"responseTimeMs"`
	Uptime         float64     `json:"uptime"` // 0..1 over the retained history
	Since          int64       `json:"since"`  // unix ms, last status change
	History        []Check     `json:"history"`
}

// Options tunes the monitor.
type Options struct {
	HistorySize int
	UserAgent   string
}

// DefaultOptions returns the default monitor options.
func DefaultOptions() Options {
	return Options{
		HistorySize: config.HistorySize,
		UserAgent:   "upmonitor/1.0",
	}
}

type tile struct {
	cfg     config.TileConfig
	client  *http.Client
	history []Check
	status  CheckStatus
	message string
	since   time.Time
}

// Monitor owns all tile checkers and the subscriber registry.
type Monitor struct {
	cfg     *config.Config
	options Options

	mu          sync.RWMutex
	tiles       map[string]*tile
	subscribers map[chan []byte]struct{}
}

// New creates a Monitor for the given configuration.
// The configuration is validated and normalized (defaults, tile IDs) if not already done.
func New(cfg *config.Config, options Options) (*Monitor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	m := &Monitor{
		cfg:         cfg,
		options:     options,
		tiles:       make(map[string]*tile, len(cfg.Tiles)),
		subscribers: make(map[chan []byte]struct{}),
	}
	for _, tileCfg := range cfg.Tiles {
		m.tiles[tileCfg.ID] = &tile{
			cfg:    tileCfg,
			client: buildClient(tileCfg.Params),
			status: StatusPending,
			since:  time.Now(),
		}
	}
	return m, nil
}

func buildClient(params config.Params) *http.Client {
	transport := &http.Transport{
		DisableKeepAlives: true,
	}
	if params.InsecureTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in via configuration
	}
	return &http.Client{Transport: transport}
}

// Start launches one checker goroutine per tile. It returns immediately.
func (m *Monitor) Start(ctx context.Context) {
	for _, t := range m.tiles {
		go m.runTile(ctx, t)
	}
}

func (m *Monitor) runTile(ctx context.Context, t *tile) {
	interval := time.Duration(t.cfg.Params.IntervalS) * time.Second
	m.check(ctx, t)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.check(ctx, t)
		}
	}
}

func (m *Monitor) check(ctx context.Context, t *tile) {
	check := m.performCheck(ctx, t)

	m.mu.Lock()
	t.history = append(t.history, check)
	if len(t.history) > m.options.HistorySize {
		t.history = t.history[len(t.history)-m.options.HistorySize:]
	}
	previous := t.status
	t.status = check.Status
	t.message = check.Message
	if previous != check.Status {
		t.since = time.UnixMilli(check.At)
	}
	state := m.tileStateLocked(t)
	m.mu.Unlock()

	m.broadcast(state)
}

func (m *Monitor) performCheck(ctx context.Context, t *tile) Check {
	params := t.cfg.Params
	timeout := time.Duration(params.TimeoutMs) * time.Millisecond

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, params.Method, params.URL, nil)
	if err != nil {
		return failedCheck(err.Error())
	}
	req.Header.Set("User-Agent", m.options.UserAgent)

	start := time.Now()
	resp, err := t.client.Do(req)
	if err != nil {
		return failedCheck(err.Error())
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	if !isExpectedStatus(resp.StatusCode, params.ExpectedStatus) {
		return Check{
			At:         start.UnixMilli(),
			Status:     StatusFailed,
			DurationMs: duration.Milliseconds(),
			Message:    "unexpected status code " + resp.Status,
		}
	}

	if params.Keyword != "" {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return failedCheck("unable to read response body: " + err.Error())
		}
		if !strings.Contains(string(body), params.Keyword) {
			return Check{
				At:         start.UnixMilli(),
				Status:     StatusFailed,
				DurationMs: duration.Milliseconds(),
				Message:    "keyword not found in response body",
			}
		}
	}

	return Check{
		At:         start.UnixMilli(),
		Status:     StatusSuccess,
		DurationMs: duration.Milliseconds(),
	}
}

func failedCheck(message string) Check {
	return Check{At: time.Now().UnixMilli(), Status: StatusFailed, Message: message}
}

func isExpectedStatus(status int, expected []int) bool {
	for _, code := range expected {
		if status == code {
			return true
		}
	}
	return false
}

// Snapshot returns the current state of every tile.
func (m *Monitor) Snapshot() []TileState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make([]TileState, 0, len(m.tiles))
	for _, tileCfg := range m.cfg.Tiles {
		if t, ok := m.tiles[tileCfg.ID]; ok {
			states = append(states, m.tileStateLocked(t))
		}
	}
	return states
}

func (m *Monitor) tileStateLocked(t *tile) TileState {
	history := make([]Check, len(t.history))
	copy(history, t.history)

	var successes int
	var lastDuration int64
	for _, check := range history {
		if check.Status == StatusSuccess {
			successes++
		}
		lastDuration = check.DurationMs
	}

	uptime := 0.0
	if len(history) > 0 {
		uptime = float64(successes) / float64(len(history))
	}

	return TileState{
		ID:             t.cfg.ID,
		Type:           t.cfg.Type,
		Label:          t.cfg.Label,
		Status:         t.status,
		Message:        t.message,
		ResponseTimeMs: lastDuration,
		Uptime:         uptime,
		Since:          t.since.UnixMilli(),
		History:        history,
	}
}

// Subscribe registers a new SSE subscriber and returns its channel and a cancel func.
func (m *Monitor) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 64)
	m.mu.Lock()
	m.subscribers[ch] = struct{}{}
	m.mu.Unlock()

	return ch, func() {
		m.mu.Lock()
		delete(m.subscribers, ch)
		m.mu.Unlock()
	}
}

func (m *Monitor) broadcast(state TileState) {
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("unable to marshal tile state: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	for ch := range m.subscribers {
		select {
		case ch <- data:
		default:
			// Slow subscriber: drop the update rather than blocking checkers.
		}
	}
}
