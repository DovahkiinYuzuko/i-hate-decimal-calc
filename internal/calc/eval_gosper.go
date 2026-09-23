package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// -------------------------------------------------------------------------
// Gosper's Algorithm & WZ Theory Core
// Reference:
// - R. W. Gosper Jr. (1978) PNAS 75 (1) 40-42
// - M. Petkovsek, H. Wilf, D. Zeilberger "A=B" (1996) Chapters 5 & 6
// - P. Lisonek, P. Paule, V. Strehl (1993) J. Symbolic Computation 16, 243-258
// -------------------------------------------------------------------------

// GosperState refers to the Gosper lifecycle states defined in eval_gosper_fsm.go.

// GosperResult contains the algebraic result of Gosper indefinite summation.
type GosperResult struct {
	FSM            *GosperLifecycleFSM
	Term           ast.Node
	SumVar         string
	A              *univariatePoly
	B              *univariatePoly
	C              *univariatePoly
	Y              *univariatePoly
	DegreeBound    int
	IsSummable     bool
	Antiderivative ast.Node // z_k such that z_{k+1} - z_k = t_k
}

// ExtractHypergeometricRatio extracts r(k) = t_{k+1}/t_k as rational ratio of univariate polynomials in k.
// Returns scalar Z and monic coprime polynomials f(k), g(k) such that r(k) = Z * f(k)/g(k).
func ExtractHypergeometricRatio(term ast.Node, kVar string) (Z *big.Rat, f, g *univariatePoly, err error) {
	if term == nil {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("gosper.err_nil_term"))
	}

	nums, dens := decomposeTermFactors(term)
	totalNum := &univariatePoly{varName: kVar, coeffs: []ast.Node{mustRational(1, 1)}}
	totalDen := &univariatePoly{varName: kVar, coeffs: []ast.Node{mustRational(1, 1)}}
	totalZ := big.NewRat(1, 1)

	// Multiply numerator factor ratios
	for _, fNode := range nums {
		fnNum, fnDen, fnZ, err := factorRatio(fNode, kVar)
		if err != nil {
			return nil, nil, nil, err
		}
		totalNum = polyMul1D(totalNum, fnNum)
		totalDen = polyMul1D(totalDen, fnDen)
		totalZ = new(big.Rat).Mul(totalZ, fnZ)
	}

	// Invert and multiply denominator factor ratios
	for _, fNode := range dens {
		fnNum, fnDen, fnZ, err := factorRatio(fNode, kVar)
		if err != nil {
			return nil, nil, nil, err
		}
		totalNum = polyMul1D(totalNum, fnDen)
		totalDen = polyMul1D(totalDen, fnNum)
		if fnZ.Sign() != 0 {
			totalZ = new(big.Rat).Quo(totalZ, fnZ)
		}
	}

	if isPolyZero(totalDen) {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("gosper.err_term_ratio_denominator_is_zero"))
	}

	// Normalize leading coefficients and reduce gcd
	gcd, gcdErr := polyGCD1D(totalNum, totalDen)
	if gcdErr == nil && gcd.degree() > 0 {
		totalNum = polyDivExact1D(totalNum, gcd)
		totalDen = polyDivExact1D(totalDen, gcd)
	}

	lcNum := totalNum.leadCoeff()
	lcDen := totalDen.leadCoeff()
	ratNum, okRNum := lcNum.(*ast.RationalNode)
	ratDen, okRDen := lcDen.(*ast.RationalNode)

	if !okRNum || !okRDen || ratDen.Val.Sign() == 0 {
		f = totalNum
		g = totalDen
		Z = totalZ
	} else {
		scaleRatio := new(big.Rat).Quo(ratNum.Val, ratDen.Val)
		Z = new(big.Rat).Mul(totalZ, scaleRatio)
		f = monicPoly1D(totalNum)
		g = monicPoly1D(totalDen)
	}

	return Z, f, g, nil
}

