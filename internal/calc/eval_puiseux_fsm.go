package calc

import (
	"fmt"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// PuiseuxState represents a discrete lifecycle state in the Newton-Puiseux series solver pipeline.
type PuiseuxState int

const (
	PuiseuxStateInit                PuiseuxState = 0
	PuiseuxStateSupportExtracted    PuiseuxState = 1
	PuiseuxStateHullConstructed     PuiseuxState = 2
	PuiseuxStateEdgeSelected        PuiseuxState = 3
	PuiseuxStateCharPolySolved      PuiseuxState = 4
	PuiseuxStateRamifiedAndShifted  PuiseuxState = 5
	PuiseuxStateRecursiveExpanded   PuiseuxState = 6
	PuiseuxStateBranchesConstructed PuiseuxState = 7
	PuiseuxStateFailed              PuiseuxState = 8
)

func (s PuiseuxState) String() string {
	switch s {
	case PuiseuxStateInit:
		return "Init"
	case PuiseuxStateSupportExtracted:
		return "SupportExtracted"
	case PuiseuxStateHullConstructed:
		return "HullConstructed"
	case PuiseuxStateEdgeSelected:
		return "EdgeSelected"
	case PuiseuxStateCharPolySolved:
		return "CharPolySolved"
	case PuiseuxStateRamifiedAndShifted:
		return "RamifiedAndShifted"
	case PuiseuxStateRecursiveExpanded:
		return "RecursiveExpanded"
	case PuiseuxStateBranchesConstructed:
		return "BranchesConstructed"
	case PuiseuxStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("PuiseuxState(%d)", int(s))
	}
}

// PuiseuxLifecycleFSM governs valid state transitions in the Newton-Puiseux pipeline.
type PuiseuxLifecycleFSM struct {
	mu           sync.RWMutex
	currentState PuiseuxState
	history      []PuiseuxState
}

// NewPuiseuxLifecycleFSM creates a new PuiseuxLifecycleFSM initialized to PuiseuxStateInit.
func NewPuiseuxLifecycleFSM() *PuiseuxLifecycleFSM {
	return &PuiseuxLifecycleFSM{
		currentState: PuiseuxStateInit,
		history:      []PuiseuxState{PuiseuxStateInit},
	}
}

// CurrentState returns the current state.
func (fsm *PuiseuxLifecycleFSM) CurrentState() PuiseuxState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// History returns a copy of the state transition history.
func (fsm *PuiseuxLifecycleFSM) History() []PuiseuxState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	cp := make([]PuiseuxState, len(fsm.history))
	copy(cp, fsm.history)
	return cp
}

// TransitionTo validates and performs a state transition.
func (fsm *PuiseuxLifecycleFSM) TransitionTo(target PuiseuxState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case PuiseuxStateInit:
		valid = (target == PuiseuxStateSupportExtracted || target == PuiseuxStateFailed)
	case PuiseuxStateSupportExtracted:
		valid = (target == PuiseuxStateHullConstructed || target == PuiseuxStateBranchesConstructed || target == PuiseuxStateFailed)
	case PuiseuxStateHullConstructed:
		valid = (target == PuiseuxStateEdgeSelected || target == PuiseuxStateBranchesConstructed || target == PuiseuxStateFailed)
	case PuiseuxStateEdgeSelected:
		valid = (target == PuiseuxStateCharPolySolved || target == PuiseuxStateFailed)
	case PuiseuxStateCharPolySolved:
		valid = (target == PuiseuxStateRamifiedAndShifted || target == PuiseuxStateBranchesConstructed || target == PuiseuxStateFailed)
	case PuiseuxStateRamifiedAndShifted:
		valid = (target == PuiseuxStateRecursiveExpanded || target == PuiseuxStateBranchesConstructed || target == PuiseuxStateFailed)
	case PuiseuxStateRecursiveExpanded:
		valid = (target == PuiseuxStateEdgeSelected || target == PuiseuxStateBranchesConstructed || target == PuiseuxStateFailed)
	case PuiseuxStateBranchesConstructed, PuiseuxStateFailed:
		valid = false // Terminal states
	}

	if !valid {
		return fmt.Errorf("%s", i18n.T("puiseux.err_invalid_fsm_transition", fsm.currentState, target))
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}
