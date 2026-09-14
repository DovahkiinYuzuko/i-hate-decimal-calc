package calc

import (
	"fmt"
	"math/big"
	"sort"
)

// -------------------------------------------------------------------------
// Multivariate Polynomial GCD & LCM via Subresultant PRS
// (Collins 1967, Brown 1971, Knuth TAOCP Vol. 2 Section 4.6.1 Algorithm C)
// -------------------------------------------------------------------------

// EvalPolyGCD computes the greatest common divisor of polynomials p and q with respect to varName.
// If varName is empty, the main variable is automatically selected.
func EvalPolyGCD(p, q Node, varName string, env *Env) (Node, error) {
	if p == nil || q == nil {
		return nil, fmt.Errorf("poly_gcd: nil argument")
	}

	evalP, err := Eval(expandNode(p))
	if err != nil {
		evalP = expandNode(p)
	}
	evalQ, err := Eval(expandNode(q))
	if err != nil {
		evalQ = expandNode(q)
	}

	if isZero(evalP) && isZero(evalQ) {
		return mustRational(0, 1), nil
	}
	if isZero(evalP) {
		return normalizePolySign(evalQ), nil
	}
	if isZero(evalQ) {
		return normalizePolySign(evalP), nil
	}

	// Determine variable
	v := varName
	if v == "" {
		varsP := collectVariables(evalP)
		varsQ := collectVariables(evalQ)
		allVars := unionStrings(varsP, varsQ)
		if len(allVars) == 0 {
			// Scalar GCD
			return scalarGCD(evalP, evalQ)
		}
		v = selectMainVariable(allVars)
	}

	return subresultantGCD(evalP, evalQ, v)
}

