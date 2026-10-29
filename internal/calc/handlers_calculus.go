package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

func init() {
	RegisterHandler("diff", handleDiff)
	RegisterHandler("integrate", handleIntegrate)
	RegisterHandler("limit", handleLimit)
	RegisterHandler("taylor", handleTaylor)
	RegisterHandler("sum", handleSum)
	RegisterHandler("expand", handleExpand)
	RegisterHandler("factor", handleFactor)
	RegisterHandler("solve", handleSolve)
	RegisterHandler("apart", handleApart)
	RegisterHandler("together", handleTogether)
	RegisterHandler("trig_expand", handleTrigExpand)
	RegisterHandler("trig_reduce", handleTrigReduce)
	RegisterHandler("residue", handleResidue)
	RegisterLazyHandler("dsolve", handleDSolve)
	RegisterHandler("gosper_sum", handleGosperSum)
	RegisterHandler("wz_cert", handleWZCert)
	RegisterHandler("risch_integrate", handleRischIntegrate)
	RegisterHandler("verify", handleVerify)
}

func handleDiff(args []Node, env *Env) (Node, error) {
	varName := "x"
	if v, ok := args[1].(*VarNode); ok {
		varName = v.Name
	} else {
		return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "diff", "second", args[1].String()))
	}
	tracer := env.GetTracer()
	if tracer != nil {
		tracer.StartScope("calculus.diff", provenance.RelationEq, nil, args[0], provenance.VisPresentable)
	}
	var res Node
	var err error
	if len(args) == 3 {
		r, ok := args[2].(*RationalNode)
		if !ok || !r.Val.IsInt() || r.Val.Sign() < 0 {
			if tracer != nil {
				tracer.EndScope(args[0], nil)
			}
			return nil, fmt.Errorf("%s", i18n.T("errors.diff_order_non_negative", args[2].String()))
		}
		order := r.Val.Num().Int64()
		res = args[0]
		for k := int64(0); k < order; k++ {
			res, err = differentiate(res, varName)
			if err != nil {
				if tracer != nil {
					tracer.EndScope(res, nil)
				}
				return nil, err
			}
		}
	} else {
		res, err = differentiate(args[0], varName)
	}
	if tracer != nil {
		tracer.EndScope(res, nil)
	}
	return res, err
}

func handleIntegrate(args []Node, env *Env) (Node, error) {
	varName := "x"
	if v, ok := args[1].(*VarNode); ok {
		varName = v.Name
	} else {
		return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "integrate", "second", args[1].String()))
	}
	tracer := env.GetTracer()
	if len(args) == 2 {
		if tracer != nil {
			tracer.StartScope("calculus.integrate", provenance.RelationEquivalentModuloC, nil, args[0], provenance.VisPresentable)
		}
		res, err := evalIndefiniteIntegral(args[0], varName)
		if tracer != nil {
			var cert provenance.AlgebraicCertificate
			if err == nil && res != nil {
				cert = &provenance.AntiderivativeCertificate{
					Integrand:      args[0],
					Antiderivative: res,
					Variable:       varName,
				}
			}
			tracer.EndScope(res, cert)
		}
		return res, err
	}
	if tracer != nil {
		tracer.StartScope("calculus.integrate_definite", provenance.RelationEq, nil, args[0], provenance.VisPresentable)
	}
	res, err := evalDefiniteIntegral(args[0], varName, args[2], args[3])
	if tracer != nil {
		tracer.EndScope(res, nil)
	}
	return res, err
}

func handleLimit(args []Node, env *Env) (Node, error) {
	var dirNode Node
	if len(args) == 4 {
		dirNode = args[3]
	}
	tracer := env.GetTracer()
	if tracer != nil {
		tracer.StartScope(provenance.RuleCalcLHopital, provenance.RelationEq, nil, args[0], provenance.VisPresentable)
	}
	res, err := EvalLimit(args[0], args[1], args[2], dirNode)
	if tracer != nil {
		tracer.EndScope(res, nil)
	}
	return res, err
}

func handleTaylor(args []Node, env *Env) (Node, error) {
	return evalTaylor(args[0], args[1], args[2], args[3])
}

func handleSum(args []Node, env *Env) (Node, error) {
	return evalSum(args[0], args[1], args[2], args[3])
}

func handleExpand(args []Node, env *Env) (Node, error) {
	tracer := env.GetTracer()
	if tracer != nil {
		tracer.StartScope(provenance.RuleAlgebraExpand, provenance.RelationEq, nil, args[0], provenance.VisPresentable)
	}
	res := expandNode(args[0])
	if tracer != nil {
		tracer.EndScope(res, nil)
	}
	return res, nil
}

func handleFactor(args []Node, env *Env) (Node, error) {
	tracer := env.GetTracer()
	if tracer != nil {
		tracer.StartScope(provenance.RuleAlgebraFactor, provenance.RelationEq, nil, args[0], provenance.VisPresentable)
	}
	var res Node
	var err error
	if len(args) == 1 {
		res, err = Factor(args[0])
	} else {
		varName := "x"
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			if tracer != nil {
				tracer.EndScope(args[0], nil)
			}
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "factor", "second", args[1].String()))
		}
		res, err = Factor(args[0], varName)
	}
	if tracer != nil {
		tracer.EndScope(res, nil)
	}
	return res, err
}

