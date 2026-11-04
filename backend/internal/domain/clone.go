package domain

// Clone returns a deep copy of the session state. The simulation reducer will
// receive a state, return a new one and emit events; callers must never share a
// mutable backing slice with the authoritative state, so every constructor,
// reducer boundary and test can rely on Clone.
func (s SessionState) Clone() SessionState {
	clone := s

	clone.PauseReasons = append([]PauseReason(nil), s.PauseReasons...)
	clone.Members = append([]WaveMember(nil), s.Members...)
	clone.Pending = append([]PendingMember(nil), s.Pending...)
	clone.XP = append([]PlayerXP(nil), s.XP...)

	clone.Wave = s.Wave.Clone()
	return clone
}

// Clone deep-copies the wave's mutable collections.
func (w WaveState) Clone() WaveState {
	clone := w
	clone.Pools = append([]EnemyPool(nil), w.Pools...)
	clone.Ballots = append([]Ballot(nil), w.Ballots...)
	return clone
}

// Clone returns a copy of the village value (all fields are scalars, so this is
// a value copy made explicit at call sites).
func (v VillageState) Clone() VillageState { return v }
