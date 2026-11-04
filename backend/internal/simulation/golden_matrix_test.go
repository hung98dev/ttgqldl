package simulation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// The independent fixture schema this test understands. A different value means
// the reference artifact changed shape and the mapping below must be reviewed
// before the matrix can run again.
const goldenFixtureSchema = 2

// manifestSchema is the backend manifest contract, independent of the fixture.
const goldenManifestSchema = 3

// expectedSweepRuns is the retained independent sweep evidence. It is not claimed
// to run through the production reducer.
const expectedSweepRuns = 1080

// requiredGoldenCaseIDs is the reviewed coverage contract. Every listed case MUST
// execute against the real reducer. A case that cannot be mapped is a hard failure,
// never a silent skip, so the matrix can never shrink back to a partial subset.
var requiredGoldenCaseIDs = []string{
	"C001", "C002", "C003", "C004", "C005", "C006", "C007", "C008", "C009",
	"C010", "C011", "C012", "C013", "C014", "C015", "C016", "C017", "C018",
	"C019", "C020", "C021", "C022", "C023", "C024", "C025",
}

// --- fixture schema (strictly typed) ---

type goldenManifest struct {
	Schema                 int                     `json:"schema"`
	FixtureSchema          int                     `json:"fixture_schema"`
	RulesVersion           string                  `json:"rules_version"`
	CanonicalSchemaVersion int                     `json:"canonical_schema_version"`
	Source                 string                  `json:"source"`
	SourceDigestSHA256     string                  `json:"source_digest_sha256"`
	FixtureDigestSHA256    string                  `json:"fixture_digest_sha256"`
	CaseCount              int                     `json:"case_count"`
	CaseIDs                []string                `json:"case_ids"`
	Historical             *goldenHistoricalRecord `json:"historical"`
	Provenance             string                  `json:"provenance"`
	Regeneration           string                  `json:"regeneration"`
}

// goldenHistoricalRecord pins the retained schema-v1 reference artifact.
type goldenHistoricalRecord struct {
	CanonicalSchemaVersion int    `json:"canonical_schema_version"`
	FixtureDigestSHA256    string `json:"fixture_digest_sha256"`
	SourceCommit           string `json:"source_commit"`
	Note                   string `json:"note"`
}

type goldenFile struct {
	GeneratedBy            string            `json:"generated_by"`
	FixtureSchema          int               `json:"fixture_schema"`
	RulesVersion           string            `json:"rules_version"`
	CanonicalSchemaVersion int               `json:"canonical_schema_version"`
	SweepNote              string            `json:"sweep_note"`
	SweepRuns              int               `json:"sweep_runs"`
	SweepDigest            string            `json:"sweep_digest"`
	Aggregates             []json.RawMessage `json:"sweep_aggregates"`
	Cases                  []goldenCase      `json:"cases"`
}

type goldenCase struct {
	ID       string           `json:"id"`
	Note     string           `json:"note"`
	Inputs   goldenInputs     `json:"inputs"`
	Outcome  goldenOutcome    `json:"outcome"`
	Boundary []goldenTraceRow `json:"boundary_trace"`
}

type goldenInputs struct {
	WaveNo            int64             `json:"wave_no"`
	ConsecutiveWins   int64             `json:"consecutive_wins"`
	ConsecutiveLosses int64             `json:"consecutive_losses"`
	WallTier          int               `json:"wall_tier"`
	WallHP            int64             `json:"wall_hp"`
	TownhallHP        int64             `json:"townhall_hp"`
	Wood              int64             `json:"wood"`
	Stone             int64             `json:"stone"`
	Food              int64             `json:"food"`
	ConstructionWork  int64             `json:"construction_work"`
	Builders          int               `json:"builders"`
	Guards            int               `json:"guards"`
	Fighters          int               `json:"fighters"`
	Ballot            string            `json:"ballot_option,omitempty"`
	Joins             []goldenJoin      `json:"explicit_joins,omitempty"`
	Ballots           []goldenBallot    `json:"ballot_events,omitempty"`
	RoleEvents        []goldenRoleEvent `json:"role_events,omitempty"`
}

type goldenJoin struct {
	Actor string `json:"actor_key"`
	Role  string `json:"role"`
	Tick  int    `json:"scheduled_tick"`
}

type goldenBallot struct {
	Actor  string `json:"actor_key"`
	Option string `json:"option"`
	Tick   int    `json:"scheduled_tick"`
}

type goldenRoleEvent struct {
	Actor string `json:"actor_key"`
	Role  string `json:"role"`
	Tick  int    `json:"scheduled_tick"`
}

type goldenOutcome struct {
	Result            string             `json:"result"`
	WaveNo            int64              `json:"wave_no"`
	Tick              int                `json:"tick"`
	Lifecycle         string             `json:"lifecycle"`
	Defeated          bool               `json:"defeated"`
	AnchorPopulation  int64              `json:"anchor_population"`
	Difficulty        int64              `json:"difficulty"`
	BaseHP            int64              `json:"base_hp"`
	WallTier          int                `json:"wall_tier"`
	WallHP            int64              `json:"wall_hp"`
	WallMax           int64              `json:"wall_max"`
	TownhallHP        int64              `json:"townhall_hp"`
	Wood              int64              `json:"wood"`
	Stone             int64              `json:"stone"`
	Food              int64              `json:"food"`
	ConstructionWork  int64              `json:"construction_work"`
	TotalWins         int64              `json:"total_wins"`
	ConsecutiveWins   int64              `json:"consecutive_wins"`
	ConsecutiveLosses int64              `json:"consecutive_losses"`
	VoteChosen        string             `json:"vote_chosen"`
	SessionXPTotal    int64              `json:"session_xp_total"`
	BossHPAt2600      int64              `json:"boss_hp_at_2600"`
	OrdinaryHPAt2600  int64              `json:"ordinary_hp_at_2600"`
	StateDigest       string             `json:"state_digest"`
	VoteCounts        *goldenVoteCounts  `json:"vote_counts,omitempty"`
	MemberState       []goldenMember     `json:"member_state,omitempty"`
	RoleLog           []goldenRoleRecord `json:"role_events,omitempty"`
}

