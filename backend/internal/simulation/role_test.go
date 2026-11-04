package simulation

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

func roleCommand(seq int64, actor domain.ActorKey, role domain.Role, wave int64, tick int) RoleCommand {
	return RoleCommand{InputSeq: seq, ActorKey: actor, Role: role, TargetWave: wave, ScheduledTick: tick, AcceptedTick: AcceptedTickUnspecified}
}

func acceptRoles(t *testing.T, state domain.SessionState, queue RoleQueue, commands ...RoleCommand) (domain.SessionState, RoleQueue, []RoleDecision) {
	t.Helper()
	next, nextQueue, decisions, err := AcceptRoleCommands(state, queue, commands)
	if err != nil {
		t.Fatalf("AcceptRoleCommands: %v", err)
	}
	return next, nextQueue, decisions
}

// startWithRole starts a wave with one accepted registration.
func startWithRole(t *testing.T, actor byte, role domain.Role) (domain.SessionState, RoleQueue) {
	t.Helper()
	state := newState(t)
	next, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(1, actorKey(actor), role, 1, 0))
	started, _, _, _, err := AdvanceWithRoles(next, queue, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: actorKey(actor), Role: role},
	})
	if err != nil {
		t.Fatalf("start wave: %v", err)
	}
	return started, RoleQueue{}
}

// advanceWithRoles steps one tick at a time, applying due role commands.
func advanceWithRoles(t *testing.T, state domain.SessionState, queue RoleQueue, target int) (domain.SessionState, RoleQueue) {
	t.Helper()
	for state.Tick() < target {
		next, nextQueue, _, _, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("advance at tick %d: %v", state.Tick(), err)
		}
		state, queue = next, nextQueue
	}
	return state, queue
}

func TestWaitingRegistrationSchedulesThenStarts(t *testing.T) {
	state := newState(t)
	next, queue, decisions := acceptRoles(t, state, RoleQueue{}, roleCommand(1, actorKey('a'), domain.RoleBuilder, 1, 0))
	if decisions[0].Outcome != OutcomeScheduled {
		t.Fatalf("outcome = %s", decisions[0].Outcome)
	}
	if !reflect.DeepEqual(next, state) {
		t.Fatal("accepting a waiting registration mutated the state")
	}
	if queue.Len() != 1 {
		t.Fatalf("queue = %d", queue.Len())
	}
	if _, _, _, _, err := ApplyDueRoles(next, queue); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("apply before start: %v", err)
	}

	started, _, _, roleEvents, err := AdvanceWithRoles(next, queue, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if started.Tick() != 0 || len(started.Members) != 1 || started.Members[0].JoinedTick != 0 {
		t.Fatalf("started state = tick %d members %+v", started.Tick(), started.Members)
	}
	if len(roleEvents) != 1 || roleEvents[0].Type != EventRoleAccepted {
		t.Fatalf("role events = %+v", roleEvents)
	}
	if !reflect.DeepEqual(started.Village, state.Village) || len(started.XP) != 0 {
		t.Fatal("registration changed resources or XP")
	}
}

func TestFutureCommandCannotMutateCurrentState(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 100)
	snapshot := state.Clone()

	next, queue, decisions := acceptRoles(t, state, RoleQueue{}, roleCommand(2, actorKey('a'), domain.RoleFighter, 1, 800))
	if decisions[0].Outcome != OutcomeScheduled {
		t.Fatalf("outcome = %s", decisions[0].Outcome)
	}
	if !reflect.DeepEqual(next, snapshot) {
		t.Fatal("accepting a future command mutated authoritative state")
	}
	if next.Members[0].Role != domain.RoleBuilder {
		t.Fatal("future command changed the role early")
	}

	at600, queue := advanceWithRoles(t, next, queue, 600)
	if at600.Members[0].Role != domain.RoleBuilder {
		t.Fatal("a future role change corrupted an earlier boundary")
	}
	if counts := ActiveCounts(at600, 600); counts.Builder != 1 {
		t.Fatalf("counts at 600 = %+v", counts)
	}

	at800, remaining := advanceWithRoles(t, at600, queue, 800)
	if at800.Members[0].Role != domain.RoleFighter {
		t.Fatal("command did not apply at its scheduled tick")
	}
	if remaining.Len() != 0 {
		t.Fatalf("queue after application = %d", remaining.Len())
	}
	if counts := ActiveCounts(at800, 800); counts.Fighter != 1 {
		t.Fatalf("counts at 800 = %+v", counts)
	}
}

