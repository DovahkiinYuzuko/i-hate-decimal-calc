package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Data Structures for Primitive Element & Isomorphism
// -------------------------------------------------------------------------

// PrimitiveElementResult holds the isomorphism data for Q(alpha, beta) = Q(theta).
type PrimitiveElementResult struct {
	MinPolyTheta *ast.PolyNode   // Minimal polynomial of primitive element theta, m_theta(x) = 0
	SymbolTheta  string          // Symbol for primitive element, default "theta"
	C            int64           // Integer coefficient c such that theta = alpha + c*beta
	RepAlpha     *ast.PolyNode   // alpha expressed as polynomial in theta: P_alpha(theta)
	RepBeta      *ast.PolyNode   // beta expressed as polynomial in theta: P_beta(theta)
	Degree       int             // Extension degree [Q(theta) : Q] = deg(m_theta)
	MinPolyAlpha *ast.PolyNode   // Defining minimal polynomial of alpha
	MinPolyBeta  *ast.PolyNode   // Defining minimal polynomial of beta
	QuotientG    *ast.PolyNode   // Certificate quotient: g(P_beta(x)) / m_theta(x)
	QuotientF    *ast.PolyNode   // Certificate quotient: f(P_alpha(x)) / m_theta(x)
}

// MultiPrimitiveElementResult holds the isomorphism data for Q(alpha_1, ..., alpha_k) = Q(theta).
type MultiPrimitiveElementResult struct {
	MinPolyTheta *ast.PolyNode   // Minimal polynomial of primitive element theta
	SymbolTheta  string          // Symbol for primitive element
	Reps         []*ast.PolyNode // Expressions of each generator alpha_i in terms of theta
	Degree       int             // Extension degree [Q(theta) : Q]
}

// -------------------------------------------------------------------------
// Polynomial Arithmetic over Algebraic Extension Field L = Q(theta)
// -------------------------------------------------------------------------

// polyOverAlgExt represents a polynomial in indeterminate varName with coefficients in Q(theta).
// coeffs[i] is the coefficient of varName^i, represented as a univariate PolyNode in symbolTheta.
type polyOverAlgExt struct {
	varName      string
	minPolyTheta *ast.PolyNode
	symbolTheta  string
	coeffs       []*ast.PolyNode
}

func newPolyOverAlgExt(varName string, minPolyTheta *ast.PolyNode, symbolTheta string, coeffs []*ast.PolyNode) *polyOverAlgExt {
	p := &polyOverAlgExt{
		varName:      varName,
		minPolyTheta: minPolyTheta,
		symbolTheta:  symbolTheta,
		coeffs:       coeffs,
	}
	p.normalize()
	return p
}

// degree returns the degree of the polynomial in varName (-1 if zero).
func (p *polyOverAlgExt) degree() int {
	return len(p.coeffs) - 1
}

// normalize removes trailing zero coefficients in Q(theta).
func (p *polyOverAlgExt) normalize() {
	zeroRat := big.NewRat(0, 1)
	for len(p.coeffs) > 0 {
		last := p.coeffs[len(p.coeffs)-1]
		if last == nil || len(last.Terms) == 0 {
			p.coeffs = p.coeffs[:len(p.coeffs)-1]
			continue
		}
		// Check if coefficient reduces to 0 modulo minPolyTheta
		_, rem, err := PolyDivRem(last, p.minPolyTheta, p.symbolTheta)
		if err == nil && (len(rem.Terms) == 0 || (len(rem.Terms) == 1 && rem.Terms[0].Coeff.Cmp(zeroRat) == 0)) {
			p.coeffs = p.coeffs[:len(p.coeffs)-1]
			continue
		}
		if err == nil {
			p.coeffs[len(p.coeffs)-1] = rem
		}
		break
	}
}

// mulScalar multiplies polynomial by scalar c(theta) in L.
func (p *polyOverAlgExt) mulScalar(c *ast.PolyNode) *polyOverAlgExt {
	resCoeffs := make([]*ast.PolyNode, len(p.coeffs))
	for i, coeff := range p.coeffs {
		if coeff == nil || len(coeff.Terms) == 0 {
			resCoeffs[i] = NewPolyNode([]string{p.symbolTheta}, ast.OrderLex, nil)
		} else {
			prod := MulPoly(coeff, c)
			_, rem, _ := PolyDivRem(prod, p.minPolyTheta, p.symbolTheta)
			resCoeffs[i] = rem
		}
	}
	return newPolyOverAlgExt(p.varName, p.minPolyTheta, p.symbolTheta, resCoeffs)
}

