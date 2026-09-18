package calc

import (
	"fmt"
	"sync"
)

// EcState represents a discrete phase in the Elliptic Curve & Nagell-Lutz pipeline lifecycle.
type EcState int

const (
	EcStateInit               EcState = 0
	EcStateNormalized         EcState = 1
	EcStateDiscrimFactored    EcState = 2
	EcStateCandidatesFound    EcState = 3
	EcStateOrderVerified      EcState = 4
	EcStateGroupDetermined    EcState = 5
	EcStateFailed             EcState = 6
)

func (s EcState) String() string {
	switch s {
	case EcStateInit:
		return "Init"
	case EcStateNormalized:
		return "Normalized"
	case EcStateDiscrimFactored:
		return "DiscrimFactored"
	case EcStateCandidatesFound:
		return "CandidatesFound"
	case EcStateOrderVerified:
		return "OrderVerified"
	case EcStateGroupDetermined:
		return "GroupDetermined"
	case EcStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("EcState(%d)", int(s))
	}
}

// EllipticLifecycleFSM governs state transitions and validation invariants for elliptic operations.
type EllipticLifecycleFSM struct {
	mu           sync.RWMutex
	currentState EcState
	history      []EcState
}

// NewEllipticLifecycleFSM creates a new EllipticLifecycleFSM initialized to EcStateInit.
func NewEllipticLifecycleFSM() *EllipticLifecycleFSM {
	return &EllipticLifecycleFSM{
		currentState: EcStateInit,
		history:      []EcState{EcStateInit},
	}
}

// CurrentState returns the current state of the FSM.
func (fsm *EllipticLifecycleFSM) CurrentState() EcState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// History returns a copy of the transition history.
func (fsm *EllipticLifecycleFSM) History() []EcState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	cp := make([]EcState, len(fsm.history))
	copy(cp, fsm.history)
	return cp
}

// TransitionTo validates and performs a state transition to target.
func (fsm *EllipticLifecycleFSM) TransitionTo(target EcState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case EcStateInit:
		valid = (target == EcStateNormalized || target == EcStateFailed)
	case EcStateNormalized:
		valid = (target == EcStateDiscrimFactored || target == EcStateGroupDetermined || target == EcStateFailed)
	case EcStateDiscrimFactored:
		valid = (target == EcStateCandidatesFound || target == EcStateFailed)
	case EcStateCandidatesFound:
		valid = (target == EcStateOrderVerified || target == EcStateFailed)
	case EcStateOrderVerified:
		valid = (target == EcStateGroupDetermined || target == EcStateFailed)
	case EcStateGroupDetermined, EcStateFailed:
		valid = false // Terminal states
	}

	if !valid {
		return fmt.Errorf("invalid Elliptic FSM transition: %s -> %s", fsm.currentState, target)
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}
