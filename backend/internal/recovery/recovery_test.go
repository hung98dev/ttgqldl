package recovery

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
)

func testPool(t *testing.T) *storage.Pool {
	t.Helper()
	dsn := os.Getenv("TTG_TEST_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		if os.Getenv("TTG_REQUIRE_DATABASE") == "1" {
			t.Fatal("TTG_TEST_DATABASE_URL is required but not set")
		}
		t.Skip("TTG_TEST_DATABASE_URL is not set")
	}
	pool, err := storage.Open(context.Background(), storage.Options{DSN: dsn, MaxConns: 8, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type testIDs struct {
	VillageID string
	SessionID string
}

func createRecoverySession(t *testing.T, pool *storage.Pool) testIDs {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var villageID, sessionID string
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.villages (id, broadcaster_key)
		VALUES (gen_random_uuid(), md5(random()::text) || md5(random()::text)) RETURNING id::text`).Scan(&villageID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version)
		VALUES (gen_random_uuid(), $1, 'mock', 'waiting', 'ttg-rules-v1') RETURNING id::text`, villageID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	payload := `{"kind":"role","role":"builder"}`
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.input_events
		(session_id, input_seq, accepted_tick, scheduled_tick, wave_no, accepted_wave, actor_key, kind, canonical_payload, rules_version)
		VALUES ($1, 1, -1, 0, 1, 1, $2, 'role', $3::jsonb, 'ttg-rules-v1')`, sessionID, strings.Repeat("a", 64), payload); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE ttg.sessions SET input_sequence = 1 WHERE id = $1`, sessionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		tx, err := pool.Begin(cleanupCtx)
		if err != nil {
			return
		}
		for _, statement := range []string{
			`DELETE FROM ttg.stream_outbox WHERE session_id = $1`,
			`DELETE FROM ttg.domain_events WHERE session_id = $1`,
			`DELETE FROM ttg.checkpoints WHERE session_id = $1`,
			`DELETE FROM ttg.reducer_transitions WHERE session_id = $1`,
			`DELETE FROM ttg.input_receipts WHERE session_id = $1`,
			`DELETE FROM ttg.input_events WHERE session_id = $1`,
			`DELETE FROM ttg.wave_members WHERE session_id = $1`,
			`DELETE FROM ttg.ballots WHERE session_id = $1`,
			`DELETE FROM ttg.session_player_progress WHERE session_id = $1`,
			`DELETE FROM ttg.sessions WHERE id = $1`,
		} {
			_, _ = tx.Exec(cleanupCtx, statement, sessionID)
		}
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.writer_leases WHERE village_id = $1`, villageID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.villages WHERE id = $1`, villageID)
		_ = tx.Commit(cleanupCtx)
	})
	return testIDs{VillageID: villageID, SessionID: sessionID}
}

