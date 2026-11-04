package domain

import (
	"strings"
	"testing"
)

// validState builds a small, fully valid mid-wave state used as the base for the
// invariant table. Every case below mutates exactly one field.
func validState(t *testing.T) SessionState {
	t.Helper()
	state := initial(t)
	state.Wave.Number = 4
	state.Wave.Tick = 950
	state.Wave.Phase = PhasePrep
	state.Wave.Pools = []EnemyPool{
		{Lane: LaneWest, Archetype: ArchetypeRunner, HP: 10},
		{Lane: LaneCenter, Archetype: ArchetypeGrunt, HP: 20},
	}
	state.Wave.Vote = VoteState{}
	state.Wave.Ballots = []Ballot{{ActorKey: actorOne, Option: VoteOptionA, Tick: 900, InputSeq: 1}}
	state.Members = []WaveMember{
		{ActorKey: actorOne, Role: RoleBuilder, JoinedTick: 0, LastChangeTick: intPtr(10), FirstSwitchUsed: true},
		{ActorKey: actorTwo, Role: RoleGuard, JoinedTick: 5},
	}
	state.Pending = []PendingMember{{ActorKey: actorKeyFor(3), Role: RoleFighter, TargetWave: 5}}
	state.XP = []PlayerXP{{ActorKey: actorOne, XP: 10}, {ActorKey: actorTwo, XP: 20}}
	state.Difficulty = Difficulty{Level: 2, PopulationAnchor: 40}
	if err := state.Validate(); err != nil {
		t.Fatalf("base state must be valid: %v", err)
	}
	return state
}

// actorKeyFor returns a deterministic valid actor key for index i.
func actorKeyFor(i int) ActorKey {
	hexDigit := "0123456789abcdef"[i%16]
	raw := strings.Repeat(string(hexDigit), 64)
	return ActorKey(raw)
}

func assertInvalid(t *testing.T, state SessionState, want string) {
	t.Helper()
	err := state.Validate()
	if want == "" {
		// An empty expectation marks a case that must remain valid after the fix.
		if err != nil {
			t.Fatalf("state must be valid: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("state was accepted but %q must fail", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not mention %q", err.Error(), want)
	}
}

func TestValidationAcceptsValidState(t *testing.T) {
	state := validState(t)
	if !state.Valid() {
		t.Fatalf("valid state rejected: %v", state.Validate())
	}
}

func TestVillageInvariants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*SessionState)
		want   string
	}{
		{"level below one", func(s *SessionState) { s.Village.Level = 0 }, "village.level"},
		{"level above fifty", func(s *SessionState) { s.Village.Level = 51 }, "village.level"},
		{"wall tier zero", func(s *SessionState) { s.Village.WallTier = 0 }, "village.wall_tier"},
		{"wall tier above twenty", func(s *SessionState) { s.Village.WallTier = 21 }, "village.wall_tier"},
		{"wall hp above tier maximum", func(s *SessionState) {
			s.Village.WallTier = 1
			s.Village.WallHP = 1001
		}, "village.wall_hp"},
		{"negative wall hp", func(s *SessionState) { s.Village.WallHP = -1 }, "village.wall_hp"},
		{"townhall above maximum", func(s *SessionState) { s.Village.TownhallHP = 501 }, "village.townhall_hp"},
		{"negative townhall", func(s *SessionState) { s.Village.TownhallHP = -1 }, "village.townhall_hp"},
		{"negative wood", func(s *SessionState) { s.Village.Resources.Wood = -1 }, "resources.wood"},
		{"wood above cap", func(s *SessionState) { s.Village.Resources.Wood = ResourceCap + 1 }, "resources.wood"},
		{"stone above cap", func(s *SessionState) { s.Village.Resources.Stone = ResourceCap + 1 }, "resources.stone"},
		{"food above cap", func(s *SessionState) { s.Village.Resources.Food = ResourceCap + 1 }, "resources.food"},
		{"negative construction work", func(s *SessionState) { s.Village.ConstructionWork = -1 }, "village.construction_work"},
		{"negative total wins", func(s *SessionState) { s.Village.TotalWins = -1 }, "village.total_wins"},
		{"negative wall tier max check", func(s *SessionState) { s.Village.WallTier = -1 }, "village.wall_tier"},
	}
	for _, tc := range cases {
		state := validState(t)
		tc.mutate(&state)
		assertInvalid(t, state, tc.want)
	}
}

