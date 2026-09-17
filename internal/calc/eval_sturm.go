package calc

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Exact Real Root Isolation & Sturm's Theorem Pack
// (Sturm 1829, Collins 1967, Akritas 1993)
// -------------------------------------------------------------------------

// EvalSturm generates the exact Sturm sequence (chain) for polynomial p with respect to varName.
func EvalSturm(p Node, varName string, env *Env) (Node, error) {
	if p == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "sturm"))
	}

	evalP, err := Eval(expandNode(p))
	if err != nil {
		evalP = expandNode(p)
	}

	v := varName
	if v == "" {
		vars := collectVariables(evalP)
		if len(vars) == 0 {
			// Constant polynomial
			return &ListNode{Elements: []Node{evalP}}, nil
		}
		if len(vars) > 1 {
			return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "sturm"))
		}
		v = vars[0]
	}

	polyP, ok := extractPoly(evalP, v)
	if !ok || !isRationalPoly(polyP) {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "sturm"))
	}

	chain, err := buildPrimitiveSturmSequence(polyP)
	if err != nil {
		return nil, err
	}

	elements := make([]Node, len(chain))
	for i, poly := range chain {
		node := poly.toNode()
		evaled, err := Eval(node)
		if err != nil {
			evaled = node
		}
		elements[i] = evaled
	}

	return &ListNode{Elements: elements}, nil
}

// EvalRootCount counts the exact number of distinct real roots of p in interval [a, b].
// If a is nil or -inf, lower bound is -infinity.
// If b is nil or +inf, upper bound is +infinity.
func EvalRootCount(p Node, varName string, a, b Node, env *Env) (Node, error) {
	if p == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "root_count"))
	}

	evalP, err := Eval(expandNode(p))
	if err != nil {
		evalP = expandNode(p)
	}

	v := varName
	if v == "" {
		vars := collectVariables(evalP)
		if len(vars) == 0 {
			// Non-zero constant has 0 roots; zero polynomial has infinite roots (error/0)
			if isZero(evalP) {
				return &VarNode{Name: "inf"}, nil
			}
			return mustRational(0, 1), nil
		}
		if len(vars) > 1 {
			return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "root_count"))
		}
		v = vars[0]
	}

	polyP, ok := extractPoly(evalP, v)
	if !ok || !isRationalPoly(polyP) {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "root_count"))
	}

	if polyP.degree() == 0 {
		if isPolyZero(polyP) {
			return &VarNode{Name: "inf"}, nil
		}
		return mustRational(0, 1), nil
	}

	chain, err := buildPrimitiveSturmSequence(polyP)
	if err != nil {
		return nil, err
	}

	// Calculate variations at lower bound
	var vA int
	isInfA, signInfA, ratA, err := parseIntervalEndpoint(a, -1, env)
	if err != nil {
		return nil, err
	}
	if isInfA {
		if signInfA > 0 {
			// lower bound is +inf: impossible for a < b, but mathematically V(+inf)
			vA = countSignVariationsAtInfinity(chain, true)
		} else {
			vA = countSignVariationsAtInfinity(chain, false)
		}
	} else {
		vA, err = countSignVariationsAtPoint(chain, ratA)
		if err != nil {
			return nil, err
		}
	}

	// Calculate variations at upper bound
	var vB int
	isInfB, signInfB, ratB, err := parseIntervalEndpoint(b, 1, env)
	if err != nil {
		return nil, err
	}
	if isInfB {
		if signInfB < 0 {
			vB = countSignVariationsAtInfinity(chain, false)
		} else {
			vB = countSignVariationsAtInfinity(chain, true)
		}
	} else {
		vB, err = countSignVariationsAtPoint(chain, ratB)
		if err != nil {
			return nil, err
		}
	}

	// Validate order if both are finite
	if !isInfA && !isInfB && ratA.Cmp(ratB) > 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_invalid_interval", "root_count", ratA.RatString(), ratB.RatString()))
	}

	diff := vA - vB
	if diff < 0 {
		diff = 0
	}

	// Sturm's theorem counts roots in (a, b].
	// If a is finite and is a root, add 1 so interval is [a, b].
	if !isInfA {
		valA, err := evalPolyAtRat(polyP, ratA)
		if err == nil && valA.Sign() == 0 {
			diff++
		}
	}

	return mustRational(int64(diff), 1), nil
}

