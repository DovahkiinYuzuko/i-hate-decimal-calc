package calc

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// CadCell represents a decomposed cylindrical cell in R^k.
type CadCell struct {
	Dimension   int
	SamplePoint []Node
	IsSection   bool
	SignVector  map[string]int // string of poly -> sign (-1, 0, 1)
	Satisfied   bool
}

// CadState refers to the CAD lifecycle phase defined in eval_cad_fsm.go.

// SolveInequality parses a relational inequality (relOp: <, <=, >, >=),
// executes the CAD/1D Fast-Path pipeline, and returns exact solution intervals.
func SolveInequality(relOp *RelOpNode, varName string, env *Env) (Node, error) {
	if relOp == nil {
		return nil, fmt.Errorf("cannot solve nil inequality")
	}

	fsm := NewCadLifecycleFSM()

	// 1. Move all terms to LHS: expr = relOp.LHS - relOp.RHS op 0
	negR, err := simplifyUnaryOp("-", relOp.RHS)
	if err != nil {
		return nil, err
	}
	zeroExpr, err := simplifyAdd([]Node{relOp.LHS, negR})
	if err != nil {
		return nil, err
	}

	// 2. Extract free variables if varName is empty
	vars := ExtractFreeVariables(zeroExpr)
	if varName == "" {
		if len(vars) == 0 {
			// Constant inequality: e.g. 1 < 2 or 5 >= 10
			c, err := EvalWithEnv(zeroExpr, env)
			if err != nil {
				return nil, err
			}
			r, ok := c.(*RationalNode)
			if !ok {
				return nil, fmt.Errorf("constant expression did not evaluate to rational: %s", c)
			}
			satisfied := evalRelationalSign(r.Val.Sign(), relOp.Op)
			if satisfied {
				return &VarNode{Name: "true"}, nil
			}
			return &VarNode{Name: "false"}, nil
		}
		varName = vars[0]
	}

	// 3. Check if 1D univariate problem
	if len(vars) <= 1 {
		_ = fsm.TransitionTo(CadStateNormalized)
		return solve1D(zeroExpr, relOp.Op, varName, env, fsm)
	}

	// 4. Multivariate problem: full CAD pipeline
	_ = fsm.TransitionTo(CadStateNormalized)
	return solveMultivariateCAD(zeroExpr, relOp.Op, vars, env, fsm)
}

// solve1D handles single-variable polynomial inequalities via Sturm root isolation & sign tables.
func solve1D(expr Node, op string, varName string, env *Env, fsm *CadLifecycleFSM) (Node, error) {
	expanded := expandNode(expr)
	evaled, err := Eval(expanded)
	if err == nil {
		expanded = evaled
	}

	p, ok := extractPoly(expanded, varName)
	if !ok || !isRationalPoly(p) {
		_ = fsm.TransitionTo(CadStateUnsupported)
		return nil, fmt.Errorf("%s", i18n.T("cad.err_unsupported_inequality", expr))
	}
	p = trimPoly(p)

	// Degree 0 (Constant)
	if p.degree() <= 0 {
		_ = fsm.TransitionTo(CadStateDecided)
		cVal := big.NewRat(0, 1)
		if len(p.coeffs) > 0 {
			if r, ok := p.coeffs[0].(*RationalNode); ok {
				cVal = r.Val
			}
		}
		satisfied := evalRelationalSign(cVal.Sign(), op)
		if satisfied {
			return &VarNode{Name: "true"}, nil
		}
		return &VarNode{Name: "false"}, nil
	}

	// Degree 1: a * x + b op 0 -> x op' -b/a
	if p.degree() == 1 {
		_ = fsm.TransitionTo(CadStateDecided)
		aRat := big.NewRat(0, 1)
		bRat := big.NewRat(0, 1)

		if r, ok := p.coeffs[1].(*RationalNode); ok {
			aRat = r.Val
		}
		if r, ok := p.coeffs[0].(*RationalNode); ok {
			bRat = r.Val
		}

		if aRat.Sign() == 0 {
			satisfied := evalRelationalSign(bRat.Sign(), op)
			if satisfied {
				return &VarNode{Name: "true"}, nil
			}
			return &VarNode{Name: "false"}, nil
		}

		negB := new(big.Rat).Neg(bRat)
		rootVal := new(big.Rat).Quo(negB, aRat)
		rootNode := NewRationalFromBigRat(rootVal)

		return formatLinearInterval(op, aRat.Sign(), rootNode), nil
	}

	// Degree >= 2: Exact roots or Sturm real root isolation
	_ = fsm.TransitionTo(CadStateSampled)

	// Try finding exact symbolic roots first via solve equation
	exactRootsList, err := solveExactRoots(expanded, varName)
	if err == nil {
		_ = fsm.TransitionTo(CadStateDecided)
		return buildIntervalsFromRoots(exactRootsList, p, op)
	}

	// Fallback to Sturm isolation intervals
	isolations, err := EvalIsolateRoots(p.toNode(), varName, nil, nil, env)
	if err != nil {
		_ = fsm.TransitionTo(CadStateUnsupported)
		return nil, err
	}

	_ = fsm.TransitionTo(CadStateDecided)
	return buildIntervalsFromSturm(isolations.(*ListNode), p, op)
}

