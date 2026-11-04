package simulation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// replayAction is one deterministic input of a replay scenario. It is applied only
// when the committed state matches its (wave, tick[, phase]) selector, so the same
// action list drives the uninterrupted baseline and every resumed continuation.
type replayAction struct {
	name  string
	wave  int64
	tick  int
	phase domain.Phase
	apply func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event)
}

func (a replayAction) matches(state domain.SessionState) bool {
	if state.Wave.Number != a.wave || state.Tick() != a.tick {
		return false
	}
	if a.phase != 0 && state.Wave.Phase != a.phase {
		return false
	}
	return true
}

// replayScenario describes one deterministic replay: fixture, scripted inputs,
// termination and the checkpoint selection policy derived from recorded state.
type replayScenario struct {
	ID      string
	Initial func(t *testing.T) (domain.SessionState, RoleQueue)
	Actions []replayAction
	// Done reports that the final scripted state has been reached.
	Done func(state domain.SessionState) bool
	// Limit bounds the number of script steps.
	Limit int
	// CheckpointEvery selects a checkpoint every N recorded steps.
	CheckpointEvery int
	// Critical holds the (wave, tick) pairs whose state and the state immediately
	// before it must be checkpointed.
	Critical map[int64]map[int]bool
	// Verify asserts the scenario's documented gameplay facts on the baseline.
	Verify func(t *testing.T, steps []replayStep)
}

// replayStep is one fully recorded observation of the common driver. It carries
// everything TASK requires: scenario id, wave, committed tick, phase, global step,
// script cursor, complete state and queues, emitted events and both integrity
// digests.
type replayStep struct {
	Scenario      string
	Step          int
	WaveNo        int64
	Tick          int
	Phase         domain.Phase
	Lifecycle     domain.Lifecycle
	Cursor        int
	State         domain.SessionState
	Queue         RoleQueue
	Events        []Event
	CanonicalHash string
	PrivateDigest string
}

// replayCheckpoint is the complete serialized restoration material.
type replayCheckpoint struct {
	State    domain.SessionState `json:"state"`
	Queue    RoleQueue           `json:"queue"`
	Envelope CheckpointEnvelope  `json:"envelope"`
	Cursor   int                 `json:"cursor"`
	Step     int                 `json:"step"`
}

// resumePoint is where a resumed execution starts.
type resumePoint struct {
	state  domain.SessionState
	queue  RoleQueue
	cursor int
	step   int
}

func (sc replayScenario) advance(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
	t.Helper()
	if state.Tick() == domain.RolloverTick {
		next, events, err := Next(state, Input{Kind: KindRollover})
		if err != nil {
			t.Fatalf("%s: rollover at wave %d: %v", sc.ID, state.Wave.Number, err)
		}
		return next, queue, events
	}
	next, nextQueue, events, _, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatalf("%s: advance at wave %d tick %d: %v", sc.ID, state.Wave.Number, state.Tick(), err)
	}
	return next, nextQueue, events
}

// record builds one step. Integrity digests are recorded on the baseline run; a
// resumed run compares complete states, so it does not pay for hashing every step.
func (sc replayScenario) record(state domain.SessionState, queue RoleQueue, cursor, step int, events []Event, withIntegrity bool) replayStep {
	recorded := replayStep{
		Scenario:  sc.ID,
		Step:      step,
		WaveNo:    state.Wave.Number,
		Tick:      state.Tick(),
		Phase:     state.Wave.Phase,
		Lifecycle: state.Lifecycle,
		Cursor:    cursor,
		State:     state.Clone(),
		Queue:     queue.Clone(),
		Events:    append([]Event(nil), events...),
	}
	if withIntegrity {
		if hash, err := canonical.Hash(state); err == nil {
			recorded.CanonicalHash = hash
		}
		if digest, err := privateRestorationDigest(state, queue); err == nil {
			recorded.PrivateDigest = digest
		}
	}
	return recorded
}

// runScenario executes the deterministic script from the given resume point and
// returns every recorded observation. The first returned step is the start state.
func runScenario(t *testing.T, sc replayScenario, from resumePoint, withIntegrity bool) []replayStep {
	t.Helper()
	state, queue, cursor, step := from.state, from.queue, from.cursor, from.step
	steps := make([]replayStep, 0, 4096)
	appendStep := func(events []Event) {
		steps = append(steps, sc.record(state, queue, cursor, step, events, withIntegrity))
	}
	appendStep(nil)
	if sc.Done != nil && sc.Done(state) {
		return steps
	}
	for step < sc.Limit {
		for cursor < len(sc.Actions) && sc.Actions[cursor].matches(state) {
			action := sc.Actions[cursor]
			next, nextQueue, events := action.apply(t, state, queue)
			state, queue, cursor = next, nextQueue, cursor+1
			step++
			appendStep(events)
		}
		if sc.Done != nil && sc.Done(state) {
			break
		}
		next, nextQueue, events := sc.advance(t, state, queue)
		state, queue = next, nextQueue
		step++
		appendStep(events)
	}
	if len(sc.Actions) > 0 {
		remaining := len(sc.Actions) - cursor
		if remaining != 0 {
			t.Fatalf("%s: script ended with %d unapplied action(s)", sc.ID, remaining)
		}
	}
	return steps
}

