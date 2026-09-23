package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// RationalInterval represents a closed interval [Low, High] subset of Q
// with arbitrary-precision rational endpoints.
type RationalInterval struct {
	Low  *big.Rat
	High *big.Rat
}

// NewRationalInterval creates a new RationalInterval ensuring Low <= High.
func NewRationalInterval(low, high *big.Rat) RationalInterval {
	l := new(big.Rat).Set(low)
	h := new(big.Rat).Set(high)
	if l.Cmp(h) > 0 {
		l, h = h, l
	}
	return RationalInterval{Low: l, High: h}
}

// NewExactRationalInterval creates a degenerate interval [q, q].
func NewExactRationalInterval(q *big.Rat) RationalInterval {
	c := new(big.Rat).Set(q)
	return RationalInterval{Low: c, High: c}
}

// Add computes interval addition [L1 + L2, H1 + H2].
func (i RationalInterval) Add(other RationalInterval) RationalInterval {
	low := new(big.Rat).Add(i.Low, other.Low)
	high := new(big.Rat).Add(i.High, other.High)
	return NewRationalInterval(low, high)
}

// Sub computes interval subtraction [L1 - H2, H1 - L2].
func (i RationalInterval) Sub(other RationalInterval) RationalInterval {
	low := new(big.Rat).Sub(i.Low, other.High)
	high := new(big.Rat).Sub(i.High, other.Low)
	return NewRationalInterval(low, high)
}

// Neg computes [-High, -Low].
func (i RationalInterval) Neg() RationalInterval {
	low := new(big.Rat).Neg(i.High)
	high := new(big.Rat).Neg(i.Low)
	return NewRationalInterval(low, high)
}

// Mul computes interval multiplication:
// [min(L1*L2, L1*H2, H1*L2, H1*H2), max(L1*L2, L1*H2, H1*L2, H1*H2)].
func (i RationalInterval) Mul(other RationalInterval) RationalInterval {
	p1 := new(big.Rat).Mul(i.Low, other.Low)
	p2 := new(big.Rat).Mul(i.Low, other.High)
	p3 := new(big.Rat).Mul(i.High, other.Low)
	p4 := new(big.Rat).Mul(i.High, other.High)

	minVal := p1
	maxVal := p1

	for _, p := range []*big.Rat{p2, p3, p4} {
		if p.Cmp(minVal) < 0 {
			minVal = p
		}
		if p.Cmp(maxVal) > 0 {
			maxVal = p
		}
	}

	return NewRationalInterval(minVal, maxVal)
}

// Inv computes reciprocal [1/High, 1/Low]. Returns error if interval contains zero.
func (i RationalInterval) Inv() (RationalInterval, error) {
	if i.ContainsZero() {
		return RationalInterval{}, fmt.Errorf("%s", i18n.T("calc.interval_zero_division", i.String()))
	}
	one := big.NewRat(1, 1)
	low := new(big.Rat).Quo(one, i.High)
	high := new(big.Rat).Quo(one, i.Low)
	return NewRationalInterval(low, high), nil
}

// Div computes interval division i * other.Inv().
func (i RationalInterval) Div(other RationalInterval) (RationalInterval, error) {
	inv, err := other.Inv()
	if err != nil {
		return RationalInterval{}, err
	}
	return i.Mul(inv), nil
}

// PowInt computes interval raised to non-negative integer power k.
func (i RationalInterval) PowInt(k int) RationalInterval {
	if k == 0 {
		return NewExactRationalInterval(big.NewRat(1, 1))
	}
	if k < 0 {
		inv, err := i.Inv()
		if err != nil {
			return RationalInterval{Low: big.NewRat(0, 1), High: big.NewRat(0, 1)}
		}
		return inv.PowInt(-k)
	}

	powRat := func(r *big.Rat, exp int) *big.Rat {
		num := new(big.Int).Exp(r.Num(), big.NewInt(int64(exp)), nil)
		denom := new(big.Int).Exp(r.Denom(), big.NewInt(int64(exp)), nil)
		return new(big.Rat).SetFrac(num, denom)
	}

	if k%2 == 1 {
		// Odd power: monotonically increasing
		return NewRationalInterval(powRat(i.Low, k), powRat(i.High, k))
	}

	// Even power:
	pLow := powRat(i.Low, k)
	pHigh := powRat(i.High, k)
	if i.ContainsZero() {
		maxVal := pLow
		if pHigh.Cmp(maxVal) > 0 {
			maxVal = pHigh
		}
		return NewRationalInterval(big.NewRat(0, 1), maxVal)
	}

	zero := big.NewRat(0, 1)
	if i.Low.Cmp(zero) >= 0 {
		return NewRationalInterval(pLow, pHigh)
	}
	return NewRationalInterval(pHigh, pLow)
}

