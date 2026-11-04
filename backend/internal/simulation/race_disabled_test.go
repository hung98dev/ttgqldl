//go:build !race

package simulation

// raceDetectorEnabled is false in the normal test binary, where the full
// QA-BAL-05 curve executes.
const raceDetectorEnabled = false
