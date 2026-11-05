package writer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/jackc/pgx/v5"
)

type testVillage struct {
	VillageID string
	SessionID string
}

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

func createVillage(t *testing.T, pool *storage.Pool, lifecycle string) testVillage {
	t.Helper()
	village := createVillageWithoutSession(t, pool)
	if lifecycle != "" {
		village.SessionID = createSession(t, pool, village.VillageID, lifecycle)
	}
	return village
}

func createVillageWithoutSession(t *testing.T, pool *storage.Pool) testVillage {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var villageID string
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.villages (id, broadcaster_key)
		VALUES (gen_random_uuid(), md5(random()::text) || md5(random()::text)) RETURNING id::text`).Scan(&villageID); err != nil {
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
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.writer_leases WHERE village_id = $1`, villageID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.sessions WHERE village_id = $1`, villageID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.villages WHERE id = $1`, villageID)
		_ = tx.Commit(cleanupCtx)
	})
	return testVillage{VillageID: villageID}
}

func createSession(t *testing.T, pool *storage.Pool, villageID, lifecycle string) string {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var sessionID string
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version)
		VALUES (gen_random_uuid(), $1, 'mock', $2, 'ttg-rules-v1') RETURNING id::text`, villageID, lifecycle).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return sessionID
}

func dbExec(t *testing.T, pool *storage.Pool, statement string, args ...any) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, statement, args...); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func dbScalarInt(t *testing.T, pool *storage.Pool, statement string, args ...any) int64 {
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

func dbScalarString(t *testing.T, pool *storage.Pool, statement string, args ...any) string {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var value string
	if err := tx.QueryRow(ctx, statement, args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func dbScalarTime(t *testing.T, pool *storage.Pool, statement string, args ...any) time.Time {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var value time.Time
	if err := tx.QueryRow(ctx, statement, args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func dbClock(t *testing.T, pool *storage.Pool) time.Time {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var now time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		t.Fatal(err)
	}
	return now
}

func waitFor(t *testing.T, timeout time.Duration, description string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func newTestManager(t *testing.T, pool *storage.Pool, instance string, duration time.Duration) *Manager {
	t.Helper()
	manager, err := New(pool, instance, Config{LeaseDuration: duration})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func TestAcquireFirstLeaseAndSessionEpoch(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", 2*time.Second)
	lease, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Epoch != 1 || lease.OwnerInstance != "instance-a" || lease.VillageID != village.VillageID || lease.SessionID != village.SessionID {
		t.Fatalf("lease = %+v", lease)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != 1 {
		t.Fatalf("lease epoch = %d", got)
	}
	if got := dbScalarString(t, pool, `SELECT owner_instance FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != "instance-a" {
		t.Fatalf("owner = %q", got)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.sessions WHERE id = $1`, village.SessionID); got != 1 {
		t.Fatalf("session epoch = %d", got)
	}
	if !manager.Status().Verified || manager.Epoch() != 1 {
		t.Fatalf("status = %+v epoch=%d", manager.Status(), manager.Epoch())
	}
}

func TestAcquireSameOwnerKeepsEpoch(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", time.Second)
	first, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	if second.Epoch != first.Epoch || second.Epoch != 1 {
		t.Fatalf("re-acquire epoch = %d, first = %d", second.Epoch, first.Epoch)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != 1 {
		t.Fatalf("lease epoch = %d", got)
	}
}

func TestCompetingAcquisitionsHaveOneOwner(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	const contenders = 8
	managers := make([]*Manager, contenders)
	for index := range managers {
		managers[index] = newTestManager(t, pool, fmt.Sprintf("instance-%d", index), 2*time.Second)
	}
	start := make(chan struct{})
	results := make(chan error, contenders)
	var wait sync.WaitGroup
	for _, manager := range managers {
		wait.Add(1)
		go func(manager *Manager) {
			defer wait.Done()
			<-start
			_, err := manager.Acquire(context.Background(), village.VillageID)
			results <- err
		}(manager)
	}
	close(start)
	wait.Wait()
	close(results)
	successes, held := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrLeaseHeld):
			held++
		default:
			t.Fatalf("unexpected acquire error: %v", err)
		}
	}
	if successes != 1 || held != contenders-1 {
		t.Fatalf("successes=%d held=%d", successes, held)
	}
	if got := dbScalarInt(t, pool, `SELECT count(*) FROM ttg.writer_leases WHERE village_id = $1 AND writer_epoch = 1`, village.VillageID); got != 1 {
		t.Fatalf("epoch-one rows = %d", got)
	}
}

func TestRenewKeepsEpochAndRejectsStaleOwners(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	owner := newTestManager(t, pool, "instance-a", 2*time.Second)
	lease, err := owner.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	before := dbScalarString(t, pool, `SELECT leased_until::text FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID)
	renewed, err := owner.Renew(context.Background(), lease)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.Epoch != lease.Epoch {
		t.Fatalf("renew changed epoch %d -> %d", lease.Epoch, renewed.Epoch)
	}
	after := dbScalarString(t, pool, `SELECT leased_until::text FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID)
	if before == after {
		t.Fatalf("renew did not advance expiry: %s", before)
	}
	other := newTestManager(t, pool, "instance-b", 2*time.Second)
	if _, err := other.Renew(context.Background(), lease); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("wrong owner renew = %v", err)
	}
	stale := lease
	stale.Epoch++
	if _, err := owner.Renew(context.Background(), stale); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("wrong epoch renew = %v", err)
	}
}

func TestExpiredLeaseDoesNotAuthorizeRenewOrWrite(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	owner := newTestManager(t, pool, "instance-a", MinLeaseDuration)
	lease, err := owner.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 2*time.Second, "database lease expiry", func() bool {
		now := dbClock(t, pool)
		expires := dbScalarTime(t, pool, `SELECT leased_until FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID)
		return now.After(expires)
	})
	if _, err := owner.Renew(context.Background(), lease); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("renew expired = %v", err)
	}
	called := false
	err = owner.WithFence(context.Background(), lease, village.SessionID, func(context.Context, FencedTx) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrLeaseExpired) || called {
		t.Fatalf("expired fence = %v called=%v", err, called)
	}
}

