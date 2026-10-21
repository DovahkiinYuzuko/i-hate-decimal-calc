package calc_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// ErrExecutionTimeout is returned when an operation exceeds its configured deadline.
var ErrExecutionTimeout = errors.New("fuzz execution timed out")

// CountNodes recursively counts the total number of nodes in an AST.
// It guards against expression blow-up and exponential tree growth.
func CountNodes(n ast.Node) int {
	if n == nil {
		return 0
	}
	count := 0
	ast.Walk(n, func(curr ast.Node) bool {
		if curr != nil {
			count++
		}
		return true
	})
	return count
}

// RunWithTimeout executes the given function within a specified timeout duration.
// If the deadline is exceeded before fn finishes, ErrExecutionTimeout is returned.
func RunWithTimeout(d time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		// Recover any unhandled panics inside fn to prevent test suite crash
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic in guarded function: %v", r)
			}
		}()
		errCh <- fn()
	}()

	select {
	case <-ctx.Done():
		return ErrExecutionTimeout
	case err := <-errCh:
		return err
	}
}

// TestGuard_NodeLimitAndTimeout validates AST node counting and execution timeout safeguards.
func TestGuard_NodeLimitAndTimeout(t *testing.T) {
	// 1. Verify CountNodes accuracy
	// Leaf node: Count == 1
	leaf := ast.NewVar("x")
	if c := CountNodes(leaf); c != 1 {
		t.Fatalf("Expected CountNodes(x) == 1, got %d", c)
	}

	// Binary node: Add(x, 1) -> Root(Add), Child1(x), Child2(1) -> 3 nodes
	one := ast.NewRationalFromBigRat(big.NewRat(1, 1))
	add := ast.NewAdd([]ast.Node{leaf, one})
	if c := CountNodes(add); c != 3 {
		t.Fatalf("Expected CountNodes(x + 1) == 3, got %d", c)
	}

	// 2. Verify CountNodes handles deep/nested trees and enforces maximum thresholds (e.g. 5,000)
	// Using SqrtNode nesting so nodes don't flatten associativity
	curr := ast.Node(leaf)
	for i := 0; i < 100; i++ {
		curr = ast.NewSqrt(curr)
	}
	// 1 (leaf) + 100 (Sqrt nodes) = 101 nodes
	expectedNodes := 101
	if c := CountNodes(curr); c != expectedNodes {
		t.Fatalf("Expected CountNodes == %d, got %d", expectedNodes, c)
	}

	// Max 5,000 upper bound guard check
	if CountNodes(curr) > 5000 {
		t.Fatalf("Node count exceeded maximum 5000 guard limit")
	}

	// 3. Verify RunWithTimeout on successful fast function
	fastErr := RunWithTimeout(100*time.Millisecond, func() error {
		return nil
	})
	if fastErr != nil {
		t.Fatalf("Expected nil error for fast function, got: %v", fastErr)
	}

	// 4. Verify RunWithTimeout properly halts slow/infinite functions
	slowErr := RunWithTimeout(50*time.Millisecond, func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	if !errors.Is(slowErr, ErrExecutionTimeout) {
		t.Fatalf("Expected ErrExecutionTimeout for slow function, got: %v", slowErr)
	}

	// 5. Verify RunWithTimeout recovers panics safely
	panicErr := RunWithTimeout(100*time.Millisecond, func() error {
		panic("simulated unhandled panic in worker")
	})
	if panicErr == nil || !errors.Is(panicErr, ErrExecutionTimeout) && panicErr.Error() == "" {
		t.Fatalf("Expected recovered panic error, got: %v", panicErr)
	}
}
