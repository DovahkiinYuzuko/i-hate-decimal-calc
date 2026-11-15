package calc

import (
	"fmt"
	"math/big"
	"strings"

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

// PowIntChecked computes interval raised to integer power k, returning an error on division by zero-containing interval.
func (i RationalInterval) PowIntChecked(k int) (RationalInterval, error) {
	if k == 0 {
		return NewExactRationalInterval(big.NewRat(1, 1)), nil
	}
	if k < 0 {
		inv, err := i.Inv()
		if err != nil {
			return RationalInterval{}, err
		}
		return inv.PowIntChecked(-k)
	}

	powRat := func(r *big.Rat, exp int) *big.Rat {
		num := new(big.Int).Exp(r.Num(), big.NewInt(int64(exp)), nil)
		denom := new(big.Int).Exp(r.Denom(), big.NewInt(int64(exp)), nil)
		return new(big.Rat).SetFrac(num, denom)
	}

	if k%2 == 1 {
		// Odd power: monotonically increasing
		return NewRationalInterval(powRat(i.Low, k), powRat(i.High, k)), nil
	}

	// Even power:
	pLow := powRat(i.Low, k)
	pHigh := powRat(i.High, k)
	if i.ContainsZero() {
		maxVal := pLow
		if pHigh.Cmp(maxVal) > 0 {
			maxVal = pHigh
		}
		return NewRationalInterval(big.NewRat(0, 1), maxVal), nil
	}

	zero := big.NewRat(0, 1)
	if i.Low.Cmp(zero) >= 0 {
		return NewRationalInterval(pLow, pHigh), nil
	}
	return NewRationalInterval(pHigh, pLow), nil
}

// PowInt computes interval raised to integer power k (deprecated for unchecked negative powers).
func (i RationalInterval) PowInt(k int) RationalInterval {
	res, err := i.PowIntChecked(k)
	if err != nil {
		return RationalInterval{Low: big.NewRat(0, 1), High: big.NewRat(0, 1)}
	}
	return res
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

// roundOutwardBinary rounds interval endpoints outward to denominators of 2^precBits
// to prevent exponential bit-length blowup during repeated squaring while strictly preserving enclosure.
func roundOutwardBinary(iv RationalInterval, precBits int) RationalInterval {
	if precBits < 16 {
		precBits = 16
	}
	scaleDenom := new(big.Int).Lsh(big.NewInt(1), uint(precBits))
	roundEndpoint := func(val *big.Rat, up bool) *big.Rat {
		scaled := new(big.Int).Lsh(val.Num(), uint(precBits))
		q, rem := new(big.Int).DivMod(scaled, val.Denom(), new(big.Int))
		if up && rem.Sign() != 0 {
			q.Add(q, big.NewInt(1))
		}
		return new(big.Rat).SetFrac(q, scaleDenom)
	}
	return NewRationalInterval(roundEndpoint(iv.Low, false), roundEndpoint(iv.High, true))
}

// RationalBoundExp computes rigorous rational enclosure for exp(x) with x in Q
// using binary scaling exp(x) = (exp(x / 2^m))^(2^m) so evaluation runs in O(steps + log|x|) steps.
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

	// Binary scaling: choose m >= 0 so y = x / 2^m lies in (0, 1).
	intPart := new(big.Int).Quo(x.Num(), x.Denom())
	m := 0
	if intPart.Sign() > 0 {
		m = intPart.BitLen()
	}

	y := new(big.Rat).Set(x)
	if m > 0 {
		pow2m := new(big.Int).Lsh(big.NewInt(1), uint(m))
		y.Quo(y, new(big.Rat).SetInt(pow2m))
	}

	n := steps + m
	if n < 12 {
		n = 12
	}

	// For y in (0, 1):
	// S_n = sum_{k=0}^n y^k / k!
	// Remainder R_n = sum_{k=n+1}^inf y^k / k! < (y^(n+1) / (n+1)!) / (1 - y/(n+2))
	sum := big.NewRat(0, 1)
	curPow := big.NewRat(1, 1)
	fact := big.NewInt(1)

	for k := 0; k <= n; k++ {
		if k > 0 {
			curPow.Mul(curPow, y)
			fact.Mul(fact, big.NewInt(int64(k)))
		}
		term := new(big.Rat).Quo(curPow, new(big.Rat).SetInt(fact))
		sum.Add(sum, term)
	}

	// Next term: y^(n+1) / (n+1)!
	nextPow := new(big.Rat).Mul(curPow, y)
	nextFact := new(big.Int).Mul(fact, big.NewInt(int64(n+1)))
	nextTerm := new(big.Rat).Quo(nextPow, new(big.Rat).SetInt(nextFact))

	// geomFactor = 1 / (1 - y/(n+2))
	yOverN2 := new(big.Rat).Quo(y, big.NewRat(int64(n+2), 1))
	oneMinus := new(big.Rat).Sub(big.NewRat(1, 1), yOverN2)
	geomFactor := new(big.Rat).Quo(big.NewRat(1, 1), oneMinus)

	rem := new(big.Rat).Mul(nextTerm, geomFactor)

	res := NewRationalInterval(sum, new(big.Rat).Add(sum, rem))
	if m > 0 {
		roundBits := steps*12 + m*4 + 64
		if roundBits > 4096 {
			roundBits = 4096
		}
		res = roundOutwardBinary(res, roundBits)
		for i := 0; i < m; i++ {
			res = roundOutwardBinary(res.Mul(res), roundBits)
		}
	}
	return res
}

func ratFloor(r *big.Rat) *big.Int {
	return new(big.Int).Div(r.Num(), r.Denom())
}

func ratCeil(r *big.Rat) *big.Int {
	q, rem := new(big.Int).DivMod(r.Num(), r.Denom(), new(big.Int))
	if rem.Sign() != 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

func ratNearestInt(r *big.Rat) *big.Int {
	twoNum := new(big.Int).Lsh(r.Num(), 1)
	twoNum.Add(twoNum, r.Denom())
	twoDen := new(big.Int).Lsh(r.Denom(), 1)
	return new(big.Int).Div(twoNum, twoDen)
}

func clampTrigInterval(iv RationalInterval) RationalInterval {
	negOne := big.NewRat(-1, 1)
	posOne := big.NewRat(1, 1)
	low := new(big.Rat).Set(iv.Low)
	if low.Cmp(negOne) < 0 {
		low.Set(negOne)
	}
	high := new(big.Rat).Set(iv.High)
	if high.Cmp(posOne) > 0 {
		high.Set(posOne)
	}
	if low.Cmp(high) > 0 {
		return NewRationalInterval(negOne, posOne)
	}
	return NewRationalInterval(low, high)
}

// sinCosPiSteps chooses Machin series steps scaled modestly for huge argument magnitudes,
// capped to preserve O(1)/bounded runtime and falling back cleanly to [-1, 1] beyond cap.
func sinCosPiSteps(steps int, vals ...*big.Rat) int {
	maxBits := 0
	for _, v := range vals {
		if v == nil || v.Sign() == 0 {
			continue
		}
		bits := v.Num().BitLen() - v.Denom().BitLen()
		if bits > maxBits {
			maxBits = bits
		}
	}
	piSteps := steps
	if maxBits > 32 {
		extra := maxBits / 8
		if extra > 64 {
			extra = 64
		}
		piSteps += extra
	}
	return piSteps
}

// evalAlternatingSinCos computes rigorous enclosure for sin(x) or cos(x) assuming |x| <= 2*pi + 1.
func evalAlternatingSinCos(x *big.Rat, isSin bool, steps int) RationalInterval {
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
		res := evalAlternatingSinCos(negX, isSin, steps)
		if isSin {
			return res.Neg()
		}
		return res
	}

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
	}

	for k := 0; k < maxK; k++ {
		term := new(big.Rat).Quo(curPow, new(big.Rat).SetInt(fact))

		if k%2 == 0 {
			sumHigh = new(big.Rat).Add(sumLow, term)
		} else {
			sumLow = new(big.Rat).Sub(sumHigh, term)
		}

		curPow.Mul(curPow, xSquared)
		if isSin {
			fact.Mul(fact, big.NewInt(int64(2*k+2)))
			fact.Mul(fact, big.NewInt(int64(2*k+3)))
		} else {
			fact.Mul(fact, big.NewInt(int64(2*k+1)))
			fact.Mul(fact, big.NewInt(int64(2*k+2)))
		}
	}

	return clampTrigInterval(NewRationalInterval(sumLow, sumHigh))
}

