package ingest

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

func TestReservationReleaseAllowsRetry(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	event := commentEvent("session-a", "mock", "E500")
	actor := actorFor(6)
	action := mustAction(t, "XÂY")

	reservation := mustAccept(t, deduper, event, actor, action)
	if err := reservation.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	if stats := deduper.Stats(); stats.StableEntries != 0 || stats.Pending != 0 {
		t.Fatalf("release left state: %+v", stats)
	}
	if err := reservation.Release(); err != nil {
		t.Fatalf("idempotent release: %v", err)
	}
	if err := reservation.Finalize(); !errors.Is(err, ErrReservationClosed) {
		t.Fatalf("finalize after release = %v", err)
	}

	retry := mustAccept(t, deduper, event, actor, action)
	mustFinalize(t, retry)
	if err := retry.Finalize(); err != nil {
		t.Fatalf("idempotent finalize: %v", err)
	}
	if err := retry.Release(); !errors.Is(err, ErrReservationClosed) {
		t.Fatalf("release after finalize = %v", err)
	}
	decision, _, err := admit(t, deduper, event, actor, action)
	if err != nil || decision.Class != ClassDuplicateProviderID {
		t.Fatalf("retry duplicate = %+v/%v", decision, err)
	}
}

func TestStableTTLExpiryBoundaries(t *testing.T) {
	deduper, clock := newTestDeduper(t, func(config *Config) {
		config.StableTTL = time.Second
	})
	event := commentEvent("session-a", "mock", "E600")
	actor := actorFor(7)
	action := mustAction(t, "THỦ")
	mustFinalize(t, mustAccept(t, deduper, event, actor, action))

	clock.Advance(time.Second - time.Nanosecond)
	decision, _, err := admit(t, deduper, event, actor, action)
	if err != nil || decision.Class != ClassDuplicateProviderID {
		t.Fatalf("before TTL = %+v/%v", decision, err)
	}
	clock.Advance(time.Nanosecond) // exactly StableTTL
	decision, reservation, err := admit(t, deduper, event, actor, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("at TTL = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)
}

func TestPendingReservationExpiry(t *testing.T) {
	deduper, clock := newTestDeduper(t, func(config *Config) {
		config.PendingTTL = time.Second
	})
	event := commentEvent("session-a", "mock", "E700")
	actor := actorFor(8)
	action := mustAction(t, "ĐÁNH")

	stale := mustAccept(t, deduper, event, actor, action)
	clock.Advance(time.Second) // exactly PendingTTL
	replacement := mustAccept(t, deduper, event, actor, action)
	if err := stale.Finalize(); !errors.Is(err, ErrReservationExpired) {
		t.Fatalf("stale finalize = %v", err)
	}
	mustFinalize(t, replacement)
	decision, _, err := admit(t, deduper, event, actor, action)
	if err != nil || decision.Class != ClassDuplicateProviderID {
		t.Fatalf("replacement duplicate = %+v/%v", decision, err)
	}
}

func TestBestEffortWindowAndCommandDistinctness(t *testing.T) {
	deduper, clock := newTestDeduper(t, func(config *Config) {
		config.Window = 10 * time.Second
	})
	actor := actorFor(9)
	event := commentEvent("session-a", "mock", "") // no stable event id

	builder := mustAction(t, "XÂY")
	first := mustAccept(t, deduper, event, actor, builder)
	mustFinalize(t, first)
	if decision := firstBestEffort(t, deduper, event, actor, builder); decision.Class != ClassRepeatNoop || !decision.BestEffort {
		t.Fatalf("repeat = %+v", decision)
	}
	// Different canonical actions remain distinguishable.
	for _, message := range []string{"THỦ", "ĐÁNH", "A", "B", "C"} {
		decision, reservation, err := admit(t, deduper, event, actor, mustAction(t, message))
		if err != nil || decision.Class != ClassAccepted || reservation == nil {
			t.Fatalf("distinct %q = %+v/%v", message, decision, err)
		}
		mustFinalize(t, reservation)
	}

	clock.Advance(10*time.Second - time.Nanosecond)
	if decision := firstBestEffort(t, deduper, event, actor, builder); decision.Class != ClassRepeatNoop {
		t.Fatalf("just before window = %+v", decision)
	}
	clock.Advance(time.Nanosecond) // exactly window from the first sample
	decision, reservation, err := admit(t, deduper, event, actor, builder)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("at window boundary = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)
	// The new sample is the sliding reference, not a fixed time bucket.
	clock.Advance(time.Nanosecond)
	if decision := firstBestEffort(t, deduper, event, actor, builder); decision.Class != ClassRepeatNoop {
		t.Fatalf("adjacent window = %+v", decision)
	}
}

func firstBestEffort(t *testing.T, deduper *Deduper, event source.Event, actor domain.ActorKey, action commands.Action) Decision {
	t.Helper()
	decision, reservation, err := deduper.Admit(event, actor, action)
	if err != nil || reservation != nil {
		t.Fatalf("best-effort classification error: %+v/%v", decision, err)
	}
	return decision
}

func TestBestEffortClockRollbackDoesNotBypass(t *testing.T) {
	deduper, clock := newTestDeduper(t, nil)
	event := commentEvent("session-a", "mock", "")
	actor := actorFor(10)
	action := mustAction(t, "ATTACK")
	mustFinalize(t, mustAccept(t, deduper, event, actor, action))
	clock.now = clock.now.Add(-time.Hour)
	if decision := firstBestEffort(t, deduper, event, actor, action); decision.Class != ClassRepeatNoop {
		t.Fatalf("clock rollback bypassed window: %+v", decision)
	}
}

func TestBestEffortReleaseRemovesSample(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	event := commentEvent("session-a", "mock", "")
	actor := actorFor(11)
	action := mustAction(t, "BUILD")
	reservation := mustAccept(t, deduper, event, actor, action)
	if err := reservation.Release(); err != nil {
		t.Fatal(err)
	}
	if stats := deduper.Stats(); stats.WindowEntries != 0 {
		t.Fatalf("release left window entries: %+v", stats)
	}
	decision, retry, err := admit(t, deduper, event, actor, action)
	if err != nil || decision.Class != ClassAccepted || retry == nil {
		t.Fatalf("retry after release = %+v/%v", decision, err)
	}
}

func TestCapacityFailsClosedWithoutEvicting(t *testing.T) {
	deduper, clock := newTestDeduper(t, func(config *Config) {
		config.MaxEntries = 2
		config.StableTTL = time.Second
	})
	actor := actorFor(12)
	action := mustAction(t, "XÂY")
	for index := 0; index < 2; index++ {
		event := commentEvent("session-a", "mock", "cap-"+string(rune('a'+index)))
		mustFinalize(t, mustAccept(t, deduper, event, actor, action))
	}
	third := commentEvent("session-a", "mock", "cap-c")
	decision, reservation, err := admit(t, deduper, third, actor, action)
	if !errors.Is(err, ErrCacheFull) || decision.Class != ClassCapacityExceeded || reservation != nil {
		t.Fatalf("capacity = %+v/%v", decision, err)
	}
	// The first receipt was not evicted to make room.
	first := commentEvent("session-a", "mock", "cap-a")
	decision, _, err = admit(t, deduper, first, actor, action)
	if err != nil || decision.Class != ClassDuplicateProviderID {
		t.Fatalf("unexpired receipt evicted: %+v/%v", decision, err)
	}
	clock.Advance(time.Second)
	decision, reservation, err = admit(t, deduper, third, actor, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("capacity after expiry = %+v/%v", decision, err)
	}
}

func TestNonCommandNeverAllocatesReservation(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	gift := source.Event{Provider: "mock", SourceSession: "session-a", ProviderEventID: "gift-1", Kind: source.KindGift}
	decision, reservation, err := admit(t, deduper, gift, "", commands.Action{})
	if err != nil || reservation != nil || decision.Class != ClassIgnoredNonCommand {
		t.Fatalf("gift = %+v/%v", decision, err)
	}
	// Missing event id non-commands are ignored without cache allocation.
	decision, reservation, err = admit(t, deduper, source.Event{Provider: "mock", SourceSession: "session-a", Kind: source.KindJoin}, "", commands.Action{})
	if err != nil || reservation != nil || decision.Class != ClassIgnoredNonCommand {
		t.Fatalf("join = %+v/%v", decision, err)
	}
}

func TestInvalidInputAndIdentity(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	actor := actorFor(13)
	action := mustAction(t, "XÂY")
	cases := []struct {
		name  string
		event source.Event
		actor domain.ActorKey
		want  error
	}{
		{"empty session", commentEvent("", "mock", "e"), actor, ErrInvalidInput},
		{"empty provider", commentEvent("session-a", "", "e"), actor, ErrInvalidInput},
		{"bad kind", source.Event{Provider: "mock", SourceSession: "session-a", Kind: 99}, actor, ErrInvalidInput},
		{"long event id", commentEvent("session-a", "mock", string(make([]byte, 300))), actor, ErrInvalidInput},
		{"missing actor", commentEvent("session-a", "mock", "e"), "", ErrInvalidIdentity},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, _, err := deduper.Admit(testCase.event, testCase.actor, action)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
		})
	}
	// A 256-rune Unicode event id is valid and must not overflow the scope buffer.
	wide := commentEvent("session-a", "mock", strings.Repeat("界", 256))
	decision, reservation, err := deduper.Admit(wide, actor, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("256-rune event id = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)
	tooWide := commentEvent("session-a", "mock", strings.Repeat("界", 257))
	if _, _, err := deduper.Admit(tooWide, actor, action); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("257-rune event id error = %v", err)
	}

	invalidAction := commands.Action{Kind: commands.KindRole, Role: domain.Role(99)}
	_, _, err = deduper.Admit(commentEvent("session-a", "mock", "e"), actor, invalidAction)
	if !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("invalid action error = %v", err)
	}
	// Comments without event ids require a stable actor identity.
	_, _, err = deduper.Admit(commentEvent("session-a", "mock", ""), "", action)
	if !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("missing best-effort identity error = %v", err)
	}
}

func TestConfigRequiresInjectedClockAndPositiveBounds(t *testing.T) {
	config := DefaultConfig()
	config.Now = nil
	if _, err := New(config); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("nil clock error = %v", err)
	}
	config = DefaultConfig()
	config.MaxEntries = -1
	if _, err := New(config); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("negative entries error = %v", err)
	}
}