func TestWaveInvariants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*SessionState)
		want   string
	}{
		{"wave number zero", func(s *SessionState) { s.Wave.Number = 0 }, "wave.number"},
		{"tick above rollover", func(s *SessionState) { s.Wave.Tick = 3001 }, "wave.tick"},
		{"tick below waiting", func(s *SessionState) { s.Wave.Tick = -2 }, "wave.tick"},
		{"phase does not match tick", func(s *SessionState) { s.Wave.Phase = PhaseAssault }, "wave.phase"},
		{"rollover tick must be results", func(s *SessionState) {
			s.Wave.Tick = RolloverTick
			s.Wave.Phase = PhasePrep
		}, "wave.phase"},
		{"negative lane hp", func(s *SessionState) { s.Wave.Pools[0].HP = -1 }, "wave.pools[0].hp"},
		{"pool order violated", func(s *SessionState) {
			s.Wave.Pools[0], s.Wave.Pools[1] = s.Wave.Pools[1], s.Wave.Pools[0]
		}, "wave.pools[1]"},
		{"duplicate pool", func(s *SessionState) {
			s.Wave.Pools[1] = EnemyPool{Lane: LaneWest, Archetype: ArchetypeRunner, HP: 5}
		}, "wave.pools[1]"},
		{"boss archetype in ordinary pool", func(s *SessionState) {
			s.Wave.Pools[0].Archetype = ArchetypeBoss
		}, errOrdinaryPoolBoss},
		{"boss before spawn tick", func(s *SessionState) {
			s.Wave.Number = 5
			s.Wave.Ballots = nil
			s.Wave.Boss = BossPool{Exists: true, HP: 10, MaxHP: 10}
		}, errBossBeforeSpawn},
		{"boss on non boss wave", func(s *SessionState) {
			s.Wave.Tick = 2700
			s.Wave.Phase = PhaseFinale
			s.Wave.Boss = BossPool{Exists: true, HP: 10, MaxHP: 10}
		}, errBossWaveMismatch},
		{"boss hp exceeds maximum", func(s *SessionState) {
			s.Wave.Number = 5
			s.Wave.Ballots = nil
			s.Wave.Tick = 2700
			s.Wave.Phase = PhaseFinale
			s.Wave.Boss = BossPool{Exists: true, HP: 11, MaxHP: 10}
		}, errBossHP},
		{"boss killed immediately is valid", func(s *SessionState) {
			s.Wave.Number = 5
			s.Wave.Ballots = nil
			s.Wave.Tick = 2600
			s.Wave.Phase = PhaseFinale
			s.Pending = nil
			s.Wave.Boss = BossPool{Exists: true, HP: 0, MaxHP: 10}
		}, ""},
		{"boss killed at final combat is valid", func(s *SessionState) {
			s.Wave.Number = 5
			s.Wave.Ballots = nil
			s.Wave.Tick = 2800
			s.Wave.Phase = PhaseResults
			s.Pending = nil
			s.Wave.Boss = BossPool{Exists: true, HP: 0, MaxHP: 10}
		}, ""},
		{"negative boss hp", func(s *SessionState) {
			s.Wave.Number = 5
			s.Wave.Ballots = nil
			s.Wave.Tick = 2700
			s.Wave.Phase = PhaseFinale
			s.Pending = nil
			s.Wave.Boss = BossPool{Exists: true, HP: -1, MaxHP: 10}
		}, errBossHP},
		{"boss hp without existence", func(s *SessionState) {
			s.Wave.Boss = BossPool{Exists: false, HP: 5, MaxHP: 10}
		}, "must be zero"},
		{"defeat during prep", func(s *SessionState) { s.Wave.Defeated = true }, "wave.defeated"},
		{"defeat before combat starts", func(s *SessionState) {
			s.Wave.Tick = 1700
			s.Wave.Phase = PhaseAssault
			s.Wave.Defeated = true
		}, "wave.defeated"},
		{"defeat at first combat boundary is valid", func(s *SessionState) {
			s.Wave.Tick = 1800
			s.Wave.Phase = PhaseAssault
			s.Wave.Defeated = true
		}, ""},
		{"result before results phase", func(s *SessionState) { s.Wave.Result = WaveResultVictory }, errResultBeforeResults},
		{"loss without defeat is valid", func(s *SessionState) {
			s.Wave.Tick = 2900
			s.Wave.Phase = PhaseResults
			s.Wave.Defeated = false
			s.Wave.Result = WaveResultDefeat
		}, ""},
		{"loss after defeat is valid", func(s *SessionState) {
			s.Wave.Tick = 2900
			s.Wave.Phase = PhaseResults
			s.Wave.Defeated = true
			s.Wave.Result = WaveResultDefeat
		}, ""},
		{"victory with defeat flag", func(s *SessionState) {
			s.Wave.Tick = 2900
			s.Wave.Phase = PhaseResults
			s.Wave.Defeated = true
			s.Wave.Result = WaveResultVictory
		}, "cannot also be a victory"},
	}
	for _, tc := range cases {
		state := validState(t)
		tc.mutate(&state)
		assertInvalid(t, state, tc.want)
	}
}

