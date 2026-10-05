package calc

import (
	"fmt"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// DiophantineState represents the operational states of the Diophantine Equation Solver.
type DiophantineState string

const (
	// DiophantineStateInit represents the initial setup state.
	DiophantineStateInit DiophantineState = "INIT"
	// DiophantineStateClassify identifies the equation category (Linear, Pell, Pythagorean).
	DiophantineStateClassify DiophantineState = "CLASSIFY"
	// DiophantineStateSolveLinear executes the extended Euclidean solver for linear equations.
	DiophantineStateSolveLinear DiophantineState = "SOLVE_LINEAR"
	// DiophantineStateSolvePell executes continued fraction convergence for Pell equations.
	DiophantineStateSolvePell DiophantineState = "SOLVE_PELL"
	// DiophantineStateSolvePythagorean applies parametric formulas for Pythagorean equations.
	DiophantineStateSolvePythagorean DiophantineState = "SOLVE_PYTHAGOREAN"
	// DiophantineStateCertify constructs Certificate IR and verifies algebraic residual.
	DiophantineStateCertify DiophantineState = "CERTIFY"
	// DiophantineStateSuccess indicates successful solution generation.
	DiophantineStateSuccess DiophantineState = "SUCCESS"
	// DiophantineStateFailure indicates insolvability or invalid equation form.
	DiophantineStateFailure DiophantineState = "FAILURE"
)

// DiophantineEquationType specifies the mathematical classification of the equation.
type DiophantineEquationType string

const (
	EquationTypeLinear      DiophantineEquationType = "LINEAR"
	EquationTypePell        DiophantineEquationType = "PELL"
	EquationTypePythagorean DiophantineEquationType = "PYTHAGOREAN"
	EquationTypeUnknown     DiophantineEquationType = "UNKNOWN"
)

// DiophantineSolverFSM governs and protects state transitions during Diophantine equation solving.
type DiophantineSolverFSM struct {
	mu           sync.RWMutex
	current      DiophantineState
	history      []DiophantineState
	validTrans   map[DiophantineState]map[DiophantineState]bool
	EqType       DiophantineEquationType
	maxLoopCount int
	loopCounter  int
}

// NewDiophantineSolverFSM initializes a new FSM in the INIT state with bounded state transition rules.
func NewDiophantineSolverFSM() *DiophantineSolverFSM {
	fsm := &DiophantineSolverFSM{
		current:      DiophantineStateInit,
		history:      []DiophantineState{DiophantineStateInit},
		EqType:       EquationTypeUnknown,
		maxLoopCount: 100,
		validTrans:   make(map[DiophantineState]map[DiophantineState]bool),
	}

	allow := func(from DiophantineState, to ...DiophantineState) {
		if fsm.validTrans[from] == nil {
			fsm.validTrans[from] = make(map[DiophantineState]bool)
		}
		for _, target := range to {
			fsm.validTrans[from][target] = true
		}
	}

	allow(DiophantineStateInit, DiophantineStateClassify, DiophantineStateFailure)
	allow(DiophantineStateClassify, DiophantineStateSolveLinear, DiophantineStateSolvePell, DiophantineStateSolvePythagorean, DiophantineStateFailure)
	allow(DiophantineStateSolveLinear, DiophantineStateCertify, DiophantineStateSuccess, DiophantineStateFailure)
	allow(DiophantineStateSolvePell, DiophantineStateCertify, DiophantineStateSuccess, DiophantineStateFailure)
	allow(DiophantineStateSolvePythagorean, DiophantineStateCertify, DiophantineStateSuccess, DiophantineStateFailure)
	allow(DiophantineStateCertify, DiophantineStateSuccess, DiophantineStateFailure)
	allow(DiophantineStateSuccess)
	allow(DiophantineStateFailure)

	return fsm
}

// Current returns the active state of the FSM.
func (f *DiophantineSolverFSM) Current() DiophantineState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current
}

// Transition moves the FSM to the target state if the transition is valid.
func (f *DiophantineSolverFSM) Transition(next DiophantineState) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.loopCounter++
	if f.loopCounter > f.maxLoopCount {
		f.current = DiophantineStateFailure
		return fmt.Errorf("%s", i18n.T("diophantine.err_invalid_fsm_transition", f.current, next))
	}

	if !f.validTrans[f.current][next] {
		prev := f.current
		f.current = DiophantineStateFailure
		return fmt.Errorf("%s", i18n.T("diophantine.err_invalid_fsm_transition", string(prev), string(next)))
	}

	f.current = next
	f.history = append(f.history, next)
	return nil
}