// EvalIsolateRoots isolates all distinct real roots of p into disjoint rational intervals [l, r].
func EvalIsolateRoots(p Node, varName string, a, b Node, env *Env) (Node, error) {
	if p == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "isolate_roots"))
	}

	evalP, err := Eval(expandNode(p))
	if err != nil {
		evalP = expandNode(p)
	}

	v := varName
	if v == "" {
		vars := collectVariables(evalP)
		if len(vars) == 0 {
			return &ListNode{Elements: []Node{}}, nil
		}
		if len(vars) > 1 {
			return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "isolate_roots"))
		}
		v = vars[0]
	}

	polyP, ok := extractPoly(evalP, v)
	if !ok || !isRationalPoly(polyP) {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "isolate_roots"))
	}

	if polyP.degree() == 0 {
		return &ListNode{Elements: []Node{}}, nil
	}

	// 1. Square-free reduction to avoid endpoint complications: P_sqfr = P / gcd(P, P')
	dPoly := polyDerivative1D(polyP)
	gcdNode, err := EvalPolyGCD(polyP.toNode(), dPoly.toNode(), v, env)
	if err == nil && gcdNode != nil {
		gPoly, okG := extractPoly(gcdNode, v)
		if okG && gPoly.degree() > 0 {
			quot, _, okDiv := polyDivide(polyP, gPoly)
			if okDiv && !isPolyZero(quot) {
				polyP = quot
			}
		}
	}

	// 2. Build Sturm chain for square-free polynomial
	chain, err := buildPrimitiveSturmSequence(polyP)
	if err != nil {
		return nil, err
	}

	// 3. Determine initial search interval [lower, upper]
	isInfA, _, ratA, err := parseIntervalEndpoint(a, -1, env)
	if err != nil {
		return nil, err
	}
	isInfB, _, ratB, err := parseIntervalEndpoint(b, 1, env)
	if err != nil {
		return nil, err
	}

	var initL, initR *big.Rat
	cauchyBound := computeCauchyBound(polyP)

	if isInfA || ratA == nil {
		initL = new(big.Rat).Neg(cauchyBound)
	} else {
		initL = new(big.Rat).Set(ratA)
	}

	if isInfB || ratB == nil {
		initR = new(big.Rat).Set(cauchyBound)
	} else {
		initR = new(big.Rat).Set(ratB)
	}

	if initL.Cmp(initR) > 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_invalid_interval", "isolate_roots", initL.RatString(), initR.RatString()))
	}

	// 4. Bisection Isolation Queue
	type interval struct {
		l *big.Rat
		r *big.Rat
	}

	// Helper to count roots in open interval (l, r)
	countRootsInOpen := func(l, r *big.Rat) int {
		vl, err1 := countSignVariationsAtPoint(chain, l)
		vr, err2 := countSignVariationsAtPoint(chain, r)
		if err1 != nil || err2 != nil {
			return 0
		}
		diff := vl - vr
		if diff < 0 {
			return 0
		}
		return diff
	}

	var isolated []interval
	queue := []interval{{l: initL, r: initR}}

	// Safety limit on iterations to prevent runaway loops
	maxIterations := 2000
	iterCount := 0

	type stepResult struct {
		newIsolated []interval
		newQueue    []interval
	}

	stepOne := func(curr interval) (stepResult, error) {
		var res stepResult
		l := curr.l
		r := curr.r

		// Check if endpoints are roots
		valL, _ := evalPolyAtRat(polyP, l)
		if valL.Sign() == 0 {
			res.newIsolated = append(res.newIsolated, interval{l: new(big.Rat).Set(l), r: new(big.Rat).Set(l)})
			// Shift l slightly right: l' = l + (r - l) / 8
			delta := new(big.Rat).Sub(r, l)
			delta.Quo(delta, big.NewRat(8, 1))
			l = new(big.Rat).Add(l, delta)
		}

		valR, _ := evalPolyAtRat(polyP, r)
		if valR.Sign() == 0 {
			res.newIsolated = append(res.newIsolated, interval{l: new(big.Rat).Set(r), r: new(big.Rat).Set(r)})
			// Shift r slightly left: r' = r - (r - l) / 8
			delta := new(big.Rat).Sub(r, l)
			delta.Quo(delta, big.NewRat(8, 1))
			r = new(big.Rat).Sub(r, delta)
		}

		if l.Cmp(r) >= 0 {
			return res, nil
		}

		k := countRootsInOpen(l, r)
		if k == 0 {
			return res, nil
		}
		if k == 1 {
			width := new(big.Rat).Sub(r, l)
			if width.Cmp(big.NewRat(1, 1)) <= 0 {
				res.newIsolated = append(res.newIsolated, interval{l: l, r: r})
				return res, nil
			}
		}

		// k > 1: Bisect
		mid := new(big.Rat).Add(l, r)
		mid.Quo(mid, big.NewRat(2, 1))

		valMid, _ := evalPolyAtRat(polyP, mid)
		if valMid.Sign() == 0 {
			// Exact rational root at midpoint!
			res.newIsolated = append(res.newIsolated, interval{l: new(big.Rat).Set(mid), r: new(big.Rat).Set(mid)})
			// Shift mid left and right for remaining intervals
			delta := new(big.Rat).Sub(r, l)
			delta.Quo(delta, big.NewRat(8, 1))
			midLeft := new(big.Rat).Sub(mid, delta)
			midRight := new(big.Rat).Add(mid, delta)
			if l.Cmp(midLeft) < 0 {
				res.newQueue = append(res.newQueue, interval{l: l, r: midLeft})
			}
			if midRight.Cmp(r) < 0 {
				res.newQueue = append(res.newQueue, interval{l: midRight, r: r})
			}
		} else {
			res.newQueue = append(res.newQueue, interval{l: l, r: mid}, interval{l: mid, r: r})
		}
		return res, nil
	}

	for len(queue) > 0 && iterCount < maxIterations {
		iterCount += len(queue)
		batchResults, _ := ParallelBatchMap(queue, stepOne, 4)
		queue = nil
		for _, b := range batchResults {
			isolated = append(isolated, b.newIsolated...)
			queue = append(queue, b.newQueue...)
		}
	}

	// Sort isolated intervals by lower bound
	sort.Slice(isolated, func(i, j int) bool {
		cmp := isolated[i].l.Cmp(isolated[j].l)
		if cmp != 0 {
			return cmp < 0
		}
		return isolated[i].r.Cmp(isolated[j].r) < 0
	})

	// Deduplicate any identical intervals (e.g. from exact roots)
	var dedup []interval
	for _, iv := range isolated {
		if len(dedup) == 0 {
			dedup = append(dedup, iv)
			continue
		}
		last := dedup[len(dedup)-1]
		if last.l.Cmp(iv.l) == 0 && last.r.Cmp(iv.r) == 0 {
			continue
		}
		dedup = append(dedup, iv)
	}

	// Format result as a list of 2-element lists [[l1, r1], [l2, r2], ...]
	resultElements := make([]Node, len(dedup))
	for i, iv := range dedup {
		resultElements[i] = &ListNode{
			Elements: []Node{
				&RationalNode{Val: iv.l},
				&RationalNode{Val: iv.r},
			},
		}
	}

	return &ListNode{Elements: resultElements}, nil
}

