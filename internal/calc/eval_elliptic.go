package calc

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Elliptic Point Structure & Representations
// -------------------------------------------------------------------------

// EllipticPoint represents an affine rational point (X, Y) on an elliptic curve,
// or the point at infinity O (IsInf == true).
type EllipticPoint struct {
	X     *big.Rat
	Y     *big.Rat
	IsInf bool
}

// NewEllipticPoint creates a finite affine point (x, y).
func NewEllipticPoint(x, y *big.Rat) EllipticPoint {
	return EllipticPoint{
		X:     new(big.Rat).Set(x),
		Y:     new(big.Rat).Set(y),
		IsInf: false,
	}
}

// InfinityPoint creates the group identity (point at infinity O).
func InfinityPoint() EllipticPoint {
	return EllipticPoint{
		X:     nil,
		Y:     nil,
		IsInf: true,
	}
}

// Equal returns true if two elliptic points are mathematically identical.
func (p EllipticPoint) Equal(other EllipticPoint) bool {
	if p.IsInf && other.IsInf {
		return true
	}
	if p.IsInf != other.IsInf {
		return false
	}
	return p.X.Cmp(other.X) == 0 && p.Y.Cmp(other.Y) == 0
}

// String returns a human-readable representation of the point.
func (p EllipticPoint) String() string {
	if p.IsInf {
		return "O"
	}
	return fmt.Sprintf("(%s, %s)", p.X.RatString(), p.Y.RatString())
}

// pointToNodeSafely converts EllipticPoint into Node using exact big.Rat rational nodes.
func pointToNodeSafely(p EllipticPoint) Node {
	if p.IsInf {
		return &VarNode{Name: "O"}
	}
	xNode := &RationalNode{Val: new(big.Rat).Set(p.X)}
	yNode := &RationalNode{Val: new(big.Rat).Set(p.Y)}
	return &ListNode{
		Elements: []Node{xNode, yNode},
	}
}

// IsOnCurve checks whether point p satisfies Weierstrass equation y^2 = x^3 + Ax + B.
func IsOnCurve(p EllipticPoint, a, b *big.Rat) bool {
	if p.IsInf {
		return true
	}
	// y^2
	y2 := new(big.Rat).Mul(p.Y, p.Y)

	// x^3 + a*x + b
	x2 := new(big.Rat).Mul(p.X, p.X)
	x3 := new(big.Rat).Mul(x2, p.X)
	ax := new(big.Rat).Mul(a, p.X)

	rhs := new(big.Rat).Add(x3, ax)
	rhs.Add(rhs, b)

	return y2.Cmp(rhs) == 0
}

// -------------------------------------------------------------------------
// Elliptic Group Operations (Chord and Tangent & Double-and-Add)
// -------------------------------------------------------------------------

// AddEllipticPoints computes P1 + P2 on elliptic curve y^2 = x^3 + Ax + B over Q.
func AddEllipticPoints(p1, p2 EllipticPoint, a, b *big.Rat) (EllipticPoint, error) {
	// Identity laws
	if p1.IsInf {
		return p2, nil
	}
	if p2.IsInf {
		return p1, nil
	}

	// Distinct X coordinates: secant line
	if p1.X.Cmp(p2.X) != 0 {
		// lambda = (y2 - y1) / (x2 - x1)
		num := new(big.Rat).Sub(p2.Y, p1.Y)
		den := new(big.Rat).Sub(p2.X, p1.X)
		lambda := new(big.Rat).Quo(num, den)

		// x3 = lambda^2 - x1 - x2
		lambda2 := new(big.Rat).Mul(lambda, lambda)
		x3 := new(big.Rat).Sub(lambda2, p1.X)
		x3.Sub(x3, p2.X)

		// y3 = lambda*(x1 - x3) - y1
		dx := new(big.Rat).Sub(p1.X, x3)
		y3 := new(big.Rat).Mul(lambda, dx)
		y3.Sub(y3, p1.Y)

		return NewEllipticPoint(x3, y3), nil
	}

	// Same X coordinates
	// If y1 != y2, then y1 = -y2 => P1 + P2 = O
	if p1.Y.Cmp(p2.Y) != 0 {
		return InfinityPoint(), nil
	}

	// P1 == P2: tangent line (doubling)
	// If y1 == 0, vertical tangent => 2*P = O
	if p1.Y.Sign() == 0 {
		return InfinityPoint(), nil
	}

	// lambda = (3*x1^2 + a) / (2*y1)
	three := big.NewRat(3, 1)
	two := big.NewRat(2, 1)

	x1Sq := new(big.Rat).Mul(p1.X, p1.X)
	num := new(big.Rat).Mul(three, x1Sq)
	num.Add(num, a)

	den := new(big.Rat).Mul(two, p1.Y)
	lambda := new(big.Rat).Quo(num, den)

	// x3 = lambda^2 - 2*x1
	lambda2 := new(big.Rat).Mul(lambda, lambda)
	twoX1 := new(big.Rat).Mul(two, p1.X)
	x3 := new(big.Rat).Sub(lambda2, twoX1)

	// y3 = lambda*(x1 - x3) - y1
	dx := new(big.Rat).Sub(p1.X, x3)
	y3 := new(big.Rat).Mul(lambda, dx)
	y3.Sub(y3, p1.Y)

	return NewEllipticPoint(x3, y3), nil
}

