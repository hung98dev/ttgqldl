package simulation

import (
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// selectionStep builds one recorded step with a real private digest, so the
// deduplication test exercises the same identity the driver uses.
func selectionStep(t *testing.T, state domain.SessionState, queue RoleQueue, cursor, global int) replayStep {
	t.Helper()
	digest, err := privateRestorationDigest(state, queue)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("selection fixture invalid: %v", err)
	}
	return replayStep{
		Scenario: "selection", Step: global, WaveNo: state.Wave.Number, Tick: state.Tick(),
		Phase: state.Wave.Phase, Lifecycle: state.Lifecycle, Cursor: cursor,
		State: state, Queue: queue, PrivateDigest: digest,
	}
}

// TestSelectCheckpointIdentity pins the equivalence rule for replay checkpoints:
// the complete private identity (state + queues) plus the exact script cursor, and
// never the observational global step.
func TestSelectCheckpointIdentity(t *testing.T) {
	scenario := replayScenario{ID: "selection", CheckpointEvery: 1}
	base, queue := queuedCheckpointFixture(t)

	t.Run("identical state and cursor collapses", func(t *testing.T) {
		steps := []replayStep{
			selectionStep(t, base, queue, 3, 10),
			selectionStep(t, base, queue, 3, 11),
		}
		positions, candidates, duplicates := selectCheckpoints(scenario, steps)
		if candidates != 2 || duplicates != 1 || len(positions) != 1 || positions[0] != 0 {
			t.Fatalf("positions=%v candidates=%d duplicates=%d", positions, candidates, duplicates)
		}
	})

	t.Run("identical state with a different cursor is retained", func(t *testing.T) {
		steps := []replayStep{
			selectionStep(t, base, queue, 3, 10),
			selectionStep(t, base, queue, 4, 11),
		}
		positions, candidates, duplicates := selectCheckpoints(scenario, steps)
		if candidates != 2 || duplicates != 0 || len(positions) != 2 {
			t.Fatalf("positions=%v candidates=%d duplicates=%d", positions, candidates, duplicates)
		}
	})

	t.Run("same canonical hash but different private state is retained", func(t *testing.T) {
		first := collisionAssaultBase(t)
		second := first.Clone()
		second.InputSequence = 5
		hashFirst, err := canonical.Hash(first)
		if err != nil {
			t.Fatal(err)
		}
		hashSecond, err := canonical.Hash(second)
		if err != nil {
			t.Fatal(err)
		}
		if hashFirst != hashSecond {
			t.Fatal("fixture must share the canonical hash to prove the private identity is used")
		}
		steps := []replayStep{
			selectionStep(t, first, RoleQueue{}, 0, 10),
			selectionStep(t, second, RoleQueue{}, 0, 11),
		}
		positions, candidates, duplicates := selectCheckpoints(scenario, steps)
		if candidates != 2 || duplicates != 0 || len(positions) != 2 {
			t.Fatalf("canonical-equal states were collapsed: positions=%v duplicates=%d", positions, duplicates)
		}
	})

	t.Run("same tick with different pause reasons is retained", func(t *testing.T) {
		first := base.Clone()
		first.Lifecycle = domain.LifecyclePaused
		first.PauseReasons = []domain.PauseReason{domain.PauseOperatorPaused}
		second := base.Clone()
		second.Lifecycle = domain.LifecyclePaused
		second.PauseReasons = []domain.PauseReason{domain.PauseOperatorPaused, domain.PauseStorageUnavailable}
		hashFirst, err := canonical.Hash(first)
		if err != nil {
			t.Fatal(err)
		}
		hashSecond, err := canonical.Hash(second)
		if err != nil {
			t.Fatal(err)
		}
		if hashFirst != hashSecond {
			t.Fatal("fixture must share the canonical hash to prove pause reasons are part of the identity")
		}
		steps := []replayStep{
			selectionStep(t, first, queue, 0, 10),
			selectionStep(t, second, queue, 0, 11),
		}
		positions, candidates, duplicates := selectCheckpoints(scenario, steps)
		if candidates != 2 || duplicates != 0 || len(positions) != 2 {
			t.Fatalf("pause-reason states were collapsed: positions=%v duplicates=%d", positions, duplicates)
		}
	})

	t.Run("repeated identical paused states collapse", func(t *testing.T) {
		paused := base.Clone()
		paused.Lifecycle = domain.LifecyclePaused
		paused.PauseReasons = []domain.PauseReason{domain.PauseOperatorPaused}
		steps := []replayStep{
			selectionStep(t, paused, queue, 5, 10),
			selectionStep(t, paused, queue, 5, 11),
			selectionStep(t, paused, queue, 5, 12),
		}
		positions, candidates, duplicates := selectCheckpoints(scenario, steps)
		if candidates != 3 || duplicates != 2 || len(positions) != 1 {
			t.Fatalf("positions=%v candidates=%d duplicates=%d", positions, candidates, duplicates)
		}
	})

	t.Run("pending ballot before and after acceptance is retained", func(t *testing.T) {
		pending := selectionStep(t, base, queue, 0, 10)
		accepted := selectionStep(t, base, RoleQueue{}, 1, 11)
		steps := []replayStep{pending, accepted}
		positions, candidates, duplicates := selectCheckpoints(scenario, steps)
		if candidates != 2 || duplicates != 0 || len(positions) != 2 {
			t.Fatalf("queue identity was not part of the checkpoint key: positions=%v duplicates=%d", positions, duplicates)
		}
	})

	t.Run("rollover boundary states are distinct", func(t *testing.T) {
		waveOne := newState(t)
		waveOne.Wave.Number = 1
		waveOne.Wave.Tick = domain.RolloverTick
		waveOne.Wave.Phase = domain.PhaseResults
		waveOne.Lifecycle = domain.LifecycleRunning
		waveOne.Wave.Result = domain.WaveResultVictory
		if err := waveOne.Validate(); err != nil {
			t.Fatalf("rollover fixture invalid: %v", err)
		}
		rolled, _, err := Next(waveOne, Input{Kind: KindRollover})
		if err != nil {
			t.Fatalf("rollover: %v", err)
		}
		steps := []replayStep{
			selectionStep(t, waveOne, RoleQueue{}, 0, 10),
			selectionStep(t, rolled, RoleQueue{}, 1, 11),
		}
		positions, candidates, duplicates := selectCheckpoints(scenario, steps)
		if candidates != 2 || duplicates != 0 || len(positions) != 2 {
			t.Fatalf("rollover states were collapsed: positions=%v duplicates=%d", positions, duplicates)
		}
	})
}
