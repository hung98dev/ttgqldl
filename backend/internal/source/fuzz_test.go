package source

import (
	"reflect"
	"testing"
)

func FuzzParseCursor(f *testing.F) {
	f.Add("")
	f.Add("ttgc1.c3ludGhldGljLXNlc3Npb24tMDAx.ZGlnZXN0.0")
	f.Add("ttgc2.c2Vzc2lvbg.ZGlnaWVzdA.4")
	f.Add("not-a-cursor")
	f.Add("2026-01-01T00:00:00Z")
	f.Fuzz(func(t *testing.T, raw string) {
		cursor, err := ParseCursor(raw)
		if err != nil {
			return
		}
		if !cursor.Valid() {
			t.Fatalf("parsed cursor is not valid: %+v", cursor)
		}
		if cursor.Version != CursorVersion {
			t.Fatalf("parsed unknown version %d", cursor.Version)
		}
		again, err := ParseCursor(cursor.Encode())
		if err != nil {
			t.Fatalf("re-encode rejected: %v", err)
		}
		if !reflect.DeepEqual(cursor, again) {
			t.Fatalf("round trip mismatch: %+v vs %+v", cursor, again)
		}
	})
}

func FuzzDecodeReplayRecord(f *testing.F) {
	f.Add([]byte(`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","provider_event_id":"e1","event_kind":"comment","message":"XÂY"}`))
	f.Add([]byte(`{"schema_version":2,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","event_kind":"comment"}`))
	f.Add([]byte(`{"schema_version":1,"source_session":"other","event_seq":1,"provider":"replay","event_kind":"comment"}`))
	f.Add([]byte(`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":2,"provider":"replay","event_kind":"comment"}`))
	f.Add([]byte(`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","event_kind":"comment","surprise":true}`))
	f.Add(append([]byte(`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","event_kind":"comment","message":"XAY`), 0xff, '"', '}'))
	f.Add([]byte(``))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, line []byte) {
		if len(line) > maxRecordBytes*2 {
			return // keep the fuzz target bounded; the scanner enforces the real limit
		}
		event, err := decodeReplayRecord(line, replaySession(), 1)
		if err != nil {
			return
		}
		if err := ValidateEvent(event, replaySession(), ModeReplay); err != nil {
			t.Fatalf("decoded event does not validate: %v", err)
		}
		again, err := decodeReplayRecord(line, replaySession(), 1)
		if err != nil {
			t.Fatalf("deterministic decode failed: %v", err)
		}
		if !reflect.DeepEqual(event, again) {
			t.Fatalf("decode is not deterministic: %+v vs %+v", event, again)
		}
	})
}
