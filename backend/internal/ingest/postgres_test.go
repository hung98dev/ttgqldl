package ingest

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
)

func testPool(t *testing.T) *storage.Pool {
	t.Helper()
	dsn := os.Getenv("TTG_TEST_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("TTG_TEST_DATABASE_URL is not set; durable ingestion tests require PostgreSQL")
	}
	pool, err := storage.Open(context.Background(), storage.Options{DSN: dsn, MaxConns: 8, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping test pool: %v", err)
	}
	return pool
}

func testUUID(t testing.TB) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatal(err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}

func execSQL(t testing.TB, pool *storage.Pool, statement string, args ...any) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, statement, args...); err != nil {
		t.Fatalf("exec %q: %v", statement, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func scalarInt(t *testing.T, pool *storage.Pool, statement string, args ...any) int64 {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var value int64
	if err := tx.QueryRow(ctx, statement, args...).Scan(&value); err != nil {
		t.Fatalf("query %q: %v", statement, err)
	}
	return value
}

type testSession struct {
	ID        string
	VillageID string
}

func createSession(t testing.TB, pool *storage.Pool, mode, lifecycle string, waveNo int64, waveTick int32, inputSequence int64) testSession {
	t.Helper()
	villageID := testUUID(t)
	sessionID := testUUID(t)
	broadcaster := strings.ReplaceAll(sessionID, "-", "") + strings.ReplaceAll(sessionID, "-", "")
	execSQL(t, pool, `INSERT INTO ttg.villages (id, broadcaster_key) VALUES ($1, $2)`, villageID, broadcaster[:64])
	execSQL(t, pool, `INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, wave_no, wave_tick, rules_version, input_sequence)
		VALUES ($1, $2, $3, $4, $5, $6, 'ttg-rules-v1', $7)`, sessionID, villageID, mode, lifecycle, waveNo, waveTick, inputSequence)
	t.Cleanup(func() {
		ctx := context.Background()
		for _, statement := range []string{
			`DELETE FROM ttg.input_receipts WHERE session_id = $1`,
			`DELETE FROM ttg.input_events WHERE session_id = $1`,
			`DELETE FROM ttg.sessions WHERE id = $1`,
			`DELETE FROM ttg.villages WHERE id = $1`,
		} {
			tx, err := pool.Begin(ctx)
			if err != nil {
				return
			}
			_, _ = tx.Exec(ctx, statement, sessionID)
			_ = tx.Commit(ctx)
		}
	})
	return testSession{ID: sessionID, VillageID: villageID}
}

func newTestAppender(t *testing.T, pool *storage.Pool, options ...AppenderOption) *PostgresAppender {
	t.Helper()
	appender, err := NewPostgresAppender(pool, options...)
	if err != nil {
		t.Fatal(err)
	}
	return appender
}

func candidateFor(session, eventID string, actor domain.ActorKey, action commands.Action) DurableCandidate {
	return DurableCandidate{
		SessionID:       session,
		Provider:        "mock",
		ProviderEventID: eventID,
		Fingerprint:     payloadFingerprint(actor, source.KindComment, action),
		ActorKey:        actor,
		Action:          action,
	}
}

func TestDurableSingleAppend(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "waiting", 1, -1, 0)
	appender := newTestAppender(t, pool)
	actor := actorFor(1)
	action := mustAction(t, "XÂY")
	results, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "event-1", actor, action),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].InputSeq != 1 || results[0].WaveNo != 1 || results[0].ScheduledTick != 0 {
		t.Fatalf("results = %+v", results)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 1 {
		t.Fatalf("input events = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_receipts WHERE session_id = $1`, session.ID); got != 1 {
		t.Fatalf("receipts = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT input_sequence FROM ttg.sessions WHERE id = $1`, session.ID); got != 1 {
		t.Fatalf("session sequence = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1 AND applied_at_tick IS NULL`, session.ID); got != 1 {
		t.Fatalf("applied_at_tick must stay NULL, got %d", got)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_receipts WHERE session_id = $1 AND payload_fingerprint IS NOT NULL AND input_seq = 1`, session.ID); got != 1 {
		t.Fatalf("receipt association missing")
	}
	var payload string
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `SELECT canonical_payload::text FROM ttg.input_events WHERE session_id = $1`, session.ID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if payload != `{"kind": "role", "role": "builder"}` {
		t.Fatalf("payload = %s", payload)
	}
}

func TestDurableBatch256(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 100, 0)
	appender := newTestAppender(t, pool)
	actor := actorFor(2)
	action := mustAction(t, "THỦ")
	candidates := make([]DurableCandidate, 256)
	for index := range candidates {
		candidates[index] = candidateFor(session.ID, fmt.Sprintf("batch-%d", index), actor, action)
	}
	results, err := appender.AppendSession(context.Background(), session.ID, candidates)
	if err != nil {
		t.Fatal(err)
	}
	for index, result := range results {
		if result.InputSeq != int64(index+1) || result.ScheduledTick != 101 || result.WaveNo != 1 {
			t.Fatalf("result %d = %+v", index, result)
		}
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 256 {
		t.Fatalf("events = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT input_sequence FROM ttg.sessions WHERE id = $1`, session.ID); got != 256 {
		t.Fatalf("sequence = %d", got)
	}
}

func TestDurableSchedulingBoundaries(t *testing.T) {
	pool := testPool(t)
	appender := newTestAppender(t, pool)
	action := mustAction(t, "BUILD")
	actor := actorFor(3)
	cases := []struct {
		name      string
		lifecycle string
		wave      int64
		tick      int32
		wantWave  int64
		wantTick  int32
	}{
		{"waiting", "waiting", 1, -1, 1, 0},
		{"tick799", "running", 1, 799, 1, 800},
		{"tick1198", "running", 1, 1198, 1, 1199},
		{"tick1199", "running", 1, 1199, 1, 1200},
		{"tick1598", "running", 1, 1598, 1, 1599},
		{"tick1599", "running", 1, 1599, 1, 1600},
		{"tick2599", "running", 1, 2599, 1, 2600},
		{"tick2799", "running", 1, 2799, 1, 2800},
		{"results2800", "running", 1, 2800, 2, 0},
		{"tick2999", "running", 1, 2999, 2, 0},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			session := createSession(t, pool, "mock", testCase.lifecycle, testCase.wave, testCase.tick, 0)
			results, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
				candidateFor(session.ID, "sched-"+testCase.name, actor, action),
			})
			if err != nil {
				t.Fatal(err)
			}
			if results[0].WaveNo != testCase.wantWave || results[0].ScheduledTick != testCase.wantTick {
				t.Fatalf("schedule = wave %d tick %d, want wave %d tick %d", results[0].WaveNo, results[0].ScheduledTick, testCase.wantWave, testCase.wantTick)
			}
			if results[0].AcceptedTick != testCase.tick {
				t.Fatalf("accepted tick = %d, want %d", results[0].AcceptedTick, testCase.tick)
			}
		})
	}
}

func TestDurableDuplicateAndConflict(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 10, 0)
	appender := newTestAppender(t, pool)
	actor := actorFor(4)
	builder := mustAction(t, "XÂY")
	fighter := mustAction(t, "ATTACK")

	first, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "dup-1", actor, builder),
	})
	if err != nil || first[0].InputSeq != 1 {
		t.Fatalf("first = %+v/%v", first, err)
	}
	duplicate, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "dup-1", actor, builder),
	})
	if err != nil || !duplicate[0].Duplicate || duplicate[0].InputSeq != 1 {
		t.Fatalf("duplicate = %+v/%v", duplicate, err)
	}
	conflict, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "dup-1", actor, fighter),
	})
	if err != nil || !conflict[0].Conflict {
		t.Fatalf("conflict = %+v/%v", conflict, err)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 1 {
		t.Fatalf("conflict allocated an input event: %d", got)
	}
}
