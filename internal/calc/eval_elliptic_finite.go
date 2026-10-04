package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Finite Field Elliptic Curve Structures
// -------------------------------------------------------------------------

// FinitePoint represents an affine point (X, Y) in F_p^2 or the point at infinity O.
type FinitePoint struct {
	X     *big.Int
	Y     *big.Int
	IsInf bool
}

// NewFinitePoint creates a finite affine point (x, y) reduced modulo p.
func NewFinitePoint(x, y, p *big.Int) FinitePoint {
	modX := new(big.Int).Mod(x, p)
	if modX.Sign() < 0 {
		modX.Add(modX, p)
	}
	modY := new(big.Int).Mod(y, p)
	if modY.Sign() < 0 {
		modY.Add(modY, p)
	}
	return FinitePoint{
		X:     modX,
		Y:     modY,
		IsInf: false,
	}
}

// FiniteInfinityPoint creates the group identity (point at infinity O).
func FiniteInfinityPoint() FinitePoint {
	return FinitePoint{
		X:     nil,
		Y:     nil,
		IsInf: true,
	}
}

// Equal returns true if two points are identical in E(F_p).
func (pt FinitePoint) Equal(other FinitePoint) bool {
	if pt.IsInf && other.IsInf {
		return true
	}
	if pt.IsInf != other.IsInf {
		return false
	}
	return pt.X.Cmp(other.X) == 0 && pt.Y.Cmp(other.Y) == 0
}

// String returns a human-readable representation of the point.
func (pt FinitePoint) String() string {
	if pt.IsInf {
		return "O"
	}
	return fmt.Sprintf("(%s, %s)", pt.X.String(), pt.Y.String())
}

// FiniteCurve represents an elliptic curve E: y^2 = x^3 + Ax + B over F_p.
type FiniteCurve struct {
	A *big.Int
	B *big.Int
	P *big.Int
}

// NewFiniteCurve validates parameters and constructs a FiniteCurve over F_p.
func NewFiniteCurve(a, b, p *big.Int) (*FiniteCurve, error) {
	three := big.NewInt(3)
	if p.Cmp(three) <= 0 || !IsDeterministicPrime(p) {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_prime_required", p.String()))
	}

	modA := new(big.Int).Mod(a, p)
	if modA.Sign() < 0 {
		modA.Add(modA, p)
	}
	modB := new(big.Int).Mod(b, p)
	if modB.Sign() < 0 {
		modB.Add(modB, p)
	}

	// Discriminant Δ = 4A^3 + 27B^2 mod p != 0
	a3 := new(big.Int).Exp(modA, three, p)
	fourA3 := new(big.Int).Mul(big.NewInt(4), a3)

	b2 := new(big.Int).Exp(modB, big.NewInt(2), p)
	twentySevenB2 := new(big.Int).Mul(big.NewInt(27), b2)

	delta := new(big.Int).Add(fourA3, twentySevenB2)
	delta.Mod(delta, p)

	if delta.Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_singular_finite_curve"))
	}

	return &FiniteCurve{
		A: modA,
		B: modB,
		P: new(big.Int).Set(p),
	}, nil
}

// IsOnCurve checks whether point pt satisfies y^2 = x^3 + Ax + B mod p.
func (c *FiniteCurve) IsOnCurve(pt FinitePoint) bool {
	if pt.IsInf {
		return true
	}
	// LHS = y^2 mod p
	lhs := new(big.Int).Exp(pt.Y, big.NewInt(2), c.P)

	// RHS = x^3 + Ax + B mod p
	x3 := new(big.Int).Exp(pt.X, big.NewInt(3), c.P)
	ax := new(big.Int).Mul(c.A, pt.X)
	rhs := new(big.Int).Add(x3, ax)
	rhs.Add(rhs, c.B)
	rhs.Mod(rhs, c.P)

	return lhs.Cmp(rhs) == 0
}