// decomposeTermFactors breaks an expression into multiplicative numerator factors and denominator factors.
func decomposeTermFactors(n ast.Node) (num, den []ast.Node) {
	if n == nil {
		return nil, nil
	}
	switch v := n.(type) {
	case *ast.MulNode:
		for _, f := range v.Factors {
			subNum, subDen := decomposeTermFactors(f)
			num = append(num, subNum...)
			den = append(den, subDen...)
		}
		return num, den
	case *ast.PowNode:
		if r, ok := v.Exp.(*ast.RationalNode); ok && r.Val.IsInt() {
			power := r.Val.Num().Int64()
			if power < 0 {
				absPower := -power
				if absPower > 20 {
					absPower = 20
				}
				for i := int64(0); i < absPower; i++ {
					subNum, subDen := decomposeTermFactors(v.Base)
					den = append(den, subNum...)
					num = append(num, subDen...)
				}
				return num, den
			} else if power > 0 {
				if power > 20 {
					power = 20
				}
				for i := int64(0); i < power; i++ {
					subNum, subDen := decomposeTermFactors(v.Base)
					num = append(num, subNum...)
					den = append(den, subDen...)
				}
				return num, den
			}
		}
		return []ast.Node{n}, nil
	case *ast.UnaryOpNode:
		if v.Op == "-" {
			subNum, subDen := decomposeTermFactors(v.Expr)
			return append([]ast.Node{mustRational(-1, 1)}, subNum...), subDen
		}
		return []ast.Node{n}, nil
	default:
		return []ast.Node{n}, nil
	}
}

// factorRatio returns (numPoly, denPoly, constScale, error) for F(k+1) / F(k).
func factorRatio(factor ast.Node, kVar string) (*univariatePoly, *univariatePoly, *big.Rat, error) {
	onePoly := &univariatePoly{varName: kVar, coeffs: []ast.Node{mustRational(1, 1)}}
	oneRat := big.NewRat(1, 1)

	if !containsVar(factor, kVar) {
		return onePoly, onePoly, oneRat, nil
	}

	// 1. Factorial: arg!
	if unary, ok := factor.(*ast.UnaryOpNode); ok && unary.Op == "!" {
		polyArg, okArg := extractPoly(unary.Expr, kVar)
		if !okArg || polyArg.degree() != 1 {
			return nil, nil, nil, fmt.Errorf("%s", i18n.T("gosper.err_factorial_argument_must_be_linear", kVar))
		}
		cNode := polyArg.coeff(1)
		cRat, okC := cNode.(*ast.RationalNode)
		if !okC || !cRat.Val.IsInt() {
			return nil, nil, nil, fmt.Errorf("%s", i18n.T("gosper.err_factorial_coefficient_of", kVar))
		}
		step := cRat.Val.Num().Int64()
		if step == 0 {
			return onePoly, onePoly, oneRat, nil
		}
		if step > 0 {
			resPoly := onePoly
			for i := int64(1); i <= step; i++ {
				termP := polyAdd1D(polyArg, &univariatePoly{varName: kVar, coeffs: []ast.Node{mustRational(i, 1)}})
				resPoly = polyMul1D(resPoly, termP)
			}
			return resPoly, onePoly, oneRat, nil
		} else {
			resPoly := onePoly
			for i := int64(0); i < -step; i++ {
				termP := polySub1D(polyArg, &univariatePoly{varName: kVar, coeffs: []ast.Node{mustRational(i, 1)}})
				resPoly = polyMul1D(resPoly, termP)
			}
			return onePoly, resPoly, oneRat, nil
		}
	}

	// 2. Binomial: comb(n, k)
	if fn, ok := factor.(*ast.FuncNode); ok && (fn.Name == "comb" || fn.Name == "binomial" || fn.Name == "nCr") {
		if len(fn.Args) == 2 {
			nArg := fn.Args[0]
			kArg := fn.Args[1]
			// Shift in k: comb(n, k+1)/comb(n, k) = (n - k) / (k + 1)
			if v, ok := kArg.(*ast.VarNode); ok && v.Name == kVar && !containsVar(nArg, kVar) {
				numP, okN := extractPoly(&ast.AddNode{Terms: []ast.Node{nArg, &ast.UnaryOpNode{Op: "-", Expr: &ast.VarNode{Name: kVar}}}}, kVar)
				denP := &univariatePoly{varName: kVar, coeffs: []ast.Node{mustRational(1, 1), mustRational(1, 1)}} // k + 1
				if okN {
					return numP, denP, oneRat, nil
				}
			}
			// Shift in n: comb(n+1, k)/comb(n, k) = (n + 1) / (n + 1 - k)
			if v, ok := nArg.(*ast.VarNode); ok && v.Name == kVar && !containsVar(kArg, kVar) {
				numP := &univariatePoly{varName: kVar, coeffs: []ast.Node{mustRational(1, 1), mustRational(1, 1)}} // n + 1
				denExpr, _ := simplifyAdd([]ast.Node{&ast.VarNode{Name: kVar}, &ast.UnaryOpNode{Op: "-", Expr: kArg}, mustRational(1, 1)})
				denP, okD := extractPoly(denExpr, kVar)
				if okD {
					return numP, denP, oneRat, nil
				}
			}
		}
	}

	// 3. Exponential: a^(c*k + d)
	if pow, ok := factor.(*ast.PowNode); ok && !containsVar(pow.Base, kVar) && containsVar(pow.Exp, kVar) {
		polyExp, okExp := extractPoly(pow.Exp, kVar)
		if okExp && polyExp.degree() == 1 {
			cNode := polyExp.coeff(1)
			basePowC, err := simplifyPow(pow.Base, cNode)
			if err == nil {
				evalBase, err := Eval(basePowC)
				if err == nil {
					if r, ok := evalBase.(*ast.RationalNode); ok {
						return onePoly, onePoly, r.Val, nil
					}
					return &univariatePoly{varName: kVar, coeffs: []ast.Node{evalBase}}, onePoly, oneRat, nil
				}
			}
		}
	}

	// 4. General Polynomial / Rational factor
	poly, ok := extractPoly(factor, kVar)
	if ok {
		polyNext := shiftPoly1D(poly, 1)
		return polyNext, poly, oneRat, nil
	}

	return nil, nil, nil, fmt.Errorf("%s", i18n.T("gosper.err_unsupported_factor_type_for_hypergeometric", factor.String()))
}

