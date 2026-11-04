package domain

// Voting constants from 02_GAME_RULES.md: voting exists only on even waves,
// opens at tick 800, accepts effective ticks 800..1199 and closes at 1200.
const (
	VoteWoodCost int64 = 50
	VoteFoodCost int64 = 50
)

// VoteAvailable reports whether the wave has a vote (even-numbered waves only).
func VoteAvailable(waveNo int64) bool { return waveNo > 0 && waveNo%2 == 0 }

// BallotTickEligible reports whether an effective tick is inside the open window.
func BallotTickEligible(tick int) bool {
	return tick >= VoteOpenTick && tick < VoteCloseTick
}

// VoteCost returns the resource charge for a selected option.
func VoteCost(option VoteOption) (wood, food int64) {
	switch option {
	case VoteOptionA:
		return VoteWoodCost, 0
	case VoteOptionB:
		return 0, VoteFoodCost
	default:
		return 0, 0
	}
}

// VoteAffordable reports whether the village can pay for an option.
func VoteAffordable(option VoteOption, wood, food int64) bool {
	switch option {
	case VoteOptionA:
		return wood >= VoteWoodCost
	case VoteOptionB:
		return food >= VoteFoodCost
	case VoteOptionC:
		return true
	default:
		return false
	}
}

// VoteGuardPercent returns the guard-block percentage for a selection:
// A raises it to 115%, C lowers it to 90%, otherwise 100%.
func VoteGuardPercent(option VoteOption) int64 {
	switch option {
	case VoteOptionA:
		return 115
	case VoteOptionC:
		return 90
	default:
		return 100
	}
}

// VoteAttackPercent returns the attack percentage for a selection: only B raises
// the total attack to 115%.
func VoteAttackPercent(option VoteOption) int64 {
	if option == VoteOptionB {
		return 115
	}
	return 100
}

// ApplyVotePercent returns floor(value*percent/100) with exact integer math.
func ApplyVotePercent(value, percent int64) (int64, error) {
	return SafeRatio(value, 100, percent)
}

// LossPenaltyAfterC is the W0207 contract: a selected C halves the computed
// resource-loss penalties. It never halves resources themselves.
func LossPenaltyAfterC(penalty int64) (int64, error) {
	return SafeRatio(penalty, 2, 1)
}

// votePriority is the deterministic tie order A > B > C.
func votePriority(option VoteOption) int {
	switch option {
	case VoteOptionA:
		return 0
	case VoteOptionB:
		return 1
	default:
		return 2
	}
}

// SelectVote implements the canonical deterministic selection: highest count
// first, tie priority A > B > C, first affordable option. With no ballots, A is
// selected when wood >= 50, otherwise C.
func SelectVote(counts VoteCounts, wood, food int64) (VoteOption, error) {
	if wood < 0 || food < 0 {
		return VoteOptionNone, ErrNegative
	}
	options := []VoteOption{VoteOptionA, VoteOptionB, VoteOptionC}
	if counts.Total() == 0 {
		if VoteAffordable(VoteOptionA, wood, food) {
			return VoteOptionA, nil
		}
		return VoteOptionC, nil
	}
	// Deterministic sort: count descending, then A > B > C.
	for i := 0; i < len(options); i++ {
		for j := i + 1; j < len(options); j++ {
			left, right := options[i], options[j]
			leftCount, rightCount := counts.count(left), counts.count(right)
			if rightCount > leftCount ||
				(rightCount == leftCount && votePriority(right) < votePriority(left)) {
				options[i], options[j] = options[j], options[i]
			}
		}
	}
	for _, option := range options {
		if VoteAffordable(option, wood, food) {
			return option, nil
		}
	}
	return VoteOptionC, nil
}

func (c VoteCounts) count(option VoteOption) int64 {
	switch option {
	case VoteOptionA:
		return c.A
	case VoteOptionB:
		return c.B
	default:
		return c.C
	}
}