func TestVoteAndBallotInvariants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*SessionState)
		want   string
	}{
		{"open vote with selection", func(s *SessionState) {
			s.Wave.Vote.Open = true
			s.Wave.Vote.Selected = VoteOptionA
		}, "an open vote cannot have a selected option"},
		{"invalid selection", func(s *SessionState) { s.Wave.Vote.Selected = VoteOption(4) }, "vote.selected"},
		{"negative vote count", func(s *SessionState) { s.Wave.Vote.Counts.C = -1 }, "vote.counts"},
		{"ballot option none", func(s *SessionState) { s.Wave.Ballots[0].Option = VoteOptionNone }, "ballots[0].option"},
		{"ballot in the future", func(s *SessionState) { s.Wave.Ballots[0].Tick = 200 }, "ballots[0].tick"},
		{"ballot negative tick", func(s *SessionState) { s.Wave.Ballots[0].Tick = -1 }, "ballots[0].tick"},
		{"duplicate ballot", func(s *SessionState) {
			s.Wave.Ballots = append(s.Wave.Ballots, s.Wave.Ballots[0])
		}, errDuplicateActor},
	}
	for _, tc := range cases {
		state := validState(t)
		tc.mutate(&state)
		assertInvalid(t, state, tc.want)
	}
}

func TestMembershipInvariants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*SessionState)
		want   string
	}{
		{"duplicate member", func(s *SessionState) {
			s.Members = append(s.Members, s.Members[1])
		}, errDuplicateActor},
		{"members not sorted", func(s *SessionState) {
			s.Members[0], s.Members[1] = s.Members[1], s.Members[0]
		}, errMemberOrder},
		{"invalid role", func(s *SessionState) { s.Members[0].Role = Role(0) }, errRoleInvalid},
		{"joined tick at wave end", func(s *SessionState) { s.Members[0].JoinedTick = WaveTicks }, errJoinTickRange},
		{"negative joined tick", func(s *SessionState) { s.Members[0].JoinedTick = -1 }, errJoinTickRange},
		{"change before join", func(s *SessionState) {
			s.Members[0].JoinedTick = 20
			s.Members[0].LastChangeTick = intPtr(10)
			s.Members[0].FirstSwitchUsed = true
		}, errChangeTickRange},
		{"change at wave end", func(s *SessionState) {
			s.Members[0].LastChangeTick = intPtr(WaveTicks)
			s.Members[0].FirstSwitchUsed = true
		}, errChangeTickRange},
		{"first switch used without a change", func(s *SessionState) {
			s.Members[1].FirstSwitchUsed = true
		}, "cannot be used before the first accepted role change"},
		{"change without first switch flag", func(s *SessionState) {
			s.Members[0].FirstSwitchUsed = false
		}, "must be true once a role change is recorded"},
		{"invalid actor key", func(s *SessionState) { s.Members[0].ActorKey = "not-a-key" }, errActorKeyFormat},
		{"pending target not future", func(s *SessionState) { s.Pending[0].TargetWave = 3 }, errTargetWaveRange},
		{"pending duplicate", func(s *SessionState) {
			s.Pending = append(s.Pending, s.Pending[0])
		}, errDuplicateActor},
		{"pending invalid role", func(s *SessionState) { s.Pending[0].Role = Role(9) }, errRoleInvalid},
		{"xp above cap", func(s *SessionState) { s.XP[0].XP = CapSessionXP + 1 }, errXPOutOfRange},
		{"negative xp", func(s *SessionState) { s.XP[0].XP = -1 }, errXPOutOfRange},
		{"xp duplicate", func(s *SessionState) { s.XP = append(s.XP, s.XP[1]) }, errDuplicateActor},
	}
	for _, tc := range cases {
		state := validState(t)
		tc.mutate(&state)
		assertInvalid(t, state, tc.want)
	}
}

