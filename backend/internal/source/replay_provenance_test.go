package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func replayJSON(seq int, message, eventID string) string {
	return fmt.Sprintf(`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":%d,"provider":"replay","provider_event_id":%q,"provider_viewer_id":"viewer-%04d","event_kind":"comment","message":%q}`,
		seq, eventID, seq, message)
}

func writeTempReplay(t *testing.T, lines []string) (string, *Replay) {
	t.Helper()
	directory := t.TempDir()
	body := []byte(strings.Join(lines, "\n") + "\n")
	if err := os.WriteFile(filepath.Join(directory, "fixture.ndjson"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	manifest := fmt.Sprintf(`{"schema":1,"fixtures":[{"name":"fixture.ndjson","schema_version":1,"sha256":%q,"records":%d}]}`,
		hex.EncodeToString(sum[:]), len(lines))
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	replay, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "fixture.ndjson"})
	if err != nil {
		t.Fatal(err)
	}
	return directory, replay
}

func TestReplayDetectsFixtureMutationBeforeStart(t *testing.T) {
	lines := []string{replayJSON(1, "XÂY", "e1"), replayJSON(2, "THỦ", "e2")}
	directory, replay := writeTempReplay(t, lines)
	if err := os.WriteFile(filepath.Join(directory, "fixture.ndjson"), []byte(strings.Join([]string{
		replayJSON(1, "ATTACK", "e1"), replayJSON(2, "THỦ", "e2"),
	}, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var events []Event
	err := replay.Start(context.Background(), replaySession(), collectSink(&events))
	if !errors.Is(err, ErrFixtureDigest) {
		t.Fatalf("mutated fixture error = %v, want ErrFixtureDigest", err)
	}
	if len(events) != 0 {
		t.Fatalf("emitted %d events from a mutated fixture", len(events))
	}
	if health := replay.Health(context.Background()); health.Status != StatusFailed || health.Delivered != 0 {
		t.Fatalf("health after mutation = %+v", health)
	}
}

func TestReplayDetectsFixtureReplacementBeforeReconnect(t *testing.T) {
	directory, replay := writeTempReplay(t, []string{replayJSON(1, "XÂY", "e1"), replayJSON(2, "THỦ", "e2")})
	if err := replay.Start(context.Background(), replaySession(), func(context.Context, Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "fixture.ndjson"), []byte(strings.Join([]string{
		replayJSON(1, "ATTACK", "e1"), replayJSON(2, "DEFEND", "e2"),
	}, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var events []Event
	err := replay.Reconnect(context.Background(), NewCursor(replaySession().ID, replay.SourceID(), 0), collectSink(&events))
	if !errors.Is(err, ErrFixtureDigest) {
		t.Fatalf("reconnect after replacement error = %v, want ErrFixtureDigest", err)
	}
	if len(events) != 0 {
		t.Fatalf("reconnect emitted %d events from a replaced fixture", len(events))
	}
}

func TestReplayDeliversVerifiedSnapshotDespiteMutationDuringDelivery(t *testing.T) {
	directory, replay := writeTempReplay(t, []string{
		replayJSON(1, "XÂY", "e1"),
		replayJSON(2, "THỦ", "e2"),
		replayJSON(3, "ĐÁNH", "e3"),
	})
	var messages []string
	err := replay.Start(context.Background(), replaySession(), func(_ context.Context, event Event) error {
		messages = append(messages, event.Message)
		if len(messages) == 1 {
			// Mutate the original path while delivery is in progress. The adapter must
			// already be streaming its private verified snapshot.
			replacement := strings.Join([]string{
				replayJSON(1, "ATTACK", "e1"),
				replayJSON(2, "DEFEND", "e2"),
				replayJSON(3, "BUILD", "e3"),
			}, "\n") + "\n"
			if writeErr := os.WriteFile(filepath.Join(directory, "fixture.ndjson"), []byte(replacement), 0o600); writeErr != nil {
				return writeErr
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("delivery with concurrent mutation: %v", err)
	}
	want := []string{"XÂY", "THỦ", "ĐÁNH"}
	for index := range want {
		if messages[index] != want[index] {
			t.Fatalf("delivered messages = %v, want verified snapshot %v", messages, want)
		}
	}
	// A later reconnect re-verifies and must fail closed.
	err = replay.Reconnect(context.Background(), NewCursor(replaySession().ID, replay.SourceID(), 0), func(context.Context, Event) error { return nil })
	if !errors.Is(err, ErrFixtureDigest) {
		t.Fatalf("reconnect after in-place mutation error = %v, want ErrFixtureDigest", err)
	}
}

func TestReplayRejectsSymlinkEscape(t *testing.T) {
	outside := t.TempDir()
	outsidePath := filepath.Join(outside, "outside.ndjson")
	body := []byte(replayJSON(1, "XÂY", "e1") + "\n")
	if err := os.WriteFile(outsidePath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	link := filepath.Join(directory, "link.ndjson")
	if err := os.Symlink(outsidePath, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	sum := sha256.Sum256(body)
	manifest := fmt.Sprintf(`{"schema":1,"fixtures":[{"name":"link.ndjson","schema_version":1,"sha256":%q,"records":1}]}`,
		hex.EncodeToString(sum[:]))
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "link.ndjson"}); !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("symlink escape error = %v, want ErrInvalidFixture", err)
	}
}