// ScalarMulEllipticPoint computes n * P using Double-and-Add algorithm over Q.
func ScalarMulEllipticPoint(p EllipticPoint, n *big.Int, a, b *big.Rat) (EllipticPoint, error) {
	if n == nil || n.Sign() == 0 || p.IsInf {
		return InfinityPoint(), nil
	}

	basePoint := p
	if n.Sign() < 0 {
		// Inverse point (x, -y)
		negY := new(big.Rat).Neg(p.Y)
		basePoint = NewEllipticPoint(p.X, negY)
	}

	k := new(big.Int).Abs(n)
	result := InfinityPoint()
	current := basePoint

	bitLen := k.BitLen()
	for i := 0; i < bitLen; i++ {
		if k.Bit(i) == 1 {
			var err error
			result, err = AddEllipticPoints(result, current, a, b)
			if err != nil {
				return InfinityPoint(), err
			}
		}
		var err error
		current, err = AddEllipticPoints(current, current, a, b)
		if err != nil {
			return InfinityPoint(), err
		}
	}

	return result, nil
}

// -------------------------------------------------------------------------
// Nagell-Lutz & Mazur Torsion Subgroup Determination
// -------------------------------------------------------------------------

// FindTorsionSubgroup determines all rational torsion points and isomorphism group structure
// for Weierstrass curve y^2 = x^3 + Ax + B over Q.
func FindTorsionSubgroup(aRat, bRat *big.Rat, fsm *EllipticLifecycleFSM) ([]EllipticPoint, string, error) {
	if fsm == nil {
		fsm = NewEllipticLifecycleFSM()
	}

	// 1. Discriminant validation & Non-singularity
	// D = 4*A^3 + 27*B^2; Delta = -16 * D
	four := big.NewRat(4, 1)
	twentySeven := big.NewRat(27, 1)

	a3 := new(big.Rat).Mul(aRat, aRat)
	a3.Mul(a3, aRat)
	fourA3 := new(big.Rat).Mul(four, a3)

	b2 := new(big.Rat).Mul(bRat, bRat)
	twentySevenB2 := new(big.Rat).Mul(twentySeven, b2)

	discrimRat := new(big.Rat).Add(fourA3, twentySevenB2)
	if discrimRat.Sign() == 0 {
		_ = fsm.TransitionTo(EcStateFailed)
		return nil, "", fmt.Errorf("%s", i18n.T("elliptic.err_singular_curve"))
	}

	if err := fsm.TransitionTo(EcStateNormalized); err != nil {
		return nil, "", err
	}

	// 2. Transform to integer coefficients via change of variables:
	// x = X / d^2, y = Y / d^3
	// Y^2 = X^3 + A' X + B'
	// where A' = A * d^4, B' = B * d^6.
	// d = lcm(denom(A), denom(B))
	denA := aRat.Denom()
	denB := bRat.Denom()
	d := computeLCM(denA, denB)

	d2 := new(big.Int).Mul(d, d)
	d3 := new(big.Int).Mul(d2, d)
	d4 := new(big.Int).Mul(d2, d2)
	d6 := new(big.Int).Mul(d3, d3)

	// A' = A * d^4
	aPrimeRat := new(big.Rat).Mul(aRat, new(big.Rat).SetInt(d4))
	if !aPrimeRat.IsInt() {
		_ = fsm.TransitionTo(EcStateFailed)
		return nil, "", fmt.Errorf("failed to integralize A: %s", aPrimeRat.RatString())
	}
	aPrime := aPrimeRat.Num()

	// B' = B * d^6
	bPrimeRat := new(big.Rat).Mul(bRat, new(big.Rat).SetInt(d6))
	if !bPrimeRat.IsInt() {
		_ = fsm.TransitionTo(EcStateFailed)
		return nil, "", fmt.Errorf("failed to integralize B: %s", bPrimeRat.RatString())
	}
	bPrime := bPrimeRat.Num()

	// Integer Discriminant D' = 4*(A')^3 + 27*(B')^2
	fourInt := big.NewInt(4)
	twentySevenInt := big.NewInt(27)

	aPrime3 := new(big.Int).Mul(aPrime, aPrime)
	aPrime3.Mul(aPrime3, aPrime)
	term1 := new(big.Int).Mul(fourInt, aPrime3)

	bPrime2 := new(big.Int).Mul(bPrime, bPrime)
	term2 := new(big.Int).Mul(twentySevenInt, bPrime2)

	dPrime := new(big.Int).Add(term1, term2)
	absDPrime := new(big.Int).Abs(dPrime)

	// Nagell-Lutz discriminant bound: y^2 | 16 * D'
	sixteenInt := big.NewInt(16)
	deltaBound := new(big.Int).Mul(sixteenInt, absDPrime)

	if err := fsm.TransitionTo(EcStateDiscrimFactored); err != nil {
		return nil, "", err
	}

	// 3. Find all square divisors Y > 0 such that Y^2 | deltaBound
	squareDivisors := listSquareDivisors(deltaBound)

	if err := fsm.TransitionTo(EcStateCandidatesFound); err != nil {
		return nil, "", err
	}

	// 4. For Y = 0 and each Y in squareDivisors, solve X^3 + A' X + (B' - Y^2) = 0 for integer X
	candidatePoints := []EllipticPoint{InfinityPoint()}

	// Case Y = 0 (2-torsion points)
	yZeroRoots := findCubicIntegerRoots(aPrime, bPrime)
	for _, xInt := range yZeroRoots {
		// x = X / d^2, y = 0
		xRat := new(big.Rat).SetFrac(xInt, d2)
		yRat := big.NewRat(0, 1)
		pt := NewEllipticPoint(xRat, yRat)
		if IsOnCurve(pt, aRat, bRat) {
			candidatePoints = append(candidatePoints, pt)
		}
	}

	// Case Y > 0
	for _, yInt := range squareDivisors {
		y2 := new(big.Int).Mul(yInt, yInt)
		cVal := new(big.Int).Sub(bPrime, y2)

		xRoots := findCubicIntegerRoots(aPrime, cVal)
		for _, xInt := range xRoots {
			xRat := new(big.Rat).SetFrac(xInt, d2)
			yRat := new(big.Rat).SetFrac(yInt, d3)

			pt1 := NewEllipticPoint(xRat, yRat)
			if IsOnCurve(pt1, aRat, bRat) {
				candidatePoints = append(candidatePoints, pt1)
				// By symmetry, (x, -y) is also on curve
				negYRat := new(big.Rat).Neg(yRat)
				pt2 := NewEllipticPoint(xRat, negYRat)
				candidatePoints = append(candidatePoints, pt2)
			}
		}
	}

	if err := fsm.TransitionTo(EcStateOrderVerified); err != nil {
		return nil, "", err
	}

	// 5. Remove duplicates and verify finite order using Mazur's bound (m <= 12)
	uniqueCandidates := deduplicatePoints(candidatePoints)
	var torsionPoints []EllipticPoint
	twoTorsionCount := 0

	for _, pt := range uniqueCandidates {
		if pt.IsInf {
			torsionPoints = append(torsionPoints, pt)
			continue
		}

		isFiniteTorsion := false
		curr := pt
		for k := 1; k <= 12; k++ {
			if curr.IsInf {
				isFiniteTorsion = true
				if k == 2 {
					twoTorsionCount++
				}
				break
			}
			var addErr error
			curr, addErr = AddEllipticPoints(curr, pt, aRat, bRat)
			if addErr != nil {
				isFiniteTorsion = false
				break
			}
		}
		if isFiniteTorsion {
			torsionPoints = append(torsionPoints, pt)
		}
	}

	if err := fsm.TransitionTo(EcStateGroupDetermined); err != nil {
		return nil, "", err
	}

	// 6. Deduplicate & Sort points for deterministic output
	torsionPoints = sortEllipticPoints(deduplicatePoints(torsionPoints))

	// 7. Determine group structure according to Mazur's Classification Theorem
	groupOrder := len(torsionPoints)
	var groupStructure string

	switch groupOrder {
	case 1:
		groupStructure = "Z/1Z"
	case 2:
		groupStructure = "Z/2Z"
	case 3:
		groupStructure = "Z/3Z"
	case 4:
		if twoTorsionCount == 3 {
			groupStructure = "Z/2Z x Z/2Z"
		} else {
			groupStructure = "Z/4Z"
		}
	case 5:
		groupStructure = "Z/5Z"
	case 6:
		groupStructure = "Z/6Z"
	case 7:
		groupStructure = "Z/7Z"
	case 8:
		if twoTorsionCount == 3 {
			groupStructure = "Z/2Z x Z/4Z"
		} else {
			groupStructure = "Z/8Z"
		}
	case 9:
		groupStructure = "Z/9Z"
	case 10:
		groupStructure = "Z/10Z"
	case 12:
		if twoTorsionCount == 3 {
			groupStructure = "Z/2Z x Z/6Z"
		} else {
			groupStructure = "Z/12Z"
		}
	case 16:
		groupStructure = "Z/2Z x Z/8Z"
	default:
		groupStructure = fmt.Sprintf("Order %d", groupOrder)
	}

	return torsionPoints, groupStructure, nil
}