func containsTrigExtremum(argInt RationalInterval, piInv RationalInterval, offset *big.Rat) bool {
	// x in [a, b] may cross (2*k + offset)*pi iff (argInt / pi - offset) / 2 contains an integer k.
	ratio := argInt.Mul(piInv)
	half := NewExactRationalInterval(big.NewRat(1, 2))
	kRange := ratio.Sub(NewExactRationalInterval(offset)).Mul(half)
	return ratCeil(kRange.Low).Cmp(ratFloor(kRange.High)) <= 0
}

func widenTrigExtrema(argInt RationalInterval, minVal, maxVal *big.Rat, isSin bool, piIv RationalInterval) RationalInterval {
	piInv, err := piIv.Inv()
	if err != nil {
		return NewRationalInterval(big.NewRat(-1, 1), big.NewRat(1, 1))
	}
	var maxOffset, minOffset *big.Rat
	if isSin {
		maxOffset = big.NewRat(1, 2)
		minOffset = big.NewRat(-1, 2)
	} else {
		maxOffset = big.NewRat(0, 1)
		minOffset = big.NewRat(1, 1)
	}
	outLow := new(big.Rat).Set(minVal)
	outHigh := new(big.Rat).Set(maxVal)
	if containsTrigExtremum(argInt, piInv, maxOffset) {
		outHigh = big.NewRat(1, 1)
	}
	if containsTrigExtremum(argInt, piInv, minOffset) {
		outLow = big.NewRat(-1, 1)
	}
	return clampTrigInterval(NewRationalInterval(outLow, outHigh))
}

