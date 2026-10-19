package calc

import (
	"runtime"
)

// GetOptimalWorkers returns the maximum recommended number of concurrent workers.
// It reflects runtime.GOMAXPROCS(0) and guarantees at least 1 worker.
func GetOptimalWorkers() int {
	w := runtime.GOMAXPROCS(0)
	if w < 1 {
		return 1
	}
	return w
}