// IsStrictlyPositive returns true if Low > 0.
func (i RationalInterval) IsStrictlyPositive() bool {
	return i.Low.Sign() > 0
}

// IsStrictlyNegative returns true if High < 0.
func (i RationalInterval) IsStrictlyNegative() bool {
	return i.High.Sign() < 0
}

// ContainsZero returns true if Low <= 0 <= High.
func (i RationalInterval) ContainsZero() bool {
	return i.Low.Sign() <= 0 && i.High.Sign() >= 0
}

// Width returns High - Low.
func (i RationalInterval) Width() *big.Rat {
	return new(big.Rat).Sub(i.High, i.Low)
}

// String returns formatted interval string "[Low, High]".
func (i RationalInterval) String() string {
	return fmt.Sprintf("[%s, %s]", i.Low.RatString(), i.High.RatString())
}

// RationalBoundPi computes rigorous rational enclosure [Low, High] for pi
// using Machin's formula: pi/4 = 4*arctan(1/5) - arctan(1/239).
// Since the Taylor series of arctan(x) is alternating for x in (0, 1),
// even and odd partial sums strictly bound the true value.
func RationalBoundPi(steps int) RationalInterval {
	if steps < 1 {
		steps = 4
	}

	calcArctanBounds := func(x *big.Rat, nTerms int) (low, high *big.Rat) {
		// S_m = sum_{k=0}^{m-1} (-1)^k * x^(2k+1) / (2k+1)
		sumLow := big.NewRat(0, 1)
		sumHigh := big.NewRat(0, 1)

		xSquared := new(big.Rat).Mul(x, x)
		curPow := new(big.Rat).Set(x)

		for k := 0; k < nTerms*2; k++ {
			denom := big.NewRat(int64(2*k+1), 1)
			term := new(big.Rat).Quo(curPow, denom)

			if k%2 == 0 {
				// Adding positive term
				sumHigh = new(big.Rat).Add(sumLow, term)
			} else {
				// Subtracting negative term
				sumLow = new(big.Rat).Sub(sumHigh, term)
			}
			curPow = new(big.Rat).Mul(curPow, xSquared)
		}
		return sumLow, sumHigh
	}

	x1 := big.NewRat(1, 5)
	low1, high1 := calcArctanBounds(x1, steps)

	x2 := big.NewRat(1, 239)
	low2, high2 := calcArctanBounds(x2, steps)

	// pi = 16 * arctan(1/5) - 4 * arctan(1/239)
	c16 := big.NewRat(16, 1)
	c4 := big.NewRat(4, 1)

	piLow := new(big.Rat).Sub(new(big.Rat).Mul(c16, low1), new(big.Rat).Mul(c4, high2))
	piHigh := new(big.Rat).Sub(new(big.Rat).Mul(c16, high1), new(big.Rat).Mul(c4, low2))

	return NewRationalInterval(piLow, piHigh)
}

// RationalBoundE computes rigorous rational enclosure [Low, High] for Euler's number e
// using series e = sum_{k=0}^n 1/k! with remainder R_n < 1 / (n * n!).
func RationalBoundE(steps int) RationalInterval {
	if steps < 2 {
		steps = 10
	}

	sum := big.NewRat(0, 1)
	fact := big.NewInt(1)

	for k := 0; k <= steps; k++ {
		if k > 0 {
			fact.Mul(fact, big.NewInt(int64(k)))
		}
		term := new(big.Rat).SetFrac(big.NewInt(1), fact)
		sum.Add(sum, term)
	}

	// Remainder R_n < 1 / (steps * steps!)
	remDenom := new(big.Int).Mul(big.NewInt(int64(steps)), fact)
	rem := new(big.Rat).SetFrac(big.NewInt(1), remDenom)

	low := new(big.Rat).Set(sum)
	high := new(big.Rat).Add(sum, rem)

	return NewRationalInterval(low, high)
}