// -------------------------------------------------------------------------
// Internal Algebraic Utilities for Sturm Sequences
// -------------------------------------------------------------------------

// buildPrimitiveSturmSequence constructs the Sturm chain using primitive PRS over Q[x].
// Sign preservation: dividing by positive content preserves algebraic sign at all points.
func buildPrimitiveSturmSequence(p *univariatePoly) ([]*univariatePoly, error) {
	if isPolyZero(p) {
		return nil, fmt.Errorf("sturm: zero polynomial")
	}

	// f_0 = p
	f0 := trimPoly(p)
	// f_1 = p'
	f1 := polyDerivative1D(f0)
	if isPolyZero(f1) {
		// Degree 0 or constant
		return []*univariatePoly{f0}, nil
	}

	chain := []*univariatePoly{f0, f1}

	u := f0
	w := f1

	for {
		_, rem, ok := polyDivide(u, w)
		if !ok {
			return nil, fmt.Errorf("sturm: polynomial division failed")
		}
		if isPolyZero(rem) {
			break
		}

		// Next term is -rem
		negRemCoeffs := make([]Node, len(rem.coeffs))
		for i, c := range rem.coeffs {
			negC, _ := simplifyUnaryOp("-", c)
			negVal, _ := Eval(negC)
			negRemCoeffs[i] = negVal
		}
		nextPoly := trimPoly(&univariatePoly{varName: p.varName, coeffs: negRemCoeffs})

		// Primitive reduction: divide by positive rational content to avoid coefficient explosion
		cont, err := polyContent(nextPoly)
		if err == nil && cont != nil {
			if rat, ok := cont.(*RationalNode); ok && rat.Val.Sign() > 0 {
				prim, err := polyExactDivideScalar(nextPoly, rat)
				if err == nil {
					nextPoly = prim
				}
			}
		}

		chain = append(chain, nextPoly)

		if nextPoly.degree() == 0 {
			break
		}

		u = w
		w = nextPoly
	}

	return chain, nil
}

