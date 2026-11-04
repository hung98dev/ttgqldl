package domain

import "sort"

// WaveMember is one participant of the current wave and mirrors the canonical
// wave_members columns: joined_tick, nullable last_change_tick, first_switch_used
// and xp_awarded. LastChangeTick is nil until the first accepted role change, so
// "no change yet" is represented explicitly rather than with a magic number.
type WaveMember struct {
	ActorKey        ActorKey
	Role            Role
	JoinedTick      int
	LastChangeTick  *int
	FirstSwitchUsed bool
	XPAwarded       bool
}

// PendingMember is a registration queued for the next wave during RESULTS.
type PendingMember struct {
	ActorKey   ActorKey
	Role       Role
	TargetWave int64
}

// PlayerXP is the per-session XP of one participant.
type PlayerXP struct {
	ActorKey ActorKey
	XP       int64
}

// Ballot is one vote with its acceptance tick.
type Ballot struct {
	ActorKey ActorKey
	Option   VoteOption
	Tick     int
}

// VoteCounts are the three tally values.
type VoteCounts struct {
	A int64
	B int64
	C int64
}

// Total is the sum of the three counters.
func (c VoteCounts) Total() int64 { return c.A + c.B + c.C }

// VoteState is the vote window for the current wave.
type VoteState struct {
	Open     bool
	Selected VoteOption
	Counts   VoteCounts
}

// SortWaveMembers orders members by ActorKey, the canonical order.
func SortWaveMembers(members []WaveMember) {
	sort.SliceStable(members, func(i, j int) bool { return members[i].ActorKey < members[j].ActorKey })
}

// SortPendingMembers orders queued members by ActorKey.
func SortPendingMembers(pending []PendingMember) {
	sort.SliceStable(pending, func(i, j int) bool { return pending[i].ActorKey < pending[j].ActorKey })
}

// SortBallots orders ballots by ActorKey.
func SortBallots(ballots []Ballot) {
	sort.SliceStable(ballots, func(i, j int) bool { return ballots[i].ActorKey < ballots[j].ActorKey })
}

// SortPlayerXP orders XP rows by ActorKey.
func SortPlayerXP(rows []PlayerXP) {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ActorKey < rows[j].ActorKey })
}

// ValidateMembers checks canonical ordering, unique identities and tick ranges
// for the current wave.
func ValidateMembers(members []WaveMember, waveNo int64) *ValidationError {
	errs := &ValidationError{}
	var previous ActorKey
	for index, member := range members {
		field := "members[" + itoa(index) + "]"
		if !member.ActorKey.Valid() {
			errs.add(field+".actor_key", errActorKeyFormat)
		}
		if index > 0 && member.ActorKey <= previous {
			if member.ActorKey == previous {
				errs.add(field+".actor_key", errDuplicateActor)
			} else {
				errs.add(field+".actor_key", errMemberOrder)
			}
		}
		previous = member.ActorKey
		if !member.Role.Valid() {
			errs.add(field+".role", errRoleInvalid)
		}
		if member.JoinedTick < 0 || member.JoinedTick >= WaveTicks {
			errs.add(field+".joined_tick", errJoinTickRange)
		}
		switch {
		case member.LastChangeTick == nil:
			// No accepted change yet: the free first switch is still available.
			if member.FirstSwitchUsed {
				errs.add(field+".first_switch_used", "cannot be used before the first accepted role change")
			}
		default:
			if !member.FirstSwitchUsed {
				errs.add(field+".first_switch_used", "must be true once a role change is recorded")
			}
			if *member.LastChangeTick < member.JoinedTick || *member.LastChangeTick >= WaveTicks {
				errs.add(field+".last_change_tick", errChangeTickRange)
			}
		}
		if waveNo < 1 {
			errs.add(field, errWaveNumberRange)
		}
	}
	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}

// ValidatePending checks queued registrations: unique, canonical order, future
// target wave only.
func ValidatePending(pending []PendingMember, currentWave int64) *ValidationError {
	errs := &ValidationError{}
	var previous ActorKey
	for index, entry := range pending {
		field := "pending[" + itoa(index) + "]"
		if !entry.ActorKey.Valid() {
			errs.add(field+".actor_key", errActorKeyFormat)
		}
		if index > 0 && entry.ActorKey <= previous {
			if entry.ActorKey == previous {
				errs.add(field+".actor_key", errDuplicateActor)
			} else {
				errs.add(field+".actor_key", errMemberOrder)
			}
		}
		previous = entry.ActorKey
		if !entry.Role.Valid() {
			errs.add(field+".role", errRoleInvalid)
		}
		// A NEXT-wave registration queued during RESULTS always targets exactly
		// current+1: a distant future wave would silently drop the registration.
		if entry.TargetWave != currentWave+1 {
			errs.add(field+".target_wave", errTargetWaveRange)
		}
	}
	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}

// ValidateXP checks per-player session XP: unique, ordered, within the cap.
func ValidateXP(rows []PlayerXP) *ValidationError {
	errs := &ValidationError{}
	var previous ActorKey
	for index, entry := range rows {
		field := "xp[" + itoa(index) + "]"
		if !entry.ActorKey.Valid() {
			errs.add(field+".actor_key", errActorKeyFormat)
		}
		if index > 0 && entry.ActorKey <= previous {
			if entry.ActorKey == previous {
				errs.add(field+".actor_key", errDuplicateActor)
			} else {
				errs.add(field+".actor_key", errMemberOrder)
			}
		}
		previous = entry.ActorKey
		if entry.XP < 0 || entry.XP > CapSessionXP {
			errs.add(field+".xp", errXPOutOfRange)
		}
	}
	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}

// ValidateBallots checks unique voters, canonical order, valid option and a tick
// that cannot be in the future relative to the wave tick.
func ValidateBallots(ballots []Ballot, waveTick int) *ValidationError {
	errs := &ValidationError{}
	var previous ActorKey
	for index, ballot := range ballots {
		field := "ballots[" + itoa(index) + "]"
		if !ballot.ActorKey.Valid() {
			errs.add(field+".actor_key", errActorKeyFormat)
		}
		if index > 0 && ballot.ActorKey <= previous {
			if ballot.ActorKey == previous {
				errs.add(field+".actor_key", errDuplicateActor)
			} else {
				errs.add(field+".actor_key", errMemberOrder)
			}
		}
		previous = ballot.ActorKey
		if !ballot.Option.Valid() || ballot.Option == VoteOptionNone {
			errs.add(field+".option", errVoteOptionInvalid)
		}
		if ballot.Tick < 0 || ballot.Tick >= VoteCloseTick {
			errs.add(field+".tick", "must be inside the voting window [0,1200)")
		}
		if waveTick >= 0 && ballot.Tick > waveTick {
			errs.add(field+".tick", "must not be in the future")
		}
	}
	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}

// ValidateVoteState checks the vote window and its counters. The window itself
// is validated against the wave tick by the caller.
func ValidateVoteState(vote VoteState, waveTick int) *ValidationError {
	errs := &ValidationError{}
	if !vote.Selected.Valid() {
		errs.add("vote.selected", errVoteOptionInvalid)
	}
	if vote.Open && vote.Selected != VoteOptionNone {
		errs.add("vote.selected", "an open vote cannot have a selected option")
	}
	if vote.Open && waveTick >= VoteCloseTick {
		errs.add("vote.open", "voting closes at tick 1200")
	}
	if vote.Counts.A < 0 || vote.Counts.B < 0 || vote.Counts.C < 0 {
		errs.add("vote.counts", "must not be negative")
	}
	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}