// GosperNormalForm performs the iterative gcd-peel on r(k) = Z * f(k)/g(k)
// to produce polynomials a(k), b(k), c(k) such that:
// r(k) = a(k)/b(k) * c(k+1)/c(k) with gcd(a(k), b(k+h)) = 1 for all non-negative integers h.
func GosperNormalForm(Z *big.Rat, f, g *univariatePoly, kVar string) (a, b, c *univariatePoly, err error) {
	degF := f.degree()
	degG := g.degree()
	hMax := degF * degG
	if degF == 0 || degG == 0 {
		hMax = 0
	}

	p := clonePoly1D(f)
	q := clonePoly1D(g)
	cPoly := &univariatePoly{varName: kVar, coeffs: []ast.Node{mustRational(1, 1)}}

	// Iterative gcd-peel for integer shifts h >= 0
	for h := 0; h <= hMax; h++ {
		qShifted := shiftPoly1D(q, int64(h))
		s, gcdErr := polyGCD1D(p, qShifted)
		if gcdErr == nil && s.degree() > 0 {
			// Divide p by s
			p = polyDivExact1D(p, s)
			// Divide q by s(k - h)
			sNegH := shiftPoly1D(s, int64(-h))
			q = polyDivExact1D(q, sNegH)

			// c(k) = c(k) * prod_{j=1}^h s(k - j)
			for j := 1; j <= h; j++ {
				sShift := shiftPoly1D(s, int64(-j))
				cPoly = polyMul1D(cPoly, sShift)
			}
		}
	}

	// a(k) = Z * p(k), b(k) = q(k)
	aPoly := scalePoly1D(p, Z)
	return aPoly, q, cPoly, nil
}

// ComputeGosperDegreeBound computes the exact degree bound d of y(k) in:
// a(k) y(k+1) - b(k-1) y(k) = c(k).
func ComputeGosperDegreeBound(a, b, c *univariatePoly) (int, bool) {
	D := c.degree()
	m := a.degree()
	l := b.degree()

	lcA := a.leadCoeff()
	lcB := b.leadCoeff()

	lcAEq := lcA.Equal(lcB)

	// Case 1: Non-cancelling branch
	if m != l || !lcAEq {
		maxML := m
		if l > maxML {
			maxML = l
		}
		d := D - maxML
		if d < 0 {
			return -1, false
		}
		return d, true
	}

	// Case 2: Highest degree cancellation branch (m == l and lc(a) == lc(b))
	d1 := D - m + 1

	// Indicial root K0 = slc(b(k-1)) - slc(a(k)) after monic normalization
	slcA := a.secondLeadCoeff()
	bShift := shiftPoly1D(b, -1)
	slcBShift := bShift.secondLeadCoeff()

	// Normalize by leading coefficient
	lcRat, ok := lcA.(*ast.RationalNode)
	if !ok || lcRat.Val.Sign() == 0 {
		// Non-constant parameter: use d1
		if d1 < 0 {
			return -1, false
		}
		return d1, true
	}

	slcARat, okA := slcA.(*ast.RationalNode)
	slcBRat, okB := slcBShift.(*ast.RationalNode)

	dCandidates := []int{d1}

	if okA && okB {
		// K0 = (slcB - slcA) / lcA
		diff := new(big.Rat).Sub(slcBRat.Val, slcARat.Val)
		k0Rat := new(big.Rat).Quo(diff, lcRat.Val)
		if k0Rat.IsInt() {
			k0 := int(k0Rat.Num().Int64())
			if k0 >= 0 {
				dCandidates = append(dCandidates, k0)
			}
		}
	}

	maxD := -1
	for _, cand := range dCandidates {
		if cand > maxD {
			maxD = cand
		}
	}

	if maxD < 0 {
		return -1, false
	}
	return maxD, true
}

