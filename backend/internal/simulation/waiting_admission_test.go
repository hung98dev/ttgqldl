package simulation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// waitingStartCommand builds the only legal wave-start registration shape:
// accepted at the -1 sentinel, effective at tick 0 of the current wave.
func waitingStartCommand(seq int64, actor domain.ActorKey, role domain.Role) RoleCommand {
	return RoleCommand{
		InputSeq: seq, ActorKey: actor, Role: role,
		TargetWave: 1, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
	}
}

// acceptWaitingRegistration accepts one command against a WAITING state.
func acceptWaitingRegistration(t *testing.T, state domain.SessionState, queue RoleQueue, command RoleCommand) (domain.SessionState, RoleQueue, error) {
	t.Helper()
	next, nextQueue, _, err := AcceptRoleCommands(state, queue, []RoleCommand{command})
	return next, nextQueue, err
}

// TestWaitingAdmissionSingleStart is the admission contract for a WAITING session:
// exactly one accepted wave-start registration may exist, a second arrival is
// rejected with a typed error, duplicated input sequences are rejected, and an
// invalid batch is atomic. It reproduces the pre-fix behaviour where every separate
// call re-armed the batch-local `first` flag.
func TestWaitingAdmissionSingleStart(t *testing.T) {
	t.Run("A one legitimate registration starts the wave", func(t *testing.T) {
		state := newState(t)
		if state.Lifecycle != domain.LifecycleWaiting || state.Tick() != domain.WaitingTick {
			t.Fatalf("fixture is not waiting: lifecycle %s tick %d", state.Lifecycle, state.Tick())
		}
		next, queue, err := acceptWaitingRegistration(t, state, RoleQueue{}, waitingStartCommand(1, actorKey('a'), domain.RoleBuilder))
		if err != nil {
			t.Fatalf("first waiting registration rejected: %v", err)
		}
		if next.Lifecycle != domain.LifecycleWaiting || next.Tick() != domain.WaitingTick {
			t.Fatalf("accepting the start registration changed the lifecycle: %s/%d", next.Lifecycle, next.Tick())
		}
		if queue.Len() != 1 {
			t.Fatalf("waiting queue = %d entries, want exactly 1", queue.Len())
		}
		if !reflect.DeepEqual(next, state) {
			t.Fatal("accepting a waiting registration mutated authoritative state")
		}

		started, remaining, events, roleEvents, err := AdvanceWithRoles(next, queue, Input{
			Kind:         KindStartWave,
			Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder},
		})
		if err != nil {
			t.Fatalf("start wave: %v", err)
		}
		if started.Tick() != 0 || started.Lifecycle != domain.LifecycleRunning {
			t.Fatalf("started state = %s/%d", started.Lifecycle, started.Tick())
		}
		if len(started.Members) != 1 || started.Members[0].ActorKey != actorKey('a') {
			t.Fatalf("started members = %+v", started.Members)
		}
		if remaining.Len() != 0 {
			t.Fatalf("start left %d queued commands", remaining.Len())
		}
		if countEventType(events, EventType(EventRoleAccepted)) != 1 {
			t.Fatalf("role_accepted events at start = %d, want exactly 1", countEventType(events, EventType(EventRoleAccepted)))
		}
		if len(roleEvents) != 1 || roleEvents[0].Type != EventRoleAccepted {
			t.Fatalf("role events = %+v", roleEvents)
		}
		if len(started.XP) != 0 {
			t.Fatalf("tick-0 start awarded XP: %+v", started.XP)
		}
	})

	t.Run("B second separate admission is rejected", func(t *testing.T) {
		state, queue := acceptedWaitingFixture(t)
		next, nextQueue, err := acceptWaitingRegistration(t, state, queue, waitingStartCommand(2, actorKey('b'), domain.RoleFighter))
		if err == nil {
			t.Fatalf("second WAITING admission was accepted: queue=%d entries %+v", nextQueue.Len(), nextQueue.Entries)
		}
		if !errors.Is(err, ErrStartPending) {
			t.Fatalf("second WAITING admission error = %v, want ErrStartPending", err)
		}
		assertWaitingUnchanged(t, state, queue, next, nextQueue)
	})

	t.Run("C reused input sequence is rejected", func(t *testing.T) {
		state, queue := acceptedWaitingFixture(t)
		next, nextQueue, err := acceptWaitingRegistration(t, state, queue, waitingStartCommand(1, actorKey('b'), domain.RoleFighter))
		if !errors.Is(err, ErrDuplicateInputSeq) {
			t.Fatalf("reused input sequence error = %v, want ErrDuplicateInputSeq", err)
		}
		assertWaitingUnchanged(t, state, queue, next, nextQueue)
	})

	t.Run("D same actor conflicting role is rejected", func(t *testing.T) {
		state, queue := acceptedWaitingFixture(t)
		next, nextQueue, err := acceptWaitingRegistration(t, state, queue, waitingStartCommand(2, actorKey('a'), domain.RoleGuard))
		if err == nil {
			t.Fatalf("conflicting duplicate start for the same actor was accepted: %+v", nextQueue.Entries)
		}
		if !errors.Is(err, ErrStartPending) {
			t.Fatalf("conflicting duplicate start error = %v, want ErrStartPending", err)
		}
		assertWaitingUnchanged(t, state, queue, next, nextQueue)
	})

	t.Run("E multiple registrations in one batch are rejected atomically", func(t *testing.T) {
		state := newState(t)
		batch := []RoleCommand{
			waitingStartCommand(1, actorKey('a'), domain.RoleBuilder),
			waitingStartCommand(2, actorKey('b'), domain.RoleFighter),
		}
		next, nextQueue, decisions, err := AcceptRoleCommands(state, RoleQueue{}, batch)
		if err == nil {
			t.Fatalf("multi-start batch was accepted: %+v", nextQueue.Entries)
		}
		if len(decisions) != 0 {
			t.Fatalf("rejected batch produced %d decisions", len(decisions))
		}
		if !reflect.DeepEqual(next, state) || nextQueue.Len() != 0 {
			t.Fatal("rejected multi-start batch mutated state or queue")
		}
	})

	t.Run("F start registration mismatch keeps ErrStartMismatch", func(t *testing.T) {
		state, queue := acceptedWaitingFixture(t)
		_, _, _, _, err := AdvanceWithRoles(state, queue, Input{
			Kind:         KindStartWave,
			Registration: &Registration{ActorKey: actorKey('b'), Role: domain.RoleFighter},
		})
		if !errors.Is(err, ErrStartMismatch) {
			t.Fatalf("start mismatch error = %v, want ErrStartMismatch", err)
		}
	})
}

// acceptedWaitingFixture returns a WAITING state with exactly one accepted start
// registration for actor 'a'.
func acceptedWaitingFixture(t *testing.T) (domain.SessionState, RoleQueue) {
	t.Helper()
	state := newState(t)
	next, queue, err := acceptWaitingRegistration(t, state, RoleQueue{}, waitingStartCommand(1, actorKey('a'), domain.RoleBuilder))
	if err != nil {
		t.Fatalf("fixture start registration: %v", err)
	}
	if queue.Len() != 1 {
		t.Fatalf("fixture queue = %d entries", queue.Len())
	}
	return next, queue
}

func assertWaitingUnchanged(t *testing.T, state domain.SessionState, queue RoleQueue, next domain.SessionState, nextQueue RoleQueue) {
	t.Helper()
	if !reflect.DeepEqual(next, state) {
		t.Fatal("rejected admission mutated authoritative state")
	}
	if !reflect.DeepEqual(nextQueue, queue) {
		t.Fatal("rejected admission mutated the accepted queue")
	}
}
