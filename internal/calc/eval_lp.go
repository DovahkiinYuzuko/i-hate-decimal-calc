package calc

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/units"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/lp"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

func init() {
	RegisterLazyHandler("lp_maximize", EvalLPMaximize)
	RegisterLazyHandler("lp_minimize", EvalLPMinimize)

	RegisterFunction(FunctionSpec{
		Name:     "lp_maximize",
		MinArgs:  2,
		MaxArgs:  4,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "lp_minimize",
		MinArgs:  2,
		MaxArgs:  4,
		LazyArgs: true,
	})
}

// EvalLPMaximize solves a linear programming maximization problem.
func EvalLPMaximize(args []Node, env *Env) (Node, error) {
	return evalLPWithSense(lp.OptMaximize, args, env)
}

// EvalLPMinimize solves a linear programming minimization problem.
func EvalLPMinimize(args []Node, env *Env) (Node, error) {
	return evalLPWithSense(lp.OptMinimize, args, env)
}

func evalLPWithSense(sense lp.OptSense, args []Node, env *Env) (Node, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("%s", i18n.T("lp.err_requires_at_least_2_args"))
	}

	tracer := env.GetTracer()
	if tracer != nil {
		opName := "lp_maximize"
		if sense == lp.OptMinimize {
			opName = "lp_minimize"
		}
		tracer.StartScope(opName, provenance.RelationEq, nil, args[0], provenance.VisPresentable)
	}

	prob, varUnits, err := parseLPProblem(sense, args, env)
	if err != nil {
		if tracer != nil {
			tracer.EndScope(nil, nil)
		}
		return nil, err
	}

	res, err := lp.Solve(prob)
	if err != nil {
		if tracer != nil {
			tracer.EndScope(nil, nil)
		}
		return nil, err
	}

	// Verify certificate internally
	if err := lp.SanitizeAndVerifyCertificate(prob, res.Certificate); err != nil {
		if tracer != nil {
			tracer.EndScope(nil, nil)
		}
		return nil, fmt.Errorf("%s", i18n.T("lp.err_internal_certificate_failed", err.Error()))
	}

	// Record pivot history into provenance tracer if active
	if tracer != nil {
		for _, step := range res.PivotHistory {
			stepDesc := fmt.Sprintf("%s", i18n.T("lp.step_pivot",
				step.Phase, step.Iteration, step.EnteringVar, step.LeavingVar))
			tracer.RecordStep("lp.pivot", provenance.RelationEq, nil,
				ast.NewVar(stepDesc), ast.NewRationalFromBigRat(step.ObjVal),
				nil, nil, provenance.TrustCertified, provenance.VisPresentable)
		}
	}

	outNode := formatLPResult(prob, res, varUnits)

	if tracer != nil {
		tracer.EndScope(outNode, nil)
	}

	return outNode, nil
}

// parseLPProblem parses AST arguments into an lp.Problem.
func parseLPProblem(sense lp.OptSense, args []Node, env *Env) (*lp.Problem, map[string]ast.DimensionVector, error) {
	if len(args) == 4 {
		return parseLPMatrixMode(sense, args, env)
	}
	return parseLPExpressionMode(sense, args[0], args[1], env)
}

