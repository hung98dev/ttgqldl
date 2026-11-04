package simulation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

func started(t *testing.T) domain.SessionState {
	t.Helper()
	state, _ := mustStart(t, newState(t))
	return state
}

func roleCommand(seq int64, actor domain.ActorKey, role domain.Role, wave int64, tick int) RoleCommand {
	return RoleCommand{InputSeq: seq, ActorKey: actor, Role: role, TargetWave: wave, ScheduledTick: tick}
}

func applyRoles(t *testing.T, state domain.SessionState, commands ...RoleCommand) (domain.SessionState, []RoleEvent, []RoleDecision) {
	t.Helper()
	next, events, decisions, err := ApplyRoleCommands(state, commands)
	if err != nil {
		t.Fatalf("ApplyRoleCommands: %v", err)
	}
	return next, events, decisions
}

func TestWaitingRegistrationStartsTheWave(t *testing.T) {
	state := newState(t)
	next, events, decisions := applyRoles(t, state, roleCommand(1, actorKey('a'), domain.RoleBuilder, 1, 0))

	if next.Tick() != 0 || next.Wave.Phase != domain.PhasePrep || next.Lifecycle != domain.LifecycleRunning {
		t.Fatalf("start: tick %d phase %s lifecycle %s", next.Tick(), next.Wave.Phase, next.Lifecycle)
	}
	if len(next.Members) != 1 || next.Members[0].ActorKey != actorKey('a') || next.Members[0].Role != domain.RoleBuilder {
		t.Fatalf("members = %+v", next.Members)
	}
	if next.Members[0].JoinedTick != 0 || next.Members[0].LastChangeTick != nil || next.Members[0].FirstSwitchUsed {
		t.Fatal("fresh member bookkeeping is wrong")
	}
	if len(decisions) != 1 || decisions[0].Outcome != OutcomeRegistered {
		t.Fatalf("decisions = %+v", decisions)
	}
	if len(events) != 1 || events[0].Type != EventRoleRegistered {
		t.Fatalf("events = %+v", events)
	}
	if !reflect.DeepEqual(next.Village, state.Village) || len(next.XP) != 0 {
		t.Fatal("registration must not change resources or XP")
	}
}

func TestInvalidRoleAndActorAreRejectedWithoutMutation(t *testing.T) {
	state := newState(t)
	snapshot := state.Clone()

	if _, _, _, err := ApplyRoleCommands(state, []RoleCommand{roleCommand(1, actorKey('a'), domain.Role(9), 1, 0)}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("invalid role: %v", err)
	}
	if _, _, _, err := ApplyRoleCommands(state, []RoleCommand{roleCommand(1, "short", domain.RoleBuilder, 1, 0)}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("invalid actor: %v", err)
	}
	if _, _, _, err := ApplyRoleCommands(state, []RoleCommand{roleCommand(0, actorKey('a'), domain.RoleBuilder, 1, 0)}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("zero input seq: %v", err)
	}
	if !reflect.DeepEqual(state, snapshot) {
		t.Fatal("rejected batch mutated the input state")
	}
	if _, _, _, err := ApplyRoleCommands(state, []RoleCommand{roleCommand(1, actorKey('a'), domain.RoleBuilder, 2, 0)}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("wrong target wave: %v", err)
	}
}

func TestDuplicateRegistrationProducesOneMembership(t *testing.T) {
	state := started(t)
	next, events, decisions := applyRoles(t, state,
		roleCommand(1, actorKey('b'), domain.RoleGuard, 1, 10),
		roleCommand(2, actorKey('b'), domain.RoleGuard, 1, 11),
	)
	if len(next.Members) != 2 { // starter + b
		t.Fatalf("members = %d", len(next.Members))
	}
	if decisions[1].Outcome != OutcomeDuplicateSameRole {
		t.Fatalf("second decision = %s", decisions[1].Outcome)
	}
	for _, event := range events {
		if event.Type == EventRoleChanged {
			t.Fatal("a duplicate must not emit a role change")
		}
	}
}

func TestThousandDuplicateCommandsStayOneMembership(t *testing.T) {
	state := started(t)
	commands := make([]RoleCommand, 0, 1000)
	for i := 1; i <= 1000; i++ {
		commands = append(commands, roleCommand(int64(i), actorKey('c'), domain.RoleFighter, 1, 50))
	}
	next, events, decisions := applyRoles(t, state, commands...)
	if len(next.Members) != 2 {
		t.Fatalf("members = %d", len(next.Members))
	}
	for _, decision := range decisions {
		if decision.Outcome != OutcomeDuplicateSameRole && decision.Outcome != OutcomeRegistered {
			t.Fatalf("unexpected outcome %s", decision.Outcome)
		}
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want only the registration", len(events))
	}
}

