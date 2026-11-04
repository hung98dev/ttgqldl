package simulation

import (
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// multiWaveStart builds wave 1 from WAITING: one accepted start command, then
// accepted registrations effective at tick 1.
func multiWaveStart(t *testing.T, participants int) domain.SessionState {
	t.Helper()
	state := newState(t)
	next, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(1, uniqueActor(1), domain.RoleBuilder, 1, 0))
	started, queue, _, _, err := AdvanceWithRoles(next, queue, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: uniqueActor(1), Role: domain.RoleBuilder},
	})
	if err != nil {
		t.Fatal(err)
	}
	commands := make([]RoleCommand, 0, participants-1)
	roles := []domain.Role{domain.RoleBuilder, domain.RoleGuard, domain.RoleFighter}
	for i := 2; i <= participants; i++ {
		commands = append(commands, RoleCommand{
			InputSeq: int64(i), ActorKey: uniqueActor(i), Role: roles[i%3],
			TargetWave: 1, ScheduledTick: 1, AcceptedTick: AcceptedTickUnspecified,
		})
	}
	accepted, queue, _, err := AcceptRoleCommands(started, queue, commands)
	if err != nil {
		t.Fatal(err)
	}
	at1, _ := advanceWithRoles(t, accepted, queue, 1)
	return at1
}

func advanceOneStep(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
	t.Helper()
	next, nextQueue, events, _, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatalf("advance at %d/%d: %v", state.Wave.Number, state.Tick(), err)
	}
	return next, nextQueue, events
}