// invertInField computes the inverse of element a(theta) in L = Q(theta) modulo minPolyTheta.
func invertInField(a, minPoly *ast.PolyNode, symbol string) (*ast.PolyNode, error) {
	if a == nil || len(a.Terms) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_nil_algebraic_number"))
	}
	res, err := canonicalPolyExtendedGCD(a, minPoly, symbol)
	if err != nil {
		return nil, err
	}
	// Check if gcd is constant 1
	if len(res.gcd.Terms) == 0 || DegreeInVar(res.gcd, symbol) != 0 {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_element_is_not_invertible_modulo", minPoly.String()))
	}
	// Monicize inverse with respect to constant gcd
	lc := res.gcd.Terms[0].Coeff
	if lc.Cmp(big.NewRat(1, 1)) != 0 {
		invLC := new(big.Rat).Inv(lc)
		scalePoly := NewPolyNode([]string{symbol}, ast.OrderLex, []ast.Monomial{{Coeff: invLC, Exponents: []int{0}}})
		inv := MulPoly(res.u, scalePoly)
		_, rem, _ := PolyDivRem(inv, minPoly, symbol)
		return rem, nil
	}
	_, rem, _ := PolyDivRem(res.u, minPoly, symbol)
	return rem, nil
}

// divRem computes quotient and remainder such that dividend = quotient * divisor + remainder in L[varName].
func polyOverAlgExtDivRem(dividend, divisor *polyOverAlgExt) (quo, rem *polyOverAlgExt, err error) {
	divisor.normalize()
	dividend.normalize()
	degDiv := divisor.degree()
	if degDiv < 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("algebra.err_polynomial_division_by_zero"))
	}
	degDivd := dividend.degree()
	if degDivd < degDiv {
		zeroQ := newPolyOverAlgExt(dividend.varName, dividend.minPolyTheta, dividend.symbolTheta, nil)
		remCopy := make([]*ast.PolyNode, len(dividend.coeffs))
		copy(remCopy, dividend.coeffs)
		return zeroQ, newPolyOverAlgExt(dividend.varName, dividend.minPolyTheta, dividend.symbolTheta, remCopy), nil
	}

	lcDiv := divisor.coeffs[degDiv]
	invLC, err := invertInField(lcDiv, divisor.minPolyTheta, divisor.symbolTheta)
	if err != nil {
		return nil, nil, fmt.Errorf("inverting divisor leading coefficient: %w", err)
	}

	remCoeffs := make([]*ast.PolyNode, len(dividend.coeffs))
	copy(remCoeffs, dividend.coeffs)
	remPoly := newPolyOverAlgExt(dividend.varName, dividend.minPolyTheta, dividend.symbolTheta, remCoeffs)

	quoLen := degDivd - degDiv + 1
	quoCoeffs := make([]*ast.PolyNode, quoLen)
	for i := range quoCoeffs {
		quoCoeffs[i] = NewPolyNode([]string{dividend.symbolTheta}, ast.OrderLex, nil)
	}

	for remPoly.degree() >= degDiv {
		curDeg := remPoly.degree()
		lcRem := remPoly.coeffs[curDeg]

		// qTermCoeff = lcRem * invLC mod minPoly
		prod := MulPoly(lcRem, invLC)
		_, qCoeff, _ := PolyDivRem(prod, dividend.minPolyTheta, dividend.symbolTheta)

		k := curDeg - degDiv
		quoCoeffs[k] = qCoeff

		// Subtract qCoeff * x^k * divisor from remPoly
		for j := 0; j <= degDiv; j++ {
			termProd := MulPoly(qCoeff, divisor.coeffs[j])
			_, normTermProd, _ := PolyDivRem(termProd, dividend.minPolyTheta, dividend.symbolTheta)
			remPoly.coeffs[j+k] = SubPoly(remPoly.coeffs[j+k], normTermProd)
			_, normDiff, _ := PolyDivRem(remPoly.coeffs[j+k], dividend.minPolyTheta, dividend.symbolTheta)
			remPoly.coeffs[j+k] = normDiff
		}
		remPoly.normalize()
	}

	return newPolyOverAlgExt(dividend.varName, dividend.minPolyTheta, dividend.symbolTheta, quoCoeffs), remPoly, nil
}