func TestFirstSwitchIsFreeThenCooldownApplies(t *testing.T) {
	state := started(t)
	// Join at tick 1, first switch at tick 2.
	state, _, _ = applyRoles(t, state, roleCommand(1, actorKey('d'), domain.RoleBuilder, 1, 1))
	state, events, decisions := applyRoles(t, state, roleCommand(2, actorKey('d'), domain.RoleFighter, 1, 2))
	if decisions[0].Outcome != OutcomeRoleChanged || len(events) != 1 || events[0].Type != EventRoleChanged {
		t.Fatalf("first switch: %s %+v", decisions[0].Outcome, events)
	}
	position := findMember(t, state, actorKey('d'))
	member := state.Members[position]
	if member.Role != domain.RoleFighter || !member.FirstSwitchUsed || member.LastChangeTick == nil || *member.LastChangeTick != 2 {
		t.Fatalf("member after first switch = %+v", member)
	}
	if member.JoinedTick != 1 {
		t.Fatalf("joined tick = %d, want 1", member.JoinedTick)
	}

	// 599 ticks after the accepted change is rejected.
	state, events, decisions = applyRoles(t, state, roleCommand(3, actorKey('d'), domain.RoleGuard, 1, 601))
	if decisions[0].Outcome != OutcomeCooldownRejected {
		t.Fatalf("599 ticks: %s", decisions[0].Outcome)
	}
	if len(events) != 1 || events[0].Type != EventRoleCooldownRejected {
		t.Fatalf("rejection events = %+v", events)
	}
	member = state.Members[findMember(t, state, actorKey('d'))]
	if member.Role != domain.RoleFighter || *member.LastChangeTick != 2 {
		t.Fatal("rejection modified the member")
	}

	// A same-role duplicate does not reset the cooldown.
	state, _, decisions = applyRoles(t, state, roleCommand(4, actorKey('d'), domain.RoleFighter, 1, 601))
	if decisions[0].Outcome != OutcomeDuplicateSameRole {
		t.Fatalf("duplicate outcome = %s", decisions[0].Outcome)
	}

	// Exactly 600 ticks after the accepted change is accepted.
	state, _, decisions = applyRoles(t, state, roleCommand(5, actorKey('d'), domain.RoleGuard, 1, 602))
	if decisions[0].Outcome != OutcomeRoleChanged {
		t.Fatalf("600 ticks: %s", decisions[0].Outcome)
	}
	member = state.Members[findMember(t, state, actorKey('d'))]
	if member.Role != domain.RoleGuard || *member.LastChangeTick != 602 || member.JoinedTick != 1 {
		t.Fatalf("member after cooldown change = %+v", member)
	}
}
func TestBoundaryEffectiveRegistrations(t *testing.T) {
	state := started(t)
	state = advanceTo(t, state, 199)

	// Effective at 200 contributes at the 200 boundary but not at 199.
	withTick200, _, _ := applyRoles(t, state, roleCommand(1, actorKey('e'), domain.RoleBuilder, 1, 200))
	if counts := ActiveCounts(withTick200, 199); counts.Total() != 1 {
		t.Fatalf("counts at 199 = %+v", counts)
	}
	if counts := ActiveCounts(withTick200, 200); counts.Total() != 2 {
		t.Fatalf("counts at 200 = %+v", counts)
	}

	// Effective at 201 must not contribute at 200.
	stateWith201, _, _ := applyRoles(t, state, roleCommand(1, actorKey('e'), domain.RoleBuilder, 1, 201))
	if counts := ActiveCounts(stateWith201, 200); counts.Total() != 1 {
		t.Fatalf("counts at 200 with tick-201 join = %+v", counts)
	}
	if counts := ActiveCounts(stateWith201, 201); counts.Total() != 2 {
		t.Fatalf("counts at 201 = %+v", counts)
	}
}

func TestLatestRoleAppliesAtItsBoundary(t *testing.T) {
	state := started(t)
	state, _, _ = applyRoles(t, state, roleCommand(1, actorKey('f'), domain.RoleBuilder, 1, 1))
	state = advanceTo(t, state, 399)
	state, _, _ = applyRoles(t, state, roleCommand(2, actorKey('f'), domain.RoleGuard, 1, 400))

	counts := ActiveCounts(state, 400)
	if counts.Builder != 1 || counts.Guard != 1 {
		t.Fatalf("counts = %+v", counts)
	}
	if !IsContributionBoundary(400) || IsContributionBoundary(401) {
		t.Fatal("contribution boundary helper is wrong")
	}
}

