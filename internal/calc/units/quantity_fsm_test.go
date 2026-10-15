package units_test

import (
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/units"
)

func TestQuantityFSM_ValidLifecycle(t *testing.T) {
	fsm := units.NewQuantityFSM()
	if fsm.CurrentState() != units.UnitStateInit {
		t.Fatalf("expected initial state %s, got %s", units.UnitStateInit, fsm.CurrentState())
	}

	// 正常なライフサイクル遷移
	steps := []units.UnitState{
		units.UnitStateTokenizing,
		units.UnitStatePrefixResolved,
		units.UnitStateBaseIdentified,
		units.UnitStateVectorSynthesized,
		units.UnitStateNormalized,
		units.UnitStateSuccess,
	}

	for _, step := range steps {
		if err := fsm.Transition(step); err != nil {
			t.Fatalf("unexpected transition error to %s: %v", step, err)
		}
		if fsm.CurrentState() != step {
			t.Fatalf("expected state %s, got %s", step, fsm.CurrentState())
		}
	}

	history := fsm.History()
	if len(history) != len(steps)+1 {
		t.Fatalf("expected history length %d, got %d", len(steps)+1, len(history))
	}
}

func TestQuantityFSM_InvalidTransitionRejection(t *testing.T) {
	fsm := units.NewQuantityFSM()

	// Init から Success への不正なジャンプ
	err := fsm.Transition(units.UnitStateSuccess)
	if err == nil {
		t.Fatalf("expected error on invalid transition Init -> Success, got nil")
	}

	if fsm.CurrentState() != units.UnitStateFailure {
		t.Fatalf("expected state to transition to Failure upon invalid transition, got %s", fsm.CurrentState())
	}
}