// polyOverAlgExtGCD computes monic gcd of a and b in L[varName] using the Euclidean algorithm.
func polyOverAlgExtGCD(a, b *polyOverAlgExt) (*polyOverAlgExt, error) {
	r0Coeffs := make([]*ast.PolyNode, len(a.coeffs))
	copy(r0Coeffs, a.coeffs)
	r0 := newPolyOverAlgExt(a.varName, a.minPolyTheta, a.symbolTheta, r0Coeffs)

	r1Coeffs := make([]*ast.PolyNode, len(b.coeffs))
	copy(r1Coeffs, b.coeffs)
	r1 := newPolyOverAlgExt(b.varName, b.minPolyTheta, b.symbolTheta, r1Coeffs)

	for r1.degree() >= 0 {
		_, rem, err := polyOverAlgExtDivRem(r0, r1)
		if err != nil {
			return nil, err
		}
		r0 = r1
		r1 = rem
	}

	// Make r0 monic
	if r0.degree() >= 0 {
		deg := r0.degree()
		lc := r0.coeffs[deg]
		invLC, err := invertInField(lc, r0.minPolyTheta, r0.symbolTheta)
		if err != nil {
			return nil, err
		}
		r0 = r0.mulScalar(invLC)
	}

	return r0, nil
}

// -------------------------------------------------------------------------
// Helper Conversions: Univariate Rational Poly to L[y]
// -------------------------------------------------------------------------

// liftPolyToAlgExt lifts univariate polynomial p to L[varName].
// The main variable of p is taken from p.Vars[0] and mapped to the indeterminate varName in L[varName].
func liftPolyToAlgExt(p *ast.PolyNode, varName, symbolTheta string, minPolyTheta *ast.PolyNode) *polyOverAlgExt {
	if p == nil || len(p.Terms) == 0 || len(p.Vars) == 0 {
		return newPolyOverAlgExt(varName, minPolyTheta, symbolTheta, nil)
	}
	origVar := p.Vars[0]
	deg := DegreeInVar(p, origVar)
	if deg < 0 {
		return newPolyOverAlgExt(varName, minPolyTheta, symbolTheta, nil)
	}
	coeffs := make([]*ast.PolyNode, deg+1)
	for i := range coeffs {
		coeffs[i] = NewPolyNode([]string{symbolTheta}, ast.OrderLex, nil)
	}

	for _, mon := range p.Terms {
		power := 0
		if len(mon.Exponents) > 0 {
			power = mon.Exponents[0]
		}
		scalarPoly := NewPolyNode([]string{symbolTheta}, ast.OrderLex, []ast.Monomial{
			{Coeff: new(big.Rat).Set(mon.Coeff), Exponents: []int{0}},
		})
		coeffs[power] = AddPoly(coeffs[power], scalarPoly)
	}

	return newPolyOverAlgExt(varName, minPolyTheta, symbolTheta, coeffs)
}

// polyDerivative computes derivative d/dx of univariate polynomial p with respect to varName.
func polyDerivative(p *ast.PolyNode, varName string) *ast.PolyNode {
	if p == nil || len(p.Terms) == 0 {
		return NewPolyNode([]string{varName}, ast.OrderLex, nil)
	}
	varIdx := -1
	for idx, v := range p.Vars {
		if v == varName {
			varIdx = idx
			break
		}
	}

	var dTerms []ast.Monomial
	for _, term := range p.Terms {
		exp := 0
		if varIdx >= 0 && varIdx < len(term.Exponents) {
			exp = term.Exponents[varIdx]
		}
		if exp > 0 {
			newCoeff := new(big.Rat).Mul(term.Coeff, big.NewRat(int64(exp), 1))
			newExps := make([]int, len(term.Exponents))
			copy(newExps, term.Exponents)
			newExps[varIdx] = exp - 1
			dTerms = append(dTerms, ast.Monomial{Coeff: newCoeff, Exponents: newExps})
		}
	}
	return NewPolyNode(p.Vars, p.Order, dTerms)
}

