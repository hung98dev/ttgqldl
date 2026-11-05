package source

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func replaySession() Session {
	return Session{ID: "synthetic-session-001", Mode: ModeReplay}
}

func newTestReplay(t *testing.T, fixture string) *Replay {
	t.Helper()
	replay, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: fixture})
	if err != nil {
		t.Fatalf("NewReplay(%s): %v", fixture, err)
	}
	return replay
}

func TestReplayManifestAndDigest(t *testing.T) {
	replay := newTestReplay(t, "replay_valid.ndjson")
	if replay.Records() != 6 {
		t.Fatalf("records = %d", replay.Records())
	}
	if len(replay.SourceID()) != 64 {
		t.Fatalf("source id = %q", replay.SourceID())
	}
	if _, err := hex.DecodeString(replay.SourceID()); err != nil {
		t.Fatalf("source id is not hex: %v", err)
	}

	if _, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: "missing.ndjson"}); !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("missing fixture error = %v", err)
	}
	if _, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: "../manifest.json"}); !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("path traversal error = %v", err)
	}
	if _, err := NewReplay(ReplayConfig{Session: testSession(ModeMock), Directory: "testdata", Fixture: "replay_valid.ndjson"}); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("wrong adapter mode error = %v", err)
	}
}

// tempFixture writes a fixture and manifest; tamper lets a test corrupt the pair.
func tempFixture(t *testing.T, body []byte, tamper func(manifest map[string]any, entry map[string]any)) *Replay {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "fixture.ndjson"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	entry := map[string]any{
		"name":           "fixture.ndjson",
		"schema_version": 1,
		"sha256":         hex.EncodeToString(sum[:]),
		"records":        bytes.Count(body, []byte("\n")),
	}
	manifest := map[string]any{"schema": 1, "fixtures": []any{entry}}
	if tamper != nil {
		tamper(manifest, entry)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	replay, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "fixture.ndjson"})
	if err != nil {
		return &Replay{open: func() (io.ReadCloser, error) { return nil, err }, sourceID: "invalid"}
	}
	return replay
}

func TestReplayRejectsTamperedFixtureAndManifest(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "replay_valid.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "fixture.ndjson"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	manifest := fmt.Sprintf(`{"schema":1,"fixtures":[{"name":"fixture.ndjson","schema_version":1,"sha256":"%s","records":6}]}`, hex.EncodeToString(sum[:]))
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "fixture.ndjson"}); err != nil {
		t.Fatalf("control fixture rejected: %v", err)
	}

	// Digest mismatch.
	badDigest := strings.Replace(manifest, hex.EncodeToString(sum[:]), strings.Repeat("0", 64), 1)
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(badDigest), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "fixture.ndjson"}); !errors.Is(err, ErrFixtureDigest) {
		t.Fatalf("digest mismatch error = %v", err)
	}

	// Record-count mismatch with correct digest.
	badCount := strings.Replace(manifest, `"records":6`, `"records":7`, 1)
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(badCount), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "fixture.ndjson"}); !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("record count mismatch error = %v", err)
	}

	// Unsupported manifest schema.
	badSchema := strings.Replace(manifest, `"schema":1`, `"schema":2`, 1)
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(badSchema), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "fixture.ndjson"}); !errors.Is(err, ErrSchemaUnsupported) {
		t.Fatalf("manifest schema error = %v", err)
	}

	// Unknown manifest field.
	unknownField := strings.Replace(manifest, `"fixtures"`, `"extra":true,"fixtures"`, 1)
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(unknownField), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "fixture.ndjson"}); !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("unknown manifest field error = %v", err)
	}
}

func TestReplayStartDeliversFixtureInFileOrder(t *testing.T) {
	replay := newTestReplay(t, "replay_valid.ndjson")
	var events []Event
	if err := replay.Start(context.Background(), replaySession(), collectSink(&events)); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(events) != 6 {
		t.Fatalf("delivered %d events", len(events))
	}
	if events[4].Kind != KindGift || events[5].Kind != KindJoin {
		t.Fatalf("event kinds = %s/%s", events[4].Kind, events[5].Kind)
	}
	for index, event := range events {
		if event.Provider != "replay" || event.SourceSession != replaySession().ID {
			t.Fatalf("event %d binding = %q/%q", index, event.Provider, event.SourceSession)
		}
	}
	health := replay.Health(context.Background())
	if health.Status != StatusEOF || health.Delivered != 6 {
		t.Fatalf("health = %+v", health)
	}
	if health.Cursor != NewCursor(replaySession().ID, replay.SourceID(), 6) {
		t.Fatalf("cursor = %+v", health.Cursor)
	}
	capabilities := replay.Capabilities(context.Background())
	if !capabilities.ReplayCursor || !capabilities.DeliveryOrder || capabilities.TimestampReliability != TimestampRecorded {
		t.Fatalf("capabilities = %+v", capabilities)
	}
}

