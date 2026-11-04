package simulation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// scriptedAction is one deterministic input in the multi-wave replay script.
type scriptedAction struct {
	wave int64
	tick int
	kind string
}

type restoreCheckpoint struct {
	State    domain.SessionState `json:"state"`
	Queue    RoleQueue           `json:"queue"`
	Envelope CheckpointEnvelope  `json:"envelope"`
	Cursor   int                 `json:"cursor"`
	Step     int                 `json:"step"`
}

func multiWaveActions() []scriptedAction {
	return []scriptedAction{
		{wave: 1, tick: 2800, kind: "queue_next_wave"},
		{wave: 2, tick: 899, kind: "ballot"},
	}
}

func applyScriptedAction(t *testing.T, state domain.SessionState, queue RoleQueue, action scriptedAction) (domain.SessionState, RoleQueue) {
	t.Helper()
	switch action.kind {
	case "queue_next_wave":
		for i := 50; i < 53; i++ {
			next, nextQueue, _, err := AcceptRoleCommands(state, queue, []RoleCommand{{
				InputSeq: int64(i), ActorKey: uniqueActor(i), Role: domain.RoleFighter,
				TargetWave: state.Wave.Number + 1, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
			}})
			if err != nil {
				t.Fatalf("queue next wave: %v", err)
			}
			state, queue = next, nextQueue
		}
	case "ballot":
		if len(state.Members) == 0 {
			t.Fatalf("ballot script has no eligible participant")
		}
		next, nextQueue, _, err := AcceptVoteCommands(state, queue, []VoteCommand{{
			InputSeq: 900, ActorKey: state.Members[0].ActorKey, TargetWave: state.Wave.Number,
			ScheduledTick: 900, Option: domain.VoteOptionA, AcceptedTick: AcceptedTickUnspecified,
		}})
		if err != nil {
			t.Fatalf("accept ballot: %v", err)
		}
		state, queue = next, nextQueue
	}
	return state, queue
}

// runScriptFrom advances the real reducer from a restored checkpoint, applying
// remaining scripted actions, and returns the canonical hashes and events per step.
func runScriptFrom(t *testing.T, state domain.SessionState, queue RoleQueue, cursor, step, totalSteps int) ([]string, [][]Event) {
	t.Helper()
	actions := multiWaveActions()
	hashes := make([]string, 0, totalSteps-step)
	events := make([][]Event, 0, totalSteps-step)
	for step < totalSteps {
		var ev []Event
		if state.Wave.Number == 1 && state.Tick() == domain.RolloverTick {
			rolled, rollEvents, err := Next(state, Input{Kind: KindRollover})
			if err != nil {
				t.Fatalf("resume rollover: %v", err)
			}
			state, ev = rolled, rollEvents
		} else {
			next, nextQueue, advanceEvents, _, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
			if err != nil {
				t.Fatalf("step %d advance: %v", step, err)
			}
			state, queue, ev = next, nextQueue, advanceEvents
		}
		step++
		for cursor < len(actions) {
			action := actions[cursor]
			if state.Wave.Number != action.wave || state.Tick() != action.tick {
				break
			}
			state, queue = applyScriptedAction(t, state, queue, action)
			cursor++
		}
		hash, err := canonical.Hash(state)
		if err != nil {
			t.Fatalf("step %d hash: %v", step, err)
		}
		hashes = append(hashes, hash)
		events = append(events, ev)
	}
	return hashes, events
}

func totalWaveSteps(t *testing.T) int {
	t.Helper()
	state := multiWaveStart(t, 10)
	steps := 0
	actions := multiWaveActions()
	cursor := 0
	queue := RoleQueue{}
	for !(state.Wave.Number == 2 && state.Tick() == domain.RolloverTick) {
		var err error
		if state.Wave.Number == 1 && state.Tick() == domain.RolloverTick {
			state, _, err = Next(state, Input{Kind: KindRollover})
		} else {
			state, _, _, _, err = AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
		}
		if err != nil {
			t.Fatalf("count steps: %v", err)
		}
		steps++
		for cursor < len(actions) {
			action := actions[cursor]
			if state.Wave.Number != action.wave || state.Tick() != action.tick {
				break
			}
			state, queue = applyScriptedAction(t, state, queue, action)
			cursor++
		}
		if steps > 12000 {
			t.Fatal("step bound exceeded")
		}
	}
	return steps
}