type goldenVoteCounts struct {
	A int64 `json:"a"`
	B int64 `json:"b"`
	C int64 `json:"c"`
}

type goldenMember struct {
	Actor           string `json:"actor_key"`
	Role            string `json:"role"`
	JoinedTick      int    `json:"joined_tick"`
	LastChangeTick  *int   `json:"last_change_tick"`
	FirstSwitchUsed bool   `json:"first_switch_used"`
	XPAwarded       bool   `json:"xp_awarded"`
	SessionXP       int64  `json:"session_xp"`
}

type goldenRoleRecord struct {
	Tick         int    `json:"tick"`
	Actor        string `json:"actor_key"`
	Role         string `json:"role"`
	PreviousRole string `json:"previous_role,omitempty"`
	Outcome      string `json:"outcome"`
}

type goldenTraceRow struct {
	Tick       int    `json:"tick"`
	Phase      string `json:"phase"`
	WallHP     int64  `json:"wall_hp"`
	WallMax    int64  `json:"wall_max"`
	TownhallHP int64  `json:"townhall_hp"`
	Wood       int64  `json:"wood"`
	Stone      int64  `json:"stone"`
	Food       int64  `json:"food"`
	Work       int64  `json:"construction_work"`
	WallTier   int    `json:"wall_tier"`
	OrdinaryHP int64  `json:"ordinary_hp"`
	BossHP     int64  `json:"boss_hp"`
	GuardBlock int64  `json:"guard_block"`
	Attack     int64  `json:"attack"`
	RawDamage  int64  `json:"raw_damage"`
	Damage     int64  `json:"damage_applied"`
	Defeated   bool   `json:"defeated"`
	SessionXP  int64  `json:"session_xp_total"`
	BossState  string `json:"boss_state"`
}

// --- observed production values ---

type goldenObserved struct {
	boundaries  map[int]goldenTraceRow
	memberByRef map[string]domain.WaveMember
	xpByActor   map[domain.ActorKey]int64
	roleEvents  []RoleEvent
	comparisons int
	rows        int
	roles       int
	events      int
}

// raceLargePopulationThreshold marks the single 10,000-participant acceptance
// case. Under race instrumentation that full wave is compute-bound, so only that
// case is excluded from the race build; the non-race unit gate always executes it.
const raceLargePopulationThreshold = 10000

// TestGoldenMatrixDifferential executes every curated independent reference case
// through the real reducer and compares the complete documented outcome plus every
// boundary-trace row. The fixture count, identities, versions and SHA-256 digests
// are enforced before a single case runs; an unsupported or missing case fails.
func TestGoldenMatrixDifferential(t *testing.T) {
	root := filepath.Join("..", "..")
	matrixPath := filepath.Join(root, "testdata", "golden", "matrix.json")
	manifestPath := filepath.Join(root, "testdata", "golden", "manifest.json")
	sourcePath := filepath.Join("..", "..", "..", "tooling", "refeval", "testdata", "golden_vectors.json")

	matrixBytes := readFixture(t, matrixPath)
	manifestBytes := readFixture(t, manifestPath)
	sourceBytes := readFixture(t, sourcePath)

	var manifest goldenManifest
	decodeStrict(t, manifestBytes, &manifest, "manifest")
	var fixture goldenFile
	decodeStrict(t, matrixBytes, &fixture, "matrix")

	verifyFixtureShape(t, matrixBytes)
	verifyManifestAndFixture(t, manifest, fixture, matrixBytes, sourceBytes)

	// The reviewed coverage contract is independent of the generated manifest:
	// the exact ID list must match, in order, with no duplicates and no extras.
	if len(fixture.Cases) != len(requiredGoldenCaseIDs) {
		t.Fatalf("fixture carries %d cases, the reviewed contract requires %d", len(fixture.Cases), len(requiredGoldenCaseIDs))
	}
	for index, want := range requiredGoldenCaseIDs {
		if fixture.Cases[index].ID != want {
			t.Fatalf("case %d is %q, the reviewed contract requires %q", index, fixture.Cases[index].ID, want)
		}
	}

	total := goldenObserved{boundaries: map[int]goldenTraceRow{}, memberByRef: map[string]domain.WaveMember{}, xpByActor: map[domain.ActorKey]int64{}}
	executed, raceExcluded := 0, 0
	for caseIndex := range fixture.Cases {
		ref := &fixture.Cases[caseIndex]
		participants := ref.Inputs.Builders + ref.Inputs.Guards + ref.Inputs.Fighters
		if raceDetectorEnabled && participants >= raceLargePopulationThreshold {
			// Explicit, reviewed exclusion: this single full-wave acceptance case is
			// compute-bound under race instrumentation. It stays mandatory in the
			// non-race unit gate, and the other 24 cases still run under -race.
			t.Logf("golden_case %s race_excluded=true reason=10k_participant_full_wave executed_by_non_race_gate=true", ref.ID)
			raceExcluded++
			continue
		}
		started := time.Now()
		observed := runGoldenCase(t, ref)
		if elapsed := time.Since(started); elapsed > 5*time.Second {
			t.Logf("golden_case %s wall=%s", ref.ID, elapsed.Round(time.Millisecond))
		}
		executed++
		total.comparisons += observed.comparisons
		total.rows += observed.rows
		total.roles += observed.roles
		total.events += observed.events
	}
	if executed+raceExcluded != len(fixture.Cases) {
		t.Fatalf("executed %d + race-excluded %d does not cover all %d cases", executed, raceExcluded, len(fixture.Cases))
	}
	if !raceDetectorEnabled && raceExcluded != 0 {
		t.Fatalf("%d case(s) were excluded outside the race build", raceExcluded)
	}
	if raceDetectorEnabled && raceExcluded > 1 {
		t.Fatalf("%d cases were excluded under race; only the single 10k case is allowed", raceExcluded)
	}

	t.Logf("golden_matrix: curated_reference_cases=%d curated_production_cases_executed=%d curated_cases_blocked=0 race_excluded_cases=%d boundary_trace_rows_compared=%d numeric_field_comparisons=%d independent_mismatches=0 fixture_digest=%s role_event_checks=%d emitted_events_seen=%d",
		len(fixture.Cases), executed, raceExcluded, total.rows, total.comparisons,
		manifest.FixtureDigestSHA256, total.roles, total.events)
}

