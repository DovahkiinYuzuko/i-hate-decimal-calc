package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// AlgebraicReal represents an isolated real root of an irreducible integer polynomial Poly(x).
type AlgebraicReal struct {
	Poly []*big.Int // Ascending order: coeffs[0] + coeffs[1]*x + ... + coeffs[d]*x^d
	Low  *big.Rat   // Lower bound of isolating interval
	High *big.Rat   // Upper bound of isolating interval
}

// NewAlgebraicRealFromRat creates an AlgebraicReal for a rational number r (poly: den*x - num = 0).
func NewAlgebraicRealFromRat(r *big.Rat) *AlgebraicReal {
	num := r.Num()
	den := r.Denom()
	// den * x - num = 0 -> poly: [-num, den]
	poly := []*big.Int{new(big.Int).Neg(num), new(big.Int).Set(den)}
	return &AlgebraicReal{
		Poly: poly,
		Low:  new(big.Rat).Set(r),
		High: new(big.Rat).Set(r),
	}
}

// AlgIsZero checks if an AlgebraicReal is algebraically zero.
func AlgIsZero(a *AlgebraicReal) bool {
	if a == nil {
		return false
	}
	if a.Low.Sign() == 0 && a.High.Sign() == 0 {
		return true
	}
	// If 0 is in [Low, High] and poly(0) == 0:
	if a.Low.Sign() <= 0 && a.High.Sign() >= 0 {
		if len(a.Poly) > 0 && (a.Poly[0] == nil || a.Poly[0].Sign() == 0) {
			return true
		}
	}
	return false
}

// AlgRefineInterval refines isolating interval of a by bisection.
func AlgRefineInterval(a *AlgebraicReal) {
	if a.Low.Cmp(a.High) == 0 {
		return
	}
	mid := new(big.Rat).Add(a.Low, a.High)
	mid.Mul(mid, big.NewRat(1, 2))

	valMid := evalIntPolyAtRat(a.Poly, mid)
	if valMid.Sign() == 0 {
		a.Low.Set(mid)
		a.High.Set(mid)
		return
	}

	valLow := evalIntPolyAtRat(a.Poly, a.Low)
	if valMid.Sign() == valLow.Sign() {
		a.Low.Set(mid)
	} else {
		a.High.Set(mid)
	}
}

// AlgCompare compares two AlgebraicReal numbers: -1 for a < b, 0 for a == b, 1 for a > b.
func AlgCompare(a, b *AlgebraicReal) int {
	if a == b {
		return 0
	}
	if a == nil || b == nil {
		return 0
	}

	// 1. Check if both are exact same polynomial and intervals overlap
	if len(a.Poly) == len(b.Poly) {
		same := true
		for i := range a.Poly {
			if a.Poly[i].Cmp(b.Poly[i]) != 0 {
				same = false
				break
			}
		}
		if same {
			// If intervals overlap, they represent the same isolated root
			if a.High.Cmp(b.Low) >= 0 && b.High.Cmp(a.Low) >= 0 {
				return 0
			}
		}
	}

	// 2. Refine until intervals are disjoint
	maxIter := 50
	for i := 0; i < maxIter; i++ {
		if a.High.Cmp(b.Low) < 0 {
			return -1
		}
		if a.Low.Cmp(b.High) > 0 {
			return 1
		}
		AlgRefineInterval(a)
		AlgRefineInterval(b)
	}

	// Fallback compare centers if tied
	midA := new(big.Rat).Add(a.Low, a.High)
	midB := new(big.Rat).Add(b.Low, b.High)
	return midA.Cmp(midB)
}

// bareissDeterminant computes exact determinant of integer matrix using Bareiss algorithm.
func bareissDeterminant(matrix [][]*big.Int) *big.Int {
	n := len(matrix)
	if n == 0 {
		return big.NewInt(1)
	}
	if n == 1 {
		return new(big.Int).Set(matrix[0][0])
	}

	// Make a copy
	mat := make([][]*big.Int, n)
	for i := range matrix {
		mat[i] = make([]*big.Int, n)
		for j := range matrix[i] {
			mat[i][j] = new(big.Int).Set(matrix[i][j])
		}
	}

	sign := 1
	prevPivot := big.NewInt(1)

	for k := 0; k < n-1; k++ {
		// Find pivot
		if mat[k][k].Sign() == 0 {
			pivotRow := -1
			for i := k + 1; i < n; i++ {
				if mat[i][k].Sign() != 0 {
					pivotRow = i
					break
				}
			}
			if pivotRow == -1 {
				return big.NewInt(0)
			}
			// Swap rows k and pivotRow
			mat[k], mat[pivotRow] = mat[pivotRow], mat[k]
			sign = -sign
		}

		pivot := mat[k][k]
		for i := k + 1; i < n; i++ {
			for j := k + 1; j < n; j++ {
				term1 := new(big.Int).Mul(mat[k][k], mat[i][j])
				term2 := new(big.Int).Mul(mat[i][k], mat[k][j])
				num := new(big.Int).Sub(term1, term2)
				mat[i][j].Div(num, prevPivot)
			}
		}
		prevPivot = pivot
	}

	res := mat[n-1][n-1]
	if sign < 0 {
		res.Neg(res)
	}
	return res
}

