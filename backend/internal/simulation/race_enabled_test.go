//go:build race

package simulation

// raceDetectorEnabled reports that the test binary was built with the race
// detector. The QA-BAL-05 curve intentionally runs the full 3,000-tick production
// pipeline for 10,000 participants, which is computationally prohibitive under
// race instrumentation; the non-race unit-test gate executes it instead.
const raceDetectorEnabled = true
