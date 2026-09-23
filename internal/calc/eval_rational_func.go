package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"math/big"
)

// polySub subtracts polynomial B from A: A - B.
func polySub(A, B *univariatePoly) *univariatePoly {
	maxDeg := A.degree()
	if B.degree() > maxDeg {
		maxDeg = B.degree()
	}
	coeffs := make([]Node, maxDeg+1)
	for i := 0; i <= maxDeg; i++ {
		coeffs[i] = mustRational(0, 1)
	}
	copy(coeffs, A.coeffs)
	for i, c := range B.coeffs {
		neg, _ := simplifyUnaryOp("-", c)
		negVal, _ := Eval(neg)
		sum, _ := simplifyAdd([]Node{coeffs[i], negVal})
		coeffs[i], _ = Eval(sum)
	}
	return trimPoly(&univariatePoly{varName: A.varName, coeffs: coeffs})
}

// polyMul multiplies two polynomials A and B.
func polyMul(A, B *univariatePoly) *univariatePoly {
	if isPolyZero(A) || isPolyZero(B) {
		return &univariatePoly{varName: A.varName, coeffs: []Node{mustRational(0, 1)}}
	}
	newDeg := A.degree() + B.degree()
	coeffs := make([]Node, newDeg+1)
	for i := 0; i <= newDeg; i++ {
		coeffs[i] = mustRational(0, 1)
	}
	for i, ca := range A.coeffs {
		for j, cb := range B.coeffs {
			prod, _ := simplifyMul([]Node{ca, cb})
			prodVal, _ := Eval(prod)
			sum, _ := simplifyAdd([]Node{coeffs[i+j], prodVal})
			coeffs[i+j], _ = Eval(sum)
		}
	}
	return trimPoly(&univariatePoly{varName: A.varName, coeffs: coeffs})
}

// polyScale scales polynomial A by a scalar node factor.
func polyScale(A *univariatePoly, factor Node) *univariatePoly {
	coeffs := make([]Node, len(A.coeffs))
	for i, c := range A.coeffs {
		prod, _ := simplifyMul([]Node{c, factor})
		coeffs[i], _ = Eval(prod)
	}
	return trimPoly(&univariatePoly{varName: A.varName, coeffs: coeffs})
}

// polyExtendedGCD computes polynomials S, T, and G such that S*A + T*B = G = gcd(A, B).
// G is normalized to be monic (leading coefficient = 1).
func polyExtendedGCD(A, B *univariatePoly) (S, T, G *univariatePoly, ok bool) {
	varName := A.varName
	zero := &univariatePoly{varName: varName, coeffs: []Node{mustRational(0, 1)}}
	one := &univariatePoly{varName: varName, coeffs: []Node{mustRational(1, 1)}}

	r0, r1 := A, B
	s0, s1 := one, zero
	t0, t1 := zero, one

	for !isPolyZero(r1) {
		q, rem, divOk := polyDivide(r0, r1)
		if !divOk {
			return nil, nil, nil, false
		}
		r0, r1 = r1, rem

		// sNext = s0 - q * s1
		qS1 := polyMul(q, s1)
		sNext := polySub(s0, qS1)
		s0, s1 = s1, sNext

		// tNext = t0 - q * t1
		qT1 := polyMul(q, t1)
		tNext := polySub(t0, qT1)
		t0, t1 = t1, tNext
	}

	// r0 is GCD. Normalize r0 to be monic.
	lead := r0.leadCoeff()
	invLead, err := simplifyPow(lead, mustRational(-1, 1))
	if err != nil {
		return nil, nil, nil, false
	}
	invLeadVal, _ := Eval(invLead)

	G = polyScale(r0, invLeadVal)
	S = polyScale(s0, invLeadVal)
	T = polyScale(t0, invLeadVal)
	return S, T, G, true
}

