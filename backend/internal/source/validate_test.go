package source

import (
	"errors"
	"strings"
	"testing"
)

func validEvent() Event {
	return Event{
		Provider:         "mock",
		SourceSession:    "synthetic-session-001",
		ProviderEventID:  "mock-event-1",
		ProviderViewerID: "mock-viewer-1",
		Alias:            "Villager 0001",
		Message:          "XÂY",
		Kind:             KindComment,
	}
}

func TestValidateEventAcceptsValidEvents(t *testing.T) {
	session := testSession(ModeMock)
	if err := ValidateEvent(validEvent(), session, ModeMock); err != nil {
		t.Fatalf("valid comment rejected: %v", err)
	}
	for _, kind := range []EventKind{KindGift, KindLike, KindShare, KindFollow, KindJoin} {
		event := validEvent()
		event.Kind = kind
		event.Message = ""
		if err := ValidateEvent(event, session, ModeMock); err != nil {
			t.Fatalf("valid %s event rejected: %v", kind, err)
		}
	}
	// Optional identifiers and aliases may be absent.
	minimal := Event{Provider: "mock", SourceSession: session.ID, Kind: KindJoin}
	if err := ValidateEvent(minimal, session, ModeMock); err != nil {
		t.Fatalf("minimal event rejected: %v", err)
	}
}

func TestValidateMessageBoundsAndUnicodeSecurity(t *testing.T) {
	session := testSession(ModeMock)
	mutations := []struct {
		name    string
		message string
		want    error
	}{
		{"exact limit", strings.Repeat("X", MaxCommentBytes), nil},
		{"over limit", strings.Repeat("X", MaxCommentBytes+1), ErrInvalidMessage},
		{"invalid utf8", "XAY\xff", ErrInvalidMessage},
		{"nul", "XAY\x00", ErrInvalidMessage},
		{"bell", "XAY\x07", ErrInvalidMessage},
		{"bidi override", "XAY\u202E", ErrInvalidMessage},
		{"zero width", "XAY\u200B", ErrInvalidMessage},
		{"newline allowed", "XAY\n", nil},
		{"tab allowed", "\tXAY", nil},
	}
	for _, testCase := range mutations {
		t.Run(testCase.name, func(t *testing.T) {
			event := validEvent()
			event.Message = testCase.message
			err := ValidateEvent(event, session, ModeMock)
			if testCase.want == nil {
				if err != nil {
					t.Fatalf("valid message rejected: %v", err)
				}
				return
			}
			if !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestValidateIdentifiers(t *testing.T) {
	session := testSession(ModeMock)
	event := validEvent()
	event.ProviderEventID = strings.Repeat("a", MaxIdentifierRunes)
	event.ProviderViewerID = strings.Repeat("b", MaxIdentifierRunes)
	if err := ValidateEvent(event, session, ModeMock); err != nil {
		t.Fatalf("identifier at limit rejected: %v", err)
	}
	event.ProviderEventID = strings.Repeat("a", MaxIdentifierRunes+1)
	if err := ValidateEvent(event, session, ModeMock); !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("oversized event id error = %v", err)
	}
	event = validEvent()
	event.ProviderViewerID = "viewer\xff"
	if err := ValidateEvent(event, session, ModeMock); !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("invalid utf8 viewer id error = %v", err)
	}
	event = validEvent()
	event.ProviderViewerID = "viewer\u202E"
	if err := ValidateEvent(event, session, ModeMock); !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("bidi viewer id error = %v", err)
	}
}

func TestValidateAlias(t *testing.T) {
	session := testSession(ModeMock)
	cases := []struct {
		name  string
		alias string
		want  error
	}{
		{"simple", "Villager 0001", nil},
		{"grapheme limit", strings.Repeat("a", MaxAliasGraphemes), nil},
		{"over grapheme limit", strings.Repeat("a", MaxAliasGraphemes+1), ErrInvalidAlias},
		{"emoji", strings.Repeat("😀", MaxAliasGraphemes), nil},
		{"bidi", "Villager\u202E0001", ErrInvalidAlias},
		{"zero width", "Villager\u200B", ErrInvalidAlias},
		{"leading mark", "\u0301Villager", ErrInvalidAlias},
		{"mark flood", "V" + strings.Repeat("\u0301", MaxAliasMarks+1), ErrInvalidAlias},
		{"invalid utf8", "Villager\xff", ErrInvalidAlias},
		{"oversized bytes", strings.Repeat("é", MaxAliasBytes), ErrInvalidAlias},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			event := validEvent()
			event.Alias = testCase.alias
			err := ValidateEvent(event, session, ModeMock)
			if testCase.want == nil {
				if err != nil {
					t.Fatalf("valid alias rejected: %v", err)
				}
				return
			}
			if !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestValidateBinding(t *testing.T) {
	session := testSession(ModeMock)
	event := validEvent()

	badProvider := event
	badProvider.Provider = "replay"
	if err := ValidateEvent(badProvider, session, ModeMock); !errors.Is(err, ErrProviderMismatch) {
		t.Fatalf("provider mismatch error = %v", err)
	}

	badSession := event
	badSession.SourceSession = "synthetic-session-002"
	if err := ValidateEvent(badSession, session, ModeMock); !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("session mismatch error = %v", err)
	}

	badKind := event
	badKind.Kind = EventKind(9)
	if err := ValidateEvent(badKind, session, ModeMock); !errors.Is(err, ErrEventKindInvalid) {
		t.Fatalf("kind error = %v", err)
	}

	live := Session{ID: session.ID, Mode: ModeAuthorizedLive}
	if err := ValidateEvent(event, live, ModeAuthorizedLive); !errors.Is(err, ErrModeUnsupported) {
		t.Fatalf("live mode error = %v", err)
	}
	if err := ValidateEvent(event, Session{}, ModeMock); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("invalid session error = %v", err)
	}

	for _, id := range []string{"", strings.Repeat("a", 129), "session\x00"} {
		if (Session{ID: id, Mode: ModeMock}).Valid() {
			t.Fatalf("invalid session id %q accepted", id)
		}
	}
	if (Session{ID: "ok", Mode: Mode(9)}).Valid() {
		t.Fatal("invalid mode accepted")
	}
}
