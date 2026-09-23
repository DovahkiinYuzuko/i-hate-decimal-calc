package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"sync"
)

// QeState represents a discrete phase in the Quantifier Elimination lifecycle.
type QeState int

const (
	QeStateInit               QeState = 0
	QeStatePrenexNormalized   QeState = 1
	QeStateVariableOrdered    QeState = 2
	QeStateCadDecomposed      QeState = 3
	QeStateTruthEvaluated     QeState = 4
	QeStateFormulaConstructed QeState = 5
	QeStateFailed             QeState = 6
)

func (s QeState) String() string {
	switch s {
	case QeStateInit:
		return "Init"
	case QeStatePrenexNormalized:
		return "PrenexNormalized"
	case QeStateVariableOrdered:
		return "VariableOrdered"
	case QeStateCadDecomposed:
		return "CadDecomposed"
	case QeStateTruthEvaluated:
		return "TruthEvaluated"
	case QeStateFormulaConstructed:
		return "FormulaConstructed"
	case QeStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("QeState(%d)", int(s))
	}
}

// QeLifecycleFSM governs state transitions and validation invariants in the QE pipeline.
type QeLifecycleFSM struct {
	mu           sync.RWMutex
	currentState QeState
	history      []QeState
}

// NewQeLifecycleFSM creates a new QE lifecycle FSM initialized to QeStateInit.
func NewQeLifecycleFSM() *QeLifecycleFSM {
	return &QeLifecycleFSM{
		currentState: QeStateInit,
		history:      []QeState{QeStateInit},
	}
}

// CurrentState returns the current state of the FSM.
func (fsm *QeLifecycleFSM) CurrentState() QeState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// History returns a copy of the transition history.
func (fsm *QeLifecycleFSM) History() []QeState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	cp := make([]QeState, len(fsm.history))
	copy(cp, fsm.history)
	return cp
}

// TransitionTo validates and performs a state transition to target.
func (fsm *QeLifecycleFSM) TransitionTo(target QeState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case QeStateInit:
		valid = (target == QeStatePrenexNormalized || target == QeStateFailed)
	case QeStatePrenexNormalized:
		valid = (target == QeStateVariableOrdered || target == QeStateFormulaConstructed || target == QeStateFailed)
	case QeStateVariableOrdered:
		valid = (target == QeStateCadDecomposed || target == QeStateFormulaConstructed || target == QeStateFailed)
	case QeStateCadDecomposed:
		valid = (target == QeStateTruthEvaluated || target == QeStateFailed)
	case QeStateTruthEvaluated:
		valid = (target == QeStateFormulaConstructed || target == QeStateFailed)
	case QeStateFormulaConstructed, QeStateFailed:
		valid = false // Terminal states
	}

	if !valid {
		return fmt.Errorf("%s", i18n.T("qe.err_invalid_qe_fsm_transition", fsm.currentState, target))
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}