// verifyManifestAndFixture enforces the artifact integrity contract. Every check
// here is about the reference artifact itself, not about production behaviour.
func verifyManifestAndFixture(t *testing.T, manifest goldenManifest, fixture goldenFile, matrixBytes, sourceBytes []byte) {
	t.Helper()
	if manifest.Schema != goldenManifestSchema {
		t.Fatalf("manifest schema = %d, want %d", manifest.Schema, goldenManifestSchema)
	}
	if manifest.FixtureSchema != goldenFixtureSchema {
		t.Fatalf("manifest fixture_schema = %d, want %d", manifest.FixtureSchema, goldenFixtureSchema)
	}
	if fixture.FixtureSchema != goldenFixtureSchema {
		t.Fatalf("fixture fixture_schema = %d, want %d", fixture.FixtureSchema, goldenFixtureSchema)
	}
	if fixture.RulesVersion != domain.RulesVersion || manifest.RulesVersion != domain.RulesVersion {
		t.Fatalf("rules version mismatch: fixture=%q manifest=%q want=%q", fixture.RulesVersion, manifest.RulesVersion, domain.RulesVersion)
	}
	if fixture.CanonicalSchemaVersion != domain.CanonicalSchemaVersion || manifest.CanonicalSchemaVersion != domain.CanonicalSchemaVersion {
		t.Fatalf("canonical schema mismatch: fixture=%d manifest=%d want=%d", fixture.CanonicalSchemaVersion, manifest.CanonicalSchemaVersion, domain.CanonicalSchemaVersion)
	}

	fixtureDigest := sha256Hex(matrixBytes)
	sourceDigest := sha256Hex(sourceBytes)
	if fixtureDigest != manifest.FixtureDigestSHA256 {
		t.Fatalf("fixture SHA256 = %s, manifest records %s", fixtureDigest, manifest.FixtureDigestSHA256)
	}
	if sourceDigest != manifest.SourceDigestSHA256 {
		t.Fatalf("reference source SHA256 = %s, manifest records %s", sourceDigest, manifest.SourceDigestSHA256)
	}
	if !bytes.Equal(matrixBytes, sourceBytes) {
		t.Fatal("backend matrix is not a byte-identical copy of the independent reference vectors")
	}
	if manifest.CaseCount != len(fixture.Cases) {
		t.Fatalf("manifest case_count = %d, fixture has %d cases", manifest.CaseCount, len(fixture.Cases))
	}
	if len(manifest.CaseIDs) != len(fixture.Cases) {
		t.Fatalf("manifest lists %d case ids, fixture has %d cases", len(manifest.CaseIDs), len(fixture.Cases))
	}
	seen := map[string]bool{}
	for index, id := range manifest.CaseIDs {
		if fixture.Cases[index].ID != id {
			t.Fatalf("manifest case %d is %q, fixture has %q", index, id, fixture.Cases[index].ID)
		}
		if seen[id] {
			t.Fatalf("duplicate case id %q in the manifest", id)
		}
		seen[id] = true
	}
	// The schema-v1 reference artifact is retained as an explicit historical
	// baseline; gameplay expectations are unchanged, only version metadata and the
	// reference state digests differ under v2.
	if manifest.Historical == nil {
		t.Fatal("manifest is missing the retained schema-v1 historical record")
	}
	if manifest.Historical.CanonicalSchemaVersion != domain.LegacyCanonicalSchemaVersion {
		t.Fatalf("historical canonical schema = %d, want %d",
			manifest.Historical.CanonicalSchemaVersion, domain.LegacyCanonicalSchemaVersion)
	}
	if len(manifest.Historical.FixtureDigestSHA256) != 64 {
		t.Fatalf("historical fixture digest = %q", manifest.Historical.FixtureDigestSHA256)
	}
	if _, err := hex.DecodeString(manifest.Historical.FixtureDigestSHA256); err != nil {
		t.Fatalf("historical fixture digest is not hex: %v", err)
	}
	if manifest.Historical.SourceCommit == "" || manifest.Historical.Note == "" {
		t.Fatal("historical record must carry its source commit and provenance note")
	}
	if manifest.Historical.FixtureDigestSHA256 == manifest.FixtureDigestSHA256 {
		t.Fatal("the historical v1 artifact must not be the current v2 artifact")
	}

	// The independent sweep is retained as reference evidence only: it is not
	// claimed to run through the production reducer.
	if fixture.SweepRuns != expectedSweepRuns || fixture.SweepDigest == "" {
		t.Fatalf("independent sweep artifact lost: runs=%d digest=%q", fixture.SweepRuns, fixture.SweepDigest)
	}
	for _, required := range requiredGoldenCaseIDs {
		if !seen[required] {
			t.Fatalf("required case %q is missing from the fixture", required)
		}
	}
	if len(fixture.Cases) != len(requiredGoldenCaseIDs) {
		t.Fatalf("unexpected case count %d, contract is %d", len(fixture.Cases), len(requiredGoldenCaseIDs))
	}
}

