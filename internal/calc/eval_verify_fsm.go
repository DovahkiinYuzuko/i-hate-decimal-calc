package calc

import (
	"fmt"
	"sync"
)

// VerifyState represents a discrete state in the self-verifying CAS lifecycle.
type VerifyState int

const (
	VerifyStateIdle                     VerifyState = 0
	VerifyStateTargetClassified         VerifyState = 1
	VerifyStateResidualConstructed      VerifyState = 2
	VerifyStateSimplificationEvaluated  VerifyState = 3
	VerifyStateCertified                VerifyState = 4
	VerifyStateRefuted                  VerifyState = 5
	VerifyStateUnsupportedDomain        VerifyState = 6
)

func (s VerifyState) String() string {
	switch s {
	case VerifyStateIdle:
		return "Idle"
	case VerifyStateTargetClassified:
		return "TargetClassified"
	case VerifyStateResidualConstructed:
		return "ResidualConstructed"
	case VerifyStateSimplificationEvaluated:
		return "SimplificationEvaluated"
	case VerifyStateCertified:
		return "Certified"
	case VerifyStateRefuted:
		return "Refuted"
	case VerifyStateUnsupportedDomain:
		return "UnsupportedDomain"
	default:
		return fmt.Sprintf("VerifyState(%d)", int(s))
	}
}

// VerifyLifecycleFSM governs the integrity of the verification process.
type VerifyLifecycleFSM struct {
	mu           sync.RWMutex
	currentState VerifyState
	history      []VerifyState
}

// NewVerifyLifecycleFSM creates a new verification FSM initialized to Idle.
func NewVerifyLifecycleFSM() *VerifyLifecycleFSM {
	return &VerifyLifecycleFSM{
		currentState: VerifyStateIdle,
		history:      []VerifyState{VerifyStateIdle},
	}
}

// CurrentState returns the current state of the FSM.
func (fsm *VerifyLifecycleFSM) CurrentState() VerifyState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// TransitionTo transitions the FSM to targetState if valid.
func (fsm *VerifyLifecycleFSM) TransitionTo(target VerifyState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case VerifyStateIdle:
		valid = (target == VerifyStateTargetClassified || target == VerifyStateUnsupportedDomain)
	case VerifyStateTargetClassified:
		valid = (target == VerifyStateResidualConstructed || target == VerifyStateUnsupportedDomain)
	case VerifyStateResidualConstructed:
		valid = (target == VerifyStateSimplificationEvaluated || target == VerifyStateRefuted)
	case VerifyStateSimplificationEvaluated:
		valid = (target == VerifyStateCertified || target == VerifyStateRefuted)
	case VerifyStateCertified, VerifyStateRefuted, VerifyStateUnsupportedDomain:
		valid = false // Terminal states
	}

	if !valid {
		return fmt.Errorf("invalid verification FSM transition: %s -> %s", fsm.currentState, target)
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}
