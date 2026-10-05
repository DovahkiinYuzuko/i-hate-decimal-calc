package calc

import (
	"fmt"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// TranscendentalState represents discrete execution stages in solving transcendental equations.
type TranscendentalState int

const (
	TranscendentalStateInit TranscendentalState = iota
	TranscendentalStateClassified
	TranscendentalStateNormalized
	TranscendentalStateLambertInverted
	TranscendentalStateRootIsolated
	TranscendentalStateVerified
	TranscendentalStateFailed
)

func (s TranscendentalState) String() string {
	switch s {
	case TranscendentalStateInit:
		return "Init"
	case TranscendentalStateClassified:
		return "Classified"
	case TranscendentalStateNormalized:
		return "Normalized"
	case TranscendentalStateLambertInverted:
		return "LambertInverted"
	case TranscendentalStateRootIsolated:
		return "RootIsolated"
	case TranscendentalStateVerified:
		return "Verified"
	case TranscendentalStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("TranscendentalState(%d)", int(s))
	}
}

// TranscendentalSolverFSM governs and protects state transitions during transcendental equation solving.
type TranscendentalSolverFSM struct {
	mu           sync.RWMutex
	currentState TranscendentalState
	history      []TranscendentalState
}

// NewTranscendentalSolverFSM creates and initializes a new FSM in TranscendentalStateInit.
func NewTranscendentalSolverFSM() *TranscendentalSolverFSM {
	return &TranscendentalSolverFSM{
		currentState: TranscendentalStateInit,
		history:      []TranscendentalState{TranscendentalStateInit},
	}
}

// CurrentState returns the current state of the FSM thread-safely.
func (fsm *TranscendentalSolverFSM) CurrentState() TranscendentalState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// History returns a copy of the state transition history.
func (fsm *TranscendentalSolverFSM) History() []TranscendentalState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	cp := make([]TranscendentalState, len(fsm.history))
	copy(cp, fsm.history)
	return cp
}

// TransitionTo validates and advances the FSM to target state.
func (fsm *TranscendentalSolverFSM) TransitionTo(target TranscendentalState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case TranscendentalStateInit:
		valid = (target == TranscendentalStateClassified || target == TranscendentalStateFailed)
	case TranscendentalStateClassified:
		valid = (target == TranscendentalStateNormalized || target == TranscendentalStateFailed)
	case TranscendentalStateNormalized:
		valid = (target == TranscendentalStateLambertInverted || target == TranscendentalStateFailed)
	case TranscendentalStateLambertInverted:
		valid = (target == TranscendentalStateRootIsolated || target == TranscendentalStateFailed)
	case TranscendentalStateRootIsolated:
		valid = (target == TranscendentalStateVerified || target == TranscendentalStateFailed)
	case TranscendentalStateVerified, TranscendentalStateFailed:
		valid = false
	}

	if !valid {
		return fmt.Errorf("%s", i18n.T("transcendental.err_invalid_fsm_transition", fsm.currentState.String(), target.String()))
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}