// selectCheckpoints derives the checkpoint distribution from the recorded states:
// a fixed stride, the final state, and both sides of every critical boundary.
// Duplicate states with the same script cursor are never counted twice.
func selectCheckpoints(sc replayScenario, steps []replayStep) []int {
	chosen := map[int]bool{}
	add := func(index int) {
		if index < 0 || index >= len(steps) {
			return
		}
		chosen[index] = true
	}
	stride := sc.CheckpointEvery
	if stride <= 0 {
		stride = 50
	}
	for index := 0; index < len(steps); index += stride {
		add(index)
	}
	add(len(steps) - 1)
	if sc.Critical != nil {
		for index, step := range steps {
			ticks := sc.Critical[step.WaveNo]
			if ticks == nil {
				continue
			}
			if ticks[step.Tick] {
				add(index)
				add(index - 1)
			}
		}
	}
	indexes := make([]int, 0, len(chosen))
	for index := range chosen {
		indexes = append(indexes, index)
	}
	sortInts(indexes)
	// Drop duplicated states: two positions with the same complete private digest
	// and the same script cursor would replay identically and prove nothing new.
	seen := map[string]bool{}
	unique := indexes[:0]
	for _, index := range indexes {
		step := steps[index]
		key := fmt.Sprintf("%s|%d|%d", step.PrivateDigest, step.Cursor, step.Step)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, index)
	}
	return unique
}

func sortInts(values []int) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

// replayOutcome aggregates the verified evidence of one scenario.
type replayOutcome struct {
	steps             int
	checkpoints       int
	restored          int
	stateMismatches   int
	queueMismatches   int
	eventMismatches   int
	finalStateChecked int
	integrityChecks   int
}

