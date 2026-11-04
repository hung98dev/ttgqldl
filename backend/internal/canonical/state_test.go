package canonical

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

const (
	villageID = domain.VillageID("11111111-1111-4111-8111-111111111111")
	sessionID = domain.SessionID("22222222-2222-4222-8222-222222222222")
)

func actor(index int) domain.ActorKey {
	digit := "0123456789abcdef"[index%16]
	return domain.ActorKey(strings.Repeat(string(digit), 64))
}

func baseState(t *testing.T) domain.SessionState {
	t.Helper()
	state, err := domain.NewInitialState(villageID, sessionID, domain.SourceModeMock)
	if err != nil {
		t.Fatal(err)
	}
	state.Members = []domain.WaveMember{{ActorKey: actor(1), Role: domain.RoleBuilder, JoinedTick: 0}}
	state.XP = []domain.PlayerXP{{ActorKey: actor(1), XP: 10}}
	return state
}

func TestJCSKnownVectors(t *testing.T) {
	cases := []struct {
		value any
		want  string
	}{
		{map[string]any{"b": int64(1), "a": "x"}, `{"a":"x","b":1}`},
		{map[string]any{"z": nil, "a": true}, `{"a":true,"z":null}`},
		{[]any{int64(1), int64(2), int64(3)}, `[1,2,3]`},
		{map[string]any{"n": maxSafeInteger}, `{"n":9007199254740991}`},
		{map[string]any{"q": "line\nquote\""}, "{\"q\":\"line\\nquote\\\"\"}"},
		{map[string]any{"nested": map[string]any{"b": false, "a": int64(0)}}, `{"nested":{"a":0,"b":false}}`},
	}
	for _, tc := range cases {
		got, err := MarshalJCS(tc.value)
		if err != nil {
			t.Fatalf("%v: %v", tc.value, err)
		}
		if string(got) != tc.want {
			t.Fatalf("JCS(%v) = %s, want %s", tc.value, got, tc.want)
		}
	}
	if _, err := MarshalJCS(3.14); err == nil {
		t.Fatal("floats must be rejected")
	}
	if _, err := MarshalJCS(map[string]any{"n": int64(9223372036854775807)}); err == nil {
		t.Fatal("integers outside the JSON safe range must be rejected")
	}
}

func TestHashIsDeterministicAndSensitive(t *testing.T) {
	state := baseState(t)
	first, err := Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Hash(state.Clone())
	if err != nil {
		t.Fatal(err)
	}
	if first != second || len(first) != 64 {
		t.Fatalf("hashes differ: %s vs %s", first, second)
	}

	mutations := map[string]func(*domain.SessionState){
		"wood":       func(s *domain.SessionState) { s.Village.Resources.Wood++ },
		"wall":       func(s *domain.SessionState) { s.Village.WallHP-- },
		"role":       func(s *domain.SessionState) { s.Members[0].Role = domain.RoleGuard },
		"xp_awarded": func(s *domain.SessionState) { s.Members[0].XPAwarded = true },
		"session_xp": func(s *domain.SessionState) { s.XP[0].XP++ },
		"privacy":    func(s *domain.SessionState) { s.PrivacyEpoch++ },
		"last_change": func(s *domain.SessionState) {
			tick := 10
			s.Members[0].LastChangeTick = &tick
			s.Members[0].FirstSwitchUsed = true
		},
	}
	for name, mutate := range mutations {
		changed := state.Clone()
		mutate(&changed)
		hash, err := Hash(changed)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if hash == first {
			t.Fatalf("%s mutation did not change the hash", name)
		}
	}
}

func TestProjectExposesNullableFieldsExplicitly(t *testing.T) {
	state := baseState(t)
	projection, err := Project(state)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := MarshalJCS(projection)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(`"last_change_tick":null`)) {
		t.Fatal("nullable last_change_tick is not explicit")
	}
	if !bytes.Contains(encoded, []byte(`"rules_version":"ttg-rules-v1"`)) {
		t.Fatal("rules version missing")
	}
}

