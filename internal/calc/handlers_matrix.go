package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
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
	RegisterHandler("lu", handleLU)
	RegisterHandler("qr", handleQR)
	RegisterHandler("cholesky", handleCholesky)
	RegisterHandler("ldlt", handleLDLT)
	RegisterHandler("pinv", handlePinv)
}

func handleDet(args []Node, env *Env) (Node, error) {
	mat, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "det", args[0].String()))
	}
	return evalDet(mat)
}

func handleInv(args []Node, env *Env) (Node, error) {
	mat, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "inv", args[0].String()))
	}
	return evalInv(mat)
}

func handleTranspose(args []Node, env *Env) (Node, error) {
	mat, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "transpose", args[0].String()))
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
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "rref", args[0].String()))
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
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "rank", args[0].String()))
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
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "trace", args[0].String()))
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
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "eigenvals", args[0].String()))
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
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "eigenvects", args[0].String()))
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

func handleLU(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "lu", args[0].String()))
	}
	return evalLU(mat)
}

func handleQR(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "qr", args[0].String()))
	}
	return evalTwoStageQR(mat)
}

func handleCholesky(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "cholesky", args[0].String()))
	}
	return evalCholesky(mat)
}

func handleLDLT(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "ldlt", args[0].String()))
	}
	return evalLDLT(mat)
}

func handlePinv(args []Node, env *Env) (Node, error) {
	evaled, err := Eval(args[0])
	if err != nil {
		return nil, err
	}
	mat, ok := evaled.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.matrix_arg_required", "pinv", args[0].String()))
	}
	return evalPinv(mat)
}

