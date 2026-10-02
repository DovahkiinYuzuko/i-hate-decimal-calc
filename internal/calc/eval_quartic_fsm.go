package calc

import (
	"fmt"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// QuarticState represents a discrete lifecycle state in exact quartic polynomial solving.
type QuarticState int

const (
	QuarticStateInit                QuarticState = 0
	QuarticStateDegreeValidated     QuarticState = 1
	QuarticStateRationalDeflated    QuarticState = 2
	QuarticStateBiquadraticResolved QuarticState = 3
	QuarticStateDepressedNormalized QuarticState = 4
	QuarticStateResolventCubicSolved QuarticState = 5
	QuarticStateRootsConstructed    QuarticState = 6
	QuarticStateFailed              QuarticState = 7
)

func (s QuarticState) String() string {
	switch s {
	case QuarticStateInit:
		return "Init"
	case QuarticStateDegreeValidated:
		return "DegreeValidated"
	case QuarticStateRationalDeflated:
		return "RationalDeflated"
	case QuarticStateBiquadraticResolved:
		return "BiquadraticResolved"
	case QuarticStateDepressedNormalized:
		return "DepressedNormalized"
	case QuarticStateResolventCubicSolved:
		return "ResolventCubicSolved"
	case QuarticStateRootsConstructed:
		return "RootsConstructed"
	case QuarticStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("QuarticState(%d)", int(s))
	}
}

// QuarticLifecycleFSM governs valid state transitions in the quartic solver pipeline.
type QuarticLifecycleFSM struct {
	mu           sync.RWMutex
	currentState QuarticState
	history      []QuarticState
}

// NewQuarticLifecycleFSM creates a new QuarticLifecycleFSM initialized to QuarticStateInit.
func NewQuarticLifecycleFSM() *QuarticLifecycleFSM {
	return &QuarticLifecycleFSM{
		currentState: QuarticStateInit,
		history:      []QuarticState{QuarticStateInit},
	}
}

// CurrentState returns the current state.
func (fsm *QuarticLifecycleFSM) CurrentState() QuarticState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// History returns a copy of the state transition history.
func (fsm *QuarticLifecycleFSM) History() []QuarticState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	cp := make([]QuarticState, len(fsm.history))
	copy(cp, fsm.history)
	return cp
}

// TransitionTo validates and performs a state transition.
func (fsm *QuarticLifecycleFSM) TransitionTo(target QuarticState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case QuarticStateInit:
		valid = (target == QuarticStateDegreeValidated || target == QuarticStateFailed)
	case QuarticStateDegreeValidated:
		valid = (target == QuarticStateRationalDeflated ||
			target == QuarticStateBiquadraticResolved ||
			target == QuarticStateDepressedNormalized ||
			target == QuarticStateRootsConstructed ||
			target == QuarticStateFailed)
	case QuarticStateRationalDeflated:
		valid = (target == QuarticStateRootsConstructed || target == QuarticStateFailed)
	case QuarticStateBiquadraticResolved:
		valid = (target == QuarticStateRootsConstructed || target == QuarticStateFailed)
	case QuarticStateDepressedNormalized:
		valid = (target == QuarticStateResolventCubicSolved ||
			target == QuarticStateRootsConstructed ||
			target == QuarticStateFailed)
	case QuarticStateResolventCubicSolved:
		valid = (target == QuarticStateRootsConstructed || target == QuarticStateFailed)
	case QuarticStateRootsConstructed, QuarticStateFailed:
		valid = false // Terminal states
	}

	if !valid {
		return fmt.Errorf("%s", i18n.T("quartic.err_invalid_quartic_fsm_transition", fsm.currentState, target))
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}
