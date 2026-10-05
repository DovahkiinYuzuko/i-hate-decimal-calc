package calc

import (
	"fmt"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// PrimitiveElementState represents operational states of the Primitive Element Solver FSM.
type PrimitiveElementState string

const (
	// PrimitiveStateInit represents initial parameter verification and state setup.
	PrimitiveStateInit PrimitiveElementState = "INIT"
	// PrimitiveStateScanningC represents deterministic selection and incrementation of scalar c.
	PrimitiveStateScanningC PrimitiveElementState = "SCANNING_C"
	// PrimitiveStateResultant represents computation of the elimination polynomial via Sylvester resultant.
	PrimitiveStateResultant PrimitiveElementState = "RESULTANT"
	// PrimitiveStateSquareFreeCheck represents square-free and degree verification of candidate minimal polynomial.
	PrimitiveStateSquareFreeCheck PrimitiveElementState = "SQUARE_FREE_CHECK"
	// PrimitiveStateSolveGenerators represents recovery of generator polynomial representations via field GCD.
	PrimitiveStateSolveGenerators PrimitiveElementState = "SOLVE_GENERATORS"
	// PrimitiveStateCertify constructs algebraic quotient certificates and verifies relations.
	PrimitiveStateCertify PrimitiveElementState = "CERTIFY"
	// PrimitiveStateSuccess indicates successful primitive element derivation.
	PrimitiveStateSuccess PrimitiveElementState = "SUCCESS"
	// PrimitiveStateFailure indicates inability to find a valid primitive element within limits.
	PrimitiveStateFailure PrimitiveElementState = "FAILURE"
)

// PrimitiveElementFSM coordinates and enforces valid state transitions during primitive element computation.
type PrimitiveElementFSM struct {
	mu           sync.RWMutex
	current      PrimitiveElementState
	history      []PrimitiveElementState
	validTrans   map[PrimitiveElementState]map[PrimitiveElementState]bool
	maxLoopCount int
	loopCounter  int
	CandidateC   int64
}

// NewPrimitiveElementFSM initializes a new PrimitiveElementFSM in the INIT state.
func NewPrimitiveElementFSM() *PrimitiveElementFSM {
	fsm := &PrimitiveElementFSM{
		current:      PrimitiveStateInit,
		history:      []PrimitiveElementState{PrimitiveStateInit},
		maxLoopCount: 150,
		validTrans:   make(map[PrimitiveElementState]map[PrimitiveElementState]bool),
		CandidateC:   1,
	}

	allow := func(from PrimitiveElementState, to ...PrimitiveElementState) {
		if fsm.validTrans[from] == nil {
			fsm.validTrans[from] = make(map[PrimitiveElementState]bool)
		}
		for _, target := range to {
			fsm.validTrans[from][target] = true
		}
	}

	allow(PrimitiveStateInit, PrimitiveStateScanningC, PrimitiveStateFailure)
	allow(PrimitiveStateScanningC, PrimitiveStateResultant, PrimitiveStateFailure)
	allow(PrimitiveStateResultant, PrimitiveStateSquareFreeCheck, PrimitiveStateFailure)
	allow(PrimitiveStateSquareFreeCheck, PrimitiveStateSolveGenerators, PrimitiveStateScanningC, PrimitiveStateFailure)
	allow(PrimitiveStateSolveGenerators, PrimitiveStateCertify, PrimitiveStateSuccess, PrimitiveStateScanningC, PrimitiveStateFailure)
	allow(PrimitiveStateCertify, PrimitiveStateSuccess, PrimitiveStateFailure)
	allow(PrimitiveStateSuccess)
	allow(PrimitiveStateFailure)

	return fsm
}

// Current returns the active state of the FSM.
func (f *PrimitiveElementFSM) Current() PrimitiveElementState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current
}

// CurrentState returns the active state of the FSM (alias for Current).
func (f *PrimitiveElementFSM) CurrentState() PrimitiveElementState {
	return f.Current()
}

// History returns a copy of the state transition history.
func (f *PrimitiveElementFSM) History() []PrimitiveElementState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	copied := make([]PrimitiveElementState, len(f.history))
	copy(copied, f.history)
	return copied
}

// Transition moves the FSM to nextState if allowed under transition governance.
func (f *PrimitiveElementFSM) Transition(next PrimitiveElementState) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.loopCounter++
	if f.loopCounter > f.maxLoopCount {
		f.current = PrimitiveStateFailure
		return fmt.Errorf("%s", i18n.T("primitive_element.err_invalid_fsm_transition", f.current, next))
	}

	if !f.validTrans[f.current][next] {
		f.current = PrimitiveStateFailure
		return fmt.Errorf("%s", i18n.T("primitive_element.err_invalid_fsm_transition", f.current, next))
	}

	f.current = next
	f.history = append(f.history, next)
	return nil
}
