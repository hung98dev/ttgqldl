package domain

import "fmt"

// Validate checks every persistent village invariant. Validation never repairs:
// an invalid state is reported, never mutated back into shape.
func (v VillageState) Validate() *ValidationError {
	errs := &ValidationError{}
	if !v.ID.Valid() {
		errs.add("village.id", errUUIDFormat)
	}
	if v.Level < 1 || v.Level > MaxVillageLevel {
		errs.add("village.level", errVillageLevelRange)
	}
	if v.WallTier < 1 || v.WallTier > MaxWallTier {
		errs.add("village.wall_tier", errWallTierRange)
	} else {
		maxWall, err := WallMax(v.WallTier)
		if err != nil {
			errs.add("village.wall_tier", err.Error())
		} else if v.WallHP < 0 || v.WallHP > maxWall {
			errs.addf("village.wall_hp", "must be within 0..%d for tier %d", maxWall, v.WallTier)
		}
	}
	if v.TownhallHP < 0 || v.TownhallHP > TownhallMax {
		errs.add("village.townhall_hp", errTownhallRange)
	}
	if resources := v.Resources.Validate(); resources != nil {
		errs.Problems = append(errs.Problems, resources.Problems...)
	}
	if v.ConstructionWork < 0 {
		errs.add("village.construction_work", "must not be negative")
	}
	if v.TotalWins < 0 {
		errs.add("village.total_wins", "must not be negative")
	}
	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}

// Validate checks the wave state against its derived phase and enemy invariants.
func (w WaveState) Validate() *ValidationError {
	errs := &ValidationError{}
	if w.Number < 1 {
		errs.add("wave.number", errWaveNumberRange)
	}
	class, err := ClassifyTick(w.Tick)
	if err != nil {
		errs.add("wave.tick", errTickRange)
	} else {
		switch class {
		case TickClassRollover:
			// Tick 3000 is a control transition; the wave is in its results phase.
			if w.Phase != PhaseResults {
				errs.add("wave.phase", errPhaseTickMismatch)
			}
		case TickClassWaiting:
			if w.Phase != PhaseWaiting {
				errs.add("wave.phase", errPhaseTickMismatch)
			}
		default:
			expected, phaseErr := PhaseForTick(w.Tick)
			if phaseErr != nil || expected != w.Phase {
				errs.add("wave.phase", errPhaseTickMismatch)
			}
		}
	}
	if !w.Phase.Valid() {
		errs.add("wave.phase", errPhaseInvalid)
	}
	if pools := ValidatePools(w.Pools, w.Boss, w.Number, w.Tick); pools != nil {
		errs.Problems = append(errs.Problems, pools.Problems...)
	}
	if vote := ValidateVoteState(w.Vote, w.Tick); vote != nil {
		errs.Problems = append(errs.Problems, vote.Problems...)
	}
	if ballots := ValidateBallots(w.Ballots, w.Tick); ballots != nil {
		errs.Problems = append(errs.Problems, ballots.Problems...)
	}
	if !w.Result.Valid() {
		errs.add("wave.result", "must be pending, victory or defeat")
	}
	// Combat starts at tick 1800 inside ASSAULT, so a defeated village is a
	// legitimate state from that boundary onward. Defeat during PREP is not.
	if w.Defeated {
		if w.Tick < CombatStartTick {
			errs.add("wave.defeated", errDefeatedPhase)
		}
		if w.Phase != PhaseAssault && w.Phase != PhaseFinale && w.Phase != PhaseResults {
			errs.add("wave.defeated", errDefeatedPhase)
		}
	}
	if w.Result != WaveResultPending && w.Phase != PhaseResults {
		errs.add("wave.result", errResultBeforeResults)
	}
	// A wave can be lost with the town hall still standing: settlement decides
	// victory conditions, while the defeated flag only records townhall HP 0.
	if w.Result == WaveResultVictory && w.Defeated {
		errs.add("wave.result", "a defeated wave cannot also be a victory")
	}
	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}