// polyDerivative1D calculates formal derivative of 1D univariate polynomial.
func polyDerivative1D(p *univariatePoly) *univariatePoly {
	if p.degree() <= 0 {
		return &univariatePoly{varName: p.varName, coeffs: []Node{mustRational(0, 1)}}
	}
	deg := p.degree()
	newCoeffs := make([]Node, deg)
	for i := 1; i <= deg; i++ {
		c := p.coeffs[i]
		scaleNode := mustRational(int64(i), 1)
		prod, _ := simplifyMul([]Node{c, scaleNode})
		evaled, err := Eval(prod)
		if err != nil {
			evaled = prod
		}
		newCoeffs[i-1] = evaled
	}
	return trimPoly(&univariatePoly{varName: p.varName, coeffs: newCoeffs})
}

// evalPolyAtRat evaluates univariate polynomial with rational coefficients at rational point x via Horner's method.
func evalPolyAtRat(p *univariatePoly, x *big.Rat) (*big.Rat, error) {
	if isPolyZero(p) {
		return new(big.Rat), nil
	}
	deg := p.degree()
	leadNode := p.coeffs[deg]
	leadRat, ok := leadNode.(*RationalNode)
	if !ok {
		return nil, fmt.Errorf("evalPolyAtRat: non-rational coefficient: %v", leadNode)
	}

	res := new(big.Rat).Set(leadRat.Val)
	for i := deg - 1; i >= 0; i-- {
		cNode := p.coeffs[i]
		cRat, ok := cNode.(*RationalNode)
		if !ok {
			return nil, fmt.Errorf("evalPolyAtRat: non-rational coefficient: %v", cNode)
		}
		res.Mul(res, x)
		res.Add(res, cRat.Val)
	}
	return res, nil
}

// countSignVariationsAtPoint calculates sign variations V(x) of the Sturm chain at finite point x.
// Standard Sturm definition skips zeros.
func countSignVariationsAtPoint(chain []*univariatePoly, x *big.Rat) (int, error) {
	prevSign := 0
	variations := 0

	for _, poly := range chain {
		val, err := evalPolyAtRat(poly, x)
		if err != nil {
			return 0, err
		}
		s := val.Sign()
		if s == 0 {
			continue
		}
		if prevSign != 0 && s != prevSign {
			variations++
		}
		prevSign = s
	}

	return variations, nil
}