// isSquareFreePoly checks if univariate polynomial p is square-free: gcd(p, p') = 1.
func isSquareFreePoly(p *ast.PolyNode, varName string) bool {
	if p == nil || len(p.Terms) == 0 {
		return false
	}
	dp := polyDerivative(p, varName)
	if len(dp.Terms) == 0 {
		return DegreeInVar(p, varName) <= 1
	}
	res, err := canonicalPolyExtendedGCD(p, dp, varName)
	if err != nil {
		return false
	}
	return DegreeInVar(res.gcd, varName) == 0
}

// -------------------------------------------------------------------------
// Core Primitive Element Algorithm (Loos 1983, Trager 1976)
// -------------------------------------------------------------------------

// FindPrimitiveElement computes primitive element theta = alpha + c*beta and representations.
func FindPrimitiveElement(minPoly1, minPoly2 *ast.PolyNode, symbolTheta string) (*PrimitiveElementResult, error) {
	if minPoly1 == nil || len(minPoly1.Terms) == 0 || minPoly2 == nil || len(minPoly2.Terms) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("primitive_element.err_nil_polynomial"))
	}
	if symbolTheta == "" {
		symbolTheta = "theta"
	}

	m1 := MonicPoly(minPoly1)
	m2 := MonicPoly(minPoly2)

	deg1 := DegreeInVar(m1, m1.Vars[0])
	deg2 := DegreeInVar(m2, m2.Vars[0])
	if deg1 <= 0 || deg2 <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("primitive_element.err_nil_polynomial"))
	}

	fsm := NewPrimitiveElementFSM()
	if err := fsm.Transition(PrimitiveStateScanningC); err != nil {
		return nil, err
	}

	// Deterministic sequence of candidates for c: 1, -1, 2, -2, 3, -3, 4, -4, ...
	candidateSequence := []int64{1, -1, 2, -2, 3, -3, 4, -4, 5, -5, 6, -6, 7, -7, 8, -8}

	targetVarX := symbolTheta
	varY := "_y"

	m1Node := PolyToNode(m1)
	m2Node := PolyToNode(m2)

	for _, c := range candidateSequence {
		fsm.CandidateC = c

		if fsm.Current() != PrimitiveStateScanningC {
			if err := fsm.Transition(PrimitiveStateScanningC); err != nil {
				return nil, err
			}
		}

		if err := fsm.Transition(PrimitiveStateResultant); err != nil {
			return nil, err
		}

		// 1. Substitute in m1: var -> (theta - c * y)
		cVal := mustRational(c, 1)
		termCY := &ast.MulNode{Factors: []ast.Node{cVal, ast.NewVar(varY)}}
		substM1Expr := &ast.AddNode{Terms: []ast.Node{
			ast.NewVar(targetVarX),
			&ast.UnaryOpNode{Op: "-", Expr: termCY},
		}}
		substM1 := ast.Substitute(m1Node, m1.Vars[0], substM1Expr)

		// Substitute in m2: var -> y
		substM2 := ast.Substitute(m2Node, m2.Vars[0], ast.NewVar(varY))

		// Compute Sylvester Resultant Res_y(m1(theta - c*y), m2(y))
		env := NewEnv()
		resNode, err := EvalResultant(substM1, substM2, varY, env)
		if err != nil {
			continue
		}

		resPoly, err := NodeToPoly(resNode, []string{targetVarX}, ast.OrderLex)
		if err != nil {
			continue
		}
		monicRes := MonicPoly(resPoly)

		if err := fsm.Transition(PrimitiveStateSquareFreeCheck); err != nil {
			return nil, err
		}

		// 2. Square-free check
		if !isSquareFreePoly(monicRes, targetVarX) {
			continue
		}

		// 3. Solve for generator representations via GCD in L[y]
		if err := fsm.Transition(PrimitiveStateSolveGenerators); err != nil {
			return nil, err
		}

		// Lift m2(y) to L[y]
		gy := liftPolyToAlgExt(m2, varY, targetVarX, monicRes)

		// Construct fy = m1(theta - c*y) in L[y]
		fyNode := substM1
		fyPoly, err := NodeToPoly(fyNode, []string{varY, targetVarX}, ast.OrderLex)
		if err != nil {
			if c == 1 {
				fmt.Printf("DEBUG c=1: fy NodeToPoly err: %v\n", err)
			}
			continue
		}

		// Group fy by powers of varY
		degY := DegreeInVar(fyPoly, varY)
		if degY < 0 {
			if c == 1 {
				fmt.Printf("DEBUG c=1: degY < 0\n")
			}
			continue
		}
		fyCoeffs := make([]*ast.PolyNode, degY+1)
		for i := range fyCoeffs {
			fyCoeffs[i] = NewPolyNode([]string{targetVarX}, ast.OrderLex, nil)
		}

		yIdx := -1
		for idx, v := range fyPoly.Vars {
			if v == varY {
				yIdx = idx
				break
			}
		}
		thetaIdx := -1
		for idx, v := range fyPoly.Vars {
			if v == targetVarX {
				thetaIdx = idx
				break
			}
		}

		for _, mon := range fyPoly.Terms {
			powY := 0
			if yIdx >= 0 && yIdx < len(mon.Exponents) {
				powY = mon.Exponents[yIdx]
			}
			powTheta := 0
			if thetaIdx >= 0 && thetaIdx < len(mon.Exponents) {
				powTheta = mon.Exponents[thetaIdx]
			}
			termTheta := NewPolyNode([]string{targetVarX}, ast.OrderLex, []ast.Monomial{
				{Coeff: new(big.Rat).Set(mon.Coeff), Exponents: []int{powTheta}},
			})
			fyCoeffs[powY] = AddPoly(fyCoeffs[powY], termTheta)
		}

		fy := newPolyOverAlgExt(varY, monicRes, targetVarX, fyCoeffs)

		// Compute GCD(fy, gy) in L[y]
		gcdPoly, err := polyOverAlgExtGCD(fy, gy)
		if err != nil {
			continue
		}

		// GCD must be linear: y - P_beta(theta)
		if gcdPoly.degree() != 1 {
			continue
		}

		// beta = - coeffs[0] (since leading coeff is 1)
		negOne := NewPolyNode([]string{targetVarX}, ast.OrderLex, []ast.Monomial{
			{Coeff: big.NewRat(-1, 1), Exponents: []int{0}},
		})
		repBeta := MulPoly(gcdPoly.coeffs[0], negOne)
		_, repBeta, _ = PolyDivRem(repBeta, monicRes, targetVarX)

		// alpha = theta - c * beta
		cBeta := MulPoly(repBeta, NewPolyNode([]string{targetVarX}, ast.OrderLex, []ast.Monomial{
			{Coeff: big.NewRat(c, 1), Exponents: []int{0}},
		}))
		thetaPoly := NewPolyNode([]string{targetVarX}, ast.OrderLex, []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{1}},
		})
		repAlpha := SubPoly(thetaPoly, cBeta)
		_, repAlpha, _ = PolyDivRem(repAlpha, monicRes, targetVarX)

		// Transition to Certify state
		if err := fsm.Transition(PrimitiveStateCertify); err != nil {
			return nil, err
		}

		// Verify correctness: g(P_beta(x)) mod m_theta(x) == 0, f(P_alpha(x)) mod m_theta(x) == 0
		compG := substitutePolyIntoPoly(m2, repBeta, targetVarX)
		quoG, remG, errG := PolyDivRem(compG, monicRes, targetVarX)
		if errG != nil || len(remG.Terms) > 0 {
			continue
		}

		compF := substitutePolyIntoPoly(m1, repAlpha, targetVarX)
		quoF, remF, errF := PolyDivRem(compF, monicRes, targetVarX)
		if errF != nil || len(remF.Terms) > 0 {
			continue
		}

		if err := fsm.Transition(PrimitiveStateSuccess); err != nil {
			return nil, err
		}

		return &PrimitiveElementResult{
			MinPolyTheta: monicRes,
			SymbolTheta:  targetVarX,
			C:            c,
			RepAlpha:     repAlpha,
			RepBeta:      repBeta,
			Degree:       DegreeInVar(monicRes, targetVarX),
			MinPolyAlpha: m1,
			MinPolyBeta:  m2,
			QuotientG:    quoG,
			QuotientF:    quoF,
		}, nil
	}

	_ = fsm.Transition(PrimitiveStateFailure)
	return nil, fmt.Errorf("%s", i18n.T("primitive_element.err_failed_to_find_c"))
}

