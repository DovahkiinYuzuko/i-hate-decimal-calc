package calc

import (
	"fmt"
)

func init() {
	RegisterHandler("det", handleDet)
	RegisterHandler("inv", handleInv)
	RegisterHandler("transpose", handleTranspose)
	RegisterHandler("rref", handleRREF)
	RegisterHandler("rank", handleRank)
	RegisterHandler("solve_linear", handleSolveLinear)
	RegisterHandler("linsolve", handleSolveLinear)
	RegisterHandler("trace", handleTrace)
	RegisterHandler("tr", handleTrace)
	RegisterHandler("eigenvals", handleEigenvals)
	RegisterHandler("eigenvects", handleEigenvects)
	RegisterHandler("dot", handleDot)
	RegisterHandler("cross", handleCross)
	RegisterHandler("norm", handleNorm)
	RegisterHandler("grad", handleGrad)
	RegisterHandler("div", handleDiv)
	RegisterHandler("curl", handleCurl)
}

func handleDet(args []Node, env *Env) (Node, error) {
	mat, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("det error: argument must be a matrix, got %s", args[0].String())
	}
	return evalDet(mat)
}

func handleInv(args []Node, env *Env) (Node, error) {
	mat, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("inv error: argument must be a matrix, got %s", args[0].String())
	}
	return evalInv(mat)
}

func handleTranspose(args []Node, env *Env) (Node, error) {
	mat, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("transpose error: argument must be a matrix, got %s", args[0].String())
	}
	return evalTranspose(mat), nil
}

func handleRREF(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("rref error: argument must be a matrix, got %s", args[0].String())
	}
	return evalRREF(mat)
}

func handleRank(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("rank error: argument must be a matrix, got %s", args[0].String())
	}
	return evalRank(mat)
}

func handleSolveLinear(args []Node, env *Env) (Node, error) {
	return evalSolveLinear(args[0], args[1])
}

func handleTrace(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("trace error: argument must be a matrix, got %s", args[0].String())
	}
	return EvalTrace(mat, env)
}

func handleEigenvals(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("eigenvals error: argument must be a matrix, got %s", args[0].String())
	}
	return EvalEigenvals(mat, env)
}

func handleEigenvects(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("eigenvects error: argument must be a matrix, got %s", args[0].String())
	}
	return EvalEigenvects(mat, env)
}

func handleDot(args []Node, env *Env) (Node, error) {
	return evalDot(args[0], args[1])
}

func handleCross(args []Node, env *Env) (Node, error) {
	return evalCross(args[0], args[1])
}

func handleNorm(args []Node, env *Env) (Node, error) {
	return evalNorm(args[0])
}

func handleGrad(args []Node, env *Env) (Node, error) {
	return evalGrad(args[0], args[1])
}

func handleDiv(args []Node, env *Env) (Node, error) {
	return evalDiv(args[0], args[1])
}

func handleCurl(args []Node, env *Env) (Node, error) {
	return evalCurl(args[0], args[1])
}