// rationalTerm represents a fraction term: num / (base^pow).
type rationalTerm struct {
	num  *univariatePoly
	base *univariatePoly
	pow  int
}

// henriciBaseExpansion decomposes num / (base^k) into sum_{j=1}^k (r_j / base^j).
// Reference: P. Henrici (1971).
func henriciBaseExpansion(num, base *univariatePoly, k int) ([]*rationalTerm, error) {
	var terms []*rationalTerm
	cur := num
	for j := k; j >= 1; j-- {
		q, rem, ok := polyDivide(cur, base)
		if !ok {
			return nil, fmt.Errorf("%s", i18n.T("rational.err_henricibaseexpansion_polynomial_division_failed"))
		}
		if !isPolyZero(rem) {
			terms = append(terms, &rationalTerm{
				num:  rem,
				base: base,
				pow:  j,
			})
		}
		cur = q
		if isPolyZero(cur) {
			break
		}
	}
	// If cur is not zero after exhausting base^k, that's a polynomial quotient
	if !isPolyZero(cur) {
		terms = append(terms, &rationalTerm{
			num:  cur,
			base: &univariatePoly{varName: base.varName, coeffs: []Node{mustRational(1, 1)}},
			pow:  1,
		})
	}
	return terms, nil
}

// polyFactorPower represents an irreducible or coprime factor raised to a power: base^pow.
type polyFactorPower struct {
	base *univariatePoly
	pow  int
}

// fullPoly returns base^pow as a univariatePoly.
func (pfp polyFactorPower) fullPoly() *univariatePoly {
	res := &univariatePoly{varName: pfp.base.varName, coeffs: []Node{mustRational(1, 1)}}
	for i := 0; i < pfp.pow; i++ {
		res = polyMul(res, pfp.base)
	}
	return res
}

// decomposeKungTong recursively splits coprime denominator factors using Bézout coefficients.
// Reference: H. T. Kung & D. M. Tong (1977).
func decomposeKungTong(num *univariatePoly, factors []polyFactorPower) ([]*rationalTerm, error) {
	if len(factors) == 0 {
		return nil, nil
	}
	if len(factors) == 1 {
		return henriciBaseExpansion(num, factors[0].base, factors[0].pow)
	}

	mid := len(factors) / 2
	leftFactors := factors[:mid]
	rightFactors := factors[mid:]

	// Multiply left factor full polynomials
	U := &univariatePoly{varName: num.varName, coeffs: []Node{mustRational(1, 1)}}
	for _, f := range leftFactors {
		U = polyMul(U, f.fullPoly())
	}

	// Multiply right factor full polynomials
	V := &univariatePoly{varName: num.varName, coeffs: []Node{mustRational(1, 1)}}
	for _, f := range rightFactors {
		V = polyMul(V, f.fullPoly())
	}

	// S*U + T*V = 1 via EEA
	S, T, G, ok := polyExtendedGCD(U, V)
	if !ok || G.degree() > 0 {
		// Not coprime or GCD failure
		return nil, fmt.Errorf("%s", i18n.T("rational.err_decomposekungtong_factors_are_not_coprime", G.degree()))
	}

	// num / (U*V) = (num * T) / U + (num * S) / V
	numT := polyMul(num, T)
	numS := polyMul(num, S)

	_, remU, okU := polyDivide(numT, U)
	_, remV, okV := polyDivide(numS, V)
	if !okU || !okV {
		return nil, fmt.Errorf("%s", i18n.T("rational.err_decomposekungtong_reduction_division_failed"))
	}

	leftTerms, err := decomposeKungTong(remU, leftFactors)
	if err != nil {
		return nil, err
	}
	rightTerms, err := decomposeKungTong(remV, rightFactors)
	if err != nil {
		return nil, err
	}

	return append(leftTerms, rightTerms...), nil
}