// -------------------------------------------------------------------------
// Helper Algorithms: Integer Roots & Divisors
// -------------------------------------------------------------------------

func computeLCM(a, b *big.Int) *big.Int {
	g := new(big.Int).GCD(nil, nil, a, b)
	prod := new(big.Int).Mul(a, b)
	res := new(big.Int).Div(prod, g)
	return res.Abs(res)
}

// findCubicIntegerRoots solves f(X) = X^3 + A*X + C = 0 for integer roots X in Z.
// Uses piecewise integer bisection bounded by Cauchy's root bounds.
func findCubicIntegerRoots(a, c *big.Int) []*big.Int {
	var roots []*big.Int

	// Cauchy bound: |X| <= 1 + max(|A|, |C|)
	absA := new(big.Int).Abs(a)
	absC := new(big.Int).Abs(c)
	maxCoeff := absA
	if absC.Cmp(maxCoeff) > 0 {
		maxCoeff = absC
	}
	mBound := new(big.Int).Add(maxCoeff, big.NewInt(2))
	negMBound := new(big.Int).Neg(mBound)

	evalF := func(x *big.Int) *big.Int {
		// x^3 + a*x + c
		x2 := new(big.Int).Mul(x, x)
		x3 := new(big.Int).Mul(x2, x)
		ax := new(big.Int).Mul(a, x)
		res := new(big.Int).Add(x3, ax)
		res.Add(res, c)
		return res
	}

	// Monotonicity intervals:
	// f'(X) = 3*X^2 + A
	if a.Sign() >= 0 {
		// Strictly increasing everywhere on [-M, M]
		if root := bisectRoot(evalF, negMBound, mBound, true); root != nil {
			roots = append(roots, root)
		}
	} else {
		// A < 0: Extrema at X = +/- sqrt(-A/3)
		negA := new(big.Int).Neg(a)
		three := big.NewInt(3)
		negAOver3 := new(big.Int).Div(negA, three)
		r := new(big.Int).Sqrt(negAOver3)
		rPlus := new(big.Int).Add(r, big.NewInt(1))
		negR := new(big.Int).Neg(rPlus)

		// Interval 1: [-M, -r] (increasing)
		if root := bisectRoot(evalF, negMBound, negR, true); root != nil {
			roots = append(roots, root)
		}
		// Interval 2: [-r, r] (decreasing)
		if root := bisectRoot(evalF, negR, rPlus, false); root != nil {
			roots = append(roots, root)
		}
		// Interval 3: [r, M] (increasing)
		if root := bisectRoot(evalF, rPlus, mBound, true); root != nil {
			roots = append(roots, root)
		}

		// Also directly check critical neighborhood points:
		for _, pt := range []*big.Int{new(big.Int).Neg(r), r, big.NewInt(0)} {
			if evalF(pt).Sign() == 0 {
				roots = append(roots, pt)
			}
		}
	}

	return deduplicateBigInts(roots)
}

