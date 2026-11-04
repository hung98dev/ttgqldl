package simulation

import (
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// TestPrivatePayloadCoversEveryAuthoritativeField is the regression guard for the
// private restoration contract. The reviewed tables below must name every exported
// field of every authoritative structure, and every mapped key must exist in the
// payload produced by privateRestorationPayload. Reflection is used only here in the
// test: a newly added SessionState field therefore fails this test until it is
// deliberately added to the payload and to the reviewed table.
func TestPrivatePayloadCoversEveryAuthoritativeField(t *testing.T) {
	state, queue := privatePayloadFixture(t)
	payload, err := privateRestorationPayload(state, queue)
	if err != nil {
		t.Fatalf("build private payload: %v", err)
	}

	for _, key := range []string{"schema_version", "rules_version", "state", "role_queue", "ballot_queue"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("private payload is missing the top-level key %q", key)
		}
	}
	statePayload := payloadMap(t, payload, "state")

	requireStructCoverage(t, "SessionState", reflect.TypeOf(domain.SessionState{}), statePayload, map[string]string{
		"Revision":          "revision",
		"ID":                "session_id",
		"VillageID":         "village_id",
		"Mode":              "mode",
		"Lifecycle":         "lifecycle",
		"PauseReasons":      "pause_reasons",
		"Village":           "village",
		"Wave":              "wave",
		"Members":           "members",
		"Pending":           "pending",
		"XP":                "xp",
		"Difficulty":        "difficulty",
		"ConsecutiveWins":   "consecutive_wins",
		"ConsecutiveLosses": "consecutive_losses",
		"PrivacyEpoch":      "privacy_epoch",
		"InputSequence":     "input_sequence",
	})

	requireStructCoverage(t, "Revision", reflect.TypeOf(domain.Revision{}), payloadMap(t, statePayload, "revision"), map[string]string{
		"SchemaVersion": "schema_version",
		"RulesVersion":  "rules_version",
	})
	requireStructCoverage(t, "VillageState", reflect.TypeOf(domain.VillageState{}), payloadMap(t, statePayload, "village"), map[string]string{
		"ID":               "id",
		"Level":            "level",
		"WallTier":         "wall_tier",
		"WallHP":           "wall_hp",
		"TownhallHP":       "townhall_hp",
		"Resources":        "resources",
		"ConstructionWork": "construction_work",
		"TotalWins":        "total_wins",
	})
	requireStructCoverage(t, "Resources", reflect.TypeOf(domain.Resources{}),
		payloadMap(t, payloadMap(t, statePayload, "village"), "resources"), map[string]string{
			"Wood":  "wood",
			"Stone": "stone",
			"Food":  "food",
		})
	wavePayload := payloadMap(t, statePayload, "wave")
	requireStructCoverage(t, "WaveState", reflect.TypeOf(domain.WaveState{}), wavePayload, map[string]string{
		"Number":       "number",
		"Tick":         "tick",
		"Phase":        "phase",
		"Pools":        "pools",
		"Boss":         "boss",
		"Vote":         "vote",
		"Ballots":      "ballots",
		"Defeated":     "defeated",
		"Result":       "result",
		"BossIncoming": "boss_incoming",
	})
	requireStructCoverage(t, "BossPool", reflect.TypeOf(domain.BossPool{}), payloadMap(t, wavePayload, "boss"), map[string]string{
		"Exists": "exists",
		"HP":     "hp",
		"MaxHP":  "max_hp",
	})
	votePayload := payloadMap(t, wavePayload, "vote")
	requireStructCoverage(t, "VoteState", reflect.TypeOf(domain.VoteState{}), votePayload, map[string]string{
		"Open":     "open",
		"Closed":   "closed",
		"Selected": "selected",
		"Counts":   "counts",
	})
	requireStructCoverage(t, "VoteCounts", reflect.TypeOf(domain.VoteCounts{}), payloadMap(t, votePayload, "counts"), map[string]string{
		"A": "a",
		"B": "b",
		"C": "c",
	})
	requireStructCoverage(t, "Difficulty", reflect.TypeOf(domain.Difficulty{}), payloadMap(t, statePayload, "difficulty"), map[string]string{
		"Level":            "level",
		"PopulationAnchor": "population_anchor",
		"Locked":           "locked",
		"ActualPopulation": "actual_population",
		"RecoveryPercent":  "recovery_percent",
		"BaseHP":           "base_hp",
		"MilitiaAttack":    "militia_attack",
		"BossMaxHP":        "boss_max_hp",
	})

	requireStructCoverage(t, "EnemyPool", reflect.TypeOf(domain.EnemyPool{}), firstElementMap(t, wavePayload, "pools"), map[string]string{
		"Lane":      "lane",
		"Archetype": "archetype",
		"HP":        "hp",
	})
	requireStructCoverage(t, "Ballot", reflect.TypeOf(domain.Ballot{}), firstElementMap(t, wavePayload, "ballots"), map[string]string{
		"ActorKey": "actor_key",
		"Option":   "option",
		"Tick":     "tick",
		"InputSeq": "input_seq",
	})
	requireStructCoverage(t, "WaveMember", reflect.TypeOf(domain.WaveMember{}), firstElementMap(t, statePayload, "members"), map[string]string{
		"ActorKey":        "actor_key",
		"Role":            "role",
		"JoinedTick":      "joined_tick",
		"LastChangeTick":  "last_change_tick",
		"FirstSwitchUsed": "first_switch_used",
		"XPAwarded":       "xp_awarded",
	})
	requireStructCoverage(t, "PendingMember", reflect.TypeOf(domain.PendingMember{}), firstElementMap(t, statePayload, "pending"), map[string]string{
		"ActorKey":        "actor_key",
		"Role":            "role",
		"TargetWave":      "target_wave",
		"LastChangeTick":  "last_change_tick",
		"FirstSwitchUsed": "first_switch_used",
	})
	requireStructCoverage(t, "PlayerXP", reflect.TypeOf(domain.PlayerXP{}), firstElementMap(t, statePayload, "xp"), map[string]string{
		"ActorKey": "actor_key",
		"XP":       "session_xp",
	})

	// The restoration queues are part of the same digest, not just the state.
	roleQueueEntry := firstElementMap(t, payload, "role_queue")
	requireStructCoverage(t, "RoleCommand", reflect.TypeOf(RoleCommand{}), roleQueueEntry, roleCommandCoverage())
	// A ballot can only be pending inside the voting window, so use the dedicated
	// vote-window fixture for that half of the queue contract.
	ballotState, ballotQueue := queuedCheckpointFixture(t)
	ballotPayload, err := privateRestorationPayload(ballotState, ballotQueue)
	if err != nil {
		t.Fatalf("build private payload for the ballot fixture: %v", err)
	}
	ballotQueueEntry := firstElementMap(t, ballotPayload, "ballot_queue")
	requireStructCoverage(t, "VoteCommand", reflect.TypeOf(VoteCommand{}), ballotQueueEntry, ballotCommandCoverage())

	// The canonical-v1 omissions from the acceptance audit must each be present in
	// the private payload; the collision test separately proves that changing them
	// changes the digest.
	for _, omission := range []string{"input_sequence", "mode"} {
		if !payloadContainsKey(t, statePayload, omission) {
			t.Fatalf("private payload lost the canonical-omission field %q", omission)
		}
	}
	for _, omission := range []string{"boss_incoming", "result"} {
		if !payloadContainsKey(t, wavePayload, omission) {
			t.Fatalf("private payload lost the canonical-omission field %q", omission)
		}
	}
	if !payloadContainsKey(t, payloadMap(t, statePayload, "village"), "level") {
		t.Fatal("private payload lost VillageState.Level")
	}
	for _, omission := range []string{"actual_population", "recovery_percent", "militia_attack", "boss_max_hp", "locked"} {
		if !payloadContainsKey(t, payloadMap(t, statePayload, "difficulty"), omission) {
			t.Fatalf("private payload lost the difficulty field %q", omission)
		}
	}
	if !payloadContainsKey(t, wavePayload, "ballots") {
		t.Fatal("private payload lost applied ballots")
	}
}