func TestSessionIdentityAndLifecycleInvariants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*SessionState)
		want   string
	}{
		{"invalid session id", func(s *SessionState) { s.ID = "x" }, "session.id"},
		{"invalid village id", func(s *SessionState) { s.VillageID = "x" }, "session.village_id"},
		{"village id mismatch", func(s *SessionState) { s.Village.ID = VillageID("33333333-3333-4333-8333-333333333333") }, errVillageIDMismatch},
		{"invalid mode", func(s *SessionState) { s.Mode = SourceMode(7) }, errSourceModeInvalid},
		{"invalid lifecycle", func(s *SessionState) { s.Lifecycle = Lifecycle(9) }, errLifecycleInvalid},
		{"schema version drift", func(s *SessionState) { s.Revision.SchemaVersion = 2 }, errRevisionSchema},
		{"rules version drift", func(s *SessionState) { s.Revision.RulesVersion = "ttg-rules-v2" }, errRevisionRules},
		{"pause reason without lifecycle", func(s *SessionState) {
			s.PauseReasons = []PauseReason{PauseOperatorPaused}
			s.Lifecycle = LifecycleRunning
		}, "active pause reason requires"},
		{"paused without reason", func(s *SessionState) { s.Lifecycle = LifecyclePaused }, "needs at least one explicit reason"},
		{"invalid pause reason", func(s *SessionState) {
			s.Lifecycle = LifecyclePaused
			s.PauseReasons = []PauseReason{PauseReason(9)}
		}, errPauseReasonInvalid},
		{"duplicate pause reason", func(s *SessionState) {
			s.Lifecycle = LifecyclePaused
			s.PauseReasons = []PauseReason{PauseOperatorPaused, PauseOperatorPaused}
		}, "duplicate pause reason"},
		{"consecutive wins above cap", func(s *SessionState) { s.ConsecutiveWins = MaxConsecutiveWins + 1 }, errConsecutiveWins},
		{"consecutive losses above cap", func(s *SessionState) { s.ConsecutiveLosses = MaxConsecutiveLosses + 1 }, errConsecutiveLosses},
		{"difficulty level too high", func(s *SessionState) { s.Difficulty.Level = 11 }, "difficulty.level"},
		{"difficulty level zero", func(s *SessionState) { s.Difficulty.Level = 0 }, "difficulty.level"},
		{"negative population anchor", func(s *SessionState) { s.Difficulty.PopulationAnchor = -1 }, "population_anchor"},
		{"negative privacy epoch", func(s *SessionState) { s.PrivacyEpoch = -1 }, "privacy_epoch"},
		{"negative input sequence", func(s *SessionState) { s.InputSequence = -1 }, "input_sequence"},
	}
	for _, tc := range cases {
		state := validState(t)
		tc.mutate(&state)
		assertInvalid(t, state, tc.want)
	}
}

