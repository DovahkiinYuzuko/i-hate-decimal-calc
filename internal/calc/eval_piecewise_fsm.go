package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// PiecewiseState represents the discrete execution phase of Piecewise algebraic evaluation and normalization.
type PiecewiseState int

const (
	PiecewiseStateInitial PiecewiseState = iota
	PiecewiseStateParsing
	PiecewiseStateCadNormalizing
	PiecewiseStatePruningEmpty
	PiecewiseStateFolding
	PiecewiseStateSimplifying
	PiecewiseStateMerging
	PiecewiseStateCompleted
	PiecewiseStateError
)

// String returns a human-readable representation of PiecewiseState.
func (s PiecewiseState) String() string {
	switch s {
	case PiecewiseStateInitial:
		return "Initial"
	case PiecewiseStateParsing:
		return "Parsing"
	case PiecewiseStateCadNormalizing:
		return "CadNormalizing"
	case PiecewiseStatePruningEmpty:
		return "PruningEmpty"
	case PiecewiseStateFolding:
		return "Folding"
	case PiecewiseStateSimplifying:
		return "Simplifying"
	case PiecewiseStateMerging:
		return "Merging"
	case PiecewiseStateCompleted:
		return "Completed"
	case PiecewiseStateError:
		return "Error"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// PiecewiseFSM governs and protects state transitions during Piecewise evaluation, CAD domain normalization, and folding.
type PiecewiseFSM struct {
	currentState PiecewiseState
	history      []PiecewiseState
}

// NewPiecewiseFSM initializes a new PiecewiseFSM in PiecewiseStateInitial.
func NewPiecewiseFSM() *PiecewiseFSM {
	return &PiecewiseFSM{
		currentState: PiecewiseStateInitial,
		history:      []PiecewiseState{PiecewiseStateInitial},
	}
}

// Current returns the current state of the FSM.
func (f *PiecewiseFSM) Current() PiecewiseState {
	return f.currentState
}

// History returns a copy of the state transition history.
func (f *PiecewiseFSM) History() []PiecewiseState {
	copied := make([]PiecewiseState, len(f.history))
	copy(copied, f.history)
	return copied
}

// TransitionTo validates and performs a state transition according to the governance state machine.
func (f *PiecewiseFSM) TransitionTo(target PiecewiseState) error {
	// Error state is universally reachable from any non-completed state
	if target == PiecewiseStateError {
		f.currentState = target
		f.history = append(f.history, target)
		return nil
	}

	valid := false
	switch f.currentState {
	case PiecewiseStateInitial:
		valid = (target == PiecewiseStateParsing || target == PiecewiseStateCadNormalizing || target == PiecewiseStateFolding)
	case PiecewiseStateParsing:
		valid = (target == PiecewiseStateCadNormalizing || target == PiecewiseStatePruningEmpty || target == PiecewiseStateSimplifying)
	case PiecewiseStateCadNormalizing:
		valid = (target == PiecewiseStatePruningEmpty || target == PiecewiseStateSimplifying || target == PiecewiseStateCompleted)
	case PiecewiseStatePruningEmpty:
		valid = (target == PiecewiseStateFolding || target == PiecewiseStateSimplifying || target == PiecewiseStateMerging || target == PiecewiseStateCompleted)
	case PiecewiseStateFolding:
		valid = (target == PiecewiseStateSimplifying || target == PiecewiseStateCadNormalizing || target == PiecewiseStateCompleted)
	case PiecewiseStateSimplifying:
		valid = (target == PiecewiseStateMerging || target == PiecewiseStateCompleted)
	case PiecewiseStateMerging:
		valid = (target == PiecewiseStateCompleted)
	case PiecewiseStateCompleted:
		// Terminal state: no transitions permitted
		valid = false
	case PiecewiseStateError:
		// Terminal failure: no transitions permitted
		valid = false
	}

	if !valid {
		return fmt.Errorf("%s", i18n.T("piecewise.err_invalid_piecewise_fsm_transition", f.currentState.String(), target.String()))
	}

	f.currentState = target
	f.history = append(f.history, target)
	return nil
}