// SolveGosperEquation solves a(k) y(k+1) - b(k-1) y(k) = c(k) for y(k) with deg(y) <= d.
// Returns y(k) and verifies identity by resubstitution.
func SolveGosperEquation(a, b, c *univariatePoly, d int, kVar string) (*univariatePoly, error) {
	if d < 0 {
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_degree_bound_is_negative_not"))
	}

	bShift := shiftPoly1D(b, -1)

	// We set up undetermined coefficients u_0, u_1, ..., u_d for y(k) = sum_{i=0}^d u_i k^i.
	// Operator on basis k^i:
	// L(k^i) = a(k) * (k+1)^i - b(k-1) * k^i
	basisImages := make([]*univariatePoly, d+1)
	maxImageDeg := 0
	for i := 0; i <= d; i++ {
		kPowI := monomialPoly1D(kVar, i)
		kPlus1PowI := shiftPoly1D(kPowI, 1)

		term1 := polyMul1D(a, kPlus1PowI)
		term2 := polyMul1D(bShift, kPowI)
		diff := polySub1D(term1, term2)
		basisImages[i] = diff
		if diff.degree() > maxImageDeg {
			maxImageDeg = diff.degree()
		}
	}

	numEqs := maxImageDeg + 1
	if c.degree()+1 > numEqs {
		numEqs = c.degree() + 1
	}

	// Matrix M (numEqs x (d+1)) and RHS vector C (numEqs)
	// Equation for coefficient of k^deg: sum_{j=0}^d M[deg][j] * u_j = C[deg]
	M := make([][]*big.Rat, numEqs)
	C := make([]*big.Rat, numEqs)

	for deg := 0; deg < numEqs; deg++ {
		M[deg] = make([]*big.Rat, d+1)
		for j := 0; j <= d; j++ {
			coeffNode := basisImages[j].coeff(deg)
			if r, ok := coeffNode.(*ast.RationalNode); ok {
				M[deg][j] = new(big.Rat).Set(r.Val)
			} else {
				M[deg][j] = big.NewRat(0, 1)
			}
		}

		cCoeffNode := c.coeff(deg)
		if r, ok := cCoeffNode.(*ast.RationalNode); ok {
			C[deg] = new(big.Rat).Set(r.Val)
		} else {
			C[deg] = big.NewRat(0, 1)
		}
	}

	// Solve overdetermined linear system using Gaussian elimination with consistency check
	uSol, err := solveRationalLinearSystem(M, C, numEqs, d+1)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_inconsistent_linear_system_not_gosper", err))
	}

	// Build y(k)
	yCoeffs := make([]ast.Node, d+1)
	for i := 0; i <= d; i++ {
		yCoeffs[i] = mustRationalBig(uSol[i])
	}
	y := &univariatePoly{varName: kVar, coeffs: yCoeffs}

	// Resubstitution Verification: a(k) y(k+1) - b(k-1) y(k) == c(k)
	yPlus1 := shiftPoly1D(y, 1)
	lhs1 := polyMul1D(a, yPlus1)
	lhs2 := polyMul1D(bShift, y)
	lhsDiff := polySub1D(lhs1, lhs2)
	checkDiff := polySub1D(lhsDiff, c)

	if !isPolyZero(checkDiff) {
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_resubstitution_verification_failed_not_gosper"))
	}

	return y, nil
}

