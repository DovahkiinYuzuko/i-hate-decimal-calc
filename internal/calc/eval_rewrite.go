package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

// PatternBindings maps wildcard names (e.g. "_x") to concrete matched AST nodes.
type PatternBindings map[string]ast.Node

// Clone returns a shallow copy of PatternBindings.
func (pb PatternBindings) Clone() PatternBindings {
	res := make(PatternBindings, len(pb))
	for k, v := range pb {
		res[k] = v
	}
	return res
}

// MatchAST attempts to match pattern against target.
// It returns true and updated bindings if the pattern matches target.
func MatchAST(pattern, target ast.Node, bindings PatternBindings) (PatternBindings, bool) {
	if pattern == nil || target == nil {
		return bindings, pattern == target
	}

	// 1. Wildcard node matching
	if wc, ok := pattern.(*ast.WildcardNode); ok {
		if bound, exists := bindings[wc.Name]; exists {
			// Non-linear pattern: must match already bound term
			if !bound.Equal(target) {
				return bindings, false
			}
			return bindings, true
		}
		newBindings := bindings.Clone()
		newBindings[wc.Name] = target
		return newBindings, true
	}

	// 2. Syntactic matching by node type
	switch p := pattern.(type) {
	case *ast.RationalNode:
		if t, ok := target.(*ast.RationalNode); ok {
			return bindings, p.Equal(t)
		}
		return bindings, false

	case *ast.ConstNode:
		if t, ok := target.(*ast.ConstNode); ok {
			return bindings, p.Name == t.Name
		}
		return bindings, false

	case *ast.VarNode:
		if t, ok := target.(*ast.VarNode); ok {
			return bindings, p.Name == t.Name
		}
		return bindings, false

	case *ast.StringNode:
		if t, ok := target.(*ast.StringNode); ok {
			return bindings, p.Value == t.Value
		}
		return bindings, false

	case *ast.SqrtNode:
		if t, ok := target.(*ast.SqrtNode); ok {
			return MatchAST(p.Radicand, t.Radicand, bindings)
		}
		return bindings, false

	case *ast.UnaryOpNode:
		if t, ok := target.(*ast.UnaryOpNode); ok {
			if p.Op != t.Op {
				return bindings, false
			}
			return MatchAST(p.Expr, t.Expr, bindings)
		}
		return bindings, false

	case *ast.PowNode:
		if t, ok := target.(*ast.PowNode); ok {
			b1, ok1 := MatchAST(p.Base, t.Base, bindings)
			if !ok1 {
				return bindings, false
			}
			return MatchAST(p.Exp, t.Exp, b1)
		}
		return bindings, false

	case *ast.FuncNode:
		if t, ok := target.(*ast.FuncNode); ok {
			if p.Name != t.Name || len(p.Args) != len(t.Args) {
				return bindings, false
			}
			curr := bindings
			for i := range p.Args {
				next, ok := MatchAST(p.Args[i], t.Args[i], curr)
				if !ok {
					return bindings, false
				}
				curr = next
			}
			return curr, true
		}
		return bindings, false

	case *ast.AddNode:
		if t, ok := target.(*ast.AddNode); ok {
			return matchTermsAC(p.Terms, t.Terms, bindings)
		}
		return bindings, false

	case *ast.MulNode:
		if t, ok := target.(*ast.MulNode); ok {
			return matchTermsAC(p.Factors, t.Factors, bindings)
		}
		return bindings, false

	case *ast.ListNode:
		if t, ok := target.(*ast.ListNode); ok {
			if len(p.Elements) != len(t.Elements) {
				return bindings, false
			}
			curr := bindings
			for i := range p.Elements {
				next, ok := MatchAST(p.Elements[i], t.Elements[i], curr)
				if !ok {
					return bindings, false
				}
				curr = next
			}
			return curr, true
		}
		return bindings, false

	case *ast.RelOpNode:
		if t, ok := target.(*ast.RelOpNode); ok {
			if p.Op != t.Op {
				return bindings, false
			}
			b1, ok1 := MatchAST(p.LHS, t.LHS, bindings)
			if !ok1 {
				return bindings, false
			}
			return MatchAST(p.RHS, t.RHS, b1)
		}
		return bindings, false

	default:
		return bindings, pattern.Equal(target)
	}
}

