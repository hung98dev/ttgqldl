package ingest

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

type admission struct {
	decision    Decision
	reservation *Reservation
	err         error
}

func TestConcurrentSameEventIDHasOneFirstSeen(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	event := commentEvent("session-a", "mock", "concurrent-1")
	actor := actorFor(30)
	action := mustAction(t, "XÂY")

	results := make(chan admission, 100)
	var wait sync.WaitGroup
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			decision, reservation, err := deduper.Admit(event, actor, action)
			results <- admission{decision: decision, reservation: reservation, err: err}
		}()
	}
	wait.Wait()
	close(results)

	accepted := 0
	var winner *Reservation
	for result := range results {
		if result.err != nil {
			t.Fatalf("unexpected error: %v", result.err)
		}
		switch result.decision.Class {
		case ClassAccepted:
			accepted++
			winner = result.reservation
		case ClassDuplicateProviderID:
		default:
			t.Fatalf("unexpected classification: %+v", result.decision)
		}
	}
	if accepted != 1 || winner == nil {
		t.Fatalf("first-seen count = %d", accepted)
	}
	mustFinalize(t, winner)
}

func TestConcurrentConflictingPayloadsHaveOneWinner(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	event := commentEvent("session-a", "mock", "concurrent-conflict")
	actor := actorFor(31)
	actions := []commands.Action{mustAction(t, "XÂY"), mustAction(t, "THỦ")}

	results := make(chan admission, 2)
	var wait sync.WaitGroup
	for _, action := range actions {
		wait.Add(1)
		go func(action commands.Action) {
			defer wait.Done()
			decision, reservation, err := deduper.Admit(event, actor, action)
			results <- admission{decision: decision, reservation: reservation, err: err}
		}(action)
	}
	wait.Wait()
	close(results)
	accepted, conflicts := 0, 0
	var winner *Reservation
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		switch result.decision.Class {
		case ClassAccepted:
			accepted++
			winner = result.reservation
		case ClassConflictingProviderID:
			conflicts++
		default:
			t.Fatalf("unexpected classification: %+v", result.decision)
		}
	}
	if accepted != 1 || conflicts != 1 || winner == nil {
		t.Fatalf("accepted=%d conflicts=%d", accepted, conflicts)
	}
	mustFinalize(t, winner)
}

func TestConcurrentUniqueActorsAllAccepted(t *testing.T) {
	deduper, _ := newTestDeduper(t, func(config *Config) {
		config.MaxEntries = 500
		config.MaxActors = 500
	})
	action := mustAction(t, "ĐÁNH")
	results := make(chan admission, 100)
	var wait sync.WaitGroup
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			event := commentEvent("session-a", "mock", "unique-"+string(rune('a'+index%26))+"-"+string(rune('a'+index/26)))
			decision, reservation, err := deduper.Admit(event, actorFor(100+index), action)
			results <- admission{decision: decision, reservation: reservation, err: err}
		}(index)
	}
	wait.Wait()
	close(results)
	for result := range results {
		if result.err != nil || result.decision.Class != ClassAccepted || result.reservation == nil {
			t.Fatalf("unique event = %+v/%v", result.decision, result.err)
		}
		mustFinalize(t, result.reservation)
	}
}

func TestConcurrentCapacityNeverOverAdmits(t *testing.T) {
	deduper, _ := newTestDeduper(t, func(config *Config) {
		config.MaxEntries = 10
	})
	action := mustAction(t, "BUILD")
	results := make(chan admission, 100)
	var wait sync.WaitGroup
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			event := commentEvent("session-a", "mock", "capacity-"+string(rune('a'+index%26))+"-"+string(rune('a'+index/26)))
			decision, reservation, err := deduper.Admit(event, actorFor(200+index), action)
			results <- admission{decision: decision, reservation: reservation, err: err}
		}(index)
	}
	wait.Wait()
	close(results)
	accepted, capacity := 0, 0
	for result := range results {
		switch result.decision.Class {
		case ClassAccepted:
			accepted++
			mustFinalize(t, result.reservation)
		case ClassCapacityExceeded:
			capacity++
			if result.err == nil {
				t.Fatal("capacity decision without explicit error")
			}
		default:
			t.Fatalf("unexpected classification: %+v", result.decision)
		}
	}
	if accepted != 10 || capacity != 90 {
		t.Fatalf("accepted=%d capacity=%d", accepted, capacity)
	}
	if stats := deduper.Stats(); stats.StableEntries != 10 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestConcurrentFinalizeIsIdempotent(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	event := commentEvent("session-a", "mock", "finalize-race")
	reservation := mustAccept(t, deduper, event, actorFor(32), mustAction(t, "XÂY"))
	var wait sync.WaitGroup
	failures := make(chan error, 50)
	for index := 0; index < 50; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if err := reservation.Finalize(); err != nil {
				failures <- err
			}
		}()
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Fatalf("finalize race: %v", err)
	}
	if err := reservation.Release(); err == nil {
		t.Fatal("release after finalize unexpectedly succeeded")
	}
}

func TestConcurrentRateLimitPerActor(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	action := mustAction(t, "THỦ")
	var wait sync.WaitGroup
	var mu sync.Mutex
	accepted := map[domain.ActorKey]int{}
	limited := map[domain.ActorKey]int{}
	for actorIndex := 0; actorIndex < 2; actorIndex++ {
		actor := actorFor(300 + actorIndex)
		for eventIndex := 0; eventIndex < 20; eventIndex++ {
			wait.Add(1)
			go func(actor domain.ActorKey, eventIndex int) {
				defer wait.Done()
				event := commentEvent("session-a", "mock", fmt.Sprintf("parallel-%d-%d", actorIndex, eventIndex))
				decision, reservation, err := deduper.Admit(event, actor, action)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					t.Errorf("admit: %v", err)
					return
				}
				switch decision.Class {
				case ClassAccepted:
					accepted[actor]++
					if reservation != nil {
						_ = reservation.Finalize()
					}
				case ClassRateLimited:
					limited[actor]++
				default:
					t.Errorf("unexpected classification: %+v", decision)
				}
			}(actor, eventIndex)
		}
	}
	wait.Wait()
	for actorIndex := 0; actorIndex < 2; actorIndex++ {
		actor := actorFor(300 + actorIndex)
		if accepted[actor] != 10 || limited[actor] != 10 {
			t.Fatalf("actor %d accepted=%d limited=%d", actorIndex, accepted[actor], limited[actor])
		}
	}
}