// privatePayloadFixture builds a rich but valid state: locked boss-wave difficulty,
// applied ballots, participants with role-change bookkeeping, retained XP, pending
// next-wave registrations, active pause reasons and one queued role command.
func privatePayloadFixture(t *testing.T) (domain.SessionState, RoleQueue) {
	t.Helper()
	state := collisionAssaultBase(t)

	changeTick := 1800
	state.Members[0].LastChangeTick = &changeTick
	state.Members[0].FirstSwitchUsed = true
	state.Pending = []domain.PendingMember{{
		ActorKey:   actorKey('p'),
		Role:       domain.RoleFighter,
		TargetWave: state.Wave.Number + 1,
	}}
	domain.SortPendingMembers(state.Pending)
	state.PauseReasons = []domain.PauseReason{domain.PauseOperatorPaused, domain.PauseStorageUnavailable}
	state.Lifecycle = domain.LifecyclePaused
	if err := state.Validate(); err != nil {
		t.Fatalf("payload fixture is invalid: %v", err)
	}
	queue := RoleQueue{Entries: []RoleCommand{{
		InputSeq: 900, ActorKey: actorKey('q'), Role: domain.RoleGuard,
		TargetWave: state.Wave.Number, ScheduledTick: state.Tick() + 1, AcceptedTick: state.Tick(),
	}}}
	return state, queue
}