// runReplayScenario executes the baseline, selects checkpoints from the recorded
// state, restores every checkpoint from serialized bytes and compares the complete
// continuation with the baseline.
func runReplayScenario(t *testing.T, sc replayScenario) replayOutcome {
	t.Helper()
	initialState, initialQueue := sc.Initial(t)
	baseline := runScenario(t, sc, resumePoint{state: initialState, queue: initialQueue}, true)
	if len(baseline) < 10 {
		t.Fatalf("%s: only %d recorded steps", sc.ID, len(baseline))
	}
	if sc.Verify != nil {
		sc.Verify(t, baseline)
	}
	positions := selectCheckpoints(sc, baseline)
	outcome := replayOutcome{steps: len(baseline), checkpoints: len(positions)}

	for _, position := range positions {
		want := baseline[position]
		envelope, err := BuildCheckpointEnvelope(want.State, want.Queue)
		if err != nil {
			t.Fatalf("%s: checkpoint %d envelope: %v", sc.ID, want.Step, err)
		}
		payload, err := json.Marshal(replayCheckpoint{State: want.State, Queue: want.Queue, Envelope: envelope, Cursor: want.Cursor, Step: want.Step})
		if err != nil {
			t.Fatalf("%s: checkpoint %d serialize: %v", sc.ID, want.Step, err)
		}
		var restored replayCheckpoint
		if err := json.Unmarshal(payload, &restored); err != nil {
			t.Fatalf("%s: checkpoint %d deserialize: %v", sc.ID, want.Step, err)
		}
		restored.State = restored.State.Clone()
		restored.Queue = restored.Queue.Clone()
		if err := restored.State.Validate(); err != nil {
			t.Fatalf("%s: checkpoint %d restored state invalid: %v", sc.ID, want.Step, err)
		}
		if !restored.State.Revision.Valid() || restored.State.Revision != domain.CurrentRevision() {
			t.Fatalf("%s: checkpoint %d restored revision %+v", sc.ID, want.Step, restored.State.Revision)
		}
		if err := VerifyCheckpointEnvelope(restored.Envelope, restored.State, restored.Queue); err != nil {
			t.Fatalf("%s: checkpoint %d envelope rejected: %v", sc.ID, want.Step, err)
		}
		outcome.integrityChecks++
		if want.CanonicalHash != "" {
			hash, err := canonical.Hash(restored.State)
			if err != nil {
				t.Fatal(err)
			}
			if hash != want.CanonicalHash {
				t.Fatalf("%s: checkpoint %d canonical hash drift after JSON round-trip", sc.ID, want.Step)
			}
			if restored.Envelope.StateHash != want.CanonicalHash {
				t.Fatalf("%s: checkpoint %d envelope does not carry the baseline canonical hash", sc.ID, want.Step)
			}
		}
		if restored.Step != want.Step || restored.Cursor != want.Cursor {
			t.Fatalf("%s: checkpoint %d restored cursor/step %d/%d", sc.ID, want.Step, restored.Cursor, restored.Step)
		}

		resumed := runScenario(t, sc, resumePoint{
			state: restored.State, queue: restored.Queue, cursor: restored.Cursor, step: restored.Step,
		}, false)
		if len(resumed) != len(baseline)-position {
			t.Fatalf("%s: checkpoint %d resumed %d steps, baseline has %d remaining",
				sc.ID, want.Step, len(resumed), len(baseline)-position)
		}
		for offset, got := range resumed {
			expected := baseline[position+offset]
			if got.Step != expected.Step || got.WaveNo != expected.WaveNo || got.Tick != expected.Tick || got.Cursor != expected.Cursor {
				t.Fatalf("%s: checkpoint %d step mismatch at offset %d: got wave %d tick %d cursor %d, want wave %d tick %d cursor %d",
					sc.ID, want.Step, offset, got.WaveNo, got.Tick, got.Cursor, expected.WaveNo, expected.Tick, expected.Cursor)
			}
			if differences := stateDifferences(got.State, expected.State); len(differences) > 0 {
				outcome.stateMismatches++
				t.Fatalf("%s: checkpoint %d full-state mismatch at global step %d: %v",
					sc.ID, want.Step, expected.Step, differences)
			}
			if !reflect.DeepEqual(got.Queue, expected.Queue) {
				outcome.queueMismatches++
				t.Fatalf("%s: checkpoint %d queue mismatch at global step %d: got %d role/%d ballot, want %d role/%d ballot",
					sc.ID, want.Step, expected.Step,
					len(got.Queue.Entries), len(got.Queue.Ballots), len(expected.Queue.Entries), len(expected.Queue.Ballots))
			}
			// The restored state itself carries no new continuation events: the events
			// that produced it belong to the finalized prefix. Every later step is
			// compared event for event, including ordering.
			if offset > 0 && !reflect.DeepEqual(got.Events, expected.Events) {
				outcome.eventMismatches++
				t.Fatalf("%s: checkpoint %d event-stream mismatch at global step %d: got %d events, want %d",
					sc.ID, want.Step, expected.Step, len(got.Events), len(expected.Events))
			}
		}
		// Final settlement, resources and XP are compared explicitly in addition
		// to the complete-state equality above.
		final, baselineFinal := resumed[len(resumed)-1].State, baseline[len(baseline)-1].State
		if final.Wave.Result != baselineFinal.Wave.Result ||
			final.Village.Resources != baselineFinal.Village.Resources ||
			final.Village.TotalWins != baselineFinal.Village.TotalWins ||
			final.Wave.Number != baselineFinal.Wave.Number ||
			final.Wave.Tick != baselineFinal.Wave.Tick ||
			!reflect.DeepEqual(final.XP, baselineFinal.XP) {
			outcome.finalStateChecked++
			t.Fatalf("%s: checkpoint %d final settlement mismatch: result=%s resources=%+v xp=%d",
				sc.ID, want.Step, final.Wave.Result, final.Village.Resources, totalXP(final))
		}
		outcome.finalStateChecked++
		outcome.restored++
	}
	return outcome
}

// stateDifferences lists the authoritative fields that differ. It exists so a
// failure names the field instead of printing two large structures.
func stateDifferences(got, want domain.SessionState) []string {
	var differences []string
	check := func(name string, left, right any) {
		if !reflect.DeepEqual(left, right) {
			differences = append(differences, fmt.Sprintf("%s=%v want %v", name, left, right))
		}
	}
	check("revision", got.Revision, want.Revision)
	check("session_id", got.ID, want.ID)
	check("village_id", got.VillageID, want.VillageID)
	check("mode", got.Mode, want.Mode)
	check("lifecycle", got.Lifecycle, want.Lifecycle)
	check("pause_reasons", got.PauseReasons, want.PauseReasons)
	check("village", got.Village, want.Village)
	check("wave", got.Wave, want.Wave)
	check("members", got.Members, want.Members)
	check("pending", got.Pending, want.Pending)
	check("xp", got.XP, want.XP)
	check("difficulty", got.Difficulty, want.Difficulty)
	check("consecutive_wins", got.ConsecutiveWins, want.ConsecutiveWins)
	check("consecutive_losses", got.ConsecutiveLosses, want.ConsecutiveLosses)
	check("privacy_epoch", got.PrivacyEpoch, want.PrivacyEpoch)
	check("input_sequence", got.InputSequence, want.InputSequence)
	return differences
}
