// Command migrate applies, rolls back and inspects the embedded goose
// migrations.
//
// It always connects with the migration DSN (the schema-owner role). The server
// itself connects as the least-privileged application role and cannot run DDL.
//
//	TTG_MIGRATION_DATABASE_URL=postgres://ttg_owner:...@127.0.0.1:5432/ttgqldl \
//	  go run ./cmd/migrate -direction up
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/hung98dev/ttgqldl/backend/db/migrations"
	"github.com/hung98dev/ttgqldl/backend/internal/config"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
)

const (
	migrationDir = "."
	defaultEnv   = ".env"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	direction := flag.String("direction", "up", "up|down|status|version")
	to := flag.Int64("to", -1, "target version for -direction down (0 rolls everything back)")
	timeout := flag.Duration("timeout", 2*time.Minute, "overall migration timeout")
	flag.Parse()

	envFile := os.Getenv("TTG_ENV_FILE")
	if envFile == "" {
		envFile = defaultEnv
	}
	applied, existed, err := config.LoadEnvFileIntoProcess(envFile)
	if err != nil {
		return err
	}
	if existed {
		fmt.Fprintf(os.Stderr, "migrate: loaded %d setting(s) from %s\n", applied, envFile)
	}

	dsn := os.Getenv("TTG_MIGRATION_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("TTG_DATABASE_URL")
	}
	if dsn == "" {
		return fmt.Errorf("TTG_MIGRATION_DATABASE_URL or TTG_DATABASE_URL must be set")
	}

	db, err := storage.OpenSQL(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("cannot reach the migration database: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	switch *direction {
	case "up":
		if err := goose.UpContext(ctx, db, migrationDir); err != nil {
			return err
		}
	case "down":
		if *to >= 0 {
			if err := goose.DownToContext(ctx, db, migrationDir, *to); err != nil {
				return err
			}
		} else if err := goose.DownContext(ctx, db, migrationDir); err != nil {
			return err
		}
	case "status":
		if err := goose.StatusContext(ctx, db, migrationDir); err != nil {
			return err
		}
	case "version":
		version, err := goose.GetDBVersionContext(ctx, db)
		if err != nil {
			return err
		}
		fmt.Printf("%d\n", version)
		return nil
	default:
		return fmt.Errorf("unknown direction %q (want up|down|status|version)", *direction)
	}

	version, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		return err
	}
	fmt.Printf("migrate: %s complete, schema version %d\n", *direction, version)
	return nil
}