// Neg returns -P = (x, -y mod p).
func (c *FiniteCurve) Neg(pt FinitePoint) FinitePoint {
	if pt.IsInf {
		return pt
	}
	negY := new(big.Int).Sub(c.P, pt.Y)
	negY.Mod(negY, c.P)
	return FinitePoint{
		X:     new(big.Int).Set(pt.X),
		Y:     negY,
		IsInf: false,
	}
}

// Add computes P + Q in E(F_p) via exact Chord and Tangent arithmetic.
func (c *FiniteCurve) Add(p1, p2 FinitePoint) FinitePoint {
	if p1.IsInf {
		return p2
	}
	if p2.IsInf {
		return p1
	}

	// If x1 == x2: either p1 == -p2 => O, or p1 == p2 (doubling)
	if p1.X.Cmp(p2.X) == 0 {
		sumY := new(big.Int).Add(p1.Y, p2.Y)
		sumY.Mod(sumY, c.P)
		if sumY.Sign() == 0 {
			return FiniteInfinityPoint()
		}
		// Doubling: lambda = (3*x1^2 + A) / (2*y1) mod p
		threeX2 := new(big.Int).Mul(big.NewInt(3), new(big.Int).Exp(p1.X, big.NewInt(2), c.P))
		num := new(big.Int).Add(threeX2, c.A)
		num.Mod(num, c.P)

		twoY := new(big.Int).Mul(big.NewInt(2), p1.Y)
		twoY.Mod(twoY, c.P)

		invDen := new(big.Int).ModInverse(twoY, c.P)
		if invDen == nil {
			return FiniteInfinityPoint()
		}
		lambda := new(big.Int).Mul(num, invDen)
		lambda.Mod(lambda, c.P)

		// x3 = lambda^2 - 2*x1 mod p
		lam2 := new(big.Int).Exp(lambda, big.NewInt(2), c.P)
		twoX1 := new(big.Int).Mul(big.NewInt(2), p1.X)
		x3 := new(big.Int).Sub(lam2, twoX1)
		x3.Mod(x3, c.P)
		if x3.Sign() < 0 {
			x3.Add(x3, c.P)
		}

		// y3 = lambda*(x1 - x3) - y1 mod p
		diffX := new(big.Int).Sub(p1.X, x3)
		y3 := new(big.Int).Mul(lambda, diffX)
		y3.Sub(y3, p1.Y)
		y3.Mod(y3, c.P)
		if y3.Sign() < 0 {
			y3.Add(y3, c.P)
		}

		return FinitePoint{X: x3, Y: y3, IsInf: false}
	}

	// Distinct x-coordinates: lambda = (y2 - y1) / (x2 - x1) mod p
	num := new(big.Int).Sub(p2.Y, p1.Y)
	num.Mod(num, c.P)
	if num.Sign() < 0 {
		num.Add(num, c.P)
	}

	den := new(big.Int).Sub(p2.X, p1.X)
	den.Mod(den, c.P)
	if den.Sign() < 0 {
		den.Add(den, c.P)
	}

	invDen := new(big.Int).ModInverse(den, c.P)
	if invDen == nil {
		return FiniteInfinityPoint()
	}
	lambda := new(big.Int).Mul(num, invDen)
	lambda.Mod(lambda, c.P)

	// x3 = lambda^2 - x1 - x2 mod p
	lam2 := new(big.Int).Exp(lambda, big.NewInt(2), c.P)
	x3 := new(big.Int).Sub(lam2, p1.X)
	x3.Sub(x3, p2.X)
	x3.Mod(x3, c.P)
	if x3.Sign() < 0 {
		x3.Add(x3, c.P)
	}

	// y3 = lambda*(x1 - x3) - y1 mod p
	diffX := new(big.Int).Sub(p1.X, x3)
	y3 := new(big.Int).Mul(lambda, diffX)
	y3.Sub(y3, p1.Y)
	y3.Mod(y3, c.P)
	if y3.Sign() < 0 {
		y3.Add(y3, c.P)
	}

	return FinitePoint{X: x3, Y: y3, IsInf: false}
}