// verifyFixtureShape enforces the exact key sets of the reviewed fixture schema.
// An added, renamed or removed key fails here before any case executes, so a
// schema change cannot be silently ignored by the decoder.
func verifyFixtureShape(t *testing.T, data []byte) {
	t.Helper()
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("fixture is not a JSON object: %v", err)
	}
	requireKeys(t, root, "fixture",
		[]string{"generated_by", "fixture_schema", "rules_version", "canonical_schema_version", "sweep_note", "sweep_runs", "sweep_digest", "sweep_aggregates", "cases"}, nil)

	var cases []map[string]json.RawMessage
	if err := json.Unmarshal(root["cases"], &cases); err != nil {
		t.Fatalf("fixture cases: %v", err)
	}
	for index, rawCase := range cases {
		where := fmt.Sprintf("cases[%d]", index)
		requireKeys(t, rawCase, where, []string{"id", "note", "inputs", "outcome", "boundary_trace"}, nil)

		var inputs map[string]json.RawMessage
		if err := json.Unmarshal(rawCase["inputs"], &inputs); err != nil {
			t.Fatalf("%s.inputs: %v", where, err)
		}
		requireKeys(t, inputs, where+".inputs",
			[]string{"wave_no", "consecutive_wins", "consecutive_losses", "wall_tier", "wall_hp", "townhall_hp", "wood", "stone", "food", "construction_work", "builders", "guards", "fighters"},
			[]string{"ballot_option", "explicit_joins", "ballot_events", "role_events"})
		for _, key := range []string{"explicit_joins", "ballot_events", "role_events"} {
			raw, ok := inputs[key]
			if !ok {
				continue
			}
			var entries []map[string]json.RawMessage
			if err := json.Unmarshal(raw, &entries); err != nil {
				t.Fatalf("%s.inputs.%s: %v", where, key, err)
			}
			for entryIndex, entry := range entries {
				entryWhere := fmt.Sprintf("%s.inputs.%s[%d]", where, key, entryIndex)
				switch key {
				case "explicit_joins":
					requireKeys(t, entry, entryWhere, []string{"actor_key", "role", "scheduled_tick"}, nil)
				case "ballot_events":
					requireKeys(t, entry, entryWhere, []string{"actor_key", "option", "scheduled_tick"}, nil)
				default:
					requireKeys(t, entry, entryWhere, []string{"actor_key", "role", "scheduled_tick"}, nil)
				}
			}
		}

		var outcome map[string]json.RawMessage
		if err := json.Unmarshal(rawCase["outcome"], &outcome); err != nil {
			t.Fatalf("%s.outcome: %v", where, err)
		}
		requireKeys(t, outcome, where+".outcome",
			[]string{"result", "wave_no", "tick", "lifecycle", "defeated", "anchor_population", "difficulty", "base_hp", "wall_tier", "wall_hp", "wall_max", "townhall_hp", "wood", "stone", "food", "construction_work", "total_wins", "consecutive_wins", "consecutive_losses", "vote_chosen", "session_xp_total", "boss_hp_at_2600", "ordinary_hp_at_2600", "state_digest"},
			[]string{"vote_counts", "member_state", "role_events"})
		if raw, ok := outcome["vote_counts"]; ok {
			var counts map[string]json.RawMessage
			if err := json.Unmarshal(raw, &counts); err != nil {
				t.Fatalf("%s.outcome.vote_counts: %v", where, err)
			}
			requireKeys(t, counts, where+".outcome.vote_counts", []string{"a", "b", "c"}, nil)
		}
		if raw, ok := outcome["member_state"]; ok {
			var members []map[string]json.RawMessage
			if err := json.Unmarshal(raw, &members); err != nil {
				t.Fatalf("%s.outcome.member_state: %v", where, err)
			}
			for memberIndex, member := range members {
				requireKeys(t, member, fmt.Sprintf("%s.outcome.member_state[%d]", where, memberIndex),
					[]string{"actor_key", "role", "joined_tick", "last_change_tick", "first_switch_used", "xp_awarded", "session_xp"}, nil)
			}
		}
		if raw, ok := outcome["role_events"]; ok {
			var records []map[string]json.RawMessage
			if err := json.Unmarshal(raw, &records); err != nil {
				t.Fatalf("%s.outcome.role_events: %v", where, err)
			}
			for recordIndex, record := range records {
				requireKeys(t, record, fmt.Sprintf("%s.outcome.role_events[%d]", where, recordIndex),
					[]string{"tick", "actor_key", "role", "outcome"}, []string{"previous_role"})
			}
		}

		var trace []map[string]json.RawMessage
		if err := json.Unmarshal(rawCase["boundary_trace"], &trace); err != nil {
			t.Fatalf("%s.boundary_trace: %v", where, err)
		}
		for rowIndex, row := range trace {
			requireKeys(t, row, fmt.Sprintf("%s.boundary_trace[%d]", where, rowIndex),
				[]string{"tick", "phase", "wall_hp", "wall_max", "townhall_hp", "wood", "stone", "food", "construction_work", "wall_tier", "ordinary_hp", "boss_hp", "guard_block", "attack", "raw_damage", "damage_applied", "defeated", "session_xp_total", "boss_state"}, nil)
		}
	}
}

func requireKeys(t *testing.T, object map[string]json.RawMessage, where string, required, optional []string) {
	t.Helper()
	allowed := make(map[string]bool, len(required)+len(optional))
	for _, key := range required {
		allowed[key] = true
		if _, ok := object[key]; !ok {
			t.Fatalf("%s is missing the required key %q", where, key)
		}
	}
	for _, key := range optional {
		allowed[key] = true
	}
	for key := range object {
		if !allowed[key] {
			t.Fatalf("%s carries the unreviewed key %q", where, key)
		}
	}
}