func roleCommandCoverage() map[string]string {
	return map[string]string{
		"InputSeq":      "input_seq",
		"ActorKey":      "actor_key",
		"Role":          "role",
		"TargetWave":    "target_wave",
		"ScheduledTick": "scheduled_tick",
		"AcceptedTick":  "accepted_tick",
	}
}

func ballotCommandCoverage() map[string]string {
	return map[string]string{
		"InputSeq":      "input_seq",
		"ActorKey":      "actor_key",
		"TargetWave":    "target_wave",
		"AcceptedTick":  "accepted_tick",
		"ScheduledTick": "scheduled_tick",
		"Option":        "option",
	}
}

// requireStructCoverage fails when a struct field has no reviewed payload mapping,
// when the mapping is stale, or when the payload does not carry the mapped key.
func requireStructCoverage(t *testing.T, typeName string, typ reflect.Type, payload map[string]any, mapping map[string]string) {
	t.Helper()
	if typ.Kind() != reflect.Struct {
		t.Fatalf("%s is not a struct", typeName)
	}
	seen := make(map[string]bool, typ.NumField())
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		key, reviewed := mapping[field.Name]
		if !reviewed {
			t.Fatalf("%s.%s is not covered by the private restoration payload; add it to privateRestorationPayload and to the reviewed coverage table",
				typeName, field.Name)
		}
		seen[field.Name] = true
		if _, exists := payload[key]; !exists {
			t.Fatalf("%s.%s maps to payload key %q, which the payload does not contain", typeName, field.Name, key)
		}
	}
	for name := range mapping {
		if !seen[name] {
			t.Fatalf("reviewed coverage table lists %s.%s, which no longer exists; update the contract", typeName, name)
		}
	}
}

func payloadMap(t *testing.T, payload map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := payload[key]
	if !ok {
		t.Fatalf("payload key %q is missing", key)
	}
	nested, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("payload key %q is %T, not an object", key, value)
	}
	return nested
}

func firstElementMap(t *testing.T, payload map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := payload[key]
	if !ok {
		t.Fatalf("payload key %q is missing", key)
	}
	elements, ok := value.([]any)
	if !ok {
		t.Fatalf("payload key %q is %T, not an array", key, value)
	}
	if len(elements) == 0 {
		t.Fatalf("payload key %q is empty; the fixture must exercise it", key)
	}
	first, ok := elements[0].(map[string]any)
	if !ok {
		t.Fatalf("payload key %q element is %T, not an object", key, elements[0])
	}
	return first
}

func payloadContainsKey(t *testing.T, payload map[string]any, key string) bool {
	t.Helper()
	_, ok := payload[key]
	return ok
}