// matchTermsAC performs associative-commutative matching on a multiset of pattern terms against target terms.
func matchTermsAC(patternTerms, targetTerms []ast.Node, bindings PatternBindings) (PatternBindings, bool) {
	if len(patternTerms) != len(targetTerms) {
		return bindings, false
	}
	used := make([]bool, len(targetTerms))

	var search func(pIdx int, currBindings PatternBindings) (PatternBindings, bool)
	search = func(pIdx int, currBindings PatternBindings) (PatternBindings, bool) {
		if pIdx >= len(patternTerms) {
			return currBindings, true
		}
		pTerm := patternTerms[pIdx]
		for tIdx, tTerm := range targetTerms {
			if used[tIdx] {
				continue
			}
			if nextBindings, ok := MatchAST(pTerm, tTerm, currBindings); ok {
				used[tIdx] = true
				if finalBindings, found := search(pIdx+1, nextBindings); found {
					return finalBindings, true
				}
				used[tIdx] = false
			}
		}
		return currBindings, false
	}

	return search(0, bindings)
}

// InstantiateTemplate replaces wildcard nodes in template with concrete bound terms.
func InstantiateTemplate(template ast.Node, bindings PatternBindings) ast.Node {
	return ast.Transform(template, func(n ast.Node) ast.Node {
		if wc, ok := n.(*ast.WildcardNode); ok {
			if val, exists := bindings[wc.Name]; exists {
				return val
			}
		}
		return n
	})
}

// TryApplyRuleAtRoot checks if rule.LHS matches target (including AC partial matching for AddNode/MulNode).
// If matched, it returns the instantiated replacement and true.
func TryApplyRuleAtRoot(rule *ast.RuleNode, target ast.Node) (ast.Node, bool) {
	if rule == nil || rule.LHS == nil || target == nil {
		return target, false
	}

	// 1. Full pattern match at root
	if bindings, ok := MatchAST(rule.LHS, target, PatternBindings{}); ok {
		return InstantiateTemplate(rule.RHS, bindings), true
	}

	// 2. AC Partial Matching: AddNode
	// If rule.LHS is AddNode(p_1, ..., p_k) and target is AddNode(t_1, ..., t_m) with k < m
	if pAdd, okP := rule.LHS.(*ast.AddNode); okP {
		if tAdd, okT := target.(*ast.AddNode); okT && len(pAdd.Terms) < len(tAdd.Terms) {
			k := len(pAdd.Terms)
			m := len(tAdd.Terms)
			indices := make([]int, k)
			var findSubMatch func(currK int, startM int, currBindings PatternBindings) (PatternBindings, []int, bool)
			findSubMatch = func(currK int, startM int, currBindings PatternBindings) (PatternBindings, []int, bool) {
				if currK == k {
					return currBindings, indices, true
				}
				pTerm := pAdd.Terms[currK]
				for j := 0; j < m; j++ {
					// Check if j is already picked
					alreadyPicked := false
					for prev := 0; prev < currK; prev++ {
						if indices[prev] == j {
							alreadyPicked = true
							break
						}
					}
					if alreadyPicked {
						continue
					}
					if nextB, ok := MatchAST(pTerm, tAdd.Terms[j], currBindings); ok {
						indices[currK] = j
						if finalB, idxs, found := findSubMatch(currK+1, 0, nextB); found {
							return finalB, idxs, true
						}
					}
				}
				return currBindings, nil, false
			}

			if bindings, matchedIndices, found := findSubMatch(0, 0, PatternBindings{}); found {
				repl := InstantiateTemplate(rule.RHS, bindings)
				matchedSet := make(map[int]bool)
				for _, idx := range matchedIndices {
					matchedSet[idx] = true
				}
				newTerms := make([]ast.Node, 0, m-k+1)
				newTerms = append(newTerms, repl)
				for j := 0; j < m; j++ {
					if !matchedSet[j] {
						newTerms = append(newTerms, tAdd.Terms[j])
					}
				}
				return ast.NewAdd(newTerms), true
			}
		}
	}

	// 3. AC Partial Matching: MulNode
	// If rule.LHS is MulNode(p_1, ..., p_k) and target is MulNode(t_1, ..., t_m) with k < m
	if pMul, okP := rule.LHS.(*ast.MulNode); okP {
		if tMul, okT := target.(*ast.MulNode); okT && len(pMul.Factors) < len(tMul.Factors) {
			k := len(pMul.Factors)
			m := len(tMul.Factors)
			indices := make([]int, k)
			var findSubMatchMul func(currK int, currBindings PatternBindings) (PatternBindings, []int, bool)
			findSubMatchMul = func(currK int, currBindings PatternBindings) (PatternBindings, []int, bool) {
				if currK == k {
					return currBindings, indices, true
				}
				pFactor := pMul.Factors[currK]
				for j := 0; j < m; j++ {
					alreadyPicked := false
					for prev := 0; prev < currK; prev++ {
						if indices[prev] == j {
							alreadyPicked = true
							break
						}
					}
					if alreadyPicked {
						continue
					}
					if nextB, ok := MatchAST(pFactor, tMul.Factors[j], currBindings); ok {
						indices[currK] = j
						if finalB, idxs, found := findSubMatchMul(currK+1, nextB); found {
							return finalB, idxs, true
						}
					}
				}
				return currBindings, nil, false
			}

			if bindings, matchedIndices, found := findSubMatchMul(0, PatternBindings{}); found {
				repl := InstantiateTemplate(rule.RHS, bindings)
				matchedSet := make(map[int]bool)
				for _, idx := range matchedIndices {
					matchedSet[idx] = true
				}
				newFactors := make([]ast.Node, 0, m-k+1)
				newFactors = append(newFactors, repl)
				for j := 0; j < m; j++ {
					if !matchedSet[j] {
						newFactors = append(newFactors, tMul.Factors[j])
					}
				}
				return ast.NewMul(newFactors), true
			}
		}
	}

	return target, false
}