// substitutePolyIntoPoly evaluates outerPoly(innerPoly) and returns polynomial in varName.
func substitutePolyIntoPoly(outer, inner *ast.PolyNode, varName string) *ast.PolyNode {
	res := NewPolyNode([]string{varName}, ast.OrderLex, nil)
	varIdx := -1
	for idx, v := range outer.Vars {
		if v == outer.Vars[0] {
			varIdx = idx
			break
		}
	}

	for _, mon := range outer.Terms {
		exp := 0
		if varIdx >= 0 && varIdx < len(mon.Exponents) {
			exp = mon.Exponents[varIdx]
		}
		// Compute coeff * (inner)^exp
		termVal := NewPolyNode([]string{varName}, ast.OrderLex, []ast.Monomial{
			{Coeff: new(big.Rat).Set(mon.Coeff), Exponents: []int{0}},
		})
		for i := 0; i < exp; i++ {
			termVal = MulPoly(termVal, inner)
		}
		res = AddPoly(res, termVal)
	}
	return res
}

// -------------------------------------------------------------------------
// Recursive Multiple Extension Collapse: Q(alpha_1, ..., alpha_k) -> Q(theta)
// -------------------------------------------------------------------------

// CollapseMultipleExtensions iteratively collapses k extensions into a single simple extension.
func CollapseMultipleExtensions(minPolys []*ast.PolyNode, symbolTheta string) (*MultiPrimitiveElementResult, error) {
	if len(minPolys) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("primitive_element.err_nil_polynomial"))
	}
	if symbolTheta == "" {
		symbolTheta = "theta"
	}

	if len(minPolys) == 1 {
		m := MonicPoly(minPolys[0])
		rep := NewPolyNode([]string{symbolTheta}, ast.OrderLex, []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{1}},
		})
		return &MultiPrimitiveElementResult{
			MinPolyTheta: m,
			SymbolTheta:  symbolTheta,
			Reps:         []*ast.PolyNode{rep},
			Degree:       DegreeInVar(m, m.Vars[0]),
		}, nil
	}

	// Pairwise collapse
	curResult, err := FindPrimitiveElement(minPolys[0], minPolys[1], symbolTheta)
	if err != nil {
		return nil, err
	}

	curMinPoly := curResult.MinPolyTheta
	reps := []*ast.PolyNode{curResult.RepAlpha, curResult.RepBeta}

	for i := 2; i < len(minPolys); i++ {
		nextMinPoly := minPolys[i]
		res, err := FindPrimitiveElement(curMinPoly, nextMinPoly, symbolTheta)
		if err != nil {
			return nil, err
		}

		// Substitute the new theta representation into existing generator representations
		newReps := make([]*ast.PolyNode, len(reps)+1)
		for j, rep := range reps {
			substituted := substitutePolyIntoPoly(rep, res.RepAlpha, symbolTheta)
			_, normSubst, _ := PolyDivRem(substituted, res.MinPolyTheta, symbolTheta)
			newReps[j] = normSubst
		}
		newReps[len(reps)] = res.RepBeta

		reps = newReps
		curMinPoly = res.MinPolyTheta
	}

	return &MultiPrimitiveElementResult{
		MinPolyTheta: curMinPoly,
		SymbolTheta:  symbolTheta,
		Reps:         reps,
		Degree:       DegreeInVar(curMinPoly, symbolTheta),
	}, nil
}