// extractNumeratorDenominator separates an AST node into numerator and denominator nodes.
func extractNumeratorDenominator(n Node) (Node, Node) {
	switch v := n.(type) {
	case *RationalNode:
		num := &RationalNode{Val: big.NewRat(v.Val.Num().Int64(), 1)}
		den := &RationalNode{Val: big.NewRat(v.Val.Denom().Int64(), 1)}
		return num, den

	case *PowNode:
		if r, ok := v.Exp.(*RationalNode); ok && r.Val.Sign() < 0 {
			posExp := &RationalNode{Val: new(big.Rat).Neg(r.Val)}
			den, _ := NewPow(v.Base, posExp)
			return mustRational(1, 1), den
		}
		return v, mustRational(1, 1)

	case *MulNode:
		var numFactors []Node
		var denFactors []Node
		for _, f := range v.Factors {
			fn, fd := extractNumeratorDenominator(f)
			if !isOne(fn) {
				numFactors = append(numFactors, fn)
			}
			if !isOne(fd) {
				denFactors = append(denFactors, fd)
			}
		}
		var numNode, denNode Node
		if len(numFactors) == 0 {
			numNode = mustRational(1, 1)
		} else if len(numFactors) == 1 {
			numNode = numFactors[0]
		} else {
			numNode = NewMul(numFactors)
		}

		if len(denFactors) == 0 {
			denNode = mustRational(1, 1)
		} else if len(denFactors) == 1 {
			denNode = denFactors[0]
		} else {
			denNode = NewMul(denFactors)
		}
		return numNode, denNode

	default:
		return n, mustRational(1, 1)
	}
}

// collectDenominatorFactors factors the denominator node and parses factor powers.
func collectDenominatorFactors(denNode Node, varName string) ([]polyFactorPower, Node, error) {
	factored, err := Factor(denNode, varName)
	if err != nil {
		factored = denNode
	}

	var factors []polyFactorPower
	var scalarCoeff Node = mustRational(1, 1)

	var processFactor func(f Node) error
	processFactor = func(f Node) error {
		switch v := f.(type) {
		case *RationalNode:
			scalarCoeff, _ = Eval(NewMul([]Node{scalarCoeff, v}))
			return nil

		case *PowNode:
			if r, ok := v.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() > 0 {
				basePoly, ok := extractPoly(v.Base, varName)
				if !ok {
					return fmt.Errorf("%s", i18n.T("rational.err_non_polynomial_factor_in_denominator"))
				}
				lead := basePoly.leadCoeff()
				if !isOne(lead) {
					// Extract leading coefficient: (c * B)^k = c^k * B^k
					leadPow, _ := simplifyPow(lead, r)
					scalarCoeff, _ = Eval(NewMul([]Node{scalarCoeff, leadPow}))
					invLead, _ := simplifyPow(lead, mustRational(-1, 1))
					invLeadVal, _ := Eval(invLead)
					basePoly = polyScale(basePoly, invLeadVal)
				}
				factors = append(factors, polyFactorPower{
					base: basePoly,
					pow:  int(r.Val.Num().Int64()),
				})
				return nil
			}
			return fmt.Errorf("%s", i18n.T("rational.err_unsupported_power_in_denominator_factor"))

		case *MulNode:
			for _, sub := range v.Factors {
				if err := processFactor(sub); err != nil {
					return err
				}
			}
			return nil

		default:
			basePoly, ok := extractPoly(v, varName)
			if !ok {
				return fmt.Errorf("%s", i18n.T("rational.err_non_polynomial_denominator_factor"))
			}
			lead := basePoly.leadCoeff()
			if !isOne(lead) {
				scalarCoeff, _ = Eval(NewMul([]Node{scalarCoeff, lead}))
				invLead, _ := simplifyPow(lead, mustRational(-1, 1))
				invLeadVal, _ := Eval(invLead)
				basePoly = polyScale(basePoly, invLeadVal)
			}
			factors = append(factors, polyFactorPower{
				base: basePoly,
				pow:  1,
			})
			return nil
		}
	}

	if err := processFactor(factored); err != nil {
		return nil, nil, err
	}

	// Merge duplicate bases by adding their powers
	var merged []polyFactorPower
	for _, f := range factors {
		found := false
		for i, m := range merged {
			// Check if f.base equals m.base
			diff := polySub(f.base, m.base)
			if isPolyZero(diff) {
				merged[i].pow += f.pow
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, f)
		}
	}

	return merged, scalarCoeff, nil
}

