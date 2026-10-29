package calc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

func TestBudget_Steps(t *testing.T) {
	cfg := calc.BudgetConfig{
		BudgetSteps: 5,
	}
	b := calc.NewExecutionBudget(context.Background(), cfg)

	for i := 0; i < 5; i++ {
		if err := b.Tick(); err != nil {
			t.Fatalf("expected step %d to succeed, got %v", i+1, err)
		}
	}

	// 6th step should exceed budget
	err := b.Tick()
	if !errors.Is(err, calc.ErrStepLimitExceeded) {
		t.Fatalf("expected ErrStepLimitExceeded, got %v", err)
	}
}

func TestBudget_Steps_Unlimited(t *testing.T) {
	cfg := calc.BudgetConfig{
		BudgetSteps: 0, // unlimited
	}
	b := calc.NewExecutionBudget(context.Background(), cfg)

	for i := 0; i < 1000; i++ {
		if err := b.Tick(); err != nil {
			t.Fatalf("unexpected error on step %d: %v", i+1, err)
		}
	}
}

func TestBudget_Nodes(t *testing.T) {
	cfg := calc.BudgetConfig{
		MaxNodes: 10,
	}
	b := calc.NewExecutionBudget(context.Background(), cfg)

	if err := b.TrackNodes(5); err != nil {
		t.Fatalf("unexpected error tracking 5 nodes: %v", err)
	}
	if err := b.TrackNodes(5); err != nil {
		t.Fatalf("unexpected error tracking another 5 nodes: %v", err)
	}

	err := b.TrackNodes(1)
	if !errors.Is(err, calc.ErrNodeLimitExceeded) {
		t.Fatalf("expected ErrNodeLimitExceeded, got %v", err)
	}
}

func TestBudget_Depth(t *testing.T) {
	cfg := calc.BudgetConfig{
		MaxDepth: 3,
	}
	b := calc.NewExecutionBudget(context.Background(), cfg)

	// Enter 3 times
	for i := 0; i < 3; i++ {
		if err := b.EnterDepth(); err != nil {
			t.Fatalf("expected enter depth %d to succeed, got %v", i+1, err)
		}
	}

	// 4th enter exceeds limit
	if err := b.EnterDepth(); !errors.Is(err, calc.ErrRecursionDepthExceeded) {
		t.Fatalf("expected ErrRecursionDepthExceeded, got %v", err)
	}

	// Leave once, now depth is 2, can enter once more to reach 3
	b.LeaveDepth()
	if err := b.EnterDepth(); err != nil {
		t.Fatalf("expected enter depth after leave to succeed, got %v", err)
	}
}

func TestBudget_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := calc.BudgetConfig{
		BudgetSteps: 0,
	}
	b := calc.NewExecutionBudget(ctx, cfg)

	// Cancel context immediately
	cancel()

	// Advance steps until amortized check triggers (default mask 0xFF means step 256 triggers check)
	var timeoutErr error
	for i := 0; i < 300; i++ {
		if err := b.Tick(); err != nil {
			timeoutErr = err
			break
		}
	}

	if timeoutErr == nil {
		t.Fatal("expected timeout error, but got nil after 300 ticks")
	}
	if !errors.Is(timeoutErr, calc.ErrTimeout) {
		t.Fatalf("expected errors.Is(ErrTimeout), got %v", timeoutErr)
	}
}
