package calc

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// MobiusTransform represents a rational Mobius transformation M(x) = (A*x + B) / (C*x + D).
type MobiusTransform struct {
	A *big.Int
	B *big.Int
	C *big.Int
	D *big.Int
}

// NewIdentityMobius creates M(x) = (1*x + 0) / (0*x + 1) = x.
func NewIdentityMobius() MobiusTransform {
	return MobiusTransform{
		A: big.NewInt(1),
		B: big.NewInt(0),
		C: big.NewInt(0),
		D: big.NewInt(1),
	}
}

// Shift applies x <- x + c: M_new(x) = M(x + c) = (A(x+c) + B) / (C(x+c) + D).
func (m MobiusTransform) Shift(c *big.Int) MobiusTransform {
	ac := new(big.Int).Mul(m.A, c)
	cc := new(big.Int).Mul(m.C, c)
	return MobiusTransform{
		A: new(big.Int).Set(m.A),
		B: new(big.Int).Add(ac, m.B),
		C: new(big.Int).Set(m.C),
		D: new(big.Int).Add(cc, m.D),
	}
}

// ReciprocalShift1 applies x <- 1/(x+1): M_new(x) = M(1/(x+1))
// M(1/(x+1)) = (B*x + (A+B)) / (D*x + (C+D))
func (m MobiusTransform) ReciprocalShift1() MobiusTransform {
	return MobiusTransform{
		A: new(big.Int).Set(m.B),
		B: new(big.Int).Add(m.A, m.B),
		C: new(big.Int).Set(m.D),
		D: new(big.Int).Add(m.C, m.D),
	}
}

// EvalAt evaluates M(x) at rational point x = num/den.
func (m MobiusTransform) EvalAt(r *big.Rat) *big.Rat {
	// (A*(rNum/rDen) + B) / (C*(rNum/rDen) + D)
	// = (A*rNum + B*rDen) / (C*rNum + D*rDen)
	rNum := r.Num()
	rDen := r.Denom()

	numA := new(big.Int).Mul(m.A, rNum)
	numB := new(big.Int).Mul(m.B, rDen)
	top := new(big.Int).Add(numA, numB)

	denC := new(big.Int).Mul(m.C, rNum)
	denD := new(big.Int).Mul(m.D, rDen)
	bot := new(big.Int).Add(denC, denD)

	if bot.Sign() == 0 {
		return nil
	}
	return new(big.Rat).SetFrac(top, bot)
}

// EvalInterval evaluates M(x) on (0, infinity), using cauchyUB if an endpoint is infinite.
func (m MobiusTransform) EvalInterval(cauchyUB *big.Rat) (*big.Rat, *big.Rat) {
	// At x = 0:
	var r0 *big.Rat
	if m.D.Sign() != 0 {
		r0 = new(big.Rat).SetFrac(m.B, m.D)
	} else if cauchyUB != nil && cauchyUB.Sign() > 0 {
		// D = 0: x = 0 is a pole. Approximate near 0 with 1/cauchyUB
		eps := new(big.Rat).Inv(cauchyUB)
		r0 = m.EvalAt(eps)
	}

	// At x -> inf:
	var rInf *big.Rat
	if m.C.Sign() != 0 {
		rInf = new(big.Rat).SetFrac(m.A, m.C)
	} else if cauchyUB != nil {
		// C = 0: M(x) = (Ax + B)/D. The root in the original space cannot exceed cauchyUB.
		rInf = new(big.Rat).Set(cauchyUB)
	}

	if r0 == nil && rInf == nil {
		return big.NewRat(0, 1), big.NewRat(1, 1)
	}
	if r0 == nil {
		return rInf, rInf
	}
	if rInf == nil {
		return r0, r0
	}

	if r0.Cmp(rInf) <= 0 {
		return r0, rInf
	}
	return rInf, r0
}

// VASInterval represents an isolated root interval or exact rational root.
type VASInterval struct {
	Low      *big.Rat
	High     *big.Rat
	IsExact  bool
	ExactVal *big.Rat
}

