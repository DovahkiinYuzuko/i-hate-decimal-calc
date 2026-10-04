package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Piecewise Function Evaluation & CAD Normalization Engine
// -------------------------------------------------------------------------

// isConditionAlwaysTrue checks whether condition statically evaluates to true.
func isConditionAlwaysTrue(cond Node, env *Env) bool {
	if cond == nil {
		return true
	}
	if v, ok := cond.(*VarNode); ok && (v.Name == "true" || v.Name == "True") {
		return true
	}
	if rel, ok := cond.(*RelOpNode); ok {
		vars := ExtractFreeVariables(rel)
		if len(vars) == 0 {
			if res, decided := EvaluateRelOpWithInterval(rel); decided {
				return res
			}
			negR, err := simplifyUnaryOp("-", rel.RHS)
			if err == nil {
				diff, err := simplifyAdd([]Node{rel.LHS, negR})
				if err == nil {
					c, err := EvalWithEnv(diff, env)
					if err == nil {
						if r, ok := c.(*RationalNode); ok && r.Val != nil {
							return evalRelationalSign(r.Val.Sign(), rel.Op)
						}
					}
				}
			}
		}
	}
	return false
}

// isConditionAlwaysFalse checks whether condition statically evaluates to false.
func isConditionAlwaysFalse(cond Node, env *Env) bool {
	if cond == nil {
		return false
	}
	if v, ok := cond.(*VarNode); ok && (v.Name == "false" || v.Name == "False") {
		return true
	}
	if rel, ok := cond.(*RelOpNode); ok {
		vars := ExtractFreeVariables(rel)
		if len(vars) == 0 {
			if res, decided := EvaluateRelOpWithInterval(rel); decided {
				return !res
			}
			negR, err := simplifyUnaryOp("-", rel.RHS)
			if err == nil {
				diff, err := simplifyAdd([]Node{rel.LHS, negR})
				if err == nil {
					c, err := EvalWithEnv(diff, env)
					if err == nil {
						if r, ok := c.(*RationalNode); ok && r.Val != nil {
							return !evalRelationalSign(r.Val.Sign(), rel.Op)
						}
					}
				}
			}
		}
	}
	return false
}

// evalConditionAt substitutes varName with val and checks if the relational condition is met.
func evalConditionAt(cond Node, varName string, val Node, env *Env) (bool, bool) {
	if cond == nil {
		return true, true
	}
	if v, ok := cond.(*VarNode); ok {
		if v.Name == "true" || v.Name == "True" {
			return true, true
		}
		if v.Name == "false" || v.Name == "False" {
			return false, true
		}
	}
	subCond := Substitute(cond, varName, val)
	rel, ok := subCond.(*RelOpNode)
	if !ok {
		evaled, err := EvalWithEnv(subCond, env)
		if err == nil {
			if b, ok := evaled.(*VarNode); ok {
				if b.Name == "true" || b.Name == "True" {
					return true, true
				}
				if b.Name == "false" || b.Name == "False" {
					return false, true
				}
			}
		}
		return false, false
	}

	if res, decided := EvaluateRelOpWithInterval(rel); decided {
		return res, true
	}

	negR, err := simplifyUnaryOp("-", rel.RHS)
	if err != nil {
		return false, false
	}
	diff, err := simplifyAdd([]Node{rel.LHS, negR})
	if err != nil {
		return false, false
	}
	c, err := EvalWithEnv(diff, env)
	if err != nil {
		return false, false
	}
	r, ok := c.(*RationalNode)
	if !ok || r.Val == nil {
		return false, false
	}
	return evalRelationalSign(r.Val.Sign(), rel.Op), true
}

