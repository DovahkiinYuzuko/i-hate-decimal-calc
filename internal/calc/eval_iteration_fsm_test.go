package calc

import (
	"bytes"
	"math/big"
	"strings"
	"testing"
)

func TestIterationFSM_NonDestructiveWarning(t *testing.T) {
	env := NewEnv()
	start := big.NewRat(1, 1)
	end := big.NewRat(10, 1)
	step := big.NewRat(1, 1)

	fsm := NewIterationFSM("k", start, end, step, env)
	warnBuf := &bytes.Buffer{}
	// Set warning limit to 4 steps, unlimited max (-1)
	fsm.SetLimits(big.NewInt(4), big.NewInt(-1), warnBuf)

	var visited []int64
	err := fsm.Run(func(curr *big.Rat) (bool, error) {
		visited = append(visited, curr.Num().Int64())
		return false, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(visited) != 10 {
		t.Fatalf("expected 10 steps to finish, got %d", len(visited))
	}

	warnOutput := warnBuf.String()
	if !strings.Contains(warnOutput, "warning") && !strings.Contains(warnOutput, "警告") {
		t.Errorf("expected warning message in buffer, got: %q", warnOutput)
	}

	// Warning should only be emitted once, not repeated on every subsequent step
	lines := strings.Split(strings.TrimSpace(warnOutput), "\n")
	if len(lines) != 1 {
		t.Errorf("expected exactly 1 warning line, got %d lines: %q", len(lines), warnOutput)
	}
}

func TestIterationFSM_MaxIterationsExceeded(t *testing.T) {
	env := NewEnv()
	start := big.NewRat(1, 1)
	end := big.NewRat(100, 1)
	step := big.NewRat(1, 1)

	fsm := NewIterationFSM("k", start, end, step, env)
	warnBuf := &bytes.Buffer{}
	// Set warning limit to 5, max limit to 15
	fsm.SetLimits(big.NewInt(5), big.NewInt(15), warnBuf)

	var visited []int64
	err := fsm.Run(func(curr *big.Rat) (bool, error) {
		visited = append(visited, curr.Num().Int64())
		return false, nil
	})

	if err == nil {
		t.Fatalf("expected error when max limit is exceeded, got nil")
	}

	if !strings.Contains(err.Error(), "exceeded maximum limit") && !strings.Contains(err.Error(), "最大上限") {
		t.Errorf("expected error message to mention maximum limit exceeded, got: %v", err)
	}

	if len(visited) != 15 {
		t.Errorf("expected exactly 15 steps before aborting on the 16th, got %d", len(visited))
	}
}

func TestIterationFSM_UnlimitedMaxIterations(t *testing.T) {
	env := NewEnv()
	start := big.NewRat(1, 1)
	end := big.NewRat(30, 1)
	step := big.NewRat(1, 1)

	fsm := NewIterationFSM("k", start, end, step, env)
	warnBuf := &bytes.Buffer{}
	// Warn at 10, unlimited (-1)
	fsm.SetLimits(big.NewInt(10), big.NewInt(-1), warnBuf)

	steps := 0
	err := fsm.Run(func(curr *big.Rat) (bool, error) {
		steps++
		return false, nil
	})

	if err != nil {
		t.Fatalf("expected nil error for unlimited iterations, got: %v", err)
	}
	if steps != 30 {
		t.Errorf("expected 30 steps, got %d", steps)
	}
}

func TestIterationFSM_GiantLimitsBigInt(t *testing.T) {
	env := NewEnv()
	start := big.NewRat(1, 1)
	end := big.NewRat(5, 1)
	step := big.NewRat(1, 1)

	fsm := NewIterationFSM("k", start, end, step, env)
	warnBuf := &bytes.Buffer{}

	giantWarn, _ := new(big.Int).SetString("1000000000000000000000000000000", 10)
	giantMax, _ := new(big.Int).SetString("2000000000000000000000000000000", 10)
	fsm.SetLimits(giantWarn, giantMax, warnBuf)

	steps := 0
	err := fsm.Run(func(curr *big.Rat) (bool, error) {
		steps++
		return false, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if steps != 5 {
		t.Errorf("expected 5 steps, got %d", steps)
	}
	if warnBuf.Len() != 0 {
		t.Errorf("no warning expected for giant limit, got: %s", warnBuf.String())
	}
}