// CountSignVariations counts sign variations in polynomial coefficients (ignoring zeros).
func CountSignVariations(coeffs []*big.Int) int {
	var lastSign int
	variations := 0
	for _, c := range coeffs {
		if c == nil || c.Sign() == 0 {
			continue
		}
		s := c.Sign()
		if lastSign != 0 && s != lastSign {
			variations++
		}
		lastSign = s
	}
	return variations
}

// taylorShift computes P(x + c) using synthetic division (Ruffini-Horner).
// coeffs is in ascending order: coeffs[0] + coeffs[1]*x + ... + coeffs[n]*x^n.
func taylorShift(coeffs []*big.Int, c *big.Int) []*big.Int {
	n := len(coeffs)
	res := make([]*big.Int, n)
	for i := 0; i < n; i++ {
		if coeffs[i] != nil {
			res[i] = new(big.Int).Set(coeffs[i])
		} else {
			res[i] = big.NewInt(0)
		}
	}
	if c == nil || c.Sign() == 0 {
		return res
	}

	for i := n - 1; i >= 0; i-- {
		for j := i; j < n-1; j++ {
			term := new(big.Int).Mul(c, res[j+1])
			res[j].Add(res[j], term)
		}
	}
	return res
}

// reciprocalShift1 computes (x+1)^n * P(1/(x+1)) = P_rev(x + 1).
func reciprocalShift1(coeffs []*big.Int) []*big.Int {
	n := len(coeffs)
	rev := make([]*big.Int, n)
	for i := 0; i < n; i++ {
		if coeffs[n-1-i] != nil {
			rev[i] = new(big.Int).Set(coeffs[n-1-i])
		} else {
			rev[i] = big.NewInt(0)
		}
	}
	return taylorShift(rev, big.NewInt(1))
}

// evalPolyAtInt64 evaluates polynomial at integer point x.
func evalPolyAtInt64(coeffs []*big.Int, x int64) *big.Int {
	res := big.NewInt(0)
	bx := big.NewInt(x)
	for i := len(coeffs) - 1; i >= 0; i-- {
		res.Mul(res, bx)
		if coeffs[i] != nil {
			res.Add(res, coeffs[i])
		}
	}
	return res
}

// evalIntPolyAtRat evaluates integer polynomial at rational point r.
func evalIntPolyAtRat(coeffs []*big.Int, r *big.Rat) *big.Rat {
	res := big.NewRat(0, 1)
	for i := len(coeffs) - 1; i >= 0; i-- {
		res.Mul(res, r)
		if coeffs[i] != nil {
			ratC := new(big.Rat).SetInt(coeffs[i])
			res.Add(res, ratC)
		}
	}
	return res
}

// computeCauchyUB computes Cauchy bound 1 + max_{i < n} |a_i| / |a_n|
func computeCauchyUB(coeffs []*big.Int) *big.Rat {
	n := len(coeffs) - 1
	for n >= 0 && (coeffs[n] == nil || coeffs[n].Sign() == 0) {
		n--
	}
	if n <= 0 {
		return big.NewRat(2, 1)
	}

	leadAbs := new(big.Int).Abs(coeffs[n])
	maxOther := big.NewInt(0)
	for i := 0; i < n; i++ {
		if coeffs[i] != nil {
			abs := new(big.Int).Abs(coeffs[i])
			if abs.Cmp(maxOther) > 0 {
				maxOther = abs
			}
		}
	}

	num := new(big.Int).Add(leadAbs, maxOther)
	return new(big.Rat).SetFrac(num, leadAbs)
}