// NormalizePiecewise cleans up redundant, impossible, or identical cases in a PiecewiseNode.
func NormalizePiecewise(pw *PiecewiseNode, env *Env) (Node, error) {
	if pw == nil {
		return nil, nil
	}

	fsm := NewPiecewiseFSM()
	if err := fsm.TransitionTo(PiecewiseStateCadNormalizing); err != nil {
		return nil, err
	}

	var activeCases []PiecewiseCase
	var fallback Node = pw.Otherwise

	for _, c := range pw.Cases {
		if isConditionAlwaysFalse(c.Condition, env) {
			// Skip impossible domain
			continue
		}
		if isConditionAlwaysTrue(c.Condition, env) {
			// Unconditional branch: truncates all remaining cases and fallback
			activeCases = append(activeCases, PiecewiseCase{
				Expr:      c.Expr,
				Condition: nil,
			})
			fallback = nil
			break
		}
		activeCases = append(activeCases, c)
	}

	if err := fsm.TransitionTo(PiecewiseStatePruningEmpty); err != nil {
		return nil, err
	}

	// If no cases left, return fallback
	if len(activeCases) == 0 {
		_ = fsm.TransitionTo(PiecewiseStateCompleted)
		if fallback == nil {
			return mustRational(0, 1), nil
		}
		return fallback, nil
	}

	// If only 1 case with no condition (or true condition), collapse to that expression
	if len(activeCases) == 1 && activeCases[0].Condition == nil {
		_ = fsm.TransitionTo(PiecewiseStateCompleted)
		return activeCases[0].Expr, nil
	}

	if err := fsm.TransitionTo(PiecewiseStateSimplifying); err != nil {
		return nil, err
	}

	// Simplify each case expression
	allIdentical := true
	firstExpr := activeCases[0].Expr
	for i := range activeCases {
		simp, err := EvalWithEnv(activeCases[i].Expr, env)
		if err == nil {
			activeCases[i].Expr = simp
		}
		if i == 0 {
			firstExpr = activeCases[i].Expr
		} else if !activeCases[i].Expr.Equal(firstExpr) {
			allIdentical = false
		}
	}
	if fallback != nil {
		simpFb, err := EvalWithEnv(fallback, env)
		if err == nil {
			fallback = simpFb
		}
		if !fallback.Equal(firstExpr) {
			allIdentical = false
		}
	}

	if err := fsm.TransitionTo(PiecewiseStateMerging); err != nil {
		return nil, err
	}

	// If all branches produce identical expression and the entire domain is covered (or unconditional), piecewise collapses to that expression
	if allIdentical && fallback != nil && len(activeCases) > 0 {
		_ = fsm.TransitionTo(PiecewiseStateCompleted)
		return firstExpr, nil
	}
	if len(activeCases) == 1 && activeCases[0].Condition == nil {
		_ = fsm.TransitionTo(PiecewiseStateCompleted)
		return activeCases[0].Expr, nil
	}

	_ = fsm.TransitionTo(PiecewiseStateCompleted)
	return NewPiecewiseNode(activeCases, fallback), nil
}