func evalPointSinCosWithPi(x *big.Rat, isSin bool, steps int, piIv, twoPi RationalInterval) RationalInterval {
	if x.Sign() == 0 {
		if isSin {
			return NewExactRationalInterval(big.NewRat(0, 1))
		}
		return NewExactRationalInterval(big.NewRat(1, 1))
	}

	twoPiMid := new(big.Rat).Add(twoPi.Low, twoPi.High)
	twoPiMid.Mul(twoPiMid, big.NewRat(1, 2))

	k := ratNearestInt(new(big.Rat).Quo(x, twoPiMid))
	if k.Sign() == 0 {
		return evalAlternatingSinCos(x, isSin, steps)
	}

	kIv := NewExactRationalInterval(new(big.Rat).SetInt(k))
	rInt := NewExactRationalInterval(x).Sub(kIv.Mul(twoPi))
	if rInt.Width().Cmp(twoPi.Low) >= 0 {
		return NewRationalInterval(big.NewRat(-1, 1), big.NewRat(1, 1))
	}

	b1 := evalAlternatingSinCos(rInt.Low, isSin, steps)
	b2 := evalAlternatingSinCos(rInt.High, isSin, steps)
	minVal := b1.Low
	if b2.Low.Cmp(minVal) < 0 {
		minVal = b2.Low
	}
	maxVal := b1.High
	if b2.High.Cmp(maxVal) > 0 {
		maxVal = b2.High
	}
	return widenTrigExtrema(rInt, minVal, maxVal, isSin, piIv)
}

func evalIntervalSinCos(argInt RationalInterval, isSin bool, steps int) RationalInterval {
	if steps < 4 {
		steps = 10
	}
	piSteps := sinCosPiSteps(steps, argInt.Low, argInt.High)
	piIv := RationalBoundPi(piSteps)
	twoPi := piIv.Mul(NewExactRationalInterval(big.NewRat(2, 1)))

	if argInt.Width().Cmp(twoPi.Low) >= 0 {
		return NewRationalInterval(big.NewRat(-1, 1), big.NewRat(1, 1))
	}

	b1 := evalPointSinCosWithPi(argInt.Low, isSin, steps, piIv, twoPi)
	b2 := evalPointSinCosWithPi(argInt.High, isSin, steps, piIv, twoPi)
	minVal := b1.Low
	if b2.Low.Cmp(minVal) < 0 {
		minVal = b2.Low
	}
	maxVal := b1.High
	if b2.High.Cmp(maxVal) > 0 {
		maxVal = b2.High
	}
	return widenTrigExtrema(argInt, minVal, maxVal, isSin, piIv)
}