func TestTakeoverInvalidatesOldWriter(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	ownerA := newTestManager(t, pool, "instance-a", MinLeaseDuration)
	leaseA, err := ownerA.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 2*time.Second, "database lease expiry", func() bool {
		now := dbClock(t, pool)
		expires := dbScalarTime(t, pool, `SELECT leased_until FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID)
		return now.After(expires)
	})
	ownerB := newTestManager(t, pool, "instance-b", 2*time.Second)
	leaseB, err := ownerB.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	if leaseB.Epoch <= leaseA.Epoch {
		t.Fatalf("takeover epoch = %d, old = %d", leaseB.Epoch, leaseA.Epoch)
	}
	if _, err := ownerA.Renew(context.Background(), leaseA); !errors.Is(err, ErrLeaseLost) && !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("old renew = %v", err)
	}
	if err := ownerA.Release(context.Background(), leaseA); !errors.Is(err, ErrLeaseLost) && !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("old release = %v", err)
	}
	called := false
	err = ownerA.WithFence(context.Background(), leaseA, village.SessionID, func(context.Context, FencedTx) error {
		called = true
		return nil
	})
	if err == nil || called {
		t.Fatalf("old fence = %v called=%v", err, called)
	}
	written := false
	if err := ownerB.WithFence(context.Background(), leaseB, village.SessionID, func(_ context.Context, tx FencedTx) error {
		written = true
		_, err := tx.Exec(context.Background(), `UPDATE ttg.sessions SET last_heartbeat_at = clock_timestamp() WHERE id = $1`, village.SessionID)
		return err
	}); err != nil || !written {
		t.Fatalf("new fence = %v written=%v", err, written)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != leaseB.Epoch {
		t.Fatalf("lease epoch = %d", got)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.sessions WHERE id = $1`, village.SessionID); got != leaseB.Epoch {
		t.Fatalf("session epoch = %d", got)
	}
}

func leaseWaiterCount(t *testing.T, pool *storage.Pool) int64 {
	t.Helper()
	return dbScalarInt(t, pool, `SELECT count(*) FROM pg_stat_activity
		WHERE datname = current_database()
		  AND wait_event_type = 'Lock'
		  AND query ILIKE '%writer_leases%'`)
}

func TestDelayedStaleTransactionCannotCommitAfterTakeover(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	ownerA := newTestManager(t, pool, "instance-a", MinLeaseDuration)
	leaseA, err := ownerA.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 2*time.Second, "lease expiry", func() bool {
		return dbClock(t, pool).After(dbScalarTime(t, pool, `SELECT leased_until FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID))
	})

	ownerB := newTestManager(t, pool, "instance-b", 2*time.Second)
	reached := make(chan struct{})
	release := make(chan struct{})
	ownerB.beforeCommit = func() error {
		close(reached)
		<-release
		return nil
	}
	type acquireResult struct {
		lease Lease
		err   error
	}
	bResult := make(chan acquireResult, 1)
	go func() {
		lease, err := ownerB.Acquire(context.Background(), village.VillageID)
		bResult <- acquireResult{lease: lease, err: err}
	}()
	<-reached

	var callbackRan atomic.Bool
	aResult := make(chan error, 1)
	go func() {
		aResult <- ownerA.WithFence(context.Background(), leaseA, village.SessionID, func(context.Context, FencedTx) error {
			callbackRan.Store(true)
			return nil
		})
	}()
	waitFor(t, 2*time.Second, "stale fenced transaction to wait on lease", func() bool {
		return leaseWaiterCount(t, pool) >= 1
	})
	close(release)
	b := <-bResult
	if b.err != nil || b.lease.Epoch <= leaseA.Epoch {
		t.Fatalf("takeover = %+v/%v", b.lease, b.err)
	}
	if err := <-aResult; err == nil {
		t.Fatal("stale fenced transaction committed after takeover")
	}
	if callbackRan.Load() {
		t.Fatal("stale callback executed")
	}
}

func TestDelayedLockWaiterUsesDatabaseClockAfterLock(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	ownerA := newTestManager(t, pool, "instance-a", 250*time.Millisecond)
	leaseA, err := ownerA.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blocker.Exec(ctx, `SELECT 1 FROM ttg.writer_leases WHERE village_id = $1 FOR UPDATE`, village.VillageID); err != nil {
		_ = blocker.Rollback(ctx)
		t.Fatal(err)
	}
	ownerB := newTestManager(t, pool, "instance-b", 2*time.Second)
	result := make(chan error, 1)
	go func() {
		_, err := ownerB.Acquire(ctx, village.VillageID)
		result <- err
	}()
	waitFor(t, 2*time.Second, "acquirer blocked on lease row", func() bool {
		return leaseWaiterCount(t, pool) >= 1
	})
	waitFor(t, 2*time.Second, "database lease expiry while blocked", func() bool {
		return dbClock(t, pool).After(dbScalarTime(t, pool, `SELECT leased_until FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID))
	})
	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatalf("takeover after delayed lock = %v", err)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != leaseA.Epoch+1 {
		t.Fatalf("epoch after delayed lock = %d", got)
	}
}

