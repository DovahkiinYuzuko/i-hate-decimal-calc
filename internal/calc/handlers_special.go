package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterHandler("gamma", handleGamma)
	RegisterHandler("beta", handleBeta)
	RegisterHandler("bernoulli", handleBernoulli)
	RegisterHandler("zeta", handleZeta)
	RegisterHandler("lambert_w", handleLambertW)
	RegisterHandler("erf", handleErf)
	RegisterHandler("interval", handleInterval)
}

func handleLambertW(args []Node, env *Env) (Node, error) {
	return EvalLambertW(args, env)
}

func handleErf(args []Node, env *Env) (Node, error) {
	return EvalErf(args, env)
}

func handleGamma(args []Node, env *Env) (Node, error) {
	return EvalGamma(args[0], env)
}

func handleBeta(args []Node, env *Env) (Node, error) {
	return EvalBeta(args[0], args[1], env)
}

func handleBernoulli(args []Node, env *Env) (Node, error) {
	evalArg, err := EvalWithEnv(args[0], env)
	if err != nil {
		evalArg = args[0]
	}
	return EvalBernoulli(evalArg)
}

func handleZeta(args []Node, env *Env) (Node, error) {
	return EvalZeta(args[0], env)
}

func handleInterval(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s", i18n.T("registry.func_args_between", "interval", 1, 2, len(args)))
	}

	evalExpr, err := EvalWithEnv(args[0], env)
	if err != nil {
		evalExpr = args[0]
	}

	eps := big.NewRat(1, 1000000)
	if len(args) == 2 {
		evalEps, err := EvalWithEnv(args[1], env)
		if err != nil {
			evalEps = args[1]
		}
		if rat, ok := evalEps.(*RationalNode); ok {
			eps = rat.Val
		} else {
			return nil, fmt.Errorf("%s", i18n.T("interval.err_invalid_epsilon", evalEps.String()))
		}
	}

	interval, err := AdaptiveRefineInterval(evalExpr, eps, 100)
	if err != nil {
		return nil, err
	}
	return NewIntervalNode(interval.Low, interval.High), nil
}

