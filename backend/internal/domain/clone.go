package domain

// cloneSlice preserves nil-ness: an empty but non-nil canonical collection stays
// empty and non-nil, so a cloned state serialises identically to its source.
func cloneSlice[T any](in []T) []T {
	if in == nil {
		return nil
	}
	out := make([]T, len(in))
	copy(out, in)
	return out
}

// Clone returns a deep copy of the session state. The simulation reducer will
// receive a state, return a new one and emit events; callers must never share a
// mutable backing slice with the authoritative state, so every constructor,
// reducer boundary and test can rely on Clone.
func (s SessionState) Clone() SessionState {
	clone := s

	clone.PauseReasons = cloneSlice(s.PauseReasons)
	clone.Members = cloneSlice(s.Members)
	for index := range clone.Members {
		if clone.Members[index].LastChangeTick != nil {
			tick := *clone.Members[index].LastChangeTick
			clone.Members[index].LastChangeTick = &tick
		}
	}
	clone.Pending = cloneSlice(s.Pending)
	clone.XP = cloneSlice(s.XP)

	clone.Wave = s.Wave.Clone()
	return clone
}

// Clone deep-copies the wave's mutable collections.
func (w WaveState) Clone() WaveState {
	clone := w
	clone.Pools = cloneSlice(w.Pools)
	clone.Ballots = cloneSlice(w.Ballots)
	return clone
}

// Clone returns a copy of the village value (all fields are scalars, so this is
// a value copy made explicit at call sites).
func (v VillageState) Clone() VillageState { return v }
