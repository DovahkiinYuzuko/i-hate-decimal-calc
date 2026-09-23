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
	Parent      *CadCell // Pointer to parent cell in R^{k-1} (Cylindrical Ancestry)
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

	// 2. Extract free variables
	vars := ExtractFreeVariables(zeroExpr)
	if len(vars) == 0 {
		// Constant inequality: e.g. 1 < 2, pi < 22/7, sin(1) > 1/2
		if res, decided := EvaluateRelOpWithInterval(relOp); decided {
			if res {
				return &VarNode{Name: "true"}, nil
			}
			return &VarNode{Name: "false"}, nil
		}
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
	if varName == "" {
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

	// Degree >= 2: True 1D CAD Cell Decomposition
	_ = fsm.TransitionTo(CadStateSampled)

	cells, roots, err := decompose1DCADInternal([]*univariatePoly{p}, varName, env)
	if err != nil {
		_ = fsm.TransitionTo(CadStateUnsupported)
		return nil, err
	}

	_ = fsm.TransitionTo(CadStateDecided)
	return reconstruct1DIntervalsFromCells(cells, roots, p, op)
}

// Decompose1DCAD decomposes R^1 into 2m + 1 sign-invariant CAD cells for the given univariate polynomials.
func Decompose1DCAD(polys []*univariatePoly, varName string, env *Env) ([]CadCell, error) {
	cells, _, err := decompose1DCADInternal(polys, varName, env)
	return cells, err
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

// cad1DRoot represents an exact algebraic root on R^1 with rational isolating intervals.
type cad1DRoot struct {
	node Node
	low  *big.Rat
	high *big.Rat
	poly *univariatePoly
}

// decompose1DCADInternal decomposes R^1 into 2m + 1 sign-invariant CAD cells.
func decompose1DCADInternal(polys []*univariatePoly, varName string, env *Env) ([]CadCell, []cad1DRoot, error) {
	var allRoots []cad1DRoot

	for _, p := range polys {
		if p == nil || p.degree() <= 0 {
			continue
		}
		pTrimmed := trimPoly(p)
		if pTrimmed.degree() <= 0 {
			continue
		}

		// 1. Try finding exact symbolic roots first
		exactRoots, err := solveExactRoots(pTrimmed.toNode(), varName)
		if err == nil && len(exactRoots) > 0 {
			for _, r := range exactRoots {
				low, high, ok := getExactRootBounds(r, pTrimmed)
				if ok {
					allRoots = append(allRoots, cad1DRoot{
						node: r,
						low:  low,
						high: high,
						poly: pTrimmed,
					})
				}
			}
		} else {
			// 2. Fallback to Sturm real root isolation
			isolations, err := EvalIsolateRoots(pTrimmed.toNode(), varName, nil, nil, env)
			if err == nil {
				if list, ok := isolations.(*ListNode); ok {
					for _, elem := range list.Elements {
						if pair, ok := elem.(*ListNode); ok && len(pair.Elements) == 2 {
							rLow, okL := pair.Elements[0].(*RationalNode)
							rHigh, okH := pair.Elements[1].(*RationalNode)
							if okL && okH {
								var rootNode Node
								if rLow.Val.Cmp(rHigh.Val) == 0 {
									rootNode = rLow
								} else {
									rootNode = pair
								}
								allRoots = append(allRoots, cad1DRoot{
									node: rootNode,
									low:  new(big.Rat).Set(rLow.Val),
									high: new(big.Rat).Set(rHigh.Val),
									poly: pTrimmed,
								})
							}
						}
					}
				}
			}
		}
	}

	// Strictly sort roots without any floating-point numbers
	sortCadRoots(allRoots)

	// Deduplicate identical roots
	dedupRoots := deduplicateCadRoots(allRoots)

	// Ensure adjacent isolating intervals are strictly disjoint: R_i < L_{i+1}
	refineRootIntervals(dedupRoots)

	m := len(dedupRoots)
	if m == 0 {
		zeroRat := big.NewRat(0, 1)
		cell := CadCell{
			Dimension:   1,
			SamplePoint: []Node{NewRationalFromBigRat(zeroRat)},
			IsSection:   false,
			SignVector:  make(map[string]int),
		}
		for _, p := range polys {
			if val, err := evalPolyAtRat(p, zeroRat); err == nil {
				cell.SignVector[polyKey(p)] = val.Sign()
			}
		}
		return []CadCell{cell}, dedupRoots, nil
	}

	numCells := 2*m + 1
	cells := make([]CadCell, numCells)

	// Sector 0: (-inf, alpha_1)
	firstLow := dedupRoots[0].low
	sample0 := floorSubOne(firstLow)
	cells[0] = makeCadSectorCell([]Node{NewRationalFromBigRat(sample0)}, sample0, polys)

	for i := 0; i < m; i++ {
		root := dedupRoots[i]
		// Section i+1: {alpha_{i+1}}
		cells[2*i+1] = makeCadSectionCell([]Node{root.node}, root, polys)

		// Sector i+1: (alpha_{i+1}, alpha_{i+2}) or (alpha_m, +inf)
		if i+1 < m {
			nextLow := dedupRoots[i+1].low
			currHigh := dedupRoots[i].high
			mid := new(big.Rat).Add(currHigh, nextLow)
			mid.Quo(mid, big.NewRat(2, 1))
			cells[2*i+2] = makeCadSectorCell([]Node{NewRationalFromBigRat(mid)}, mid, polys)
		} else {
			lastHigh := dedupRoots[m-1].high
			sampleLast := ceilAddOne(lastHigh)
			cells[2*m] = makeCadSectorCell([]Node{NewRationalFromBigRat(sampleLast)}, sampleLast, polys)
		}
	}

	return cells, dedupRoots, nil
}

func getExactRootBounds(r Node, p *univariatePoly) (*big.Rat, *big.Rat, bool) {
	if rat, ok := r.(*RationalNode); ok {
		return new(big.Rat).Set(rat.Val), new(big.Rat).Set(rat.Val), true
	}
	iv, err := EvalNodeInterval(r, 6)
	if err == nil && iv.Low != nil && iv.High != nil {
		return new(big.Rat).Set(iv.Low), new(big.Rat).Set(iv.High), true
	}
	return nil, nil, false
}

func sortCadRoots(roots []cad1DRoot) {
	sort.SliceStable(roots, func(i, j int) bool {
		return compareCadRoots(roots[i], roots[j]) < 0
	})
}

func compareCadRoots(a, b cad1DRoot) int {
	if a.high.Cmp(b.low) < 0 {
		return -1
	}
	if b.high.Cmp(a.low) < 0 {
		return 1
	}
	if a.node != nil && b.node != nil && a.node.Equal(b.node) {
		return 0
	}
	if a.node != nil && b.node != nil {
		if res, decided := EvaluateRelOpWithInterval(&RelOpNode{LHS: a.node, Op: "<", RHS: b.node}); decided {
			if res {
				return -1
			}
			return 1
		}
	}
	// Midpoint comparison as fallback
	midA := new(big.Rat).Add(a.low, a.high)
	midA.Quo(midA, big.NewRat(2, 1))
	midB := new(big.Rat).Add(b.low, b.high)
	midB.Quo(midB, big.NewRat(2, 1))
	return midA.Cmp(midB)
}

func deduplicateCadRoots(roots []cad1DRoot) []cad1DRoot {
	if len(roots) == 0 {
		return nil
	}
	dedup := []cad1DRoot{roots[0]}
	for i := 1; i < len(roots); i++ {
		prev := dedup[len(dedup)-1]
		curr := roots[i]
		if compareCadRoots(prev, curr) == 0 {
			continue
		}
		dedup = append(dedup, curr)
	}
	return dedup
}

func refineRootIntervals(roots []cad1DRoot) {
	for i := 0; i+1 < len(roots); i++ {
		// If high[i] >= low[i+1], shrink them
		if roots[i].high.Cmp(roots[i+1].low) >= 0 {
			mid := new(big.Rat).Add(roots[i].high, roots[i+1].low)
			mid.Quo(mid, big.NewRat(2, 1))
			delta := new(big.Rat).Sub(roots[i+1].high, roots[i].low)
			if delta.Sign() > 0 {
				delta.Quo(delta, big.NewRat(16, 1))
				roots[i].high = new(big.Rat).Sub(mid, delta)
				roots[i+1].low = new(big.Rat).Add(mid, delta)
			}
		}
	}
}

func floorSubOne(r *big.Rat) *big.Rat {
	intNum := new(big.Int).Quo(r.Num(), r.Denom())
	if r.Sign() < 0 && new(big.Int).Rem(r.Num(), r.Denom()).Sign() != 0 {
		intNum.Sub(intNum, big.NewInt(1))
	}
	intNum.Sub(intNum, big.NewInt(1))
	return new(big.Rat).SetInt(intNum)
}

func ceilAddOne(r *big.Rat) *big.Rat {
	intNum := new(big.Int).Quo(r.Num(), r.Denom())
	if r.Sign() > 0 && new(big.Int).Rem(r.Num(), r.Denom()).Sign() != 0 {
		intNum.Add(intNum, big.NewInt(1))
	}
	intNum.Add(intNum, big.NewInt(1))
	return new(big.Rat).SetInt(intNum)
}

func polyKey(p *univariatePoly) string {
	if p == nil {
		return ""
	}
	return p.toNode().String()
}

func makeCadSectorCell(samplePoint []Node, sampleRat *big.Rat, polys []*univariatePoly) CadCell {
	cell := CadCell{
		Dimension:   1,
		SamplePoint: samplePoint,
		IsSection:   false,
		SignVector:  make(map[string]int),
	}
	for _, p := range polys {
		if p != nil {
			if val, err := evalPolyAtRat(p, sampleRat); err == nil {
				cell.SignVector[polyKey(p)] = val.Sign()
			}
		}
	}
	return cell
}

func makeCadSectionCell(samplePoint []Node, root cad1DRoot, polys []*univariatePoly) CadCell {
	cell := CadCell{
		Dimension:   0,
		SamplePoint: samplePoint,
		IsSection:   true,
		SignVector:  make(map[string]int),
	}
	for _, p := range polys {
		if p == nil {
			continue
		}
		if root.poly == p {
			cell.SignVector[polyKey(p)] = 0
		} else {
			// Evaluate at mid of isolating interval
			mid := new(big.Rat).Add(root.low, root.high)
			mid.Quo(mid, big.NewRat(2, 1))
			if val, err := evalPolyAtRat(p, mid); err == nil {
				cell.SignVector[polyKey(p)] = val.Sign()
			}
		}
	}
	return cell
}

func reconstruct1DIntervalsFromCells(cells []CadCell, roots []cad1DRoot, p *univariatePoly, op string) (Node, error) {
	pKey := polyKey(p)

	allSatisfied := true
	noneSatisfied := true

	for i := range cells {
		sign := cells[i].SignVector[pKey]
		satisfied := evalRelationalSign(sign, op)
		cells[i].Satisfied = satisfied
		if satisfied {
			noneSatisfied = false
		} else {
			allSatisfied = false
		}
	}

	if allSatisfied {
		return &VarNode{Name: "true"}, nil
	}
	if noneSatisfied {
		return &VarNode{Name: "false"}, nil
	}

	negInf := &VarNode{Name: "-inf"}
	posInf := &VarNode{Name: "inf"}

	var satisfyingIntervals []Node

	m := len(roots)
	i := 0
	for i < len(cells) {
		if !cells[i].Satisfied {
			i++
			continue
		}

		start := i
		for i < len(cells) && cells[i].Satisfied {
			i++
		}
		end := i - 1

		// Determine left endpoint
		var leftNode Node
		if start == 0 {
			leftNode = negInf
		} else if start%2 == 1 {
			// Section (start - 1)/2
			rootIdx := (start - 1) / 2
			leftNode = roots[rootIdx].node
		} else {
			// Sector start/2 - 1 -> starts from rootIdx = start/2 - 1
			rootIdx := start/2 - 1
			leftNode = roots[rootIdx].node
		}

		// Determine right endpoint
		var rightNode Node
		if end == 2*m {
			rightNode = posInf
		} else if end%2 == 1 {
			// Section (end - 1)/2
			rootIdx := (end - 1) / 2
			rightNode = roots[rootIdx].node
		} else {
			// Sector end/2 -> ends at rootIdx = end/2
			rootIdx := end / 2
			rightNode = roots[rootIdx].node
		}

		satisfyingIntervals = append(satisfyingIntervals, &ListNode{Elements: []Node{leftNode, rightNode}})
	}

	if len(satisfyingIntervals) == 0 {
		return &VarNode{Name: "false"}, nil
	}
	return &ListNode{Elements: satisfyingIntervals}, nil
}

// solveMultivariateCAD executes Brown-McCallum projection and lifting for multi-variable formulas.
func solveMultivariateCAD(expr Node, op string, vars []string, env *Env, fsm *CadLifecycleFSM) (Node, error) {
	cells, err := CADDecomposeCells([]Node{expr}, vars, env)
	if err != nil {
		return nil, err
	}

	isSatisfied := false
	var witness []Node

	for i := range cells {
		signVal := cells[i].SignVector[expr.String()]
		if evalRelationalSign(signVal, op) {
			cells[i].Satisfied = true
			isSatisfied = true
			if len(witness) == 0 {
				witness = cells[i].SamplePoint
			}
		}
	}

	_ = fsm.TransitionTo(CadStateDecided)

	if isSatisfied {
		return &ListNode{Elements: witness}, nil
	}
	return &VarNode{Name: "false"}, nil
}

// specializePolyForCAD substitutes sample coordinates into polynomial p,
// simplifies it, and extracts a univariate polynomial in targetVar.
// If the polynomial vanishes identically (isZero), a nullification error is returned.
func specializePolyForCAD(p Node, vars []string, samplePoint []Node, targetVar string, env *Env) (*univariatePoly, bool, error) {
	curr := p
	for i := 0; i < len(samplePoint); i++ {
		curr = Substitute(curr, vars[i], samplePoint[i])
	}
	simplified, err := EvalWithEnv(curr, env)
	if err == nil {
		curr = simplified
	}

	// Nullification Check:
	if isZero(curr) {
		return nil, false, fmt.Errorf("nullification detected: %s vanished identically over cell with sample %v", p.String(), samplePoint)
	}

	uPoly, ok := extractPoly(curr, targetVar)
	if !ok {
		return nil, false, fmt.Errorf("failed to extract univariate polynomial in %s from %s", targetVar, curr.String())
	}
	uPoly = trimPoly(uPoly)

	if uPoly.degree() <= 0 {
		return uPoly, true, nil // constant
	}
	return uPoly, false, nil
}

// computeBrownMcCallumProjection constructs projection factor sets P_1, P_2, ..., P_n
// where P_k contains polynomials in vars[0..k-1].
// vars is ordered [x1, x2, ..., xn].
// Elimination proceeds in reverse: xn, x_{n-1}, ..., x2.
// Returned slice projSets has length len(vars), where projSets[k-1] corresponds to Level k (in vars[0..k-1]).
func computeBrownMcCallumProjection(polys []Node, vars []string, env *Env) ([][]Node, error) {
	n := len(vars)
	projSets := make([][]Node, n)
	projSets[n-1] = polys

	currentSet := polys
	for k := n - 1; k >= 1; k-- {
		elimVar := vars[k]
		var nextSet []Node
		seen := make(map[string]bool)

		for i, p := range currentSet {
			polyObj, ok := extractPoly(p, elimVar)
			if !ok || polyObj.degree() <= 0 {
				if !isConstantNode(p) {
					sP := p.String()
					if !seen[sP] {
						seen[sP] = true
						nextSet = append(nextSet, p)
					}
				}
				continue
			}

			// 1. Leading coefficient lc(p)
			lcNode := polyObj.leadCoeff()
			if !isConstantNode(lcNode) {
				sLC := lcNode.String()
				if !seen[sLC] {
					seen[sLC] = true
					nextSet = append(nextSet, lcNode)
				}
			}

			// 2. Discriminant via resultant with derivative
			if polyObj.degree() > 1 {
				dp, err := differentiate(p, elimVar)
				if err == nil {
					res, err := EvalResultant(p, dp, elimVar, env)
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
				res, err := EvalResultant(p, q, elimVar, env)
				if err == nil && !isConstantNode(res) {
					if isZero(res) {
						g, err := EvalPolyGCD(p, q, elimVar, env)
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
		projSets[k-1] = nextSet
		currentSet = nextSet
	}

	return projSets, nil
}

func isConstantNode(n Node) bool {
	if _, ok := n.(*RationalNode); ok {
		return true
	}
	return len(ExtractFreeVariables(n)) == 0
}

// liftCADCells recursively lifts cells from R^1 up to R^n over projection factor sets.
func liftCADCells(projSets [][]Node, vars []string, env *Env, fsm *CadLifecycleFSM) ([]CadCell, error) {
	n := len(vars)
	if n == 0 {
		return nil, fmt.Errorf("no variables specified")
	}

	// 1. Base decomposition at Level 1 (vars[0])
	baseVar := vars[0]
	basePolys := projSets[0]
	var nonConstBase []*univariatePoly
	for _, bp := range basePolys {
		if uPoly, ok := extractPoly(bp, baseVar); ok {
			uPoly = trimPoly(uPoly)
			if uPoly.degree() > 0 {
				nonConstBase = append(nonConstBase, uPoly)
			}
		}
	}

	var currentLevelCells []*CadCell
	if len(nonConstBase) > 0 {
		base1DCells, _, err := decompose1DCADInternal(nonConstBase, baseVar, env)
		if err != nil {
			return nil, err
		}
		for _, bc := range base1DCells {
			c := bc
			currentLevelCells = append(currentLevelCells, &c)
		}
	} else {
		// Entire R^1 is a single sector
		currentLevelCells = append(currentLevelCells, &CadCell{
			Dimension:   1,
			SamplePoint: []Node{mustRational(0, 1)},
			IsSection:   false,
			SignVector:  make(map[string]int),
		})
	}

	_ = fsm.TransitionTo(CadStateSampled)

	// 2. Recursive Lifting for levels 2..n
	for level := 2; level <= n; level++ {
		targetVar := vars[level-1]
		levelPolys := projSets[level-1]
		var nextLevelCells []*CadCell

		for _, parent := range currentLevelCells {
			var specializedPolys []*univariatePoly
			hasNullification := false

			for _, p := range levelPolys {
				uPoly, isConst, err := specializePolyForCAD(p, vars[:level-1], parent.SamplePoint, targetVar, env)
				if err != nil {
					if strings.Contains(err.Error(), "nullification") {
						hasNullification = true
						break
					}
					return nil, err
				}
				if !isConst && uPoly.degree() > 0 {
					specializedPolys = append(specializedPolys, uPoly)
				}
			}

			if hasNullification {
				_ = fsm.TransitionTo(CadStateUnsupported)
				return nil, fmt.Errorf("CAD lifting failed due to nullification on cell %v", parent.SamplePoint)
			}

			if len(specializedPolys) == 0 {
				// No roots in this cylinder: entire line is a single sector over parent
				liftedSample := make([]Node, len(parent.SamplePoint)+1)
				copy(liftedSample, parent.SamplePoint)
				liftedSample[len(parent.SamplePoint)] = mustRational(0, 1)

				nextLevelCells = append(nextLevelCells, &CadCell{
					Dimension:   parent.Dimension + 1,
					SamplePoint: liftedSample,
					IsSection:   false,
					SignVector:  make(map[string]int),
					Parent:      parent,
				})
			} else {
				stack1D, _, err := decompose1DCADInternal(specializedPolys, targetVar, env)
				if err != nil {
					return nil, err
				}
				for _, c1D := range stack1D {
					liftedSample := make([]Node, len(parent.SamplePoint)+len(c1D.SamplePoint))
					copy(liftedSample, parent.SamplePoint)
					copy(liftedSample[len(parent.SamplePoint):], c1D.SamplePoint)

					dim := parent.Dimension
					if !c1D.IsSection {
						dim++
					}

					nextLevelCells = append(nextLevelCells, &CadCell{
						Dimension:   dim,
						SamplePoint: liftedSample,
						IsSection:   c1D.IsSection,
						SignVector:  make(map[string]int),
						Parent:      parent,
					})
				}
			}
		}

		currentLevelCells = nextLevelCells
	}

	_ = fsm.TransitionTo(CadStateLifted)

	// 3. Evaluate SignVector for original polynomials on all cells
	originalPolys := projSets[n-1]
	for _, cell := range currentLevelCells {
		cell.SignVector = make(map[string]int)
		for _, p := range originalPolys {
			curr := p
			for i, v := range vars {
				if i < len(cell.SamplePoint) {
					curr = Substitute(curr, v, cell.SamplePoint[i])
				}
			}
			val, err := EvalWithEnv(curr, env)
			if err != nil {
				val = curr
			}
			if r, ok := val.(*RationalNode); ok {
				cell.SignVector[p.String()] = r.Val.Sign()
			} else {
				sign := exactSignEval(val, 0)
				cell.SignVector[p.String()] = int(sign)
			}
		}
	}

	res := make([]CadCell, len(currentLevelCells))
	for i, c := range currentLevelCells {
		res[i] = *c
	}
	return res, nil
}

// CADDecomposeCells decomposes R^n into sign-invariant CAD cells for the given polynomials.
func CADDecomposeCells(polys []Node, vars []string, env *Env) ([]CadCell, error) {
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

	cells, err := liftCADCells(projSets, vars, env, fsm)
	if err != nil {
		return nil, err
	}

	_ = fsm.TransitionTo(CadStateDecided)
	return cells, nil
}

// CADDecompose performs full Cylindrical Algebraic Decomposition and returns cell sample points.
func CADDecompose(polys []Node, vars []string, env *Env) (*ListNode, error) {
	cells, err := CADDecomposeCells(polys, vars, env)
	if err != nil {
		return nil, err
	}

	var sampleNodes []Node
	for _, cell := range cells {
		sampleNodes = append(sampleNodes, &ListNode{Elements: cell.SamplePoint})
	}
	return &ListNode{Elements: sampleNodes}, nil
}