func TestRollbackLeavesNoOwnershipGeneration(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", time.Second)
	manager.beforeCommit = func() error { return errors.New("injected rollback") }
	if _, err := manager.Acquire(context.Background(), village.VillageID); err == nil {
		t.Fatal("acquire unexpectedly succeeded")
	}
	if got := dbScalarInt(t, pool, `SELECT count(*) FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != 0 {
		t.Fatalf("lease row survived rollback: %d", got)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.sessions WHERE id = $1`, village.SessionID); got != 0 {
		t.Fatalf("session epoch = %d", got)
	}
	if manager.Status().Verified {
		t.Fatal("rollback left a verified local lease")
	}
}

func TestUnknownCommitReconcilesWithoutNewEpoch(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", time.Second)
	manager.commit = func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return errors.New("lost acknowledgment")
	}
	if _, err := manager.Acquire(context.Background(), village.VillageID); !errors.Is(err, ErrCommitUnknown) {
		t.Fatalf("unknown commit = %v", err)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != 1 {
		t.Fatalf("committed epoch = %d", got)
	}
	if manager.Status().Verified {
		t.Fatal("unknown commit left a verified lease")
	}
	manager.commit = nil
	lease, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Epoch != 1 {
		t.Fatalf("reconciliation allocated epoch %d", lease.Epoch)
	}
}

