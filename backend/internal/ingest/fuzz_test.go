package ingest

import (
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

func FuzzStableScope(f *testing.F) {
	f.Add("session-a", "mock", "event-1")
	f.Add("", "", "")
	f.Add("a|b", "c", "d")
	f.Add("a", "b|c", "d")
	f.Fuzz(func(t *testing.T, session, provider, eventID string) {
		if len(session) > 128 || len(provider) > 64 || len(eventID) > 256 {
			return // scope helpers require validated ingress bounds
		}
		first := stableScope(session, provider, eventID)
		second := stableScope(session, provider, eventID)
		if first != second {
			t.Fatal("stable scope is not deterministic")
		}
		if stableScope(session, provider, eventID+"x") == first {
			t.Fatal("different event ids share a stable scope")
		}
		if stableScope(session, provider+"x", eventID) == first {
			t.Fatal("different providers share a stable scope")
		}
	})
}

func FuzzAdmitIdempotency(f *testing.F) {
	f.Add("event-1", "XÂY")
	f.Add("", "XÂY")
	f.Add("e|v", "BUILD")
	f.Add("event-2", "THỦ")
	f.Add("event-3", "not a command")
	f.Fuzz(func(t *testing.T, eventID, message string) {
		if len(eventID) > 300 || len(message) > 500 {
			return
		}
		action, err := commands.ParseComment(message)
		if err != nil {
			return
		}
		clock := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
		config := DefaultConfig()
		config.Now = clock.Now
		config.MaxEntries = 4
		config.MaxActors = 4
		deduper, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		event := source.Event{
			Provider:        "mock",
			SourceSession:   "session-a",
			ProviderEventID: eventID,
			Kind:            source.KindComment,
		}
		actor := actorFor(999)
		decision, reservation, err := deduper.Admit(event, actor, action)
		if err != nil {
			return // capacity is bounded and explicit
		}
		if decision.Class == ClassAccepted {
			if reservation == nil {
				t.Fatal("accepted without reservation")
			}
			if err := reservation.Finalize(); err != nil {
				t.Fatal(err)
			}
			second, secondReservation, err := deduper.Admit(event, actor, action)
			if err != nil || secondReservation != nil {
				t.Fatalf("second admission = %+v/%v", second, err)
			}
			want := ClassDuplicateProviderID
			if eventID == "" {
				want = ClassRepeatNoop
			}
			if second.Class != want {
				t.Fatalf("second admission class = %s, want %s", second.Class, want)
			}
		}
		stats := deduper.Stats()
		if stats.StableEntries > config.MaxEntries || stats.WindowEntries > config.MaxEntries {
			t.Fatalf("unbounded stats: %+v", stats)
		}
	})
}

func FuzzReservationLifecycle(f *testing.F) {
	f.Add([]byte{0, 1})
	f.Add([]byte{1, 0, 1, 1})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, operations []byte) {
		if len(operations) > 64 {
			return
		}
		clock := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
		config := DefaultConfig()
		config.Now = clock.Now
		deduper, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		event := source.Event{
			Provider:        "mock",
			SourceSession:   "session-a",
			ProviderEventID: "lifecycle",
			Kind:            source.KindComment,
		}
		action, err := commands.ParseComment("XÂY")
		if err != nil {
			t.Fatal(err)
		}
		_, reservation, err := deduper.Admit(event, actorFor(1), action)
		if err != nil || reservation == nil {
			t.Fatalf("admission = %v", err)
		}
		for _, operation := range operations {
			if operation%2 == 0 {
				_ = reservation.Finalize()
			} else {
				_ = reservation.Release()
			}
		}
		// Terminal operations remain idempotent or explicitly closed, never panics.
		_ = reservation.Finalize()
		_ = reservation.Release()
		stats := deduper.Stats()
		if stats.StableEntries > 1 || stats.Pending > 1 {
			t.Fatalf("unbounded lifecycle stats: %+v", stats)
		}
	})
}