// GosperIndefiniteSum computes the indefinite sum (antidifference) z_k such that z_{k+1} - z_k = t_k.
func GosperIndefiniteSum(term ast.Node, kVar string) (ast.Node, error) {
	fsm := NewGosperLifecycleFSM()

	// 1. Extract ratio
	Z, f, g, err := ExtractHypergeometricRatio(term, kVar)
	if err != nil {
		_ = fsm.TransitionTo(GosperStateNotSummable)
		return nil, err
	}
	_ = fsm.TransitionTo(GosperStateRatioExtracted)

	// 2. Normal form
	a, b, c, err := GosperNormalForm(Z, f, g, kVar)
	if err != nil {
		_ = fsm.TransitionTo(GosperStateNotSummable)
		return nil, err
	}
	_ = fsm.TransitionTo(GosperStateNormalized)

	// 3. Degree bound
	d, ok := ComputeGosperDegreeBound(a, b, c)
	if !ok {
		_ = fsm.TransitionTo(GosperStateNotSummable)
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_term_is_not_gosper_summable"))
	}
	_ = fsm.TransitionTo(GosperStateDegreeBounded)

	// 4. Solve polynomial equation
	y, err := SolveGosperEquation(a, b, c, d, kVar)
	if err != nil {
		_ = fsm.TransitionTo(GosperStateNotSummable)
		return nil, err
	}
	_ = fsm.TransitionTo(GosperStateSolved)

	// 5. Construct antiderivative z_k = (b(k-1) * y(k) / c(k)) * t_k
	bShift := shiftPoly1D(b, -1)
	numPoly := polyMul1D(bShift, y)

	// Cancel common factors between numPoly and c
	gcdPoly, err := polyGCD1D(numPoly, c)
	if err == nil && gcdPoly.degree() > 0 {
		numPoly = polyDivExact1D(numPoly, gcdPoly)
		c = polyDivExact1D(c, gcdPoly)
	}

	numNode := numPoly.toNode()
	denNode := c.toNode()

	var multNode ast.Node
	if isOne(denNode) {
		multNode = numNode
	} else {
		invDen, _ := simplifyPow(denNode, mustRational(-1, 1))
		multNode, _ = simplifyMul([]ast.Node{numNode, invDen})
	}

	zNode, err := simplifyMul([]ast.Node{multNode, term})
	if err != nil {
		return nil, err
	}

	return Eval(expandNode(zNode))
}

// GosperDefiniteSum computes sum_{k=start}^{end} t_k via discrete Newton-Leibniz: z_{end+1} - z_{start}.
func GosperDefiniteSum(term ast.Node, kVar string, startNode, endNode ast.Node) (ast.Node, error) {
	zK, err := GosperIndefiniteSum(term, kVar)
	if err != nil {
		return nil, err
	}

	// z_{end+1}
	endPlus1, _ := simplifyAdd([]ast.Node{endNode, mustRational(1, 1)})
	subEnd := NewEnv()
	subEnd.Set(kVar, endPlus1)
	zEndPlus1, err := substituteVariables(zK, subEnd)
	if err != nil {
		return nil, err
	}

	// z_{start}
	subStart := NewEnv()
	subStart.Set(kVar, startNode)
	zStart, err := substituteVariables(zK, subStart)
	if err != nil {
		return nil, err
	}

	negZStart, _ := simplifyMul([]ast.Node{mustRational(-1, 1), zStart})
	diffNode, err := simplifyAdd([]ast.Node{zEndPlus1, negZStart})
	if err != nil {
		return nil, err
	}

	return Eval(expandNode(diffNode))
}