// runGoldenCase maps one reference case onto the real production APIs: explicit
// scheduled registrations, explicit ballots, role-change events, then the actual
// reducer tick by tick. Expected values are always read from the fixture; nothing
// is derived from the production state being checked.
func runGoldenCase(t *testing.T, ref *goldenCase) goldenObserved {
	t.Helper()
	state, queue := buildGoldenFixture(t, ref)
	observed := goldenObserved{
		boundaries:  map[int]goldenTraceRow{},
		memberByRef: map[string]domain.WaveMember{},
		xpByActor:   map[domain.ActorKey]int64{},
	}

	traceTicks := map[int]bool{}
	for _, row := range ref.Boundary {
		traceTicks[row.Tick] = true
	}

	joins := goldenJoins(ref)
	first := joins[0]
	nextSeq := int64(1)
	startCmd := RoleCommand{
		InputSeq: nextSeq, ActorKey: goldenActor(first.Actor), Role: goldenRole(t, first.Role),
		TargetWave: state.Wave.Number, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
	}
	nextSeq++
	accepted, startQueue, _, err := AcceptRoleCommands(state, queue, []RoleCommand{startCmd})
	if err != nil {
		t.Fatalf("%s: accept wave start: %v", ref.ID, err)
	}
	state, queue, startEvents, startRoleEvents, err := AdvanceWithRoles(accepted, startQueue, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: startCmd.ActorKey, Role: startCmd.Role},
	})
	if err != nil {
		t.Fatalf("%s: start wave: %v", ref.ID, err)
	}
	observed.roleEvents = append(observed.roleEvents, startRoleEvents...)
	observed.events += len(startEvents)

	// Every scheduled input is accepted at committed_tick = effective_tick - 1,
	// exactly the ingest contract, and in one batch per tick so a 10,000-participant
	// case stays a single validation pass.
	type scheduled struct {
		tick   int
		role   *RoleCommand
		ballot *VoteCommand
	}
	var schedule []scheduled
	for _, join := range joins[1:] {
		command := RoleCommand{
			InputSeq: nextSeq, ActorKey: goldenActor(join.Actor), Role: goldenRole(t, join.Role),
			TargetWave: state.Wave.Number, ScheduledTick: join.Tick, AcceptedTick: AcceptedTickUnspecified,
		}
		nextSeq++
		schedule = append(schedule, scheduled{tick: join.Tick, role: &command})
	}
	for _, change := range ref.Inputs.RoleEvents {
		command := RoleCommand{
			InputSeq: nextSeq, ActorKey: goldenActor(change.Actor), Role: goldenRole(t, change.Role),
			TargetWave: state.Wave.Number, ScheduledTick: change.Tick, AcceptedTick: AcceptedTickUnspecified,
		}
		nextSeq++
		schedule = append(schedule, scheduled{tick: change.Tick, role: &command})
	}
	for _, ballot := range goldenBallots(ref) {
		command := VoteCommand{
			InputSeq: nextSeq, ActorKey: goldenActor(ballot.Actor), TargetWave: state.Wave.Number,
			ScheduledTick: ballot.Tick, Option: goldenVoteOption(t, ballot.Option), AcceptedTick: AcceptedTickUnspecified,
		}
		nextSeq++
		schedule = append(schedule, scheduled{tick: ballot.Tick, ballot: &command})
	}
	sort.SliceStable(schedule, func(i, j int) bool { return schedule[i].tick < schedule[j].tick })

	record := func(events []Event) {
		tick := state.Tick()
		if !traceTicks[tick] {
			return
		}
		if _, exists := observed.boundaries[tick]; exists {
			t.Fatalf("%s: duplicate boundary record at tick %d", ref.ID, tick)
		}
		observed.boundaries[tick] = projectBoundary(state, events)
	}
	advanceTo := func(target int) {
		for state.Tick() < target {
			next, nextQueue, events, roleEvents, advanceErr := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
			if advanceErr != nil {
				t.Fatalf("%s: advance at tick %d: %v", ref.ID, state.Tick(), advanceErr)
			}
			state, queue = next, nextQueue
			observed.events += len(events)
			observed.roleEvents = append(observed.roleEvents, roleEvents...)
			record(events)
		}
	}
	applySchedule := func(upTo int) {
		for len(schedule) > 0 && schedule[0].tick <= upTo {
			groupTick := schedule[0].tick
			advanceTo(groupTick - 1)
			var roles []RoleCommand
			var ballots []VoteCommand
			for len(schedule) > 0 && schedule[0].tick == groupTick {
				entry := schedule[0]
				schedule = schedule[1:]
				if entry.role != nil {
					roles = append(roles, *entry.role)
				}
				if entry.ballot != nil {
					ballots = append(ballots, *entry.ballot)
				}
			}
			if len(roles) > 0 {
				accepted, nextQueue, _, acceptErr := AcceptRoleCommands(state, queue, roles)
				if acceptErr != nil {
					t.Fatalf("%s: accept %d role command(s) effective at %d: %v", ref.ID, len(roles), groupTick, acceptErr)
				}
				state, queue = accepted, nextQueue
			}
			if len(ballots) > 0 {
				accepted, nextQueue, _, acceptErr := AcceptVoteCommands(state, queue, ballots)
				if acceptErr != nil {
					t.Fatalf("%s: accept %d ballot(s) effective at %d: %v", ref.ID, len(ballots), groupTick, acceptErr)
				}
				state, queue = accepted, nextQueue
			}
		}
	}

	for state.Tick() < domain.WaveTicks-1 {
		applySchedule(state.Tick() + 1)
		advanceTo(state.Tick() + 1)
	}
	if len(schedule) > 0 {
		t.Fatalf("%s: %d scheduled inputs were never applied", ref.ID, len(schedule))
	}
	if state.Tick() != domain.WaveTicks-1 {
		t.Fatalf("%s: reached tick %d, want %d", ref.ID, state.Tick(), domain.WaveTicks-1)
	}

	// Role-decision events: the independent log records the attempted role, while
	// the production rejection event reports the still-active role. The decision
	// outcome and tick are compared exactly; the retained role is proven by the
	// participant state below.
	if len(ref.Outcome.RoleLog) > 0 {
		if len(observed.roleEvents) != len(ref.Outcome.RoleLog) {
			t.Fatalf("%s: production emitted %d role events, reference expects %d", ref.ID, len(observed.roleEvents), len(ref.Outcome.RoleLog))
		}
		for index, want := range ref.Outcome.RoleLog {
			got := observed.roleEvents[index]
			wantType := goldenRoleEventType(t, want.Outcome)
			if got.Tick != want.Tick || got.Type != wantType {
				t.Fatalf("%s: role decision %d = (tick %d, %s), reference expects (tick %d, %s)",
					ref.ID, index, got.Tick, got.Type, want.Tick, wantType)
			}
			if want.PreviousRole != "" && got.PreviousRole.String() != want.PreviousRole {
				t.Fatalf("%s: role decision %d previous role = %s, reference expects %s",
					ref.ID, index, got.PreviousRole, want.PreviousRole)
			}
			observed.roles++
		}
	}

	// Complete participant state, keyed by the reference actor name.
	actorByKey := map[domain.ActorKey]string{}
	for _, join := range joins {
		actorByKey[goldenActor(join.Actor)] = join.Actor
	}
	for _, member := range state.Members {
		name, ok := actorByKey[member.ActorKey]
		if !ok {
			t.Fatalf("%s: production registered an unexpected participant", ref.ID)
		}
		observed.memberByRef[name] = member
	}
	for _, row := range state.XP {
		observed.xpByActor[row.ActorKey] = row.XP
	}

	compareGoldenOutcome(t, ref, state, &observed)
	compareGoldenTrace(t, ref, &observed)
	return observed
}