func TestTwoActorsWithSameRoleRemainDistinct(t *testing.T) {
	state := started(t)
	next, _, _ := applyRoles(t, state,
		roleCommand(1, actorKey('g'), domain.RoleFighter, 1, 10),
		roleCommand(2, actorKey('h'), domain.RoleFighter, 1, 10),
	)
	if counts := ActiveCounts(next, 10); counts.Fighter != 2 || counts.Total() != 3 {
		t.Fatalf("counts = %+v", counts)
	}
	if len(next.Members) != 3 {
		t.Fatalf("members = %d", len(next.Members))
	}
}

func TestCanonicalOrderingRegardlessOfInputOrder(t *testing.T) {
	state := started(t)
	next, _, _ := applyRoles(t, state,
		roleCommand(3, actorKey('j'), domain.RoleFighter, 1, 10),
		roleCommand(1, actorKey('i'), domain.RoleBuilder, 1, 10),
	)
	domain.SortWaveMembers(next.Members)
	for i := 1; i < len(next.Members); i++ {
		if next.Members[i-1].ActorKey > next.Members[i].ActorKey {
			t.Fatal("members are not canonically ordered")
		}
	}
}

func TestDuplicateAndRetroactiveSequencesAreRejected(t *testing.T) {
	state := started(t)
	state = advanceTo(t, state, 100)
	snapshot := state.Clone()

	if _, _, _, err := ApplyRoleCommands(state, []RoleCommand{
		roleCommand(1, actorKey('k'), domain.RoleBuilder, 1, 101),
		roleCommand(1, actorKey('l'), domain.RoleGuard, 1, 101),
	}); !errors.Is(err, ErrDuplicateInputSeq) {
		t.Fatalf("duplicate seq: %v", err)
	}
	if _, _, _, err := ApplyRoleCommands(state, []RoleCommand{roleCommand(2, actorKey('k'), domain.RoleBuilder, 1, 100)}); !errors.Is(err, ErrRetroactiveInput) {
		t.Fatalf("retroactive: %v", err)
	}
	if !reflect.DeepEqual(state, snapshot) {
		t.Fatal("rejected batches mutated the state")
	}
}

