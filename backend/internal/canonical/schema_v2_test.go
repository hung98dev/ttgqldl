package canonical

import (
	"sort"
	"strings"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// Frozen schema-v1 values captured from the accepted implementation at
// 429fa9399d9f5eb805327798c2f4cfbb9791c6fa before schema v2 was introduced.
// They are historical compatibility fixtures: they must never be regenerated from
// the v2 implementation.
const (
	historicalV1BaseHash = "438cc8924df9e067bd75ef1cb02e96ec5e9a26d6cb6bbf5826fe174a328a2357"
	historicalV1RichHash = "b0a5bad26b0548810b003359e774462c3ddddc715e363760ad8bd56e2196b86c"

	historicalV1BaseJSON = `{"anchor_population":0,"base_hp":0,"boss_hp":0,"boss_spawned":false,"consecutive_losses":0,"consecutive_wins":0,"construction_work":0,"defeated":false,"difficulty":1,"food":300,"lane_pools":[{"brute_hp":0,"grunt_hp":0,"lane":"west","runner_hp":0},{"brute_hp":0,"grunt_hp":0,"lane":"center","runner_hp":0},{"brute_hp":0,"grunt_hp":0,"lane":"east","runner_hp":0}],"lifecycle":"waiting","members":[{"actor_key":"1111111111111111111111111111111111111111111111111111111111111111","first_switch_used":false,"joined_tick":0,"last_change_tick":null,"role":"builder","session_xp":10,"xp_awarded":false}],"privacy_epoch":0,"queued_next_wave":[],"rules_version":"ttg-rules-v1","schema_version":1,"session_id":"22222222-2222-4222-8222-222222222222","stone":250,"total_wins":0,"townhall_hp":500,"village_id":"11111111-1111-4111-8111-111111111111","vote":{"closed":false,"counts":{"a":0,"b":0,"c":0},"open":false,"selected":null},"wall_hp":1000,"wall_tier":1,"wave_no":1,"wave_tick":-1,"wood":400}`
)

// v1Revision returns an explicit historical schema-v1 revision of the same state.
// The revision tag, not a rewrite of the field set, selects the projection version.
func v1Revision(state domain.SessionState) domain.SessionState {
	clone := state.Clone()
	clone.Revision = domain.Revision{
		SchemaVersion: domain.LegacyCanonicalSchemaVersion,
		RulesVersion:  domain.RulesVersion,
	}
	return clone
}

// richFixture is the second frozen v1 fixture: two members with role-change
// bookkeeping and two persisted XP rows.
func richFixture(t *testing.T) domain.SessionState {
	t.Helper()
	state := baseState(t)
	changeTick := 100
	state.Members = append(state.Members, domain.WaveMember{
		ActorKey: actor(2), Role: domain.RoleFighter, JoinedTick: 1,
		LastChangeTick: &changeTick, FirstSwitchUsed: true, XPAwarded: true,
	})
	state.XP = append(state.XP, domain.PlayerXP{ActorKey: actor(2), XP: 300})
	return state
}

// TestCanonicalV1HistoricalContract freezes the exact v1 JSON and SHA-256 of two
// accepted fixtures. The active schema is v2; this test is the proof that the
// historical serialization did not drift and that v1 fixtures remain verifiable
// byte-for-byte.
func TestCanonicalV1HistoricalContract(t *testing.T) {
	legacy := v1Revision(baseState(t))
	projection, err := Project(legacy)
	if err != nil {
		t.Fatalf("historical v1 projection rejected: %v", err)
	}
	encoded, err := MarshalJCS(projection)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != historicalV1BaseJSON {
		t.Fatalf("v1 canonical JSON drifted:\n got: %s\nwant: %s", encoded, historicalV1BaseJSON)
	}
	hash, err := Hash(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if hash != historicalV1BaseHash {
		t.Fatalf("v1 base hash = %s, want historical %s", hash, historicalV1BaseHash)
	}

	richHash, err := Hash(v1Revision(richFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	if richHash != historicalV1RichHash {
		t.Fatalf("v1 rich hash = %s, want historical %s", richHash, historicalV1RichHash)
	}
}

// TestCanonicalV2ExtendsV1WithoutReinterpretingIt proves the exact schema diff:
// v2 carries every v1 key unchanged except schema_version, plus session_progress.
func TestCanonicalV2ExtendsV1WithoutReinterpretingIt(t *testing.T) {
	active := richFixture(t)
	legacy := v1Revision(active)

	v1Projection, err := Project(legacy)
	if err != nil {
		t.Fatal(err)
	}
	v2Projection, err := Project(active)
	if err != nil {
		t.Fatal(err)
	}
	if v1Projection["schema_version"] != int64(domain.LegacyCanonicalSchemaVersion) {
		t.Fatalf("v1 schema tag = %v", v1Projection["schema_version"])
	}
	if v2Projection["schema_version"] != int64(domain.CanonicalSchemaVersion) {
		t.Fatalf("v2 schema tag = %v", v2Projection["schema_version"])
	}
	if _, exists := v1Projection["session_progress"]; exists {
		t.Fatal("the v1 projection must not contain session_progress")
	}
	if _, exists := v2Projection["session_progress"]; !exists {
		t.Fatal("the v2 projection must contain session_progress")
	}

	v1Keys := projectionKeys(v1Projection)
	v2Keys := projectionKeys(v2Projection)
	extra := make([]string, 0, 1)
	for _, key := range v2Keys {
		if !containsString(v1Keys, key) {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	if strings.Join(extra, ",") != "session_progress" {
		t.Fatalf("v2 added unexpected fields: %v", extra)
	}
	for _, key := range v1Keys {
		if key == "schema_version" {
			continue
		}
		if !equalProjection(v1Projection[key], v2Projection[key]) {
			t.Fatalf("v2 reinterpreted the v1 field %q", key)
		}
	}

	hashV1, err := Hash(legacy)
	if err != nil {
		t.Fatal(err)
	}
	hashV2, err := Hash(active)
	if err != nil {
		t.Fatal(err)
	}
	if hashV1 == hashV2 {
		t.Fatal("v1 and v2 hashes must differ because the schema tag and payload differ")
	}
}

// TestRetainedXPIsCoveredByCanonicalV2 is acceptance Test A: XP retained by an
// actor outside the current-wave member list changes the active canonical hash.
func TestRetainedXPIsCoveredByCanonicalV2(t *testing.T) {
	base := baseState(t)
	base.XP = []domain.PlayerXP{{ActorKey: actor(1), XP: 40}, {ActorKey: actor(2), XP: 50}}
	changed := base.Clone()
	changed.XP = []domain.PlayerXP{{ActorKey: actor(1), XP: 40}, {ActorKey: actor(2), XP: 60}}
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := changed.Validate(); err != nil {
		t.Fatal(err)
	}
	hashBase, err := Hash(base)
	if err != nil {
		t.Fatal(err)
	}
	hashChanged, err := Hash(changed)
	if err != nil {
		t.Fatal(err)
	}
	if hashBase == hashChanged {
		t.Fatal("v2 must cover retained non-member XP")
	}

	// Historical v1 keeps the documented gap so old hashes remain meaningful.
	v1Base, err := Hash(v1Revision(base))
	if err != nil {
		t.Fatal(err)
	}
	v1Changed, err := Hash(v1Revision(changed))
	if err != nil {
		t.Fatal(err)
	}
	if v1Base != v1Changed {
		t.Fatal("the historical v1 compatibility path must remain unchanged")
	}
}

// TestSessionProgressOrderingAndDuplicates is acceptance Tests B and C: the
// canonical contract is fail-closed on non-canonical order and duplicate actor
// keys. The projection never normalizes a state the domain considers invalid.
func TestSessionProgressOrderingAndDuplicates(t *testing.T) {
	unsorted := baseState(t)
	unsorted.XP = []domain.PlayerXP{{ActorKey: actor(2), XP: 10}, {ActorKey: actor(1), XP: 10}}
	if err := unsorted.Validate(); err == nil {
		t.Fatal("unsorted session XP must be rejected by domain validation")
	}
	if _, err := Project(unsorted); err == nil {
		t.Fatal("unsorted session XP must be rejected by the projection")
	}

	duplicate := baseState(t)
	duplicate.XP = []domain.PlayerXP{{ActorKey: actor(1), XP: 10}, {ActorKey: actor(1), XP: 20}}
	if err := duplicate.Validate(); err == nil {
		t.Fatal("duplicate session XP actor must be rejected")
	}
	if _, err := Project(duplicate); err == nil {
		t.Fatal("duplicate session XP actor must be rejected by the projection")
	}

	sorted := baseState(t)
	sorted.XP = []domain.PlayerXP{{ActorKey: actor(1), XP: 10}, {ActorKey: actor(2), XP: 20}}
	first, err := Hash(sorted)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Hash(sorted.Clone())
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("equivalent accepted states must hash identically")
	}
}

// TestSessionProgressBoundaries is acceptance Test D.
func TestSessionProgressBoundaries(t *testing.T) {
	valid := baseState(t)
	valid.XP = []domain.PlayerXP{{ActorKey: actor(1), XP: 0}, {ActorKey: actor(2), XP: domain.CapSessionXP}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid XP boundaries rejected: %v", err)
	}
	if _, err := Hash(valid); err != nil {
		t.Fatalf("valid XP boundaries not hashable: %v", err)
	}

	negative := baseState(t)
	negative.XP = []domain.PlayerXP{{ActorKey: actor(1), XP: -1}}
	if _, err := Project(negative); err == nil {
		t.Fatal("negative session XP must be rejected")
	}
	overflow := baseState(t)
	overflow.XP = []domain.PlayerXP{{ActorKey: actor(1), XP: domain.CapSessionXP + 1}}
	if _, err := Project(overflow); err == nil {
		t.Fatal("session XP above the cap must be rejected")
	}
}

// TestEmptySessionProgressEncoding is acceptance Test E: an empty collection is an
// explicit array, never null or an omitted field.
func TestEmptySessionProgressEncoding(t *testing.T) {
	state := baseState(t)
	state.XP = []domain.PlayerXP{}
	projection, err := Project(state)
	if err != nil {
		t.Fatal(err)
	}
	progress, ok := projection["session_progress"].([]any)
	if !ok || len(progress) != 0 {
		t.Fatalf("session_progress = %#v, want an empty array", projection["session_progress"])
	}
	encoded, err := MarshalJCS(projection)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"session_progress":[]`) {
		t.Fatalf("empty session_progress is not an explicit array: %s", encoded)
	}
}

// TestMemberProjectedXPAgreesWithProgress pins that the retained member field and
// the new collection cannot disagree: both read the same authoritative rows.
func TestMemberProjectedXPAgreesWithProgress(t *testing.T) {
	state := baseState(t)
	state.XP = []domain.PlayerXP{{ActorKey: actor(1), XP: 250}}
	projection, err := Project(state)
	if err != nil {
		t.Fatal(err)
	}
	members := projection["members"].([]any)
	member := members[0].(map[string]any)
	if member["session_xp"] != int64(250) {
		t.Fatalf("member session_xp = %v", member["session_xp"])
	}
	progress := projection["session_progress"].([]any)
	row := progress[0].(map[string]any)
	if row["actor_key"] != string(actor(1)) || row["session_xp"] != int64(250) {
		t.Fatalf("session_progress row = %v", row)
	}
}

// TestHashingDoesNotMutateAuthoritativeState keeps hashing a pure read.
func TestHashingDoesNotMutateAuthoritativeState(t *testing.T) {
	state := richFixture(t)
	before := state.Clone()
	if _, err := Hash(state); err != nil {
		t.Fatal(err)
	}
	if differences := stateDifferencesForTest(state, before); len(differences) > 0 {
		t.Fatalf("hashing mutated the state: %v", differences)
	}
}

func projectionKeys(projection map[string]any) []string {
	keys := make([]string, 0, len(projection))
	for key := range projection {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func equalProjection(left, right any) bool {
	leftJSON, leftErr := MarshalJCS(left)
	rightJSON, rightErr := MarshalJCS(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftJSON) == string(rightJSON)
}

func stateDifferencesForTest(got, want domain.SessionState) []string {
	var differences []string
	if got.Revision != want.Revision {
		differences = append(differences, "revision")
	}
	if len(got.Members) != len(want.Members) {
		differences = append(differences, "members length")
	}
	if len(got.XP) != len(want.XP) {
		differences = append(differences, "xp length")
	}
	if got.Village.Resources != want.Village.Resources {
		differences = append(differences, "resources")
	}
	return differences
}