// Validate checks the whole authoritative session state. Every broken invariant
// is collected so one call reports the complete picture.
func (s SessionState) Validate() error {
	errs := &ValidationError{}
	appendProblems := func(v *ValidationError) {
		if v != nil {
			errs.Problems = append(errs.Problems, v.Problems...)
		}
	}

	if !s.Revision.Valid() {
		if s.Revision.SchemaVersion != CanonicalSchemaVersion {
			errs.add("revision.schema_version", errRevisionSchema)
		}
		if s.Revision.RulesVersion != RulesVersion {
			errs.add("revision.rules_version", errRevisionRules)
		}
	}
	if !s.ID.Valid() {
		errs.add("session.id", errUUIDFormat)
	}
	if !s.VillageID.Valid() {
		errs.add("session.village_id", errUUIDFormat)
	}
	if s.Village.ID != s.VillageID {
		errs.add("village.id", errVillageIDMismatch)
	}
	if !s.Mode.Valid() {
		errs.add("session.mode", errSourceModeInvalid)
	}
	if !s.Lifecycle.Valid() {
		errs.add("session.lifecycle", errLifecycleInvalid)
	}

	seen := map[PauseReason]bool{}
	for index, reason := range s.PauseReasons {
		field := "session.pause_reasons[" + itoa(index) + "]"
		if !reason.Valid() {
			errs.add(field, errPauseReasonInvalid)
		}
		if seen[reason] {
			errs.add(field, "duplicate pause reason")
		}
		seen[reason] = true
	}
	switch s.Lifecycle {
	case LifecyclePaused:
		if len(s.PauseReasons) == 0 {
			errs.add("session.pause_reasons", "a paused session needs at least one explicit reason")
		}
	case LifecycleSourceDisconnected:
		if !seen[PauseSourceDisconnected] {
			errs.add("session.pause_reasons", "source_disconnected requires the source_disconnected reason")
		}
	case LifecycleStorageUnavailable:
		if !seen[PauseStorageUnavailable] {
			errs.add("session.pause_reasons", "storage_unavailable requires the storage_unavailable reason")
		}
	case LifecycleWaiting, LifecycleRunning:
		if len(s.PauseReasons) > 0 {
			errs.add("session.pause_reasons", "an active pause reason requires a paused or blocked lifecycle")
		}
	case LifecycleStopped:
		if len(s.PauseReasons) > 0 {
			errs.add("session.pause_reasons", "a stopped session cannot carry pause reasons")
		}
	}

	appendProblems(s.Village.Validate())
	appendProblems(s.Wave.Validate())
	appendProblems(ValidateMembers(s.Members, s.Wave.Number))
	appendProblems(ValidatePending(s.Pending, s.Wave.Number))
	appendProblems(ValidateXP(s.XP))

	if lock := validateEnemyLock(s.Wave, s.Difficulty); lock != nil {
		errs.Problems = append(errs.Problems, lock.Problems...)
	}
	if !s.Difficulty.Locked && (s.Difficulty.Level < 1 || s.Difficulty.Level > 10) {
		errs.add("session.difficulty.level", "must be within 1..10")
	}
	if !s.Difficulty.Locked && (s.Difficulty.PopulationAnchor < 0 || s.Difficulty.PopulationAnchor > MaxPopulationAnchor) {
		errs.add("session.difficulty.population_anchor", "must be within 0..10000")
	}
	if s.ConsecutiveWins < 0 || s.ConsecutiveWins > MaxConsecutiveWins {
		errs.add("session.consecutive_wins", errConsecutiveWins)
	}
	if s.ConsecutiveLosses < 0 || s.ConsecutiveLosses > MaxConsecutiveLosses {
		errs.add("session.consecutive_losses", errConsecutiveLosses)
	}
	if s.PrivacyEpoch < 0 {
		errs.add("session.privacy_epoch", "must not be negative")
	}
	if s.InputSequence < 0 {
		errs.add("session.input_sequence", "must not be negative")
	}

	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}

// Valid reports whether the state satisfies every invariant.
func (s SessionState) Valid() bool { return s.Validate() == nil }

// Describe returns a short, allocation-light summary for diagnostics.
func (s SessionState) Describe() string {
	return fmt.Sprintf("session=%s village=%s wave=%d tick=%d lifecycle=%s phase=%s",
		s.ID, s.VillageID, s.Wave.Number, s.Wave.Tick, s.Lifecycle, s.Wave.Phase)
}