// -------------------------------------------------------------------------
// Expression Extractor & CLI Dispatcher
// -------------------------------------------------------------------------

// extractMinPolyFromNode extracts or derives the minimal polynomial for an algebraic expression node.
func extractMinPolyFromNode(expr ast.Node, varName string) (*ast.PolyNode, error) {
	if varName == "" {
		varName = "x"
	}

	// If already an equation: lhs == rhs or lhs == 0
	if rel, ok := expr.(*ast.RelOpNode); ok && (rel.Op == "==" || rel.Op == "=") {
		diff := &ast.AddNode{Terms: []ast.Node{rel.LHS, &ast.UnaryOpNode{Op: "-", Expr: rel.RHS}}}
		p, err := NodeToPoly(diff, []string{varName}, ast.OrderLex)
		if err == nil && len(p.Terms) > 0 {
			return MonicPoly(p), nil
		}
	}

	// If already a PolyNode
	if poly, ok := expr.(*ast.PolyNode); ok {
		return MonicPoly(poly), nil
	}

	// Rational number q: minimal polynomial x - q
	if rat, ok := expr.(*ast.RationalNode); ok {
		negQ := new(big.Rat).Neg(rat.Val)
		return NewPolyNode([]string{varName}, ast.OrderLex, []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{1}},
			{Coeff: negQ, Exponents: []int{0}},
		}), nil
	}

	// Sqrt(q): minimal polynomial x^2 - q
	if sqrtNode, ok := expr.(*ast.SqrtNode); ok {
		if rat, isRat := sqrtNode.Radicand.(*ast.RationalNode); isRat {
			negQ := new(big.Rat).Neg(rat.Val)
			return NewPolyNode([]string{varName}, ast.OrderLex, []ast.Monomial{
				{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
				{Coeff: negQ, Exponents: []int{0}},
			}), nil
		}
	}

	// Pow(q, 1/n): minimal polynomial x^n - q
	if powNode, ok := expr.(*ast.PowNode); ok {
		if expRat, isExpRat := powNode.Exp.(*ast.RationalNode); isExpRat && expRat.Val.Sign() > 0 {
			if baseRat, isBaseRat := powNode.Base.(*ast.RationalNode); isBaseRat {
				n := int(expRat.Val.Denom().Int64())
				m := int(expRat.Val.Num().Int64())
				if m == 1 && n > 0 {
					negQ := new(big.Rat).Neg(baseRat.Val)
					return NewPolyNode([]string{varName}, ast.OrderLex, []ast.Monomial{
						{Coeff: big.NewRat(1, 1), Exponents: []int{n}},
						{Coeff: negQ, Exponents: []int{0}},
					}), nil
				}
			}
		}
	}

	// AlgebraicNumberNode
	if alg, ok := expr.(*ast.AlgebraicNumberNode); ok && alg.MinPoly != nil {
		return MonicPoly(alg.MinPoly), nil
	}

	// Try NodeToPoly directly
	p, err := NodeToPoly(expr, []string{varName}, ast.OrderLex)
	if err == nil && len(p.Terms) > 0 && DegreeInVar(p, varName) > 0 {
		return MonicPoly(p), nil
	}

	return nil, fmt.Errorf("%s", i18n.T("primitive_element.err_not_algebraic", expr.String()))
}

