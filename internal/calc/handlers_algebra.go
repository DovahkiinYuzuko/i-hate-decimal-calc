package calc

import (
	"fmt"
)

func init() {
	RegisterHandler("poly_gcd", handlePolyGCD)
	RegisterHandler("poly_lcm", handlePolyLCM)
	RegisterHandler("resultant", handleResultant)
	RegisterHandler("groebner", handleGroebner)
}

func handlePolyGCD(args []Node, env *Env) (Node, error) {
	var vName string
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			vName = v.Name
		} else {
			return nil, fmt.Errorf("poly_gcd error: third argument must be a variable name, got %s", args[2].String())
		}
	}
	return EvalPolyGCD(args[0], args[1], vName, env)
}

func handlePolyLCM(args []Node, env *Env) (Node, error) {
	var vName string
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			vName = v.Name
		} else {
			return nil, fmt.Errorf("poly_lcm error: third argument must be a variable name, got %s", args[2].String())
		}
	}
	return EvalPolyLCM(args[0], args[1], vName, env)
}

func handleResultant(args []Node, env *Env) (Node, error) {
	var vName string
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			vName = v.Name
		} else {
			return nil, fmt.Errorf("resultant error: third argument must be a variable name, got %s", args[2].String())
		}
	}
	return EvalResultant(args[0], args[1], vName, env)
}

func handleGroebner(args []Node, env *Env) (Node, error) {
	var orderOpt Node
	if len(args) >= 3 {
		orderOpt = args[2]
	}
	return EvalGroebner(args[0], args[1], orderOpt, env)
}
