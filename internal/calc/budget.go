package calc

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrTimeout is returned when execution exceeds wall-clock time limit.
	ErrTimeout = errors.New("evaluation timed out")

	// ErrStepLimitExceeded is returned when execution exceeds step limit.
	ErrStepLimitExceeded = errors.New("evaluation step budget exceeded")

	// ErrNodeLimitExceeded is returned when cumulative AST nodes exceed limit.
	ErrNodeLimitExceeded = errors.New("cumulative AST node limit exceeded")

	// ErrRecursionDepthExceeded is returned when recursive descent exceeds limit.
	ErrRecursionDepthExceeded = errors.New("maximum recursion depth exceeded")
)

const (
	// DefaultCheckIntervalMask defines frequency of amortized context cancellation check.
	// 0xFF means checking every 256 steps.
	DefaultCheckIntervalMask uint64 = 0xFF

	// DefaultMaxRecursionDepth prevents unrecoverable fatal throw: stack overflow.
	DefaultMaxRecursionDepth uint64 = 1000
)

// BudgetConfig defines the resource limits for evaluator execution.
type BudgetConfig struct {
	Timeout       time.Duration `json:"timeout,omitempty"`
	BudgetSteps   uint64        `json:"budget_steps,omitempty"` // 0 = unlimited
	MaxNodes      uint64        `json:"max_nodes,omitempty"`    // 0 = unlimited
	MaxDepth      uint64        `json:"max_depth,omitempty"`    // 0 = use DefaultMaxRecursionDepth
	Deterministic bool          `json:"deterministic,omitempty"`
}

// ExecutionBudget tracks runtime resources for a single top-level evaluation.
type ExecutionBudget struct {
	ctx          context.Context
	cfg          BudgetConfig
	currentSteps uint64
	currentNodes uint64
	currentDepth uint64
	checkMask    uint64
}

// NewExecutionBudget initializes an execution budget tracker.
func NewExecutionBudget(ctx context.Context, cfg BudgetConfig) *ExecutionBudget {
	depth := cfg.MaxDepth
	if depth == 0 {
		depth = DefaultMaxRecursionDepth
	}
	return &ExecutionBudget{
		ctx:          ctx,
		cfg:          cfg,
		currentDepth: 0,
		checkMask:    DefaultCheckIntervalMask,
	}
}

// Tick consumes one evaluation step and performs amortized context checking.
func (b *ExecutionBudget) Tick() error {
	if b == nil {
		return nil
	}

	b.currentSteps++
	if b.cfg.BudgetSteps > 0 && b.currentSteps > b.cfg.BudgetSteps {
		return ErrStepLimitExceeded
	}

	// Amortized context check via bitmask: no channel operations on hot path
	if (b.currentSteps&b.checkMask) == 0 && b.ctx != nil {
		select {
		case <-b.ctx.Done():
			return fmt.Errorf("%w: %v", ErrTimeout, b.ctx.Err())
		default:
		}
	}
	return nil
}

// TrackNodes adds newly created node count.
func (b *ExecutionBudget) TrackNodes(count uint64) error {
	if b == nil {
		return nil
	}

	b.currentNodes += count
	if b.cfg.MaxNodes > 0 && b.currentNodes > b.cfg.MaxNodes {
		return ErrNodeLimitExceeded
	}
	return nil
}

// EnterDepth tracks recursion frames.
func (b *ExecutionBudget) EnterDepth() error {
	if b == nil {
		return nil
	}

	if b.cfg.MaxDepth > 0 && b.currentDepth >= b.cfg.MaxDepth {
		return ErrRecursionDepthExceeded
	}
	b.currentDepth++
	return nil
}

// LeaveDepth unwinds one recursion frame.
func (b *ExecutionBudget) LeaveDepth() {
	if b == nil {
		return
	}

	if b.currentDepth > 0 {
		b.currentDepth--
	}
}

// Steps returns the number of evaluated steps so far.
func (b *ExecutionBudget) Steps() uint64 {
	if b == nil {
		return 0
	}
	return b.currentSteps
}

// Nodes returns the cumulative number of tracked nodes so far.
func (b *ExecutionBudget) Nodes() uint64 {
	if b == nil {
		return 0
	}
	return b.currentNodes
}

// Depth returns the current recursion depth.
func (b *ExecutionBudget) Depth() uint64 {
	if b == nil {
		return 0
	}
	return b.currentDepth
}