// sylvesterResultant computes Res_y(P(y), Q(y)) using Sylvester matrix determinant.
// P and Q are in ascending order.
func sylvesterResultant(p, q []*big.Int) *big.Int {
	// Trim leading zeros
	degP := len(p) - 1
	for degP >= 0 && (p[degP] == nil || p[degP].Sign() == 0) {
		degP--
	}
	degQ := len(q) - 1
	for degQ >= 0 && (q[degQ] == nil || q[degQ].Sign() == 0) {
		degQ--
	}

	if degP < 0 || degQ < 0 {
		return big.NewInt(0)
	}
	if degP == 0 && degQ == 0 {
		return big.NewInt(1)
	}
	if degP == 0 {
		// Res(c, Q) = c^degQ
		return new(big.Int).Exp(p[0], big.NewInt(int64(degQ)), nil)
	}
	if degQ == 0 {
		// Res(P, c) = c^degP
		return new(big.Int).Exp(q[0], big.NewInt(int64(degP)), nil)
	}

	n := degP + degQ
	mat := make([][]*big.Int, n)
	for i := range mat {
		mat[i] = make([]*big.Int, n)
		for j := range mat[i] {
			mat[i][j] = big.NewInt(0)
		}
	}

	// P has degree degP: p[degP]*y^degP + ... + p[0]
	// First degQ rows for P
	for i := 0; i < degQ; i++ {
		for j := 0; j <= degP; j++ {
			mat[i][i+j] = p[degP-j]
		}
	}

	// Next degP rows for Q
	for i := 0; i < degP; i++ {
		for j := 0; j <= degQ; j++ {
			mat[degQ+i][i+j] = q[degQ-j]
		}
	}

	return bareissDeterminant(mat)
}

// lagrangeInterpolate reconstructs integer polynomial from points (x_i, y_i)
// using exact rational arithmetic to avoid integer division truncation errors.
func lagrangeInterpolate(xs, ys []*big.Int) []*big.Int {
	n := len(xs)
	resRat := make([]*big.Rat, n)
	for i := range resRat {
		resRat[i] = big.NewRat(0, 1)
	}

	for i := 0; i < n; i++ {
		basis := []*big.Rat{big.NewRat(1, 1)}
		denom := big.NewInt(1)

		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			diff := new(big.Int).Sub(xs[i], xs[j])
			denom.Mul(denom, diff)

			newBasis := make([]*big.Rat, len(basis)+1)
			for k := range newBasis {
				newBasis[k] = big.NewRat(0, 1)
			}
			negXj := new(big.Rat).SetInt(new(big.Int).Neg(xs[j]))
			oneRat := big.NewRat(1, 1)

			for k, c := range basis {
				termConst := new(big.Rat).Mul(c, negXj)
				newBasis[k].Add(newBasis[k], termConst)

				termX := new(big.Rat).Mul(c, oneRat)
				newBasis[k+1].Add(newBasis[k+1], termX)
			}
			basis = newBasis
		}

		factor := new(big.Rat).SetFrac(ys[i], denom)
		for k, c := range basis {
			term := new(big.Rat).Mul(c, factor)
			resRat[k].Add(resRat[k], term)
		}
	}

	res := make([]*big.Int, n)
	for i := range resRat {
		res[i] = new(big.Int).Set(resRat[i].Num())
	}

	for len(res) > 1 && res[len(res)-1].Sign() == 0 {
		res = res[:len(res)-1]
	}
	return res
}

// intCoeffsToUnivariatePoly converts []*big.Int to univariatePoly.
func intCoeffsToUnivariatePoly(coeffs []*big.Int, varName string) *univariatePoly {
	cNodes := make([]Node, len(coeffs))
	for i, c := range coeffs {
		if c != nil {
			cNodes[i] = &RationalNode{Val: new(big.Rat).SetInt(c)}
		} else {
			cNodes[i] = mustRational(0, 1)
		}
	}
	return &univariatePoly{varName: varName, coeffs: cNodes}
}