func TestMultipleFutureCommandsApplyAtTheirOwnTicks(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 10)

	next, queue, _ := acceptRoles(t, state, RoleQueue{},
		roleCommand(2, actorKey('a'), domain.RoleFighter, 1, 100),
		roleCommand(3, actorKey('a'), domain.RoleGuard, 1, 700),
	)
	if queue.Len() != 2 {
		t.Fatalf("queue = %d", queue.Len())
	}

	at99, _ := advanceWithRoles(t, next, queue.Clone(), 99)
	if at99.Members[0].Role != domain.RoleBuilder {
		t.Fatal("role changed before its tick")
	}
	at100, _ := advanceWithRoles(t, next, queue.Clone(), 100)
	if at100.Members[0].Role != domain.RoleFighter {
		t.Fatal("first future change did not apply at 100")
	}
	at700, finalQueue := advanceWithRoles(t, next, queue.Clone(), 700)
	if at700.Members[0].Role != domain.RoleGuard {
		t.Fatal("second future change did not apply at 700")
	}
	if finalQueue.Len() != 0 {
		t.Fatalf("queue after 700 = %d", finalQueue.Len())
	}
}

func TestRetroactiveAndDuplicateSequenceCommandsAreRejected(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 100)
	snapshot := state.Clone()

	if _, _, _, err := AcceptRoleCommands(state, RoleQueue{}, []RoleCommand{roleCommand(5, actorKey('b'), domain.RoleGuard, 1, 100)}); !errors.Is(err, ErrRetroactiveInput) {
		t.Fatalf("retroactive: %v", err)
	}
	if _, _, _, err := AcceptRoleCommands(state, RoleQueue{}, []RoleCommand{
		roleCommand(6, actorKey('b'), domain.RoleGuard, 1, 200),
		roleCommand(6, actorKey('c'), domain.RoleGuard, 1, 200),
	}); !errors.Is(err, ErrDuplicateInputSeq) {
		t.Fatalf("duplicate seq: %v", err)
	}
	if _, _, _, err := AcceptRoleCommands(state, RoleQueue{}, []RoleCommand{roleCommand(7, actorKey('b'), domain.RoleGuard, 2, 200)}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("wrong target wave: %v", err)
	}
	if !reflect.DeepEqual(state, snapshot) {
		t.Fatal("rejected acceptance mutated state")
	}
}

func TestCooldownAppliesOnlyAfterFirstSwitch(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 10)
	state, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(2, actorKey('a'), domain.RoleFighter, 1, 11))
	state, _ = advanceWithRoles(t, state, queue, 11)
	if state.Members[0].Role != domain.RoleFighter || !state.Members[0].FirstSwitchUsed {
		t.Fatalf("first switch state = %+v", state.Members[0])
	}

	state, queue, _ = acceptRoles(t, state, RoleQueue{}, roleCommand(3, actorKey('a'), domain.RoleGuard, 1, 610))
	state, _ = advanceWithRoles(t, state, queue, 610)
	if state.Members[0].Role != domain.RoleFighter {
		t.Fatal("cooldown rejection changed the role")
	}
	state, queue, _ = acceptRoles(t, state, RoleQueue{}, roleCommand(4, actorKey('a'), domain.RoleGuard, 1, 611))
	state, _ = advanceWithRoles(t, state, queue, 611)
	if state.Members[0].Role != domain.RoleGuard {
		t.Fatal("600-tick change was not accepted")
	}
}

