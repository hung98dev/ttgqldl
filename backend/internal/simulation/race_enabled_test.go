//go:build race

package simulation

// raceDetectorEnabled reports that the test binary was built with the race
// detector.
//
// Runtime policy (measured on the audit host, not assumed):
//   - QA-BAL-05 runs the full 3,000-tick production pipeline for 10,000
//     participants across several candidates; under race instrumentation that is
//     the single most expensive test in the repository, so the non-race unit gate
//     executes the curve in full.
//   - The golden matrix skips exactly one of its 25 cases under race: the
//     10,000-participant full wave (C015). The other 24 curated cases, including
//     the boss and vote vectors, still run under -race.
//   - All four replay scenarios (A/B/C/D) run under race. The common driver no
//     longer hashes every step, which brought Scenario A from ~65 s to a few
//     seconds in the non-race build.
//
// A test excluded in one configuration is always mandatory in the other; no test
// is skipped in every configuration.
const raceDetectorEnabled = true