func bisectRoot(f func(*big.Int) *big.Int, low, high *big.Int, increasing bool) *big.Int {
	fLow := f(low)
	fHigh := f(high)

	if fLow.Sign() == 0 {
		return new(big.Int).Set(low)
	}
	if fHigh.Sign() == 0 {
		return new(big.Int).Set(high)
	}

	if increasing {
		if fLow.Sign() > 0 || fHigh.Sign() < 0 {
			return nil
		}
	} else {
		if fLow.Sign() < 0 || fHigh.Sign() > 0 {
			return nil
		}
	}

	l := new(big.Int).Set(low)
	r := new(big.Int).Set(high)
	one := big.NewInt(1)
	diff := new(big.Int)

	for diff.Sub(r, l).Cmp(one) > 0 {
		mid := new(big.Int).Add(l, r)
		mid.Div(mid, big.NewInt(2))

		fMid := f(mid)
		if fMid.Sign() == 0 {
			return mid
		}

		if increasing {
			if fMid.Sign() < 0 {
				l.Set(mid)
			} else {
				r.Set(mid)
			}
		} else {
			if fMid.Sign() > 0 {
				l.Set(mid)
			} else {
				r.Set(mid)
			}
		}
	}

	if f(l).Sign() == 0 {
		return l
	}
	if f(r).Sign() == 0 {
		return r
	}
	return nil
}

