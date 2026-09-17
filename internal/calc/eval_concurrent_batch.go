package calc

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// DefaultBatchCutoff is the minimum slice length to justify spawning goroutines.
const DefaultBatchCutoff int = 4

// ParallelBatchMap applies workerFunc to each element in items concurrently,
// preserving the exact original ordering of the results.
// If len(items) < cutoff or optimal workers == 1, it executes sequentially.
// It manages the execution lifecycle using TaskFSM.
func ParallelBatchMap[T any, R any](items []T, workerFunc func(T) (R, error), cutoff int) ([]R, error) {
	n := len(items)
	if n == 0 {
		return nil, nil
	}

	if cutoff <= 0 {
		cutoff = DefaultBatchCutoff
	}

	// Sequential fast-path
	if n < cutoff || GetOptimalWorkers() <= 1 {
		results := make([]R, n)
		for i, item := range items {
			res, err := workerFunc(item)
			if err != nil {
				return nil, err
			}
			results[i] = res
		}
		return results, nil
	}

	fsm := NewTaskFSM()
	if err := fsm.Transition(StateRunning); err != nil {
		return nil, err
	}

	results := make([]R, n)
	workers := GetOptimalWorkers()
	sem := make(chan struct{}, workers)

	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error
	var hasErr atomic.Bool
	var panicVal any
	var hasPanic atomic.Bool

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, val T) {
			defer func() {
				<-sem
				wg.Done()
				if r := recover(); r != nil {
					errMu.Lock()
					if panicVal == nil {
						panicVal = r
						hasPanic.Store(true)
					}
					errMu.Unlock()
				}
			}()

			// Early bail-out if an error or panic has occurred
			if hasErr.Load() || hasPanic.Load() {
				return
			}

			res, err := workerFunc(val)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
					hasErr.Store(true)
				}
				errMu.Unlock()
				return
			}
			results[idx] = res
		}(i, item)
	}

	wg.Wait()

	if panicVal != nil {
		_ = fsm.Transition(StateFailed)
		return nil, fmt.Errorf("concurrent task panicked: %v", panicVal)
	}
	if firstErr != nil {
		_ = fsm.Transition(StateFailed)
		return nil, firstErr
	}

	if err := fsm.Transition(StateCompleted); err != nil {
		return nil, err
	}

	return results, nil
}