// univariatePolyToIntCoeffs converts univariatePoly to []*big.Int with integer scaling.
func univariatePolyToIntCoeffs(p *univariatePoly) []*big.Int {
	deg := p.degree()
	if deg < 0 {
		return []*big.Int{big.NewInt(0)}
	}
	dens := make([]*big.Int, deg+1)
	nums := make([]*big.Int, deg+1)
	for i := 0; i <= deg; i++ {
		if r, ok := p.coeffs[i].(*RationalNode); ok && r.Val != nil {
			nums[i] = new(big.Int).Set(r.Val.Num())
			dens[i] = new(big.Int).Set(r.Val.Denom())
		} else {
			nums[i] = big.NewInt(0)
			dens[i] = big.NewInt(1)
		}
	}
	lcm := big.NewInt(1)
	for _, d := range dens {
		gcd := new(big.Int).GCD(nil, nil, lcm, d)
		lcm.Mul(lcm, d)
		lcm.Div(lcm, gcd)
	}
	res := make([]*big.Int, deg+1)
	for i := 0; i <= deg; i++ {
		scale := new(big.Int).Div(lcm, dens[i])
		res[i] = new(big.Int).Mul(nums[i], scale)
	}
	return polyMakePrimitive(res)
}

// polyContentInt computes GCD of all coefficients.
func polyContentInt(poly []*big.Int) *big.Int {
	gcd := big.NewInt(0)
	for _, c := range poly {
		if c != nil && c.Sign() != 0 {
			gcd.GCD(nil, nil, gcd, new(big.Int).Abs(c))
		}
	}
	if gcd.Sign() == 0 {
		return big.NewInt(1)
	}
	return gcd
}

// polyMakePrimitive divides coefficients by content and ensures positive leading coefficient.
func polyMakePrimitive(poly []*big.Int) []*big.Int {
	for len(poly) > 1 && poly[len(poly)-1].Sign() == 0 {
		poly = poly[:len(poly)-1]
	}
	c := polyContentInt(poly)
	res := make([]*big.Int, len(poly))
	for i := range poly {
		res[i] = new(big.Int).Div(poly[i], c)
	}
	if len(res) > 0 && res[len(res)-1].Sign() < 0 {
		for i := range res {
			res[i].Neg(res[i])
		}
	}
	return res
}

// squareFreePart computes primitive square-free part of integer polynomial: P / gcd(P, P').
func squareFreePart(poly []*big.Int) []*big.Int {
	pPrim := polyMakePrimitive(poly)
	polyP := intCoeffsToUnivariatePoly(pPrim, "x")
	if polyP.degree() <= 1 {
		return pPrim
	}

	dPoly := polyDerivative1D(polyP)
	gcdNode, err := EvalPolyGCD(polyP.toNode(), dPoly.toNode(), "x", nil)
	if err == nil && gcdNode != nil {
		gPoly, okG := extractPoly(gcdNode, "x")
		if okG && gPoly.degree() > 0 {
			quot, _, okDiv := polyDivide(polyP, gPoly)
			if okDiv && !isPolyZero(quot) {
				return univariatePolyToIntCoeffs(quot)
			}
		}
	}

	return pPrim
}

