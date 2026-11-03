// Command server runs the TTGQLDL backend HTTP service.
//
// M01 Phase A serves only the liveness and readiness contract. Gameplay,
// authentication, the input source adapter and the operator API arrive in later
// milestones; this binary refuses to pretend otherwise.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/app"
	"github.com/hung98dev/ttgqldl/backend/internal/config"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/telemetry"
)

// defaultEnvFile is read from the working directory unless TTG_ENV_FILE points
// elsewhere. Development scripts set it to the repository root .env.
const defaultEnvFile = ".env"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ttgqldl server: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	envFile := os.Getenv("TTG_ENV_FILE")
	if envFile == "" {
		envFile = defaultEnvFile
	}
	values, existed, err := config.LoadDotEnvFile(envFile)
	if err != nil {
		return err
	}
	if existed {
		applied, applyErr := config.ApplyDotEnv(values, os.LookupEnv, os.Setenv)
		if applyErr != nil {
			return applyErr
		}
		fmt.Fprintf(os.Stderr, "ttgqldl server: loaded %d setting(s) from %s\n", len(applied), envFile)
	}

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	logger, err := telemetry.NewLogger(os.Stdout, cfg.LogLevel, "ttgqldl-backend",
		slog.String("env", string(cfg.Env)),
		slog.String("source", string(cfg.Source)),
		slog.String("version", buildVersion()),
	)
	if err != nil {
		return err
	}

	// The pool connects lazily, so the process starts even when PostgreSQL is
	// down and reports the truth through readiness instead of crash-looping.
	pool, err := storage.Open(context.Background(), storage.Options{
		DSN:             cfg.DatabaseURL,
		MaxConns:        10,
		ConnectTimeout:  5 * time.Second,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(app.Options{
		Logger: logger,
		Config: app.Config{
			HTTPAddr:         cfg.HTTPAddr,
			ReadinessTimeout: cfg.ReadinessTimeout,
			ShutdownTimeout:  cfg.ShutdownTimeout,
		},
		Checks: []app.Check{
			{Name: "postgres", Run: pool.Ping},
		},
		// No simulation writer lease exists before M04, so no fencing epoch.
		WriterEpoch: nil,
		Version:     buildVersion(),
	})
	if err != nil {
		return err
	}

	logger.Info("backend starting",
		slog.String("http_addr", cfg.HTTPAddr),
		slog.String("database_host", "configured"),
		slog.String("note", "M01 Phase A serves health and readiness only; source adapter, gameplay and admin API are not implemented"),
	)
	if err := application.Run(ctx); err != nil {
		return err
	}
	logger.Info("backend stopped")
	return nil
}

// buildVersion reports the VCS revision when the binary was built from a
// checkout, falling back to the module version.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	revision := ""
	dirty := ""
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if revision != "" {
		return revision + dirty
	}
	if info.Main.Version != "" {
		return info.Main.Version
	}
	return "unknown"
}
