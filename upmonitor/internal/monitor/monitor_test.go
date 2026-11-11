package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Alangave/monitoror/upmonitor/internal/config"
)

func testConfig(t *testing.T, url string, intervalS int) *config.Config {
	t.Helper()
	cfg := &config.Config{
		Version: "1.0",
		Columns: 2,
		Tiles: []config.TileConfig{
			{
				Type:  config.TypeUptime,
				Label: "Test",
				Params: config.Params{
					URL:       url,
					TimeoutMs: 1000,
					IntervalS: intervalS,
				},
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
	return cfg
}

func mustMonitor(t *testing.T, cfg *config.Config, options Options) *Monitor {
	t.Helper()
	m, err := New(cfg, options)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return m
}

func waitForStatus(t *testing.T, m *Monitor, want CheckStatus) TileState {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		states := m.Snapshot()
		if len(states) == 1 && states[0].Status == want && len(states[0].History) > 0 {
			return states[0]
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for status %q", want)
	return TileState{}
}

func TestMonitorSuccessAndUptime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := mustMonitor(t, testConfig(t, server.URL, config.MinIntervalS), Options{HistorySize: config.HistorySize})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	state := waitForStatus(t, m, StatusSuccess)
	if state.Uptime != 1 {
		t.Errorf("Uptime = %f, want 1", state.Uptime)
	}
	if state.Status != StatusSuccess {
		t.Errorf("Status = %q, want SUCCESS", state.Status)
	}
	if state.ResponseTimeMs < 0 {
		t.Errorf("ResponseTimeMs = %d, should be positive", state.ResponseTimeMs)
	}
	if len(state.History) != 1 {
		t.Errorf("len(History) = %d, want 1", len(state.History))
	}
}

func TestMonitorUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	m := mustMonitor(t, testConfig(t, server.URL, config.MinIntervalS), Options{HistorySize: config.HistorySize})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	state := waitForStatus(t, m, StatusFailed)
	if state.Message == "" {
		t.Error("Message should describe the unexpected status code")
	}
	if state.Uptime != 0 {
		t.Errorf("Uptime = %f, want 0", state.Uptime)
	}
}

func TestMonitorKeyword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>hello world</html>"))
	}))
	defer server.Close()

	cfg := testConfig(t, server.URL, config.MinIntervalS)
	cfg.Tiles[0].Params.Keyword = "goodbye"

	m := mustMonitor(t, cfg, Options{HistorySize: config.HistorySize})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	state := waitForStatus(t, m, StatusFailed)
	if state.Message != "keyword not found in response body" {
		t.Errorf("Message = %q, want keyword error", state.Message)
	}
}

func TestMonitorConnectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	m := mustMonitor(t, testConfig(t, url, config.MinIntervalS), Options{HistorySize: config.HistorySize})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	state := waitForStatus(t, m, StatusFailed)
	if state.Message == "" {
		t.Error("Message should describe the connection error")
	}
}

func TestMonitorHistoryIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	options := Options{HistorySize: 3}
	m := mustMonitor(t, testConfig(t, server.URL, config.MinIntervalS), options)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if len(m.Snapshot()[0].History) == options.HistorySize {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := len(m.Snapshot()[0].History); got != options.HistorySize {
		t.Fatalf("len(History) = %d, want %d", got, options.HistorySize)
	}
}

func TestSubscribeReceivesUpdates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := mustMonitor(t, testConfig(t, server.URL, config.MinIntervalS), Options{HistorySize: config.HistorySize})
	ch, cancel := m.Subscribe()
	defer cancel()

	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	m.Start(ctx)

	select {
	case data := <-ch:
		var state TileState
		if err := json.Unmarshal(data, &state); err != nil {
			t.Fatalf("unmarshal update: %v", err)
		}
		if state.ID == "" {
			t.Error("update should carry the tile ID")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a broadcast")
	}
}

func TestMonitorGroupAggregation(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failServer.Close()

	cfg := &config.Config{
		Version: "1.0",
		Columns: 2,
		Tiles: []config.TileConfig{
			{
				Type:  config.TypeGroup,
				Label: "Prod",
				Tiles: []config.TileConfig{
					{Type: config.TypeUptime, Label: "OK", Params: config.Params{URL: okServer.URL, TimeoutMs: 1000, IntervalS: config.MinIntervalS}},
					{Type: config.TypeUptime, Label: "KO", Params: config.Params{URL: failServer.URL, TimeoutMs: 1000, IntervalS: config.MinIntervalS}},
				},
			},
		},
	}

	m := mustMonitor(t, cfg, Options{HistorySize: config.HistorySize})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		states := m.Snapshot()
		if len(states) != 1 {
			t.Fatalf("len(states) = %d, want 1 (only the group holds state)", len(states))
		}
		group := states[0]
		if group.Type != config.TypeGroup {
			t.Fatalf("Type = %q, want %q", group.Type, config.TypeGroup)
		}
		if group.Status == StatusFailed && group.Uptime == 0.5 {
			if len(group.Tiles) != 2 {
				t.Errorf("len(Tiles) = %d, want 2 sub-tile IDs", len(group.Tiles))
			}
			if group.Since == 0 {
				t.Error("group Since should be set from sub-tiles")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for aggregated group state (want FAILED, uptime 0.5)")
}

func TestMonitorEmptyTileHoldsNoState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Version: "1.0",
		Columns: 2,
		Tiles: []config.TileConfig{
			{Type: config.TypeEmpty},
			{Type: config.TypeUptime, Label: "Test", Params: config.Params{URL: server.URL, TimeoutMs: 1000, IntervalS: config.MinIntervalS}},
			{Type: config.TypeEmpty},
		},
	}

	m := mustMonitor(t, cfg, Options{HistorySize: config.HistorySize})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		states := m.Snapshot()
		if len(states) == 1 {
			if states[0].Type != config.TypeUptime {
				t.Errorf("Type = %q, want %q", states[0].Type, config.TypeUptime)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("Snapshot should hold exactly one state (EMPTY tiles excluded), got %d", len(m.Snapshot()))
}
