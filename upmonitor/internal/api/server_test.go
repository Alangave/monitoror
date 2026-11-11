package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alangave/monitoror/upmonitor/internal/config"
	"github.com/Alangave/monitoror/upmonitor/internal/monitor"
)

func testCfg(t *testing.T, url string) *config.Config {
	t.Helper()
	cfg := &config.Config{
		Version: "1.0",
		Columns: 2,
		Tiles: []config.TileConfig{{
			Type:  config.TypeUptime,
			Label: "Test",
			Params: config.Params{
				URL:       url,
				TimeoutMs: 1000,
				IntervalS: config.MinIntervalS,
			},
		}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
	return cfg
}

func TestConfigEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testCfg(t, server.URL)
	mon, err := monitor.New(cfg, monitor.DefaultOptions())
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	New(cfg, mon, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var payload configPayload
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Columns != 2 || len(payload.Tiles) != 1 {
		t.Errorf("unexpected payload: %s", recorder.Body.String())
	}
	if payload.Tiles[0].ID != cfg.Tiles[0].ID {
		t.Errorf("tile ID = %q, want %q", payload.Tiles[0].ID, cfg.Tiles[0].ID)
	}
}

func TestEventsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testCfg(t, server.URL)
	mon, err := monitor.New(cfg, monitor.DefaultOptions())
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mon.Start(ctx)

	handler := New(cfg, mon, nil)

	events := make(chan string, 1)
	request := httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()

	go func() {
		handler.ServeHTTP(recorder, request)
		close(events)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		body := recorder.Body.String()
		if strings.Contains(body, "event: tile") && strings.Contains(body, `"id":"`) {
			if !strings.Contains(body, "\"uptime\"") {
				t.Errorf("tile event should contain uptime: %s", body)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no tile event received, body: %s", recorder.Body.String())
}

func TestHealthzEndpoint(t *testing.T) {
	cfg := testCfg(t, "http://127.0.0.1:1")
	mon, err := monitor.New(cfg, monitor.DefaultOptions())
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	New(cfg, mon, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "ok") {
		t.Errorf("healthz = %d %q", recorder.Code, recorder.Body.String())
	}
}