// countSignVariationsAtInfinity calculates sign variations V(+inf) or V(-inf) of the Sturm chain.
func countSignVariationsAtInfinity(chain []*univariatePoly, positiveInf bool) int {
	prevSign := 0
	variations := 0

	for _, poly := range chain {
		if isPolyZero(poly) {
			continue
		}
		lead := poly.leadCoeff()
		leadRat, ok := lead.(*RationalNode)
		if !ok {
			continue
		}
		sign := leadRat.Val.Sign()
		if sign == 0 {
			continue
		}

		if !positiveInf {
			// At -inf, sign is sign(lead) * (-1)^degree
			if poly.degree()%2 != 0 {
				sign = -sign
			}
		}

		if prevSign != 0 && sign != prevSign {
			variations++
		}
		prevSign = sign
	}

	return variations
}

// computeCauchyBound computes Cauchy's root bound M = 1 + max |a_i / a_n| for all real/complex roots.
func computeCauchyBound(p *univariatePoly) *big.Rat {
	deg := p.degree()
	if deg <= 0 {
		return big.NewRat(1, 1)
	}
	leadRat, ok := p.leadCoeff().(*RationalNode)
	if !ok || leadRat.Val.Sign() == 0 {
		return big.NewRat(1, 1)
	}
	leadAbs := new(big.Rat).Abs(leadRat.Val)

	maxCoeff := new(big.Rat)
	for i := 0; i < deg; i++ {
		cRat, ok := p.coeffs[i].(*RationalNode)
		if !ok {
			continue
		}
		ratio := new(big.Rat).Abs(cRat.Val)
		ratio.Quo(ratio, leadAbs)
		if ratio.Cmp(maxCoeff) > 0 {
			maxCoeff.Set(ratio)
		}
	}

	// Bound M = 1 + maxCoeff
	bound := new(big.Rat).Add(big.NewRat(1, 1), maxCoeff)
	// Ceiling to next integer for clean integer intervals
	num := bound.Num()
	denom := bound.Denom()
	ceilInt := new(big.Int).Div(num, denom)
	if new(big.Int).Rem(num, denom).Sign() > 0 {
		ceilInt.Add(ceilInt, big.NewInt(1))
	}
	return new(big.Rat).SetInt(ceilInt)
}

// isRationalPoly returns true if all coefficients of p are RationalNodes.
func isRationalPoly(p *univariatePoly) bool {
	if p == nil {
		return false
	}
	for _, c := range p.coeffs {
		if _, ok := c.(*RationalNode); !ok {
			return false
		}
	}
	return true
}

// parseIntervalEndpoint parses a node into infinity or a finite rational number.
func parseIntervalEndpoint(n Node, defaultInfSign int, env *Env) (isInf bool, signInf int, val *big.Rat, err error) {
	if n == nil {
		return true, defaultInfSign, nil, nil
	}

	evaled, err := EvalWithEnv(n, env)
	if err != nil {
		evaled = n
	}

	// Check for inf / -inf
	if v, ok := evaled.(*VarNode); ok {
		if v.Name == "inf" || v.Name == "Infinity" {
			return true, 1, nil, nil
		}
	}
	if u, ok := evaled.(*UnaryOpNode); ok && u.Op == "-" {
		if v, ok := u.Expr.(*VarNode); ok {
			if v.Name == "inf" || v.Name == "Infinity" {
				return true, -1, nil, nil
			}
		}
	}

	// Finite rational
	if r, ok := evaled.(*RationalNode); ok {
		return false, 0, new(big.Rat).Set(r.Val), nil
	}

	return false, 0, nil, fmt.Errorf("interval endpoint must be a rational number or infinity, got %v", evaled)
}