func TestClosedPoolIsStorageUnavailable(t *testing.T) {
	pool := testPool(t)
	dsn := os.Getenv("TTG_TEST_DATABASE_URL")
	broken, err := storage.Open(context.Background(), storage.Options{DSN: dsn, MaxConns: 1, ConnectTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	broken.Close()
	manager, err := New(broken, "instance-a", Config{LeaseDuration: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Acquire(context.Background(), "00000000-0000-4000-8000-000000000001"); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("closed pool acquire = %v", err)
	}
	if manager.Status().Verified {
		t.Fatal("closed pool reported verified ownership")
	}
	_ = pool
}

func TestReleaseAndTakeoverSemantics(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	ownerA := newTestManager(t, pool, "instance-a", 2*time.Second)
	leaseA, err := ownerA.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	ownerB := newTestManager(t, pool, "instance-b", 2*time.Second)
	if err := ownerB.Release(context.Background(), leaseA); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("wrong owner release = %v", err)
	}
	if err := ownerA.Release(context.Background(), leaseA); err != nil {
		t.Fatal(err)
	}
	if got := dbScalarString(t, pool, `SELECT coalesce(owner_instance, '') FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != "" {
		t.Fatalf("owner after release = %q", got)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.sessions WHERE id = $1`, village.SessionID); got != 0 {
		t.Fatalf("session epoch after release = %d", got)
	}
	leaseB, err := ownerB.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	if leaseB.Epoch != leaseA.Epoch+1 {
		t.Fatalf("post-release epoch = %d", leaseB.Epoch)
	}
	if _, err := ownerA.Renew(context.Background(), leaseA); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("stale renew after release = %v", err)
	}
	called := false
	if err := ownerA.WithFence(context.Background(), leaseA, village.SessionID, func(context.Context, FencedTx) error {
		called = true
		return nil
	}); err == nil || called {
		t.Fatalf("stale fence after release = %v called=%v", err, called)
	}
}

func TestCloseIsConditionalAndBounded(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", 2*time.Second)
	if _, err := manager.Acquire(context.Background(), village.VillageID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := manager.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if manager.Epoch() != 0 || manager.Status().Verified {
		t.Fatalf("close left verified state: %+v", manager.Status())
	}
	if got := dbScalarString(t, pool, `SELECT coalesce(owner_instance, '') FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != "" {
		t.Fatalf("owner after close = %q", got)
	}
	if err := manager.Close(ctx); err != nil {
		t.Fatalf("second close = %v", err)
	}
}

func TestVillageIsolation(t *testing.T) {
	pool := testPool(t)
	villageA := createVillage(t, pool, "running")
	villageB := createVillage(t, pool, "running")
	ownerA := newTestManager(t, pool, "instance-a", 2*time.Second)
	ownerB := newTestManager(t, pool, "instance-b", 2*time.Second)
	leaseA, err := ownerA.Acquire(context.Background(), villageA.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	leaseB, err := ownerB.Acquire(context.Background(), villageB.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	if leaseA.Epoch != 1 || leaseB.Epoch != 1 {
		t.Fatalf("isolated epochs = %d/%d", leaseA.Epoch, leaseB.Epoch)
	}
	if _, err := ownerB.Acquire(context.Background(), villageA.VillageID); !errors.Is(err, ErrManagerBusy) {
		t.Fatalf("manager busy acquire = %v", err)
	}
	ownerC := newTestManager(t, pool, "instance-c", 2*time.Second)
	if _, err := ownerC.Acquire(context.Background(), villageA.VillageID); !errors.Is(err, ErrLeaseHeld) {
		t.Fatalf("cross-village lease acquire = %v", err)
	}
	if err := ownerB.WithFence(context.Background(), leaseB, villageB.SessionID, func(context.Context, FencedTx) error { return nil }); err != nil {
		t.Fatalf("village B fence after A contention = %v", err)
	}
}

func TestSessionLifecycleAndRecreation(t *testing.T) {
	pool := testPool(t)
	for _, lifecycle := range []string{"waiting", "running", "paused", "source_disconnected", "storage_unavailable"} {
		t.Run(lifecycle, func(t *testing.T) {
			village := createVillage(t, pool, lifecycle)
			manager := newTestManager(t, pool, "instance-"+lifecycle, 2*time.Second)
			lease, err := manager.Acquire(context.Background(), village.VillageID)
			if err != nil {
				t.Fatal(err)
			}
			if lease.SessionID != village.SessionID {
				t.Fatalf("session = %q", lease.SessionID)
			}
			if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.sessions WHERE id = $1`, village.SessionID); got != lease.Epoch {
				t.Fatalf("session epoch = %d", got)
			}
			if err := manager.WithFence(context.Background(), lease, village.SessionID, func(context.Context, FencedTx) error { return nil }); err != nil {
				t.Fatalf("fence in %s: %v", lifecycle, err)
			}
		})
	}
	t.Run("stopped", func(t *testing.T) {
		village := createVillage(t, pool, "stopped")
		manager := newTestManager(t, pool, "instance-stopped", 2*time.Second)
		lease, err := manager.Acquire(context.Background(), village.VillageID)
		if err != nil {
			t.Fatal(err)
		}
		if lease.SessionID != "" {
			t.Fatalf("stopped session returned %q", lease.SessionID)
		}
		if err := manager.WithFence(context.Background(), lease, village.SessionID, func(context.Context, FencedTx) error { return nil }); !errors.Is(err, ErrSessionInactive) {
			t.Fatalf("stopped fence = %v", err)
		}
	})
	t.Run("recreated", func(t *testing.T) {
		village := createVillage(t, pool, "running")
		manager := newTestManager(t, pool, "instance-recreated", 2*time.Second)
		lease, err := manager.Acquire(context.Background(), village.VillageID)
		if err != nil {
			t.Fatal(err)
		}
		dbExec(t, pool, `UPDATE ttg.sessions SET lifecycle = 'stopped', stopped_at = clock_timestamp() WHERE id = $1`, village.SessionID)
		newSession := createSession(t, pool, village.VillageID, "waiting")
		renewed, err := manager.Acquire(context.Background(), village.VillageID)
		if err != nil {
			t.Fatal(err)
		}
		if renewed.SessionID != newSession || renewed.Epoch != lease.Epoch {
			t.Fatalf("recreated lease = %+v", renewed)
		}
		if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.sessions WHERE id = $1`, newSession); got != lease.Epoch {
			t.Fatalf("new session epoch = %d", got)
		}
	})
}

func TestFencedTransactionRollbackAndBinding(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	other := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", 2*time.Second)
	lease, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	before := dbScalarTime(t, pool, `SELECT last_heartbeat_at FROM ttg.sessions WHERE id = $1`, village.SessionID)
	err = manager.WithFence(context.Background(), lease, village.SessionID, func(ctx context.Context, tx FencedTx) error {
		if _, err := tx.Exec(ctx, `UPDATE ttg.sessions SET last_heartbeat_at = clock_timestamp() WHERE id = $1`, village.SessionID); err != nil {
			return err
		}
		return errors.New("injected callback failure")
	})
	if err == nil {
		t.Fatal("callback failure was not returned")
	}
	after := dbScalarTime(t, pool, `SELECT last_heartbeat_at FROM ttg.sessions WHERE id = $1`, village.SessionID)
	if !after.Equal(before) {
		t.Fatalf("callback write survived rollback: %s vs %s", before, after)
	}
	if err := manager.WithFence(context.Background(), lease, other.SessionID, func(context.Context, FencedTx) error { return nil }); !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("session mismatch = %v", err)
	}
	dbExec(t, pool, `UPDATE ttg.sessions SET writer_epoch = 0 WHERE id = $1`, village.SessionID)
	if err := manager.WithFence(context.Background(), lease, village.SessionID, func(context.Context, FencedTx) error { return nil }); !errors.Is(err, ErrStaleEpoch) {
		t.Fatalf("stale session epoch = %v", err)
	}
}

func TestEpochOverflowAndInvalidEpochFailClosed(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", 2*time.Second)
	if _, err := manager.Acquire(context.Background(), village.VillageID); err != nil {
		t.Fatal(err)
	}
	dbExec(t, pool, `UPDATE ttg.writer_leases SET writer_epoch = 9223372036854775807, owner_instance = NULL, leased_until = NULL WHERE village_id = $1`, village.VillageID)
	next := newTestManager(t, pool, "instance-b", 2*time.Second)
	if _, err := next.Acquire(context.Background(), village.VillageID); !errors.Is(err, ErrEpochExhausted) {
		t.Fatalf("overflow acquire = %v", err)
	}
	dbExec(t, pool, `UPDATE ttg.writer_leases SET writer_epoch = -1, owner_instance = NULL, leased_until = NULL WHERE village_id = $1`, village.VillageID)
	if _, err := next.Acquire(context.Background(), village.VillageID); !errors.Is(err, ErrInvalidEpoch) {
		t.Fatalf("negative epoch acquire = %v", err)
	}
}

func TestValidationConfigAndMissingVillage(t *testing.T) {
	pool := testPool(t)
	if _, err := New(nil, "instance", Config{}); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("nil pool = %v", err)
	}
	for _, instance := range []string{"", strings.Repeat("a", MaxInstanceBytes+1), "bad\x00instance"} {
		if _, err := New(pool, instance, Config{}); !errors.Is(err, ErrInvalidInstance) {
			t.Fatalf("instance %q error = %v", instance, err)
		}
	}
	for _, duration := range []time.Duration{time.Millisecond, MaxLeaseDuration + time.Second} {
		if _, err := New(pool, "instance", Config{LeaseDuration: duration}); !errors.Is(err, ErrInvalidDuration) {
			t.Fatalf("duration %s error = %v", duration, err)
		}
	}
	manager := newTestManager(t, pool, "instance-a", time.Second)
	if _, err := manager.Acquire(context.Background(), "not-a-uuid"); !errors.Is(err, ErrInvalidVillage) {
		t.Fatalf("invalid village = %v", err)
	}
	if _, err := manager.Acquire(context.Background(), "00000000-0000-4000-8000-000000000099"); !errors.Is(err, ErrVillageNotFound) {
		t.Fatalf("missing village = %v", err)
	}
}

func TestNoActiveSessionAndCallbackErrorPreservation(t *testing.T) {
	pool := testPool(t)
	village := createVillageWithoutSession(t, pool)
	manager := newTestManager(t, pool, "instance-a", time.Second)
	lease, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	if lease.SessionID != "" {
		t.Fatalf("session = %q", lease.SessionID)
	}
	if err := manager.WithFence(context.Background(), lease, "00000000-0000-4000-8000-000000000001", func(context.Context, FencedTx) error { return nil }); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("missing session fence = %v", err)
	}
	if err := manager.WithFence(context.Background(), lease, "not-a-session", func(context.Context, FencedTx) error { return nil }); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("invalid session fence = %v", err)
	}
	withSession := createVillage(t, pool, "running")
	secondManager := newTestManager(t, pool, "instance-b", time.Second)
	lease, err = secondManager.Acquire(context.Background(), withSession.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("business rule rejected")
	err = secondManager.WithFence(context.Background(), lease, withSession.SessionID, func(context.Context, FencedTx) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("callback error was not preserved: %v", err)
	}
}

func TestConcurrentRenewAndFencedWritesRemainSerialized(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", 2*time.Second)
	lease, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	const workers = 12
	var wait sync.WaitGroup
	errs := make(chan error, workers*2)
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := manager.Renew(context.Background(), lease); err != nil {
				errs <- err
			}
		}()
		wait.Add(1)
		go func() {
			defer wait.Done()
			err := manager.WithFence(context.Background(), lease, village.SessionID, func(ctx context.Context, tx FencedTx) error {
				_, err := tx.Exec(ctx, `UPDATE ttg.sessions SET last_heartbeat_at = clock_timestamp() WHERE id = $1`, village.SessionID)
				return err
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent operation: %v", err)
	}
	if got := dbScalarInt(t, pool, `SELECT writer_epoch FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID); got != lease.Epoch {
		t.Fatalf("epoch changed under concurrent operations: %d", got)
	}
}

