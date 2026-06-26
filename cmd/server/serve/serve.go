// Package serve runs the voltpilot HTTP server with graceful shutdown and a
// background EnBW subscription-key refresher.
package serve

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamcore/voltpilot/internal/api"
	"github.com/tamcore/voltpilot/internal/chargers"
	"github.com/tamcore/voltpilot/internal/config"
	"github.com/tamcore/voltpilot/internal/enbw"
)

const (
	readHeaderTime  = 10 * time.Second
	shutdownTimeout = 10 * time.Second
)

// Run starts the HTTP server and blocks until SIGINT/SIGTERM.
func Run() error {
	cfg := config.Load()

	keys := enbw.NewKeyManager(cfg.ENBWMapURL, cfg.ENBWKeySeed, &http.Client{Timeout: 15 * time.Second})
	// Scrape the live key once at startup; the seed (if any) covers the gap.
	if err := keys.Refresh(context.Background()); err != nil {
		slog.Warn("initial enbw key scrape failed; relying on seed key", "err", err)
	}
	if keys.Key() == "" {
		slog.Warn("no enbw subscription key available; charger endpoints will error until the next scrape succeeds")
	}

	client := enbw.NewClient(keys, "", nil)
	svc := chargers.NewService(client)

	refreshCtx, refreshCancel := context.WithCancel(context.Background())
	defer refreshCancel()
	go keyRefreshLoop(refreshCtx, keys, cfg.KeyRefreshInterval)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           api.NewRouter(svc),
		ReadHeaderTimeout: readHeaderTime,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-stop:
		slog.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func keyRefreshLoop(ctx context.Context, keys *enbw.KeyManager, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := keys.Refresh(ctx); err != nil {
				slog.Warn("scheduled enbw key refresh failed", "err", err)
			}
		}
	}
}
