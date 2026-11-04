package domain

import (
	"errors"
	"testing"
)

func TestRulesIdentityIsPinned(t *testing.T) {
	if RulesVersion != "ttg-rules-v1" {
		t.Fatalf("RulesVersion = %q", RulesVersion)
	}
	if CanonicalSchemaVersion != 1 {
		t.Fatalf("CanonicalSchemaVersion = %d", CanonicalSchemaVersion)
	}
	if got := CurrentRevision(); !got.Valid() {
		t.Fatalf("current revision is not valid: %+v", got)
	}
}

func TestSimulationGeometryIsPinned(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"TicksPerSecond", TicksPerSecond, 10},
		{"WaveTicks", WaveTicks, 3000},
		{"WaitingTick", WaitingTick, -1},
		{"RolloverTick", RolloverTick, 3000},
		{"PrepEndTick", PrepEndTick, 1600},
		{"AssaultEndTick", AssaultEndTick, 2600},
		{"FinaleEndTick", FinaleEndTick, 2800},
		{"ResultsEndTick", ResultsEndTick, 3000},
		{"VoteCloseTick", VoteCloseTick, 1200},
		{"PopulationLockTick", PopulationLockTick, 1600},
		{"CombatStartTick", CombatStartTick, 1800},
		{"BossSpawnTick", BossSpawnTick, 2600},
		{"FinalCombatTick", FinalCombatTick, 2800},
		{"RoleChangeCooldownTick", RoleChangeCooldownTick, 600},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
		}
	}
	// One wave is 300 seconds at 10 Hz.
	if WaveTicks != 300*TicksPerSecond {
		t.Fatalf("wave geometry mismatch: %d ticks != 300s * %dHz", WaveTicks, TicksPerSecond)
	}
}

func TestBoundsArePinned(t *testing.T) {
	if MaxVillageLevel != 50 || MaxWallTier != 20 {
		t.Fatalf("level/tier bounds changed: %d/%d", MaxVillageLevel, MaxWallTier)
	}
	if BaseWallMax != 1000 || WallMaxStep != 200 || TownhallMax != 500 {
		t.Fatalf("wall/townhall constants changed: %d/%d/%d", BaseWallMax, WallMaxStep, TownhallMax)
	}
	if ResourceCap != 1_000_000 {
		t.Fatalf("ResourceCap = %d", ResourceCap)
	}
	if CapSessionXP != 300 || XPPerWave != 10 {
		t.Fatalf("XP constants changed: %d/%d", CapSessionXP, XPPerWave)
	}
	if MaxConsecutiveWins != 10 || MaxConsecutiveLosses != 3 {
		t.Fatalf("streak caps changed: %d/%d", MaxConsecutiveWins, MaxConsecutiveLosses)
	}
}

func TestWallMax(t *testing.T) {
	cases := []struct {
		tier    int
		want    int64
		wantErr bool
	}{
		{1, 1000, false},
		{2, 1200, false},
		{10, 2800, false},
		{20, 4800, false},
		{0, 0, true},
		{21, 0, true},
		{-1, 0, true},
	}
	for _, tc := range cases {
		got, err := WallMax(tc.tier)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("WallMax(%d) accepted an invalid tier", tc.tier)
			}
			var field *FieldError
			if !errors.As(err, &field) || field.Field != "wall_tier" {
				t.Fatalf("WallMax(%d) error = %v, want wall_tier FieldError", tc.tier, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("WallMax(%d): %v", tc.tier, err)
		}
		if got != tc.want {
			t.Errorf("WallMax(%d) = %d, want %d", tc.tier, got, tc.want)
		}
	}
}

func TestBossWaveCadence(t *testing.T) {
	for wave := int64(1); wave <= 20; wave++ {
		want := wave%5 == 0
		if got := IsBossWave(wave); got != want {
			t.Errorf("IsBossWave(%d) = %v, want %v", wave, got, want)
		}
	}
	for _, invalid := range []int64{0, -5} {
		if IsBossWave(invalid) {
			t.Errorf("IsBossWave(%d) must be false", invalid)
		}
	}
}

func TestRoles(t *testing.T) {
	cases := []struct {
		role Role
		name string
	}{
		{RoleBuilder, "builder"},
		{RoleGuard, "guard"},
		{RoleFighter, "fighter"},
	}
	for _, tc := range cases {
		if !tc.role.Valid() {
			t.Errorf("%v must be valid", tc.role)
		}
		if tc.role.String() != tc.name {
			t.Errorf("role %d String() = %q", tc.role, tc.role.String())
		}
		parsed, err := ParseRole(tc.name)
		if err != nil || parsed != tc.role {
			t.Errorf("ParseRole(%q) = %v, %v", tc.name, parsed, err)
		}
	}
	if Role(0).Valid() || Role(4).Valid() || Role(0).String() != "invalid" {
		t.Fatal("undefined roles must be rejected")
	}
	for _, bad := range []string{"", "Builder", "admin", "builder ", "xay"} {
		if _, err := ParseRole(bad); err == nil {
			t.Errorf("ParseRole(%q) accepted a non-canonical role", bad)
		}
	}
}