// RationalBoundExp computes rigorous rational enclosure for exp(x) with x in Q.
func RationalBoundExp(x *big.Rat, steps int) RationalInterval {
	if steps < 4 {
		steps = 12
	}

	if x.Sign() == 0 {
		return NewExactRationalInterval(big.NewRat(1, 1))
	}

	if x.Sign() < 0 {
		// exp(x) = 1 / exp(-x)
		negX := new(big.Rat).Neg(x)
		posBound := RationalBoundExp(negX, steps)
		inv, err := posBound.Inv()
		if err == nil {
			return inv
		}
	}

	// For x > 0:
	// S_n = sum_{k=0}^n x^k / k!
	// Remainder R_n = sum_{k=n+1}^inf x^k / k! < (x^(n+1) / (n+1)!) / (1 - x/(n+2))
	// provided n+2 > x.
	n := steps
	xInt := new(big.Int).Quo(x.Num(), x.Denom()).Int64()
	if int64(n) < xInt+3 {
		n = int(xInt) + 4
	}

	sum := big.NewRat(0, 1)
	curPow := big.NewRat(1, 1)
	fact := big.NewInt(1)

	for k := 0; k <= n; k++ {
		if k > 0 {
			curPow.Mul(curPow, x)
			fact.Mul(fact, big.NewInt(int64(k)))
		}
		term := new(big.Rat).Quo(curPow, new(big.Rat).SetInt(fact))
		sum.Add(sum, term)
	}

	// Next term: x^(n+1) / (n+1)!
	nextPow := new(big.Rat).Mul(curPow, x)
	nextFact := new(big.Int).Mul(fact, big.NewInt(int64(n+1)))
	nextTerm := new(big.Rat).Quo(nextPow, new(big.Rat).SetInt(nextFact))

	// geomFactor = 1 / (1 - x/(n+2))
	xOverN2 := new(big.Rat).Quo(x, big.NewRat(int64(n+2), 1))
	oneMinus := new(big.Rat).Sub(big.NewRat(1, 1), xOverN2)
	geomFactor := new(big.Rat).Quo(big.NewRat(1, 1), oneMinus)

	rem := new(big.Rat).Mul(nextTerm, geomFactor)

	low := new(big.Rat).Set(sum)
	high := new(big.Rat).Add(sum, rem)

	return NewRationalInterval(low, high)
}

// RationalBoundSinCos computes rigorous rational enclosure for sin(x) or cos(x) with x in Q.
func RationalBoundSinCos(x *big.Rat, isSin bool, steps int) RationalInterval {
	if steps < 4 {
		steps = 10
	}

	if x.Sign() == 0 {
		if isSin {
			return NewExactRationalInterval(big.NewRat(0, 1))
		}
		return NewExactRationalInterval(big.NewRat(1, 1))
	}

	if x.Sign() < 0 {
		negX := new(big.Rat).Neg(x)
		if isSin {
			return RationalBoundSinCos(negX, true, steps).Neg()
		}
		return RationalBoundSinCos(negX, false, steps)
	}

	// Ensure steps is large enough so that the terms strictly decrease: (2k+2)(2k+3) > x^2
	xInt := new(big.Int).Quo(x.Num(), x.Denom()).Int64()
	if int64(steps) < xInt+2 {
		steps = int(xInt) + 4
	}

	sumLow := big.NewRat(0, 1)
	sumHigh := big.NewRat(0, 1)

	fact := big.NewInt(1)
	curPow := big.NewRat(1, 1)
	xSquared := new(big.Rat).Mul(x, x)

	maxK := steps * 2
	if isSin {
		curPow.Set(x)
		fact.SetInt64(1)
	} else {
		curPow.Set(big.NewRat(1, 1))
		fact.SetInt64(1)
	}

	for k := 0; k < maxK; k++ {
		term := new(big.Rat).Quo(curPow, new(big.Rat).SetInt(fact))

		if k%2 == 0 {
			// + term
			sumHigh = new(big.Rat).Add(sumLow, term)
		} else {
			// - term
			sumLow = new(big.Rat).Sub(sumHigh, term)
		}

		// Advance power and factorial for next step
		curPow.Mul(curPow, xSquared)
		if isSin {
			idx1 := int64(2*k + 2)
			idx2 := int64(2*k + 3)
			fact.Mul(fact, big.NewInt(idx1))
			fact.Mul(fact, big.NewInt(idx2))
		} else {
			idx1 := int64(2*k + 1)
			idx2 := int64(2*k + 2)
			fact.Mul(fact, big.NewInt(idx1))
			fact.Mul(fact, big.NewInt(idx2))
		}
	}

	return NewRationalInterval(sumLow, sumHigh)
}