// refineInterval narrows down [Low, High] for polynomial poly using integer checks and bisection.
func refineInterval(poly []*big.Int, low, high *big.Rat) VASInterval {
	// First check if endpoints are exact roots
	valLow := evalIntPolyAtRat(poly, low)
	if valLow.Sign() == 0 {
		return VASInterval{Low: low, High: low, IsExact: true, ExactVal: low}
	}

	valHigh := evalIntPolyAtRat(poly, high)
	if valHigh.Sign() == 0 {
		return VASInterval{Low: high, High: high, IsExact: true, ExactVal: high}
	}

	// Check integer points in [ceil(low), floor(high)]
	// If low = 0 and high = 2, integers to check: 1, 2
	lowFloor := new(big.Int).Quo(low.Num(), low.Denom())
	highFloor := new(big.Int).Quo(high.Num(), high.Denom())

	for k := new(big.Int).Set(lowFloor); k.Cmp(new(big.Int).Add(highFloor, big.NewInt(1))) <= 0; k.Add(k, big.NewInt(1)) {
		kRat := new(big.Rat).SetInt(k)
		if kRat.Cmp(low) >= 0 && kRat.Cmp(high) <= 0 {
			if evalIntPolyAtRat(poly, kRat).Sign() == 0 {
				return VASInterval{Low: kRat, High: kRat, IsExact: true, ExactVal: kRat}
			}
		}
	}

	currLow := new(big.Rat).Set(low)
	currHigh := new(big.Rat).Set(high)
	signLow := valLow.Sign()

	half := big.NewRat(1, 2)
	diff := new(big.Rat).Sub(currHigh, currLow)

	// Bisection loop: refine until width <= 1/2 or exact root found
	for diff.Cmp(half) > 0 {
		mid := new(big.Rat).Add(currLow, currHigh)
		mid.Mul(mid, big.NewRat(1, 2))

		valMid := evalIntPolyAtRat(poly, mid)
		if valMid.Sign() == 0 {
			return VASInterval{Low: mid, High: mid, IsExact: true, ExactVal: mid}
		}

		if valMid.Sign() == signLow {
			currLow.Set(mid)
		} else {
			currHigh.Set(mid)
		}
		diff.Sub(currHigh, currLow)
	}

	return VASInterval{
		Low:     currLow,
		High:    currHigh,
		IsExact: false,
	}
}

type vasTask struct {
	poly []*big.Int
	m    MobiusTransform
}

// isolatePositiveRootsVAS isolates all roots in (0, infinity).
func isolatePositiveRootsVAS(poly []*big.Int, origPoly []*big.Int) []VASInterval {
	var intervals []VASInterval

	for len(poly) > 0 && (poly[len(poly)-1] == nil || poly[len(poly)-1].Sign() == 0) {
		poly = poly[:len(poly)-1]
	}
	if len(poly) <= 1 {
		return nil
	}

	cauchyUB := computeCauchyUB(poly)

	stack := []vasTask{{
		poly: poly,
		m:    NewIdentityMobius(),
	}}

	maxIter := 5000
	iter := 0

	for len(stack) > 0 {
		iter++
		if iter > maxIter {
			break
		}

		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		f := curr.poly
		m := curr.m

		v := CountSignVariations(f)
		if v == 0 {
			continue
		}
		if v == 1 {
			low, high := m.EvalInterval(cauchyUB)
			// Refine interval against original polynomial
			refined := refineInterval(origPoly, low, high)
			intervals = append(intervals, refined)
			continue
		}

		// Check if x = 1 is an exact root: f(1) == 0
		val1 := evalPolyAtInt64(f, 1)
		if val1.Sign() == 0 {
			exactNum := new(big.Int).Add(m.A, m.B)
			exactDen := new(big.Int).Add(m.C, m.D)
			if exactDen.Sign() != 0 {
				exactRat := new(big.Rat).SetFrac(exactNum, exactDen)
				intervals = append(intervals, VASInterval{
					Low:      exactRat,
					High:     exactRat,
					IsExact:  true,
					ExactVal: exactRat,
				})
			}
		}

		// Branch 1: (0, 1) -> x <- 1/(x+1)
		f1 := reciprocalShift1(f)
		m1 := m.ReciprocalShift1()
		v1 := CountSignVariations(f1)
		if v1 > 0 {
			stack = append(stack, vasTask{poly: f1, m: m1})
		}

		// Branch 2: (1, inf) -> x <- x + 1
		f2 := taylorShift(f, big.NewInt(1))
		m2 := m.Shift(big.NewInt(1))
		v2 := CountSignVariations(f2)
		if v2 > 0 {
			stack = append(stack, vasTask{poly: f2, m: m2})
		}
	}

	return intervals
}