// AlgAdd computes a + b.
func AlgAdd(a, b *AlgebraicReal) *AlgebraicReal {
	if a == nil || b == nil {
		return nil
	}
	if AlgIsZero(a) {
		return b
	}
	if AlgIsZero(b) {
		return a
	}

	degA := len(a.Poly) - 1
	degB := len(b.Poly) - 1
	targetDeg := degA * degB
	if targetDeg == 0 {
		targetDeg = 1
	}

	numPoints := targetDeg + 1
	xs := make([]*big.Int, numPoints)
	ys := make([]*big.Int, numPoints)

	for k := 0; k < numPoints; k++ {
		xs[k] = big.NewInt(int64(k))
		// A_k(y) = A(k - y)
		// A(x) = sum a_i x^i -> A(k - y) = sum a_i (k - y)^i
		ak := make([]*big.Int, degA+1)
		for i := range ak {
			ak[i] = big.NewInt(0)
		}
		for i := 0; i <= degA; i++ {
			c := a.Poly[i]
			if c == nil || c.Sign() == 0 {
				continue
			}
			// Expand (k - y)^i
			binom := make([]*big.Int, i+1)
			for j := 0; j <= i; j++ {
				binom[j] = big.NewInt(0)
			}
			binom[0] = big.NewInt(1)
			for step := 0; step < i; step++ {
				next := make([]*big.Int, step+2)
				for j := range next {
					next[j] = big.NewInt(0)
				}
				for j := 0; j <= step; j++ {
					// (k - y) * binom[j] y^j = k*binom[j] y^j - binom[j] y^(j+1)
					termK := new(big.Int).Mul(xs[k], binom[j])
					next[j].Add(next[j], termK)
					next[j+1].Sub(next[j+1], binom[j])
				}
				binom = next
			}
			for j := 0; j <= i; j++ {
				term := new(big.Int).Mul(c, binom[j])
				ak[j].Add(ak[j], term)
			}
		}

		ys[k] = sylvesterResultant(ak, b.Poly)
	}

	polySum := lagrangeInterpolate(xs, ys)
	polySum = squareFreePart(polySum)

	// Isolate roots of composite polynomial
	intervals := IsolateRealRootsVAS(polySum)

	// Target interval
	initLow := new(big.Rat).Add(a.Low, b.Low)
	initHigh := new(big.Rat).Add(a.High, b.High)

	// Match root: choose interval with strictly positive overlap
	var bestMatch *VASInterval
	maxOverlap := big.NewRat(0, 1)

	for _, iv := range intervals {
		start := iv.Low
		if initLow.Cmp(start) > 0 {
			start = initLow
		}
		end := iv.High
		if initHigh.Cmp(end) < 0 {
			end = initHigh
		}

		// Check if exact rational root falls within [initLow, initHigh]
		if iv.IsExact && iv.ExactVal != nil {
			if iv.ExactVal.Cmp(initLow) >= 0 && iv.ExactVal.Cmp(initHigh) <= 0 {
				match := iv
				bestMatch = &match
				break
			}
		}

		if end.Cmp(start) > 0 {
			overlap := new(big.Rat).Sub(end, start)
			if overlap.Cmp(maxOverlap) > 0 {
				maxOverlap = overlap
				match := iv
				bestMatch = &match
			}
		} else if end.Cmp(start) == 0 && bestMatch == nil {
			// Single point boundary intersection
			match := iv
			bestMatch = &match
		}
	}

	if bestMatch == nil && len(intervals) > 0 {
		bestMatch = &intervals[0]
	}

	if bestMatch != nil {
		return &AlgebraicReal{
			Poly: polySum,
			Low:  bestMatch.Low,
			High: bestMatch.High,
		}
	}

	return &AlgebraicReal{
		Poly: polySum,
		Low:  initLow,
		High: initHigh,
	}
}

// AlgSub computes a - b.
func AlgSub(a, b *AlgebraicReal) *AlgebraicReal {
	if a == nil || b == nil {
		return nil
	}
	if AlgCompare(a, b) == 0 {
		return NewAlgebraicRealFromRat(big.NewRat(0, 1))
	}
	// -b: negate odd degree coefficients of b.Poly
	negBPoly := make([]*big.Int, len(b.Poly))
	for i, c := range b.Poly {
		if i%2 == 1 {
			negBPoly[i] = new(big.Int).Neg(c)
		} else {
			negBPoly[i] = new(big.Int).Set(c)
		}
	}
	negB := &AlgebraicReal{
		Poly: negBPoly,
		Low:  new(big.Rat).Neg(b.High),
		High: new(big.Rat).Neg(b.Low),
	}
	return AlgAdd(a, negB)
}