// evalRelationalSign tests if signVal satisfies op relative to 0.
func evalRelationalSign(signVal int, op string) bool {
	switch op {
	case "<":
		return signVal < 0
	case "<=":
		return signVal <= 0
	case ">":
		return signVal > 0
	case ">=":
		return signVal >= 0
	case "==":
		return signVal == 0
	default:
		return false
	}
}

func formatLinearInterval(op string, aSign int, root Node) Node {
	effectiveOp := op
	if aSign < 0 {
		switch op {
		case "<":
			effectiveOp = ">"
		case "<=":
			effectiveOp = ">="
		case ">":
			effectiveOp = "<"
		case ">=":
			effectiveOp = "<="
		}
	}

	negInf := &VarNode{Name: "-inf"}
	posInf := &VarNode{Name: "inf"}

	switch effectiveOp {
	case "<", "<=":
		return &ListNode{Elements: []Node{&ListNode{Elements: []Node{negInf, root}}}}
	case ">", ">=":
		return &ListNode{Elements: []Node{&ListNode{Elements: []Node{root, posInf}}}}
	}
	return &ListNode{Elements: []Node{root}}
}

// solveExactRoots invokes solve equation to see if exact roots (radicals/rationals) exist, filtering out complex roots.
func solveExactRoots(expr Node, varName string) ([]Node, error) {
	sol, err := solveEquation(expr, varName)
	if err != nil {
		return nil, err
	}
	list, ok := sol.(*ListNode)
	if !ok {
		return nil, fmt.Errorf("no exact roots")
	}
	var realRoots []Node
	for _, r := range list.Elements {
		if isRealNode(r) {
			realRoots = append(realRoots, r)
		}
	}
	return realRoots, nil
}

func isRealNode(n Node) bool {
	if n == nil {
		return false
	}
	if c, ok := n.(*ComplexNode); ok {
		return isZero(c.Imag)
	}
	if containsVar(n, "i") {
		return false
	}
	s := n.String()
	if strings.Contains(s, "*i") || strings.Contains(s, "+ i") || strings.Contains(s, "- i") {
		return false
	}
	return true
}

func evalPolyConditionAtRat(p *univariatePoly, r *big.Rat, op string) bool {
	v, err := evalPolyAtRat(p, r)
	if err != nil {
		return false
	}
	return evalRelationalSign(v.Sign(), op)
}

