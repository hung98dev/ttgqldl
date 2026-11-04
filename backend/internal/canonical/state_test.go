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
		{map[string]any{"n": int64(9223372036854775807)}, `{"n":9223372036854775807}`},
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
	badSchema.Revision.SchemaVersion = 2
	if _, err := Project(badSchema); err == nil {
		t.Fatal("foreign schema version must be rejected")
	}
}