func TestMultiWaveSerializedRestoreHundred(t *testing.T) {
	baselineState := multiWaveStart(t, 10)
	actions := multiWaveActions()
	total := totalWaveSteps(t)
	// Checkpoint every 50 steps plus both sides of every critical boundary.
	positions := map[int]bool{}
	for step := 0; step < total; step += 50 {
		positions[step] = true
	}
	for _, tick := range []int{199, 200, 799, 800, 899, 900, 1199, 1200, 1599, 1600, 1799, 1800, 2599, 2600, 2799, 2800, 2999, 3000} {
		positions[tick] = true
		positions[tick-1] = true
	}
	// Wave-2 positions are offset by wave 1's 3001 steps.
	waveOne := 3001
	for _, tick := range []int{0, 199, 200, 799, 800, 899, 900, 1199, 1200, 1599, 1600, 1799, 1800, 2599, 2600, 2799, 2800, 2999} {
		positions[waveOne+tick] = true
	}

	state, queue := baselineState.Clone(), RoleQueue{}
	cursor, step := 0, 0
	baselineHashes := make([]string, 0, total)
	baselineEvents := make([][]Event, 0, total)
	checkpoints := make([]restoreCheckpoint, 0, 130)
	hash, err := canonical.Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	if positions[0] {
		envelope, err := BuildCheckpointEnvelope(state, queue)
		if err != nil {
			t.Fatal(err)
		}
		checkpoints = append(checkpoints, restoreCheckpoint{State: state.Clone(), Queue: queue.Clone(), Envelope: envelope, Cursor: cursor, Step: step})
	}
	_ = hash
	for step < total {
		var ev []Event
		if state.Wave.Number == 1 && state.Tick() == domain.RolloverTick {
			rolled, rollEvents, err := Next(state, Input{Kind: KindRollover})
			if err != nil {
				t.Fatalf("baseline rollover: %v", err)
			}
			state, ev = rolled, rollEvents
		} else {
			next, nextQueue, advanceEvents, _, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
			if err != nil {
				t.Fatalf("baseline step %d: %v", step, err)
			}
			state, queue, ev = next, nextQueue, advanceEvents
		}
		step++
		for cursor < len(actions) {
			action := actions[cursor]
			if state.Wave.Number != action.wave || state.Tick() != action.tick {
				break
			}
			state, queue = applyScriptedAction(t, state, queue, action)
			cursor++
		}
		hash, err := canonical.Hash(state)
		if err != nil {
			t.Fatal(err)
		}
		baselineHashes = append(baselineHashes, hash)
		baselineEvents = append(baselineEvents, ev)
		if positions[step] {
			envelope, err := BuildCheckpointEnvelope(state, queue)
			if err != nil {
				t.Fatal(err)
			}
			checkpoints = append(checkpoints, restoreCheckpoint{State: state.Clone(), Queue: queue.Clone(), Envelope: envelope, Cursor: cursor, Step: step})
		}
	}
	if len(checkpoints) < 100 {
		t.Fatalf("only %d checkpoint positions", len(checkpoints))
	}
	if total < 6000 {
		t.Fatalf("scenario too short: %d steps", total)
	}

	passed, failed := 0, 0
	for _, checkpoint := range checkpoints {
		payload, err := json.Marshal(checkpoint)
		if err != nil {
			t.Fatalf("serialize checkpoint %d: %v", checkpoint.Step, err)
		}
		var restored restoreCheckpoint
		if err := json.Unmarshal(payload, &restored); err != nil {
			t.Fatalf("deserialize checkpoint %d: %v", checkpoint.Step, err)
		}
		if err := restored.State.Validate(); err != nil {
			t.Fatalf("restored state %d invalid: %v", checkpoint.Step, err)
		}
		if err := VerifyCheckpointEnvelope(restored.Envelope, restored.State, restored.Queue); err != nil {
			t.Fatalf("restored envelope %d: %v", checkpoint.Step, err)
		}
		restoredHash, err := canonical.Hash(restored.State)
		if err != nil {
			t.Fatal(err)
		}
		if restored.Step > 0 && restoredHash != baselineHashes[restored.Step-1] {
			failed++
			t.Fatalf("checkpoint %d pre-resume hash mismatch", restored.Step)
		}
		hashes, events := runScriptFrom(t, restored.State, restored.Queue, restored.Cursor, restored.Step, total)
		for offset := range hashes {
			if hashes[offset] != baselineHashes[restored.Step+offset] {
				failed++
				t.Fatalf("checkpoint %d hash mismatch at step %d", restored.Step, restored.Step+offset)
			}
			if !reflect.DeepEqual(events[offset], baselineEvents[restored.Step+offset]) {
				failed++
				t.Fatalf("checkpoint %d event mismatch at step %d", restored.Step, restored.Step+offset)
			}
		}
		passed++
	}
	fmt.Printf("multi-wave replay: %d checkpoints passed, %d failed, %d steps\n", passed, failed, total)
}
