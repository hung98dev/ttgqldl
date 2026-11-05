package integration

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/hung98dev/ttgqldl/backend/db/migrations"
	"github.com/hung98dev/ttgqldl/backend/internal/schema"
)

// migrationDB opens a connection for the schema-owner role.
//
// This file destroys and recreates the schema, so it only runs when
// TTG_TEST_DESTRUCTIVE=1 explicitly marks the database as disposable.
func migrationDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("TTG_TEST_DESTRUCTIVE") != "1" {
		t.Skip("destructive migration test; set TTG_TEST_DESTRUCTIVE=1 for a disposable database")
	}
	dsn := os.Getenv("TTG_MIGRATION_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("TTG_MIGRATION_DATABASE_URL is not set; the destructive test needs the owner role")
	}
	db, err := sql.Open("pgx", schema.WithSearchPath(dsn, schema.SchemaName))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("dialect: %v", err)
	}
	return db
}

func dbVersion(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	version, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		t.Fatalf("db version: %v", err)
	}
	return version
}

// TestMigrationLifecycleDestructive proves the upgrade path and the rollback
// guard on a database that is explicitly marked disposable:
//
//	version 3 data survives the upgrade to version 7, re-applying is idempotent,
//	and a rollback that would destroy identity mappings or unexported deletion
//	events fails closed.
func TestMigrationLifecycleDestructive(t *testing.T) {
	db := migrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start from the last released version and seed data at that level.
	if err := goose.DownToContext(ctx, db, ".", 3); err != nil {
		t.Fatalf("down to version 3: %v", err)
	}
	if got := dbVersion(t, db); got != 3 {
		t.Fatalf("expected version 3, got %d", got)
	}

	villageID := newUUID(t)
	playerID := newUUID(t)
	broadcasterKey := (villageID + strings.Repeat("0", 64))[:64]
	if _, err := db.ExecContext(ctx,
		`INSERT INTO ttg.villages (id, broadcaster_key) VALUES ($1, $2)`, villageID, broadcasterKey); err != nil {
		t.Fatalf("seed village at version 3: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO ttg.players (id, village_id, actor_key, ephemeral) VALUES ($1, $2, $3, FALSE)`,
		playerID, villageID, strings.Repeat("d", 64)); err != nil {
		t.Fatalf("seed player at version 3: %v", err)
	}

	// Upgrade and confirm the pre-existing rows survived.
	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatalf("up to latest: %v", err)
	}
	if got := dbVersion(t, db); got != 7 {
		t.Fatalf("expected version 7 after upgrade, got %d", got)
	}
	var surviving int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM ttg.villages WHERE id = $1`, villageID).Scan(&surviving); err != nil {
		t.Fatalf("count village: %v", err)
	}
	if surviving != 1 {
		t.Fatal("data created at version 3 did not survive the upgrade to version 7")
	}
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM ttg.players WHERE id = $1`, playerID).Scan(&surviving); err != nil {
		t.Fatalf("count player: %v", err)
	}
	if surviving != 1 {
		t.Fatal("player created at version 3 did not survive the upgrade to version 7")
	}

	// Re-applying at the latest version is idempotent.
	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatalf("re-applying migrations must be a no-op: %v", err)
	}
	if got := dbVersion(t, db); got != 7 {
		t.Fatalf("version changed on a repeat run: %d", got)
	}

	// The rollback guard must refuse to destroy a live identity mapping.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id)
		VALUES ($1, $2, '1', $3)`, villageID, strings.Repeat("e", 64), playerID); err != nil {
		t.Fatalf("insert alias: %v", err)
	}
	err := goose.DownToContext(ctx, db, ".", 3)
	if err == nil {
		t.Fatal("rollback dropped a populated identity-alias table")
	}
	if !strings.Contains(err.Error(), "refusing to drop ttg.actor_identity_aliases") {
		t.Fatalf("rollback failed for the wrong reason: %v", err)
	}
	if got := dbVersion(t, db); got != 4 {
		t.Fatalf("failed rollback must leave the schema at version 4, got %d", got)
	}

	// Same guarantee for an unexported deletion event.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO ttg.privacy_tombstones (id, actor_ref_protected, scope)
		VALUES ($1, $2, 'village')`, newUUID(t), strings.Repeat("f", 64)); err != nil {
		t.Fatalf("insert tombstone: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload)
		SELECT $1, id, '{"marker":"anon"}'::jsonb FROM ttg.privacy_tombstones
		ORDER BY created_at DESC LIMIT 1`, newUUID(t)); err != nil {
		t.Fatalf("insert outbox event: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM ttg.actor_identity_aliases`); err != nil {
		t.Fatalf("clear aliases: %v", err)
	}
	err = goose.DownToContext(ctx, db, ".", 3)
	if err == nil || !strings.Contains(err.Error(), "refusing to drop ttg.privacy_ledger_outbox") {
		t.Fatalf("rollback must be blocked by an unexported deletion event, got %v", err)
	}

	// Once the disposable data is cleared the rollback is allowed, and the schema
	// can be rebuilt from zero.
	if _, err := db.ExecContext(ctx, `DELETE FROM ttg.privacy_ledger_outbox`); err != nil {
		t.Fatalf("clear outbox: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM ttg.privacy_tombstones`); err != nil {
		t.Fatalf("clear tombstones: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM ttg.players`); err != nil {
		t.Fatalf("clear players: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM ttg.sessions`); err != nil {
		t.Fatalf("clear sessions: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM ttg.villages`); err != nil {
		t.Fatalf("clear villages: %v", err)
	}
	if err := goose.DownToContext(ctx, db, ".", 0); err != nil {
		t.Fatalf("down to zero on an empty disposable database: %v", err)
	}
	if got := dbVersion(t, db); got != 0 {
		t.Fatalf("expected version 0 after a full rollback, got %d", got)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatalf("rebuild after full rollback: %v", err)
	}
	if got := dbVersion(t, db); got != 7 {
		t.Fatalf("expected version 7 after rebuilding, got %d", got)
	}
}