func handleSolve(args []Node, env *Env) (Node, error) {
	if v, ok := args[0].(*VarNode); ok && (v.Name == "true" || v.Name == "false") {
		return v, nil
	}

	tracer := env.GetTracer()
	if tracer != nil {
		tracer.StartScope("solver.solve", provenance.RelationIff, nil, args[0], provenance.VisPresentable)
	}

	varName := "x"
	if len(args) >= 2 {
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			if tracer != nil {
				tracer.EndScope(args[0], nil)
			}
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "solve", "second", args[1].String()))
		}
	} else {
		// Auto-extract free variable
		vars := ExtractFreeVariables(args[0])
		if len(vars) > 0 {
			varName = vars[0]
		}
	}

	var res Node
	var err error

	// Check if relational inequality or equality
	if rel, ok := args[0].(*RelOpNode); ok {
		switch rel.Op {
		case "<", "<=", ">", ">=":
			res, err = SolveInequality(rel, varName, env)
		case "==":
			negRHS, negErr := simplifyUnaryOp("-", rel.RHS)
			if negErr != nil {
				negRHS = &UnaryOpNode{Op: "-", Expr: rel.RHS}
			}
			diff, addErr := simplifyAdd([]Node{rel.LHS, negRHS})
			if addErr != nil {
				diff = NewAdd([]Node{rel.LHS, negRHS})
			}
			res, err = solveEquation(diff, varName)
		default:
			res, err = solveEquation(args[0], varName)
		}
	} else {
		res, err = solveEquation(args[0], varName)
	}

	if tracer != nil {
		tracer.EndScope(res, nil)
	}
	return res, err
}

func handleApart(args []Node, env *Env) (Node, error) {
	tracer := env.GetTracer()
	if tracer != nil {
		tracer.StartScope(provenance.RuleAlgebraApart, provenance.RelationEq, nil, args[0], provenance.VisPresentable)
	}
	var res Node
	var err error
	if len(args) == 1 {
		res, err = EvalApart(args[0])
	} else {
		varName := ""
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			if tracer != nil {
				tracer.EndScope(args[0], nil)
			}
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "apart", "second", args[1].String()))
		}
		res, err = EvalApart(args[0], varName)
	}
	if tracer != nil {
		tracer.EndScope(res, nil)
	}
	return res, err
}

func handleTogether(args []Node, env *Env) (Node, error) {
	tracer := env.GetTracer()
	if tracer != nil {
		tracer.StartScope(provenance.RuleAlgebraTogether, provenance.RelationEq, nil, args[0], provenance.VisPresentable)
	}
	res, err := EvalTogether(args[0])
	if tracer != nil {
		tracer.EndScope(res, nil)
	}
	return res, err
}

func handleTrigExpand(args []Node, env *Env) (Node, error) {
	return EvalTrigExpand(args[0], env)
}

func handleTrigReduce(args []Node, env *Env) (Node, error) {
	return EvalTrigReduce(args[0], env)
}

func handleDSolve(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("%s", i18n.T("errors.dsolve_args_count", len(args)))
	}
	var yName, xName string
	if len(args) >= 2 {
		if vy, ok := args[1].(*VarNode); ok {
			yName = vy.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "dsolve", "second", args[1].String()))
		}
	}
	if len(args) >= 3 {
		if vx, ok := args[2].(*VarNode); ok {
			xName = vx.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "dsolve", "third", args[2].String()))
		}
	}
	return EvalDSolve(args[0], yName, xName, env)
}

func handleResidue(args []Node, env *Env) (Node, error) {
	return EvalResidue(args[0], args[1], args[2], env)
}

func handleGosperSum(args []Node, env *Env) (Node, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("%s", i18n.T("handlers.err_gosper_sum_requires_at_least"))
	}
	kVar := "k"
	if v, ok := args[1].(*VarNode); ok {
		kVar = v.Name
	}
	if len(args) == 2 {
		return GosperIndefiniteSum(args[0], kVar)
	}
	if len(args) == 4 {
		return GosperDefiniteSum(args[0], kVar, args[2], args[3])
	}
	return nil, fmt.Errorf("%s", i18n.T("handlers.err_gosper_sum_requires_2_arguments"))
}

func handleWZCert(args []Node, env *Env) (Node, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("%s", i18n.T("verify.err_wz_cert_requires_3_arguments"))
	}
	nVar := "n"
	if vn, ok := args[1].(*VarNode); ok {
		nVar = vn.Name
	}
	kVar := "k"
	if vk, ok := args[2].(*VarNode); ok {
		kVar = vk.Name
	}
	return GenerateWZCertificate(args[0], nVar, kVar)
}

func handleRischIntegrate(args []Node, env *Env) (Node, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("%s", i18n.T("handlers.err_risch_integrate_requires_at_least"))
	}
	varName := "x"
	if len(args) >= 2 {
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "risch_integrate", "second", args[1].String()))
		}
	}
	return RischIntegrate(args[0], varName)
}

func handleVerify(args []Node, env *Env) (Node, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("%s", i18n.T("handlers.err_verify_requires_at_least_1"))
	}
	if len(args) == 2 {
		cert, err := VerifyAlgebraicEquivalence(args[0], args[1], env)
		if err != nil {
			return nil, err
		}
		return &VarNode{Name: cert.String()}, nil
	}

	if rel, ok := args[0].(*RelOpNode); ok && rel.Op == "==" {
		cert, err := VerifyAlgebraicEquivalence(rel.LHS, rel.RHS, env)
		if err != nil {
			return nil, err
		}
		return &VarNode{Name: cert.String()}, nil
	}

	// 1 argument: evaluate expression first, then verify
	evalRes, err := Eval(args[0])
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("handlers.err_verify_failed_to_evaluate_expression", err))
	}
	cert, err := VerifyComputation(args[0], evalRes, env)
	if err != nil {
		return nil, err
	}
	return &VarNode{Name: cert.String()}, nil
}