// EvalApart implements partial fraction decomposition (PFD).
// Usage: apart(expr, [var])
func EvalApart(expr Node, varNames ...string) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("%s", i18n.T("rational.err_apart_nil_expression"))
	}
	simplified, err := Eval(expr)
	if err != nil {
		return nil, err
	}

	// Determine target variable
	var v string
	if len(varNames) > 0 && varNames[0] != "" {
		v = varNames[0]
	} else {
		vars := collectVariables(simplified)
		if len(vars) == 0 {
			return simplified, nil
		}
		v = selectMainVariable(vars)
	}

	// If expr is an AddNode, decompose each term and sum
	if addNode, ok := simplified.(*AddNode); ok {
		var decomposedTerms []Node
		for _, t := range addNode.Terms {
			dec, err := EvalApart(t, v)
			if err != nil {
				return nil, err
			}
			decomposedTerms = append(decomposedTerms, dec)
		}
		sum, err := simplifyAdd(decomposedTerms)
		if err != nil {
			return nil, err
		}
		return Eval(sum)
	}

	numNode, denNode := extractNumeratorDenominator(simplified)
	if isOne(denNode) {
		// Pure polynomial / constant, nothing to decompose
		return simplified, nil
	}

	polyNum, okNum := extractPoly(numNode, v)
	polyDen, okDen := extractPoly(denNode, v)
	if !okNum || !okDen || isPolyZero(polyDen) {
		return simplified, nil
	}

	// 1. Polynomial long division: num = Q * den + R
	quot, rem, divOk := polyDivide(polyNum, polyDen)
	if !divOk {
		return nil, fmt.Errorf("%s", i18n.T("rational.err_apart_polynomial_division_failed"))
	}

	var resultTerms []Node
	if !isPolyZero(quot) {
		resultTerms = append(resultTerms, quot.toNode())
	}

	if isPolyZero(rem) {
		if len(resultTerms) == 0 {
			return mustRational(0, 1), nil
		}
		sum, _ := simplifyAdd(resultTerms)
		return Eval(sum)
	}

	// 2. Factor denominator
	factors, scalarCoeff, err := collectDenominatorFactors(denNode, v)
	if err != nil || len(factors) == 0 {
		// Cannot factor, return original quotient + remainder
		frac, _ := simplifyMul([]Node{rem.toNode(), mustInv(polyDen.toNode())})
		resultTerms = append(resultTerms, frac)
		sum, _ := simplifyAdd(resultTerms)
		return Eval(sum)
	}

	// Divide remainder by scalarCoeff
	if !isOne(scalarCoeff) {
		invScalar, _ := simplifyPow(scalarCoeff, mustRational(-1, 1))
		invScalarVal, _ := Eval(invScalar)
		rem = polyScale(rem, invScalarVal)
	}

	// 3. Kung & Tong (1977) + Henrici (1971) decomposition
	terms, err := decomposeKungTong(rem, factors)
	if err != nil {
		// Fallback to quotient + remainder / den
		frac, _ := simplifyMul([]Node{rem.toNode(), mustInv(polyDen.toNode())})
		resultTerms = append(resultTerms, frac)
		sum, _ := simplifyAdd(resultTerms)
		return Eval(sum)
	}

	// Convert terms to AST nodes
	for _, term := range terms {
		baseNode := term.base.toNode()
		var denTerm Node
		if term.pow == 1 {
			denTerm = baseNode
		} else {
			denTerm, _ = NewPow(baseNode, mustRational(int64(term.pow), 1))
		}
		invDen, _ := simplifyPow(denTerm, mustRational(-1, 1))
		frac, _ := simplifyMul([]Node{term.num.toNode(), invDen})
		fracVal, _ := Eval(frac)
		resultTerms = append(resultTerms, fracVal)
	}

	if len(resultTerms) == 0 {
		return mustRational(0, 1), nil
	}
	if len(resultTerms) == 1 {
		return resultTerms[0], nil
	}
	sum, err := simplifyAdd(resultTerms)
	if err != nil {
		return nil, err
	}
	return Eval(sum)
}

