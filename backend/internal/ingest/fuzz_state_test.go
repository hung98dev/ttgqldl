package ingest

import (
	"fmt"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

type fuzzAdmission struct {
	event       source.Event
	actor       string
	action      commands.Action
	reservation *Reservation
}

// FuzzDeduperStateMachine drives a bounded deterministic sequence of admissions,
// redeliveries, conflicts, finalize/release calls and clock changes. It asserts the
// cache remains bounded and terminal operations never panic or corrupt pending state.
func FuzzDeduperStateMachine(f *testing.F) {
	f.Add([]byte{0, 0, 1, 4, 4})
	f.Add([]byte{0, 1, 2, 3, 5, 6, 0, 4})
	f.Add([]byte{8, 8, 0, 1, 7, 0, 4, 5})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, operations []byte) {
		if len(operations) > 48 {
			operations = operations[:48]
		}
		clock := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
		config := DefaultConfig()
		config.Now = clock.Now
		config.MaxEntries = 8
		config.MaxActors = 4
		deduper, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		actionA, _ := commands.ParseComment("XÂY")
		actionB, _ := commands.ParseComment("THỦ")
		var admitted []fuzzAdmission
		var pending []*Reservation
		next := 0
		for _, operation := range operations {
			switch operation % 10 {
			case 0:
				event := source.Event{
					Provider: "mock", SourceSession: "session-a",
					ProviderEventID: fmt.Sprintf("stable-%d", next), Kind: source.KindComment,
				}
				next++
				decision, reservation, err := deduper.Admit(event, actorFor(next%5), actionA)
				if err == nil && decision.Class == ClassAccepted {
					if reservation == nil {
						t.Fatal("accepted without reservation")
					}
					admitted = append(admitted, fuzzAdmission{event: event, actor: string(actorFor(next % 5)), action: actionA, reservation: reservation})
					pending = append(pending, reservation)
				}
			case 1:
				if len(admitted) > 0 {
					last := admitted[len(admitted)-1]
					decision, reservation, err := deduper.Admit(last.event, actorFor(1), last.action)
					if err == nil && decision.Class == ClassAccepted && reservation != nil {
						pending = append(pending, reservation)
					}
				}
			case 2:
				if len(admitted) > 0 {
					last := admitted[len(admitted)-1]
					decision, reservation, err := deduper.Admit(last.event, actorFor(1), actionB)
					if err == nil && decision.Class == ClassAccepted && reservation != nil {
						pending = append(pending, reservation)
					}
				}
			case 3:
				event := source.Event{Provider: "mock", SourceSession: "session-a", Kind: source.KindComment}
				decision, reservation, err := deduper.Admit(event, actorFor(2), actionA)
				if err == nil && decision.Class == ClassAccepted && reservation != nil {
					pending = append(pending, reservation)
				}
			case 4, 5:
				if len(pending) > 0 {
					index := int(operation) % len(pending)
					reservation := pending[index]
					if operation%10 == 4 {
						_ = reservation.Finalize()
						_ = reservation.Finalize()
					} else {
						_ = reservation.Release()
						_ = reservation.Release()
					}
					pending = append(pending[:index], pending[index+1:]...)
				}
			case 6:
				clock.Advance(time.Duration(1+operation%10) * time.Second)
			case 7:
				step := time.Duration(1+operation%10) * time.Second
				if clock.now.After(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Add(step)) {
					clock.now = clock.now.Add(-step)
				}
			case 8:
				clock.Advance(time.Hour)
			case 9:
				stats := deduper.Stats()
				if stats.StableEntries > config.MaxEntries || stats.WindowEntries > config.MaxEntries {
					t.Fatalf("cache exceeded bound: %+v", stats)
				}
				if stats.Pending > config.MaxEntries {
					t.Fatalf("pending exceeded bound: %+v", stats)
				}
			}
			stats := deduper.Stats()
			if stats.StableEntries > config.MaxEntries || stats.WindowEntries > config.MaxEntries || stats.Pending > config.MaxEntries {
				t.Fatalf("unbounded stats: %+v", stats)
			}
		}
		for _, reservation := range pending {
			_ = reservation.Finalize()
			_ = reservation.Release()
		}
	})
}