// GenerateWZCertificate computes the Wilf-Zeilberger rational certificate R(n, k) = G(n, k) / F(n, k)
// for a hypergeometric summand F(n, k) satisfying F(n+1, k) - F(n, k) = G(n, k+1) - G(n, k).
func GenerateWZCertificate(term ast.Node, nVar, kVar string) (ast.Node, error) {
	// 1. Extract r_n(n, k) = F(n+1, k) / F(n, k)
	Zn, fn, gn, err := ExtractHypergeometricRatio(term, nVar)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_term_is_not_hypergeometric_in", nVar, err))
	}

	// 2. r_n - 1 = (Zn * fn - gn) / gn
	scaledFn := scalePoly1D(fn, Zn)
	pNum := polySub1D(scaledFn, gn)
	pDen := gn

	// 3. Extract r_k(n, k) = F(n, k+1) / F(n, k)
	Zk, fk, gk, err := ExtractHypergeometricRatio(term, kVar)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_term_is_not_hypergeometric_in", kVar, err))
	}

	// 4. Combined ratio of Delta_n F with respect to k:
	// r_Delta(k) = (pNum(k+1)/pNum(k)) * (pDen(k)/pDen(k+1)) * (Zk * fk(k)/gk(k))
	pNumK, okNumK := extractPoly(pNum.toNode(), kVar)
	pDenK, okDenK := extractPoly(pDen.toNode(), kVar)
	if !okNumK || !okDenK {
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_difference_factor_is_not_polynomial", kVar))
	}

	pNumKShift := shiftPoly1D(pNumK, 1)
	pDenKShift := shiftPoly1D(pDenK, 1)

	// totalNum = pNumKShift * pDenK * fk
	totalNum := polyMul1D(polyMul1D(pNumKShift, pDenK), fk)
	// totalDen = pNumK * pDenKShift * gk
	totalDen := polyMul1D(polyMul1D(pNumK, pDenKShift), gk)
	totalZ := Zk

	// Reduce GCD
	gcd, gcdErr := polyGCD1D(totalNum, totalDen)
	if gcdErr == nil && gcd.degree() > 0 {
		totalNum = polyDivExact1D(totalNum, gcd)
		totalDen = polyDivExact1D(totalDen, gcd)
	}

	fNorm := monicPoly1D(totalNum)
	gNorm := monicPoly1D(totalDen)

	lcNum := totalNum.leadCoeff()
	lcDen := totalDen.leadCoeff()
	if ratNum, ok1 := lcNum.(*ast.RationalNode); ok1 {
		if ratDen, ok2 := lcDen.(*ast.RationalNode); ok2 && ratDen.Val.Sign() != 0 {
			scaleRatio := new(big.Rat).Quo(ratNum.Val, ratDen.Val)
			totalZ = new(big.Rat).Mul(totalZ, scaleRatio)
		}
	}

	// 5. Gosper normal form on r_Delta(k)
	a, b, c, err := GosperNormalForm(totalZ, fNorm, gNorm, kVar)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_gosper_normal_form_failed_on", err))
	}

	d, ok := ComputeGosperDegreeBound(a, b, c)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_summand_does_not_admit_a"))
	}

	y, err := SolveGosperEquation(a, b, c, d, kVar)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("gosper.err_summand_does_not_admit_a_1", err))
	}

	// 6. R(n, k) = (b(k-1) * y(k) / c(k)) * (pNum(k) / pDen(k))
	bShift := shiftPoly1D(b, -1)
	certNumPoly := polyMul1D(polyMul1D(bShift, y), pNumK)
	certDenPoly := polyMul1D(c, pDenK)

	gcdCert, err := polyGCD1D(certNumPoly, certDenPoly)
	if err == nil && gcdCert.degree() > 0 {
		certNumPoly = polyDivExact1D(certNumPoly, gcdCert)
		certDenPoly = polyDivExact1D(certDenPoly, gcdCert)
	}

	certNumNode := certNumPoly.toNode()
	certDenNode := certDenPoly.toNode()

	if isOne(certDenNode) {
		return Eval(expandNode(certNumNode))
	}

	invDen, _ := simplifyPow(certDenNode, mustRational(-1, 1))
	resNode, _ := simplifyMul([]ast.Node{certNumNode, invDen})
	return Eval(expandNode(resNode))
}

func mustRationalBig(r *big.Rat) ast.Node {
	return ast.NewRationalFromBigRat(r)
}

func substituteVariables(n ast.Node, env *Env) (ast.Node, error) {
	if env == nil {
		return n, nil
	}
	curr := n
	for k, val := range env.All() {
		curr = substituteVar(curr, k, val)
	}
	return curr, nil
}

// -------------------------------------------------------------------------
// Helper Polynomial Arithmetic on univariatePoly
// -------------------------------------------------------------------------

func clonePoly1D(p *univariatePoly) *univariatePoly {
	coeffs := make([]ast.Node, len(p.coeffs))
	copy(coeffs, p.coeffs)
	return &univariatePoly{varName: p.varName, coeffs: coeffs}
}

func (p *univariatePoly) coeff(deg int) ast.Node {
	if deg < 0 || deg >= len(p.coeffs) {
		return mustRational(0, 1)
	}
	return p.coeffs[deg]
}

func (p *univariatePoly) secondLeadCoeff() ast.Node {
	deg := p.degree()
	if deg < 1 {
		return mustRational(0, 1)
	}
	return p.coeffs[deg-1]
}

func monomialPoly1D(varName string, deg int) *univariatePoly {
	coeffs := make([]ast.Node, deg+1)
	for i := 0; i < deg; i++ {
		coeffs[i] = mustRational(0, 1)
	}
	coeffs[deg] = mustRational(1, 1)
	return &univariatePoly{varName: varName, coeffs: coeffs}
}