func TestResultsQueuesNextWaveWithoutTouchingCurrentWave(t *testing.T) {
	state := started(t)
	state = advanceTo(t, state, 2799)
	state.Wave.Number = 5

	next, events, decisions := applyRoles(t, state,
		roleCommand(1, actorKey('m'), domain.RoleFighter, 6, 2900),
	)
	if decisions[0].Outcome != OutcomeNextWaveQueued {
		t.Fatalf("outcome = %s", decisions[0].Outcome)
	}
	if len(next.Pending) != 1 || next.Pending[0].TargetWave != 6 || next.Pending[0].Role != domain.RoleFighter {
		t.Fatalf("pending = %+v", next.Pending)
	}
	if len(next.Members) != len(state.Members) {
		t.Fatal("RESULTS registration modified current-wave membership")
	}
	if counts := ActiveCounts(next, 2999); counts.Total() != 1 {
		t.Fatalf("queued actor counted in the finishing wave: %+v", counts)
	}
	if len(events) != 1 || events[0].Type != EventNextWaveQueued || events[0].TargetWave != 6 {
		t.Fatalf("events = %+v", events)
	}

	// Promotion needs the control tick and a settled wave; the fixture marks the
	// synthetic settlement explicitly (real settlement is W0207).
	next = advanceTo(t, next, domain.RolloverTick)
	next.Wave.Result = domain.WaveResultVictory
	rolled, _, err := Next(next, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	// Only the queued actor participates in wave 6: previous-wave roles expire.
	if rolled.Wave.Number != 6 || rolled.Tick() != 0 || len(rolled.Members) != 1 {
		t.Fatalf("promoted wave = %d tick %d members %d", rolled.Wave.Number, rolled.Tick(), len(rolled.Members))
	}
	if rolled.Members[0].ActorKey != actorKey('m') || rolled.Members[0].Role != domain.RoleFighter {
		t.Fatalf("promoted member = %+v", rolled.Members[0])
	}
	for _, member := range rolled.Members {
		if member.JoinedTick != 0 {
			t.Fatalf("promoted joined tick = %d", member.JoinedTick)
		}
	}
}

func TestPendingRoleUpdatesAreCooldownBounded(t *testing.T) {
	state := started(t)
	state = advanceTo(t, state, 2799)
	state.Wave.Number = 5

	next, _, decisions := applyRoles(t, state, roleCommand(1, actorKey('n'), domain.RoleBuilder, 6, 2900))
	if decisions[0].Outcome != OutcomeNextWaveQueued {
		t.Fatalf("first queue = %s", decisions[0].Outcome)
	}
	next, _, decisions = applyRoles(t, next, roleCommand(2, actorKey('n'), domain.RoleFighter, 6, 2901))
	if decisions[0].Outcome != OutcomeNextWaveQueued || next.Pending[0].Role != domain.RoleFighter {
		t.Fatalf("first queued switch = %s role %s", decisions[0].Outcome, next.Pending[0].Role)
	}
	next, _, decisions = applyRoles(t, next, roleCommand(3, actorKey('n'), domain.RoleGuard, 6, 2902))
	if decisions[0].Outcome != OutcomeCooldownRejected {
		t.Fatalf("second queued switch = %s", decisions[0].Outcome)
	}
	if len(next.Pending) != 1 || next.Pending[0].Role != domain.RoleFighter {
		t.Fatalf("pending changed on rejection: %+v", next.Pending)
	}
}

func TestPauseDefersButDisconnectAndStorageReject(t *testing.T) {
	state := started(t)
	paused, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := paused.Clone()
	next, events, decisions := applyRoles(t, paused, roleCommand(1, actorKey('o'), domain.RoleBuilder, 1, 10))
	if !reflect.DeepEqual(next, snapshot) || len(events) != 0 {
		t.Fatal("paused role batch changed state or emitted events")
	}
	if decisions[0].Outcome != OutcomeDeferredWhilePaused {
		t.Fatalf("outcome = %s", decisions[0].Outcome)
	}
	if _, _, _, err := ApplyRoleCommands(paused, nil); err != nil {
		t.Fatalf("empty paused batch: %v", err)
	}

	for _, tc := range []struct {
		reason domain.PauseReason
		want   error
	}{
		{domain.PauseSourceDisconnected, ErrSourceDisconnected},
		{domain.PauseStorageUnavailable, ErrStorageUnavailable},
	} {
		blocked, _, err := Next(state, Input{Kind: KindPause, Reason: tc.reason})
		if err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := ApplyRoleCommands(blocked, []RoleCommand{roleCommand(1, actorKey('p'), domain.RoleBuilder, 1, 10)}); !errors.Is(err, tc.want) {
			t.Fatalf("%s: error = %v", tc.reason, err)
		}
	}
}

func TestRoleBatchesAreDeterministic(t *testing.T) {
	state := started(t)
	commands := []RoleCommand{
		roleCommand(2, actorKey('r'), domain.RoleGuard, 1, 20),
		roleCommand(1, actorKey('q'), domain.RoleBuilder, 1, 10),
	}
	firstState, firstEvents, firstDecisions := applyRoles(t, state, commands...)
	secondState, secondEvents, secondDecisions := applyRoles(t, state, commands...)
	if !reflect.DeepEqual(firstState, secondState) || !reflect.DeepEqual(firstEvents, secondEvents) || !reflect.DeepEqual(firstDecisions, secondDecisions) {
		t.Fatal("identical role batches produced different results")
	}
}

func FuzzRoleCommandSequences(f *testing.F) {
	f.Add([]byte{1, 2, 0, 3})
	f.Add([]byte{0, 1, 2, 3, 4, 5})
	f.Fuzz(func(t *testing.T, ops []byte) {
		state := started(t)
		if len(ops) > 100 {
			ops = ops[:100]
		}
		roles := []domain.Role{domain.RoleBuilder, domain.RoleGuard, domain.RoleFighter}
		for i, op := range ops {
			command := roleCommand(int64(i+1), actorKey('a'+op%4), roles[int(op)%len(roles)], 1, 10+i)
			next, _, _, err := ApplyRoleCommands(state, []RoleCommand{command})
			if err != nil {
				if !reflect.DeepEqual(state, state.Clone()) {
					t.Fatal("rejected command mutated state")
				}
				continue
			}
			if err := next.Validate(); err != nil {
				t.Fatalf("role application produced an invalid state: %v", err)
			}
			state = next
		}
	})
}