// ApplyRuleOnceBottomUp applies a list of rules to the expression tree bottom-up.
// It returns the transformed expression and true if any rule was applied.
func ApplyRuleOnceBottomUp(expr ast.Node, rules []*ast.RuleNode, recorder *provenance.TraceRecorder) (ast.Node, bool) {
	if expr == nil || len(rules) == 0 {
		return expr, false
	}

	var anyApplied bool

	var visit func(n ast.Node) (ast.Node, bool)
	visit = func(n ast.Node) (ast.Node, bool) {
		if n == nil {
			return nil, false
		}

		// First recursively transform children
		var childChanged bool
		var transformedChild ast.Node

		switch v := n.(type) {
		case *ast.SqrtNode:
			newR, ch := visit(v.Radicand)
			if ch {
				childChanged = true
				transformedChild = ast.NewSqrt(newR)
			} else {
				transformedChild = v
			}
		case *ast.FuncNode:
			newArgs := make([]ast.Node, len(v.Args))
			for i, a := range v.Args {
				na, ch := visit(a)
				if ch {
					childChanged = true
				}
				newArgs[i] = na
			}
			if childChanged {
				transformedChild = &ast.FuncNode{Name: v.Name, Args: newArgs}
			} else {
				transformedChild = v
			}
		case *ast.PowNode:
			newBase, chB := visit(v.Base)
			newExp, chE := visit(v.Exp)
			if chB || chE {
				childChanged = true
				transformedChild = &ast.PowNode{Base: newBase, Exp: newExp}
			} else {
				transformedChild = v
			}
		case *ast.UnaryOpNode:
			newE, ch := visit(v.Expr)
			if ch {
				childChanged = true
				transformedChild = &ast.UnaryOpNode{Op: v.Op, Expr: newE}
			} else {
				transformedChild = v
			}
		case *ast.AddNode:
			newTerms := make([]ast.Node, len(v.Terms))
			for i, t := range v.Terms {
				nt, ch := visit(t)
				if ch {
					childChanged = true
				}
				newTerms[i] = nt
			}
			if childChanged {
				transformedChild = ast.NewAdd(newTerms)
			} else {
				transformedChild = v
			}
		case *ast.MulNode:
			newFactors := make([]ast.Node, len(v.Factors))
			for i, f := range v.Factors {
				nf, ch := visit(f)
				if ch {
					childChanged = true
				}
				newFactors[i] = nf
			}
			if childChanged {
				transformedChild = ast.NewMul(newFactors)
			} else {
				transformedChild = v
			}
		case *ast.ListNode:
			newElems := make([]ast.Node, len(v.Elements))
			for i, e := range v.Elements {
				ne, ch := visit(e)
				if ch {
					childChanged = true
				}
				newElems[i] = ne
			}
			if childChanged {
				transformedChild = ast.NewList(newElems)
			} else {
				transformedChild = v
			}
		case *ast.RelOpNode:
			newL, chL := visit(v.LHS)
			newR, chR := visit(v.RHS)
			if chL || chR {
				childChanged = true
				transformedChild = ast.NewRelOp(newL, v.Op, newR)
			} else {
				transformedChild = v
			}
		default:
			transformedChild = n
		}

		// Try applying rules at transformedChild node
		for _, r := range rules {
			if repl, matched := TryApplyRuleAtRoot(r, transformedChild); matched {
				anyApplied = true
				if recorder != nil {
					recorder.RecordStep(
						"TRS_RULE_APPLY",
						provenance.RelationEq,
						nil,
						transformedChild,
						repl,
						map[string]ast.Node{"rule": r},
						nil,
						provenance.TrustHeuristic,
						provenance.VisPresentable,
					)
				}
				return repl, true
			}
		}

		return transformedChild, childChanged
	}

	res, changed := visit(expr)
	return res, changed || anyApplied
}

