package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterLazyHandler("table", handleTable)
	RegisterLazyHandler("product", handleProduct)
	RegisterLazyHandler("for", handleFor)
}

func parseIterationBounds(startNode, endNode, stepNode Node, env *Env, fnName string) (*big.Rat, *big.Rat, *big.Rat, error) {
	evalStart, err := EvalWithEnv(startNode, env)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, err))
	}
	rStart, okStart := evalStart.(*RationalNode)
	if !okStart {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, "start bound must evaluate to rational, got "+evalStart.String()))
	}

	evalEnd, err := EvalWithEnv(endNode, env)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, err))
	}
	rEnd, okEnd := evalEnd.(*RationalNode)
	if !okEnd {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, "end bound must evaluate to rational, got "+evalEnd.String()))
	}

	step := big.NewRat(1, 1)
	if stepNode != nil {
		evalStep, err := EvalWithEnv(stepNode, env)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, err))
		}
		rStep, okStep := evalStep.(*RationalNode)
		if !okStep {
			return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, "step must evaluate to rational, got "+evalStep.String()))
		}
		step = new(big.Rat).Set(rStep.Val)
	}

	return new(big.Rat).Set(rStart.Val), new(big.Rat).Set(rEnd.Val), step, nil
}

func handleTable(args []Node, env *Env) (Node, error) {
	if len(args) < 4 || len(args) > 5 {
		return nil, fmt.Errorf("table requires 4 or 5 arguments (expr, var, start, end, [step]), got %d", len(args))
	}

	expr := args[0]
	v, ok := args[1].(*VarNode)
	if !ok {
		if _, isComp := args[1].(*ComplexNode); isComp {
			return nil, fmt.Errorf("%s (note: 'i' is the imaginary unit, please use 'k', 'j', 'n', etc.)", i18n.T("errors.iteration_arg_must_be_var", "table", args[1].String()))
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.iteration_arg_must_be_var", "table", args[1].String()))
	}

	var stepNode Node
	if len(args) == 5 {
		stepNode = args[4]
	}

	start, end, step, err := parseIterationBounds(args[2], args[3], stepNode, env, "table")
	if err != nil {
		return nil, err
	}

	elements := make([]Node, 0)
	fsm := NewIterationFSM(v.Name, start, end, step, env)
	err = fsm.Run(func(curr *big.Rat) (bool, error) {
		val, err := EvalWithEnv(expr, env)
		if err != nil {
			return false, err
		}
		elements = append(elements, val)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return NewList(elements), nil
}

func handleProduct(args []Node, env *Env) (Node, error) {
	if len(args) < 4 || len(args) > 5 {
		return nil, fmt.Errorf("product requires 4 or 5 arguments (expr, var, start, end, [step]), got %d", len(args))
	}

	expr := args[0]
	v, ok := args[1].(*VarNode)
	if !ok {
		if _, isComp := args[1].(*ComplexNode); isComp {
			return nil, fmt.Errorf("%s (note: 'i' is the imaginary unit, please use 'k', 'j', 'n', etc.)", i18n.T("errors.iteration_arg_must_be_var", "product", args[1].String()))
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.iteration_arg_must_be_var", "product", args[1].String()))
	}

	var stepNode Node
	if len(args) == 5 {
		stepNode = args[4]
	}

	start, end, step, err := parseIterationBounds(args[2], args[3], stepNode, env, "product")
	if err != nil {
		return nil, err
	}

	acc := Node(mustRational(1, 1))
	fsm := NewIterationFSM(v.Name, start, end, step, env)
	err = fsm.Run(func(curr *big.Rat) (bool, error) {
		val, err := EvalWithEnv(expr, env)
		if err != nil {
			return false, err
		}

		if r, ok := val.(*RationalNode); ok && r.Val.Sign() == 0 {
			acc = mustRational(0, 1)
			return true, nil
		}

		acc, err = simplifyMul([]Node{acc, val})
		if err != nil {
			acc = NewMul([]Node{acc, val})
		}

		if r, ok := acc.(*RationalNode); ok && r.Val.Sign() == 0 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return acc, nil
}

func handleFor(args []Node, env *Env) (Node, error) {
	if len(args) < 4 || len(args) > 5 {
		return nil, fmt.Errorf("for requires 4 or 5 arguments (var, start, end, [step], body), got %d", len(args))
	}

	v, ok := args[0].(*VarNode)
	if !ok {
		if _, isComp := args[0].(*ComplexNode); isComp {
			return nil, fmt.Errorf("%s (note: 'i' is the imaginary unit, please use 'k', 'j', 'n', etc.)", i18n.T("errors.iteration_arg_must_be_var", "for", args[0].String()))
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.iteration_arg_must_be_var", "for", args[0].String()))
	}

	var stepNode Node
	var bodyNode Node
	if len(args) == 4 {
		stepNode = nil
		bodyNode = args[3]
	} else {
		stepNode = args[3]
		bodyNode = args[4]
	}

	start, end, step, err := parseIterationBounds(args[1], args[2], stepNode, env, "for")
	if err != nil {
		return nil, err
	}

	lastRes := Node(mustRational(0, 1))
	fsm := NewIterationFSM(v.Name, start, end, step, env)
	err = fsm.Run(func(curr *big.Rat) (bool, error) {
		val, err := EvalWithEnv(bodyNode, env)
		if err != nil {
			return false, err
		}
		lastRes = val
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return lastRes, nil
}
