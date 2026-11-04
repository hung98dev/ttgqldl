package simulation

import (
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// FuzzTransitionSequences applies arbitrary transition sequences and asserts the
// invariants that must hold for every one of them: no panic, no mutation of a
// rejected input, a valid state after every accepted transition, at most one tick
// per advance, and no fabricated gameplay values.
func FuzzTransitionSequences(f *testing.F) {
	f.Add([]byte{2})
	f.Add([]byte{2, 3, 3, 3, 3})
	f.Add([]byte{4, 3, 3, 5, 3})
	f.Add([]byte{2, 2, 2, 2, 6, 3})
	f.Fuzz(func(t *testing.T, ops []byte) {
		state := newState(t)
		village := state.Village
		if len(ops) > 200 {
			ops = ops[:200]
		}
		for _, op := range ops {
			snapshot := state.Clone()
			var input Input
			switch op % 6 {
			case 0:
				input = Input{Kind: KindAdvanceTick}
			case 1:
				input = Input{Kind: KindStartWave, Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder}}
			case 2:
				input = Input{Kind: KindRollover}
			case 3:
				input = Input{Kind: KindPause, Reason: domain.PauseOperatorPaused}
			case 4:
				input = Input{Kind: KindResume, Reason: domain.PauseOperatorPaused}
			case 5:
				input = Input{Kind: KindStop}
			}

			next, events, err := Next(state, input)
			if err != nil {
				if !reflect.DeepEqual(state, snapshot) {
					t.Fatalf("rejected input mutated the state (op %d)", op%6)
				}
				continue
			}
			if err := next.Validate(); err != nil {
				t.Fatalf("accepted transition produced an invalid state: %v", err)
			}
			if !reflect.DeepEqual(next.Village, village) && next.Wave.Tick < domain.RolloverTick {
				// No W0202 transition may change village resources or progress.
				if next.Village.Resources != village.Resources || next.Village.TotalWins != village.TotalWins {
					t.Fatalf("tick progression changed village values: %+v", next.Village)
				}
			}
			if input.Kind == KindAdvanceTick && next.Tick() != state.Tick()+1 {
				t.Fatalf("advance moved tick from %d to %d", state.Tick(), next.Tick())
			}
			for _, event := range events {
				if event.WaveNo != next.Wave.Number || event.Tick != next.Wave.Tick {
					t.Fatalf("event metadata does not match the produced state: %+v", event)
				}
			}
			state = next
		}
	})
}

// FuzzPauseInterleavings asserts that no interleaving of pause, resume and advance
// ever moves the tick while a pause reason is active, and that clearing one of
// several reasons never resumes time.
func FuzzPauseInterleavings(f *testing.F) {
	f.Add([]byte{0, 1, 0})
	f.Add([]byte{0, 2, 1, 0, 1, 0})
	f.Add([]byte{2, 2, 1, 1, 0})
	f.Fuzz(func(t *testing.T, ops []byte) {
		state, _ := mustStart(t, newState(t))
		if len(ops) > 100 {
			ops = ops[:100]
		}
		reasons := []domain.PauseReason{
			domain.PauseOperatorPaused,
			domain.PauseSourceDisconnected,
			domain.PauseStorageUnavailable,
		}
		for _, op := range ops {
			reason := reasons[int(op)%len(reasons)]
			switch op % 3 {
			case 0:
				next, _, err := Next(state, Input{Kind: KindPause, Reason: reason})
				if err == nil {
					state = next
				}
			case 1:
				next, _, err := Next(state, Input{Kind: KindResume, Reason: reason})
				if err == nil {
					state = next
				}
			case 2:
				tick := state.Tick()
				next, _, err := Next(state, Input{Kind: KindAdvanceTick})
				if len(state.PauseReasons) > 0 {
					if err == nil {
						t.Fatalf("tick advanced while %v was paused", state.PauseReasons)
					}
					if state.Tick() != tick {
						t.Fatal("paused tick moved")
					}
					continue
				}
				if err != nil {
					t.Fatalf("advance while unpaused: %v", err)
				}
				state = next
			}
			if err := state.Validate(); err != nil {
				t.Fatalf("interleaving produced an invalid state: %v", err)
			}
		}
	})
}