// buildGoldenFixture constructs the documented initial fixture, then starts the
// wave through the accepted-registration API. The persisted village values come
// from the reference inputs; no gameplay outcome is assigned.
func buildGoldenFixture(t *testing.T, ref *goldenCase) (domain.SessionState, RoleQueue) {
	t.Helper()
	state, err := domain.NewInitialState(villageID, sessionID, domain.SourceModeMock)
	if err != nil {
		t.Fatalf("%s: initial state: %v", ref.ID, err)
	}
	inputs := ref.Inputs
	state.Wave.Number = inputs.WaveNo
	state.ConsecutiveWins = inputs.ConsecutiveWins
	state.ConsecutiveLosses = inputs.ConsecutiveLosses
	state.Village.WallTier = inputs.WallTier
	state.Village.WallHP = inputs.WallHP
	state.Village.TownhallHP = inputs.TownhallHP
	state.Village.Resources = domain.Resources{Wood: inputs.Wood, Stone: inputs.Stone, Food: inputs.Food}
	state.Village.ConstructionWork = inputs.ConstructionWork
	if err := state.Validate(); err != nil {
		t.Fatalf("%s: fixture state is invalid: %v", ref.ID, err)
	}
	return state, RoleQueue{}
}

func compareGoldenOutcome(t *testing.T, ref *goldenCase, state domain.SessionState, observed *goldenObserved) {
	t.Helper()
	var xpTotal int64
	for _, row := range state.XP {
		xpTotal += row.XP
	}
	wallMax, err := state.Village.WallMax()
	if err != nil {
		t.Fatalf("%s: wall max: %v", ref.ID, err)
	}
	checks := []struct {
		field string
		want  any
		got   any
	}{
		{"wave_no", ref.Outcome.WaveNo, state.Wave.Number},
		{"wave_tick", ref.Outcome.Tick, state.Tick()},
		{"lifecycle", ref.Outcome.Lifecycle, state.Lifecycle.String()},
		{"defeated", ref.Outcome.Defeated, state.Wave.Defeated},
		{"anchor_population", ref.Outcome.AnchorPopulation, state.Difficulty.PopulationAnchor},
		{"difficulty", ref.Outcome.Difficulty, state.Difficulty.Level},
		{"base_hp", ref.Outcome.BaseHP, state.Difficulty.BaseHP},
		{"wall_tier", ref.Outcome.WallTier, state.Village.WallTier},
		{"wall_hp", ref.Outcome.WallHP, state.Village.WallHP},
		{"wall_max", ref.Outcome.WallMax, wallMax},
		{"townhall_hp", ref.Outcome.TownhallHP, state.Village.TownhallHP},
		{"wood", ref.Outcome.Wood, state.Village.Resources.Wood},
		{"stone", ref.Outcome.Stone, state.Village.Resources.Stone},
		{"food", ref.Outcome.Food, state.Village.Resources.Food},
		{"construction_work", ref.Outcome.ConstructionWork, state.Village.ConstructionWork},
		{"total_wins", ref.Outcome.TotalWins, state.Village.TotalWins},
		{"consecutive_wins", ref.Outcome.ConsecutiveWins, state.ConsecutiveWins},
		{"consecutive_losses", ref.Outcome.ConsecutiveLosses, state.ConsecutiveLosses},
		{"vote_chosen", ref.Outcome.VoteChosen, state.Wave.Vote.Selected.String()},
		{"session_xp_total", ref.Outcome.SessionXPTotal, xpTotal},
		{"final_result", normalizeGoldenResult(ref.Outcome.Result), state.Wave.Result.String()},
	}
	for _, check := range checks {
		observed.comparisons++
		if fmt.Sprint(check.want) != fmt.Sprint(check.got) {
			t.Fatalf("case_id=%s wave_no=%d tick=%d field=%s expected=%v actual=%v",
				ref.ID, ref.Outcome.WaveNo, state.Tick(), check.field, check.want, check.got)
		}
	}

	if ref.Outcome.VoteCounts != nil {
		want := ref.Outcome.VoteCounts
		got := state.Wave.Vote.Counts
		observed.comparisons++
		if want.A != got.A || want.B != got.B || want.C != got.C {
			t.Fatalf("case_id=%s applied ballot tally expected A=%d B=%d C=%d actual A=%d B=%d C=%d",
				ref.ID, want.A, want.B, want.C, got.A, got.B, got.C)
		}
	}

	// The tick-2600 snapshot is compared from the recorded boundary row, which is
	// present exactly when the reference produced one (a frozen post-defeat wave
	// stops emitting trace rows).
	if row, ok := observed.boundaries[domain.BossSpawnTick]; ok && !ref.Outcome.Defeated {
		observed.comparisons += 2
		if ref.Outcome.BossHPAt2600 != row.BossHP {
			t.Fatalf("case_id=%s boss_hp_at_2600 expected=%d actual=%d", ref.ID, ref.Outcome.BossHPAt2600, row.BossHP)
		}
		if ref.Outcome.OrdinaryHPAt2600 != row.OrdinaryHP {
			t.Fatalf("case_id=%s ordinary_hp_at_2600 expected=%d actual=%d", ref.ID, ref.Outcome.OrdinaryHPAt2600, row.OrdinaryHP)
		}
	}

	// Complete participant state for cases that publish it.
	if len(ref.Outcome.MemberState) > 0 {
		if len(ref.Outcome.MemberState) != len(observed.memberByRef) {
			t.Fatalf("case_id=%s participant count expected=%d actual=%d", ref.ID, len(ref.Outcome.MemberState), len(observed.memberByRef))
		}
		for _, want := range ref.Outcome.MemberState {
			got, ok := observed.memberByRef[want.Actor]
			if !ok {
				t.Fatalf("case_id=%s participant %s is missing", ref.ID, want.Actor)
			}
			observed.comparisons++
			if got.Role.String() != want.Role {
				t.Fatalf("case_id=%s participant %s role expected=%s actual=%s", ref.ID, want.Actor, want.Role, got.Role)
			}
			observed.comparisons++
			if got.JoinedTick != want.JoinedTick {
				t.Fatalf("case_id=%s participant %s joined_tick expected=%d actual=%d", ref.ID, want.Actor, want.JoinedTick, got.JoinedTick)
			}
			observed.comparisons++
			if !sameLastChange(got.LastChangeTick, want.LastChangeTick) {
				t.Fatalf("case_id=%s participant %s last_change_tick expected=%v actual=%v", ref.ID, want.Actor, want.LastChangeTick, got.LastChangeTick)
			}
			observed.comparisons++
			if got.FirstSwitchUsed != want.FirstSwitchUsed {
				t.Fatalf("case_id=%s participant %s first_switch_used expected=%t actual=%t", ref.ID, want.Actor, want.FirstSwitchUsed, got.FirstSwitchUsed)
			}
			observed.comparisons++
			if got.XPAwarded != want.XPAwarded {
				t.Fatalf("case_id=%s participant %s xp_awarded expected=%t actual=%t", ref.ID, want.Actor, want.XPAwarded, got.XPAwarded)
			}
			observed.comparisons++
			if observed.xpByActor[goldenActor(want.Actor)] != want.SessionXP {
				t.Fatalf("case_id=%s participant %s session_xp expected=%d actual=%d",
					ref.ID, want.Actor, want.SessionXP, observed.xpByActor[goldenActor(want.Actor)])
			}
		}
	}
}

