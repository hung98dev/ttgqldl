package source

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzValidateEvent exercises the real source trust boundary. It asserts that an
// accepted event satisfies the documented provider/session/kind/encoding/bounds
// contract and that rejection is deterministic. It does not interpret command
// grammar; the parser and cross-boundary target own that.
func FuzzValidateEvent(f *testing.F) {
	f.Add(uint8(ModeMock), uint8(KindComment), "mock", "synthetic-session-001", "event-1", "viewer-1", "Villager 0001", "XÂY")
	f.Add(uint8(ModeReplay), uint8(KindComment), "replay", "synthetic-session-001", "event-2", "viewer-2", "", "THỦ")
	f.Add(uint8(ModeMock), uint8(KindComment), "mock", "synthetic-session-001", "", "", "", "")
	f.Add(uint8(ModeMock), uint8(KindComment), "mock", "synthetic-session-001", strings.Repeat("e", 256), strings.Repeat("v", 256), "Villager", strings.Repeat("A", 500))
	f.Add(uint8(ModeMock), uint8(KindComment), "mock", "synthetic-session-001", strings.Repeat("e", 257), "viewer", "Villager", strings.Repeat("A", 501))
	f.Add(uint8(ModeMock), uint8(KindGift), "mock", "synthetic-session-001", "gift-1", "viewer-1", "Villager", "XÂY")
	f.Add(uint8(ModeMock), uint8(99), "mock", "synthetic-session-001", "event-1", "viewer-1", "", "XÂY")
	f.Add(uint8(ModeMock), uint8(KindComment), "wrong", "synthetic-session-001", "event-1", "viewer-1", "", "XÂY")
	f.Add(uint8(ModeMock), uint8(KindComment), "mock", "wrong-session", "event-1", "viewer-1", "", "XÂY")
	f.Add(uint8(ModeMock), uint8(KindComment), "mock", "synthetic-session-001", "event-1", "viewer-1", strings.Repeat("a", 65), "XÂY")
	f.Add(uint8(ModeMock), uint8(KindComment), "mock", "synthetic-session-001", "event-1", "viewer-1", "\u202Eevil", "XÂY")
	f.Add(uint8(ModeMock), uint8(KindComment), "mock", "synthetic-session-001", "event-1", "viewer-1", "", "XÂY\xff")
	f.Fuzz(func(t *testing.T, modeByte, kindByte uint8, provider, sessionID, eventID, viewer, alias, message string) {
		if len(provider) > 128 || len(sessionID) > 128 || len(eventID) > 300 || len(viewer) > 300 || len(alias) > 1100 || len(message) > 600 {
			return
		}
		mode := ModeMock
		switch modeByte % 4 {
		case 1:
			mode = ModeReplay
		case 2:
			mode = ModeAuthorizedLive
		case 3:
			mode = Mode(99)
		}
		session := Session{ID: sessionID, Mode: mode}
		event := Event{
			Provider: provider, SourceSession: sessionID,
			ProviderEventID: eventID, ProviderViewerID: viewer,
			Alias: alias, Message: message, Kind: EventKind(kindByte % 7),
		}
		err := ValidateEvent(event, session, mode)
		again := ValidateEvent(event, session, mode)
		if (err == nil) != (again == nil) || (err != nil && again != nil && err.Error() != again.Error()) {
			t.Fatalf("non-deterministic validation: %v then %v", err, again)
		}
		if err != nil {
			return
		}
		if !session.Valid() || (mode != ModeMock && mode != ModeReplay) {
			t.Fatalf("accepted unsupported mode/session: mode=%q session=%+v", mode, session)
		}
		if event.Provider != mode.Provider() || event.SourceSession != session.ID {
			t.Fatalf("accepted mismatched binding: %+v", event)
		}
		if !event.Kind.Valid() {
			t.Fatalf("accepted invalid kind: %v", event.Kind)
		}
		if len(event.Message) > MaxCommentBytes || !utf8.ValidString(event.Message) {
			t.Fatalf("accepted invalid message bytes=%d", len(event.Message))
		}
		if event.ProviderEventID != "" {
			if !utf8.ValidString(event.ProviderEventID) || utf8.RuneCountInString(event.ProviderEventID) > MaxIdentifierRunes {
				t.Fatalf("accepted invalid event id")
			}
		}
		if event.ProviderViewerID != "" {
			if !utf8.ValidString(event.ProviderViewerID) || utf8.RuneCountInString(event.ProviderViewerID) > MaxIdentifierRunes {
				t.Fatalf("accepted invalid viewer id")
			}
		}
		if event.Alias != "" {
			if !utf8.ValidString(event.Alias) || len(event.Alias) > MaxAliasBytes || graphemeClusters(event.Alias) > MaxAliasGraphemes {
				t.Fatalf("accepted invalid alias")
			}
		}
	})
}