func TestReplayUnicodeAndDuplicateFixtures(t *testing.T) {
	unicode := newTestReplay(t, "replay_unicode.ndjson")
	var unicodeEvents []Event
	if err := unicode.Start(context.Background(), replaySession(), collectSink(&unicodeEvents)); err != nil {
		t.Fatal(err)
	}
	if len(unicodeEvents) != 6 || unicodeEvents[0].Message != "XÂY" || unicodeEvents[4].Message != "ＡＴＴＡＣＫ" {
		t.Fatalf("unicode events = %+v", unicodeEvents)
	}

	duplicates := newTestReplay(t, "replay_duplicates.ndjson")
	var duplicateEvents []Event
	if err := duplicates.Start(context.Background(), replaySession(), collectSink(&duplicateEvents)); err != nil {
		t.Fatal(err)
	}
	if len(duplicateEvents) != 4 {
		t.Fatalf("duplicate fixture delivered %d events", len(duplicateEvents))
	}
	if duplicateEvents[0].ProviderEventID != duplicateEvents[1].ProviderEventID {
		t.Fatal("duplicate provider event ids were not preserved for later deduplication")
	}
	if duplicateEvents[2].ProviderEventID != duplicateEvents[3].ProviderEventID {
		t.Fatal("second duplicate pair was not preserved")
	}
}

func TestReplayDoesNotSortByUpstreamTimestamp(t *testing.T) {
	replay := newTestReplay(t, "replay_out_of_order.ndjson")
	var events []Event
	if err := replay.Start(context.Background(), replaySession(), collectSink(&events)); err != nil {
		t.Fatal(err)
	}
	messages := []string{events[0].Message, events[1].Message, events[2].Message}
	want := []string{"XÂY", "THỦ", "ĐÁNH"}
	for index := range want {
		if messages[index] != want[index] {
			t.Fatalf("delivery order = %v, want file order %v", messages, want)
		}
	}
	if events[0].UpstreamAt == nil || events[1].UpstreamAt == nil {
		t.Fatal("fixture upstream timestamps missing")
	}
	if !events[0].UpstreamAt.After(*events[2].UpstreamAt) {
		t.Fatal("fixture is not out of order; test is vacuous")
	}
}

func TestReplayMalformedFixturesFailClosed(t *testing.T) {
	cases := []struct {
		fixture      string
		want         error
		line         int
		record       int
		deliveredMin int
	}{
		{"replay_malformed.ndjson", ErrMalformedRecord, 2, 2, 1},
		{"replay_bad_session.ndjson", ErrSessionMismatch, 1, 1, 0},
		{"replay_oversize_message.ndjson", ErrInvalidMessage, 1, 1, 0},
		{"replay_control.ndjson", ErrInvalidMessage, 1, 1, 0},
		{"replay_unknown_field.ndjson", ErrMalformedRecord, 1, 1, 0},
		{"replay_bad_utf8.ndjson", ErrRecordEncoding, 1, 1, 0},
		{"replay_oversize_record.ndjson", ErrRecordTooLong, 1, 1, 0},
		{"replay_truncated.ndjson", ErrMalformedRecord, 1, 1, 0},
	}
	for _, testCase := range cases {
		t.Run(testCase.fixture, func(t *testing.T) {
			replay := newTestReplay(t, testCase.fixture)
			delivered := 0
			err := replay.Start(context.Background(), replaySession(), func(context.Context, Event) error {
				delivered++
				return nil
			})
			if !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
			var recordErr *RecordError
			if !errors.As(err, &recordErr) {
				t.Fatalf("error is not a RecordError: %v", err)
			}
			if recordErr.Line != testCase.line || recordErr.Record != testCase.record {
				t.Fatalf("location = line %d record %d, want line %d record %d", recordErr.Line, recordErr.Record, testCase.line, testCase.record)
			}
			if delivered != testCase.deliveredMin {
				t.Fatalf("delivered %d events before failure, want %d", delivered, testCase.deliveredMin)
			}
			if health := replay.Health(context.Background()); health.Status != StatusFailed {
				t.Fatalf("health status = %s", health.Status)
			}
		})
	}
}