func TestLifecycle(t *testing.T) {
	values := []struct {
		lifecycle Lifecycle
		name      string
	}{
		{LifecycleWaiting, "waiting"},
		{LifecycleRunning, "running"},
		{LifecyclePaused, "paused"},
		{LifecycleSourceDisconnected, "source_disconnected"},
		{LifecycleStorageUnavailable, "storage_unavailable"},
		{LifecycleStopped, "stopped"},
	}
	for _, tc := range values {
		if !tc.lifecycle.Valid() || tc.lifecycle.String() != tc.name {
			t.Errorf("lifecycle %d = %q valid=%v", tc.lifecycle, tc.lifecycle.String(), tc.lifecycle.Valid())
		}
	}
	if Lifecycle(0).Valid() || Lifecycle(7).Valid() {
		t.Fatal("undefined lifecycle values must be rejected")
	}
}

func TestPhasesAndPauseReasons(t *testing.T) {
	phases := []struct {
		phase Phase
		name  string
	}{
		{PhaseWaiting, "waiting"},
		{PhasePrep, "prep"},
		{PhaseAssault, "assault"},
		{PhaseFinale, "finale"},
		{PhaseResults, "results"},
	}
	for _, tc := range phases {
		if !tc.phase.Valid() || tc.phase.String() != tc.name {
			t.Errorf("phase %d = %q", tc.phase, tc.phase.String())
		}
	}
	reasons := []struct {
		reason PauseReason
		name   string
	}{
		{PauseOperatorPaused, "operator_paused"},
		{PauseSourceDisconnected, "source_disconnected"},
		{PauseStorageUnavailable, "storage_unavailable"},
	}
	for _, tc := range reasons {
		if !tc.reason.Valid() || tc.reason.String() != tc.name {
			t.Errorf("pause reason %d = %q", tc.reason, tc.reason.String())
		}
	}
	if Phase(0).Valid() || PauseReason(0).Valid() || PauseReason(4).Valid() {
		t.Fatal("undefined phase/pause values must be rejected")
	}
}

func TestWaveResultAndSourceMode(t *testing.T) {
	results := map[WaveResult]string{WaveResultPending: "pending", WaveResultVictory: "victory", WaveResultDefeat: "defeat"}
	for value, name := range results {
		if !value.Valid() || value.String() != name {
			t.Errorf("wave result %d = %q", value, value.String())
		}
	}
	if WaveResult(3).Valid() {
		t.Fatal("undefined wave result must be rejected")
	}
	modes := map[SourceMode]string{SourceModeMock: "mock", SourceModeReplay: "replay", SourceModeAuthorizedLive: "authorized_live"}
	for value, name := range modes {
		if !value.Valid() || value.String() != name {
			t.Errorf("source mode %d = %q", value, value.String())
		}
	}
	if SourceMode(0).Valid() || SourceMode(4).Valid() {
		t.Fatal("undefined source modes must be rejected")
	}
}

func TestLanesArchetypesAndVoteOptions(t *testing.T) {
	lanes := map[Lane]string{LaneWest: "west", LaneCenter: "center", LaneEast: "east"}
	for value, name := range lanes {
		if !value.Valid() || value.String() != name {
			t.Errorf("lane %d = %q", value, value.String())
		}
	}
	if Lane(3).Valid() || Lane(3).String() != "invalid" {
		t.Fatal("undefined lane must be rejected")
	}
	archetypes := map[Archetype]string{
		ArchetypeRunner: "runner", ArchetypeGrunt: "grunt", ArchetypeBrute: "brute", ArchetypeBoss: "boss",
	}
	for value, name := range archetypes {
		if !value.Valid() || value.String() != name {
			t.Errorf("archetype %d = %q", value, value.String())
		}
	}
	if Archetype(4).Valid() {
		t.Fatal("undefined archetype must be rejected")
	}
	options := map[VoteOption]string{
		VoteOptionNone: "", VoteOptionA: "A", VoteOptionB: "B", VoteOptionC: "C",
	}
	for value, name := range options {
		if !value.Valid() || value.String() != name {
			t.Errorf("vote option %d = %q", value, value.String())
		}
	}
	if VoteOption(4).Valid() {
		t.Fatal("undefined vote option must be rejected")
	}
}
