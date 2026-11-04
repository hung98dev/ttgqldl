package simulation

import (
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// scenarioMultiWave is Scenario A: a real two-wave replay. Wave 1 starts from
// WAITING with ten accepted registrations, resolves PREP economy, population lock,
// combat, boss handling and settlement, queues three NEXT-wave registrations during
// RESULTS, rolls over at tick 3000 and auto-starts wave 2, where a ballot is
// accepted at 899 for effective tick 900 and the vote closes at 1200.
func scenarioMultiWave(t *testing.T) replayScenario {
	return replayScenario{
		ID: "A",
		Initial: func(t *testing.T) (domain.SessionState, RoleQueue) {
			return multiWaveStart(t, 10), RoleQueue{}
		},
		Actions: []replayAction{
			{
				name: "queue-next-wave-registrations", wave: 1, tick: domain.FinalCombatTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					var commands []RoleCommand
					for index := 50; index < 53; index++ {
						commands = append(commands, RoleCommand{
							InputSeq: int64(index), ActorKey: uniqueActor(index), Role: domain.RoleFighter,
							TargetWave: state.Wave.Number + 1, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
						})
					}
					next, nextQueue, _, err := AcceptRoleCommands(state, queue, commands)
					if err != nil {
						t.Fatalf("queue next-wave registrations: %v", err)
					}
					return next, nextQueue, nil
				},
			},
			{
				name: "ballot", wave: 2, tick: 899,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					if len(state.Members) == 0 {
						t.Fatal("ballot script has no eligible participant")
					}
					next, nextQueue, _, err := AcceptVoteCommands(state, queue, []VoteCommand{{
						InputSeq: 900, ActorKey: state.Members[0].ActorKey, TargetWave: state.Wave.Number,
						ScheduledTick: 900, Option: domain.VoteOptionA, AcceptedTick: AcceptedTickUnspecified,
					}})
					if err != nil {
						t.Fatalf("accept ballot: %v", err)
					}
					return next, nextQueue, nil
				},
			},
		},
		Done: func(state domain.SessionState) bool {
			return state.Wave.Number == 2 && state.Tick() == domain.RolloverTick
		},
		Limit:           8000,
		CheckpointEvery: 50,
		Critical: map[int64]map[int]bool{
			1: {199: true, 200: true, 799: true, 800: true, 899: true, 900: true, 1199: true, 1200: true,
				1599: true, 1600: true, 1799: true, 1800: true, 2599: true, 2600: true, 2799: true, 2800: true,
				2999: true, 3000: true},
			2: {0: true, 199: true, 200: true, 799: true, 800: true, 899: true, 900: true, 1199: true, 1200: true,
				1599: true, 1600: true, 1799: true, 1800: true, 2599: true, 2600: true, 2799: true, 2800: true,
				2999: true, 3000: true},
		},
		Verify: func(t *testing.T, steps []replayStep) {
			t.Helper()
			waveTwo := findReplayStep(t, steps, 2, 1200)
			if !waveTwo.State.Wave.Vote.Closed || waveTwo.State.Wave.Vote.Selected != domain.VoteOptionA {
				t.Fatalf("wave-2 vote at 1200 = %+v", waveTwo.State.Wave.Vote)
			}
			if len(waveTwo.State.Wave.Ballots) != 1 {
				t.Fatalf("wave-2 applied ballots = %d", len(waveTwo.State.Wave.Ballots))
			}
			rolled := findReplayStep(t, steps, 2, 0)
			if len(rolled.State.Members) != 3 {
				t.Fatalf("promoted wave-2 members = %d, want 3", len(rolled.State.Members))
			}
			for _, member := range rolled.State.Members {
				if member.JoinedTick != 0 {
					t.Fatalf("promoted member joined_tick = %d, want 0", member.JoinedTick)
				}
			}
			if len(rolled.State.Pending) != 0 {
				t.Fatalf("pending registrations were not promoted: %d left", len(rolled.State.Pending))
			}
		},
	}
}

func findReplayStep(t *testing.T, steps []replayStep, wave int64, tick int) replayStep {
	t.Helper()
	for _, step := range steps {
		if step.WaveNo == wave && step.Tick == tick {
			return step
		}
	}
	t.Fatalf("no recorded step at wave %d tick %d", wave, tick)
	return replayStep{}
}

// TestScenarioAMultiWaveRestore keeps the previously achieved Scenario A
// serialized restores: every checkpoint is JSON-serialized, restored, validated,
// integrity-checked and resumed, and the complete state, queues and event stream are
// compared after every remaining step. The sweep also runs under -race because the
// driver no longer hashes every step.
func TestScenarioAMultiWaveRestore(t *testing.T) {
	outcome := runReplayScenario(t, scenarioMultiWave(t))
	if outcome.checkpoints < 148 {
		t.Fatalf("scenario A produced %d checkpoints, the retained minimum is 148", outcome.checkpoints)
	}
	t.Logf("replay scenario=A steps=%d checkpoints=%d restores=%d state_mismatches=%d queue_mismatches=%d event_mismatches=%d final_checks=%d integrity_checks=%d",
		outcome.steps, outcome.checkpoints, outcome.restored, outcome.stateMismatches,
		outcome.queueMismatches, outcome.eventMismatches, outcome.finalStateChecked, outcome.integrityChecks)
}
