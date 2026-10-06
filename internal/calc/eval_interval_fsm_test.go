package calc

import (
	"testing"
)

func TestIntervalRefinerFSM_NormalFlow(t *testing.T) {
	fsm := NewIntervalRefinerFSM(50)
	if fsm.CurrentState() != IntervalStateInit {
		t.Fatalf("expected initial state Init, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(IntervalStateInitialEnclosure); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := fsm.TransitionTo(IntervalStateCheckingTolerance); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := fsm.TransitionTo(IntervalStateRefining); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := fsm.TransitionTo(IntervalStateCheckingTolerance); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := fsm.TransitionTo(IntervalStateConverged); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !fsm.IsTerminal() {
		t.Fatalf("expected terminal state")
	}

	// Transition from terminal state should fail
	if err := fsm.TransitionTo(IntervalStateRefining); err == nil {
		t.Fatalf("expected error on transition from terminal state")
	}

	history := fsm.History()
	if len(history) != 6 {
		t.Fatalf("expected history length 6, got %d", len(history))
	}
}

func TestIntervalRefinerFSM_MaxStepsReached(t *testing.T) {
	fsm := NewIntervalRefinerFSM(10)
	_ = fsm.TransitionTo(IntervalStateInitialEnclosure)
	_ = fsm.TransitionTo(IntervalStateCheckingTolerance)

	if err := fsm.TransitionTo(IntervalStateMaxStepsReached); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !fsm.IsTerminal() {
		t.Fatalf("expected terminal state")
	}
}

func TestIntervalRefinerFSM_FailureFlow(t *testing.T) {
	fsm := NewIntervalRefinerFSM(10)
	if err := fsm.TransitionTo(IntervalStateFailed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fsm.IsTerminal() {
		t.Fatalf("expected terminal state")
	}
}

func TestIntervalRefinerFSM_InvalidTransitions(t *testing.T) {
	fsm := NewIntervalRefinerFSM(10)
	// Invalid: Init -> Converged directly
	if err := fsm.TransitionTo(IntervalStateConverged); err == nil {
		t.Fatalf("expected error on invalid transition Init -> Converged")
	}
}