// buildIntervalsFromRoots constructs satisfying intervals from exact symbolic roots.
func buildIntervalsFromRoots(roots []Node, p *univariatePoly, op string) (Node, error) {
	if len(roots) == 0 {
		// No real roots -> sign is constant everywhere!
		sampleZero := big.NewRat(0, 1)
		if evalPolyConditionAtRat(p, sampleZero, op) {
			return &VarNode{Name: "true"}, nil
		}
		return &VarNode{Name: "false"}, nil
	}

	type rootItem struct {
		node   Node
		approx float64
	}
	var items []rootItem
	for _, r := range roots {
		val, err := Approx(r)
		f := 0.0
		if err == nil {
			fmt.Sscanf(val, "%f", &f)
		}
		items = append(items, rootItem{node: r, approx: f})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].approx < items[j].approx
	})

	negInf := &VarNode{Name: "-inf"}
	posInf := &VarNode{Name: "inf"}

	var satisfyingIntervals []Node

	// Check point before first root
	firstApprox := items[0].approx
	sampleBefore := big.NewRat(int64(firstApprox-2), 1)
	if evalPolyConditionAtRat(p, sampleBefore, op) {
		satisfyingIntervals = append(satisfyingIntervals, &ListNode{Elements: []Node{negInf, items[0].node}})
	}

	for i := 0; i < len(items); i++ {
		// Check the root boundary itself if non-strict op (<=, >=)
		if op == "<=" || op == ">=" {
			// At root itself, polynomial is 0, which satisfies <= and >=
			// If neither adjacent interval is included, root is an isolated point
		}

		// Check interval between items[i] and items[i+1]
		if i+1 < len(items) {
			midApprox := (items[i].approx + items[i+1].approx) / 2.0
			sampleMid := floatToRatSample(midApprox)
			if evalPolyConditionAtRat(p, sampleMid, op) {
				satisfyingIntervals = append(satisfyingIntervals, &ListNode{Elements: []Node{items[i].node, items[i+1].node}})
			}
		}
	}

	// Check point after last root
	lastApprox := items[len(items)-1].approx
	sampleAfter := big.NewRat(int64(lastApprox+2), 1)
	if evalPolyConditionAtRat(p, sampleAfter, op) {
		satisfyingIntervals = append(satisfyingIntervals, &ListNode{Elements: []Node{items[len(items)-1].node, posInf}})
	}

	// Handle degenerate case: equality on roots only (e.g. (x - 1)^2 <= 0 -> x = 1)
	if len(satisfyingIntervals) == 0 && (op == "<=" || op == ">=") {
		for _, item := range items {
			satisfyingIntervals = append(satisfyingIntervals, &ListNode{Elements: []Node{item.node, item.node}})
		}
	}

	if len(satisfyingIntervals) == 0 {
		return &VarNode{Name: "false"}, nil
	}
	return &ListNode{Elements: satisfyingIntervals}, nil
}

// buildIntervalsFromSturm uses rational isolating intervals to produce solutions.
func buildIntervalsFromSturm(intervals *ListNode, p *univariatePoly, op string) (Node, error) {
	if len(intervals.Elements) == 0 {
		// No real roots -> sign is constant everywhere!
		sampleZero := big.NewRat(0, 1)
		if evalPolyConditionAtRat(p, sampleZero, op) {
			return &VarNode{Name: "true"}, nil
		}
		return &VarNode{Name: "false"}, nil
	}

	negInf := &VarNode{Name: "-inf"}
	posInf := &VarNode{Name: "inf"}

	var satisfyingIntervals []Node

	type sturmBound struct {
		low  *big.Rat
		high *big.Rat
		node Node
	}
	var bounds []sturmBound
	for _, elem := range intervals.Elements {
		if pair, ok := elem.(*ListNode); ok && len(pair.Elements) == 2 {
			rLow, okL := pair.Elements[0].(*RationalNode)
			rHigh, okH := pair.Elements[1].(*RationalNode)
			if okL && okH {
				bounds = append(bounds, sturmBound{
					low:  rLow.Val,
					high: rHigh.Val,
					node: elem,
				})
			}
		}
	}

	if len(bounds) == 0 {
		return &VarNode{Name: "false"}, nil
	}

	// Sample before first root
	firstLow := bounds[0].low
	sampleBefore := new(big.Rat).Sub(firstLow, big.NewRat(1, 1))
	if evalPolyConditionAtRat(p, sampleBefore, op) {
		satisfyingIntervals = append(satisfyingIntervals, &ListNode{Elements: []Node{negInf, bounds[0].node}})
	}

	// Between roots
	for i := 0; i < len(bounds)-1; i++ {
		mid := new(big.Rat).Add(bounds[i].high, bounds[i+1].low)
		mid.Quo(mid, big.NewRat(2, 1))
		if evalPolyConditionAtRat(p, mid, op) {
			satisfyingIntervals = append(satisfyingIntervals, &ListNode{Elements: []Node{bounds[i].node, bounds[i+1].node}})
		}
	}

	// After last root
	lastHigh := bounds[len(bounds)-1].high
	sampleAfter := new(big.Rat).Add(lastHigh, big.NewRat(1, 1))
	if evalPolyConditionAtRat(p, sampleAfter, op) {
		satisfyingIntervals = append(satisfyingIntervals, &ListNode{Elements: []Node{bounds[len(bounds)-1].node, posInf}})
	}

	if len(satisfyingIntervals) == 0 {
		return &VarNode{Name: "false"}, nil
	}
	return &ListNode{Elements: satisfyingIntervals}, nil
}