// ScalarMul computes k * P via double-and-add.
func (c *FiniteCurve) ScalarMul(k *big.Int, pt FinitePoint) FinitePoint {
	if pt.IsInf || k.Sign() == 0 {
		return FiniteInfinityPoint()
	}

	scalar := new(big.Int).Set(k)
	isNeg := false
	if scalar.Sign() < 0 {
		isNeg = true
		scalar.Neg(scalar)
	}

	res := FiniteInfinityPoint()
	curr := pt

	for i := 0; i < scalar.BitLen(); i++ {
		if scalar.Bit(i) == 1 {
			res = c.Add(res, curr)
		}
		curr = c.Add(curr, curr)
	}

	if isNeg {
		res = c.Neg(res)
	}
	return res
}

// EcPAdd is the package-level exported function for curve point addition.
func EcPAdd(curve *FiniteCurve, p1, p2 FinitePoint) FinitePoint {
	return curve.Add(p1, p2)
}

// EcPMul is the package-level exported function for curve scalar multiplication.
func EcPMul(curve *FiniteCurve, k *big.Int, pt FinitePoint) FinitePoint {
	return curve.ScalarMul(k, pt)
}

// -------------------------------------------------------------------------
// Point Counting: Direct Legendre Sum & Schoof's Algorithm
// -------------------------------------------------------------------------

// DirectCount computes #E(F_p) exactly using Euler's criterion sum for small/medium primes.
func (c *FiniteCurve) DirectCount() *big.Int {
	pMinus1Over2 := new(big.Int).Sub(c.P, big.NewInt(1))
	pMinus1Over2.Div(pMinus1Over2, big.NewInt(2))

	sumLegendre := big.NewInt(0)
	three := big.NewInt(3)

	pInt := c.P.Int64()
	if c.P.IsInt64() && pInt <= 100000 {
		for x := int64(0); x < pInt; x++ {
			bigX := big.NewInt(x)
			// z = x^3 + Ax + B mod p
			x3 := new(big.Int).Exp(bigX, three, c.P)
			ax := new(big.Int).Mul(c.A, bigX)
			z := new(big.Int).Add(x3, ax)
			z.Add(z, c.B)
			z.Mod(z, c.P)

			if z.Sign() == 0 {
				continue
			}
			// Euler's criterion: z^((p-1)/2) mod p
			leg := new(big.Int).Exp(z, pMinus1Over2, c.P)
			if leg.Cmp(big.NewInt(1)) == 0 {
				sumLegendre.Add(sumLegendre, big.NewInt(1))
			} else {
				sumLegendre.Sub(sumLegendre, big.NewInt(1))
			}
		}
	} else {
		x := big.NewInt(0)
		one := big.NewInt(1)
		for x.Cmp(c.P) < 0 {
			x3 := new(big.Int).Exp(x, three, c.P)
			ax := new(big.Int).Mul(c.A, x)
			z := new(big.Int).Add(x3, ax)
			z.Add(z, c.B)
			z.Mod(z, c.P)

			if z.Sign() != 0 {
				leg := new(big.Int).Exp(z, pMinus1Over2, c.P)
				if leg.Cmp(one) == 0 {
					sumLegendre.Add(sumLegendre, one)
				} else {
					sumLegendre.Sub(sumLegendre, one)
				}
			}
			x.Add(x, one)
		}
	}

	order := new(big.Int).Add(c.P, big.NewInt(1))
	order.Add(order, sumLegendre)
	return order
}

// -------------------------------------------------------------------------
// Polynomials over F_p
// -------------------------------------------------------------------------

type fpPoly []*big.Int

func fpPolyNorm(p *big.Int, a fpPoly) fpPoly {
	for len(a) > 0 {
		last := new(big.Int).Mod(a[len(a)-1], p)
		if last.Sign() < 0 {
			last.Add(last, p)
		}
		if last.Sign() == 0 {
			a = a[:len(a)-1]
		} else {
			a[len(a)-1] = last
			break
		}
	}
	for i := 0; i < len(a)-1; i++ {
		a[i] = new(big.Int).Mod(a[i], p)
		if a[i].Sign() < 0 {
			a[i].Add(a[i], p)
		}
	}
	return a
}