// extractRules converts an ast.Node (RuleNode or ListNode of RuleNodes) into []*ast.RuleNode.
func extractRules(node ast.Node) ([]*ast.RuleNode, error) {
	if node == nil {
		return nil, fmt.Errorf("rules cannot be nil")
	}
	if r, ok := node.(*ast.RuleNode); ok {
		return []*ast.RuleNode{r}, nil
	}
	if list, ok := node.(*ast.ListNode); ok {
		res := make([]*ast.RuleNode, 0, len(list.Elements))
		for _, elem := range list.Elements {
			if r, ok := elem.(*ast.RuleNode); ok {
				res = append(res, r)
			} else {
				return nil, fmt.Errorf("list must contain only rule(...) objects, got: %s", elem)
			}
		}
		return res, nil
	}
	return nil, fmt.Errorf("expected rule or list of rules, got: %s", node)
}

// EvalRule implements rule(lhs, rhs).
func EvalRule(args []ast.Node, env *Env) (ast.Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("rule requires exactly 2 arguments (lhs, rhs), got %d", len(args))
	}
	// LHS and RHS are unevaluated patterns
	return ast.NewRuleNode(args[0], args[1]), nil
}

// EvalReplace implements replace(expr, rule_or_list).
func EvalReplace(args []ast.Node, env *Env) (ast.Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("replace requires 2 arguments (expr, rules), got %d", len(args))
	}
	targetExpr, err := EvalWithEnv(args[0], env)
	if err != nil {
		return nil, err
	}
	ruleArg, err := EvalWithEnv(args[1], env)
	if err != nil {
		return nil, err
	}
	rules, err := extractRules(ruleArg)
	if err != nil {
		return nil, err
	}

	var recorder *provenance.TraceRecorder
	if env != nil {
		recorder = env.GetTracer()
	}

	res, _ := ApplyRuleOnceBottomUp(targetExpr, rules, recorder)
	return res, nil
}

// EvalRewrite implements rewrite(expr, rule_or_list, [max_steps]).
func EvalRewrite(args []ast.Node, env *Env) (ast.Node, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("rewrite requires 2 or 3 arguments (expr, rules, [max_steps]), got %d", len(args))
	}

	targetExpr, err := EvalWithEnv(args[0], env)
	if err != nil {
		return nil, err
	}
	ruleArg, err := EvalWithEnv(args[1], env)
	if err != nil {
		return nil, err
	}
	rules, err := extractRules(ruleArg)
	if err != nil {
		return nil, err
	}

	// Determine max steps: argument > config > default 1000
	maxSteps := 1000
	cfg, _ := i18n.LoadConfig()
	if cfg != nil {
		maxSteps = cfg.GetRewriteMaxSteps()
	}
	if len(args) == 3 {
		stepNode, err := EvalWithEnv(args[2], env)
		if err != nil {
			return nil, err
		}
		if rat, ok := stepNode.(*ast.RationalNode); ok && rat.Val.IsInt() {
			maxSteps = int(rat.Val.Num().Int64())
		} else {
			return nil, fmt.Errorf("rewrite max_steps must be an integer, got %s", stepNode)
		}
	}

	var recorder *provenance.TraceRecorder
	if env != nil {
		recorder = env.GetTracer()
	}

	curr := targetExpr
	history := make(map[string]bool)
	history[curr.String()] = true

	for step := 0; step < maxSteps; step++ {
		next, applied := ApplyRuleOnceBottomUp(curr, rules, recorder)
		if !applied || curr.Equal(next) {
			// Fixed point reached!
			return next, nil
		}

		strRepr := next.String()
		if history[strRepr] {
			// Cycle/Oscillation detected! Stop to prevent infinite loop.
			return next, nil
		}
		history[strRepr] = true
		curr = next
	}

	// Maximum step bound reached
	return curr, nil
}
