// Package api exposes the HTTP API (config, SSE events, health) and serves the bundled UI.
package api

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/Alangave/monitoror/upmonitor/internal/config"
	"github.com/Alangave/monitoror/upmonitor/internal/monitor"
)

type tileInfo struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type configPayload struct {
	Version string     `json:"version"`
	Columns int        `json:"columns"`
	Tiles   []tileInfo `json:"tiles"`
}

type tileEvent struct {
	Event string            `json:"event"`
	State monitor.TileState `json:"state"`
}

// New builds the HTTP handler serving the API and the embedded UI.
func New(cfg *config.Config, mon *monitor.Monitor, ui fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		tiles := make([]tileInfo, 0, len(cfg.Tiles))
		for _, tile := range cfg.Tiles {
			tiles = append(tiles, tileInfo{ID: tile.ID, Type: tile.Type, Label: tile.Label})
		}
		writeJSON(w, http.StatusOK, configPayload{
			Version: cfg.Version,
			Columns: cfg.Columns,
			Tiles:   tiles,
		})
	})

	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		updates, cancel := mon.Subscribe()
		defer cancel()

		if err := sendSnapshot(w, flusher, mon); err != nil {
			return
		}

		ctx := r.Context()
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case data := <-updates:
				if _, err := w.Write([]byte("event: tile\ndata: " + string(data) + "\n\n")); err != nil {
					return
				}
				flusher.Flush()
			case <-heartbeat.C:
				if _, err := w.Write([]byte(": heartbeat\n\n")); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	})

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	if ui != nil {
		fileServer := http.FileServer(http.FS(ui))
		mux.Handle("GET /", fileServer)
	}

	return logRequests(mux)
}

func sendSnapshot(w http.ResponseWriter, flusher http.Flusher, mon *monitor.Monitor) error {
	states := mon.Snapshot()
	for _, state := range states {
		data, err := json.Marshal(state)
		if err != nil {
			continue
		}
		if _, err := w.Write([]byte("event: tile\ndata: " + string(data) + "\n\n")); err != nil {
			return err
		}
	}
	flusher.Flush()
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("unable to encode response: %v", err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if r.URL.Path == "/api/events" {
			return
		}
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
