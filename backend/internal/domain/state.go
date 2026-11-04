package domain

import (
	"fmt"
	"regexp"
)

// Revision is the canonical serialization identity covered by the state hash.
type Revision struct {
	SchemaVersion int
	RulesVersion  string
}

// CurrentRevision returns the only revision this package produces.
func CurrentRevision() Revision {
	return Revision{SchemaVersion: CanonicalSchemaVersion, RulesVersion: RulesVersion}
}

func (r Revision) Valid() bool {
	return r.SchemaVersion == CanonicalSchemaVersion && r.RulesVersion == RulesVersion
}

var (
	uuidPattern     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	actorKeyPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// VillageID is a canonical lowercase UUID.
type VillageID string

func (v VillageID) Valid() bool { return uuidPattern.MatchString(string(v)) }

// SessionID is a canonical lowercase UUID.
type SessionID string

func (s SessionID) Valid() bool { return uuidPattern.MatchString(string(s)) }

// ActorKey is a scoped 64-character lowercase hex HMAC identity. Raw provider
// identifiers never enter the domain.
type ActorKey string

func (a ActorKey) Valid() bool { return actorKeyPattern.MatchString(string(a)) }

// Resources are the three persistent resource counters.
type Resources struct {
	Wood  int64
	Stone int64
	Food  int64
}

func (r Resources) Validate() *ValidationError {
	errs := &ValidationError{}
	if r.Wood < 0 || r.Wood > ResourceCap {
		errs.add("resources.wood", errResourceRange)
	}
	if r.Stone < 0 || r.Stone > ResourceCap {
		errs.add("resources.stone", errResourceRange)
	}
	if r.Food < 0 || r.Food > ResourceCap {
		errs.add("resources.food", errResourceRange)
	}
	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}

// VillageState is the persistent village: level, wall, townhall, resources and
// lifetime progress.
type VillageState struct {
	ID               VillageID
	Level            int
	WallTier         int
	WallHP           int64
	TownhallHP       int64
	Resources        Resources
	ConstructionWork int64
	TotalWins        int64
}

// WallMax returns the maximum wall HP for this village's tier.
func (v VillageState) WallMax() (int64, error) { return WallMax(v.WallTier) }

// WaveState is the per-wave runtime state. It never carries a wall-clock time.
type WaveState struct {
	Number   int64
	Tick     int
	Phase    Phase
	Pools    []EnemyPool
	Boss     BossPool
	Vote     VoteState
	Ballots  []Ballot
	Defeated bool
	Result   WaveResult
}

// Difficulty is the derived wave difficulty and the locked population anchor.
// PopulationAnchor is 0 until the anchor locks at PopulationLockTick.
type Difficulty struct {
	Level            int
	PopulationAnchor int
}

// SessionState is the complete authoritative state of one session. It is a
// value type: callers receive a deep copy through Clone and never share mutable
// backing slices with the simulation.
type SessionState struct {
	Revision          Revision
	ID                SessionID
	VillageID         VillageID
	Mode              SourceMode
	Lifecycle         Lifecycle
	PauseReasons      []PauseReason
	Village           VillageState
	Wave              WaveState
	Members           []WaveMember
	Pending           []PendingMember
	XP                []PlayerXP
	Difficulty        Difficulty
	ConsecutiveWins   int64
	ConsecutiveLosses int64
	PrivacyEpoch      int64
	InputSequence     int64
}

// WaveNo is the current wave number.
func (s SessionState) WaveNo() int64 { return s.Wave.Number }

// Tick is the current local wave tick (-1 while waiting).
func (s SessionState) Tick() int { return s.Wave.Tick }

// NewInitialState builds the deterministic initial state of a new village and
// session. Identifiers are supplied by the caller: nothing is generated here, no
// clock is read and no randomness is used, so identical inputs always produce an
// identical state.
func NewInitialState(villageID VillageID, sessionID SessionID, mode SourceMode) (SessionState, error) {
	if !villageID.Valid() {
		return SessionState{}, newError("village_id", errUUIDFormat)
	}
	if !sessionID.Valid() {
		return SessionState{}, newError("session_id", errUUIDFormat)
	}
	if !mode.Valid() {
		return SessionState{}, newError("mode", errSourceModeInvalid)
	}

	wallMax, err := WallMax(InitialWallTier)
	if err != nil {
		return SessionState{}, err
	}

	state := SessionState{
		Revision:  CurrentRevision(),
		ID:        sessionID,
		VillageID: villageID,
		Mode:      mode,
		Lifecycle: LifecycleWaiting,
		// Empty, non-nil slices keep JSON output stable and make it obvious that
		// no participant exists yet.
		PauseReasons: []PauseReason{},
		Village: VillageState{
			ID:               villageID,
			Level:            InitialVillageLevel,
			WallTier:         InitialWallTier,
			WallHP:           wallMax,
			TownhallHP:       InitialTownhallHP,
			Resources:        Resources{Wood: InitialWood, Stone: InitialStone, Food: InitialFood},
			ConstructionWork: InitialConstructionWork,
			TotalWins:        InitialTotalWins,
		},
		Wave: WaveState{
			Number:  InitialWaveNo,
			Tick:    WaitingTick,
			Phase:   PhaseWaiting,
			Pools:   []EnemyPool{},
			Boss:    BossPool{},
			Vote:    VoteState{},
			Ballots: []Ballot{},
			Result:  WaveResultPending,
		},
		Members:           []WaveMember{},
		Pending:           []PendingMember{},
		XP:                []PlayerXP{},
		Difficulty:        Difficulty{Level: 1, PopulationAnchor: 0},
		ConsecutiveWins:   0,
		ConsecutiveLosses: 0,
		PrivacyEpoch:      0,
		InputSequence:     0,
	}

	if err := state.Validate(); err != nil {
		return SessionState{}, fmt.Errorf("domain: initial state failed validation: %w", err)
	}
	return state, nil
}