func scalePoly1D(p *univariatePoly, s *big.Rat) *univariatePoly {
	res := make([]ast.Node, len(p.coeffs))
	scaleNode := mustRationalBig(s)
	for i, c := range p.coeffs {
		mul, _ := simplifyMul([]ast.Node{scaleNode, c})
		res[i] = mul
	}
	return &univariatePoly{varName: p.varName, coeffs: res}
}

func monicPoly1D(p *univariatePoly) *univariatePoly {
	lc := p.leadCoeff()
	if rat, ok := lc.(*ast.RationalNode); ok && rat.Val.Sign() != 0 {
		inv := new(big.Rat).Inv(rat.Val)
		return scalePoly1D(p, inv)
	}
	return clonePoly1D(p)
}

func shiftPoly1D(p *univariatePoly, h int64) *univariatePoly {
	if h == 0 || len(p.coeffs) == 0 {
		return clonePoly1D(p)
	}
	// Evaluate p(k + h)
	subEnv := NewEnv()
	kPlusH, _ := simplifyAdd([]ast.Node{&ast.VarNode{Name: p.varName}, mustRational(h, 1)})
	subEnv.Set(p.varName, kPlusH)
	nodeForm := p.toNode()
	shiftedNode, _ := substituteVariables(nodeForm, subEnv)
	evalNode, _ := Eval(expandNode(shiftedNode))
	res, ok := extractPoly(evalNode, p.varName)
	if !ok {
		return clonePoly1D(p)
	}
	return res
}

func polyAdd1D(p1, p2 *univariatePoly) *univariatePoly {
	maxLen := len(p1.coeffs)
	if len(p2.coeffs) > maxLen {
		maxLen = len(p2.coeffs)
	}
	coeffs := make([]ast.Node, maxLen)
	for i := 0; i < maxLen; i++ {
		c1 := p1.coeff(i)
		c2 := p2.coeff(i)
		add, _ := simplifyAdd([]ast.Node{c1, c2})
		coeffs[i] = add
	}
	// Trim leading zeroes
	for len(coeffs) > 1 && isZero(coeffs[len(coeffs)-1]) {
		coeffs = coeffs[:len(coeffs)-1]
	}
	return &univariatePoly{varName: p1.varName, coeffs: coeffs}
}

func polySub1D(p1, p2 *univariatePoly) *univariatePoly {
	maxLen := len(p1.coeffs)
	if len(p2.coeffs) > maxLen {
		maxLen = len(p2.coeffs)
	}
	coeffs := make([]ast.Node, maxLen)
	for i := 0; i < maxLen; i++ {
		c1 := p1.coeff(i)
		c2 := p2.coeff(i)
		negC2, _ := simplifyMul([]ast.Node{mustRational(-1, 1), c2})
		sub, _ := simplifyAdd([]ast.Node{c1, negC2})
		coeffs[i] = sub
	}
	for len(coeffs) > 1 && isZero(coeffs[len(coeffs)-1]) {
		coeffs = coeffs[:len(coeffs)-1]
	}
	return &univariatePoly{varName: p1.varName, coeffs: coeffs}
}

func polyMul1D(p1, p2 *univariatePoly) *univariatePoly {
	if isPolyZero(p1) || isPolyZero(p2) {
		return &univariatePoly{varName: p1.varName, coeffs: []ast.Node{mustRational(0, 1)}}
	}
	deg1 := p1.degree()
	deg2 := p2.degree()
	resCoeffs := make([]ast.Node, deg1+deg2+1)
	for i := range resCoeffs {
		resCoeffs[i] = mustRational(0, 1)
	}

	for i, c1 := range p1.coeffs {
		if isZero(c1) {
			continue
		}
		for j, c2 := range p2.coeffs {
			if isZero(c2) {
				continue
			}
			prod, _ := simplifyMul([]ast.Node{c1, c2})
			add, _ := simplifyAdd([]ast.Node{resCoeffs[i+j], prod})
			resCoeffs[i+j] = add
		}
	}
	for len(resCoeffs) > 1 && isZero(resCoeffs[len(resCoeffs)-1]) {
		resCoeffs = resCoeffs[:len(resCoeffs)-1]
	}
	return &univariatePoly{varName: p1.varName, coeffs: resCoeffs}
}