// IsolateRealRootsVAS isolates all real roots of integer polynomial poly.
func IsolateRealRootsVAS(poly []*big.Int) []VASInterval {
	start := 0
	for start < len(poly) && (poly[start] == nil || poly[start].Sign() == 0) {
		start++
	}

	end := len(poly)
	for end > start && (poly[end-1] == nil || poly[end-1].Sign() == 0) {
		end--
	}

	if start >= end {
		return nil
	}

	var results []VASInterval

	// x = 0 is a root if start > 0
	if start > 0 {
		zeroRat := big.NewRat(0, 1)
		results = append(results, VASInterval{
			Low:      zeroRat,
			High:     zeroRat,
			IsExact:  true,
			ExactVal: zeroRat,
		})
	}

	trimmed := poly[start:end]
	if len(trimmed) <= 1 {
		return results
	}

	// Apply square-free factorization to eliminate multiple roots
	// and ensure Descartes' Rule of Signs contracts correctly in VAS.
	sqfTrimmed := squareFreePart(trimmed)
	if len(sqfTrimmed) > 1 {
		trimmed = sqfTrimmed
	}

	// 1. Positive roots: Q(x)
	posRoots := isolatePositiveRootsVAS(trimmed, trimmed)
	results = append(results, posRoots...)

	// 2. Negative roots: Q(-x)
	negPoly := make([]*big.Int, len(trimmed))
	for i, c := range trimmed {
		if c != nil {
			if i%2 == 1 {
				negPoly[i] = new(big.Int).Neg(c)
			} else {
				negPoly[i] = new(big.Int).Set(c)
			}
		}
	}
	negRoots := isolatePositiveRootsVAS(negPoly, negPoly)
	for _, nr := range negRoots {
		invLow := new(big.Rat).Neg(nr.High)
		invHigh := new(big.Rat).Neg(nr.Low)
		var exactVal *big.Rat
		if nr.IsExact && nr.ExactVal != nil {
			exactVal = new(big.Rat).Neg(nr.ExactVal)
		}
		results = append(results, VASInterval{
			Low:      invLow,
			High:     invHigh,
			IsExact:  nr.IsExact,
			ExactVal: exactVal,
		})
	}

	// Remove duplicate exact roots or overlapping intervals if any
	// Sort intervals ascending by Low bound
	sort.Slice(results, func(i, j int) bool {
		cmp := results[i].Low.Cmp(results[j].Low)
		if cmp != 0 {
			return cmp < 0
		}
		return results[i].High.Cmp(results[j].High) < 0
	})

	return results
}

// EvalIsolateRootsVAS provides a REPL/Evaluator accessible function for VAS real root isolation.
func EvalIsolateRootsVAS(p Node, varName string, env *Env) (Node, error) {
	if p == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "isolate_roots_vas"))
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
			return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "isolate_roots_vas"))
		}
		v = vars[0]
	}

	polyP, ok := extractPoly(evalP, v)
	if !ok || !isRationalPoly(polyP) {
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_poly_required", "isolate_roots_vas"))
	}

	deg := polyP.degree()
	if deg <= 0 {
		return &ListNode{Elements: []Node{}}, nil
	}

	dens := make([]*big.Int, deg+1)
	nums := make([]*big.Int, deg+1)
	for i := 0; i <= deg; i++ {
		cNode := polyP.coeffs[i]
		if rNode, ok := cNode.(*RationalNode); ok && rNode != nil && rNode.Val != nil {
			nums[i] = new(big.Int).Set(rNode.Val.Num())
			dens[i] = new(big.Int).Set(rNode.Val.Denom())
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

	intCoeffs := make([]*big.Int, deg+1)
	for i := 0; i <= deg; i++ {
		scale := new(big.Int).Div(lcm, dens[i])
		intCoeffs[i] = new(big.Int).Mul(nums[i], scale)
	}

	intervals := IsolateRealRootsVAS(intCoeffs)

	elements := make([]Node, len(intervals))
	for i, iv := range intervals {
		if iv.IsExact && iv.ExactVal != nil {
			elements[i] = &RationalNode{Val: iv.ExactVal}
		} else {
			lowNode := &RationalNode{Val: iv.Low}
			highNode := &RationalNode{Val: iv.High}
			elements[i] = &ListNode{Elements: []Node{lowNode, highNode}}
		}
	}

	return &ListNode{Elements: elements}, nil
}
