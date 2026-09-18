package calc

import (
	"fmt"
	"sync"
)

// CadState represents a discrete phase in the Cylindrical Algebraic Decomposition lifecycle.
type CadState int

const (
	CadStateInit         CadState = 0
	CadStateNormalized   CadState = 1
	CadStateProjected    CadState = 2
	CadStateValidated    CadState = 3
	CadStateSampled      CadState = 4
	CadStateLifted       CadState = 5
	CadStateDecided      CadState = 6
	CadStateUnsupported  CadState = 7
)

func (s CadState) String() string {
	switch s {
	case CadStateInit:
		return "Init"
	case CadStateNormalized:
		return "Normalized"
	case CadStateProjected:
		return "Projected"
	case CadStateValidated:
		return "Validated"
	case CadStateSampled:
		return "Sampled"
	case CadStateLifted:
		return "Lifted"
	case CadStateDecided:
		return "Decided"
	case CadStateUnsupported:
		return "Unsupported"
	default:
		return fmt.Sprintf("CadState(%d)", int(s))
	}
}

// CadLifecycleFSM governs the state transitions and invariants of the CAD engine.
type CadLifecycleFSM struct {
	mu           sync.RWMutex
	currentState CadState
	history      []CadState
}

// NewCadLifecycleFSM creates a new CAD lifecycle FSM initialized to CadStateInit.
func NewCadLifecycleFSM() *CadLifecycleFSM {
	return &CadLifecycleFSM{
		currentState: CadStateInit,
		history:      []CadState{CadStateInit},
	}
}

// CurrentState returns the current state of the FSM.
func (fsm *CadLifecycleFSM) CurrentState() CadState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// TransitionTo validates and performs a state transition to target.
func (fsm *CadLifecycleFSM) TransitionTo(target CadState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case CadStateInit:
		// Can transition to Normalized or directly to Unsupported (if input malformed)
		valid = (target == CadStateNormalized || target == CadStateUnsupported)
	case CadStateNormalized:
		// After normalization, can proceed to Projected (multivariate) or Sampled (1D Fast Path)
		valid = (target == CadStateProjected || target == CadStateSampled || target == CadStateUnsupported)
	case CadStateProjected:
		valid = (target == CadStateValidated || target == CadStateUnsupported)
	case CadStateValidated:
		valid = (target == CadStateSampled || target == CadStateUnsupported)
	case CadStateSampled:
		valid = (target == CadStateLifted || target == CadStateDecided || target == CadStateUnsupported)
	case CadStateLifted:
		valid = (target == CadStateDecided || target == CadStateUnsupported)
	case CadStateDecided, CadStateUnsupported:
		valid = false // Terminal states
	}

	if !valid {
		return fmt.Errorf("invalid CAD FSM transition: %s -> %s", fsm.currentState, target)
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}
