package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// IterationState represents the lifecycle states of an iteration construct.
type IterationState int

const (
	StateIterInit IterationState = iota
	StateIterScopedSetup
	StateIterStep
	StateIterTeardown
	StateIterDone
	StateIterError
)

// IterationFSM controls and executes an iteration loop with guaranteed cleanup and dynamic scoping.
type IterationFSM struct {
	State   IterationState
	VarName string
	Start   *big.Rat
	End     *big.Rat
	Step    *big.Rat
	Env     *Env
	Err     error
}

// NewIterationFSM initializes a new iteration state machine.
func NewIterationFSM(varName string, start, end, step *big.Rat, env *Env) *IterationFSM {
	return &IterationFSM{
		State:   StateIterInit,
		VarName: varName,
		Start:   start,
		End:     end,
		Step:    step,
		Env:     env,
	}
}

// Run executes the iteration loop.
// For each step, stepFn is invoked with the current iteration value.
// If stepFn returns stop=true or an error, execution breaks immediately.
func (fsm *IterationFSM) Run(stepFn func(curr *big.Rat) (stop bool, err error)) error {
	fsm.State = StateIterInit

	// 1. Validation: step size cannot be zero
	if fsm.Step.Sign() == 0 {
		fsm.State = StateIterError
		fsm.Err = fmt.Errorf("%s", i18n.T("errors.iteration_step_zero", "iteration"))
		return fsm.Err
	}

	// 2. Check if iteration range is empty (step direction mismatch)
	diff := new(big.Rat).Sub(fsm.End, fsm.Start)
	if (fsm.Step.Sign() > 0 && diff.Sign() < 0) || (fsm.Step.Sign() < 0 && diff.Sign() > 0) {
		// Empty set / zero steps
		fsm.State = StateIterDone
		return nil
	}

	// 3. Setup dynamic scoping
	fsm.State = StateIterScopedSetup
	initVal := NewRationalFromBigRat(new(big.Rat).Set(fsm.Start))

	err := fsm.Env.WithScopedVar(fsm.VarName, initVal, func(setVar func(Node)) error {
		fsm.State = StateIterStep

		curr := new(big.Rat).Set(fsm.Start)
		step := fsm.Step

		for {
			// Check termination condition
			if step.Sign() > 0 && curr.Cmp(fsm.End) > 0 {
				break
			}
			if step.Sign() < 0 && curr.Cmp(fsm.End) < 0 {
				break
			}

			// Update variable in environment
			setVar(NewRationalFromBigRat(new(big.Rat).Set(curr)))

			// Execute step callback
			stop, err := stepFn(curr)
			if err != nil {
				fsm.State = StateIterError
				fsm.Err = err
				return err
			}
			if stop {
				break
			}

			// curr = curr + step
			curr = new(big.Rat).Add(curr, step)
		}

		fsm.State = StateIterTeardown
		return nil
	})

	if err != nil {
		fsm.State = StateIterError
		fsm.Err = err
		return err
	}

	fsm.State = StateIterDone
	return nil
}
