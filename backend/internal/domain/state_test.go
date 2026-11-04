package domain

import (
	"errors"
	"reflect"
	"testing"
)

const (
	villageID = VillageID("11111111-1111-4111-8111-111111111111")
	sessionID = SessionID("22222222-2222-4222-8222-222222222222")
	actorOne  = ActorKey("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	actorTwo  = ActorKey("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
)

func initial(t *testing.T) SessionState {
	t.Helper()
	state, err := NewInitialState(villageID, sessionID, SourceModeMock)
	if err != nil {
		t.Fatalf("NewInitialState: %v", err)
	}
	return state
}

func TestInitialStateValues(t *testing.T) {
	state := initial(t)

	if state.Revision != CurrentRevision() {
		t.Fatalf("revision = %+v", state.Revision)
	}
	if state.ID != sessionID || state.VillageID != villageID || state.Village.ID != villageID {
		t.Fatalf("identity mismatch: %+v", state)
	}
	if state.Mode != SourceModeMock {
		t.Fatalf("mode = %s", state.Mode)
	}
	if state.Lifecycle != LifecycleWaiting {
		t.Fatalf("lifecycle = %s", state.Lifecycle)
	}
	if state.Wave.Number != InitialWaveNo || state.Wave.Tick != WaitingTick || state.Wave.Phase != PhaseWaiting {
		t.Fatalf("wave = %+v", state.Wave)
	}

	village := state.Village
	if village.Level != 1 || village.WallTier != 1 {
		t.Fatalf("level/tier = %d/%d", village.Level, village.WallTier)
	}
	if village.WallHP != 1000 {
		t.Fatalf("wall hp = %d", village.WallHP)
	}
	if village.TownhallHP != 500 {
		t.Fatalf("townhall hp = %d", village.TownhallHP)
	}
	if village.Resources.Wood != 400 || village.Resources.Stone != 250 || village.Resources.Food != 300 {
		t.Fatalf("resources = %+v", village.Resources)
	}
	if village.ConstructionWork != 0 || village.TotalWins != 0 {
		t.Fatalf("work/wins = %d/%d", village.ConstructionWork, village.TotalWins)
	}
	if state.ConsecutiveWins != 0 || state.ConsecutiveLosses != 0 {
		t.Fatalf("streaks = %d/%d", state.ConsecutiveWins, state.ConsecutiveLosses)
	}
	if state.Difficulty.Level != 1 || state.Difficulty.PopulationAnchor != 0 {
		t.Fatalf("difficulty = %+v", state.Difficulty)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("initial state must be valid: %v", err)
	}
}

func TestInitialStateHasNoFabricatedParticipants(t *testing.T) {
	state := initial(t)
	if len(state.Members) != 0 || len(state.Pending) != 0 || len(state.XP) != 0 {
		t.Fatalf("initial participants must be empty: %d/%d/%d", len(state.Members), len(state.Pending), len(state.XP))
	}
	if len(state.Wave.Pools) != 0 || len(state.Wave.Ballots) != 0 {
		t.Fatalf("initial wave collections must be empty")
	}
	if state.Wave.Boss.Exists || state.Wave.Vote.Open {
		t.Fatalf("no boss or vote may exist initially")
	}
	if state.Wave.Result != WaveResultPending {
		t.Fatalf("wave result = %s", state.Wave.Result)
	}
	// Empty collections must be non-nil for stable canonical serialization.
	if state.Members == nil || state.Pending == nil || state.XP == nil ||
		state.Wave.Pools == nil || state.Wave.Ballots == nil || state.PauseReasons == nil {
		t.Fatal("initial collections must be non-nil")
	}
}

func TestInitialStateIsDeterministic(t *testing.T) {
	first := initial(t)
	second := initial(t)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("two constructions from identical inputs differ")
	}
	if first.Describe() != second.Describe() {
		t.Fatal("diagnostics differ between identical constructions")
	}
}

func TestInitialStateRejectsInvalidIdentifiers(t *testing.T) {
	cases := []struct {
		name      string
		village   VillageID
		session   SessionID
		mode      SourceMode
		wantField string
	}{
		{"empty village", "", sessionID, SourceModeMock, "village_id"},
		{"uppercase village", "11111111-1111-4111-8111-11111111111A", sessionID, SourceModeMock, "village_id"},
		{"short village", "11111111-1111-4111-8111", sessionID, SourceModeMock, "village_id"},
		{"empty session", villageID, "", SourceModeMock, "session_id"},
		{"not a uuid", villageID, "session-1", SourceModeMock, "session_id"},
		{"invalid mode", villageID, sessionID, SourceMode(0), "mode"},
		{"unknown mode", villageID, sessionID, SourceMode(9), "mode"},
	}
	for _, tc := range cases {
		_, err := NewInitialState(tc.village, tc.session, tc.mode)
		if err == nil {
			t.Fatalf("%s: invalid input was accepted", tc.name)
		}
		var field *FieldError
		if !errors.As(err, &field) || field.Field != tc.wantField {
			t.Fatalf("%s: error = %v, want field %s", tc.name, err, tc.wantField)
		}
	}
}

func TestCloneIsDeep(t *testing.T) {
	state := initial(t)
	state.PauseReasons = append(state.PauseReasons, PauseOperatorPaused)
	state.Lifecycle = LifecyclePaused
	state.Members = append(state.Members, WaveMember{ActorKey: actorOne, Role: RoleBuilder, JoinedTick: 0, LastChangeTick: -1})
	state.Pending = append(state.Pending, PendingMember{ActorKey: actorTwo, Role: RoleGuard, TargetWave: 2})
	state.XP = append(state.XP, PlayerXP{ActorKey: actorOne, XP: 10})
	state.Wave.Pools = append(state.Wave.Pools, EnemyPool{Lane: LaneWest, Archetype: ArchetypeRunner, HP: 10})
	state.Wave.Ballots = append(state.Wave.Ballots, Ballot{ActorKey: actorOne, Option: VoteOptionA, Tick: 5})
	state.Wave.Tick = 5
	state.Wave.Phase = PhasePrep

	clone := state.Clone()

	clone.PauseReasons[0] = PauseStorageUnavailable
	clone.Members[0].Role = RoleFighter
	clone.Pending[0].TargetWave = 99
	clone.XP[0].XP = 300
	clone.Wave.Pools[0].HP = 999
	clone.Wave.Ballots[0].Option = VoteOptionC

	if state.PauseReasons[0] != PauseOperatorPaused {
		t.Fatal("pause reasons are shared between the clone and the original")
	}
	if state.Members[0].Role != RoleBuilder {
		t.Fatal("members are shared between the clone and the original")
	}
	if state.Pending[0].TargetWave != 2 {
		t.Fatal("pending members are shared")
	}
	if state.XP[0].XP != 10 {
		t.Fatal("XP rows are shared")
	}
	if state.Wave.Pools[0].HP != 10 {
		t.Fatal("pools are shared")
	}
	if state.Wave.Ballots[0].Option != VoteOptionA {
		t.Fatal("ballots are shared")
	}
}

func TestDescribe(t *testing.T) {
	state := initial(t)
	got := state.Describe()
	want := "session=22222222-2222-4222-8222-222222222222 village=11111111-1111-4111-8111-111111111111 wave=1 tick=-1 lifecycle=waiting phase=waiting"
	if got != want {
		t.Fatalf("Describe() = %q", got)
	}
}
