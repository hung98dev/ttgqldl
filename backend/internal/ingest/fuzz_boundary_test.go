package ingest

import (
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/identity"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

// FuzzCrossBoundary composes the real source validation, scoped identity, command
// parser and deduper. It asserts that an accepted candidate always has a valid source
// identity and typed action, and that non-comment events never become gameplay
// candidates.
func FuzzCrossBoundary(f *testing.F) {
	f.Add(uint8(source.KindComment), "XÂY", "viewer-1", "event-1", "synthetic-session-001")
	f.Add(uint8(source.KindComment), "THỦ", "viewer-2", "event-2", "synthetic-session-001")
	f.Add(uint8(source.KindComment), "A", "viewer-3", "event-3", "synthetic-session-001")
	f.Add(uint8(source.KindComment), "XAY DANH", "viewer-4", "event-4", "synthetic-session-001")
	f.Add(uint8(source.KindComment), "XÂY", "", "event-5", "synthetic-session-001")
	f.Add(uint8(source.KindGift), "XÂY", "viewer-6", "gift-1", "synthetic-session-001")
	f.Add(uint8(source.KindJoin), "XÂY", "viewer-7", "join-1", "synthetic-session-001")
	f.Add(uint8(source.KindComment), "XÂY", "viewer-8", "event-8", "wrong-session")
	f.Add(uint8(source.KindComment), "\u202EXÂY", "viewer-9", "event-9", "synthetic-session-001")
	f.Fuzz(func(t *testing.T, kindByte uint8, message, viewer, eventID, sessionID string) {
		if len(message) > 500 || len(viewer) > 256 || len(eventID) > 256 || len(sessionID) > 128 {
			return
		}
		if sessionID == "" {
			sessionID = "synthetic-session-001"
		}
		session := source.Session{ID: sessionID, Mode: source.ModeMock}
		event := source.Event{
			Provider: "mock", SourceSession: sessionID,
			ProviderEventID: eventID, ProviderViewerID: viewer,
			Message: message, Kind: source.EventKind(kindByte % 7),
		}
		if err := source.ValidateEvent(event, session, source.ModeMock); err != nil {
			return
		}
		clock := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
		config := DefaultConfig()
		config.Now = clock.Now
		config.MaxEntries = 8
		config.MaxActors = 8
		deduper, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind != source.KindComment {
			decision, reservation, err := deduper.Admit(event, "", commands.Action{})
			if err != nil || decision.Class != ClassIgnoredNonCommand || reservation != nil {
				t.Fatalf("non-comment reached gameplay: %+v/%v", decision, err)
			}
			return
		}
		key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
		if err != nil {
			t.Fatal(err)
		}
		keys, err := identity.NewKeySet(key, nil)
		if err != nil {
			t.Fatal(err)
		}
		resolved, err := keys.Derive(identity.Input{
			Provider: event.Provider, ProviderVersion: "v1",
			BroadcasterKey: "village-fuzz", ViewerID: event.ProviderViewerID,
		})
		if err != nil {
			return
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			return
		}
		decision, reservation, err := deduper.Admit(event, resolved.ActorKey, action)
		if err != nil {
			return
		}
		if decision.Class != ClassAccepted {
			if reservation != nil {
				t.Fatalf("non-accepted decision returned a reservation: %+v", decision)
			}
			return
		}
		if reservation == nil {
			t.Fatal("accepted candidate has no reservation")
		}
		if !action.Valid() || !resolved.ActorKey.Valid() {
			t.Fatalf("accepted candidate has invalid action/identity: %+v/%s", action, resolved.ActorKey)
		}
		if err := reservation.Finalize(); err != nil {
			t.Fatalf("finalize: %v", err)
		}
	})
}
