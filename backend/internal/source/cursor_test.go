package source

import (
	"errors"
	"testing"
)

func testSession(mode Mode) Session {
	return Session{ID: "synthetic-session-001", Mode: mode}
}

func TestCursorRoundTrip(t *testing.T) {
	cases := []Cursor{
		NewCursor("synthetic-session-001", "digest-abc", 0),
		NewCursor("synthetic-session-001", "digest-abc", 42),
		NewCursor("session with spaces", "a/b+c=", 1<<40),
	}
	for _, want := range cases {
		encoded := want.Encode()
		got, err := ParseCursor(encoded)
		if err != nil {
			t.Fatalf("ParseCursor(%q): %v", encoded, err)
		}
		if got != want {
			t.Fatalf("round trip = %+v, want %+v", got, want)
		}
	}
}

func TestCursorRejectsInvalidValues(t *testing.T) {
	valid := NewCursor("synthetic-session-001", "digest-abc", 3).Encode()
	cases := []string{
		"",
		"ttgc1",
		"ttgc2." + "c2Vzc2lvbg" + ".ZGlnaWVzdA" + ".0",
		"ttgc1..ZGlnaWVzdA.0",
		"ttgc1.c2Vzc2lvbg..0",
		"ttgc1.c2Vzc2lvbg.ZGlnaWVzdA.-1",
		"ttgc1.c2Vzc2lvbg.ZGlnaWVzdA",
		valid + ".extra",
		"random-provider-id",
		"2026-01-01T00:00:00Z",
	}
	for _, raw := range cases {
		if _, err := ParseCursor(raw); !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("ParseCursor(%q) error = %v, want ErrInvalidCursor", raw, err)
		}
	}
}

func TestCursorBindingRejectsForeignSessionOrSource(t *testing.T) {
	session := testSession(ModeMock)
	cursor := NewCursor(session.ID, "source-a", 2)
	if err := validateCursorBinding(cursor, session, "source-a"); err != nil {
		t.Fatalf("valid binding rejected: %v", err)
	}
	if err := validateCursorBinding(cursor, session, "source-b"); !errors.Is(err, ErrCursorMismatch) {
		t.Fatalf("foreign source error = %v", err)
	}
	other := Session{ID: "synthetic-session-002", Mode: ModeMock}
	if err := validateCursorBinding(cursor, other, "source-a"); !errors.Is(err, ErrCursorMismatch) {
		t.Fatalf("foreign session error = %v", err)
	}
	bad := Cursor{Version: 9, Session: session.ID, SourceID: "source-a"}
	if err := validateCursorBinding(bad, session, "source-a"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("unknown version error = %v", err)
	}
}