func TestReplaySessionBindingAndSchemaRejection(t *testing.T) {
	replay := newTestReplay(t, "replay_valid.ndjson")
	other := Session{ID: "synthetic-session-002", Mode: ModeReplay}
	if err := replay.Start(context.Background(), other, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("start session mismatch error = %v", err)
	}

	unsupported := tempFixture(t, []byte(`{"schema_version":2,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","event_kind":"comment","message":"XÂY"}`+"\n"), nil)
	if unsupported.open == nil || unsupported.sourceID == "invalid" {
		t.Fatal("fixture setup failed")
	}
	var events []Event
	err := unsupported.Start(context.Background(), replaySession(), collectSink(&events))
	if !errors.Is(err, ErrSchemaUnsupported) {
		t.Fatalf("record schema error = %v", err)
	}
}

func TestReplayCursorResumeAndForeignCursors(t *testing.T) {
	replay := newTestReplay(t, "replay_valid.ndjson")
	all := make([]Event, 0, 6)
	if err := replay.Start(context.Background(), replaySession(), collectSink(&all)); err != nil {
		t.Fatal(err)
	}
	var resumed []Event
	if err := replay.Reconnect(context.Background(), NewCursor(replaySession().ID, replay.SourceID(), 2), collectSink(&resumed)); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if len(resumed) != 4 || resumed[0].ProviderEventID != all[2].ProviderEventID {
		t.Fatalf("resumed %d events", len(resumed))
	}
	if health := replay.Health(context.Background()); health.Delivered != 10 || health.Status != StatusEOF {
		t.Fatalf("health after reconnect = %+v", health)
	}

	var atEOF []Event
	if err := replay.Reconnect(context.Background(), NewCursor(replaySession().ID, replay.SourceID(), 6), collectSink(&atEOF)); err != nil {
		t.Fatalf("reconnect at EOF: %v", err)
	}
	if len(atEOF) != 0 {
		t.Fatalf("reconnect at EOF delivered %d", len(atEOF))
	}

	foreignFixture := newTestReplay(t, "replay_unicode.ndjson")
	foreign := NewCursor(replaySession().ID, foreignFixture.SourceID(), 0)
	if err := replay.Reconnect(context.Background(), foreign, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrCursorMismatch) {
		t.Fatalf("foreign fixture cursor error = %v", err)
	}
	foreignSession := NewCursor("synthetic-session-002", replay.SourceID(), 0)
	if err := replay.Reconnect(context.Background(), foreignSession, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrCursorMismatch) {
		t.Fatalf("foreign session cursor error = %v", err)
	}
	badVersion := NewCursor(replaySession().ID, replay.SourceID(), 0)
	badVersion.Version = 2
	if err := replay.Reconnect(context.Background(), badVersion, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("bad version error = %v", err)
	}
	if err := replay.Reconnect(context.Background(), NewCursor(replaySession().ID, replay.SourceID(), 7), func(context.Context, Event) error { return nil }); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("out-of-range cursor error = %v", err)
	}

	fresh := newTestReplay(t, "replay_valid.ndjson")
	if err := fresh.Reconnect(context.Background(), NewCursor(replaySession().ID, fresh.SourceID(), 0), func(context.Context, Event) error { return nil }); !errors.Is(err, ErrNotReconnectable) {
		t.Fatalf("reconnect before start error = %v", err)
	}
}

func TestReplayCallbackFailureRetry(t *testing.T) {
	replay := newTestReplay(t, "replay_valid.ndjson")
	delivered := 0
	err := replay.Start(context.Background(), replaySession(), func(context.Context, Event) error {
		delivered++
		if delivered == 3 {
			return errors.New("reject")
		}
		return nil
	})
	if !errors.Is(err, ErrDeliveryRejected) {
		t.Fatalf("delivery error = %v", err)
	}
	health := replay.Health(context.Background())
	if health.Status != StatusDisconnected || health.Delivered != 2 || health.Cursor.NextEvent != 2 {
		t.Fatalf("health = %+v", health)
	}
	var resumed []Event
	if err := replay.Reconnect(context.Background(), health.Cursor, collectSink(&resumed)); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(resumed) != 4 {
		t.Fatalf("retry delivered %d events", len(resumed))
	}
}

