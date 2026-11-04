package simulation

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// checkpoint is the private pure-reducer snapshot used by the replay tests. It
// carries everything needed for future deterministic evolution, including the
// pending accepted inputs, and is validated by canonical hash on restore.
type checkpoint struct {
	State domain.SessionState `json:"state"`
	Queue RoleQueue           `json:"queue"`
	Hash  string              `json:"hash"`
}

func roundTripCheckpoint(t *testing.T, state domain.SessionState, queue RoleQueue) checkpoint {
	t.Helper()
	hash, err := canonical.Hash(state)
	if err != nil {
		t.Fatalf("checkpoint hash: %v", err)
	}
	encoded, err := json.Marshal(checkpoint{State: state, Queue: queue, Hash: hash})
	if err != nil {
		t.Fatalf("encode checkpoint: %v", err)
	}
	var restored checkpoint
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatalf("decode checkpoint: %v", err)
	}
	if err := restored.State.Validate(); err != nil {
		t.Fatalf("restored state invalid: %v", err)
	}
	recomputed, err := canonical.Hash(restored.State)
	if err != nil || recomputed != restored.Hash {
		t.Fatalf("restored hash mismatch: %v", err)
	}
	return restored
}

func runFullWave(t *testing.T, state domain.SessionState) ([]string, [][]Event) {
	t.Helper()
	hashes := make([]string, 0, domain.RolloverTick)
	events := make([][]Event, 0, domain.RolloverTick)
	for state.Tick() < domain.RolloverTick {
		next, queue, ev, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("advance at %d: %v", state.Tick(), err)
		}
		state = next
		_ = queue
		hash, err := canonical.Hash(state)
		if err != nil {
			t.Fatalf("hash at %d: %v", state.Tick(), err)
		}
		hashes = append(hashes, hash)
		events = append(events, ev)
	}
	return hashes, events
}

func TestSerializedCheckpointRestoreAcrossBoundaries(t *testing.T) {
	state := goldenWaveFixture(t, 2, 2, 6)
	positions := []int{0, 1, 199, 200, 799, 800, 1199, 1200, 1599, 1600, 1799, 1800, 2000, 2400, 2599, 2600, 2799, 2800, 2999, 3000}

	hashes, events := runFullWave(t, state)
	// Re-run once to capture the serialized checkpoint at each position.
	current := state
	restoredCount := 0
	position := 0
	for step := 0; step <= domain.RolloverTick; step++ {
		if position < len(positions) && step == positions[position] {
			restored := roundTripCheckpoint(t, current.Clone(), RoleQueue{})
			resume := restored.State
			for resume.Tick() < domain.RolloverTick {
				next, _, ev, _, err := AdvanceWithRoles(resume, RoleQueue{}, Input{Kind: KindAdvanceTick})
				if err != nil {
					t.Fatalf("resume from %d: %v", step, err)
				}
				index := next.Tick() - 1
				if index >= 0 && index < len(hashes) {
					hash, err := canonical.Hash(next)
					if err != nil {
						t.Fatal(err)
					}
					if hash != hashes[index] {
						t.Fatalf("checkpoint %d diverged at tick %d", step, next.Tick())
					}
					if !reflect.DeepEqual(ev, events[index]) {
						t.Fatalf("checkpoint %d event stream diverged at tick %d", step, next.Tick())
					}
				}
				resume = next
			}
			restoredCount++
			position++
		}
		if step == domain.RolloverTick {
			break
		}
		next, _, _, _, err := AdvanceWithRoles(current, RoleQueue{}, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("capture advance at %d: %v", step, err)
		}
		current = next
	}
	if restoredCount != len(positions) {
		t.Fatalf("restored %d checkpoints, want %d", restoredCount, len(positions))
	}
}

func TestCheckpointCorruptionFailsClosed(t *testing.T) {
	state := goldenWaveFixture(t, 2, 2, 6)
	for i := 0; i < 1600; i++ {
		next, _, _, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatal(err)
		}
		state = next
	}
	encoded, err := json.Marshal(checkpoint{State: state, Queue: RoleQueue{}})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	stateJSON := payload["state"].(map[string]any)
	village := stateJSON["Village"].(map[string]any)
	resources := village["Resources"].(map[string]any)
	resources["Wood"] = 999999.0
	corrupted, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var restored checkpoint
	if err := json.Unmarshal(corrupted, &restored); err != nil {
		t.Fatal(err)
	}
	if err := restored.State.Validate(); err != nil {
		t.Fatalf("mutated state should still be schema-valid: %v", err)
	}
	hash, err := canonical.Hash(restored.State)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := canonical.Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	if hash == stored {
		t.Fatal("tampered checkpoint produced the original hash")
	}
}
