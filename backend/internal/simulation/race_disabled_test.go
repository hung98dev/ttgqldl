//go:build !race

package simulation

// raceDetectorEnabled is false in the normal test binary. The full golden matrix
// (including the 10,000-participant case), the full QA-BAL-05 curve and all four
// replay scenarios execute here.
const raceDetectorEnabled = false