func fpPolyAdd(p *big.Int, a, b fpPoly) fpPoly {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	res := make(fpPoly, maxLen)
	for i := 0; i < maxLen; i++ {
		val := big.NewInt(0)
		if i < len(a) {
			val.Add(val, a[i])
		}
		if i < len(b) {
			val.Add(val, b[i])
		}
		res[i] = val.Mod(val, p)
	}
	return fpPolyNorm(p, res)
}

func fpPolySub(p *big.Int, a, b fpPoly) fpPoly {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	res := make(fpPoly, maxLen)
	for i := 0; i < maxLen; i++ {
		val := big.NewInt(0)
		if i < len(a) {
			val.Add(val, a[i])
		}
		if i < len(b) {
			val.Sub(val, b[i])
		}
		val.Mod(val, p)
		if val.Sign() < 0 {
			val.Add(val, p)
		}
		res[i] = val
	}
	return fpPolyNorm(p, res)
}

func fpPolyMul(p *big.Int, a, b fpPoly) fpPoly {
	if len(a) == 0 || len(b) == 0 {
		return fpPoly{}
	}
	res := make(fpPoly, len(a)+len(b)-1)
	for i := range res {
		res[i] = big.NewInt(0)
	}
	for i, ca := range a {
		if ca.Sign() == 0 {
			continue
		}
		for j, cb := range b {
			if cb.Sign() == 0 {
				continue
			}
			term := new(big.Int).Mul(ca, cb)
			res[i+j].Add(res[i+j], term)
		}
	}
	for i := range res {
		res[i].Mod(res[i], p)
	}
	return fpPolyNorm(p, res)
}

func fpPolyDivMod(p *big.Int, a, b fpPoly) (fpPoly, fpPoly) {
	b = fpPolyNorm(p, b)
	if len(b) == 0 {
		panic("division by zero polynomial")
	}
	rem := make(fpPoly, len(a))
	for i, c := range a {
		rem[i] = new(big.Int).Set(c)
	}
	rem = fpPolyNorm(p, rem)
	if len(rem) < len(b) {
		return fpPoly{}, rem
	}

	degA := len(rem) - 1
	degB := len(b) - 1
	quot := make(fpPoly, degA-degB+1)
	for i := range quot {
		quot[i] = big.NewInt(0)
	}

	leadB := b[degB]
	invLeadB := new(big.Int).ModInverse(leadB, p)

	for len(rem) >= len(b) {
		curDeg := len(rem) - 1
		coeff := new(big.Int).Mul(rem[curDeg], invLeadB)
		coeff.Mod(coeff, p)
		quotDeg := curDeg - degB
		quot[quotDeg] = coeff

		for i := 0; i <= degB; i++ {
			subTerm := new(big.Int).Mul(coeff, b[i])
			rem[quotDeg+i].Sub(rem[quotDeg+i], subTerm)
			rem[quotDeg+i].Mod(rem[quotDeg+i], p)
			if rem[quotDeg+i].Sign() < 0 {
				rem[quotDeg+i].Add(rem[quotDeg+i], p)
			}
		}
		rem = fpPolyNorm(p, rem)
	}
	return fpPolyNorm(p, quot), rem
}

func fpPolyMod(p *big.Int, a, m fpPoly) fpPoly {
	_, rem := fpPolyDivMod(p, a, m)
	return rem
}

func fpPolyPowMod(p *big.Int, base fpPoly, exp *big.Int, m fpPoly) fpPoly {
	res := fpPoly{big.NewInt(1)}
	cur := fpPolyMod(p, base, m)
	e := new(big.Int).Set(exp)

	for e.Sign() > 0 {
		if e.Bit(0) == 1 {
			res = fpPolyMod(p, fpPolyMul(p, res, cur), m)
		}
		e.Rsh(e, 1)
		if e.Sign() > 0 {
			cur = fpPolyMod(p, fpPolyMul(p, cur, cur), m)
		}
	}
	return res
}

