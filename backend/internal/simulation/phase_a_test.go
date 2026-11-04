package simulation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

func TestStartWaveRequiresTheAcceptedQueueEntry(t *testing.T) {
	state := newState(t)
	// Empty queue: no wave can start.
	if _, _, _, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindStartWave}); !errors.Is(err, ErrNoRegistration) {
		t.Fatalf("empty queue start: %v", err)
	}

	accepted, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(1, actorKey('a'), domain.RoleBuilder, 1, 0))

	// Unrelated actor or role must not become the authoritative participant.
	if _, _, _, _, err := AdvanceWithRoles(accepted, queue, Input{Kind: KindStartWave, Registration: &Registration{ActorKey: actorKey('b'), Role: domain.RoleBuilder}}); !errors.Is(err, ErrStartMismatch) {
		t.Fatalf("mismatched actor start: %v", err)
	}
	if _, _, _, _, err := AdvanceWithRoles(accepted, queue, Input{Kind: KindStartWave, Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleGuard}}); !errors.Is(err, ErrStartMismatch) {
		t.Fatalf("mismatched role start: %v", err)
	}

	started, leftover, _, roleEvents, err := AdvanceWithRoles(accepted, queue, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder},
	})
	if err != nil {
		t.Fatalf("authorized start: %v", err)
	}
	if len(started.Members) != 1 || started.Members[0].ActorKey != actorKey('a') {
		t.Fatalf("members = %+v", started.Members)
	}
	if leftover.Len() != 0 {
		t.Fatalf("accepted start command was not consumed: queue=%d", leftover.Len())
	}
	acceptedEvents := 0
	for _, event := range roleEvents {
		if event.Type == EventRoleAccepted {
			acceptedEvents++
		}
	}
	if acceptedEvents != 1 {
		t.Fatalf("role_accepted events = %d, want exactly 1", acceptedEvents)
	}
	if len(started.XP) != 0 {
		t.Fatal("start granted XP")
	}

	// A replayed start request is rejected because the wave already runs.
	if _, _, _, _, err := AdvanceWithRoles(started, leftover, Input{Kind: KindStartWave, Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder}}); err == nil {
		t.Fatal("replayed start request was accepted")
	}
}

func TestTwoCompetingFirstRegistrationsAreRejected(t *testing.T) {
	state := newState(t)
	_, _, _, err := AcceptRoleCommands(state, RoleQueue{}, []RoleCommand{
		roleCommand(1, actorKey('a'), domain.RoleBuilder, 1, 0),
		roleCommand(2, actorKey('b'), domain.RoleGuard, 1, 0),
	})
	if !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("competing first registrations: %v", err)
	}
}

func TestFinalBoundaryCommandOwnership(t *testing.T) {
	// Case A: accepted at 2799 for the current wave at 2800 participates in the
	// final contribution/combat boundary.
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 2799)
	accepted, queue, decisions := acceptRoles(t, state, RoleQueue{},
		roleCommand(2, actorKey('a'), domain.RoleFighter, 1, 2800))
	if decisions[0].Outcome != OutcomeScheduled || queue.Len() != 1 {
		t.Fatalf("final-boundary accepted: %s queue=%d", decisions[0].Outcome, queue.Len())
	}
	at2800, remaining := advanceWithRoles(t, accepted, queue, 2800)
	if remaining.Len() != 0 {
		t.Fatal("final-boundary command was not consumed at 2800")
	}
	if at2800.Members[0].Role != domain.RoleFighter {
		t.Fatal("final-boundary command did not apply to the current wave")
	}

	// Case B: accepted at 2800, targets next wave, effective tick 0.
	nextWave, _, decisions := acceptRoles(t, at2800, RoleQueue{},
		roleCommand(3, actorKey('b'), domain.RoleGuard, 2, 0))
	if decisions[0].Outcome != OutcomeNextWaveQueued {
		t.Fatalf("RESULTS command outcome = %s", decisions[0].Outcome)
	}
	if len(nextWave.Pending) != 1 || nextWave.Pending[0].TargetWave != 2 || nextWave.Pending[0].Role != domain.RoleGuard {
		t.Fatalf("pending = %+v", nextWave.Pending)
	}
	if len(nextWave.Members) != len(at2800.Members) {
		t.Fatal("RESULTS command changed the finishing wave membership")
	}
}

func TestNextWaveRegistrationIsRejectedOutsideResults(t *testing.T) {
	for _, tick := range []int{100, 800, 1600, 2799} {
		state, _ := startWithRole(t, 'a', domain.RoleBuilder)
		state, _ = advanceWithRoles(t, state, RoleQueue{}, tick)
		if _, _, _, err := AcceptRoleCommands(state, RoleQueue{}, []RoleCommand{
			roleCommand(5, actorKey('z'), domain.RoleFighter, state.Wave.Number+1, 0),
		}); !errors.Is(err, ErrInvalidCommand) {
			t.Fatalf("tick %d accepted a next-wave registration", tick)
		}
	}
}

func TestBossIncomingIsPerWave(t *testing.T) {
	state := lockedFixture(t, []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}, 0, 0, 5)
	state, _ = advanceOne(t, state) // locked at 1600
	for i := range state.Wave.Pools {
		state.Wave.Pools[i].HP = 0
	}
	notified, events := advanceOne(t, state)
	if !hasEvent(events, EventBossIncoming) || !notified.Wave.BossIncoming {
		t.Fatalf("boss_incoming not emitted: %+v", events)
	}

	settled, _ := advanceWithRoles(t, notified, RoleQueue{}, 3000)
	settled.Wave.Result = domain.WaveResultVictory
	rolled, _, err := Next(settled, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if rolled.Wave.BossIncoming {
		t.Fatal("BossIncoming leaked into the next wave")
	}
	if rolled.Wave.Number != 6 {
		t.Fatalf("wave = %d", rolled.Wave.Number)
	}
}

func TestInvalidPoolMutationsAreRejected(t *testing.T) {
	state := lockedFixture(t, []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}, 0, 0, 1)
	state, _ = advanceOne(t, state) // lock at 1600

	raised := state.Clone()
	raised.Wave.Pools[0].HP = raised.Difficulty.BaseHP + 1
	if err := raised.Validate(); err == nil {
		t.Fatal("pool above its allocation was accepted")
	}

	swapped := state.Clone()
	swapped.Wave.Pools[0], swapped.Wave.Pools[3] = swapped.Wave.Pools[3], swapped.Wave.Pools[0]
	if err := swapped.Validate(); err == nil {
		t.Fatal("non-canonical pool order was accepted")
	}

	depleted := state.Clone()
	for i := range depleted.Wave.Pools {
		depleted.Wave.Pools[i].HP = 0
	}
	if err := depleted.Validate(); err != nil {
		t.Fatalf("legitimately depleted pools rejected: %v", err)
	}

	negative := state.Clone()
	negative.Wave.Pools[1].HP = -1
	if err := negative.Validate(); err == nil {
		t.Fatal("negative pool HP was accepted")
	}
	_ = reflect.DeepEqual(state, state)
}
