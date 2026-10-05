package calc

import (
	"fmt"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// SNFState represents the operational states of the Smith Normal Form algorithm.
type SNFState string

const (
	// SNFStateInit represents the initial setup state before reduction.
	SNFStateInit SNFState = "INIT"
	// SNFStateSelectPivot locates the non-zero entry with minimum absolute value in the active submatrix.
	SNFStateSelectPivot SNFState = "SELECT_PIVOT"
	// SNFStateEliminateRowCol eliminates non-pivot entries along the active row and column via Euclidean reductions.
	SNFStateEliminateRowCol SNFState = "ELIMINATE_ROW_COL"
	// SNFStateCheckDivisibility verifies whether the current pivot divides all entries in the remaining submatrix.
	SNFStateCheckDivisibility SNFState = "CHECK_DIVISIBILITY"
	// SNFStateFixDivisibility adds an indivisible row to the pivot row to force GCD reduction.
	SNFStateFixDivisibility SNFState = "FIX_DIVISIBILITY"
	// SNFStateNextBlock advances the reduction frontier to the next diagonal element.
	SNFStateNextBlock SNFState = "NEXT_BLOCK"
	// SNFStateSuccess indicates successful derivation of SNF satisfying the divisibility chain.
	SNFStateSuccess SNFState = "SUCCESS"
	// SNFStateFailure indicates an unrecoverable failure during calculation.
	SNFStateFailure SNFState = "FAILURE"
)

// SmithNormalFormFSM governs and protects state transitions during Smith Normal Form computation.
type SmithNormalFormFSM struct {
	mu           sync.RWMutex
	current      SNFState
	history      []SNFState
	validTrans   map[SNFState]map[SNFState]bool
	maxLoopCount int
	loopCounter  int
}

// NewSmithNormalFormFSM initializes a new FSM in the INIT state with bounded loop protection.
func NewSmithNormalFormFSM() *SmithNormalFormFSM {
	fsm := &SmithNormalFormFSM{
		current:      SNFStateInit,
		history:      []SNFState{SNFStateInit},
		maxLoopCount: 2000,
		validTrans:   make(map[SNFState]map[SNFState]bool),
	}

	allow := func(from SNFState, to ...SNFState) {
		if fsm.validTrans[from] == nil {
			fsm.validTrans[from] = make(map[SNFState]bool)
		}
		for _, target := range to {
			fsm.validTrans[from][target] = true
		}
	}

	allow(SNFStateInit, SNFStateSelectPivot, SNFStateSuccess, SNFStateFailure)
	allow(SNFStateSelectPivot, SNFStateEliminateRowCol, SNFStateSuccess, SNFStateFailure)
	allow(SNFStateEliminateRowCol, SNFStateEliminateRowCol, SNFStateCheckDivisibility, SNFStateFailure)
	allow(SNFStateCheckDivisibility, SNFStateFixDivisibility, SNFStateNextBlock, SNFStateFailure)
	allow(SNFStateFixDivisibility, SNFStateEliminateRowCol, SNFStateFailure)
	allow(SNFStateNextBlock, SNFStateSelectPivot, SNFStateSuccess, SNFStateFailure)

	return fsm
}

// CurrentState returns the current state of the FSM thread-safely.
func (fsm *SmithNormalFormFSM) CurrentState() SNFState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.current
}

// History returns a copy of the state transition history thread-safely.
func (fsm *SmithNormalFormFSM) History() []SNFState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	copied := make([]SNFState, len(fsm.history))
	copy(copied, fsm.history)
	return copied
}

// TransitionTo validates and performs a state transition.
func (fsm *SmithNormalFormFSM) TransitionTo(target SNFState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	allowedTargets := fsm.validTrans[fsm.current]
	if allowedTargets == nil || !allowedTargets[target] {
		return fmt.Errorf("%s", i18n.T("matrix_integer.err_invalid_snf_fsm_transition", string(fsm.current), string(target)))
	}

	fsm.loopCounter++
	if fsm.loopCounter > fsm.maxLoopCount {
		fsm.current = SNFStateFailure
		fsm.history = append(fsm.history, SNFStateFailure)
		return fmt.Errorf("smith normal form reduction exceeded maximum iterations (%d)", fsm.maxLoopCount)
	}

	fsm.current = target
	fsm.history = append(fsm.history, target)
	return nil
}