func TestMultiWaveReplayFromWaiting(t *testing.T) {
	state := multiWaveStart(t, 10)
	queue := RoleQueue{}
	waveResults := map[int64]int{}

	state, queue = advanceWithRoles(t, state, queue, 200)
	if state.Village.Resources.Wood <= 400 {
		t.Fatalf("PREP economy did not run at 200: wood=%d", state.Village.Resources.Wood)
	}
	state, queue = advanceWithRoles(t, state, queue, 1600)
	if !state.Difficulty.Locked || state.Difficulty.PopulationAnchor != 10 {
		t.Fatalf("population lock = %+v", state.Difficulty)
	}
	if len(state.Wave.Pools) != 9 {
		t.Fatalf("enemy pools = %d", len(state.Wave.Pools))
	}
	for state.Tick() < domain.FinalCombatTick {
		var events []Event
		state, queue, events = advanceOneStep(t, state, queue)
		for _, event := range events {
			if event.Type == EventWaveResult {
				waveResults[event.WaveNo]++
			}
		}
	}
	if state.Wave.Result == domain.WaveResultPending {
		t.Fatal("wave 1 was not settled at 2800")
	}
	if waveResults[1] != 1 {
		t.Fatalf("wave 1 wave_result count = %d", waveResults[1])
	}

	// NEXT-wave registrations are accepted during RESULTS (2800..2999), never at
	// the tick-3000 control boundary.
	if state.Tick() != domain.FinalCombatTick {
		t.Fatalf("expected committed tick 2800, got %d", state.Tick())
	}
	for i := 50; i < 53; i++ {
		next, nextQueue, _, err := AcceptRoleCommands(state, queue, []RoleCommand{{
			InputSeq: int64(i), ActorKey: uniqueActor(i), Role: domain.RoleFighter,
			TargetWave: 2, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
		}})
		if err != nil {
			t.Fatalf("queue next wave at %d: %v", state.Tick(), err)
		}
		state, queue = next, nextQueue
	}
	if len(state.Pending) != 3 {
		t.Fatalf("pending = %d", len(state.Pending))
	}
	state, queue = advanceWithRoles(t, state, queue, domain.RolloverTick)
	rolled, _, err := Next(state, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	state = rolled
	queue = RoleQueue{}
	if state.Wave.Number != 2 || state.Tick() != 0 {
		t.Fatalf("wave 2 auto-start = wave %d tick %d", state.Wave.Number, state.Tick())
	}
	if len(state.Members) != 3 {
		t.Fatalf("promoted members = %d", len(state.Members))
	}
	for _, member := range state.Members {
		if member.JoinedTick != 0 || member.Role != domain.RoleFighter || member.XPAwarded {
			t.Fatalf("promoted member = %+v", member)
		}
	}
	if len(state.XP) == 0 {
		t.Fatal("session XP was lost at rollover")
	}

	state, queue = advanceWithRoles(t, state, queue, 799)
	if state.Wave.Vote.Open {
		t.Fatal("vote opened before 800")
	}
	state, queue = advanceWithRoles(t, state, queue, 800)
	if !state.Wave.Vote.Open {
		t.Fatal("vote did not open at 800")
	}
	state, queue = advanceWithRoles(t, state, queue, 899)
	if state.Tick() != 899 || len(state.Wave.Ballots) != 0 {
		t.Fatalf("ballot leaked before its tick: tick=%d ballots=%d", state.Tick(), len(state.Wave.Ballots))
	}
	accepted, queue, _, err := AcceptVoteCommands(state, queue, []VoteCommand{{
		InputSeq: 900, ActorKey: state.Members[0].ActorKey, TargetWave: 2,
		ScheduledTick: 900, Option: domain.VoteOptionA, AcceptedTick: AcceptedTickUnspecified,
	}})
	if err != nil {
		t.Fatalf("accept ballot at 899: %v", err)
	}
	state = accepted
	state, queue, _ = advanceOneStep(t, state, queue)
	if state.Tick() != 900 {
		t.Fatalf("tick = %d", state.Tick())
	}
	if len(state.Wave.Ballots) != 1 || state.Wave.Ballots[0].Tick != 900 || state.Wave.Ballots[0].Option != domain.VoteOptionA {
		t.Fatalf("ballot at 900 = %+v", state.Wave.Ballots)
	}

	state, queue = advanceWithRoles(t, state, queue, 1199)
	woodBeforeClose := state.Village.Resources.Wood
	state, queue = advanceWithRoles(t, state, queue, 1200)
	if !state.Wave.Vote.Closed || state.Wave.Vote.Selected != domain.VoteOptionA {
		t.Fatalf("wave-2 vote = %+v", state.Wave.Vote)
	}
	builders := int64(0)
	for _, member := range state.Members {
		if member.Role == domain.RoleBuilder {
			builders++
		}
	}
	root, _ := domain.Isqrt(builders)
	economy := 2*root + builders/50
	if state.Village.Resources.Wood != woodBeforeClose-50+economy {
		t.Fatalf("vote A charge wrong: before=%d after=%d economy=%d", woodBeforeClose, state.Village.Resources.Wood, economy)
	}

	for state.Tick() < domain.FinalCombatTick {
		var events []Event
		state, queue, events = advanceOneStep(t, state, queue)
		for _, event := range events {
			if event.Type == EventWaveResult {
				waveResults[event.WaveNo]++
			}
		}
	}
	if state.Wave.Result == domain.WaveResultPending {
		t.Fatal("wave 2 was not settled")
	}
	if waveResults[2] != 1 {
		t.Fatalf("wave 2 wave_result count = %d", waveResults[2])
	}
}

func TestNextWaveRegistrationRejectedAtControlTick(t *testing.T) {
	state := multiWaveStart(t, 10)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, domain.RolloverTick)
	state.Wave.Result = domain.WaveResultVictory
	before := state.Clone()
	if _, _, _, err := AcceptRoleCommands(state, RoleQueue{}, []RoleCommand{{
		InputSeq: 99, ActorKey: uniqueActor(99), Role: domain.RoleFighter,
		TargetWave: 2, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
	}}); err == nil {
		t.Fatal("NEXT-wave registration accepted at tick 3000")
	}
	if state.Wave.Number != before.Wave.Number || len(state.Pending) != len(before.Pending) {
		t.Fatal("rejected command mutated the state")
	}
}
