package runner

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// RunPool executes test cases using a worker pool with concurrency determined by jobs.
// If jobs <= 0, runtime.NumCPU() is used.
func RunPool(ctx context.Context, defaultBinary string, verifierBinary string, cases []TestCase, jobs int, onResult func(res TestResult, completed int, total int)) []TestResult {
	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}
	if jobs <= 0 {
		jobs = 1
	}

	total := len(cases)
	results := make([]TestResult, total)
	caseChan := make(chan struct {
		idx int
		tc  TestCase
	}, jobs*2)

	var completedCount int32
	var wg sync.WaitGroup

	for w := 0; w < jobs; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range caseChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				res := ExecuteCase(ctx, defaultBinary, verifierBinary, item.tc)
				results[item.idx] = res

				c := atomic.AddInt32(&completedCount, 1)
				if onResult != nil {
					onResult(res, int(c), total)
				}
			}
		}()
	}

	// Feed cases
	for i, tc := range cases {
		caseChan <- struct {
			idx int
			tc  TestCase
		}{idx: i, tc: tc}
	}
	close(caseChan)

	wg.Wait()
	return results
}

// FormatProgress returns a formatted string for terminal display.
func FormatProgress(res TestResult, completed int, total int) string {
	percent := float64(completed) / float64(total) * 100
	if res.Passed {
		return fmt.Sprintf("[%5.1f%% %4d/%4d] [PASS] %s: %s (%v)",
			percent, completed, total, res.Case.ID, res.Case.Description, res.Duration.Round(time.Millisecond))
	}
	return fmt.Sprintf("[%5.1f%% %4d/%4d] \x1b[31m[FAIL]\x1b[0m %s: %s\n       --> %s",
		percent, completed, total, res.Case.ID, res.Case.Description, res.ErrorMsg)
}