func parseLPExpressionMode(sense lp.OptSense, objNode, consNode Node, env *Env) (*lp.Problem, map[string]ast.DimensionVector, error) {
	objEval, err := evalEnvConstants(objNode, env)
	if err != nil {
		return nil, nil, err
	}
	consEval, err := evalEnvConstants(consNode, env)
	if err != nil {
		return nil, nil, err
	}

	consList, ok := consEval.(*ast.ListNode)
	if !ok {
		return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_expected_constraints_list", consEval))
	}

	varSet := make(map[string]bool)
	collectLPVariables(objEval, varSet)
	for _, c := range consList.Elements {
		collectLPVariables(c, varSet)
	}

	varNames := make([]string, 0, len(varSet))
	for v := range varSet {
		// Filter out physical units
		if _, _, err := units.ResolveUnitSymbol(v); err != nil {
			varNames = append(varNames, v)
		}
	}
	sort.Strings(varNames)

	if len(varNames) == 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_no_decision_variables"))
	}

	varIdx := make(map[string]int, len(varNames))
	for i, v := range varNames {
		varIdx[v] = i
	}

	objCoeffs, objConst, varUnits, err := extractLinearCoeffs(objEval, varNames, varIdx)
	if err != nil {
		return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_invalid_objective", err.Error()))
	}

	constraints := make([]lp.Constraint, 0, len(consList.Elements))
	for i, cNode := range consList.Elements {
		relOp, ok := cNode.(*ast.RelOpNode)
		if !ok {
			return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_constraint_not_relational", i+1))
		}

		var relSense lp.RelSense
		switch relOp.Op {
		case "<=", "<":
			relSense = lp.RelLe
		case ">=", ">":
			relSense = lp.RelGe
		case "==", "=":
			relSense = lp.RelEq
		default:
			return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_unsupported_rel_op", relOp.Op, i+1))
		}

		negRHS, _ := ast.NewUnaryOp("-", relOp.RHS)
		diffNode := ast.NewAdd([]Node{relOp.LHS, negRHS})
		diffSimp, err := Eval(diffNode)
		if err != nil {
			diffSimp = diffNode
		}

		coeffs, constTerm, _, err := extractLinearCoeffs(diffSimp, varNames, varIdx)
		if err != nil {
			return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_constraint_not_linear", i+1, err.Error()))
		}

		rhs := new(big.Rat).Neg(constTerm)

		constraints = append(constraints, lp.Constraint{
			Name:   fmt.Sprintf("c%d", i+1),
			Coeffs: coeffs,
			Sense:  relSense,
			RHS:    rhs,
		})
	}

	return &lp.Problem{
		Sense:       sense,
		Objective:   objCoeffs,
		ObjConstant: objConst,
		VarNames:    varNames,
		Constraints: constraints,
	}, varUnits, nil
}

func parseLPMatrixMode(sense lp.OptSense, args []Node, env *Env) (*lp.Problem, map[string]ast.DimensionVector, error) {
	cList, ok := args[0].(*ast.ListNode)
	if !ok {
		return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_expected_objective_list", args[0]))
	}

	var AData [][]Node
	if mat, ok := args[1].(*ast.MatrixNode); ok {
		AData = mat.Data
	} else if aList, ok := args[1].(*ast.ListNode); ok {
		AData = make([][]Node, len(aList.Elements))
		for i, rowElem := range aList.Elements {
			rowList, ok := rowElem.(*ast.ListNode)
			if !ok {
				return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_matrix_row_must_be_list", i+1, rowElem))
			}
			AData[i] = rowList.Elements
		}
	} else {
		return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_expected_matrix_or_lists", args[1]))
	}

	relList, ok := args[2].(*ast.ListNode)
	if !ok {
		return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_expected_rel_list", args[2]))
	}
	bList, ok := args[3].(*ast.ListNode)
	if !ok {
		return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_expected_rhs_list", args[3]))
	}

	n := len(cList.Elements)
	m := len(AData)
	if len(relList.Elements) != m || len(bList.Elements) != m {
		return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_matrix_dim_mismatch", m, len(relList.Elements), len(bList.Elements)))
	}

	varNames := make([]string, n)
	for j := 0; j < n; j++ {
		varNames[j] = fmt.Sprintf("x%d", j+1)
	}

	objCoeffs := make([]*big.Rat, n)
	for j, elem := range cList.Elements {
		ratVal, err := nodeToRational(elem)
		if err != nil {
			return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_obj_coeff_not_rational", j+1, err.Error()))
		}
		objCoeffs[j] = ratVal
	}

	constraints := make([]lp.Constraint, m)
	for i, rowList := range AData {
		if len(rowList) != n {
			return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_row_element_count_mismatch", i+1, n, len(rowList)))
		}
		rowCoeffs := make([]*big.Rat, n)
		for j, aVal := range rowList {
			ratVal, err := nodeToRational(aVal)
			if err != nil {
				return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_a_elem_not_rational", i+1, j+1, err.Error()))
			}
			rowCoeffs[j] = ratVal
		}

		relStr := relList.Elements[i].String()
		relStr = strings.Trim(relStr, "\"")
		var relSense lp.RelSense
		switch relStr {
		case "<=", "<":
			relSense = lp.RelLe
		case ">=", ">":
			relSense = lp.RelGe
		case "==", "=":
			relSense = lp.RelEq
		default:
			return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_unsupported_relation_string", relStr, i+1))
		}

		bVal, err := nodeToRational(bList.Elements[i])
		if err != nil {
			return nil, nil, fmt.Errorf("%s", i18n.T("lp.err_rhs_not_rational", i+1, err.Error()))
		}

		constraints[i] = lp.Constraint{
			Name:   fmt.Sprintf("c%d", i+1),
			Coeffs: rowCoeffs,
			Sense:  relSense,
			RHS:    bVal,
		}
	}

	return &lp.Problem{
		Sense:       sense,
		Objective:   objCoeffs,
		ObjConstant: new(big.Rat),
		VarNames:    varNames,
		Constraints: constraints,
	}, nil, nil
}