// listSquareDivisors factors delta and finds all positive integers Y such that Y^2 | delta.
func listSquareDivisors(delta *big.Int) []*big.Int {
	if delta.Sign() == 0 {
		return nil
	}

	factors := factorIntPrimes(delta)
	// We want Y = prod p_i^a_i where 0 <= a_i <= floor(e_i / 2)
	type factorHalf struct {
		prime *big.Int
		maxA  int
	}

	var halfs []factorHalf
	for _, f := range factors {
		maxA := f.exp / 2
		if maxA > 0 {
			halfs = append(halfs, factorHalf{prime: f.prime, maxA: maxA})
		}
	}

	var results []*big.Int
	var generateCombos func(idx int, current *big.Int)
	generateCombos = func(idx int, current *big.Int) {
		if idx == len(halfs) {
			results = append(results, new(big.Int).Set(current))
			return
		}
		pPow := big.NewInt(1)
		for a := 0; a <= halfs[idx].maxA; a++ {
			next := new(big.Int).Mul(current, pPow)
			generateCombos(idx+1, next)
			pPow.Mul(pPow, halfs[idx].prime)
		}
	}

	generateCombos(0, big.NewInt(1))
	return results
}

type primeExp struct {
	prime *big.Int
	exp   int
}

// factorIntPrimes factors n into prime powers using trial division and Miller-Rabin test.
func factorIntPrimes(n *big.Int) []primeExp {
	val := new(big.Int).Abs(n)
	if val.Cmp(big.NewInt(1)) <= 0 {
		return nil
	}

	var factors []primeExp
	two := big.NewInt(2)
	three := big.NewInt(3)
	rem := new(big.Int)

	// Factor 2
	c2 := 0
	for {
		rem.Mod(val, two)
		if rem.Sign() == 0 {
			c2++
			val.Div(val, two)
		} else {
			break
		}
	}
	if c2 > 0 {
		factors = append(factors, primeExp{prime: big.NewInt(2), exp: c2})
	}

	// Factor 3
	c3 := 0
	for {
		rem.Mod(val, three)
		if rem.Sign() == 0 {
			c3++
			val.Div(val, three)
		} else {
			break
		}
	}
	if c3 > 0 {
		factors = append(factors, primeExp{prime: big.NewInt(3), exp: c3})
	}

	// 6k +/- 1 wheel
	d := big.NewInt(5)
	step := big.NewInt(2)
	d2 := new(big.Int).Mul(d, d)

	for d2.Cmp(val) <= 0 {
		if IsDeterministicPrime(val) {
			break
		}
		count := 0
		for {
			rem.Mod(val, d)
			if rem.Sign() == 0 {
				count++
				val.Div(val, d)
			} else {
				break
			}
		}
		if count > 0 {
			factors = append(factors, primeExp{prime: new(big.Int).Set(d), exp: count})
		}
		d.Add(d, step)
		if step.Cmp(two) == 0 {
			step.SetInt64(4)
		} else {
			step.SetInt64(2)
		}
		d2.Mul(d, d)
	}

	if val.Cmp(big.NewInt(1)) > 0 {
		factors = append(factors, primeExp{prime: new(big.Int).Set(val), exp: 1})
	}

	return factors
}