// compareGoldenTrace consumes every boundary row the reference produced and
// compares every field the production projection exposes. guard_block and
// raw_damage are not part of the authoritative projection or event payload, so
// they are compared only for non-combat boundaries where both are defined as 0;
// combat boundaries compare attack and damage_applied from the emitted events.
func compareGoldenTrace(t *testing.T, ref *goldenCase, observed *goldenObserved) {
	t.Helper()
	for _, want := range ref.Boundary {
		got, ok := observed.boundaries[want.Tick]
		if !ok {
			t.Fatalf("case_id=%s: boundary trace row %d was not produced", ref.ID, want.Tick)
		}
		checks := []struct {
			field string
			want  any
			got   any
		}{
			{"tick", want.Tick, got.Tick},
			{"phase", want.Phase, got.Phase},
			{"wall_hp", want.WallHP, got.WallHP},
			{"wall_max", want.WallMax, got.WallMax},
			{"townhall_hp", want.TownhallHP, got.TownhallHP},
			{"wood", want.Wood, got.Wood},
			{"stone", want.Stone, got.Stone},
			{"food", want.Food, got.Food},
			{"construction_work", want.Work, got.Work},
			{"wall_tier", want.WallTier, got.WallTier},
			{"ordinary_hp", want.OrdinaryHP, got.OrdinaryHP},
			{"boss_hp", want.BossHP, got.BossHP},
			{"defeated", want.Defeated, got.Defeated},
			{"session_xp_total", want.SessionXP, got.SessionXP},
			{"boss_state", want.BossState, got.BossState},
			{"attack", want.Attack, got.Attack},
			{"damage_applied", want.Damage, got.Damage},
		}
		if want.Tick == domain.FinalCombatTick {
			// The independent trace records the tick-2800 row between final combat
			// and the one-time settlement, while the production snapshot is the end
			// of the same boundary. Only settlement-mutated fields differ; the exact
			// post-settlement resources and streaks are compared against the fixture
			// outcome, which is captured at the same tick.
			checks = withoutSettlementFields(checks, want.Defeated)
		}
		if want.Tick < domain.CombatStartTick {
			checks = append(checks,
				struct {
					field string
					want  any
					got   any
				}{"guard_block", want.GuardBlock, got.GuardBlock},
				struct {
					field string
					want  any
					got   any
				}{"raw_damage", want.RawDamage, got.RawDamage})
		}
		for _, check := range checks {
			observed.comparisons++
			if fmt.Sprint(check.want) != fmt.Sprint(check.got) {
				t.Fatalf("case_id=%s boundary_tick=%d field=%s expected=%v actual=%v",
					ref.ID, want.Tick, check.field, check.want, check.got)
			}
		}
		observed.rows++
	}
}

