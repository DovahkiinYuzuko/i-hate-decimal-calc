package calc

import (
	"fmt"
	"sync"
)

// RischState represents the transactional states of Bronstein's transcendental Risch integration algorithm.
type RischState int

const (
	RischStateIdle RischState = iota
	RischStateFieldTowerConstructed
	RischStateTowerValidated
	RischStateConstantFieldDetermined
	RischStateMonomialsNormalized
	RischStateHermiteReduced
	RischStateResiduePolesExtracted
	RischStatePolynomialPartIntegrated
	RischStateRischDESolved
	RischStateNonelementaryCertified
	RischStateUnsupported
	RischStateInvalidTower
)

func (s RischState) String() string {
	switch s {
	case RischStateIdle:
		return "Idle"
	case RischStateFieldTowerConstructed:
		return "FieldTowerConstructed"
	case RischStateTowerValidated:
		return "TowerValidated"
	case RischStateConstantFieldDetermined:
		return "ConstantFieldDetermined"
	case RischStateMonomialsNormalized:
		return "MonomialsNormalized"
	case RischStateHermiteReduced:
		return "HermiteReduced"
	case RischStateResiduePolesExtracted:
		return "ResiduePolesExtracted"
	case RischStatePolynomialPartIntegrated:
		return "PolynomialPartIntegrated"
	case RischStateRischDESolved:
		return "RischDESolved"
	case RischStateNonelementaryCertified:
		return "NonelementaryCertified"
	case RischStateUnsupported:
		return "Unsupported"
	case RischStateInvalidTower:
		return "InvalidTower"
	default:
		return fmt.Sprintf("Unknown(%d)", int(s))
	}
}

// NonelementaryIntegralError represents a mathematically certified proof that no elementary
// antiderivative exists in any Liouville extension of the base field (Liouville's theorem).
type NonelementaryIntegralError struct {
	Expr   Node
	Reason string
}

func (e *NonelementaryIntegralError) Error() string {
	if e.Expr == nil {
		return fmt.Sprintf("nonelementary integral: %s", e.Reason)
	}
	return fmt.Sprintf("nonelementary integral for %s: %s", e.Expr.String(), e.Reason)
}

// RischLifecycleFSM manages the transactional progression and mathematical proof invariant
// of the Risch decision procedure, strictly distinguishing between solvable, provably nonelementary,
// and unsupported algebraic/constant cases.
type RischLifecycleFSM struct {
	mu           sync.RWMutex
	state        RischState
	complete     bool
	history      []RischState
	reasonLog    string
}

// NewRischLifecycleFSM instantiates a fresh Risch lifecycle FSM in the Idle state.
func NewRischLifecycleFSM() *RischLifecycleFSM {
	return &RischLifecycleFSM{
		state:   RischStateIdle,
		history: []RischState{RischStateIdle},
	}
}

// CurrentState returns the active state of the FSM thread-safely.
func (f *RischLifecycleFSM) CurrentState() RischState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state
}

// SetCompleteness flags whether the algorithm's decision procedure is complete for the active tower.
func (f *RischLifecycleFSM) SetCompleteness(proven bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.complete = proven
}

// IsComplete reports whether the decision procedure guarantees mathematical completeness.
func (f *RischLifecycleFSM) IsComplete() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.complete
}

// TransitionTo validates and performs a state transition according to Bronstein's invariants.
func (f *RischLifecycleFSM) TransitionTo(next RischState) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Universal Sinks: Unsupported and InvalidTower can be entered from any non-terminal state
	if next == RischStateUnsupported || next == RischStateInvalidTower {
		f.state = next
		f.history = append(f.history, next)
		return nil
	}

	valid := false
	switch f.state {
	case RischStateIdle:
		// Rational-only paths start directly at HermiteReduced; tower paths start at FieldTowerConstructed
		valid = (next == RischStateFieldTowerConstructed || next == RischStateMonomialsNormalized || next == RischStateHermiteReduced)

	case RischStateFieldTowerConstructed:
		valid = (next == RischStateTowerValidated || next == RischStateInvalidTower)

	case RischStateTowerValidated:
		valid = (next == RischStateConstantFieldDetermined)

	case RischStateConstantFieldDetermined:
		valid = (next == RischStateMonomialsNormalized || next == RischStateHermiteReduced)

	case RischStateMonomialsNormalized:
		valid = (next == RischStateHermiteReduced)

	case RischStateHermiteReduced:
		valid = (next == RischStateResiduePolesExtracted || next == RischStatePolynomialPartIntegrated)

	case RischStateResiduePolesExtracted:
		valid = (next == RischStatePolynomialPartIntegrated || next == RischStateRischDESolved)

	case RischStatePolynomialPartIntegrated:
		valid = (next == RischStateRischDESolved)

	case RischStateRischDESolved:
		valid = (next == RischStateNonelementaryCertified)

	case RischStateNonelementaryCertified, RischStateUnsupported, RischStateInvalidTower:
		// Terminal states cannot transition further
		valid = false
	}

	if next == RischStateNonelementaryCertified && !f.complete {
		return fmt.Errorf("cannot transition to NonelementaryCertified: algorithmic completeness not proven for this tower")
	}

	if !valid {
		return fmt.Errorf("invalid Risch FSM transition: %s -> %s", f.state.String(), next.String())
	}

	f.state = next
	f.history = append(f.history, next)
	return nil
}