// AlgMul computes a * b.
func AlgMul(a, b *AlgebraicReal) *AlgebraicReal {
	if a == nil || b == nil {
		return nil
	}
	if AlgIsZero(a) || AlgIsZero(b) {
		return NewAlgebraicRealFromRat(big.NewRat(0, 1))
	}

	degA := len(a.Poly) - 1
	degB := len(b.Poly) - 1
	targetDeg := degA * degB
	if targetDeg == 0 {
		targetDeg = 1
	}

	numPoints := targetDeg + 1
	xs := make([]*big.Int, numPoints)
	ys := make([]*big.Int, numPoints)

	for k := 0; k < numPoints; k++ {
		xs[k] = big.NewInt(int64(k))
		// A_k(y) = y^degA * A(k / y) = sum a_i k^i y^(degA - i)
		ak := make([]*big.Int, degA+1)
		for i := range ak {
			ak[i] = big.NewInt(0)
		}
		for i := 0; i <= degA; i++ {
			c := a.Poly[i]
			if c == nil || c.Sign() == 0 {
				continue
			}
			kPow := new(big.Int).Exp(xs[k], big.NewInt(int64(i)), nil)
			term := new(big.Int).Mul(c, kPow)
			ak[degA-i].Add(ak[degA-i], term)
		}

		ys[k] = sylvesterResultant(ak, b.Poly)
	}

	polyProd := lagrangeInterpolate(xs, ys)
	polyProd = squareFreePart(polyProd)

	intervals := IsolateRealRootsVAS(polyProd)

	// Target interval: product of intervals
	p1 := new(big.Rat).Mul(a.Low, b.Low)
	p2 := new(big.Rat).Mul(a.Low, b.High)
	p3 := new(big.Rat).Mul(a.High, b.Low)
	p4 := new(big.Rat).Mul(a.High, b.High)

	minP := new(big.Rat).Set(p1)
	maxP := new(big.Rat).Set(p1)
	for _, p := range []*big.Rat{p2, p3, p4} {
		if p.Cmp(minP) < 0 {
			minP.Set(p)
		}
		if p.Cmp(maxP) > 0 {
			maxP.Set(p)
		}
	}

	// Match root: choose interval with strictly positive overlap with [minP, maxP]
	var bestMatch *VASInterval
	maxOverlap := big.NewRat(0, 1)

	for _, iv := range intervals {
		// Overlap between [iv.Low, iv.High] and [minP, maxP]
		// start = max(iv.Low, minP), end = min(iv.High, maxP)
		start := iv.Low
		if minP.Cmp(start) > 0 {
			start = minP
		}
		end := iv.High
		if maxP.Cmp(end) < 0 {
			end = maxP
		}

		// Check if exact rational root falls within [minP, maxP]
		if iv.IsExact && iv.ExactVal != nil {
			if iv.ExactVal.Cmp(minP) >= 0 && iv.ExactVal.Cmp(maxP) <= 0 {
				match := iv
				bestMatch = &match
				break
			}
		}

		if end.Cmp(start) > 0 {
			overlap := new(big.Rat).Sub(end, start)
			if overlap.Cmp(maxOverlap) > 0 {
				maxOverlap = overlap
				match := iv
				bestMatch = &match
			}
		} else if end.Cmp(start) == 0 && bestMatch == nil {
			// Single point boundary intersection
			match := iv
			bestMatch = &match
		}
	}

	if bestMatch == nil && len(intervals) > 0 {
		bestMatch = &intervals[0]
	}

	if bestMatch != nil {
		return &AlgebraicReal{
			Poly: polyProd,
			Low:  bestMatch.Low,
			High: bestMatch.High,
		}
	}

	return &AlgebraicReal{
		Poly: polyProd,
		Low:  minP,
		High: maxP,
	}
}

// AlgDiv computes a / b (b != 0).
func AlgDiv(a, b *AlgebraicReal) (*AlgebraicReal, error) {
	if a == nil || b == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.domain_error", "alg_div"))
	}
	if AlgIsZero(b) {
		return nil, fmt.Errorf("%s", i18n.T("errors.zero_division", "alg_div"))
	}
	if AlgIsZero(a) {
		return NewAlgebraicRealFromRat(big.NewRat(0, 1)), nil
	}

	// Refine isolating interval of b until 0 is strictly excluded:
	// either b.Low > 0 or b.High < 0.
	zero := big.NewRat(0, 1)
	refineLimit := 100
	for refineLimit > 0 && b.Low.Cmp(zero) <= 0 && b.High.Cmp(zero) >= 0 {
		AlgRefineInterval(b)
		refineLimit--
	}
	if b.Low.Cmp(zero) <= 0 && b.High.Cmp(zero) >= 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.zero_division", "alg_div"))
	}

	// 1 / b: reciprocal of polynomial b.Poly
	degB := len(b.Poly) - 1
	recipBPoly := make([]*big.Int, degB+1)
	for i := 0; i <= degB; i++ {
		recipBPoly[i] = new(big.Int).Set(b.Poly[degB-i])
	}

	invLow := new(big.Rat).Inv(b.High)
	invHigh := new(big.Rat).Inv(b.Low)
	if invLow.Cmp(invHigh) > 0 {
		invLow, invHigh = invHigh, invLow
	}

	recipB := &AlgebraicReal{
		Poly: recipBPoly,
		Low:  invLow,
		High: invHigh,
	}

	return AlgMul(a, recipB), nil
}
