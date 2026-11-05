package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func snapshotFixture(t *testing.T, body []byte, maxBytes int64) (string, *Replay) {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "fixture.ndjson"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	manifest := fmt.Sprintf(`{"schema":1,"fixtures":[{"name":"fixture.ndjson","schema_version":1,"sha256":%q,"records":1}]}`,
		hex.EncodeToString(sum[:]))
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	replay, err := NewReplay(ReplayConfig{
		Session:         replaySession(),
		Directory:       directory,
		Fixture:         "fixture.ndjson",
		MaxFixtureBytes: maxBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	return directory, replay
}

func TestReplayFixtureSizePolicy(t *testing.T) {
	body := []byte(replayJSON(1, "XÂY", "e1") + "\n")
	if _, err := NewReplay(ReplayConfig{
		Session:         replaySession(),
		Directory:       t.TempDir(),
		Fixture:         "fixture.ndjson",
		MaxFixtureBytes: 1,
	}); !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("missing fixture error = %v", err)
	}

	// Exactly at the limit is accepted and streams.
	_, replay := snapshotFixture(t, body, int64(len(body)))
	var events []Event
	if err := replay.Start(context.Background(), replaySession(), collectSink(&events)); err != nil {
		t.Fatalf("fixture at limit: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d", len(events))
	}

	// One byte below the fixture size fails closed before delivery.
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "fixture.ndjson"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	manifest := fmt.Sprintf(`{"schema":1,"fixtures":[{"name":"fixture.ndjson","schema_version":1,"sha256":%q,"records":1}]}`,
		hex.EncodeToString(sum[:]))
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewReplay(ReplayConfig{
		Session:         replaySession(),
		Directory:       directory,
		Fixture:         "fixture.ndjson",
		MaxFixtureBytes: int64(len(body) - 1),
	}); !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("oversized fixture error = %v", err)
	}
}

func TestReplayDetectsGrowthBeyondSizeLimit(t *testing.T) {
	body := []byte(replayJSON(1, "XÂY", "e1") + "\n")
	directory, replay := snapshotFixture(t, body, int64(len(body)))
	file, err := os.OpenFile(filepath.Join(directory, "fixture.ndjson"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("padding that pushes the fixture over the configured limit\n"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	var events []Event
	err = replay.Start(context.Background(), replaySession(), collectSink(&events))
	if !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("growth error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("delivered %d events from an oversized fixture", len(events))
	}
}

func TestReplaySnapshotCreationFailureIsRetryable(t *testing.T) {
	body := []byte(replayJSON(1, "XÂY", "e1") + "\n")
	_, replay := snapshotFixture(t, body, 0)
	replay.snapshotTemp = func() (*os.File, error) { return nil, errors.New("disk full") }
	var events []Event
	err := replay.Start(context.Background(), replaySession(), collectSink(&events))
	if !errors.Is(err, ErrSnapshotIO) {
		t.Fatalf("snapshot creation error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("delivered %d events without a snapshot", len(events))
	}
	if health := replay.Health(context.Background()); health.Status != StatusDisconnected {
		t.Fatalf("status = %s", health.Status)
	}

	// Restoring the temp facility makes a reconnect from the start succeed.
	replay.snapshotTemp = func() (*os.File, error) { return os.CreateTemp(t.TempDir(), "snap-*") }
	if err := replay.Reconnect(context.Background(), NewCursor(replaySession().ID, replay.SourceID(), 0), collectSink(&events)); err != nil {
		t.Fatalf("reconnect after snapshot failure: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("reconnect delivered %d events", len(events))
	}
}

func TestReplaySnapshotCancellationCleansUp(t *testing.T) {
	body := []byte(replayJSON(1, "XÂY", "e1") + "\n")
	_, replay := snapshotFixture(t, body, 0)
	directory := t.TempDir()
	replay.snapshotTemp = func() (*os.File, error) { return os.CreateTemp(directory, "snap-*") }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var events []Event
	err := replay.Start(ctx, replaySession(), collectSink(&events))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("delivered %d events after cancellation", len(events))
	}
	entries, readErr := os.ReadDir(directory)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("cancelled snapshot left %d temporary files", len(entries))
	}
}

func TestReplaySnapshotWriteFailureCleansUp(t *testing.T) {
	body := []byte(replayJSON(1, "XÂY", "e1") + "\n")
	_, replay := snapshotFixture(t, body, 0)
	directory := t.TempDir()
	readOnlyPath := filepath.Join(directory, "readonly")
	if err := os.WriteFile(readOnlyPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	readOnly, err := os.Open(readOnlyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer readOnly.Close()
	replay.snapshotTemp = func() (*os.File, error) { return readOnly, nil }
	var events []Event
	err = replay.Start(context.Background(), replaySession(), collectSink(&events))
	if !errors.Is(err, ErrSnapshotIO) {
		t.Fatalf("write failure error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("delivered %d events after write failure", len(events))
	}
}

func TestReplaySnapshotIsRemovedAfterSuccess(t *testing.T) {
	body := []byte(replayJSON(1, "XÂY", "e1") + "\n")
	_, replay := snapshotFixture(t, body, 0)
	directory := t.TempDir()
	replay.snapshotTemp = func() (*os.File, error) { return os.CreateTemp(directory, "snap-*") }
	var events []Event
	if err := replay.Start(context.Background(), replaySession(), collectSink(&events)); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("successful snapshot left %d temporary files", len(entries))
	}
}
