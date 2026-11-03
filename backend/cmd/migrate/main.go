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
	"path/filepath"
	"strings"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/hung98dev/ttgqldl/backend/db/migrations"
	"github.com/hung98dev/ttgqldl/backend/internal/config"
	"github.com/hung98dev/ttgqldl/backend/internal/schema"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
)

const (
	migrationDir = "."
	defaultEnv   = ".env"
	// schemaName is owned by docs/plan/04_DATABASE.md: every authoritative object
	// lives in `ttg`, including goose's own bookkeeping table, so that a role
	// without a search_path cannot scatter objects into `public`.
	schemaName = "ttg"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	direction := flag.String("direction", "up", "up|down|status|version|fingerprint")
	to := flag.Int64("to", -1, "target version for -direction down (0 rolls everything back)")
	timeout := flag.Duration("timeout", 2*time.Minute, "overall migration timeout")
	out := flag.String("out", "", "write the -direction fingerprint output to this file instead of stdout")
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

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// goose creates its version table before the first migration runs, so the
	// target schema must exist and be selected first. Doing this explicitly
	// makes the location deterministic: a role whose search_path is the default
	// ("$user", public) would otherwise put goose_db_version in public.
	if err := ensureSchema(ctx, dsn); err != nil {
		return err
	}

	db, err := storage.OpenSQL(schema.WithSearchPath(dsn, schemaName))
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

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
	case "fingerprint":
		fp, err := schema.Read(ctx, db)
		if err != nil {
			return err
		}
		data, err := schema.Marshal(fp)
		if err != nil {
			return err
		}
		if *out != "" {
			// The path comes from the operator running this local CLI (tooling/
			// schema-dump passes an absolute path inside the repository). Traversal
			// sequences are rejected so a mistyped flag cannot escape the working
			// tree.
			cleaned := filepath.Clean(*out)
			if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
				return fmt.Errorf("refusing output path %q: it escapes the working tree", *out)
			}
			// #nosec G703 G306 -- operator-supplied local CLI path, not untrusted input; the
			// fingerprint is a committed public artifact that must be world-readable.
			if err := os.WriteFile(cleaned, data, 0o644); err != nil {
				return fmt.Errorf("write fingerprint: %w", err)
			}
			fmt.Fprintf(os.Stderr, "migrate: wrote schema fingerprint to %s (%d tables)\n", *out, len(fp.Tables))
			return nil
		}
		_, err = os.Stdout.Write(data)
		return err
	default:
		return fmt.Errorf("unknown direction %q (want up|down|status|version|fingerprint)", *direction)
	}

	version, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		return err
	}
	fmt.Printf("migrate: %s complete, schema version %d\n", *direction, version)
	return nil
}

// ensureSchema creates the owned schema before goose touches the database.
func ensureSchema(ctx context.Context, dsn string) error {
	db, err := storage.OpenSQL(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("cannot reach the migration database: %w", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+schemaName); err != nil {
		return fmt.Errorf("create schema %s (the migration role needs CREATE on the database): %w", schemaName, err)
	}
	return nil
}