func extractLinearCoeffs(expr Node, varNames []string, varIdx map[string]int) ([]*big.Rat, *big.Rat, map[string]ast.DimensionVector, error) {
	n := len(varNames)
	coeffs := make([]*big.Rat, n)
	for j := 0; j < n; j++ {
		coeffs[j] = new(big.Rat)
	}
	constTerm := new(big.Rat)
	varUnits := make(map[string]ast.DimensionVector)

	var objTotalDim *ast.DimensionVector

	terms := flattenAddTerms(expr)
	for _, term := range terms {
		c, v, u, err := extractSingleTerm(term)
		if err != nil {
			return nil, nil, nil, err
		}
		if v == "" {
			constTerm.Add(constTerm, c)
		} else {
			idx, ok := varIdx[v]
			if !ok {
				return nil, nil, nil, fmt.Errorf("%s", i18n.T("lp.err_unknown_variable", v, term.String()))
			}
			coeffs[idx].Add(coeffs[idx], c)
			if !u.IsZero() {
				if objTotalDim == nil {
					objTotalDim = &u
				} else if !objTotalDim.Equal(u) {
					return nil, nil, nil, fmt.Errorf("%s", i18n.T("lp.err_incompatible_physical_dim", objTotalDim.String(), u.String()))
				}

				if existingU, ok := varUnits[v]; ok {
					if !existingU.Equal(u) {
						return nil, nil, nil, fmt.Errorf("%s", i18n.T("lp.err_var_dim_mismatch", v, existingU.String(), u.String()))
					}
				} else {
					varUnits[v] = u
				}
			}
		}
	}

	return coeffs, constTerm, varUnits, nil
}

func extractSingleTerm(term Node) (*big.Rat, string, ast.DimensionVector, error) {
	zeroDim := ast.NewDimensionVector()
	switch t := term.(type) {
	case *ast.RationalNode:
		return new(big.Rat).Set(t.Val), "", zeroDim, nil
	case *ast.VarNode:
		return big.NewRat(1, 1), t.Name, zeroDim, nil
	case *ast.UnaryOpNode:
		if t.Op == "-" {
			c, v, u, err := extractSingleTerm(t.Expr)
			if err != nil {
				return nil, "", zeroDim, err
			}
			c.Neg(c)
			return c, v, u, nil
		}
		return nil, "", zeroDim, fmt.Errorf("%s", i18n.T("lp.err_unsupported_unary_op", t.Op))
	case *ast.MulNode:
		coeff := big.NewRat(1, 1)
		var varName string
		dims := ast.NewDimensionVector()

		for _, rawFactor := range t.Factors {
			factor := rawFactor
			if ev, err := Eval(rawFactor); err == nil {
				factor = ev
			}
			switch f := factor.(type) {
			case *ast.RationalNode:
				coeff.Mul(coeff, f.Val)
			case *ast.VarNode:
				scale, d, err := units.ResolveUnitSymbol(f.Name)
				if err == nil {
					coeff.Mul(coeff, scale)
					for i := 0; i < 7; i++ {
						dims[i].Add(dims[i], d[i])
					}
				} else {
					if varName != "" {
						return nil, "", zeroDim, fmt.Errorf("%s", i18n.T("lp.err_nonlinear_term_multiple_vars", varName, f.Name))
					}
					varName = f.Name
				}
			case *ast.QuantityNode:
				coeff.Mul(coeff, f.Scale)
				for i := 0; i < 7; i++ {
					dims[i].Add(dims[i], f.Dimensions[i])
				}
			case *ast.UnaryOpNode:
				if f.Op == "-" {
					c, v, u, err := extractSingleTerm(f)
					if err != nil {
						return nil, "", zeroDim, err
					}
					coeff.Mul(coeff, c)
					if v != "" {
						if varName != "" {
							return nil, "", zeroDim, fmt.Errorf("%s", i18n.T("lp.err_nonlinear_term"))
						}
						varName = v
					}
					for i := 0; i < 7; i++ {
						dims[i].Add(dims[i], u[i])
					}
				}
			default:
				return nil, "", zeroDim, fmt.Errorf("%s", i18n.T("lp.err_unsupported_factor", factor, factor.String()))
			}
		}
		return coeff, varName, dims, nil
	case *ast.QuantityNode:
		return new(big.Rat).Set(t.Scale), "", t.Dimensions.Clone(), nil
	default:
		return nil, "", zeroDim, fmt.Errorf("%s", i18n.T("lp.err_unsupported_term_type", term, term.String()))
	}
}