func TestResultsCommandsQueueNextWaveIncludingTick3000(t *testing.T) {
	for _, acceptedAt := range []int{2800, 2900, 3000} {
		state, _ := startWithRole(t, 'a', domain.RoleBuilder)
		state, _ = advanceWithRoles(t, state, RoleQueue{}, acceptedAt)
		state.Wave.Number = 5
		state.Difficulty = domain.Difficulty{Level: 1}
		_ = state.Validate()

		next, queue, decisions := acceptRoles(t, state, RoleQueue{},
			roleCommand(9, actorKey('m'), domain.RoleFighter, 6, 0))
		if decisions[0].Outcome != OutcomeNextWaveQueued {
			t.Fatalf("accepted at %d: outcome = %s", acceptedAt, decisions[0].Outcome)
		}
		if queue.Len() != 0 {
			t.Fatal("RESULTS command entered the current-wave queue")
		}
		if len(next.Pending) != 1 || next.Pending[0].TargetWave != 6 || next.Pending[0].Role != domain.RoleFighter {
			t.Fatalf("accepted at %d: pending = %+v", acceptedAt, next.Pending)
		}
		if len(next.Members) != 1 {
			t.Fatal("RESULTS command changed current membership")
		}

		advanced, _ := advanceWithRoles(t, next, RoleQueue{}, 3000)
		advanced.Wave.Result = domain.WaveResultVictory
		rolled, _, err := Next(advanced, Input{Kind: KindRollover})
		if err != nil {
			t.Fatalf("accepted at %d: rollover: %v", acceptedAt, err)
		}
		if len(rolled.Members) != 1 || rolled.Members[0].ActorKey != actorKey('m') || rolled.Members[0].JoinedTick != 0 {
			t.Fatalf("accepted at %d: promoted = %+v", acceptedAt, rolled.Members)
		}
		if rolled.Members[0].Role != domain.RoleFighter {
			t.Fatal("queued role was lost")
		}
	}
}

func TestResultsDuplicateQueuesOneRegistration(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 2800)
	state.Wave.Number = 5
	state.Difficulty = domain.Difficulty{Level: 1}

	next, _, _ := acceptRoles(t, state, RoleQueue{},
		roleCommand(1, actorKey('n'), domain.RoleBuilder, 6, 0),
		roleCommand(2, actorKey('n'), domain.RoleBuilder, 6, 0))
	if len(next.Pending) != 1 {
		t.Fatalf("pending = %d", len(next.Pending))
	}
	next, _, _ = acceptRoles(t, next, RoleQueue{}, roleCommand(3, actorKey('n'), domain.RoleGuard, 6, 0))
	if next.Pending[0].Role != domain.RoleGuard {
		t.Fatal("queued role update lost")
	}
	next, _, decisions := acceptRoles(t, next, RoleQueue{}, roleCommand(4, actorKey('n'), domain.RoleFighter, 6, 0))
	if decisions[0].Outcome != OutcomeCooldownRejected {
		t.Fatalf("second queued change = %s", decisions[0].Outcome)
	}
	if len(next.Pending) != 1 || next.Pending[0].Role != domain.RoleGuard {
		t.Fatal("pending state corrupted by rejection")
	}
}

func TestPausedCommandsAreDeferredNotApplied(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 50)
	paused, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := paused.Clone()

	_, queue, decisions, err := AcceptRoleCommands(paused, RoleQueue{}, []RoleCommand{roleCommand(2, actorKey('b'), domain.RoleGuard, 1, 100)})
	var deferred *DeferredError
	if !errors.As(err, &deferred) || !errors.Is(err, ErrInputDeferred) {
		t.Fatalf("paused acceptance error = %v", err)
	}
	if len(deferred.Commands) != 1 || deferred.Commands[0].ActorKey != actorKey('b') {
		t.Fatal("deferred commands were not returned to the caller")
	}
	if !reflect.DeepEqual(paused, snapshot) || queue.Len() != 0 || len(decisions) != 0 {
		t.Fatal("paused acceptance mutated state or reported success")
	}
	if _, _, _, err := AcceptRoleCommands(paused, RoleQueue{}, []RoleCommand{roleCommand(3, "bad", domain.RoleGuard, 1, 100)}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("paused invalid command: %v", err)
	}

	resumed, _, err := Next(paused, Input{Kind: KindResume, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	accepted, queue, _ := acceptRoles(t, resumed, RoleQueue{}, deferred.Commands...)
	applied, _ := advanceWithRoles(t, accepted, queue, 100)
	if len(applied.Members) != 2 {
		t.Fatalf("members after deferred recovery = %d", len(applied.Members))
	}
}

func TestDisconnectAndStorageRejectInput(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
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
		if _, _, _, err := AcceptRoleCommands(blocked, RoleQueue{}, []RoleCommand{roleCommand(1, actorKey('b'), domain.RoleGuard, 1, 10)}); !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v", tc.reason, err)
		}
	}
}

