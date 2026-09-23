package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"sync"
)

// GosperState represents the lifecycle states of the Gosper / WZ summation pipeline.
type GosperState int

const (
	// GosperStateIdle indicates the summation task is not yet started.
	GosperStateIdle GosperState = iota
	// GosperStateRatioExtracted indicates the consecutive term ratio r(k) = t_{k+1}/t_k was successfully extracted.
	GosperStateRatioExtracted
	// GosperStateNormalized indicates Gosper-Petkovsek normal form (a, b, c) was constructed via iterative gcd-peel.
	GosperStateNormalized
	// GosperStateDegreeBounded indicates the exact degree bound d for the polynomial equation was determined.
	GosperStateDegreeBounded
	// GosperStateSolved indicates polynomial y(k) was found and verified by resubstitution.
	GosperStateSolved
	// GosperStateNotSummable indicates the term is deterministically proven not to be Gosper-summable.
	GosperStateNotSummable
	// GosperStateCertified indicates a Wilf-Zeilberger (WZ) certificate was generated and verified.
	GosperStateCertified
)

// String returns a human-readable name of the Gosper pipeline state.
func (s GosperState) String() string {
	switch s {
	case GosperStateIdle:
		return "Idle"
	case GosperStateRatioExtracted:
		return "RatioExtracted"
	case GosperStateNormalized:
		return "Normalized"
	case GosperStateDegreeBounded:
		return "DegreeBounded"
	case GosperStateSolved:
		return "Solved"
	case GosperStateNotSummable:
		return "NotSummable"
	case GosperStateCertified:
		return "Certified"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// GosperLifecycleFSM guards the execution order and termination status of Gosper/WZ algorithms.
type GosperLifecycleFSM struct {
	mu    sync.RWMutex
	state GosperState
}

// NewGosperLifecycleFSM creates a new FSM initialized to GosperStateIdle.
func NewGosperLifecycleFSM() *GosperLifecycleFSM {
	return &GosperLifecycleFSM{
		state: GosperStateIdle,
	}
}

// State returns the current state of the FSM.
func (fsm *GosperLifecycleFSM) State() GosperState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.state
}

// TransitionTo validates and performs a state transition.
func (fsm *GosperLifecycleFSM) TransitionTo(next GosperState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	switch next {
	case GosperStateRatioExtracted:
		if fsm.state != GosperStateIdle {
			return fmt.Errorf("%s", i18n.T("gosper.err_invalid_transition_to_ratioextracted_from", fsm.state))
		}
	case GosperStateNormalized:
		if fsm.state != GosperStateRatioExtracted {
			return fmt.Errorf("%s", i18n.T("gosper.err_invalid_transition_to_normalized_from", fsm.state))
		}
	case GosperStateDegreeBounded:
		if fsm.state != GosperStateNormalized {
			return fmt.Errorf("%s", i18n.T("gosper.err_invalid_transition_to_degreebounded_from", fsm.state))
		}
	case GosperStateSolved:
		if fsm.state != GosperStateDegreeBounded {
			return fmt.Errorf("%s", i18n.T("gosper.err_invalid_transition_to_solved_from", fsm.state))
		}
	case GosperStateNotSummable:
		// Can transition to NotSummable from any intermediate calculation phase
		if fsm.state == GosperStateSolved || fsm.state == GosperStateCertified {
			return fmt.Errorf("%s", i18n.T("gosper.err_cannot_mark_notsummable_once_solved", fsm.state))
		}
	case GosperStateCertified:
		if fsm.state != GosperStateSolved {
			return fmt.Errorf("%s", i18n.T("gosper.err_invalid_transition_to_certified_from", fsm.state))
		}
	default:
		return fmt.Errorf("%s", i18n.T("gosper.err_unknown_target_state", next))
	}

	fsm.state = next
	return nil
}