func newCoordinator(t *testing.T, pool *storage.Pool, ids testIDs, instance string, lease time.Duration) *Coordinator {
	t.Helper()
	manager, err := writer.New(pool, instance, writer.Config{LeaseDuration: lease})
	if err != nil {
		t.Fatal(err)
	}
	red, err := reducer.New(manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := New(pool, manager, red, ids.SessionID, ids.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	return coordinator
}

func sessionRevision(t *testing.T, pool *storage.Pool, sessionID string) (int64, int32, string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var revision int64
	var tick int32
	var hash *string
	if err := tx.QueryRow(ctx, `SELECT reducer_revision, wave_tick, last_state_hash FROM ttg.sessions WHERE id = $1`, sessionID).Scan(&revision, &tick, &hash); err != nil {
		t.Fatal(err)
	}
	value := ""
	if hash != nil {
		value = *hash
	}
	return revision, tick, value
}

func TestCoordinatorRecoversPristineAndCommittedState(t *testing.T) {
	pool := testPool(t)
	ids := createRecoverySession(t, pool)
	first := newCoordinator(t, pool, ids, "recovery-a", 2*time.Second)
	result, err := first.Recover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pristine || first.State() != StateReady {
		t.Fatalf("pristine recovery = %+v state=%s", result, first.State())
	}
	if _, err := first.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	revision, tick, hash := sessionRevision(t, pool, ids.SessionID)
	if revision != 1 || tick != 0 {
		t.Fatalf("committed = %d/%d", revision, tick)
	}
	if err := first.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	second := newCoordinator(t, pool, ids, "recovery-b", 2*time.Second)
	recovered, err := second.Recover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Pristine || recovered.Revision != revision || recovered.Tick != int(tick) || recovered.StateHash != hash {
		t.Fatalf("recovered = %+v", recovered)
	}
	if _, err := second.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	nextRevision, nextTick, _ := sessionRevision(t, pool, ids.SessionID)
	if nextRevision != revision+1 || nextTick != tick+1 {
		t.Fatalf("continued = %d/%d", nextRevision, nextTick)
	}
}

func TestCoordinatorRejectsCorruptCheckpoint(t *testing.T) {
	pool := testPool(t)
	ids := createRecoverySession(t, pool)
	first := newCoordinator(t, pool, ids, "recovery-a", 2*time.Second)
	if _, err := first.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE ttg.checkpoints SET state_hash = repeat('0', 64) WHERE session_id = $1`, ids.SessionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	second := newCoordinator(t, pool, ids, "recovery-b", 2*time.Second)
	if _, err := second.Recover(context.Background()); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("corrupt checkpoint recovery = %v", err)
	}
	if second.State() != StateIntegrityFail {
		t.Fatalf("state = %s", second.State())
	}
}

func TestHundredAbandonedLeaseRecoveryPositions(t *testing.T) {
	pool := testPool(t)
	ids := createRecoverySession(t, pool)
	for index := 0; index < 100; index++ {
		// Each iteration abandons the previous lease without a graceful release, then
		// waits for PostgreSQL-authoritative expiry and takes over with a new epoch.
		coordinator := newCoordinator(t, pool, ids, fmt.Sprintf("recovery-%03d", index), writer.MinLeaseDuration)
		if _, err := coordinator.Recover(context.Background()); err != nil {
			t.Fatalf("recovery %d: %v", index, err)
		}
		if _, err := coordinator.Step(context.Background()); err != nil {
			t.Fatalf("step %d: %v", index, err)
		}
		waitLeaseExpiry(t, pool, ids.VillageID)
	}
	revision, _, _ := sessionRevision(t, pool, ids.SessionID)
	if revision != 100 {
		t.Fatalf("revision = %d", revision)
	}
}

func TestRecoveryHelperProcess(t *testing.T) {
	if os.Getenv("TTG_RECOVERY_CHILD") != "1" {
		t.Skip("helper process only")
	}
	dsn := os.Getenv("TTG_TEST_DATABASE_URL")
	sessionID := os.Getenv("TTG_RECOVERY_SESSION")
	villageID := os.Getenv("TTG_RECOVERY_VILLAGE")
	mode := os.Getenv("TTG_RECOVERY_MODE")
	instance := os.Getenv("TTG_RECOVERY_INSTANCE")
	leaseMS := 1000
	fmt.Sscanf(os.Getenv("TTG_RECOVERY_LEASE_MS"), "%d", &leaseMS)
	pool, err := storage.Open(context.Background(), storage.Options{DSN: dsn, MaxConns: 4, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	coordinator := newCoordinator(t, pool, testIDs{VillageID: villageID, SessionID: sessionID}, instance, time.Duration(leaseMS)*time.Millisecond)
	result, err := coordinator.Recover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mode == "pre" {
		fmt.Println("READY")
		select {}
	}
	if mode == "recover" {
		fmt.Printf("RECOVERED revision=%d tick=%d\n", result.Revision, result.Tick)
		continued, err := coordinator.Step(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("CONTINUED revision=%d tick=%d hash=%s\n", continued.Revision, continued.Tick, continued.StateHash)
		if err := coordinator.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		return
	}
	if _, err := coordinator.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	ticks := 3
	fmt.Sscanf(os.Getenv("TTG_RECOVERY_TICKS"), "%d", &ticks)
	for index := 1; index < ticks; index++ {
		if _, err := coordinator.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	revision, tick, hash := sessionRevision(t, pool, sessionID)
	fmt.Printf("COMMITTED revision=%d tick=%d hash=%s\n", revision, tick, hash)
	select {}
}

func spawnRecoveryChild(t *testing.T, ids testIDs, mode, instance string) (*exec.Cmd, *bufio.Scanner) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestRecoveryHelperProcess")
	cmd.Env = append(os.Environ(),
		"TTG_RECOVERY_CHILD=1", "TTG_RECOVERY_SESSION="+ids.SessionID,
		"TTG_RECOVERY_VILLAGE="+ids.VillageID, "TTG_RECOVERY_MODE="+mode,
		"TTG_RECOVERY_INSTANCE="+instance, "TTG_RECOVERY_LEASE_MS=1000", "TTG_RECOVERY_TICKS=4")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd, bufio.NewScanner(stdout)
}

func waitLine(t *testing.T, scanner *bufio.Scanner, prefix string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			return line
		}
		if time.Now().After(deadline) {
			break
		}
	}
	t.Fatalf("did not see %s", prefix)
	return ""
}

func waitLeaseExpiry(t *testing.T, pool *storage.Pool, villageID string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ctx := context.Background()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var expired bool
		_ = tx.QueryRow(ctx, `SELECT leased_until IS NULL OR leased_until <= clock_timestamp() FROM ttg.writer_leases WHERE village_id = $1`, villageID).Scan(&expired)
		_ = tx.Rollback(ctx)
		if expired {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("lease did not expire")
}

func TestRealProcessKillRecoveryAfterCommit(t *testing.T) {
	pool := testPool(t)
	ids := createRecoverySession(t, pool)
	childA, scannerA := spawnRecoveryChild(t, ids, "commit", "process-a")
	defer func() { _ = childA.Process.Kill(); _, _ = childA.Process.Wait() }()
	committed := waitLine(t, scannerA, "COMMITTED", 10*time.Second)
	if err := childA.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_, _ = childA.Process.Wait()
	revisionBefore, tickBefore, hashBefore := sessionRevision(t, pool, ids.SessionID)
	if !strings.Contains(committed, fmt.Sprintf("revision=%d", revisionBefore)) {
		t.Fatalf("child marker %q vs revision %d", committed, revisionBefore)
	}
	outboxBefore := countRecoveryRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1`, ids.SessionID)
	waitLeaseExpiry(t, pool, ids.VillageID)
	childB, scannerB := spawnRecoveryChild(t, ids, "recover", "process-b")
	defer func() { _ = childB.Process.Kill(); _, _ = childB.Process.Wait() }()
	recovered := waitLine(t, scannerB, "RECOVERED", 15*time.Second)
	continued := waitLine(t, scannerB, "CONTINUED", 15*time.Second)
	if err := childB.Wait(); err != nil {
		t.Fatalf("recovery child: %v", err)
	}
	revisionAfter, tickAfter, _ := sessionRevision(t, pool, ids.SessionID)
	if !strings.Contains(recovered, fmt.Sprintf("revision=%d", revisionBefore)) || !strings.Contains(recovered, fmt.Sprintf("tick=%d", tickBefore)) {
		t.Fatalf("recovered marker %q before=%d/%d", recovered, revisionBefore, tickBefore)
	}
	if !strings.Contains(continued, fmt.Sprintf("revision=%d", revisionBefore+1)) || !strings.Contains(continued, fmt.Sprintf("tick=%d", tickBefore+1)) {
		t.Fatalf("continued marker %q", continued)
	}
	if revisionAfter != revisionBefore+1 || tickAfter != tickBefore+1 {
		t.Fatalf("after = %d/%d", revisionAfter, tickAfter)
	}
	if outboxBefore != countRecoveryRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1`, ids.SessionID) {
		t.Fatal("unexpected outbox change")
	}
	if hashBefore == "" {
		t.Fatal("missing pre-crash hash")
	}
}

func TestRealProcessKillBeforeTransaction(t *testing.T) {
	pool := testPool(t)
	ids := createRecoverySession(t, pool)
	childA, scannerA := spawnRecoveryChild(t, ids, "pre", "process-pre-a")
	defer func() { _ = childA.Process.Kill(); _, _ = childA.Process.Wait() }()
	_ = waitLine(t, scannerA, "READY", 10*time.Second)
	if err := childA.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_, _ = childA.Process.Wait()
	revision, tick, _ := sessionRevision(t, pool, ids.SessionID)
	if revision != 0 || tick != -1 {
		t.Fatalf("pre-crash state = %d/%d", revision, tick)
	}
	waitLeaseExpiry(t, pool, ids.VillageID)
	childB, scannerB := spawnRecoveryChild(t, ids, "recover", "process-pre-b")
	defer func() { _ = childB.Process.Kill(); _, _ = childB.Process.Wait() }()
	recovered := waitLine(t, scannerB, "RECOVERED", 15*time.Second)
	continued := waitLine(t, scannerB, "CONTINUED", 15*time.Second)
	if err := childB.Wait(); err != nil {
		t.Fatalf("recovery child: %v", err)
	}
	if !strings.Contains(recovered, "revision=0") || !strings.Contains(recovered, "tick=-1") {
		t.Fatalf("recovered pristine = %q", recovered)
	}
	if !strings.Contains(continued, "revision=1") || !strings.Contains(continued, "tick=0") {
		t.Fatalf("continued pristine = %q", continued)
	}
}

func countRecoveryRows(t *testing.T, pool *storage.Pool, statement string, args ...any) int64 {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var value int64
	if err := tx.QueryRow(ctx, statement, args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}
