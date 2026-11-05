package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func newTestDeduper(t *testing.T, mutate func(*Config)) (*Deduper, *testClock) {
	t.Helper()
	clock := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	config := DefaultConfig()
	config.Now = clock.Now
	config.MaxEntries = 1000
	config.MaxActors = 1000
	if mutate != nil {
		mutate(&config)
	}
	deduper, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	return deduper, clock
}

func actorFor(index int) domain.ActorKey {
	sum := sha256.Sum256([]byte(fmt.Sprintf("test-actor-%d", index)))
	return domain.ActorKey(hex.EncodeToString(sum[:]))
}

func mustAction(t *testing.T, message string) commands.Action {
	t.Helper()
	action, err := commands.ParseComment(message)
	if err != nil {
		t.Fatal(err)
	}
	return action
}

func commentEvent(session, provider, eventID string) source.Event {
	return source.Event{
		Provider:        provider,
		SourceSession:   session,
		ProviderEventID: eventID,
		Kind:            source.KindComment,
	}
}

func admit(t *testing.T, deduper *Deduper, event source.Event, actor domain.ActorKey, action commands.Action) (Decision, *Reservation, error) {
	t.Helper()
	return deduper.Admit(event, actor, action)
}

func mustAccept(t *testing.T, deduper *Deduper, event source.Event, actor domain.ActorKey, action commands.Action) *Reservation {
	t.Helper()
	decision, reservation, err := deduper.Admit(event, actor, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("expected accepted candidate, got %+v err=%v", decision, err)
	}
	return reservation
}

func mustFinalize(t *testing.T, reservation *Reservation) {
	t.Helper()
	if err := reservation.Finalize(); err != nil {
		t.Fatalf("finalize: %v", err)
	}
}

func TestStableFirstSeenDuplicateAndScopeIsolation(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	event := commentEvent("session-a", "mock", "E100")
	actor := actorFor(1)
	builder := mustAction(t, "XÂY")

	reservation := mustAccept(t, deduper, event, actor, builder)
	mustFinalize(t, reservation)

	decision, second, err := admit(t, deduper, event, actor, builder)
	if err != nil || decision.Class != ClassDuplicateProviderID || second != nil {
		t.Fatalf("duplicate = %+v/%v", decision, err)
	}

	// Same event id in another provider or session is independent.
	for _, other := range []source.Event{
		commentEvent("session-a", "replay", "E100"),
		commentEvent("session-b", "mock", "E100"),
	} {
		decision, reservation, err := admit(t, deduper, other, actor, builder)
		if err != nil || decision.Class != ClassAccepted {
			t.Fatalf("independent scope %+v rejected: %+v/%v", other, decision, err)
		}
		mustFinalize(t, reservation)
	}
}

func TestConflictingProviderEventID(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	event := commentEvent("session-a", "mock", "E200")
	actor := actorFor(2)
	builder := mustAction(t, "XÂY")
	fighter := mustAction(t, "ĐÁNH")
	ballotA := mustAction(t, "A")

	first := mustAccept(t, deduper, event, actor, builder)
	mustFinalize(t, first)

	cases := []struct {
		name   string
		actor  domain.ActorKey
		action commands.Action
	}{
		{"different command", actor, fighter},
		{"different actor", actorFor(3), builder},
		{"different ballot", actor, ballotA},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			decision, reservation, err := admit(t, deduper, event, testCase.actor, testCase.action)
			if err != nil || decision.Class != ClassConflictingProviderID || reservation != nil {
				t.Fatalf("conflict = %+v/%v", decision, err)
			}
		})
	}

	// The first fingerprint wins and is not overwritten by a conflict.
	decision, reservation, err := admit(t, deduper, event, actor, builder)
	if err != nil || decision.Class != ClassDuplicateProviderID || reservation != nil {
		t.Fatalf("original identity after conflict = %+v/%v", decision, err)
	}
	if stats := deduper.Stats(); stats.StableEntries != 1 {
		t.Fatalf("conflict allocated entries: %+v", stats)
	}
}

func TestPresentationAndUnicodeVariantsAreDuplicates(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	actor := actorFor(4)
	event := commentEvent("session-a", "mock", "E300")
	first := mustAccept(t, deduper, event, actor, mustAction(t, "XÂY"))
	mustFinalize(t, first)

	for _, variant := range []string{"XAY", "xây", "BUILD", " build "} {
		decision, reservation, err := admit(t, deduper, event, actor, mustAction(t, variant))
		if err != nil || decision.Class != ClassDuplicateProviderID || reservation != nil {
			t.Fatalf("variant %q = %+v/%v", variant, decision, err)
		}
	}
	// Distinct actions remain distinguishable.
	for _, distinct := range []string{"THỦ", "ĐÁNH", "A", "B"} {
		decision, reservation, err := admit(t, deduper, event, actor, mustAction(t, distinct))
		if err != nil || decision.Class != ClassConflictingProviderID || reservation != nil {
			t.Fatalf("distinct action %q = %+v/%v", distinct, decision, err)
		}
	}
}

func TestDifferentEventKindConflicts(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	actor := actorFor(5)
	comment := commentEvent("session-a", "mock", "E400")
	gift := source.Event{Provider: "mock", SourceSession: "session-a", ProviderEventID: "E400", Kind: source.KindGift}

	decision, reservation, err := admit(t, deduper, gift, "", commands.Action{})
	if err != nil || decision.Class != ClassIgnoredNonCommand || reservation != nil {
		t.Fatalf("gift = %+v/%v", decision, err)
	}
	decision, reservation, err = admit(t, deduper, comment, actor, mustAction(t, "XÂY"))
	if err != nil || decision.Class != ClassConflictingProviderID || reservation != nil {
		t.Fatalf("comment reusing gift id = %+v/%v", decision, err)
	}

	// A non-command reused with the same kind is a duplicate but never a candidate.
	decision, reservation, err = admit(t, deduper, gift, "", commands.Action{})
	if err != nil || decision.Class != ClassDuplicateProviderID || reservation != nil {
		t.Fatalf("gift duplicate = %+v/%v", decision, err)
	}
}
