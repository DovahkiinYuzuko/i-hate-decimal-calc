package calc

import (
	"fmt"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// IntervalState represents discrete execution stages in adaptive interval refinement.
type IntervalState int

const (
	IntervalStateInit IntervalState = iota
	IntervalStateInitialEnclosure
	IntervalStateCheckingTolerance
	IntervalStateRefining
	IntervalStateConverged
	IntervalStateMaxStepsReached
	IntervalStateFailed
)

func (s IntervalState) String() string {
	switch s {
	case IntervalStateInit:
		return "Init"
	case IntervalStateInitialEnclosure:
		return "InitialEnclosure"
	case IntervalStateCheckingTolerance:
		return "CheckingTolerance"
	case IntervalStateRefining:
		return "Refining"
	case IntervalStateConverged:
		return "Converged"
	case IntervalStateMaxStepsReached:
		return "MaxStepsReached"
	case IntervalStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("IntervalState(%d)", int(s))
	}
}

// IntervalRefinerFSM governs and protects state transitions during adaptive interval refinement.
type IntervalRefinerFSM struct {
	mu           sync.RWMutex
	currentState IntervalState
	history      []IntervalState
	maxSteps     int
}

// NewIntervalRefinerFSM creates and initializes a new FSM in IntervalStateInit.
func NewIntervalRefinerFSM(maxSteps int) *IntervalRefinerFSM {
	if maxSteps <= 0 {
		maxSteps = 100
	}
	return &IntervalRefinerFSM{
		currentState: IntervalStateInit,
		history:      []IntervalState{IntervalStateInit},
		maxSteps:     maxSteps,
	}
}

// CurrentState returns the current state of the FSM thread-safely.
func (fsm *IntervalRefinerFSM) CurrentState() IntervalState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// History returns a copy of the state transition history.
func (fsm *IntervalRefinerFSM) History() []IntervalState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	cp := make([]IntervalState, len(fsm.history))
	copy(cp, fsm.history)
	return cp
}

// MaxSteps returns the maximum allowable refinement steps.
func (fsm *IntervalRefinerFSM) MaxSteps() int {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.maxSteps
}

// TransitionTo validates and performs a state transition.
func (fsm *IntervalRefinerFSM) TransitionTo(next IntervalState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case IntervalStateInit:
		valid = (next == IntervalStateInitialEnclosure || next == IntervalStateFailed)
	case IntervalStateInitialEnclosure:
		valid = (next == IntervalStateCheckingTolerance || next == IntervalStateFailed)
	case IntervalStateCheckingTolerance:
		valid = (next == IntervalStateConverged || next == IntervalStateRefining || next == IntervalStateMaxStepsReached || next == IntervalStateFailed)
	case IntervalStateRefining:
		valid = (next == IntervalStateCheckingTolerance || next == IntervalStateFailed)
	case IntervalStateConverged, IntervalStateMaxStepsReached, IntervalStateFailed:
		// Terminal states: no further transitions permitted
		valid = false
	}

	if !valid {
		return fmt.Errorf("%s", i18n.T("interval.err_invalid_fsm_transition", fsm.currentState.String(), next.String()))
	}

	fsm.currentState = next
	fsm.history = append(fsm.history, next)
	return nil
}

// IsTerminal returns true if the FSM has reached a terminal state.
func (fsm *IntervalRefinerFSM) IsTerminal() bool {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState == IntervalStateConverged ||
		fsm.currentState == IntervalStateMaxStepsReached ||
		fsm.currentState == IntervalStateFailed
}