func polyDivExact1D(dividend, divisor *univariatePoly) *univariatePoly {
	if isPolyZero(divisor) {
		return dividend
	}
	if dividend.degree() < divisor.degree() {
		return &univariatePoly{varName: dividend.varName, coeffs: []ast.Node{mustRational(0, 1)}}
	}

	rem := clonePoly1D(dividend)
	degDiv := divisor.degree()
	lcDiv := divisor.leadCoeff()

	invLcDiv, _ := simplifyPow(lcDiv, mustRational(-1, 1))

	quotCoeffs := make([]ast.Node, dividend.degree()-degDiv+1)
	for i := range quotCoeffs {
		quotCoeffs[i] = mustRational(0, 1)
	}

	for rem.degree() >= degDiv && !isPolyZero(rem) {
		degDiff := rem.degree() - degDiv
		lcRem := rem.leadCoeff()
		coeff, _ := simplifyMul([]ast.Node{lcRem, invLcDiv})
		quotCoeffs[degDiff] = coeff

		subTerm := monomialPoly1D(dividend.varName, degDiff)
		subTerm = scalePolyNode1D(subTerm, coeff)
		subPoly := polyMul1D(divisor, subTerm)
		rem = polySub1D(rem, subPoly)
	}

	for len(quotCoeffs) > 1 && isZero(quotCoeffs[len(quotCoeffs)-1]) {
		quotCoeffs = quotCoeffs[:len(quotCoeffs)-1]
	}
	return &univariatePoly{varName: dividend.varName, coeffs: quotCoeffs}
}

func scalePolyNode1D(p *univariatePoly, factor ast.Node) *univariatePoly {
	coeffs := make([]ast.Node, len(p.coeffs))
	for i, c := range p.coeffs {
		mul, _ := simplifyMul([]ast.Node{factor, c})
		coeffs[i] = mul
	}
	return &univariatePoly{varName: p.varName, coeffs: coeffs}
}

func polyGCD1D(p1, p2 *univariatePoly) (*univariatePoly, error) {
	node1 := p1.toNode()
	node2 := p2.toNode()
	gcdNode, err := EvalPolyGCD(node1, node2, p1.varName, nil)
	if err != nil {
		return nil, err
	}
	polyGCD, ok := extractPoly(gcdNode, p1.varName)
	if !ok {
		return &univariatePoly{varName: p1.varName, coeffs: []ast.Node{mustRational(1, 1)}}, nil
	}
	return monicPoly1D(polyGCD), nil
}

func solveRationalLinearSystem(M [][]*big.Rat, C []*big.Rat, m, n int) ([]*big.Rat, error) {
	// Gaussian elimination on augmented matrix [M | C]
	A := make([][]*big.Rat, m)
	for i := 0; i < m; i++ {
		A[i] = make([]*big.Rat, n+1)
		for j := 0; j < n; j++ {
			A[i][j] = new(big.Rat).Set(M[i][j])
		}
		A[i][n] = new(big.Rat).Set(C[i])
	}

	pivotRow := 0
	for col := 0; col < n && pivotRow < m; col++ {
		// Find non-zero pivot
		sel := -1
		for r := pivotRow; r < m; r++ {
			if A[r][col].Sign() != 0 {
				sel = r
				break
			}
		}
		if sel == -1 {
			continue
		}

		A[pivotRow], A[sel] = A[sel], A[pivotRow]
		pivotVal := new(big.Rat).Set(A[pivotRow][col])

		// Normalize pivot row
		for c := col; c <= n; c++ {
			A[pivotRow][c].Quo(A[pivotRow][c], pivotVal)
		}

		// Eliminate below and above
		for r := 0; r < m; r++ {
			if r != pivotRow && A[r][col].Sign() != 0 {
				factor := new(big.Rat).Set(A[r][col])
				for c := col; c <= n; c++ {
					sub := new(big.Rat).Mul(factor, A[pivotRow][c])
					A[r][c].Sub(A[r][c], sub)
				}
			}
		}
		pivotRow++
	}

	// Check consistency: all rows from pivotRow to m must have A[r][n] == 0
	for r := pivotRow; r < m; r++ {
		if A[r][n].Sign() != 0 {
			return nil, fmt.Errorf("%s", i18n.T("gosper.err_inconsistent_system_at_row", r))
		}
	}

	// Back-substitute solution
	sol := make([]*big.Rat, n)
	for j := 0; j < n; j++ {
		sol[j] = big.NewRat(0, 1)
	}

	for r := 0; r < pivotRow; r++ {
		leadCol := -1
		for c := 0; c < n; c++ {
			if A[r][c].Sign() != 0 {
				leadCol = c
				break
			}
		}
		if leadCol != -1 {
			sol[leadCol] = new(big.Rat).Set(A[r][n])
		}
	}

	return sol, nil
}