func TestCanonicalOrderingIsAssertedDirectly(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	accepted, queue, _ := acceptRoles(t, state, RoleQueue{},
		roleCommand(2, actorKey('f'), domain.RoleGuard, 1, 20),
		roleCommand(3, actorKey('b'), domain.RoleFighter, 1, 20))
	at20, _ := advanceWithRoles(t, accepted, queue, 20)

	for i := 1; i < len(at20.Members); i++ {
		if at20.Members[i-1].ActorKey > at20.Members[i].ActorKey {
			t.Fatalf("members are not canonically ordered: %s > %s", at20.Members[i-1].ActorKey, at20.Members[i].ActorKey)
		}
	}
}

func TestDuplicateMessagesStayOneMembership(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	commands := make([]RoleCommand, 0, 1000)
	for i := 1; i <= 1000; i++ {
		commands = append(commands, roleCommand(int64(i+10), actorKey('c'), domain.RoleFighter, 1, 50))
	}
	accepted, queue, _ := acceptRoles(t, state, RoleQueue{}, commands...)
	applied, _ := advanceWithRoles(t, accepted, queue, 50)
	if len(applied.Members) != 2 {
		t.Fatalf("members = %d", len(applied.Members))
	}
	for _, member := range applied.Members {
		if member.ActorKey == actorKey('c') && (member.LastChangeTick != nil || member.FirstSwitchUsed) {
			t.Fatal("duplicate messages created a role change")
		}
	}
}

func TestRoleMessageEventsUseCanonicalNames(t *testing.T) {
	if EventRoleAccepted != "role_accepted" || EventRoleChanged != "role_changed" ||
		EventRoleRejectedCooldown != "role_rejected_cooldown" || EventNextWaveQueued != "next_wave_queued" {
		t.Fatal("role event identifiers drifted from the canonical list")
	}
	if EventRolledOver != "wave_rollover" {
		t.Fatal("rollover event must use the canonical wave_rollover name")
	}
}

func TestRoleFuzzSequencesDoNotCorruptState(t *testing.T) {
	state, queue := startWithRole(t, 'a', domain.RoleBuilder)
	for step := 0; step < 60; step++ {
		before := state.Clone()
		queueBefore := queue.Clone()
		command := roleCommand(int64(step+10), actorKey(byte('a'+step%3)), domain.Role(1+step%3), 1, before.Tick()+1+step%5)
		next, nextQueue, _, err := AcceptRoleCommands(state, queue, []RoleCommand{command})
		if err != nil {
			if !reflect.DeepEqual(state, before) || !reflect.DeepEqual(queue, queueBefore) {
				t.Fatal("rejected acceptance mutated state or queue")
			}
			continue
		}
		state, queue = next, nextQueue
		if state.Tick() < 2999 {
			advanced, remaining, _, _, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
			if err != nil {
				t.Fatalf("advance: %v", err)
			}
			state, queue = advanced, remaining
		}
		if err := state.Validate(); err != nil {
			t.Fatalf("state became invalid: %v", err)
		}
	}
}

func BenchmarkRoleRegistration(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		b.Run("actors="+strconv.Itoa(size), func(b *testing.B) {
			initial, err := domain.NewInitialState(
				domain.VillageID("11111111-1111-4111-8111-111111111111"),
				domain.SessionID("22222222-2222-4222-8222-222222222222"),
				domain.SourceModeMock)
			if err != nil {
				b.Fatalf("initial state: %v", err)
			}
			state, _, err := Next(initial, Input{
				Kind:         KindStartWave,
				Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder},
			})
			if err != nil {
				b.Fatalf("start: %v", err)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				commands := make([]RoleCommand, 0, size)
				for n := 0; n < size; n++ {
					commands = append(commands, RoleCommand{
						InputSeq:      int64(n + 1),
						ActorKey:      actorKey(byte('a' + n%16)),
						Role:          domain.RoleBuilder,
						TargetWave:    1,
						ScheduledTick: 20,
					})
				}
				if _, _, _, err := AcceptRoleCommands(state, RoleQueue{}, commands); err != nil {
					b.Fatalf("accept: %v", err)
				}
			}
		})
	}
}