func TestLanePoolsUseTheBindingThreeLaneShape(t *testing.T) {
	state := baseState(t)
	state.Wave.Pools = []domain.EnemyPool{}
	for lane := domain.LaneWest; lane <= domain.LaneEast; lane++ {
		state.Wave.Pools = append(state.Wave.Pools,
			domain.EnemyPool{Lane: lane, Archetype: domain.ArchetypeRunner, HP: int64(lane) + 1},
			domain.EnemyPool{Lane: lane, Archetype: domain.ArchetypeGrunt, HP: int64(lane) + 2},
			domain.EnemyPool{Lane: lane, Archetype: domain.ArchetypeBrute, HP: int64(lane) + 3},
		)
	}
	projection, err := Project(state)
	if err != nil {
		t.Fatal(err)
	}
	lanes, ok := projection["lane_pools"].([]any)
	if !ok || len(lanes) != domain.LaneCount {
		t.Fatalf("lane_pools = %v", projection["lane_pools"])
	}
	first, ok := lanes[0].(map[string]any)
	if !ok || first["lane"] != "west" || first["runner_hp"] != int64(1) || first["grunt_hp"] != int64(2) || first["brute_hp"] != int64(3) {
		t.Fatalf("west lane = %v", lanes[0])
	}
}

func TestAnchorPopulationUsesTheLockedAnchor(t *testing.T) {
	state := baseState(t)
	state.Difficulty = domain.Difficulty{Level: 1, PopulationAnchor: 10_000, Locked: true, ActualPopulation: 12_000, RecoveryPercent: 100, BaseHP: 3920, MilitiaAttack: 0}
	state.Members = nil
	for index := 1; index <= 3; index++ {
		state.Members = append(state.Members, domain.WaveMember{ActorKey: actor(index), Role: domain.RoleBuilder, JoinedTick: 0})
	}
	state.XP = []domain.PlayerXP{}
	state.Wave.Pools = []domain.EnemyPool{}
	for lane := domain.LaneWest; lane <= domain.LaneEast; lane++ {
		for archetype := domain.ArchetypeRunner; archetype <= domain.ArchetypeBrute; archetype++ {
			state.Wave.Pools = append(state.Wave.Pools, domain.EnemyPool{Lane: lane, Archetype: archetype, HP: 1})
		}
	}
	state.Wave.Number = 2
	projection, err := Project(state)
	if err != nil {
		t.Fatal(err)
	}
	if projection["anchor_population"] != int64(10_000) {
		t.Fatalf("anchor_population = %v, want the locked cap 10000", projection["anchor_population"])
	}
}

// TestPersistedXPOutsideCurrentMembersIsNotHashedInV1 documents the historical
// schema-v1 contract gap. It is retained deliberately: v1 fixtures and checkpoints
// must keep verifying with their original hashes. The v2 fix is covered by
// TestRetainedXPIsCoveredByCanonicalV2.
func TestPersistedXPOutsideCurrentMembersIsNotHashedInV1(t *testing.T) {
	state := v1Revision(baseState(t))
	base, err := Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	state.XP = append(state.XP, domain.PlayerXP{ActorKey: actor(2), XP: 50})
	changed, err := Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	if changed != base {
		t.Fatal("the historical v1 contract must not hash non-member XP; update the compatibility record")
	}
}

func TestProjectRejectsUnsortedAndInvalidStates(t *testing.T) {
	state := baseState(t)
	state.Members = []domain.WaveMember{
		{ActorKey: actor(2), Role: domain.RoleBuilder, JoinedTick: 0},
		{ActorKey: actor(1), Role: domain.RoleBuilder, JoinedTick: 0},
	}
	if _, err := Project(state); err == nil {
		t.Fatal("unsorted members must be rejected")
	}

	bad := baseState(t)
	bad.Revision.RulesVersion = "ttg-rules-v2"
	if _, err := Project(bad); err == nil {
		t.Fatal("foreign rules version must be rejected")
	}

	badSchema := baseState(t)
	badSchema.Revision.SchemaVersion = 3
	if _, err := Project(badSchema); err == nil {
		t.Fatal("unknown schema version must be rejected")
	}
}