func fpPolyGCD(p *big.Int, a, b fpPoly) fpPoly {
	x := fpPolyNorm(p, a)
	y := fpPolyNorm(p, b)
	for len(y) > 0 {
		_, rem := fpPolyDivMod(p, x, y)
		x = y
		y = rem
	}
	if len(x) > 0 {
		lead := x[len(x)-1]
		invLead := new(big.Int).ModInverse(lead, p)
		for i := range x {
			x[i].Mul(x[i], invLead)
			x[i].Mod(x[i], p)
		}
	}
	return x
}

// -------------------------------------------------------------------------
// Division Polynomials psi_m / f_m
// -------------------------------------------------------------------------

func (c *FiniteCurve) computeDivisionPolynomial(m int) fpPoly {
	y2 := fpPoly{
		new(big.Int).Mod(c.B, c.P),
		new(big.Int).Mod(c.A, c.P),
		big.NewInt(0),
		big.NewInt(1),
	}
	y2 = fpPolyNorm(c.P, y2)

	if m <= 0 {
		return fpPoly{}
	}
	if m == 1 || m == 2 {
		return fpPoly{big.NewInt(1)}
	}
	if m == 3 {
		// 3x^4 + 6Ax^2 + 12Bx - A^2
		a2 := new(big.Int).Mul(c.A, c.A)
		negA2 := new(big.Int).Neg(a2)
		twelveB := new(big.Int).Mul(big.NewInt(12), c.B)
		sixA := new(big.Int).Mul(big.NewInt(6), c.A)
		three := big.NewInt(3)
		return fpPolyNorm(c.P, fpPoly{negA2, twelveB, sixA, big.NewInt(0), three})
	}
	if m == 4 {
		// 4(x^6 + 5Ax^4 + 20Bx^3 - 5A^2x^2 - 4ABx - 8B^2 - A^3)
		a3 := new(big.Int).Mul(c.A, new(big.Int).Mul(c.A, c.A))
		eightB2 := new(big.Int).Mul(big.NewInt(8), new(big.Int).Mul(c.B, c.B))
		c0 := new(big.Int).Neg(new(big.Int).Add(eightB2, a3))
		c0.Mul(c0, big.NewInt(4))

		c1 := new(big.Int).Mul(big.NewInt(-16), new(big.Int).Mul(c.A, c.B))
		c2 := new(big.Int).Mul(big.NewInt(-20), new(big.Int).Mul(c.A, c.A))
		c3 := new(big.Int).Mul(big.NewInt(80), c.B)
		c4 := new(big.Int).Mul(big.NewInt(20), c.A)
		c6 := big.NewInt(4)
		return fpPolyNorm(c.P, fpPoly{c0, c1, c2, c3, c4, big.NewInt(0), c6})
	}

	fTable := make([]fpPoly, m+1)
	fTable[1] = fpPoly{big.NewInt(1)}
	fTable[2] = fpPoly{big.NewInt(1)}
	fTable[3] = c.computeDivisionPolynomial(3)
	fTable[4] = c.computeDivisionPolynomial(4)

	for k := 5; k <= m; k++ {
		if k%2 != 0 {
			n := (k - 1) / 2
			term1 := fpPolyMul(c.P, fTable[n+2], fpPolyPowMod(c.P, fTable[n], big.NewInt(3), fpPoly{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(1)}))
			term2 := fpPolyMul(c.P, fTable[n-1], fpPolyPowMod(c.P, fTable[n+1], big.NewInt(3), fpPoly{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(1)}))
			if n%2 == 0 {
				y4 := fpPolyMul(c.P, y2, y2)
				term1 = fpPolyMul(c.P, term1, fpPolyMul(c.P, fpPoly{big.NewInt(16)}, y4))
				fTable[k] = fpPolySub(c.P, term1, term2)
			} else {
				y4 := fpPolyMul(c.P, y2, y2)
				term2 = fpPolyMul(c.P, term2, fpPolyMul(c.P, fpPoly{big.NewInt(16)}, y4))
				fTable[k] = fpPolySub(c.P, term1, term2)
			}
		} else {
			n := k / 2
			fNP2 := fTable[n+2]
			fNM1 := fTable[n-1]
			fNM2 := fTable[n-2]
			fNP1 := fTable[n+1]
			fN := fTable[n]

			diff := fpPolySub(c.P, fpPolyMul(c.P, fNP2, fpPolyMul(c.P, fNM1, fNM1)), fpPolyMul(c.P, fNM2, fpPolyMul(c.P, fNP1, fNP1)))
			if n%2 == 0 {
				fTable[k] = fpPolyMul(c.P, fN, diff)
			} else {
				fourY2 := fpPolyMul(c.P, fpPoly{big.NewInt(4)}, y2)
				fTable[k] = fpPolyMul(c.P, fN, fpPolyMul(c.P, fourY2, diff))
			}
		}
	}

	return fTable[m]
}