// EvalPolyLCM computes the least common multiple of polynomials p and q with respect to varName:
// LCM(p, q) = (p * q) / GCD(p, q)
func EvalPolyLCM(p, q Node, varName string, env *Env) (Node, error) {
	if p == nil || q == nil {
		return nil, fmt.Errorf("poly_lcm: nil argument")
	}

	evalP, err := Eval(expandNode(p))
	if err != nil {
		evalP = expandNode(p)
	}
	evalQ, err := Eval(expandNode(q))
	if err != nil {
		evalQ = expandNode(q)
	}

	if isZero(evalP) || isZero(evalQ) {
		return mustRational(0, 1), nil
	}

	gcdNode, err := EvalPolyGCD(evalP, evalQ, varName, env)
	if err != nil {
		return nil, err
	}

	v := varName
	if v == "" {
		varsP := collectVariables(evalP)
		varsQ := collectVariables(evalQ)
		allVars := unionStrings(varsP, varsQ)
		if len(allVars) == 0 {
			// Scalar LCM = (a * b) / gcd(a, b)
			prod, err := simplifyMul([]Node{evalP, evalQ})
			if err != nil {
				return nil, err
			}
			invG, err := simplifyPow(gcdNode, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			lcmNode, err := simplifyMul([]Node{prod, invG})
			if err != nil {
				return nil, err
			}
			return Eval(lcmNode)
		}
		v = selectMainVariable(allVars)
	}

	polyP, okP := extractPoly(evalP, v)
	polyG, okG := extractPoly(gcdNode, v)
	polyQ, okQ := extractPoly(evalQ, v)

	if !okP || !okG || !okQ {
		// Fallback to algebraic mul and division
		prod, err := simplifyMul([]Node{evalP, evalQ})
		if err != nil {
			return nil, err
		}
		invG, err := simplifyPow(gcdNode, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		res, err := simplifyMul([]Node{prod, invG})
		if err != nil {
			return nil, err
		}
		return Eval(expandNode(res))
	}

	quot, rem, ok := polyDivide(polyP, polyG)
	if !ok || !isPolyZero(rem) {
		// Fallback
		prod, _ := simplifyMul([]Node{evalP, evalQ})
		invG, _ := simplifyPow(gcdNode, mustRational(-1, 1))
		res, _ := simplifyMul([]Node{prod, invG})
		return Eval(expandNode(res))
	}

	lcmPoly := polyMul(quot, polyQ)
	return normalizePolySign(lcmPoly.toNode()), nil
}

// subresultantGCD performs Collins (1967) / Brown (1971) / Knuth Algorithm C.
func subresultantGCD(p, q Node, v string) (Node, error) {
	polyP, okP := extractPoly(p, v)
	polyQ, okQ := extractPoly(q, v)

	if !okP || !okQ {
		return nil, fmt.Errorf("poly_gcd: failed to extract polynomials in %s", v)
	}

	if isPolyZero(polyP) && isPolyZero(polyQ) {
		return mustRational(0, 1), nil
	}
	if isPolyZero(polyP) {
		return normalizePolySign(polyQ.toNode()), nil
	}
	if isPolyZero(polyQ) {
		return normalizePolySign(polyP.toNode()), nil
	}

	// 1. Content and Primitive Part separation
	contP, err := polyContent(polyP, v)
	if err != nil {
		return nil, err
	}
	contQ, err := polyContent(polyQ, v)
	if err != nil {
		return nil, err
	}

	commonCont, err := EvalPolyGCD(contP, contQ, "", nil)
	if err != nil {
		return nil, err
	}

	ppP, err := polyExactDivideScalar(polyP, contP)
	if err != nil {
		ppP = polyP
	}
	ppQ, err := polyExactDivideScalar(polyQ, contQ)
	if err != nil {
		ppQ = polyQ
	}

	u := ppP
	w := ppQ
	if u.degree() < w.degree() {
		u, w = w, u
	}

	// 2. Subresultant PRS Loop (Knuth TAOCP Vol. 2 Section 4.6.1 Algorithm C)
	var g Node = mustRational(1, 1)
	var h Node = mustRational(1, 1)
	lastNonZero := w

	for !isPolyZero(w) {
		d := u.degree() - w.degree()
		r, ok := pseudoRemainder(u, w)
		if !ok {
			return nil, fmt.Errorf("poly_gcd: pseudo-remainder failed in %s", v)
		}

		if isPolyZero(r) {
			lastNonZero = w
			break
		}

		if r.degree() == 0 {
			// Remainder is degree 0 (scalar / constant with respect to v)
			// Thus gcd primitive part is 1
			return normalizePolySign(commonCont), nil
		}

		u = w

		// Divisor = g * h^d
		hPowD, err := simplifyPow(h, mustRational(int64(d), 1))
		if err != nil {
			return nil, err
		}
		divisor, err := simplifyMul([]Node{g, hPowD})
		if err != nil {
			return nil, err
		}
		divisorVal, err := Eval(divisor)
		if err != nil {
			divisorVal = divisor
		}

		wNext, err := polyExactDivideScalar(r, divisorVal)
		if err != nil {
			return nil, fmt.Errorf("poly_gcd: subresultant reduction failed: %w", err)
		}
		if !isPolyZero(wNext) {
			lastNonZero = wNext
		}
		w = wNext

		g = u.leadCoeff()

		// h = h^(1 - d) * g^d
		if d == 1 {
			h = g
		} else if d > 1 {
			// h = (g^d) / (h^(d - 1))
			gPowD, _ := simplifyPow(g, mustRational(int64(d), 1))
			hPowPrev, _ := simplifyPow(h, mustRational(int64(d-1), 1))
			invHPow, _ := simplifyPow(hPowPrev, mustRational(-1, 1))
			newH, _ := simplifyMul([]Node{gPowD, invHPow})
			h, _ = Eval(newH)
		}
		// If d == 0, h stays unchanged
	}

	// Final primitive part is pp(lastNonZero)
	ppWCont, err := polyContent(lastNonZero, v)
	if err == nil && !isZero(ppWCont) && !isOne(ppWCont) {
		lastNonZero, _ = polyExactDivideScalar(lastNonZero, ppWCont)
	}

	resPoly := polyScale(lastNonZero, commonCont)
	return normalizePolySign(resPoly.toNode()), nil
}

// pseudoRemainder calculates the pseudo-remainder r = prem(u, v) such that:
// lc(v)^(deg(u) - deg(v) + 1) * u = q * v + r
func pseudoRemainder(u, v *univariatePoly) (*univariatePoly, bool) {
	if isPolyZero(v) {
		return nil, false
	}
	degU := u.degree()
	degV := v.degree()
	if degU < degV {
		return u, true
	}

	delta := degU - degV
	leadV := v.leadCoeff()

	// r = u
	rCoeffs := make([]Node, len(u.coeffs))
	copy(rCoeffs, u.coeffs)
	r := &univariatePoly{varName: u.varName, coeffs: rCoeffs}

	for k := degU; k >= degV; k-- {
		if r.degree() == k {
			leadR := r.leadCoeff()
			degDiff := k - degV

			// r = leadV * r - leadR * x^degDiff * v
			scaleR := polyScale(r, leadV)

			// construct term = leadR * x^degDiff * v
			shiftedVCoeffs := make([]Node, len(v.coeffs)+degDiff)
			for i := range shiftedVCoeffs {
				shiftedVCoeffs[i] = mustRational(0, 1)
			}
			for i, c := range v.coeffs {
				prod, _ := simplifyMul([]Node{leadR, c})
				prodVal, _ := Eval(prod)
				shiftedVCoeffs[i+degDiff] = prodVal
			}
			shiftedV := trimPoly(&univariatePoly{varName: u.varName, coeffs: shiftedVCoeffs})

			r = polySub(scaleR, shiftedV)
		} else {
			r = polyScale(r, leadV)
		}
	}

	_ = delta
	return trimPoly(r), true
}

// polyExactDivideScalar divides all coefficients of P by scalarNode.
func polyExactDivideScalar(P *univariatePoly, scalarNode Node) (*univariatePoly, error) {
	if isZero(scalarNode) {
		return nil, fmt.Errorf("poly division by zero scalar")
	}
	if isOne(scalarNode) {
		return P, nil
	}

	// 1. If scalarNode is a rational number, scale directly
	if rat, ok := scalarNode.(*RationalNode); ok {
		if rat.Val.Sign() == 0 {
			return nil, fmt.Errorf("poly division by zero rational")
		}
		invRat := &RationalNode{Val: new(big.Rat).Inv(rat.Val)}
		newCoeffs := make([]Node, len(P.coeffs))
		for i, c := range P.coeffs {
			prod, _ := simplifyMul([]Node{c, invRat})
			newCoeffs[i], _ = Eval(expandNode(prod))
		}
		return trimPoly(&univariatePoly{varName: P.varName, coeffs: newCoeffs}), nil
	}

	// 2. If scalarNode contains variables, use exact polynomial division on each coefficient
	scalarVars := collectVariables(scalarNode)
	if len(scalarVars) > 0 {
		sVar := selectMainVariable(scalarVars)
		sPoly, okS := extractPoly(scalarNode, sVar)
		if okS && !isPolyZero(sPoly) {
			newCoeffs := make([]Node, len(P.coeffs))
			allDivided := true
			for i, c := range P.coeffs {
				if isZero(c) {
					newCoeffs[i] = mustRational(0, 1)
					continue
				}
				cPoly, okC := extractPoly(c, sVar)
				if okC {
					quot, rem, ok := polyDivide(cPoly, sPoly)
					if ok && isPolyZero(rem) {
						newCoeffs[i] = quot.toNode()
						continue
					}
				}
				allDivided = false
				break
			}
			if allDivided {
				return trimPoly(&univariatePoly{varName: P.varName, coeffs: newCoeffs}), nil
			}
		}
	}

	// 3. Fallback: simplifyMul with inverse
	invScalar, err := simplifyPow(scalarNode, mustRational(-1, 1))
	if err != nil {
		return nil, err
	}
	newCoeffs := make([]Node, len(P.coeffs))
	for i, c := range P.coeffs {
		divExpr, err := simplifyMul([]Node{c, invScalar})
		if err != nil {
			return nil, err
		}
		evaled, err := Eval(expandNode(divExpr))
		if err != nil {
			evaled = divExpr
		}
		newCoeffs[i] = evaled
	}

	return trimPoly(&univariatePoly{varName: P.varName, coeffs: newCoeffs}), nil
}


// polyContent computes the GCD of all coefficients of P with respect to varName.
func polyContent(P *univariatePoly, varName string) (Node, error) {
	if isPolyZero(P) {
		return mustRational(0, 1), nil
	}

	var nonZeroCoeffs []Node
	for _, c := range P.coeffs {
		if !isZero(c) {
			nonZeroCoeffs = append(nonZeroCoeffs, c)
		}
	}
	if len(nonZeroCoeffs) == 0 {
		return mustRational(0, 1), nil
	}
	if len(nonZeroCoeffs) == 1 {
		return normalizePolySign(nonZeroCoeffs[0]), nil
	}

	// Check if all coefficients are pure rationals
	allRational := true
	for _, c := range nonZeroCoeffs {
		if _, ok := c.(*RationalNode); !ok {
			allRational = false
			break
		}
	}

	if allRational {
		commonLcm := big.NewInt(1)
		for _, c := range nonZeroCoeffs {
			r := c.(*RationalNode)
			commonLcm = lcmInt(commonLcm, r.Val.Denom())
		}
		var commonGcd *big.Int
		for _, c := range nonZeroCoeffs {
			r := c.(*RationalNode)
			mult := new(big.Int).Div(commonLcm, r.Val.Denom())
			scaled := new(big.Int).Mul(r.Val.Num(), mult)
			scaled.Abs(scaled)
			if commonGcd == nil {
				commonGcd = scaled
			} else {
				commonGcd.GCD(nil, nil, commonGcd, scaled)
			}
		}
		if commonGcd == nil || commonGcd.Sign() == 0 {
			return mustRational(1, 1), nil
		}
		resRat := new(big.Rat).SetFrac(commonGcd, commonLcm)
		return &RationalNode{Val: resRat}, nil
	}

	// Multivariate: iteratively compute GCD of coefficients
	curGCD := nonZeroCoeffs[0]
	for i := 1; i < len(nonZeroCoeffs); i++ {
		nextGCD, err := EvalPolyGCD(curGCD, nonZeroCoeffs[i], "", nil)
		if err != nil {
			return mustRational(1, 1), nil
		}
		curGCD = nextGCD
		if isOne(curGCD) {
			return mustRational(1, 1), nil
		}
	}

	return normalizePolySign(curGCD), nil
}

// scalarGCD calculates GCD of two numbers or expressions containing no shared polynomial variables.
func scalarGCD(a, b Node) (Node, error) {
	ratA, okA := a.(*RationalNode)
	ratB, okB := b.(*RationalNode)
	if okA && okB {
		if ratA.Val.Sign() == 0 && ratB.Val.Sign() == 0 {
			return mustRational(0, 1), nil
		}
		if ratA.Val.Sign() == 0 {
			return &RationalNode{Val: new(big.Rat).Abs(ratB.Val)}, nil
		}
		if ratB.Val.Sign() == 0 {
			return &RationalNode{Val: new(big.Rat).Abs(ratA.Val)}, nil
		}
		lcmDenom := lcmInt(ratA.Val.Denom(), ratB.Val.Denom())
		multA := new(big.Int).Div(lcmDenom, ratA.Val.Denom())
		multB := new(big.Int).Div(lcmDenom, ratB.Val.Denom())
		numA := new(big.Int).Abs(new(big.Int).Mul(ratA.Val.Num(), multA))
		numB := new(big.Int).Abs(new(big.Int).Mul(ratB.Val.Num(), multB))
		gcdNum := new(big.Int).GCD(nil, nil, numA, numB)
		return &RationalNode{Val: new(big.Rat).SetFrac(gcdNum, lcmDenom)}, nil
	}

	if isZero(a) {
		return normalizePolySign(b), nil
	}
	if isZero(b) {
		return normalizePolySign(a), nil
	}

	return mustRational(1, 1), nil
}

// normalizePolySign ensures the leading coefficient has a positive sign, and if monic, leading coeff is 1.
func normalizePolySign(n Node) Node {
	evaled, err := Eval(expandNode(n))
	if err != nil {
		evaled = n
	}

	vars := collectVariables(evaled)
	if len(vars) == 0 {
		if r, ok := evaled.(*RationalNode); ok {
			return &RationalNode{Val: new(big.Rat).Abs(r.Val)}
		}
		return evaled
	}

	mainVar := selectMainVariable(vars)
	p, ok := extractPoly(evaled, mainVar)
	if !ok || isPolyZero(p) {
		return evaled
	}

	lead := p.leadCoeff()

	// If leading coefficient is a rational, we can make it monic over rationals:
	if ratLead, ok := lead.(*RationalNode); ok && ratLead.Val.Sign() != 0 {
		if isOne(lead) {
			return evaled
		}
		invLead, err := simplifyPow(lead, mustRational(-1, 1))
		if err == nil {
			invVal, err := Eval(invLead)
			if err == nil {
				monicP := polyScale(p, invVal)
				res, err := Eval(expandNode(monicP.toNode()))
				if err == nil {
					return res
				}
			}
		}
	}

	// Otherwise, check if leading coefficient is negative
	if s, err := signNode(lead); err == nil && s < 0 {
		negP := polyScale(p, mustRational(-1, 1))
		res, err := Eval(expandNode(negP.toNode()))
		if err == nil {
			return res
		}
	}

	return evaled
}

func unionStrings(a, b []string) []string {
	seen := make(map[string]bool)
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		seen[s] = true
	}
	res := make([]string, 0, len(seen))
	for s := range seen {
		res = append(res, s)
	}
	sort.Strings(res)
	return res
}