// EvalTogether puts sums of rational expressions over a common denominator.
// Usage: together(expr)
func EvalTogether(expr Node) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("%s", i18n.T("rational.err_together_nil_expression"))
	}
	simplified, err := Eval(expr)
	if err != nil {
		return nil, err
	}

	addNode, ok := simplified.(*AddNode)
	if !ok {
		// Not a sum, already together
		return simplified, nil
	}

	// Extract numerator and denominator for each term
	type fracItem struct {
		num Node
		den Node
	}
	var items []fracItem
	var denList []Node

	for _, t := range addNode.Terms {
		num, den := extractNumeratorDenominator(t)
		items = append(items, fracItem{num: num, den: den})
		if !isOne(den) {
			denList = append(denList, den)
		}
	}

	if len(denList) == 0 {
		// No fractions
		return simplified, nil
	}

	// Construct common denominator D = product of distinct denominators
	var distinctDens []Node
	for _, d := range denList {
		alreadyExists := false
		for _, ed := range distinctDens {
			diff, _ := simplifyAdd([]Node{d, mustNeg(ed)})
			diffVal, _ := Eval(diff)
			if isZero(diffVal) {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			distinctDens = append(distinctDens, d)
		}
	}

	var commonDen Node
	if len(distinctDens) == 1 {
		commonDen = distinctDens[0]
	} else {
		commonDen = NewMul(distinctDens)
	}
	commonDenVal, err := Eval(commonDen)
	if err != nil {
		return nil, err
	}

	vars := collectVariables(simplified)
	var v string
	if len(vars) > 0 {
		v = selectMainVariable(vars)
	}

	// Expand each numerator scaled by (commonDen / den)
	var newNumTerms []Node
	for _, it := range items {
		var scale Node
		if isOne(it.den) {
			scale = commonDenVal
		} else {
			if v != "" {
				pCommon, okC := extractPoly(commonDenVal, v)
				pDen, okD := extractPoly(it.den, v)
				if okC && okD && !isPolyZero(pDen) {
					q, rem, divOk := polyDivide(pCommon, pDen)
					if divOk && isPolyZero(rem) {
						scale = q.toNode()
					}
				}
			}
			if scale == nil {
				invDen, _ := simplifyPow(it.den, mustRational(-1, 1))
				scaleNode, _ := simplifyMul([]Node{commonDenVal, invDen})
				scale, _ = Eval(scaleNode)
			}
		}
		termProd, _ := simplifyMul([]Node{it.num, scale})
		expanded := expandNode(termProd)
		newNumTerms = append(newNumTerms, expanded)
	}

	totalNum, err := simplifyAdd(newNumTerms)
	if err != nil {
		return nil, err
	}
	totalNumExpanded := expandNode(totalNum)

	invCommonDen, _ := simplifyPow(commonDenVal, mustRational(-1, 1))
	return NewMul([]Node{totalNumExpanded, invCommonDen}), nil
}

func mustNeg(n Node) Node {
	neg, _ := simplifyUnaryOp("-", n)
	val, _ := Eval(neg)
	return val
}

func mustInv(n Node) Node {
	inv, _ := simplifyPow(n, mustRational(-1, 1))
	val, _ := Eval(inv)
	return val
}