func deduplicatePoints(pts []EllipticPoint) []EllipticPoint {
	var result []EllipticPoint
	for _, p := range pts {
		found := false
		for _, existing := range result {
			if p.Equal(existing) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, p)
		}
	}
	return result
}

func deduplicateBigInts(nums []*big.Int) []*big.Int {
	var result []*big.Int
	for _, n := range nums {
		found := false
		for _, existing := range result {
			if n.Cmp(existing) == 0 {
				found = true
				break
			}
		}
		if !found {
			result = append(result, n)
		}
	}
	return result
}

func sortEllipticPoints(pts []EllipticPoint) []EllipticPoint {
	sort.Slice(pts, func(i, j int) bool {
		// Infinity comes first
		if pts[i].IsInf {
			return true
		}
		if pts[j].IsInf {
			return false
		}
		cmpX := pts[i].X.Cmp(pts[j].X)
		if cmpX != 0 {
			return cmpX < 0
		}
		return pts[i].Y.Cmp(pts[j].Y) < 0
	})
	return pts
}

// -------------------------------------------------------------------------
// CLI / Evaluator Handlers & Registration
// -------------------------------------------------------------------------

func init() {
	RegisterHandler("ec_add", evalEcAdd)
	RegisterHandler("ec_mul", evalEcMul)
	RegisterHandler("ec_torsion", evalEcTorsion)
}

func evalEcAdd(args []Node, env *Env) (Node, error) {
	if len(args) < 4 || len(args) > 6 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_invalid_args"))
	}

	aRat, err := evalToRat(args[0], env)
	if err != nil {
		return nil, fmt.Errorf("A parameter error: %w", err)
	}
	bRat, err := evalToRat(args[1], env)
	if err != nil {
		return nil, fmt.Errorf("B parameter error: %w", err)
	}

	var p1, p2 EllipticPoint
	if len(args) == 4 {
		p1, err = parseEllipticPointNode(args[2], env)
		if err != nil {
			return nil, err
		}
		p2, err = parseEllipticPointNode(args[3], env)
		if err != nil {
			return nil, err
		}
	} else if len(args) == 6 {
		x1, err := evalToRat(args[2], env)
		if err != nil {
			return nil, err
		}
		y1, err := evalToRat(args[3], env)
		if err != nil {
			return nil, err
		}
		x2, err := evalToRat(args[4], env)
		if err != nil {
			return nil, err
		}
		y2, err := evalToRat(args[5], env)
		if err != nil {
			return nil, err
		}
		p1 = NewEllipticPoint(x1, y1)
		p2 = NewEllipticPoint(x2, y2)
	} else {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_invalid_args"))
	}

	sum, err := AddEllipticPoints(p1, p2, aRat, bRat)
	if err != nil {
		return nil, err
	}

	return pointToNodeSafely(sum), nil
}

