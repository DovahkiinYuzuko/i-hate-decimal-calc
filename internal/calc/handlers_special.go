package calc

func init() {
	RegisterHandler("gamma", handleGamma)
	RegisterHandler("beta", handleBeta)
	RegisterHandler("bernoulli", handleBernoulli)
	RegisterHandler("zeta", handleZeta)
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