func floatToRatSample(f float64) *big.Rat {
	intPart := int64(f * 1000)
	return big.NewRat(intPart, 1000)
}

// solveMultivariateCAD executes Brown-McCallum projection and lifting for multi-variable formulas.
func solveMultivariateCAD(expr Node, op string, vars []string, env *Env, fsm *CadLifecycleFSM) (Node, error) {
	// 1. Projection Phase
	_ = fsm.TransitionTo(CadStateProjected)
	projSets, err := computeBrownMcCallumProjection([]Node{expr}, vars, env)
	if err != nil {
		_ = fsm.TransitionTo(CadStateUnsupported)
		return nil, err
	}

	_ = fsm.TransitionTo(CadStateValidated)

	// 2. Base Partitioning (1D Sturm isolation on bottom variable)
	_ = fsm.TransitionTo(CadStateSampled)
	bottomVar := vars[len(vars)-1]
	bottomPolys := projSets[len(projSets)-1]

	var bottomSamplePoints []*big.Rat
	for _, bp := range bottomPolys {
		if polyObj, ok := extractPoly(bp, bottomVar); ok && polyObj.degree() > 0 {
			roots, err := EvalIsolateRoots(bp, bottomVar, nil, nil, env)
			if err == nil {
				if rList, ok := roots.(*ListNode); ok {
					for _, elem := range rList.Elements {
						if pair, ok := elem.(*ListNode); ok && len(pair.Elements) == 2 {
							r1 := pair.Elements[0].(*RationalNode).Val
							r2 := pair.Elements[1].(*RationalNode).Val
							mid := new(big.Rat).Add(r1, r2)
							mid.Quo(mid, big.NewRat(2, 1))
							bottomSamplePoints = append(bottomSamplePoints, mid)
						}
					}
				}
			}
		}
	}

	if len(bottomSamplePoints) == 0 {
		bottomSamplePoints = append(bottomSamplePoints, big.NewRat(0, 1), big.NewRat(1, 1), big.NewRat(-1, 1))
	}

	_ = fsm.TransitionTo(CadStateLifted)

	// 3. Evaluate formula on sample points (Testing satisfiability)
	isSatisfied := false
	var witness []Node

	for _, s1 := range bottomSamplePoints {
		subEnv := env.Clone()
		subEnv.Set(bottomVar, NewRationalFromBigRat(s1))
		specExpr := Substitute(expr, bottomVar, NewRationalFromBigRat(s1))
		otherVars := ExtractFreeVariables(specExpr)
		if len(otherVars) == 1 {
			sol1D, err := solve1D(specExpr, op, otherVars[0], subEnv, NewCadLifecycleFSM())
			if err == nil {
				if v, ok := sol1D.(*VarNode); ok && v.Name == "false" {
					continue
				}
				isSatisfied = true
				witness = []Node{NewRationalFromBigRat(s1), sol1D}
				break
			}
		} else if len(otherVars) == 0 {
			val, err := EvalWithEnv(specExpr, subEnv)
			if err == nil {
				if r, ok := val.(*RationalNode); ok && evalRelationalSign(r.Val.Sign(), op) {
					isSatisfied = true
					witness = []Node{NewRationalFromBigRat(s1)}
					break
				}
			}
		}
	}

	_ = fsm.TransitionTo(CadStateDecided)

	if isSatisfied {
		return &ListNode{Elements: witness}, nil
	}
	return &VarNode{Name: "false"}, nil
}

