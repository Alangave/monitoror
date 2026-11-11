// Command upmonitor is a single-binary URL monitoring wall inspired by Monitoror.
// It serves the bundled UI and exposes a config endpoint plus an SSE events stream.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	upmonitor "github.com/Alangave/monitoror/upmonitor"
	"github.com/Alangave/monitoror/upmonitor/internal/api"
	"github.com/Alangave/monitoror/upmonitor/internal/config"
	"github.com/Alangave/monitoror/upmonitor/internal/monitor"
)

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	configPath := envOr("UPMONITOR_CONFIG", "config.json")
	addr := envOr("UPMONITOR_ADDR", ":"+envOr("UPMONITOR_PORT", "8080"))

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mon, err := monitor.New(cfg, monitor.DefaultOptions())
	if err != nil {
		log.Fatalf("monitor error: %v", err)
	}
	mon.Start(ctx)

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(cfg, mon, upmonitor.Dist()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("upmonitor listening on %s (config: %s)", addr, configPath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
