package calc

import (
	"testing"
)

func TestCubicLifecycleFSM(t *testing.T) {
	fsm := NewCubicLifecycleFSM()
	if fsm.CurrentState() != CubicStateInit {
		t.Fatalf("expected CubicStateInit, got %v", fsm.CurrentState())
	}

	// Valid sequence: Init -> DegreeValidated -> DepressedNormalized -> CardanoResolved -> RootsConstructed
	if err := fsm.TransitionTo(CubicStateDegreeValidated); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(CubicStateDepressedNormalized); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(CubicStateCardanoResolved); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(CubicStateRootsConstructed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid transition from terminal state
	if err := fsm.TransitionTo(CubicStateInit); err == nil {
		t.Fatalf("expected error on transition from terminal state, got nil")
	}

	// Test history tracking
	history := fsm.History()
	expectedStates := []CubicState{
		CubicStateInit,
		CubicStateDegreeValidated,
		CubicStateDepressedNormalized,
		CubicStateCardanoResolved,
		CubicStateRootsConstructed,
	}
	if len(history) != len(expectedStates) {
		t.Fatalf("expected %d history states, got %d", len(expectedStates), len(history))
	}
	for i, s := range expectedStates {
		if history[i] != s {
			t.Errorf("history[%d]: expected %v, got %v", i, s, history[i])
		}
	}
}

func TestCubicLifecycleFSM_RationalDeflated(t *testing.T) {
	fsm := NewCubicLifecycleFSM()
	// Sequence: Init -> DegreeValidated -> RationalDeflated -> RootsConstructed
	if err := fsm.TransitionTo(CubicStateDegreeValidated); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(CubicStateRationalDeflated); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(CubicStateRootsConstructed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