func TestReplayReaderFailuresAndCancellation(t *testing.T) {
	openFailure := newTestReplay(t, "replay_valid.ndjson")
	openFailure.open = func() (io.ReadCloser, error) { return nil, errors.New("disk") }
	var events []Event
	if err := openFailure.Start(context.Background(), replaySession(), collectSink(&events)); !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("open failure error = %v", err)
	}
	if health := openFailure.Health(context.Background()); health.Status != StatusFailed {
		t.Fatalf("status = %s", health.Status)
	}

	midFailure := newTestReplay(t, "replay_valid.ndjson")
	midFailure.open = func() (io.ReadCloser, error) {
		first := `{"schema_version":1,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","provider_event_id":"e1","event_kind":"comment","message":"XÂY"}` + "\n"
		return io.NopCloser(io.MultiReader(strings.NewReader(first), iotest.ErrReader(errors.New("read")))), nil
	}
	var partial []Event
	if err := midFailure.Start(context.Background(), replaySession(), collectSink(&partial)); !errors.Is(err, ErrInvalidFixture) {
		t.Fatalf("mid-read failure error = %v", err)
	}
	if len(partial) != 0 {
		t.Fatalf("delivered %d events before snapshot verification completed", len(partial))
	}

	cancelled := newTestReplay(t, "replay_disconnect.ndjson")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	err := cancelled.Start(ctx, replaySession(), func(sinkCtx context.Context, _ Event) error {
		calls++
		if calls == 2 {
			cancel()
			<-sinkCtx.Done()
			return sinkCtx.Err()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
	if health := cancelled.Health(context.Background()); health.Status != StatusDisconnected || health.Delivered != 1 {
		t.Fatalf("health after cancellation = %+v", health)
	}
}

func TestReplayLargeFixtureStreams(t *testing.T) {
	if testing.Short() {
		t.Skip("large replay fixture")
	}
	const recordCount = 50000
	directory := t.TempDir()
	path := filepath.Join(directory, "large.ndjson")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	writer := io.MultiWriter(file, hash)
	for index := 1; index <= recordCount; index++ {
		line := fmt.Sprintf(`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":%d,"provider":"replay","provider_event_id":"e%d","provider_viewer_id":"v%d","event_kind":"comment","message":"XÂY"}`+"\n", index, index, index)
		if _, err := io.WriteString(writer, line); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`{"schema":1,"fixtures":[{"name":"large.ndjson","schema_version":1,"sha256":"%s","records":%d}]}`,
		hex.EncodeToString(hash.Sum(nil)), recordCount)
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	replay, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "large.ndjson"})
	if err != nil {
		t.Fatal(err)
	}
	delivered := 0
	started := time.Now()
	if err := replay.Start(context.Background(), replaySession(), func(context.Context, Event) error {
		delivered++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if delivered != recordCount {
		t.Fatalf("delivered %d of %d", delivered, recordCount)
	}
	t.Logf("streamed %d records in %s", delivered, time.Since(started).Round(time.Millisecond))

	if err := replay.Reconnect(context.Background(), NewCursor(replaySession().ID, replay.SourceID(), recordCount-1), func(context.Context, Event) error {
		delivered++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if delivered != recordCount+1 {
		t.Fatalf("resume delivered %d events", delivered-recordCount)
	}
}

func TestReplayLifecycleStartOnceAndStopBeforeStart(t *testing.T) {
	replay := newTestReplay(t, "replay_valid.ndjson")
	if err := replay.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := replay.Start(context.Background(), replaySession(), func(context.Context, Event) error { return nil }); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("start after stop-before-start error = %v", err)
	}
	fresh := newTestReplay(t, "replay_valid.ndjson")
	if err := fresh.Start(context.Background(), replaySession(), func(context.Context, Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := fresh.Start(context.Background(), replaySession(), func(context.Context, Event) error { return nil }); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("repeated start error = %v", err)
	}
	if err := fresh.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestReplayRejectsManifestPathTraversal(t *testing.T) {
	body := []byte(`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","event_kind":"comment","message":"XÂY"}` + "\n")
	digest := sha256.Sum256(body)
	for _, name := range []string{"../escape.ndjson", "sub/escape.ndjson", "/tmp/escape.ndjson"} {
		directory := t.TempDir()
		manifest := fmt.Sprintf(`{"schema":1,"fixtures":[{"name":"fixture.ndjson","schema_version":1,"sha256":%q,"records":1},{"name":%q,"schema_version":1,"sha256":%q,"records":1}]}`,
			hex.EncodeToString(digest[:]), name, strings.Repeat("0", 64))
		if err := os.WriteFile(filepath.Join(directory, "fixture.ndjson"), body, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: directory, Fixture: "fixture.ndjson"}); !errors.Is(err, ErrInvalidFixture) {
			t.Fatalf("manifest name %q error = %v, want ErrInvalidFixture", name, err)
		}
	}
}
