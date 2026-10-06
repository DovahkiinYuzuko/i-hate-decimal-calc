package calc

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// DefaultBatchCutoff is the minimum slice length to justify spawning goroutines.
const DefaultBatchCutoff int = 4

// GetCadLiftingCutoff returns the minimum number of parent cells required to justify parallel execution.
// It scales dynamically as GetOptimalWorkers() * 2 to avoid scheduling overhead on low-core machines.
func GetCadLiftingCutoff() int {
	w := GetOptimalWorkers()
	if w <= 1 {
		return 0
	}
	return w * 2
}

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
		return nil, fmt.Errorf("%s", i18n.T("concurrent.err_concurrent_task_panicked", panicVal))
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

// ParallelBatchSearch concurrently searches items using searchFunc.
// If searchFunc returns (res, true, nil), the search terminates immediately,
// cancelling remaining workers via context and atomic flags (Early-Exit).
// If found, it returns (res, true, nil).
// If no element satisfies the predicate, it returns (nil, false, nil).
// If an error or panic occurs, it terminates and returns the error.
func ParallelBatchSearch[T any, R any](items []T, searchFunc func(T) (*R, bool, error), cutoff int) (*R, bool, error) {
	n := len(items)
	if n == 0 {
		return nil, false, nil
	}

	if cutoff <= 0 {
		cutoff = DefaultBatchCutoff
	}

	// Sequential fast-path
	if n < cutoff || GetOptimalWorkers() <= 1 {
		for _, item := range items {
			res, found, err := searchFunc(item)
			if err != nil {
				return nil, false, err
			}
			if found {
				return res, true, nil
			}
		}
		return nil, false, nil
	}

	fsm := NewTaskFSM()
	if err := fsm.Transition(StateRunning); err != nil {
		return nil, false, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workers := GetOptimalWorkers()
	sem := make(chan struct{}, workers)

	var wg sync.WaitGroup
	var foundMu sync.Mutex
	var foundVal *R
	var hasFound atomic.Bool

	var errMu sync.Mutex
	var firstErr error
	var hasErr atomic.Bool
	var panicVal any
	var hasPanic atomic.Bool

searchLoop:
	for _, item := range items {
		if hasFound.Load() || hasErr.Load() || hasPanic.Load() {
			break
		}

		select {
		case <-ctx.Done():
			break searchLoop
		case sem <- struct{}{}:
		}

		// Double-check after acquiring semaphore
		if hasFound.Load() || hasErr.Load() || hasPanic.Load() {
			<-sem
			break
		}

		wg.Add(1)
		go func(val T) {
			defer func() {
				<-sem
				wg.Done()
				if r := recover(); r != nil {
					errMu.Lock()
					if panicVal == nil {
						panicVal = r
						hasPanic.Store(true)
						cancel()
					}
					errMu.Unlock()
				}
			}()

			if hasFound.Load() || hasErr.Load() || hasPanic.Load() {
				return
			}

			res, found, err := searchFunc(val)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
					hasErr.Store(true)
					cancel()
				}
				errMu.Unlock()
				return
			}

			if found {
				foundMu.Lock()
				if !hasFound.Load() {
					foundVal = res
					hasFound.Store(true)
					cancel()
				}
				foundMu.Unlock()
			}
		}(item)
	}

	wg.Wait()

	if panicVal != nil {
		_ = fsm.Transition(StateFailed)
		return nil, false, fmt.Errorf("%s", i18n.T("concurrent.err_concurrent_task_panicked", panicVal))
	}
	if firstErr != nil {
		_ = fsm.Transition(StateFailed)
		return nil, false, firstErr
	}

	if hasFound.Load() {
		_ = fsm.Transition(StateCompleted)
		return foundVal, true, nil
	}

	_ = fsm.Transition(StateCompleted)
	return nil, false, nil
}