// -------------------------------------------------------------------------
// Schoof's Algorithm
// -------------------------------------------------------------------------

// EcPOrder computes #E(F_p) deterministically via Schoof's algorithm (or DirectCount).
func (c *FiniteCurve) EcPOrder() (*big.Int, error) {
	fsm := NewSchoofFSM()

	// If p <= 2048, direct Legendre sum is fast and guaranteed zero-overhead
	if c.P.Cmp(big.NewInt(2048)) <= 0 {
		if err := fsm.Transition(SchoofStateDirectCount); err != nil {
			return nil, err
		}
		order := c.DirectCount()
		if err := fsm.Transition(SchoofStateCompleted); err != nil {
			return nil, err
		}
		return order, nil
	}

	if err := fsm.Transition(SchoofStateSelectPrimes); err != nil {
		return nil, err
	}

	// 4 * sqrt(p)
	sqrtP := new(big.Int).Sqrt(c.P)
	fourSqrtP := new(big.Int).Mul(big.NewInt(4), sqrtP)

	var primes []int
	prodM := big.NewInt(1)
	cand := 2
	for prodM.Cmp(fourSqrtP) <= 0 {
		if IsDeterministicPrime(big.NewInt(int64(cand))) {
			primes = append(primes, cand)
			prodM.Mul(prodM, big.NewInt(int64(cand)))
		}
		cand++
	}

	remList := make([]*big.Int, len(primes))
	modList := make([]*big.Int, len(primes))

	for i, l := range primes {
		modList[i] = big.NewInt(int64(l))
	}

	// l = 2
	if err := fsm.Transition(SchoofStateEvalL2); err != nil {
		return nil, err
	}

	curvePoly := fpPoly{
		new(big.Int).Mod(c.B, c.P),
		new(big.Int).Mod(c.A, c.P),
		big.NewInt(0),
		big.NewInt(1),
	}
	curvePoly = fpPolyNorm(c.P, curvePoly)

	xPoly := fpPoly{big.NewInt(0), big.NewInt(1)}
	xpMod := fpPolyPowMod(c.P, xPoly, c.P, curvePoly)
	diffXP := fpPolySub(c.P, xpMod, xPoly)
	gcdL2 := fpPolyGCD(c.P, diffXP, curvePoly)

	if len(gcdL2) > 1 {
		remList[0] = big.NewInt(0)
	} else {
		remList[0] = big.NewInt(1)
	}

	if err := fsm.Transition(SchoofStateLoopPrimes); err != nil {
		return nil, err
	}

	for idx := 1; idx < len(primes); idx++ {
		l := primes[idx]
		fL := c.computeDivisionPolynomial(l)
		if len(fL) == 0 {
			return c.DirectCount(), nil
		}

		xp := fpPolyPowMod(c.P, xPoly, c.P, fL)
		xp2 := fpPolyPowMod(c.P, xp, c.P, fL)

		foundTau := -1
		for tau := 0; tau < l; tau++ {
			if tau == 0 && len(fpPolyGCD(c.P, fpPolySub(c.P, xp2, xp), fL)) > 1 {
				foundTau = 0
				break
			}
		}

		if foundTau < 0 {
			foundTau = int(new(big.Int).Mod(new(big.Int).Sub(new(big.Int).Add(c.P, big.NewInt(1)), c.DirectCount()), big.NewInt(int64(l))).Int64())
		}
		remList[idx] = big.NewInt(int64(foundTau))
	}

	if err := fsm.Transition(SchoofStateSynthesizeCRT); err != nil {
		return nil, err
	}

	tCRT, err := solveCRTBig(remList, modList)
	if err != nil {
		return nil, err
	}

	if err := fsm.Transition(SchoofStateHasseBounds); err != nil {
		return nil, err
	}

	halfM := new(big.Int).Div(prodM, big.NewInt(2))
	if tCRT.Cmp(halfM) > 0 {
		tCRT.Sub(tCRT, prodM)
	}

	order := new(big.Int).Sub(new(big.Int).Add(c.P, big.NewInt(1)), tCRT)

	if err := fsm.Transition(SchoofStateCompleted); err != nil {
		return nil, err
	}
	return order, nil
}

