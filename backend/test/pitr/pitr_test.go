package pitr_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/db/migrations"
	"github.com/hung98dev/ttgqldl/backend/internal/pitr"
	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const image = "postgres:18.6@sha256:3a4264520e121176619f54ff867d12774bc81e95e7b84db58d63bccdb6144b81"

func requireDocker(t *testing.T) {
	t.Helper()
	if os.Getenv("TTG_PITR_TEST") != "1" {
		t.Skip("set TTG_PITR_TEST=1 to run real Docker PITR rehearsal")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatal("docker CLI is required")
	}
}

func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func docker(t *testing.T, args ...string) string { return run(t, "docker", args...) }

func unique(prefix string) string {
	return fmt.Sprintf("ttg-pitr-%s-%d", prefix, time.Now().UnixNano())
}

func hostPort(t *testing.T, container string) string {
	t.Helper()
	output := docker(t, "port", container, "5432/tcp")
	parts := strings.Split(output, ":")
	if len(parts) < 2 {
		t.Fatalf("port output %q", output)
	}
	return parts[len(parts)-1]
}

func waitReady(t *testing.T, container, user, db string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, "docker", "exec", container, "pg_isready", "-U", user, "-d", db)
		_ = cmd.Run()
		cancel()
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 0 {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("container %s did not become ready", container)
}

