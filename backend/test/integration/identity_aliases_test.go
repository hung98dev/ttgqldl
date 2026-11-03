package integration

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// seedVillageAndPlayer inserts a village plus one player inside tx and returns
// both ids. Callers roll the transaction back.
func seedVillageAndPlayer(t *testing.T, tx *sql.Tx, actorKey string) (villageID, playerID string) {
	t.Helper()
	ctx := context.Background()
	villageID = newUUID(t)
	playerID = newUUID(t)
	// broadcaster_key is UNIQUE and CHAR(64); derive it from the generated id so
	// several villages can be seeded inside one transaction.
	broadcasterKey := (villageID + strings.Repeat("0", 64))[:64]
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO ttg.villages (id, broadcaster_key) VALUES ($1, $2)`,
		villageID, broadcasterKey); err != nil {
		t.Fatalf("insert village: %v", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO ttg.players (id, village_id, actor_key, ephemeral) VALUES ($1, $2, $3, FALSE)`,
		playerID, villageID, actorKey); err != nil {
		t.Fatalf("insert player: %v", err)
	}
	return villageID, playerID
}

func mustReject(t *testing.T, tx *sql.Tx, name, query string, args ...any) string {
	t.Helper()
	ctx := context.Background()
	if _, err := tx.ExecContext(ctx, "SAVEPOINT alias_case"); err != nil {
		t.Fatalf("%s: savepoint: %v", name, err)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	if err == nil {
		t.Fatalf("%s: expected the schema to reject the statement", name)
	}
	if _, rbErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT alias_case"); rbErr != nil {
		t.Fatalf("%s: rollback to savepoint: %v", name, rbErr)
	}
	return err.Error()
}

// TestIdentityAliasesContract covers docs/plan/04_DATABASE.md
// ("Privacy restore ledger and HMAC rotation"): the alias table must resolve a
// versioned HMAC key to a player in the SAME village and must never store a raw
// provider id.
func TestIdentityAliasesContract(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	keyA := strings.Repeat("a", 64)
	villageA, playerA := seedVillageAndPlayer(t, tx, keyA)

	// A well-formed mapping is accepted.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id)
		VALUES ($1, $2, '1', $3)`, villageA, strings.Repeat("1", 64), playerA); err != nil {
		t.Fatalf("valid alias rejected: %v", err)
	}

	// Same actor key may be recorded for another key version (rotation).
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id)
		VALUES ($1, $2, '2', $3)`, villageA, strings.Repeat("2", 64), playerA); err != nil {
		t.Fatalf("second key version rejected: %v", err)
	}

	// Duplicate (village, actor_key, key_version) is rejected.
	if msg := mustReject(t, tx, "duplicate mapping", `
		INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id)
		VALUES ($1, $2, '1', $3)`, villageA, strings.Repeat("1", 64), playerA); !strings.Contains(msg, "duplicate key") {
		t.Fatalf("duplicate mapping: unexpected error %q", msg)
	}

	// Cross-village mapping is rejected by the composite foreign key, not only by
	// application code.
	villageB, _ := seedVillageAndPlayer(t, tx, strings.Repeat("b", 64))
	if msg := mustReject(t, tx, "cross-village mapping", `
		INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id)
		VALUES ($1, $2, '1', $3)`, villageB, strings.Repeat("3", 64), playerA); !strings.Contains(msg, "foreign key") {
		t.Fatalf("cross-village mapping: unexpected error %q", msg)
	}

	// A missing player is rejected.
	if msg := mustReject(t, tx, "missing player", `
		INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id)
		VALUES ($1, $2, '1', $3)`, villageA, strings.Repeat("4", 64), newUUID(t)); !strings.Contains(msg, "foreign key") {
		t.Fatalf("missing player: unexpected error %q", msg)
	}

	// A raw provider id is not an HMAC hex digest and must be refused.
	if msg := mustReject(t, tx, "raw provider id", `
		INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id)
		VALUES ($1, $2, '1', $3)`, villageA, "user-12345", playerA); !strings.Contains(msg, "actor_key_is_hmac_hex") {
		t.Fatalf("raw provider id: unexpected error %q", msg)
	}
	// Hex digits but the wrong length is also refused.
	if msg := mustReject(t, tx, "short digest", `
		INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id)
		VALUES ($1, $2, '1', $3)`, villageA, strings.Repeat("5", 63), playerA); !strings.Contains(msg, "actor_key_is_hmac_hex") {
		t.Fatalf("short digest: unexpected error %q", msg)
	}
	// Key version must be a bounded, simple token; it names a secret generation.
	if msg := mustReject(t, tx, "malformed key version", `
		INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id)
		VALUES ($1, $2, 'not a version!', $3)`, villageA, strings.Repeat("6", 64), playerA); !strings.Contains(msg, "key_version_shape") {
		t.Fatalf("malformed key version: unexpected error %q", msg)
	}

	// Erasing a player removes its mappings (CASCADE), so a deleted subject
	// cannot leave a live pseudonymous pointer behind.
	if _, err := tx.ExecContext(ctx, `DELETE FROM ttg.players WHERE id = $1`, playerA); err != nil {
		t.Fatalf("delete player: %v", err)
	}
	var remaining int
	if err := tx.QueryRowContext(ctx,
		`SELECT count(*) FROM ttg.actor_identity_aliases WHERE player_id = $1`, playerA).Scan(&remaining); err != nil {
		t.Fatalf("count aliases: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("%d alias rows survived the player deletion", remaining)
	}
}

