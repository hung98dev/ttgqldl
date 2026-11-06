package pitr_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/db/migrations"
	"github.com/hung98dev/ttgqldl/backend/internal/pitr"
	"github.com/hung98dev/ttgqldl/backend/internal/privacy"
	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestRealPITRBackupBeforeErasureReapply(t *testing.T) {
	requireDocker(t)
	primary := unique("privacy-primary")
	pgdata := unique("privacy-pgdata")
	wal := unique("privacy-wal")
	backup := unique("privacy-backup")
	dir := t.TempDir()
	var restoreNames, restoreVolumes []string
	defer func() {
		_ = exec.Command("docker", "rm", "-f", primary).Run()
		for _, name := range restoreNames {
			_ = exec.Command("docker", "rm", "-f", name).Run()
		}
		for _, volume := range append([]string{pgdata, wal, backup}, restoreVolumes...) {
			_ = exec.Command("docker", "volume", "rm", "-f", volume).Run()
		}
	}()
	for _, volume := range []string{wal, backup} {
		docker(t, "volume", "create", volume)
		docker(t, "run", "--rm", "-u", "root", "-v", volume+":/v", image, "chown", "-R", "postgres:postgres", "/v")
	}
	archive := "set -e; tmp=/walarchive/.%f.tmp.$$; cp %p $tmp; sync $tmp; test ! -f /walarchive/%f; mv $tmp /walarchive/%f; sync"
	docker(t, "run", "-d", "--name", primary, "-e", "POSTGRES_PASSWORD=pitr_synthetic_password", "-e", "POSTGRES_DB=pitr",
		"-v", pgdata+":/var/lib/postgresql", "-v", wal+":/walarchive", "-v", backup+":/backup",
		"-p", "127.0.0.1::5432", image, "postgres", "-c", "wal_level=replica", "-c", "archive_mode=on", "-c", "archive_command="+archive, "-c", "archive_timeout=60")
	waitReady(t, primary, "postgres", "pitr")
	time.Sleep(2 * time.Second)
	port := hostPort(t, primary)
	dsn := fmt.Sprintf("postgres://postgres:pitr_synthetic_password@127.0.0.1:%s/pitr?sslmode=disable", port)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(db, "."); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	pool, err := storage.Open(ctx, storage.Options{DSN: dsn, MaxConns: 8, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var villageID, sessionID, playerID string
	actor := strings.Repeat("a", 64)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.villages (id, broadcaster_key) VALUES (gen_random_uuid(), md5(random()::text) || md5(random()::text)) RETURNING id::text`).Scan(&villageID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version) VALUES (gen_random_uuid(), $1, 'mock', 'waiting', 'ttg-rules-v1') RETURNING id::text`, villageID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.players (id, village_id, actor_key, ephemeral, alias) VALUES (gen_random_uuid(), $1, $2, false, 'SyntheticAlice') RETURNING id::text`, villageID, actor).Scan(&playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id) VALUES ($1, $2, 'v1', $3), ($1, $2, 'v2', $3)`, villageID, actor, playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.session_player_progress (session_id, actor_key, session_xp) VALUES ($1, $2, 42)`, sessionID, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.input_events (session_id, input_seq, accepted_tick, scheduled_tick, wave_no, accepted_wave, actor_key, kind, canonical_payload, rules_version) VALUES ($1, 1, -1, 0, 1, 1, $2, 'role', '{"kind":"role","role":"builder"}'::jsonb, 'ttg-rules-v1')`, sessionID, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE ttg.sessions SET input_sequence = 1 WHERE id = $1`, sessionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	manager, err := writer.New(pool, "privacy-primary", writer.Config{LeaseDuration: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	red, err := reducer.New(manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := manager.Acquire(ctx, villageID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := red.Step(ctx, lease, sessionID); err != nil {
		t.Fatal(err)
	}
	var revisionBefore int64
	if err := db.QueryRowContext(ctx, `SELECT reducer_revision FROM ttg.sessions WHERE id = $1`, sessionID).Scan(&revisionBefore); err != nil {
		t.Fatal(err)
	}

	// Physical backup BEFORE erasure.
	docker(t, "exec", "-u", "postgres", primary, "pg_basebackup", "-D", "/backup/base", "-Fp", "-X", "stream", "-U", "postgres", "-h", "127.0.0.1")
	docker(t, "exec", "-u", "postgres", primary, "tar", "-C", "/backup", "-cf", "/backup/base.tar", "base")
	hostTar := filepath.Join(dir, "base.tar")
	docker(t, "cp", primary+":/backup/base.tar", hostTar)
	var markerTime time.Time
	if err := db.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&markerTime); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	chain := make([]byte, 32)
	_, _ = rand.Read(key)
	_, _ = rand.Read(chain)
	encrypted := filepath.Join(dir, "base.tar.enc")
	manifest, err := pitr.EncryptFile(hostTar, encrypted, key)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := privacy.OpenLedger(filepath.Join(dir, "ledger"), "privacy-ledger", key, chain)
	if err != nil {
		t.Fatal(err)
	}
	service := &privacy.Service{Pool: pool, Ledger: ledger, Reducer: red}
	request, err := service.Request(ctx, sessionID, actor, "postbackup erasure", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Export(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Apply(ctx, lease, request.ID); err != nil {
		t.Fatal(err)
	}
	var tombstoneID string
	if err := db.QueryRowContext(ctx, `SELECT id::text FROM ttg.privacy_tombstones WHERE actor_ref_protected = $1 ORDER BY created_at DESC LIMIT 1`, actor).Scan(&tombstoneID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT pg_switch_wal()`); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	// Destroy primary data; backup and WAL remain independent.
	_ = exec.Command("docker", "rm", "-f", primary).Run()
	_ = exec.Command("docker", "volume", "rm", "-f", pgdata).Run()

	restoreName := unique("privacy-restore")
	restoreVolume := unique("privacy-restore-data")
	restoreNames = append(restoreNames, restoreName)
	restoreVolumes = append(restoreVolumes, restoreVolume)
	hostDir := t.TempDir()
	if err := os.Chmod(hostDir, 0o755); err != nil {
		t.Fatal(err)
	}
	baseTar := filepath.Join(hostDir, "base.tar")
	if _, err := pitr.DecryptFile(encrypted, baseTar, key); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(baseTar, 0o644); err != nil {
		t.Fatal(err)
	}
	docker(t, "volume", "create", restoreVolume)
	docker(t, "run", "--rm", "-u", "root", "-v", restoreVolume+":/var/lib/postgresql", image, "chown", "-R", "postgres:postgres", "/var/lib/postgresql")
	script := `set -e
mkdir -p /var/lib/postgresql/18/docker
tar -C /var/lib/postgresql/18/docker --strip-components=1 -xf /restore/base.tar
chmod 700 /var/lib/postgresql/18/docker
cat >> /var/lib/postgresql/18/docker/postgresql.auto.conf <<EOF
restore_command = 'cp /walarchive/%f %p'
recovery_target_time = '` + markerTime.UTC().Format("2006-01-02 15:04:05.999999+00") + `'
recovery_target_action = 'promote'
EOF
touch /var/lib/postgresql/18/docker/recovery.signal
exec postgres -D /var/lib/postgresql/18/docker`
	docker(t, "run", "-d", "--name", restoreName, "-e", "POSTGRES_PASSWORD=pitr_synthetic_password", "-v", restoreVolume+":/var/lib/postgresql", "-v", hostDir+":/restore:ro", "-v", wal+":/walarchive", "-p", "127.0.0.1::5432", "--user", "postgres", image, "bash", "-c", script)
	waitReady(t, restoreName, "postgres", "pitr")
	restoredPort := hostPort(t, restoreName)
	restoredDSN := fmt.Sprintf("postgres://postgres:pitr_synthetic_password@127.0.0.1:%s/pitr?sslmode=disable", restoredPort)
	_ = waitQuery(t, restoredDSN, `SELECT pg_is_in_recovery()::text`)
	restoredDB, err := sql.Open("pgx", restoredDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer restoredDB.Close()
	var restoredRevision int64
	var playerCount, aliasCount int
	if err := restoredDB.QueryRowContext(ctx, `SELECT reducer_revision FROM ttg.sessions WHERE id = $1`, sessionID).Scan(&restoredRevision); err != nil {
		t.Fatal(err)
	}
	if restoredRevision != revisionBefore {
		t.Fatalf("old backup revision=%d want %d", restoredRevision, revisionBefore)
	}
	if err := restoredDB.QueryRowContext(ctx, `SELECT count(*) FROM ttg.players WHERE village_id = $1`, villageID).Scan(&playerCount); err != nil {
		t.Fatal(err)
	}
	if err := restoredDB.QueryRowContext(ctx, `SELECT count(*) FROM ttg.actor_identity_aliases WHERE village_id = $1`, villageID).Scan(&aliasCount); err != nil {
		t.Fatal(err)
	}
	if playerCount != 1 || aliasCount != 2 {
		t.Fatalf("old backup did not contain pre-erasure subject: players=%d aliases=%d", playerCount, aliasCount)
	}

	// Reapply the independently durable deletion ledger to the restored cluster.
	restoredPool, err := storage.Open(ctx, storage.Options{DSN: restoredDSN, MaxConns: 8, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer restoredPool.Close()
	restoredManager, err := writer.New(restoredPool, "privacy-restored", writer.Config{LeaseDuration: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	restoredRed, err := reducer.New(restoredManager, restoredPool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restoredDB.ExecContext(ctx, `UPDATE ttg.writer_leases SET leased_until = clock_timestamp() - interval '1 second' WHERE village_id = $1`, villageID); err != nil {
		t.Fatal(err)
	}
	coordinator := &privacy.RestoreCoordinator{
		Ledger: ledger, Pool: restoredPool,
		NewManager: func(village string) (*writer.Manager, error) {
			return writer.New(restoredPool, "privacy-restore-coordinator", writer.Config{LeaseDuration: 5 * time.Second})
		},
		NewReducer: func(manager *writer.Manager) (*reducer.Reducer, error) {
			return reducer.New(manager, restoredPool)
		},
	}
	restoreTarget := randomRestoreUUID(t)
	restoreResult, err := coordinator.Reapply(ctx, restoreTarget)
	if err != nil {
		t.Fatalf("automatic ledger replay: %v", err)
	}
	if restoreResult.Applied == 0 || restoreResult.LastSeq == 0 {
		t.Fatalf("restore result %+v", restoreResult)
	}

	if err := restoredDB.QueryRowContext(ctx, `SELECT count(*) FROM ttg.players WHERE village_id = $1`, villageID).Scan(&playerCount); err != nil {
		t.Fatal(err)
	}
	if err := restoredDB.QueryRowContext(ctx, `SELECT count(*) FROM ttg.actor_identity_aliases WHERE village_id = $1`, villageID).Scan(&aliasCount); err != nil {
		t.Fatal(err)
	}
	if playerCount != 0 || aliasCount != 0 {
		t.Fatalf("resurrection after ledger reapply: players=%d aliases=%d", playerCount, aliasCount)
	}
	proof, err := ledger.Verify(restoredDSN)
	if err != nil {
		t.Fatal(err)
	}
	if err := privacy.AllowClientServing(proof, markerTime, []string{tombstoneID}); err != nil {
		t.Fatalf("ledger coverage: %v", err)
	}
	if _, err := restoredRed.VerifyReplay(ctx, sessionID, 0); err != nil {
		t.Fatalf("replay after reapply: %v", err)
	}
	if _, err := restoredDB.ExecContext(ctx, `UPDATE ttg.writer_leases SET leased_until = clock_timestamp() - interval '1 second' WHERE village_id = $1`, villageID); err != nil {
		t.Fatal(err)
	}
	secondCoordinator := &privacy.RestoreCoordinator{
		Ledger: ledger, Pool: restoredPool,
		NewManager: func(village string) (*writer.Manager, error) {
			return writer.New(restoredPool, "privacy-restore-idempotent", writer.Config{LeaseDuration: 5 * time.Second})
		},
		NewReducer: func(manager *writer.Manager) (*reducer.Reducer, error) { return reducer.New(manager, restoredPool) },
	}
	second, err := secondCoordinator.Reapply(ctx, restoreTarget)
	if err != nil {
		t.Fatalf("idempotent ledger replay: %v", err)
	}
	if second.Applied != 0 {
		t.Fatalf("second replay applied %d erasures", second.Applied)
	}
	if !restoreResult.Ready || !second.Ready {
		t.Fatal("restore did not produce readiness proof")
	}
	gate := pitr.PrivacyGate{LedgerAvailable: true, LedgerVerified: true}
	if err := gate.AllowClientServingWithLedgerProof(restoreResult.Proof, markerTime, restoreResult.AppliedErasures); err != nil {
		t.Fatalf("readiness gate: %v", err)
	}
	var storedTarget string
	if err := restoredDB.QueryRowContext(ctx, `SELECT target_identity FROM ttg.privacy_restore_runs LIMIT 1`).Scan(&storedTarget); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(storedTarget, "postgres://") || strings.Contains(storedTarget, "pitr_synthetic_password") {
		t.Fatal("DSN persisted as target identity")
	}
	_ = manifest
}
func randomRestoreUUID(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}