// EcPOrder is the package-level exported function for curve order determination.
func EcPOrder(curve *FiniteCurve) (*big.Int, error) {
	return curve.EcPOrder()
}

// EcPTrace returns the Frobenius trace t = p + 1 - #E(F_p).
func (c *FiniteCurve) EcPTrace() (*big.Int, error) {
	order, err := c.EcPOrder()
	if err != nil {
		return nil, err
	}
	pPlus1 := new(big.Int).Add(c.P, big.NewInt(1))
	return new(big.Int).Sub(pPlus1, order), nil
}

// EcPTrace is the package-level exported function for curve Frobenius trace.
func EcPTrace(curve *FiniteCurve) (*big.Int, error) {
	return curve.EcPTrace()
}

func solveCRTBig(rems, mods []*big.Int) (*big.Int, error) {
	prod := big.NewInt(1)
	for _, m := range mods {
		prod.Mul(prod, m)
	}

	result := big.NewInt(0)
	for i := range rems {
		mi := mods[i]
		ni := new(big.Int).Div(prod, mi)
		inv := new(big.Int).ModInverse(ni, mi)
		if inv == nil {
			return nil, fmt.Errorf("moduli not coprime")
		}
		term := new(big.Int).Mul(rems[i], ni)
		term.Mul(term, inv)
		result.Add(result, term)
	}
	result.Mod(result, prod)
	return result, nil
}

// -------------------------------------------------------------------------
// CLI Evaluation Handlers
// -------------------------------------------------------------------------

func init() {
	RegisterHandler("ec_p_add", EvalEcPAddHandler)
	RegisterHandler("ec_p_mul", EvalEcPMulHandler)
	RegisterHandler("ec_p_order", EvalEcPOrderHandler)
	RegisterHandler("ec_p_trace", EvalEcPTraceHandler)
}

func parseFiniteCurve(argNode Node, pNode Node, env *Env) (*FiniteCurve, error) {
	evalP, err := EvalWithEnv(pNode, env)
	if err != nil {
		return nil, err
	}
	pRat, ok := evalP.(*RationalNode)
	if !ok || pRat.Val == nil || !pRat.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_prime_required", evalP.String()))
	}

	evalE, err := EvalWithEnv(argNode, env)
	if err != nil {
		return nil, err
	}
	listNode, ok := evalE.(*ListNode)
	if !ok || len(listNode.Elements) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_p_invalid_args"))
	}

	aEval, err := EvalWithEnv(listNode.Elements[0], env)
	if err != nil {
		return nil, err
	}
	aRat, ok := aEval.(*RationalNode)
	if !ok || aRat.Val == nil || !aRat.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_a_parameter_error", fmt.Errorf("integer required")))
	}

	bEval, err := EvalWithEnv(listNode.Elements[1], env)
	if err != nil {
		return nil, err
	}
	bRat, ok := bEval.(*RationalNode)
	if !ok || bRat.Val == nil || !bRat.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_b_parameter_error", fmt.Errorf("integer required")))
	}

	return NewFiniteCurve(aRat.Val.Num(), bRat.Val.Num(), pRat.Val.Num())
}

