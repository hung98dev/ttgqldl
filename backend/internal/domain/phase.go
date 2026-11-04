package domain

import "fmt"

// ClassifyTick separates the waiting sentinel, gameplay ticks and the single
// control rollover boundary. Tick 3000 is never a gameplay resolution.
func ClassifyTick(tick int) (TickClass, error) {
	switch {
	case tick == WaitingTick:
		return TickClassWaiting, nil
	case tick >= 0 && tick < WaveTicks:
		return TickClassGameplay, nil
	case tick == RolloverTick:
		return TickClassRollover, nil
	default:
		return 0, newError("wave_tick", fmt.Sprintf("must be -1, 0..2999 or 3000, got %d", tick))
	}
}

// PhaseForTick derives the phase from the local wave tick only.
//
//	-1        waiting
//	0..1599   prep
//	1600..2599 assault
//	2600..2799 finale
//	2800..2999 results
//
// Tick 3000 is the control rollover boundary and is rejected here: callers must
// use ClassifyTick and perform the single control transition instead.
func PhaseForTick(tick int) (Phase, error) {
	class, err := ClassifyTick(tick)
	if err != nil {
		return 0, err
	}
	switch class {
	case TickClassRollover:
		return 0, newError("wave_tick", "3000 is a control rollover boundary, not a gameplay tick")
	case TickClassWaiting:
		return PhaseWaiting, nil
	}
	switch {
	case tick < PrepEndTick:
		return PhasePrep, nil
	case tick < AssaultEndTick:
		return PhaseAssault, nil
	case tick < FinaleEndTick:
		return PhaseFinale, nil
	default:
		return PhaseResults, nil
	}
}

// IsRolloverTick reports the single control transition tick.
func IsRolloverTick(tick int) bool { return tick == RolloverTick }

// IsGameplayTick reports whether the tick may resolve gameplay.
func IsGameplayTick(tick int) bool { return tick >= 0 && tick < WaveTicks }
