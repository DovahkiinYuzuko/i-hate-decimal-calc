package calc

import (
	"fmt"
)

func init() {
	RegisterHandler("laplace", handleLaplace)
	RegisterHandler("inv_laplace", handleInvLaplace)
	RegisterHandler("fourier_series", handleFourierSeries)
}

func handleLaplace(args []Node, env *Env) (Node, error) {
	var tName, sName string
	if len(args) >= 2 {
		if v, ok := args[1].(*VarNode); ok {
			tName = v.Name
		} else {
			return nil, fmt.Errorf("laplace error: second argument must be a variable name, got %s", args[1].String())
		}
	}
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			sName = v.Name
		} else {
			return nil, fmt.Errorf("laplace error: third argument must be a variable name, got %s", args[2].String())
		}
	}
	return EvalLaplace(args[0], tName, sName, env)
}

func handleInvLaplace(args []Node, env *Env) (Node, error) {
	var sName, tName string
	if len(args) >= 2 {
		if v, ok := args[1].(*VarNode); ok {
			sName = v.Name
		} else {
			return nil, fmt.Errorf("inv_laplace error: second argument must be a variable name, got %s", args[1].String())
		}
	}
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			tName = v.Name
		} else {
			return nil, fmt.Errorf("inv_laplace error: third argument must be a variable name, got %s", args[2].String())
		}
	}
	return EvalInvLaplace(args[0], sName, tName, env)
}

func handleFourierSeries(args []Node, env *Env) (Node, error) {
	return EvalFourierSeries(args, env)
}