// PiecewiseFold lifts nested PiecewiseNode instances out of arithmetic expressions.
// e.g., c * piecewise([[f1, C1], [f2, C2]]) -> piecewise([[c*f1, C1], [c*f2, C2]])
func PiecewiseFold(node Node, env *Env) (Node, error) {
	if node == nil {
		return nil, nil
	}

	switch v := node.(type) {
	case *UnaryOpNode:
		if v.Op == "-" {
			foldedInner, err := PiecewiseFold(v.Expr, env)
			if err != nil {
				return nil, err
			}
			if pw, ok := foldedInner.(*PiecewiseNode); ok {
				newCases := make([]PiecewiseCase, len(pw.Cases))
				for i, c := range pw.Cases {
					negExpr, err := simplifyUnaryOp("-", c.Expr)
					if err != nil {
						negExpr = &UnaryOpNode{Op: "-", Expr: c.Expr}
					}
					newCases[i] = PiecewiseCase{
						Expr:      negExpr,
						Condition: c.Condition,
					}
				}
				var newOtherwise Node
				if pw.Otherwise != nil {
					negOtherwise, err := simplifyUnaryOp("-", pw.Otherwise)
					if err != nil {
						negOtherwise = &UnaryOpNode{Op: "-", Expr: pw.Otherwise}
					}
					newOtherwise = negOtherwise
				}
				return NormalizePiecewise(NewPiecewiseNode(newCases, newOtherwise), env)
			}
			return &UnaryOpNode{Op: "-", Expr: foldedInner}, nil
		}

	case *MulNode:
		// Check if any factor is a PiecewiseNode
		pwIdx := -1
		for i, f := range v.Factors {
			if _, ok := f.(*PiecewiseNode); ok {
				pwIdx = i
				break
			}
		}
		if pwIdx != -1 {
			pw := v.Factors[pwIdx].(*PiecewiseNode)
			var otherFactors []Node
			for i, f := range v.Factors {
				if i != pwIdx {
					otherFactors = append(otherFactors, f)
				}
			}
			newCases := make([]PiecewiseCase, len(pw.Cases))
			for i, c := range pw.Cases {
				factors := append([]Node{c.Expr}, otherFactors...)
				prod, err := simplifyMul(factors)
				if err != nil {
					prod = NewMul(factors)
				}
				newCases[i] = PiecewiseCase{
					Expr:      prod,
					Condition: c.Condition,
				}
			}
			var newOtherwise Node
			if pw.Otherwise != nil {
				factors := append([]Node{pw.Otherwise}, otherFactors...)
				prod, err := simplifyMul(factors)
				if err != nil {
					prod = NewMul(factors)
				}
				newOtherwise = prod
			}
			return NormalizePiecewise(NewPiecewiseNode(newCases, newOtherwise), env)
		}

	case *AddNode:
		// Check if any term is a PiecewiseNode
		pwIdx := -1
		for i, t := range v.Terms {
			if _, ok := t.(*PiecewiseNode); ok {
				pwIdx = i
				break
			}
		}
		if pwIdx != -1 {
			pw := v.Terms[pwIdx].(*PiecewiseNode)
			var otherTerms []Node
			for i, t := range v.Terms {
				if i != pwIdx {
					otherTerms = append(otherTerms, t)
				}
			}
			newCases := make([]PiecewiseCase, len(pw.Cases))
			for i, c := range pw.Cases {
				terms := append([]Node{c.Expr}, otherTerms...)
				sum, err := simplifyAdd(terms)
				if err != nil {
					sum = NewAdd(terms)
				}
				newCases[i] = PiecewiseCase{
					Expr:      sum,
					Condition: c.Condition,
				}
			}
			var newOtherwise Node
			if pw.Otherwise != nil {
				terms := append([]Node{pw.Otherwise}, otherTerms...)
				sum, err := simplifyAdd(terms)
				if err != nil {
					sum = NewAdd(terms)
				}
				newOtherwise = sum
			}
			return NormalizePiecewise(NewPiecewiseNode(newCases, newOtherwise), env)
		}

	case *PiecewiseNode:
		return NormalizePiecewise(v, env)
	}

	return node, nil
}

// EvalPiecewiseHandler parses and constructs a PiecewiseNode from CLI arguments.
// piecewise([[expr1, cond1], [expr2, cond2], ...], otherwise)
func EvalPiecewiseHandler(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s", i18n.T("piecewise.err_invalid_piecewise_args"))
	}

	evalCasesArg, err := EvalWithEnv(args[0], env)
	if err != nil {
		return nil, err
	}

	var cases []PiecewiseCase
	switch m := evalCasesArg.(type) {
	case *MatrixNode:
		if m.Cols != 2 {
			return nil, fmt.Errorf("%s", i18n.T("piecewise.err_invalid_cases_format"))
		}
		for r := 0; r < m.Rows; r++ {
			cases = append(cases, PiecewiseCase{
				Expr:      m.Data[r][0],
				Condition: m.Data[r][1],
			})
		}
	case *ListNode:
		for _, elem := range m.Elements {
			pair, ok := elem.(*ListNode)
			if !ok || len(pair.Elements) != 2 {
				return nil, fmt.Errorf("%s", i18n.T("piecewise.err_case_pair_format", elem.String()))
			}
			cases = append(cases, PiecewiseCase{
				Expr:      pair.Elements[0],
				Condition: pair.Elements[1],
			})
		}
	default:
		return nil, fmt.Errorf("%s", i18n.T("piecewise.err_invalid_cases_format"))
	}

	var otherwise Node
	if len(args) == 2 {
		evalOtherwise, err := EvalWithEnv(args[1], env)
		if err != nil {
			return nil, err
		}
		otherwise = evalOtherwise
	}

	pwNode := NewPiecewiseNode(cases, otherwise)
	return NormalizePiecewise(pwNode, env)
}
