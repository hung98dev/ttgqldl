package domain

import (
	"errors"
	"fmt"
	"strings"
)

// FieldError reports one violated invariant. It names the field and the rule so
// a caller can act on it instead of parsing prose.
type FieldError struct {
	Field string
	Rule  string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("domain: %s: %s", e.Field, e.Rule)
}

func newError(field, rule string) error {
	return &FieldError{Field: field, Rule: rule}
}

// ValidationError aggregates every broken invariant found in one state.
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "domain: invalid state: " + strings.Join(e.Problems, "; ")
}

// Is lets callers test for a validation failure with errors.Is.
func (e *ValidationError) Is(target error) bool {
	var other *ValidationError
	return errors.As(target, &other)
}

func (e *ValidationError) add(field, rule string) {
	e.Problems = append(e.Problems, fmt.Sprintf("%s: %s", field, rule))
}

func (e *ValidationError) addf(field, format string, args ...any) {
	e.Problems = append(e.Problems, field+": "+fmt.Sprintf(format, args...))
}

// MarshalJSON keeps validation errors machine-readable if a caller serialises
// them into a diagnostic payload.
func (e *ValidationError) MarshalJSON() ([]byte, error) {
	return []byte(`{"problems":["` + strings.Join(e.Problems, `","`) + `"]}`), nil
}

const (
	errWallTierRange       = "must be within 1..20"
	errVillageLevelRange   = "must be within 1..50"
	errTownhallRange       = "must be within 0..500"
	errResourceRange       = "must be within 0..1000000"
	errXPOutOfRange        = "must be within 0..300"
	errConsecutiveWins     = "must be within 0..10"
	errConsecutiveLosses   = "must be within 0..3"
	errNegativeHP          = "must not be negative"
	errWaveNumberRange     = "must be at least 1"
	errTickRange           = "must be within -1..3000"
	errActorKeyFormat      = "must be 64 lowercase hex characters"
	errUUIDFormat          = "must be a lowercase UUID"
	errRoleInvalid         = "must be builder, guard or fighter"
	errLifecycleInvalid    = "must be a documented lifecycle value"
	errPhaseInvalid        = "must be a documented phase value"
	errPhaseTickMismatch   = "does not match the phase derived from the wave tick"
	errPhaseRoleMismatch   = "does not match the phase derived from the wave tick"
	errVoteOptionInvalid   = "must be A, B, C or empty"
	errDuplicateActor      = "duplicate actor key in canonical collection"
	errMemberOrder         = "canonical members must be sorted by actor key"
	errPoolOrder           = "canonical pools must be ordered by lane then archetype"
	errBossBeforeSpawn     = "boss pool exists before the boss spawn tick"
	errBossWaveMismatch    = "boss pool exists on a wave that is not a boss wave"
	errBossHP              = "boss HP must be positive and no greater than its maximum"
	errOrdinaryPoolBoss    = "ordinary enemy pools must not contain the boss archetype"
	errLaneInvalid         = "must be west, center or east"
	errArchetypeInvalid    = "must be runner, grunt, brute or boss"
	errJoinTickRange       = "joined tick must be within 0..2999"
	errChangeTickRange     = "last change tick must be -1 or within the member's wave"
	errTargetWaveRange     = "pending target wave must be greater than the current wave"
	errVillageIDMismatch   = "does not match the session village"
	errWaveNumberMismatch  = "does not match the session wave number"
	errRevisionSchema      = "must equal 1"
	errRevisionRules       = "must equal ttg-rules-v1"
	errPauseReasonInvalid  = "must be a documented pause reason"
	errDefeatedPhase       = "defeat can only be recorded during the finale or results"
	errResultBeforeResults = "a wave result can only be recorded in the results phase"
	errResultDefeatFlag    = "a defeat result requires the defeated flag"
	errSourceModeInvalid   = "must be mock, replay or authorized_live"
)
