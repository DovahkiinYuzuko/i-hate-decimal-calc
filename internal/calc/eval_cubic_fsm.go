package calc

import (
	"fmt"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// CubicState represents a discrete lifecycle state in exact cubic polynomial solving.
type CubicState int

const (
	CubicStateInit                CubicState = 0
	CubicStateDegreeValidated     CubicState = 1
	CubicStateRationalDeflated    CubicState = 2
	CubicStateDepressedNormalized CubicState = 3
	CubicStateCardanoResolved     CubicState = 4
	CubicStateRootsConstructed    CubicState = 5
	CubicStateFailed              CubicState = 6
)

func (s CubicState) String() string {
	switch s {
	case CubicStateInit:
		return "Init"
	case CubicStateDegreeValidated:
		return "DegreeValidated"
	case CubicStateRationalDeflated:
		return "RationalDeflated"
	case CubicStateDepressedNormalized:
		return "DepressedNormalized"
	case CubicStateCardanoResolved:
		return "CardanoResolved"
	case CubicStateRootsConstructed:
		return "RootsConstructed"
	case CubicStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("CubicState(%d)", int(s))
	}
}

// CubicLifecycleFSM governs valid state transitions in the cubic solver pipeline.
type CubicLifecycleFSM struct {
	mu           sync.RWMutex
	currentState CubicState
	history      []CubicState
}

// NewCubicLifecycleFSM creates a new CubicLifecycleFSM initialized to CubicStateInit.
func NewCubicLifecycleFSM() *CubicLifecycleFSM {
	return &CubicLifecycleFSM{
		currentState: CubicStateInit,
		history:      []CubicState{CubicStateInit},
	}
}

// CurrentState returns the current state.
func (fsm *CubicLifecycleFSM) CurrentState() CubicState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// History returns a copy of the state transition history.
func (fsm *CubicLifecycleFSM) History() []CubicState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	cp := make([]CubicState, len(fsm.history))
	copy(cp, fsm.history)
	return cp
}

// TransitionTo validates and performs a state transition.
func (fsm *CubicLifecycleFSM) TransitionTo(target CubicState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case CubicStateInit:
		valid = (target == CubicStateDegreeValidated || target == CubicStateFailed)
	case CubicStateDegreeValidated:
		valid = (target == CubicStateRationalDeflated || target == CubicStateDepressedNormalized || target == CubicStateRootsConstructed || target == CubicStateFailed)
	case CubicStateRationalDeflated:
		valid = (target == CubicStateRootsConstructed || target == CubicStateFailed)
	case CubicStateDepressedNormalized:
		valid = (target == CubicStateCardanoResolved || target == CubicStateRootsConstructed || target == CubicStateFailed)
	case CubicStateCardanoResolved:
		valid = (target == CubicStateRootsConstructed || target == CubicStateFailed)
	case CubicStateRootsConstructed, CubicStateFailed:
		valid = false // Terminal states
	}

	if !valid {
		return fmt.Errorf("%s", i18n.T("cubic.err_invalid_cubic_fsm_transition", fsm.currentState, target))
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}