// RationalBoundSinCos computes rigorous rational enclosure for sin(x) or cos(x) with x in Q
// using O(1) mod 2pi range reduction and sound [-1, 1] fallback when reduction width exceeds 2pi.
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
	piSteps := sinCosPiSteps(steps, x)
	piIv := RationalBoundPi(piSteps)
	twoPi := piIv.Mul(NewExactRationalInterval(big.NewRat(2, 1)))
	return evalPointSinCosWithPi(x, isSin, steps, piIv, twoPi)
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
		return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_sqrt_of_negative_rational"))
	}
	if x.Sign() == 0 {
		return NewExactRationalInterval(big.NewRat(0, 1)), nil
	}

	// Precision scale: k bits of precision (clamped to [16, 16384] for guaranteed linear bit complexity)
	k := steps * 4
	if k < 16 {
		k = 16
	}
	if k > 16384 {
		k = 16384
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

// bigIntCbrtFloor computes floor(n^(1/3)) for non-negative integer n using monotonically
// descending integer Newton-Raphson followed by exact cube comparison adjustment.
func bigIntCbrtFloor(n *big.Int) *big.Int {
	if n.Sign() <= 0 {
		return big.NewInt(0)
	}
	exp := (n.BitLen() + 2) / 3
	u := new(big.Int).Lsh(big.NewInt(1), uint(exp))
	two := big.NewInt(2)
	three := big.NewInt(3)
	one := big.NewInt(1)
	u2 := new(big.Int)
	next := new(big.Int)

	for {
		u2.Mul(u, u)
		next.Quo(n, u2)
		next.Add(next, new(big.Int).Mul(two, u))
		next.Quo(next, three)
		if next.Cmp(u) >= 0 {
			break
		}
		u.Set(next)
	}

	cube := new(big.Int)
	for {
		cube.Mul(u, u)
		cube.Mul(cube, u)
		if cube.Cmp(n) <= 0 {
			break
		}
		u.Sub(u, one)
	}
	uPlus1 := new(big.Int)
	for {
		uPlus1.Add(u, one)
		cube.Mul(uPlus1, uPlus1)
		cube.Mul(cube, uPlus1)
		if cube.Cmp(n) > 0 {
			break
		}
		u.Set(uPlus1)
	}
	return u
}

// RationalBoundCbrt computes certified rational upper and lower bounds for cbrt(x)
// using integer cube root with dyadic scaling, guaranteeing O(k) bit complexity.
func RationalBoundCbrt(x *big.Rat, steps int) (RationalInterval, error) {
	if x.Sign() == 0 {
		return NewExactRationalInterval(big.NewRat(0, 1)), nil
	}
	neg := x.Sign() < 0
	absX := new(big.Rat).Abs(x)

	k := steps * 4
	if k < 16 {
		k = 16
	}
	if k > 16384 {
		k = 16384
	}

	scaledNum := new(big.Int).Lsh(absX.Num(), uint(3*k))
	div := new(big.Int).Quo(scaledNum, absX.Denom())

	m := bigIntCbrtFloor(div)
	mPlusOne := new(big.Int).Add(m, big.NewInt(1))
	denom := new(big.Int).Lsh(big.NewInt(1), uint(k))

	low := new(big.Rat).SetFrac(m, denom)
	high := new(big.Rat).SetFrac(mPlusOne, denom)

	if neg {
		return NewRationalInterval(new(big.Rat).Neg(high), new(big.Rat).Neg(low)), nil
	}
	return NewRationalInterval(low, high), nil
}

// EvalNodeInterval recursively evaluates an AST expression into a RationalInterval enclosure.
func EvalNodeInterval(n Node, steps int) (RationalInterval, error) {
	if n == nil {
		return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_nil_node"))
	}

	switch node := n.(type) {
	case *RationalNode:
		return NewExactRationalInterval(node.Val), nil

	case *IntervalNode:
		return NewRationalInterval(node.Low, node.High), nil

	case *SqrtNode:
		sub, err := EvalNodeInterval(node.Radicand, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		if sub.Low.Sign() < 0 {
			return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_sqrt_of_negative_interval"))
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
		return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_unsupported_constant_in_interval_evaluation", node.Name))

	case *VarNode:
		if node.Name == "pi" || node.Name == "π" {
			return RationalBoundPi(steps), nil
		}
		if node.Name == "e" {
			return RationalBoundE(steps), nil
		}
		return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_unsupported_free_variable_in_interval", node.Name))

	case *UnaryOpNode:
		sub, err := EvalNodeInterval(node.Expr, steps)
		if err != nil {
			return RationalInterval{}, err
		}
		if node.Op == "-" {
			return sub.Neg(), nil

		}
		if node.Op == "!" {
			if sub.Low.Cmp(sub.High) != 0 || !sub.Low.IsInt() || sub.Low.Sign() < 0 {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("approx.err_factorial_requires_non_negative_integer"))
			}
			if !sub.Low.Num().IsInt64() || sub.Low.Num().Int64() > 10000 {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_factorial_too_large", sub.Low.Num()))
			}
			nVal := sub.Low.Num().Int64()
			fact := big.NewInt(1)
			for i := int64(2); i <= nVal; i++ {
				fact.Mul(fact, big.NewInt(i))
			}
			return NewExactRationalInterval(new(big.Rat).SetInt(fact)), nil
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
			if !expRat.Val.Num().IsInt64() {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_unsupported_node_for_interval_evaluation", node.String()))
			}
			exp64 := expRat.Val.Num().Int64()
			expInt := int(exp64)
			if int64(expInt) != exp64 {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_unsupported_node_for_interval_evaluation", node.String()))
			}
			baseInt, err := EvalNodeInterval(node.Base, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			return baseInt.PowIntChecked(expInt)
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
			return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_non_positive_base_in_real", baseInt.String()))
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
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_exp_expects_1_argument"))
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
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_ln_expects_1_argument"))
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			if !argInt.IsStrictlyPositive() {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_ln_argument_must_be_strictly", argInt.String()))
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
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_sin_expects_1_argument"))
			}
			if r, isR := node.Args[0].(*RationalNode); isR {
				return RationalBoundSinCos(r.Val, true, steps), nil
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			return evalIntervalSinCos(argInt, true, steps), nil

		case "cos":
			if len(node.Args) != 1 {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_cos_expects_1_argument"))
			}
			if r, isR := node.Args[0].(*RationalNode); isR {
				return RationalBoundSinCos(r.Val, false, steps), nil
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			return evalIntervalSinCos(argInt, false, steps), nil

		case "cbrt":
			if len(node.Args) != 1 {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("approx.err_invalid_function_call", "cbrt"))
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			lowBound, err := RationalBoundCbrt(argInt.Low, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			highBound, err := RationalBoundCbrt(argInt.High, steps)
			if err != nil {
				return RationalInterval{}, err
			}
			return NewRationalInterval(lowBound.Low, highBound.High), nil

		case "abs":
			if len(node.Args) != 1 {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("approx.err_invalid_function_call", "abs"))
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			if argInt.Low.Sign() >= 0 {
				return argInt, nil
			}
			if argInt.High.Sign() <= 0 {
				return argInt.Neg(), nil
			}
			negLow := new(big.Rat).Neg(argInt.Low)
			maxVal := argInt.High
			if negLow.Cmp(maxVal) > 0 {
				maxVal = negLow
			}
			return NewRationalInterval(big.NewRat(0, 1), maxVal), nil

		case "sqrt":
			if len(node.Args) != 1 {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_sqrt_expects_1_argument"))
			}
			argInt, err := EvalNodeInterval(node.Args[0], steps)
			if err != nil {
				return RationalInterval{}, err
			}
			if argInt.Low.Sign() < 0 {
				return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_sqrt_of_negative_interval"))
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

	return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_unsupported_node_for_interval_evaluation", n.String()))
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

// AdaptiveRefineInterval iteratively refines a RationalInterval enclosure for AST node n
// until the interval width (High - Low) is less than or equal to epsilon.
// Refinement is governed by IntervalRefinerFSM to guarantee safe termination and trace history.
func AdaptiveRefineInterval(n Node, eps *big.Rat, maxSteps int) (RationalInterval, error) {
	if eps == nil {
		eps = big.NewRat(1, 1000000) // Default 10^-6
	}
	if eps.Sign() <= 0 {
		return RationalInterval{}, fmt.Errorf("%s", i18n.T("interval.err_invalid_epsilon", eps.RatString()))
	}
	if maxSteps <= 0 {
		maxSteps = 100
	}

	fsm := NewIntervalRefinerFSM(maxSteps)
	curSteps := 10

	if err := fsm.TransitionTo(IntervalStateInitialEnclosure); err != nil {
		return RationalInterval{}, err
	}

	interval, err := EvalNodeInterval(n, curSteps)
	if err != nil {
		_ = fsm.TransitionTo(IntervalStateFailed)
		return RationalInterval{}, err
	}

	for {
		if err := fsm.TransitionTo(IntervalStateCheckingTolerance); err != nil {
			return interval, err
		}

		width := interval.Width()
		if width.Cmp(eps) <= 0 {
			if err := fsm.TransitionTo(IntervalStateConverged); err != nil {
				return interval, err
			}
			return interval, nil
		}

		if curSteps >= maxSteps {
			if err := fsm.TransitionTo(IntervalStateMaxStepsReached); err != nil {
				return interval, err
			}
			return interval, nil
		}

		if err := fsm.TransitionTo(IntervalStateRefining); err != nil {
			return interval, err
		}

		nextSteps := curSteps * 2
		if nextSteps > maxSteps {
			nextSteps = maxSteps
		}
		curSteps = nextSteps

		interval, err = EvalNodeInterval(n, curSteps)
		if err != nil {
			_ = fsm.TransitionTo(IntervalStateFailed)
			return RationalInterval{}, err
		}
	}
}

// FormatDecimalEnclosure formats a RationalInterval into a guaranteed decimal interval enclosure [L_dec, R_dec]
// by rounding the lower bound towards -infinity (Floor) and the upper bound towards +infinity (Ceil).
// Floating-point arithmetic is completely avoided; all calculations use big.Int Euclidean division.
func FormatDecimalEnclosure(interval RationalInterval, decimalPlaces int) string {
	if decimalPlaces < 1 {
		decimalPlaces = 6
	}
	if decimalPlaces > 100 {
		decimalPlaces = 100
	}

	ten := big.NewInt(10)
	scale := new(big.Int).Exp(ten, big.NewInt(int64(decimalPlaces)), nil)

	formatBound := func(r *big.Rat, isUpper bool) string {
		p := new(big.Int).Mul(r.Num(), scale)
		q := r.Denom()

		quot := new(big.Int)
		rem := new(big.Int)
		quot.QuoRem(p, q, rem)

		// Directed outward rounding:
		// For lower bound (floor toward -inf):
		// If rem != 0 and r < 0, subtract 1 from quotient.
		// For upper bound (ceil toward +inf):
		// If rem != 0 and r > 0, add 1 to quotient.
		if rem.Sign() != 0 {
			if isUpper {
				if r.Sign() > 0 {
					quot.Add(quot, big.NewInt(1))
				}
			} else {
				if r.Sign() < 0 {
					quot.Sub(quot, big.NewInt(1))
				}
			}
		}

		neg := quot.Sign() < 0
		absQuot := new(big.Int).Abs(quot)
		s := absQuot.String()

		if len(s) <= decimalPlaces {
			s = strings.Repeat("0", decimalPlaces-len(s)+1) + s
		}

		splitIdx := len(s) - decimalPlaces
		intPart := s[:splitIdx]
		fracPart := s[splitIdx:]

		res := intPart + "." + fracPart
		if neg {
			res = "-" + res
		}
		return res
	}

	lowStr := formatBound(interval.Low, false)
	highStr := formatBound(interval.High, true)
	return fmt.Sprintf("[%s, %s]", lowStr, highStr)
}