func flattenAddTerms(expr Node) []Node {
	switch n := expr.(type) {
	case *ast.AddNode:
		var terms []Node
		for _, term := range n.Terms {
			terms = append(terms, flattenAddTerms(term)...)
		}
		return terms
	case *ast.UnaryOpNode:
		if n.Op == "-" {
			innerTerms := flattenAddTerms(n.Expr)
			var terms []Node
			for _, it := range innerTerms {
				neg, _ := ast.NewUnaryOp("-", it)
				terms = append(terms, neg)
			}
			return terms
		}
	}
	return []Node{expr}
}

func collectLPVariables(node Node, varSet map[string]bool) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ast.VarNode:
		varSet[n.Name] = true
	case *ast.AddNode:
		for _, t := range n.Terms {
			collectLPVariables(t, varSet)
		}
	case *ast.MulNode:
		for _, f := range n.Factors {
			collectLPVariables(f, varSet)
		}
	case *ast.UnaryOpNode:
		collectLPVariables(n.Expr, varSet)
	case *ast.RelOpNode:
		collectLPVariables(n.LHS, varSet)
		collectLPVariables(n.RHS, varSet)
	case *ast.ListNode:
		for _, elem := range n.Elements {
			collectLPVariables(elem, varSet)
		}
	}
}

func evalEnvConstants(node Node, env *Env) (Node, error) {
	if node == nil || env == nil {
		return node, nil
	}
	switch n := node.(type) {
	case *ast.VarNode:
		if val, ok := env.Get(n.Name); ok {
			return val, nil
		}
		return n, nil
	case *ast.AddNode:
		newTerms := make([]Node, len(n.Terms))
		for i, t := range n.Terms {
			ev, err := evalEnvConstants(t, env)
			if err != nil {
				return nil, err
			}
			newTerms[i] = ev
		}
		return ast.NewAdd(newTerms), nil
	case *ast.MulNode:
		newFactors := make([]Node, len(n.Factors))
		for i, f := range n.Factors {
			ev, err := evalEnvConstants(f, env)
			if err != nil {
				return nil, err
			}
			newFactors[i] = ev
		}
		return ast.NewMul(newFactors), nil
	case *ast.UnaryOpNode:
		ev, err := evalEnvConstants(n.Expr, env)
		if err != nil {
			return nil, err
		}
		return ast.NewUnaryOp(n.Op, ev)
	case *ast.RelOpNode:
		l, err := evalEnvConstants(n.LHS, env)
		if err != nil {
			return nil, err
		}
		r, err := evalEnvConstants(n.RHS, env)
		if err != nil {
			return nil, err
		}
		return ast.NewRelOp(l, n.Op, r), nil
	case *ast.ListNode:
		newElems := make([]Node, len(n.Elements))
		for i, elem := range n.Elements {
			ev, err := evalEnvConstants(elem, env)
			if err != nil {
				return nil, err
			}
			newElems[i] = ev
		}
		return ast.NewList(newElems), nil
	}
	return node, nil
}

func nodeToRational(node Node) (*big.Rat, error) {
	switch n := node.(type) {
	case *ast.RationalNode:
		return new(big.Rat).Set(n.Val), nil
	case *ast.UnaryOpNode:
		if n.Op == "-" {
			val, err := nodeToRational(n.Expr)
			if err != nil {
				return nil, err
			}
			return new(big.Rat).Neg(val), nil
		}
	}
	return nil, fmt.Errorf("%s", i18n.T("lp.err_cannot_convert_to_rational", node, node.String()))
}

func formatLPResult(prob *lp.Problem, res *lp.Result, varUnits map[string]ast.DimensionVector) Node {
	switch res.Status {
	case lp.StatusOptimal:
		var elems []Node
		for _, v := range prob.VarNames {
			val := res.PrimalSolution[v]
			var valNode Node = ast.NewRationalFromBigRat(val)
			if u, ok := varUnits[v]; ok && !u.IsZero() {
				valNode = ast.NewQuantityNode(valNode, val, u, u.String())
			}
			elems = append(elems, ast.NewRelOp(ast.NewVar(v), "==", valNode))
		}
		optValNode := ast.NewRationalFromBigRat(res.OptimalValue)
		elems = append(elems, ast.NewRelOp(ast.NewVar("optimal_value"), "==", optValNode))
		return ast.NewList(elems)

	case lp.StatusInfeasible:
		return ast.NewVar("infeasible")

	case lp.StatusUnbounded:
		return ast.NewVar("unbounded")

	default:
		return ast.NewVar("unknown")
	}
}