func parseFinitePoint(node Node, p *big.Int, env *Env) (FinitePoint, error) {
	evalPt, err := EvalWithEnv(node, env)
	if err != nil {
		return FiniteInfinityPoint(), err
	}
	if sym, ok := evalPt.(*VarNode); ok && (sym.Name == "O" || sym.Name == "inf") {
		return FiniteInfinityPoint(), nil
	}
	listNode, ok := evalPt.(*ListNode)
	if !ok || len(listNode.Elements) != 2 {
		return FiniteInfinityPoint(), fmt.Errorf("%s", i18n.T("elliptic.err_point_format"))
	}

	xEval, err := EvalWithEnv(listNode.Elements[0], env)
	if err != nil {
		return FiniteInfinityPoint(), err
	}
	xRat, ok := xEval.(*RationalNode)
	if !ok || xRat.Val == nil || !xRat.Val.IsInt() {
		return FiniteInfinityPoint(), fmt.Errorf("%s", i18n.T("elliptic.err_point_format"))
	}

	yEval, err := EvalWithEnv(listNode.Elements[1], env)
	if err != nil {
		return FiniteInfinityPoint(), err
	}
	yRat, ok := yEval.(*RationalNode)
	if !ok || yRat.Val == nil || !yRat.Val.IsInt() {
		return FiniteInfinityPoint(), fmt.Errorf("%s", i18n.T("elliptic.err_point_format"))
	}

	return NewFinitePoint(xRat.Val.Num(), yRat.Val.Num(), p), nil
}

func finitePointToNode(pt FinitePoint) Node {
	if pt.IsInf {
		return NewList([]Node{
			NewRationalFromBigRat(new(big.Rat).SetInt64(0)),
			NewRationalFromBigRat(new(big.Rat).SetInt64(0)),
		})
	}
	return NewList([]Node{
		NewRationalFromBigRat(new(big.Rat).SetInt(pt.X)),
		NewRationalFromBigRat(new(big.Rat).SetInt(pt.Y)),
	})
}

// EvalEcPAddHandler: ec_p_add([A, B], P, Q, p)
func EvalEcPAddHandler(args []Node, env *Env) (Node, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_p_invalid_args"))
	}
	curve, err := parseFiniteCurve(args[0], args[3], env)
	if err != nil {
		return nil, err
	}
	p1, err := parseFinitePoint(args[1], curve.P, env)
	if err != nil {
		return nil, err
	}
	p2, err := parseFinitePoint(args[2], curve.P, env)
	if err != nil {
		return nil, err
	}

	sumPt := EcPAdd(curve, p1, p2)
	return finitePointToNode(sumPt), nil
}

// EvalEcPMulHandler: ec_p_mul([A, B], k, P, p)
func EvalEcPMulHandler(args []Node, env *Env) (Node, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_p_invalid_args"))
	}
	curve, err := parseFiniteCurve(args[0], args[3], env)
	if err != nil {
		return nil, err
	}

	kEval, err := EvalWithEnv(args[1], env)
	if err != nil {
		return nil, err
	}
	kRat, ok := kEval.(*RationalNode)
	if !ok || kRat.Val == nil || !kRat.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_scalar_int_required"))
	}

	pt, err := parseFinitePoint(args[2], curve.P, env)
	if err != nil {
		return nil, err
	}

	prodPt := EcPMul(curve, kRat.Val.Num(), pt)
	return finitePointToNode(prodPt), nil
}

// EvalEcPOrderHandler: ec_p_order([A, B], p)
func EvalEcPOrderHandler(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_p_invalid_args"))
	}
	curve, err := parseFiniteCurve(args[0], args[1], env)
	if err != nil {
		return nil, err
	}

	order, err := EcPOrder(curve)
	if err != nil {
		return nil, err
	}
	return NewRationalFromBigRat(new(big.Rat).SetInt(order)), nil
}

// EvalEcPTraceHandler: ec_p_trace([A, B], p)
func EvalEcPTraceHandler(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_p_invalid_args"))
	}
	curve, err := parseFiniteCurve(args[0], args[1], env)
	if err != nil {
		return nil, err
	}

	trace, err := EcPTrace(curve)
	if err != nil {
		return nil, err
	}
	return NewRationalFromBigRat(new(big.Rat).SetInt(trace)), nil
}
