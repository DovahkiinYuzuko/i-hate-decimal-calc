package calc

import (
	"errors"
	"math/big"
	"testing"
)

func TestTaskFSM_Lifecycle(t *testing.T) {
	fsm := NewTaskFSM()
	if fsm.CurrentState() != StateIdle {
		t.Fatalf("expected initial StateIdle, got %v", fsm.CurrentState())
	}
	if fsm.IsDone() {
		t.Fatalf("expected IsDone to be false in StateIdle")
	}

	// Idle -> Running
	if err := fsm.Transition(StateRunning); err != nil {
		t.Fatalf("unexpected transition error: %v", err)
	}
	if fsm.CurrentState() != StateRunning {
		t.Fatalf("expected StateRunning, got %v", fsm.CurrentState())
	}

	// Invalid transition: Running -> Idle
	if err := fsm.Transition(StateIdle); err == nil {
		t.Fatalf("expected error on transition Running -> Idle")
	}

	// Running -> Completed
	if err := fsm.Transition(StateCompleted); err != nil {
		t.Fatalf("unexpected transition error: %v", err)
	}
	if !fsm.IsDone() {
		t.Fatalf("expected IsDone to be true in StateCompleted")
	}

	// Completed -> Running (terminal state violation)
	if err := fsm.Transition(StateRunning); err == nil {
		t.Fatalf("expected error on transition from Completed")
	}
}

func TestParallelProductTree(t *testing.T) {
	res := ParallelProductTree(1, 20, 4)

	expected := big.NewInt(1)
	for i := int64(1); i <= 20; i++ {
		expected.Mul(expected, big.NewInt(i))
	}

	if res.Cmp(expected) != 0 {
		t.Fatalf("ParallelProductTree(1, 20) = %s, want %s", res.String(), expected.String())
	}

	one := ParallelProductTree(5, 5, 4)
	if one.Cmp(big.NewInt(5)) != 0 {
		t.Errorf("expected 5, got %s", one.String())
	}

	empty := ParallelProductTree(10, 5, 4)
	if empty.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("expected 1, got %s", empty.String())
	}

	resLarge := ParallelProductTree(1, 1000, 16)
	expectedLarge := big.NewInt(1)
	for i := int64(1); i <= 1000; i++ {
		expectedLarge.Mul(expectedLarge, big.NewInt(i))
	}
	if resLarge.Cmp(expectedLarge) != 0 {
		t.Fatalf("ParallelProductTree(1, 1000) mismatch with sequential")
	}
}

func TestParallelBatchMap(t *testing.T) {
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	results, err := ParallelBatchMap(items, func(v int) (int, error) {
		return v * 2, nil
	}, 4)
	if err != nil {
		t.Fatalf("ParallelBatchMap failed: %v", err)
	}

	if len(results) != 100 {
		t.Fatalf("expected 100 results, got %d", len(results))
	}
	for i, r := range results {
		if r != i*2 {
			t.Fatalf("results[%d] = %d, want %d", i, r, i*2)
		}
	}

	_, errFail := ParallelBatchMap(items, func(v int) (int, error) {
		if v == 42 {
			return 0, errors.New("boom")
		}
		return v, nil
	}, 4)
	if errFail == nil {
		t.Fatalf("expected error propagation from worker")
	}
}

func BenchmarkProductTree_Parallel(b *testing.B) {
	for b.Loop() {
		_ = ParallelProductTree(1, 10000, 64)
	}
}

func BenchmarkProductTree_Sequential(b *testing.B) {
	for b.Loop() {
		res := big.NewInt(1)
		cur := new(big.Int)
		for i := int64(2); i <= 10000; i++ {
			cur.SetInt64(i)
			res.Mul(res, cur)
		}
	}
}