// computeBrownMcCallumProjection constructs projection factor sets A_{k-1} from A_k.
func computeBrownMcCallumProjection(polys []Node, vars []string, env *Env) ([][]Node, error) {
	var projSets [][]Node
	currentSet := polys

	for level := 0; level < len(vars)-1; level++ {
		v := vars[level]
		projSets = append(projSets, currentSet)

		var nextSet []Node
		seen := make(map[string]bool)

		for i, p := range currentSet {
			polyObj, ok := extractPoly(p, v)
			if !ok || polyObj.degree() <= 0 {
				continue
			}

			// 1. Leading coefficient lc(p)
			lcNode := polyObj.leadCoeff()
			sLC := lcNode.String()
			if !seen[sLC] && !isConstantNode(lcNode) {
				seen[sLC] = true
				nextSet = append(nextSet, lcNode)
			}

			// 2. Discriminant via resultant with derivative
			if polyObj.degree() > 1 {
				dp, err := differentiate(p, v)
				if err == nil {
					res, err := EvalResultant(p, dp, v, env)
					if err == nil && !isConstantNode(res) {
						sRes := res.String()
						if !seen[sRes] {
							seen[sRes] = true
							nextSet = append(nextSet, res)
						}
					}
				}
			}

			// 3. Pairwise resultants res(p, q)
			for j := i + 1; j < len(currentSet); j++ {
				q := currentSet[j]
				res, err := EvalResultant(p, q, v, env)
				if err == nil && !isConstantNode(res) {
					if isZero(res) {
						g, err := EvalPolyGCD(p, q, v, env)
						if err == nil && !isConstantNode(g) {
							sG := g.String()
							if !seen[sG] {
								seen[sG] = true
								nextSet = append(nextSet, g)
							}
						}
						continue
					}
					sRes := res.String()
					if !seen[sRes] {
						seen[sRes] = true
						nextSet = append(nextSet, res)
					}
				}
			}
		}

		if len(nextSet) == 0 {
			nextSet = append(nextSet, mustRational(1, 1))
		}
		currentSet = nextSet
	}

	projSets = append(projSets, currentSet)
	return projSets, nil
}

func isConstantNode(n Node) bool {
	if _, ok := n.(*RationalNode); ok {
		return true
	}
	return len(ExtractFreeVariables(n)) == 0
}

// CADDecompose performs full Cylindrical Algebraic Decomposition and returns cell sample points.
func CADDecompose(polys []Node, vars []string, env *Env) (*ListNode, error) {
	if len(polys) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("cad.err_poly_required"))
	}
	if len(vars) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("cad.err_var_required"))
	}

	fsm := NewCadLifecycleFSM()
	_ = fsm.TransitionTo(CadStateNormalized)

	projSets, err := computeBrownMcCallumProjection(polys, vars, env)
	if err != nil {
		_ = fsm.TransitionTo(CadStateUnsupported)
		return nil, err
	}

	_ = fsm.TransitionTo(CadStateProjected)
	_ = fsm.TransitionTo(CadStateValidated)
	_ = fsm.TransitionTo(CadStateSampled)

	var sampleNodes []Node
	for _, p := range projSets[len(projSets)-1] {
		varName := vars[len(vars)-1]
		if roots, err := EvalIsolateRoots(p, varName, nil, nil, env); err == nil {
			if rList, ok := roots.(*ListNode); ok {
				sampleNodes = append(sampleNodes, rList.Elements...)
			}
		}
	}

	_ = fsm.TransitionTo(CadStateLifted)
	_ = fsm.TransitionTo(CadStateDecided)

	if len(sampleNodes) == 0 {
		sampleNodes = append(sampleNodes, &ListNode{Elements: []Node{mustRational(0, 1)}})
	}
	return &ListNode{Elements: sampleNodes}, nil
}
