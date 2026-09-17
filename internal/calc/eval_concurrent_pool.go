package calc

import (
	"math"
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

// GetMaxTreeDepth calculates the optimal tree depth for recursive binary splitting
// based on available CPU cores.
func GetMaxTreeDepth() int {
	workers := GetOptimalWorkers()
	if workers <= 1 {
		return 0
	}
	depth := int(math.Ceil(math.Log2(float64(workers)))) + 1
	if depth < 1 {
		return 1
	}
	return depth
}