// EvalToPrimitiveElement handles CLI call to_primitive_element([expr1, expr2, ...]) or (p1, p2).
func EvalToPrimitiveElement(args []ast.Node, env *Env) (ast.Node, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("primitive_element.err_args_count"))
	}

	var targetNodes []ast.Node
	if list, ok := args[0].(*ast.ListNode); ok && len(args) == 1 {
		targetNodes = list.Elements
	} else {
		targetNodes = args
	}

	if len(targetNodes) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("primitive_element.err_args_count"))
	}

	var minPolys []*ast.PolyNode
	for _, node := range targetNodes {
		simplified, err := Eval(node)
		if err != nil {
			simplified = node
		}
		mp, err := extractMinPolyFromNode(simplified, "x")
		if err != nil {
			return nil, err
		}
		minPolys = append(minPolys, mp)
	}

	collapseRes, err := CollapseMultipleExtensions(minPolys, "theta")
	if err != nil {
		return nil, err
	}

	// Construct result list: [theta, minPoly(theta), [rep1, rep2, ...]]
	var repNodes []ast.Node
	for _, rep := range collapseRes.Reps {
		node := PolyToNode(rep)
		if simplified, err := Eval(node); err == nil {
			node = simplified
		}
		repNodes = append(repNodes, node)
	}

	minPolyNode := PolyToNode(collapseRes.MinPolyTheta)
	if simplified, err := Eval(minPolyNode); err == nil {
		minPolyNode = simplified
	}

	resultElements := []ast.Node{
		ast.NewVar(collapseRes.SymbolTheta),
		minPolyNode,
		&ast.ListNode{Elements: repNodes},
	}

	return &ast.ListNode{Elements: resultElements}, nil
}
