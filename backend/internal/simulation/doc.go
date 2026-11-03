// Package simulation will own the pure 10 Hz reducer, the deterministic tick
// order and canonical state hashing described by docs/plan/03_SIMULATION.md.
//
// It is intentionally empty in M01 Phase A. M02 owns the implementation; the
// reducer must not perform database, network, clock or OS I/O.
package simulation
