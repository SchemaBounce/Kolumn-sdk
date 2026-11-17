//go:build race

package logging

// raceEnabled reports whether the test binary was built with -race. The race
// detector slows execution by an unpredictable factor, so wall-clock
// assertions are skipped when it is on.
const raceEnabled = true