// withoutSettlementFields drops the trace fields that the one-time tick-2800
// settlement rewrites between the reference capture point (immediately after final
// combat) and the end-of-boundary production snapshot.
func withoutSettlementFields(checks []struct {
	field string
	want  any
	got   any
}, defeated bool) []struct {
	field string
	want  any
	got   any
} {
	mutated := map[string]bool{"wood": true, "stone": true, "food": true}
	if defeated {
		// Defeat recovery floors are applied after the resource penalties.
		mutated["wall_hp"] = true
		mutated["townhall_hp"] = true
	}
	out := checks[:0]
	for _, check := range checks {
		if mutated[check.field] {
			continue
		}
		out = append(out, check)
	}
	return out
}

// projectBoundary reads the authoritative post-boundary state and the events that
// boundary emitted. attack and damage are only ever taken from canonical events.
func projectBoundary(state domain.SessionState, events []Event) goldenTraceRow {
	ordinary := int64(0)
	for _, pool := range state.Wave.Pools {
		ordinary += pool.HP
	}
	var xp int64
	for _, row := range state.XP {
		xp += row.XP
	}
	wallMax, _ := state.Village.WallMax()
	attack, damage := int64(0), int64(0)
	for _, event := range events {
		switch event.Type {
		case EventAttackResolved:
			attack = event.Amount
		case EventDamageApplied:
			damage = event.Amount
		}
	}
	return goldenTraceRow{
		Tick:       state.Tick(),
		Phase:      state.Wave.Phase.String(),
		WallHP:     state.Village.WallHP,
		WallMax:    wallMax,
		TownhallHP: state.Village.TownhallHP,
		Wood:       state.Village.Resources.Wood,
		Stone:      state.Village.Resources.Stone,
		Food:       state.Village.Resources.Food,
		Work:       state.Village.ConstructionWork,
		WallTier:   state.Village.WallTier,
		OrdinaryHP: ordinary,
		BossHP:     state.Wave.Boss.HP,
		Defeated:   state.Wave.Defeated,
		SessionXP:  xp,
		BossState:  productionBossState(state, ordinary),
		Attack:     attack,
		Damage:     damage,
	}
}

// productionBossState derives the same banner state as the reference model, from
// authoritative production fields only. Non-combat boundaries report "absent",
// matching the reference trace: the boss pool does not exist before tick 2600 and
// its banner state is only meaningful once combat has started.
func productionBossState(state domain.SessionState, ordinary int64) string {
	if state.Tick() < domain.CombatStartTick || !domain.IsBossWave(state.Wave.Number) {
		return "absent"
	}
	switch {
	case state.Wave.Boss.Exists && state.Wave.Boss.HP > 0:
		return "active"
	case state.Wave.Boss.Exists:
		return "destroyed"
	case ordinary == 0:
		return "incoming"
	default:
		return "scheduled"
	}
}

// --- fixture mapping helpers ---

func goldenJoins(ref *goldenCase) []goldenJoin {
	if len(ref.Inputs.Joins) > 0 {
		return ref.Inputs.Joins
	}
	// Synthetic population: builders, then guards, then fighters. The first
	// registration starts the wave at tick 0; the rest are accepted at tick 0 and
	// become effective at tick 1. The independent model uses the same shape.
	var joins []goldenJoin
	add := func(count int, prefix, role string) {
		for index := 0; index < count; index++ {
			tick := 1
			if len(joins) == 0 {
				tick = 0
			}
			joins = append(joins, goldenJoin{Actor: fmt.Sprintf("%s%05d", prefix, index), Role: role, Tick: tick})
		}
	}
	add(ref.Inputs.Builders, "b", "builder")
	add(ref.Inputs.Guards, "g", "guard")
	add(ref.Inputs.Fighters, "f", "fighter")
	return joins
}

func goldenBallots(ref *goldenCase) []goldenBallot {
	if len(ref.Inputs.Ballots) > 0 {
		return ref.Inputs.Ballots
	}
	if ref.Inputs.Ballot == "" {
		return nil
	}
	// Legacy shorthand: every participant casts the same option at tick 900.
	var ballots []goldenBallot
	for _, join := range goldenJoins(ref) {
		ballots = append(ballots, goldenBallot{Actor: join.Actor, Option: ref.Inputs.Ballot, Tick: 900})
	}
	return ballots
}

// goldenActor maps a reference actor name onto a valid scoped 64-hex actor key.
func goldenActor(name string) domain.ActorKey {
	sum := sha256.Sum256([]byte("ttg-reference-actor:" + name))
	return domain.ActorKey(hex.EncodeToString(sum[:]))
}

func goldenRole(t *testing.T, name string) domain.Role {
	t.Helper()
	role, err := domain.ParseRole(name)
	if err != nil {
		t.Fatalf("fixture role %q: %v", name, err)
	}
	return role
}

func goldenVoteOption(t *testing.T, name string) domain.VoteOption {
	t.Helper()
	switch name {
	case "A":
		return domain.VoteOptionA
	case "B":
		return domain.VoteOptionB
	case "C":
		return domain.VoteOptionC
	default:
		t.Fatalf("fixture ballot option %q is invalid", name)
		return domain.VoteOptionNone
	}
}

func goldenRoleEventType(t *testing.T, outcome string) RoleEventType {
	t.Helper()
	switch outcome {
	case "registered":
		return EventRoleAccepted
	case "role_changed":
		return EventRoleChanged
	case "cooldown_rejected":
		return EventRoleRejectedCooldown
	default:
		t.Fatalf("fixture role outcome %q is unknown", outcome)
		return ""
	}
}

func sameLastChange(got *int, want *int) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}

func normalizeGoldenResult(result string) string {
	switch result {
	case "loss":
		return "defeat"
	case "win":
		return "victory"
	default:
		return result
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}

// decodeStrict rejects unknown fields and trailing data, so a fixture shape change
// fails until the mapping above is reviewed and updated.
func decodeStrict(t *testing.T, data []byte, target any, what string) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("%s is not the reviewed fixture shape: %v", what, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("%s carries trailing JSON", what)
	}
}