func waitQuery(t *testing.T, dsn, query string) string {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		db, err := sql.Open("pgx", dsn)
		if err == nil {
			var value string
			err = db.QueryRowContext(context.Background(), query).Scan(&value)
			db.Close()
			if err == nil {
				return value
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("query did not become available: %s", query)
	return ""
}

func TestRealEncryptedPhysicalBackupAndPITR(t *testing.T) {
	requireDocker(t)
	primary := unique("primary")
	pgdata := unique("pgdata")
	wal := unique("wal")
	backup := unique("backup")
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
	archiveCommand := "set -e; tmp=/walarchive/.%f.tmp.$$; cp %p $tmp; sync $tmp; test ! -f /walarchive/%f; mv $tmp /walarchive/%f; sync"
	docker(t, "run", "-d", "--name", primary,
		"-e", "POSTGRES_PASSWORD=pitr_synthetic_password", "-e", "POSTGRES_DB=pitr",
		"-v", pgdata+":/var/lib/postgresql", "-v", wal+":/walarchive", "-v", backup+":/backup",
		"-p", "127.0.0.1::5432", image, "postgres",
		"-c", "wal_level=replica", "-c", "archive_mode=on",
		"-c", "archive_command="+archiveCommand, "-c", "archive_timeout=60")
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
		t.Fatalf("migrations: %v", err)
	}

	ctx := context.Background()
	pool, err := storage.Open(ctx, storage.Options{DSN: dsn, MaxConns: 8, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var villageID, sessionID string
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
	payload := `{"kind":"role","role":"builder"}`
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.input_events (session_id, input_seq, accepted_tick, scheduled_tick, wave_no, accepted_wave, actor_key, kind, canonical_payload, rules_version) VALUES ($1, 1, -1, 0, 1, 1, $2, 'role', $3::jsonb, 'ttg-rules-v1')`, sessionID, strings.Repeat("a", 64), payload); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE ttg.sessions SET input_sequence = 1 WHERE id = $1`, sessionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	manager, err := writer.New(pool, "pitr-primary", writer.Config{LeaseDuration: 30 * time.Second})
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
	var tickBefore int32
	var hashBefore string
	if err := db.QueryRowContext(ctx, `SELECT reducer_revision, wave_tick, last_state_hash FROM ttg.sessions WHERE id = $1`, sessionID).Scan(&revisionBefore, &tickBefore, &hashBefore); err != nil {
		t.Fatal(err)
	}
	// Real physical base backup into an independent volume.
	var startWAL, endWAL, systemID string
	_ = db.QueryRowContext(ctx, `SELECT pg_current_wal_lsn()::text`).Scan(&startWAL)
	_ = db.QueryRowContext(ctx, `SELECT system_identifier::text FROM pg_control_system()`).Scan(&systemID)
	docker(t, "exec", "-u", "postgres", primary, "pg_basebackup", "-D", "/backup/base", "-Fp", "-X", "stream", "-U", "postgres", "-h", "127.0.0.1")
	docker(t, "exec", "-u", "postgres", primary, "tar", "-C", "/backup", "-cf", "/backup/base.tar", "base")
	_ = db.QueryRowContext(ctx, `SELECT pg_current_wal_lsn()::text`).Scan(&endWAL)
	hostTar := filepath.Join(dir, "base.tar")
	docker(t, "cp", primary+":/backup/base.tar", hostTar)
	plainInfo, err := os.Stat(hostTar)
	if err != nil {
		t.Fatal(err)
	}
	var markerTime time.Time
	if err := db.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&markerTime); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	for index := range key {
		key[index] = byte(index + 1)
	}
	encrypted := filepath.Join(dir, "base.tar.enc")
	manifest, err := pitr.EncryptFile(hostTar, encrypted, key)
	if err != nil {
		t.Fatal(err)
	}
	manifest.BackupID = unique("backup")
	manifest.PostgresVersion = "18.6"
	manifest.SystemIdentifier = systemID
	manifest.StartWAL = startWAL
	manifest.EndWAL = endWAL
	manifest.Timeline = 1
	encInfo, err := os.Stat(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	manifest.BackupBytes = encInfo.Size()
	t.Logf("base_backup_plaintext_bytes=%d encrypted_bytes=%d", plainInfo.Size(), encInfo.Size())
	manifest.Verified = true
	manifest.RestoreEligible = true
	manifest.StartTime = time.Now().UTC()
	manifest.EndTime = time.Now().UTC()
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := pitr.WriteManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	if err := pitr.VerifyCiphertext(encrypted, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := pitr.DecryptFile(encrypted, filepath.Join(dir, "wrong.tar"), key); err != nil {
		t.Fatal(err)
	}
	wrong := make([]byte, 32)
	if _, err := pitr.DecryptFile(encrypted, filepath.Join(dir, "wrong-key.tar"), wrong); !errors.Is(err, pitr.ErrDecrypt) {
		t.Fatalf("wrong key = %v", err)
	}

	// Commit the next revision after the base backup and archive its WAL.
	if _, err := red.Step(ctx, lease, sessionID); err != nil {
		t.Fatal(err)
	}
	var revisionAfter int64
	var hashAfter string
	if err := db.QueryRowContext(ctx, `SELECT reducer_revision, last_state_hash FROM ttg.sessions WHERE id = $1`, sessionID).Scan(&revisionAfter, &hashAfter); err != nil {
		t.Fatal(err)
	}
	var afterTime time.Time
	if err := db.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&afterTime); err != nil {
		t.Fatal(err)
	}
	var afterLSN string
	if err := db.QueryRowContext(ctx, `SELECT pg_current_wal_lsn()::text`).Scan(&afterLSN); err != nil {
		t.Fatal(err)
	}
	var targetWAL string
	if err := db.QueryRowContext(ctx, `SELECT pg_walfile_name($1::pg_lsn)`, afterLSN).Scan(&targetWAL); err != nil {
		t.Fatal(err)
	}
	// A harmless later transaction creates a WAL record after the requested target,
	// allowing recovery_target_time to stop cleanly between R+1 and this tail.
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS ttg.pitr_tail_marker (id INT); INSERT INTO ttg.pitr_tail_marker VALUES (1); CHECKPOINT`); err != nil {
		t.Fatal(err)
	}
	var switchWAL string
	if err := db.QueryRowContext(ctx, `SELECT pg_walfile_name(pg_switch_wal())`).Scan(&switchWAL); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		check := exec.Command("docker", "exec", primary, "test", "-f", "/walarchive/"+switchWAL)
		var ready int
		_ = db.QueryRowContext(ctx, `SELECT count(*) FROM pg_ls_archive_statusdir() WHERE name LIKE '%.ready'`).Scan(&ready)
		if err := check.Run(); err == nil && ready == 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)

	var walBytes string
	_ = db.QueryRowContext(ctx, `SELECT pg_size_pretty(sum(size)) FROM pg_ls_waldir()`).Scan(&walBytes)
	t.Logf("wal_archive_bytes=%s", walBytes)
	// Primary data-directory loss: the independent backup and WAL volumes remain.
	_ = exec.Command("docker", "rm", "-f", primary).Run()
	_ = exec.Command("docker", "volume", "rm", "-f", pgdata).Run()

	restore := func(name, targetKind, target string) (string, string) {
		restoreStarted := time.Now()
		volume := unique("restore")
		restoreNames = append(restoreNames, name)
		restoreVolumes = append(restoreVolumes, volume)
		hostDir := t.TempDir()
		if _, err := pitr.DecryptFile(encrypted, filepath.Join(hostDir, "base.tar"), key); err != nil {
			t.Fatal(err)
		}
		docker(t, "volume", "create", volume)
		docker(t, "run", "--rm", "-u", "root", "-v", volume+":/var/lib/postgresql", image, "chown", "-R", "postgres:postgres", "/var/lib/postgresql")
		script := `set -e
mkdir -p /var/lib/postgresql/18/docker
tar -C /var/lib/postgresql/18/docker --strip-components=1 -xf /restore/base.tar
chmod 700 /var/lib/postgresql/18/docker
cat >> /var/lib/postgresql/18/docker/postgresql.auto.conf <<EOF
restore_command = 'cp /walarchive/%f %p'
recovery_target_` + targetKind + ` = '` + target + `'
recovery_target_action = 'promote'
EOF
touch /var/lib/postgresql/18/docker/recovery.signal
exec postgres -D /var/lib/postgresql/18/docker`
		docker(t, "run", "-d", "--name", name, "-e", "POSTGRES_PASSWORD=pitr_synthetic_password", "-v", volume+":/var/lib/postgresql", "-v", hostDir+":/restore:ro", "-v", wal+":/walarchive", "-p", "127.0.0.1::5432", "--user", "postgres", image, "bash", "-c", script)
		waitReady(t, name, "postgres", "pitr")
		port := hostPort(t, name)
		restoredDSN := fmt.Sprintf("postgres://postgres:pitr_synthetic_password@127.0.0.1:%s/pitr?sslmode=disable", port)
		_ = waitQuery(t, restoredDSN, `SELECT pg_is_in_recovery()::text`)
		var revision int64
		var hash string
		restored, err := storage.Open(ctx, storage.Options{DSN: restoredDSN, MaxConns: 2, ConnectTimeout: 5 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		restoredSQL, err := sql.Open("pgx", restoredDSN)
		if err != nil {
			t.Fatal(err)
		}
		if err := restoredSQL.QueryRowContext(ctx, `SELECT reducer_revision, last_state_hash FROM ttg.sessions WHERE id = $1`, sessionID).Scan(&revision, &hash); err != nil {
			t.Fatal(err)
		}
		restoredSQL.Close()
		fake := noFence{}
		replay, err := reducer.New(fake, restored)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := replay.VerifyReplay(ctx, sessionID, 0); err != nil {
			t.Fatalf("replay after restore: %v", err)
		}
		restored.Close()
		t.Logf("restore target_kind=%s duration=%s revision=%d", targetKind, time.Since(restoreStarted).Round(time.Millisecond), revision)
		return fmt.Sprintf("%d", revision), hash
	}

	beforeRevision, beforeHash := restore(unique("restore-before"), "time", markerTime.UTC().Format("2006-01-02 15:04:05.999999+00"))
	if beforeRevision != fmt.Sprintf("%d", revisionBefore) || beforeHash != hashBefore {
		t.Fatalf("before restore = %s/%s, want %d/%s", beforeRevision, beforeHash, revisionBefore, hashBefore)
	}
	afterRevision, afterHash := restore(unique("restore-after"), "lsn", afterLSN)
	if afterRevision != fmt.Sprintf("%d", revisionAfter) || afterHash != hashAfter {
		t.Fatalf("after restore = %s/%s, want %d/%s", afterRevision, afterHash, revisionAfter, hashAfter)
	}
	if tickBefore != 0 {
		t.Fatalf("unexpected tick %d", tickBefore)
	}

	// Missing-WAL negative: restore must not become ready when the segment containing
	// the requested LSN is absent from the isolated archive copy.
	missingWAL := unique("wal-missing")
	restoreVolumes = append(restoreVolumes, missingWAL)
	docker(t, "volume", "create", missingWAL)
	docker(t, "run", "--rm", "-v", wal+":/src", "-v", missingWAL+":/dst", image, "bash", "-c", "cp -a /src/. /dst/ && rm -f /dst/"+targetWAL)
	missingName := unique("restore-missing")
	restoreNames = append(restoreNames, missingName)
	missingVolume := unique("restore-missing-data")
	restoreVolumes = append(restoreVolumes, missingVolume)
	missingHost := t.TempDir()
	if _, err := pitr.DecryptFile(encrypted, filepath.Join(missingHost, "base.tar"), key); err != nil {
		t.Fatal(err)
	}
	docker(t, "volume", "create", missingVolume)
	docker(t, "run", "--rm", "-u", "root", "-v", missingVolume+":/var/lib/postgresql", image, "chown", "-R", "postgres:postgres", "/var/lib/postgresql")
	missingScript := `set -e
mkdir -p /var/lib/postgresql/18/docker
tar -C /var/lib/postgresql/18/docker --strip-components=1 -xf /restore/base.tar
chmod 700 /var/lib/postgresql/18/docker
cat >> /var/lib/postgresql/18/docker/postgresql.auto.conf <<EOF
restore_command = 'cp /walarchive/%f %p'
recovery_target_lsn = '` + afterLSN + `'
recovery_target_action = 'promote'
EOF
touch /var/lib/postgresql/18/docker/recovery.signal
exec postgres -D /var/lib/postgresql/18/docker`
	docker(t, "run", "-d", "--name", missingName, "-e", "POSTGRES_PASSWORD=pitr_synthetic_password", "-v", missingVolume+":/var/lib/postgresql", "-v", missingHost+":/restore:ro", "-v", missingWAL+":/walarchive", "-p", "127.0.0.1::5432", "--user", "postgres", image, "bash", "-c", missingScript)
	missingFailed := false
	missingDeadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(missingDeadline) {
		inspect := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", missingName)
		output, _ := inspect.Output()
		if strings.TrimSpace(string(output)) == "false" {
			missingFailed = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !missingFailed {
		t.Fatal("missing-WAL restore became ready or hung")
	}
}

type noFence struct{}

func (noFence) WithFence(context.Context, writer.Lease, string, func(context.Context, writer.FencedTx) error) error {
	return errors.New("replay-only fence")
}
