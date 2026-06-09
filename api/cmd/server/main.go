// Command server runs the Mooring Line Management API.
//
// Usage:
//
//	server                 run the HTTP server
//	server dump-openapi    write the OpenAPI 3.1 spec to api/openapi/openapi.json and exit
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ncl/mooring-api/internal/config"
	"github.com/ncl/mooring-api/internal/dbmigrate"
	"github.com/ncl/mooring-api/internal/httpapi"
	"github.com/ncl/mooring-api/internal/seed"
	"github.com/ncl/mooring-api/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "dump-openapi":
			if err := dumpOpenAPI(); err != nil {
				log.Error("dump-openapi failed", "err", err)
				os.Exit(1)
			}
			return
		case "migrate":
			if err := runMigrate(log, os.Args[2:]); err != nil {
				log.Error("migrate failed", "err", err)
				os.Exit(1)
			}
			return
		case "seed":
			if err := runSeed(log, os.Args[2:]); err != nil {
				log.Error("seed failed", "err", err)
				os.Exit(1)
			}
			return
		}
	}

	if err := run(log); err != nil {
		log.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.AutoMigrate {
		if err := dbmigrate.Up(cfg.DatabaseURL); err != nil {
			log.Warn("auto-migrate failed; continuing", "err", err)
		} else {
			log.Info("auto-migrate ok")
		}
	}

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		// DB optional at boot so the skeleton runs before migrations exist; log and continue.
		log.Warn("database unavailable at startup; continuing without it", "err", err)
		st = nil
	}
	if st != nil {
		defer st.Close()
		// Outbound webhook dispatcher: polls the outbox and delivers to subscriptions.
		go st.RunWebhookDispatcher(ctx, log)
	}

	handler, _ := httpapi.NewAPI(&httpapi.Server{Cfg: cfg, Store: st})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Info("starting server", "addr", cfg.HTTPAddr, "scope", cfg.Scope, "vessel_id", cfg.VesselID)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("server stopped")
	return nil
}

// runMigrate applies or rolls back schema migrations: `server migrate up|down`.
func runMigrate(log *slog.Logger, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	dir := "up"
	if len(args) > 0 {
		dir = args[0]
	}
	switch dir {
	case "up":
		if err := dbmigrate.Up(cfg.DatabaseURL); err != nil {
			return err
		}
		log.Info("migrations applied")
	case "down":
		if err := dbmigrate.Down(cfg.DatabaseURL); err != nil {
			return err
		}
		log.Info("migrations rolled back")
	default:
		return errors.New("usage: server migrate up|down")
	}
	return nil
}

// runSeed loads demo data: `server seed [--reset]`.
func runSeed(log *slog.Logger, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()
	reset := len(args) > 0 && args[0] == "--reset"
	if err := seed.Run(ctx, st.Pool, reset); err != nil {
		return err
	}
	log.Info("seed complete", "reset", reset)
	return nil
}

// dumpOpenAPI builds the API with no live dependencies and writes the spec.
func dumpOpenAPI() error {
	cfg := &config.Config{Scope: config.ScopeShore}
	_, api := httpapi.NewAPI(&httpapi.Server{Cfg: cfg})

	spec, err := api.OpenAPI().YAML()
	if err != nil {
		return err
	}
	_ = spec // keep YAML available; we publish JSON for openapi-typescript

	b, err := json.MarshalIndent(api.OpenAPI(), "", "  ")
	if err != nil {
		return err
	}
	out := filepath.Join("openapi", "openapi.json")
	if err := os.MkdirAll("openapi", 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, b, 0o644)
}