// TestPrivacyLedgerOutboxContract covers the transactional export state required
// before an external deletion ledger can be trusted (M04 implements the exporter).
func TestPrivacyLedgerOutboxContract(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	villageID, _ := seedVillageAndPlayer(t, tx, strings.Repeat("c", 64))
	sessionID := newUUID(t)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version)
		VALUES ($1, $2, 'mock', 'waiting', 'ttg-rules-v1')`, sessionID, villageID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	tombstoneID := newUUID(t)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.privacy_tombstones (id, actor_ref_protected, scope)
		VALUES ($1, $2, 'village')`, tombstoneID, strings.Repeat("z", 64)); err != nil {
		t.Fatalf("insert tombstone: %v", err)
	}

	// A queued, unexported event is valid.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload)
		VALUES ($1, $2, '{"marker":"anon","scope":"village"}'::jsonb)`, newUUID(t), tombstoneID); err != nil {
		t.Fatalf("valid outbox event rejected: %v", err)
	}

	// Acknowledged export requires both the timestamp and the ledger reference.
	if msg := mustReject(t, tx, "ack without timestamp", `
		INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload, ledger_ack_ref)
		VALUES ($1, $2, '{}'::jsonb, 'ledger-1')`, newUUID(t), tombstoneID); !strings.Contains(msg, "ack_consistency") {
		t.Fatalf("ack without timestamp: unexpected error %q", msg)
	}
	if msg := mustReject(t, tx, "timestamp without ack", `
		INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload, exported_at)
		VALUES ($1, $2, '{}'::jsonb, now())`, newUUID(t), tombstoneID); !strings.Contains(msg, "ack_consistency") {
		t.Fatalf("timestamp without ack: unexpected error %q", msg)
	}

	// The payload must be an object (the exporter writes a protected marker).
	if msg := mustReject(t, tx, "array payload", `
		INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload)
		VALUES ($1, $2, '[]'::jsonb)`, newUUID(t), tombstoneID); !strings.Contains(msg, "payload_is_object") {
		t.Fatalf("array payload: unexpected error %q", msg)
	}

	// Negative attempt counters and missing tombstones are refused.
	if msg := mustReject(t, tx, "negative attempts", `
		INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload, attempts)
		VALUES ($1, $2, '{}'::jsonb, -1)`, newUUID(t), tombstoneID); !strings.Contains(msg, "attempts_non_negative") {
		t.Fatalf("negative attempts: unexpected error %q", msg)
	}
	if msg := mustReject(t, tx, "missing tombstone", `
		INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload)
		VALUES ($1, $2, '{}'::jsonb)`, newUUID(t), newUUID(t)); !strings.Contains(msg, "foreign key") {
		t.Fatalf("missing tombstone: unexpected error %q", msg)
	}

	// A fully acknowledged row is accepted.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload, exported_at, ledger_ack_ref)
		VALUES ($1, $2, '{}'::jsonb, now(), 'ledger-ack-1')`, newUUID(t), tombstoneID); err != nil {
		t.Fatalf("acknowledged event rejected: %v", err)
	}
}