func TestValidationViaErrorsIs(t *testing.T) {
	state := validState(t)
	state.Village.Level = 0
	err := state.Validate()
	var validation *ValidationError
	if !errorsAs(err, &validation) {
		t.Fatalf("Validate() must return *ValidationError, got %T", err)
	}
	if len(validation.Problems) == 0 {
		t.Fatal("validation error must list the broken invariants")
	}
}

// errorsAs avoids importing errors just for one assertion in this file.
func errorsAs(err error, target **ValidationError) bool {
	value, ok := err.(*ValidationError)
	if !ok {
		return false
	}
	*target = value
	return true
}

func TestSortHelpersProduceCanonicalOrder(t *testing.T) {
	members := []WaveMember{
		{ActorKey: actorKeyFor(2), Role: RoleGuard, JoinedTick: 0},
		{ActorKey: actorKeyFor(1), Role: RoleBuilder, JoinedTick: 0},
		{ActorKey: actorKeyFor(3), Role: RoleFighter, JoinedTick: 0},
	}
	SortWaveMembers(members)
	for i := 1; i < len(members); i++ {
		if members[i-1].ActorKey > members[i].ActorKey {
			t.Fatal("members are not sorted by actor key")
		}
	}

	pools := []EnemyPool{
		{Lane: LaneEast, Archetype: ArchetypeRunner, HP: 1},
		{Lane: LaneWest, Archetype: ArchetypeBrute, HP: 1},
		{Lane: LaneWest, Archetype: ArchetypeRunner, HP: 1},
		{Lane: LaneCenter, Archetype: ArchetypeGrunt, HP: 1},
	}
	SortPools(pools)
	want := []EnemyPool{
		{Lane: LaneWest, Archetype: ArchetypeRunner, HP: 1},
		{Lane: LaneWest, Archetype: ArchetypeBrute, HP: 1},
		{Lane: LaneCenter, Archetype: ArchetypeGrunt, HP: 1},
		{Lane: LaneEast, Archetype: ArchetypeRunner, HP: 1},
	}
	for i := range want {
		if pools[i] != want[i] {
			t.Fatalf("pool %d = %+v, want %+v", i, pools[i], want[i])
		}
	}

	ballots := []Ballot{{ActorKey: actorKeyFor(2), Option: VoteOptionA, Tick: 1}, {ActorKey: actorKeyFor(1), Option: VoteOptionB, Tick: 1}}
	SortBallots(ballots)
	if ballots[0].ActorKey > ballots[1].ActorKey {
		t.Fatal("ballots are not sorted")
	}

	xp := []PlayerXP{{ActorKey: actorKeyFor(2), XP: 1}, {ActorKey: actorKeyFor(1), XP: 2}}
	SortPlayerXP(xp)
	if xp[0].ActorKey > xp[1].ActorKey {
		t.Fatal("xp rows are not sorted")
	}

	pending := []PendingMember{{ActorKey: actorKeyFor(2), Role: RoleGuard, TargetWave: 2}, {ActorKey: actorKeyFor(1), Role: RoleBuilder, TargetWave: 2}}
	SortPendingMembers(pending)
	if pending[0].ActorKey > pending[1].ActorKey {
		t.Fatal("pending members are not sorted")
	}
}

func TestVoteCountsTotal(t *testing.T) {
	counts := VoteCounts{A: 1, B: 2, C: 3}
	if counts.Total() != 6 {
		t.Fatalf("Total() = %d", counts.Total())
	}
	if (VoteCounts{}).Total() != 0 {
		t.Fatal("empty counts must total zero")
	}
}
