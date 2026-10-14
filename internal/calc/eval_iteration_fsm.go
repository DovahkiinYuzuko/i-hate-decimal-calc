package calc

import (
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// IterationState represents the lifecycle states of an iteration construct.
type IterationState int

const (
	StateIterInit IterationState = iota
	StateIterScopedSetup
	StateIterStep
	StateIterWarn
	StateIterTeardown
	StateIterDone
	StateIterError
)

// IterationFSM controls and executes an iteration loop with guaranteed cleanup and dynamic scoping.
type IterationFSM struct {
	State          IterationState
	VarName        string
	Start          *big.Rat
	End            *big.Rat
	Step           *big.Rat
	Env            *Env
	Err            error
	WarnIterations *big.Int
	MaxIterations  *big.Int
	WarnWriter     io.Writer
	StepCount      *big.Int
	Warned         bool
}

// NewIterationFSM initializes a new iteration state machine with defaults from config.
func NewIterationFSM(varName string, start, end, step *big.Rat, env *Env) *IterationFSM {
	cfg, _ := i18n.LoadConfig()
	warnLimit := i18n.GetWarnIterationsBig(cfg)
	maxLimit := i18n.GetMaxIterationsBig(cfg)

	return &IterationFSM{
		State:          StateIterInit,
		VarName:        varName,
		Start:          start,
		End:            end,
		Step:           step,
		Env:            env,
		WarnIterations: warnLimit,
		MaxIterations:  maxLimit,
		WarnWriter:     os.Stderr,
		StepCount:      big.NewInt(0),
		Warned:         false,
	}
}

// SetLimits configures custom iteration limits and warning output writer.
func (fsm *IterationFSM) SetLimits(warnLimit, maxLimit *big.Int, warnWriter io.Writer) {
	if warnLimit != nil {
		fsm.WarnIterations = new(big.Int).Set(warnLimit)
	}
	if maxLimit != nil {
		fsm.MaxIterations = new(big.Int).Set(maxLimit)
	}
	if warnWriter != nil {
		fsm.WarnWriter = warnWriter
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

			// Increment step counter
			fsm.StepCount.Add(fsm.StepCount, big.NewInt(1))

			// Check maximum iterations limit
			if fsm.MaxIterations != nil && fsm.MaxIterations.Sign() > 0 && fsm.StepCount.Cmp(fsm.MaxIterations) > 0 {
				fsm.State = StateIterError
				fsm.Err = fmt.Errorf("%s", i18n.T("errors.iteration_max_exceeded", fsm.VarName, fsm.MaxIterations.String()))
				return fsm.Err
			}

			// Check non-destructive warning threshold
			if fsm.WarnIterations != nil && fsm.StepCount.Cmp(fsm.WarnIterations) > 0 && !fsm.Warned {
				prevState := fsm.State
				fsm.State = StateIterWarn
				fsm.Warned = true
				if fsm.WarnWriter != nil {
					fmt.Fprintf(fsm.WarnWriter, "%s\n", i18n.T("warnings.iteration_limit_exceeded", fsm.VarName, fsm.WarnIterations.String()))
				}
				fsm.State = prevState
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
