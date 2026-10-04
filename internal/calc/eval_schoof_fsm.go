package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// SchoofState represents the discrete execution phase of Schoof's algorithm FSM.
type SchoofState int

const (
	SchoofStateInit SchoofState = iota
	SchoofStateDirectCount
	SchoofStateSelectPrimes
	SchoofStateEvalL2
	SchoofStateLoopPrimes
	SchoofStateSynthesizeCRT
	SchoofStateHasseBounds
	SchoofStateCompleted
	SchoofStateFailed
)

// String returns a human-readable representation of SchoofState.
func (s SchoofState) String() string {
	switch s {
	case SchoofStateInit:
		return "Init"
	case SchoofStateDirectCount:
		return "DirectCount"
	case SchoofStateSelectPrimes:
		return "SelectPrimes"
	case SchoofStateEvalL2:
		return "EvalL2"
	case SchoofStateLoopPrimes:
		return "LoopPrimes"
	case SchoofStateSynthesizeCRT:
		return "SynthesizeCRT"
	case SchoofStateHasseBounds:
		return "HasseBounds"
	case SchoofStateCompleted:
		return "Completed"
	case SchoofStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// SchoofFSM governs and protects state transitions during Schoof's point counting.
type SchoofFSM struct {
	currentState SchoofState
	history      []SchoofState
}

// NewSchoofFSM initializes a new SchoofFSM in SchoofStateInit.
func NewSchoofFSM() *SchoofFSM {
	return &SchoofFSM{
		currentState: SchoofStateInit,
		history:      []SchoofState{SchoofStateInit},
	}
}

// State returns the current FSM state.
func (fsm *SchoofFSM) State() SchoofState {
	return fsm.currentState
}

// History returns a copy of state transition history.
func (fsm *SchoofFSM) History() []SchoofState {
	res := make([]SchoofState, len(fsm.history))
	copy(res, fsm.history)
	return res
}

// Transition validates and performs a state transition to target.
func (fsm *SchoofFSM) Transition(target SchoofState) error {
	if target == SchoofStateFailed {
		fsm.currentState = SchoofStateFailed
		fsm.history = append(fsm.history, SchoofStateFailed)
		return nil
	}

	valid := false
	switch fsm.currentState {
	case SchoofStateInit:
		valid = (target == SchoofStateDirectCount || target == SchoofStateSelectPrimes)
	case SchoofStateDirectCount:
		valid = (target == SchoofStateCompleted)
	case SchoofStateSelectPrimes:
		valid = (target == SchoofStateEvalL2)
	case SchoofStateEvalL2:
		valid = (target == SchoofStateLoopPrimes || target == SchoofStateSynthesizeCRT)
	case SchoofStateLoopPrimes:
		valid = (target == SchoofStateLoopPrimes || target == SchoofStateSynthesizeCRT)
	case SchoofStateSynthesizeCRT:
		valid = (target == SchoofStateHasseBounds)
	case SchoofStateHasseBounds:
		valid = (target == SchoofStateCompleted)
	case SchoofStateCompleted, SchoofStateFailed:
		valid = false
	}

	if !valid {
		err := fmt.Errorf("%s", i18n.T("elliptic.err_invalid_schoof_fsm_transition", fsm.currentState.String(), target.String()))
		fsm.currentState = SchoofStateFailed
		fsm.history = append(fsm.history, SchoofStateFailed)
		return err
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}