func TestLocalExpiryStopsReadinessClaim(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", MinLeaseDuration)
	if _, err := manager.Acquire(context.Background(), village.VillageID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(MinLeaseDuration + 50*time.Millisecond)
	if manager.Epoch() != 0 || manager.Status().Verified {
		t.Fatalf("locally expired lease still claims readiness: %+v epoch=%d", manager.Status(), manager.Epoch())
	}
}

func TestSameOwnerReacquisitionAfterExpiryGetsNewEpoch(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", MinLeaseDuration)
	first, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 2*time.Second, "lease expiry", func() bool {
		return dbClock(t, pool).After(dbScalarTime(t, pool, `SELECT leased_until FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID))
	})
	second, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	if second.Epoch != first.Epoch+1 {
		t.Fatalf("same-owner reacquisition epoch = %d, first = %d", second.Epoch, first.Epoch)
	}
	if err := manager.Release(context.Background(), first); !errors.Is(err, ErrLeaseLost) && !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("old release after reacquisition = %v", err)
	}
}

func TestExpiredLeaseCannotBeReleased(t *testing.T) {
	pool := testPool(t)
	village := createVillage(t, pool, "running")
	manager := newTestManager(t, pool, "instance-a", MinLeaseDuration)
	lease, err := manager.Acquire(context.Background(), village.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 2*time.Second, "lease expiry", func() bool {
		return dbClock(t, pool).After(dbScalarTime(t, pool, `SELECT leased_until FROM ttg.writer_leases WHERE village_id = $1`, village.VillageID))
	})
	if err := manager.Release(context.Background(), lease); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expired release = %v", err)
	}
}