func evalEcMul(args []Node, env *Env) (Node, error) {
	if len(args) < 4 || len(args) > 5 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_invalid_args"))
	}

	aRat, err := evalToRat(args[0], env)
	if err != nil {
		return nil, fmt.Errorf("A parameter error: %w", err)
	}
	bRat, err := evalToRat(args[1], env)
	if err != nil {
		return nil, fmt.Errorf("B parameter error: %w", err)
	}

	nRat, err := evalToRat(args[2], env)
	if err != nil {
		return nil, fmt.Errorf("scalar error: %w", err)
	}
	if !nRat.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_scalar_int_required"))
	}
	nInt := nRat.Num()

	var p EllipticPoint
	if len(args) == 4 {
		p, err = parseEllipticPointNode(args[3], env)
		if err != nil {
			return nil, err
		}
	} else {
		x, err := evalToRat(args[3], env)
		if err != nil {
			return nil, err
		}
		y, err := evalToRat(args[4], env)
		if err != nil {
			return nil, err
		}
		p = NewEllipticPoint(x, y)
	}

	prod, err := ScalarMulEllipticPoint(p, nInt, aRat, bRat)
	if err != nil {
		return nil, err
	}

	return pointToNodeSafely(prod), nil
}

func evalEcTorsion(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_invalid_args"))
	}

	aRat, err := evalToRat(args[0], env)
	if err != nil {
		return nil, fmt.Errorf("A parameter error: %w", err)
	}
	bRat, err := evalToRat(args[1], env)
	if err != nil {
		return nil, fmt.Errorf("B parameter error: %w", err)
	}

	fsm := NewEllipticLifecycleFSM()
	points, groupStr, err := FindTorsionSubgroup(aRat, bRat, fsm)
	if err != nil {
		return nil, err
	}

	ptNodes := make([]Node, len(points))
	for i, pt := range points {
		ptNodes[i] = pointToNodeSafely(pt)
	}

	return &ListNode{
		Elements: []Node{
			&VarNode{Name: groupStr},
			&ListNode{Elements: ptNodes},
		},
	}, nil
}

func evalToRat(n Node, env *Env) (*big.Rat, error) {
	ev, err := EvalWithEnv(n, env)
	if err != nil {
		ev = n
	}
	if r, ok := ev.(*RationalNode); ok {
		return new(big.Rat).Set(r.Val), nil
	}
	return nil, fmt.Errorf("expected rational number, got: %s", n.String())
}

func parseEllipticPointNode(n Node, env *Env) (EllipticPoint, error) {
	ev, err := EvalWithEnv(n, env)
	if err != nil {
		ev = n
	}

	if v, ok := ev.(*VarNode); ok {
		if v.Name == "O" || v.Name == "inf" || v.Name == "Inf" || v.Name == "infinity" {
			return InfinityPoint(), nil
		}
	}

	if list, ok := ev.(*ListNode); ok {
		if len(list.Elements) == 0 {
			return InfinityPoint(), nil
		}
		if len(list.Elements) == 2 {
			x, err := evalToRat(list.Elements[0], env)
			if err != nil {
				return EllipticPoint{}, err
			}
			y, err := evalToRat(list.Elements[1], env)
			if err != nil {
				return EllipticPoint{}, err
			}
			return NewEllipticPoint(x, y), nil
		}
		return EllipticPoint{}, fmt.Errorf("%s", i18n.T("elliptic.err_point_format"))
	}

	return EllipticPoint{}, fmt.Errorf("%s: %s", i18n.T("elliptic.err_point_format"), n.String())
}