// RationalBoundLn computes rigorous rational enclosure for ln(x) with x in Q, x > 0
// using series ln(x) = 2 * sum_{k=0}^n y^(2k+1) / (2k+1) where y = (x-1)/(x+1) in (0, 1).
func RationalBoundLn(x *big.Rat, steps int) (RationalInterval, error) {
	if x.Sign() <= 0 {
		return RationalInterval{}, fmt.Errorf("%s", i18n.T("calc.interval_ln_domain", x.RatString()))
	}
	if x.Cmp(big.NewRat(1, 1)) == 0 {
		return NewExactRationalInterval(big.NewRat(0, 1)), nil
	}

	if steps < 4 {
		steps = 10
	}

	if x.Cmp(big.NewRat(1, 1)) < 0 {
		// ln(x) = -ln(1/x)
		invX := new(big.Rat).Inv(x)
		b, err := RationalBoundLn(invX, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		return b.Neg(), nil
	}

	// x > 1: y = (x-1)/(x+1) in (0, 1)
	one := big.NewRat(1, 1)
	y := new(big.Rat).Quo(new(big.Rat).Sub(x, one), new(big.Rat).Add(x, one))
	ySquared := new(big.Rat).Mul(y, y)

	sum := big.NewRat(0, 1)
	curPow := new(big.Rat).Set(y)

	for k := 0; k <= steps; k++ {
		denom := big.NewRat(int64(2*k+1), 1)
		term := new(big.Rat).Quo(curPow, denom)
		sum.Add(sum, term)
		curPow.Mul(curPow, ySquared)
	}

	sum.Mul(sum, big.NewRat(2, 1))

	// Remainder R_n < 2/(2n+3) * y^(2n+3) / (1 - y^2)
	twoOver := big.NewRat(2, int64(2*steps+3))
	oneMinusYSq := new(big.Rat).Sub(big.NewRat(1, 1), ySquared)
	remGeom := new(big.Rat).Quo(curPow, oneMinusYSq)
	rem := new(big.Rat).Mul(twoOver, remGeom)

	low := new(big.Rat).Set(sum)
	high := new(big.Rat).Add(sum, rem)

	return NewRationalInterval(low, high), nil
}

// RationalBoundSqrt computes rigorous rational enclosure for sqrt(x) with x in Q, x >= 0
// using integer square root (big.Int.Sqrt) with dyadic scaling, guaranteeing O(k) bit complexity without bit explosion.
func RationalBoundSqrt(x *big.Rat, steps int) (RationalInterval, error) {
	if x.Sign() < 0 {
		return RationalInterval{}, fmt.Errorf("sqrt of negative rational")
	}
	if x.Sign() == 0 {
		return NewExactRationalInterval(big.NewRat(0, 1)), nil
	}

	// Precision scale: k bits of precision (clamped to [16, 128] for guaranteed linear bit complexity)
	k := steps * 4
	if k < 16 {
		k = 16
	}
	if k > 128 {
		k = 128
	}

	p := x.Num()
	q := x.Denom()
	scaledNum := new(big.Int).Lsh(p, uint(2*k))
	div := new(big.Int).Quo(scaledNum, q)

	m := new(big.Int).Sqrt(div)
	mPlusOne := new(big.Int).Add(m, big.NewInt(1))
	denom := new(big.Int).Lsh(big.NewInt(1), uint(k))

	low := new(big.Rat).SetFrac(m, denom)
	high := new(big.Rat).SetFrac(mPlusOne, denom)

	return NewRationalInterval(low, high), nil
}

// EvalNodeInterval recursively evaluates an AST expression into a RationalInterval enclosure.
func EvalNodeInterval(n Node, steps int) (RationalInterval, error) {
	if n == nil {
		return RationalInterval{}, fmt.Errorf("nil node")
	}

	switch node := n.(type) {
	case *RationalNode:
		return NewExactRationalInterval(node.Val), nil

	case *SqrtNode:
		sub, err := EvalNodeInterval(node.Radicand, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		if sub.Low.Sign() < 0 {
			return RationalInterval{}, fmt.Errorf("sqrt of negative interval")
		}
		lowBound, err := RationalBoundSqrt(sub.Low, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		highBound, err := RationalBoundSqrt(sub.High, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		return NewRationalInterval(lowBound.Low, highBound.High), nil

	case *ConstNode:
		if node.Name == "pi" || node.Name == "π" {
			return RationalBoundPi(steps), nil
		}
		if node.Name == "e" {
			return RationalBoundE(steps), nil
		}
		if node.Name == "deg" {
			piBound := RationalBoundPi(steps)
			deg180 := big.NewRat(180, 1)
			return NewRationalInterval(new(big.Rat).Quo(piBound.Low, deg180), new(big.Rat).Quo(piBound.High, deg180)), nil
		}
		return RationalInterval{}, fmt.Errorf("unsupported constant in interval evaluation: %s", node.Name)

	case *VarNode:
		if node.Name == "pi" || node.Name == "π" {
			return RationalBoundPi(steps), nil
		}
		if node.Name == "e" {
			return RationalBoundE(steps), nil
		}
		return RationalInterval{}, fmt.Errorf("unsupported free variable in interval evaluation: %s", node.Name)

	case *UnaryOpNode:
		sub, err := EvalNodeInterval(node.Expr, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		if node.Op == "-" {
			return sub.Neg(), nil
		}
		return sub, nil

	case *AddNode:
		if len(node.Terms) == 0 {
			return NewExactRationalInterval(big.NewRat(0, 1)), nil
		}
		acc, err := EvalNodeInterval(node.Terms[0], steps)
		if err != nil {
			return RationalInterval{}, err
		}
		for _, t := range node.Terms[1:] {
			sub, err := EvalNodeInterval(t, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			acc = acc.Add(sub)
		}
		return acc, nil

	case *MulNode:
		if len(node.Factors) == 0 {
			return NewExactRationalInterval(big.NewRat(1, 1)), nil
		}
		acc, err := EvalNodeInterval(node.Factors[0], steps)
		if err != nil {
			return RationalInterval{}, err
		}
		for _, f := range node.Factors[1:] {
			sub, err := EvalNodeInterval(f, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			acc = acc.Mul(sub)
		}
		return acc, nil

	case *PowNode:
		// Base ^ Exp
		// Special case 1: Integer power
		if expRat, isRat := node.Exp.(*RationalNode); isRat && expRat.Val.IsInt() {
			baseInt, err := EvalNodeInterval(node.Base, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			return baseInt.PowInt(int(expRat.Val.Num().Int64())), nil
		}

		// Special case 2: e ^ Exp => exp(Exp)
		isBaseE := false
		if v, isV := node.Base.(*VarNode); isV && v.Name == "e" {
			isBaseE = true
		}
		if c, isC := node.Base.(*ConstNode); isC && c.Name == "e" {
			isBaseE = true
		}
		if isBaseE {
			expInt, err := EvalNodeInterval(node.Exp, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			lowBound := RationalBoundExp(expInt.Low, steps)
			highBound := RationalBoundExp(expInt.High, steps)
			return NewRationalInterval(lowBound.Low, highBound.High), nil
		}

		// General real power A^B = exp(B * ln(A))
		baseInt, err := EvalNodeInterval(node.Base, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		expInt, err := EvalNodeInterval(node.Exp, steps)
		if err != nil {
			return RationalInterval{}, err
		}

		if !baseInt.IsStrictlyPositive() {
			return RationalInterval{}, fmt.Errorf("non-positive base in real power: %s", baseInt.String())
		}

		// ln(baseInt) = [ln(baseInt.Low).Low, ln(baseInt.High).High]
		lnLow, err := RationalBoundLn(baseInt.Low, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		lnHigh, err := RationalBoundLn(baseInt.High, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		lnInterval := NewRationalInterval(lnLow.Low, lnHigh.High)

		// product = B * ln(A)
		prod := expInt.Mul(lnInterval)

		// exp(product)
		eLow := RationalBoundExp(prod.Low, steps)
		eHigh := RationalBoundExp(prod.High, steps)
		return NewRationalInterval(eLow.Low, eHigh.High), nil

	case *FuncNode:
		switch node.Name {
		case "exp":
			if len(node.Args) != 1 {
				return RationalInterval{}, fmt.Errorf("exp expects 1 argument")
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			lowBound := RationalBoundExp(argInt.Low, steps)
			highBound := RationalBoundExp(argInt.High, steps)
			return NewRationalInterval(lowBound.Low, highBound.High), nil

		case "ln", "log":
			if len(node.Args) != 1 {
				return RationalInterval{}, fmt.Errorf("ln expects 1 argument")
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			if !argInt.IsStrictlyPositive() {
				return RationalInterval{}, fmt.Errorf("ln argument must be strictly positive: %s", argInt.String())
			}
			lowBound, err := RationalBoundLn(argInt.Low, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			highBound, err := RationalBoundLn(argInt.High, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			return NewRationalInterval(lowBound.Low, highBound.High), nil

		case "sin":
			if len(node.Args) != 1 {
				return RationalInterval{}, fmt.Errorf("sin expects 1 argument")
			}
			if r, isR := node.Args[0].(*RationalNode); isR {
				return RationalBoundSinCos(r.Val, true, steps), nil
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			// For narrow intervals within monotone region or evaluation at endpoints:
			b1 := RationalBoundSinCos(argInt.Low, true, steps)
			b2 := RationalBoundSinCos(argInt.High, true, steps)
			minVal := b1.Low
			if b2.Low.Cmp(minVal) < 0 {
				minVal = b2.Low
			}
			maxVal := b1.High
			if b2.High.Cmp(maxVal) > 0 {
				maxVal = b2.High
			}
			return NewRationalInterval(minVal, maxVal), nil

		case "cos":
			if len(node.Args) != 1 {
				return RationalInterval{}, fmt.Errorf("cos expects 1 argument")
			}
			if r, isR := node.Args[0].(*RationalNode); isR {
				return RationalBoundSinCos(r.Val, false, steps), nil
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			b1 := RationalBoundSinCos(argInt.Low, false, steps)
			b2 := RationalBoundSinCos(argInt.High, false, steps)
			minVal := b1.Low
			if b2.Low.Cmp(minVal) < 0 {
				minVal = b2.Low
			}
			maxVal := b1.High
			if b2.High.Cmp(maxVal) > 0 {
				maxVal = b2.High
			}
			return NewRationalInterval(minVal, maxVal), nil

		case "sqrt":
			if len(node.Args) != 1 {
				return RationalInterval{}, fmt.Errorf("sqrt expects 1 argument")
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			if argInt.Low.Sign() < 0 {
				return RationalInterval{}, fmt.Errorf("sqrt of negative interval")
			}
			lowBound, err := RationalBoundSqrt(argInt.Low, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			highBound, err := RationalBoundSqrt(argInt.High, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			return NewRationalInterval(lowBound.Low, highBound.High), nil
		}
	}

	return RationalInterval{}, fmt.Errorf("unsupported node for interval evaluation: %s", n.String())
}

// EvaluateRelOpWithInterval evaluates a relational operator (==, !=, <, <=, >, >=)
// using adaptive precision rational interval arithmetic on the difference (LHS - RHS).
// Returns (result, decided). If the difference interval strictly excludes zero, decided is true.
func EvaluateRelOpWithInterval(rel *RelOpNode) (bool, bool) {
	if rel == nil {
		return false, false
	}

	// Construct diff = LHS - RHS
	negRHS := &UnaryOpNode{Op: "-", Expr: rel.RHS}
	diffNode := &AddNode{Terms: []Node{rel.LHS, negRHS}}

	precisions := []int{6, 12, 24}

	for _, prec := range precisions {
		interval, err := EvalNodeInterval(diffNode, prec)
		if err != nil {
			return false, false
		}

		if interval.IsStrictlyPositive() {
			// diff > 0 => LHS > RHS
			switch rel.Op {
			case ">", ">=":
				return true, true
			case "<", "<=", "==":
				return false, true
			case "!=":
				return true, true
			}
		}

		if interval.IsStrictlyNegative() {
			// diff < 0 => LHS < RHS
			switch rel.Op {
			case "<", "<=":
				return true, true
			case ">", ">=", "==":
				return false, true
			case "!=":
				return true, true
			}
		}

		// Interval contains zero: try higher precision
	}

	return false, false
}
