package calc

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Elliptic Descent Lifecycle FSM
// -------------------------------------------------------------------------

// EcDescentState represents execution phases of the 2-descent pipeline.
type EcDescentState int

const (
	StateDescentInit EcDescentState = iota
	StateDescentNormalized
	StateDescentTwoTorsionFactored
	StateDescentIsogenyConstructed
	StateDescentDivisorsEnumerated
	StateDescentLocallyFiltered
	StateDescentSelmerSolved
	StateDescentPointsSearched
	StateDescentCompleted
	StateDescentFailed
)

func (s EcDescentState) String() string {
	switch s {
	case StateDescentInit:
		return "Init"
	case StateDescentNormalized:
		return "Normalized"
	case StateDescentTwoTorsionFactored:
		return "TwoTorsionFactored"
	case StateDescentIsogenyConstructed:
		return "IsogenyConstructed"
	case StateDescentDivisorsEnumerated:
		return "DivisorsEnumerated"
	case StateDescentLocallyFiltered:
		return "LocallyFiltered"
	case StateDescentSelmerSolved:
		return "SelmerSolved"
	case StateDescentPointsSearched:
		return "PointsSearched"
	case StateDescentCompleted:
		return "Completed"
	case StateDescentFailed:
		return "Failed"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// EllipticDescentLifecycleFSM manages transitions across 2-descent execution phases.
type EllipticDescentLifecycleFSM struct {
	currentState EcDescentState
	history      []EcDescentState
}

// NewEllipticDescentLifecycleFSM creates a new FSM initialized to StateDescentInit.
func NewEllipticDescentLifecycleFSM() *EllipticDescentLifecycleFSM {
	return &EllipticDescentLifecycleFSM{
		currentState: StateDescentInit,
		history:      []EcDescentState{StateDescentInit},
	}
}

// CurrentState returns the current state of the descent FSM.
func (f *EllipticDescentLifecycleFSM) CurrentState() EcDescentState {
	return f.currentState
}

// TransitionTo validates and performs a state transition.
func (f *EllipticDescentLifecycleFSM) TransitionTo(next EcDescentState) error {
	if next == StateDescentFailed {
		f.currentState = next
		f.history = append(f.history, next)
		return nil
	}

	valid := false
	switch f.currentState {
	case StateDescentInit:
		valid = (next == StateDescentNormalized)
	case StateDescentNormalized:
		valid = (next == StateDescentTwoTorsionFactored)
	case StateDescentTwoTorsionFactored:
		valid = (next == StateDescentIsogenyConstructed)
	case StateDescentIsogenyConstructed:
		valid = (next == StateDescentDivisorsEnumerated)
	case StateDescentDivisorsEnumerated:
		valid = (next == StateDescentLocallyFiltered)
	case StateDescentLocallyFiltered:
		valid = (next == StateDescentSelmerSolved)
	case StateDescentSelmerSolved:
		valid = (next == StateDescentPointsSearched)
	case StateDescentPointsSearched:
		valid = (next == StateDescentCompleted)
	case StateDescentCompleted, StateDescentFailed:
		valid = false
	}

	if !valid {
		return fmt.Errorf("%s: %s -> %s", i18n.T("elliptic.err_invalid_elliptic_fsm_transition", f.currentState.String(), next.String()), f.currentState.String(), next.String())
	}

	f.currentState = next
	f.history = append(f.history, next)
	return nil
}

// -------------------------------------------------------------------------
// 2-Division Polynomial Factoring & Rational Roots
// -------------------------------------------------------------------------

// FindTwoTorsionRationalRoots finds all rational roots of x^3 + Ax + B = 0 over Q.
// Rejects singular curves where 4A^3 + 27B^2 == 0.
func FindTwoTorsionRationalRoots(aRat, bRat *big.Rat) ([]*big.Rat, error) {
	// 1. Discriminant validation
	four := big.NewRat(4, 1)
	twentySeven := big.NewRat(27, 1)

	a3 := new(big.Rat).Mul(aRat, aRat)
	a3.Mul(a3, aRat)
	fourA3 := new(big.Rat).Mul(four, a3)

	b2 := new(big.Rat).Mul(bRat, bRat)
	twentySevenB2 := new(big.Rat).Mul(twentySeven, b2)

	discrimRat := new(big.Rat).Add(fourA3, twentySevenB2)
	if discrimRat.Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_singular_curve"))
	}

	// 2. Transform to integer monic cubic X^3 + A' X + B' = 0
	// x = X / d^2, where d = lcm(denom(A), denom(B))
	// A' = A * d^4, B' = B * d^6
	denA := aRat.Denom()
	denB := bRat.Denom()
	d := computeLCM(denA, denB)

	d2 := new(big.Int).Mul(d, d)
	d4 := new(big.Int).Mul(d2, d2)
	d6 := new(big.Int).Mul(d4, d2)

	aPrimeRat := new(big.Rat).Mul(aRat, new(big.Rat).SetInt(d4))
	bPrimeRat := new(big.Rat).Mul(bRat, new(big.Rat).SetInt(d6))

	if !aPrimeRat.IsInt() || !bPrimeRat.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_failed_to_integralize_a", aPrimeRat.RatString()))
	}

	aPrime := aPrimeRat.Num()
	bPrime := bPrimeRat.Num()

	// 3. Find integer roots of X^3 + A' X + B' = 0
	intRoots := findCubicIntegerRoots(aPrime, bPrime)

	var rationalRoots []*big.Rat
	d2Rat := new(big.Rat).SetInt(d2)

	for _, intR := range intRoots {
		rRat := new(big.Rat).Quo(new(big.Rat).SetInt(intR), d2Rat)

		// Exact verification on x^3 + Ax + B == 0
		r2 := new(big.Rat).Mul(rRat, rRat)
		r3 := new(big.Rat).Mul(r2, rRat)
		ar := new(big.Rat).Mul(aRat, rRat)
		val := new(big.Rat).Add(r3, ar)
		val.Add(val, bRat)

		if val.Sign() == 0 {
			rationalRoots = append(rationalRoots, rRat)
		}
	}

	// Sort roots ascending
	sort.Slice(rationalRoots, func(i, j int) bool {
		return rationalRoots[i].Cmp(rationalRoots[j]) < 0
	})

	return rationalRoots, nil
}

// -------------------------------------------------------------------------
// Dual 2-Isogenous Curve Construction
// -------------------------------------------------------------------------

// BuildDualIsogenousPair shifts rational root x0 to (0,0) and constructs
// E: y^2 = x^3 + c*x^2 + d*x and its dual E': Y^2 = X^3 + cPrime*X^2 + dPrime*X
// with integer coefficients c, d, cPrime, dPrime.
func BuildDualIsogenousPair(aRat, rootRat *big.Rat) (c, d, cPrime, dPrime, scaleD *big.Int, err error) {
	// x -> x + rootRat
	// y^2 = (x + r)^3 + A*(x + r) + B
	//     = x^3 + 3*r*x^2 + (3*r^2 + A)*x + (r^3 + A*r + B)
	// Constant term is 0.
	threeRat := big.NewRat(3, 1)
	cRat := new(big.Rat).Mul(threeRat, rootRat) // c_Q = 3*r

	r2 := new(big.Rat).Mul(rootRat, rootRat)
	threeR2 := new(big.Rat).Mul(threeRat, r2)
	dRat := new(big.Rat).Add(threeR2, aRat) // d_Q = 3*r^2 + A

	// Integralize: x = x' / u^2, y = y' / u^3
	// c_int = c_Q * u^2
	// d_int = d_Q * u^4
	// u = lcm(denom(c_Q), denom(d_Q))
	u := computeLCM(cRat.Denom(), dRat.Denom())
	u2 := new(big.Int).Mul(u, u)
	u4 := new(big.Int).Mul(u2, u2)

	cIntRat := new(big.Rat).Mul(cRat, new(big.Rat).SetInt(u2))
	dIntRat := new(big.Rat).Mul(dRat, new(big.Rat).SetInt(u4))

	if !cIntRat.IsInt() || !dIntRat.IsInt() {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to integralize shifted coefficients")
	}

	c = cIntRat.Num()
	d = dIntRat.Num()

	if d.Sign() == 0 {
		return nil, nil, nil, nil, nil, fmt.Errorf("%s", i18n.T("elliptic.err_singular_shifted_curve"))
	}

	// Dual curve E': Y^2 = X^3 - 2c X^2 + (c^2 - 4d) X
	twoInt := big.NewInt(2)
	cPrime = new(big.Int).Mul(big.NewInt(-1), new(big.Int).Mul(twoInt, c)) // -2c

	c2 := new(big.Int).Mul(c, c)
	fourD := new(big.Int).Mul(big.NewInt(4), d)
	dPrime = new(big.Int).Sub(c2, fourD) // c^2 - 4d

	if dPrime.Sign() == 0 {
		return nil, nil, nil, nil, nil, fmt.Errorf("%s", i18n.T("elliptic.err_singular_dual_curve"))
	}

	deltaE, deltaEPrime := ComputeTwoDescentDiscriminants(d, dPrime)
	if deltaE.Sign() == 0 || deltaEPrime.Sign() == 0 {
		return nil, nil, nil, nil, nil, fmt.Errorf("%s", i18n.T("elliptic.err_singular_dual_curve"))
	}

	return c, d, cPrime, dPrime, u, nil
}

// ComputeTwoDescentDiscriminants calculates the exact integer discriminants
// Delta(E) = 16 * d^2 * d' and Delta(E') = 256 * dPrime^2 * d
// for E: y^2 = x^3 + cx^2 + dx and dual E': Y^2 = X^3 - 2cX^2 + d'X with d' = c^2 - 4d.
func ComputeTwoDescentDiscriminants(d, dPrime *big.Int) (deltaE, deltaEPrime *big.Int) {
	// Delta(E) = 16 * d^2 * d'
	sixteen := big.NewInt(16)
	d2 := new(big.Int).Mul(d, d)
	deltaE = new(big.Int).Mul(sixteen, new(big.Int).Mul(d2, dPrime))

	// Delta(E') = 256 * dPrime^2 * d
	twoFiftySix := big.NewInt(256)
	dPrime2 := new(big.Int).Mul(dPrime, dPrime)
	deltaEPrime = new(big.Int).Mul(twoFiftySix, new(big.Int).Mul(dPrime2, d))

	return deltaE, deltaEPrime
}

// -------------------------------------------------------------------------
// Local Solvability Engines (R, Q_2, Q_p)
// -------------------------------------------------------------------------

// evalQuartic computes F(M, e) = d1*M^4 + c*M^2*e^2 + d2*e^4.
func evalQuartic(d1, c, d2, m, e *big.Int) *big.Int {
	m2 := new(big.Int).Mul(m, m)
	m4 := new(big.Int).Mul(m2, m2)
	term1 := new(big.Int).Mul(d1, m4)

	e2 := new(big.Int).Mul(e, e)
	e4 := new(big.Int).Mul(e2, e2)
	term3 := new(big.Int).Mul(d2, e4)

	m2e2 := new(big.Int).Mul(m2, e2)
	term2 := new(big.Int).Mul(c, m2e2)

	res := new(big.Int).Add(term1, term2)
	res.Add(res, term3)
	return res
}

// IsQuarticRealSoluble tests if N^2 = d1 M^4 + c M^2 e^2 + d2 e^4 is soluble over R.
func IsQuarticRealSoluble(d1, c, d2 *big.Int) bool {
	// If d1 > 0, as M -> infty, F(M, 1) -> +infty > 0.
	if d1.Sign() > 0 {
		return true
	}
	// If d2 > 0, at (M=0, e=1), F(0, 1) = d2 > 0.
	if d2.Sign() > 0 {
		return true
	}
	// If d1 <= 0 and d2 <= 0:
	if d1.Sign() == 0 && d2.Sign() == 0 {
		return false
	}
	// Parabola q(z) = d1 z^2 + c z + d2 for z = (M/e)^2 >= 0.
	// Since d1 < 0 and d2 < 0, q(z) >= 0 for some z >= 0 iff
	// c > 0 and discriminant Delta = c^2 - 4*d1*d2 >= 0.
	if c.Sign() <= 0 {
		return false
	}
	c2 := new(big.Int).Mul(c, c)
	fourD1D2 := new(big.Int).Mul(big.NewInt(4), new(big.Int).Mul(d1, d2))
	delta := new(big.Int).Sub(c2, fourD1D2)
	return delta.Sign() >= 0
}

// IsQuarticTwoAdicSoluble tests whether N^2 = d1 M^4 + c M^2 e^2 + d2 e^4 has a solution in Q_2.
// Uses projective pair enumeration over Z/64Z (direct 2-adic square criterion).
func IsQuarticTwoAdicSoluble(d1, c, d2 *big.Int) bool {
	eval := func(mInt, eInt int64) bool {
		m := big.NewInt(mInt)
		e := big.NewInt(eInt)
		f := evalQuartic(d1, c, d2, m, e)
		if f.Sign() == 0 {
			return true
		}
		// Direct 2-adic square criterion:
		// x in Q_2^* is a square iff v_2(x) is even, and unit part x / 2^{v_2(x)} = 1 (mod 8).
		val := new(big.Int).Set(f)
		v2 := 0
		rem := new(big.Int)
		two := big.NewInt(2)
		for {
			rem.Mod(val, two)
			if rem.Sign() == 0 {
				v2++
				val.Div(val, two)
			} else {
				break
			}
		}
		if v2%2 != 0 {
			return false
		}
		mod8 := new(big.Int).Mod(val, big.NewInt(8))
		if mod8.Sign() < 0 {
			mod8.Add(mod8, big.NewInt(8))
		}
		return mod8.Cmp(big.NewInt(1)) == 0
	}

	// Chart 1: e = 1, M in 0..63
	for m := int64(0); m < 64; m++ {
		if eval(m, 1) {
			return true
		}
	}
	// Chart 2: M = 1, e even in 0..62
	for e := int64(0); e < 64; e += 2 {
		if eval(1, e) {
			return true
		}
	}
	return false
}

// legendreSymbol computes the Legendre symbol (a / p) for odd prime p.
// Returns 1 if a is quadratic residue mod p, -1 if non-residue, 0 if p | a.
func legendreSymbol(a, p *big.Int) int {
	rem := new(big.Int).Mod(a, p)
	if rem.Sign() < 0 {
		rem.Add(rem, p)
	}
	if rem.Sign() == 0 {
		return 0
	}

	// Euler's criterion: a^{(p-1)/2} mod p
	exp := new(big.Int).Sub(p, big.NewInt(1))
	exp.Div(exp, big.NewInt(2))
	modExp := new(big.Int).Exp(rem, exp, p)

	if modExp.Cmp(big.NewInt(1)) == 0 {
		return 1
	}
	neg1 := new(big.Int).Sub(p, big.NewInt(1))
	if modExp.Cmp(neg1) == 0 {
		return -1
	}
	return 0
}

// IsQuarticOddPrimeSoluble tests whether N^2 = d1 M^4 + c M^2 e^2 + d2 e^4 has a solution in Q_p for odd prime p.
func IsQuarticOddPrimeSoluble(d1, c, d2 *big.Int, p *big.Int) bool {
	// If p is 2, delegate to 2-adic solver
	if p.Cmp(big.NewInt(2)) == 0 {
		return IsQuarticTwoAdicSoluble(d1, c, d2)
	}

	testPair := func(m, e *big.Int) bool {
		f := evalQuartic(d1, c, d2, m, e)
		if f.Sign() == 0 {
			// Check smooth root
			// dF/dM = 4*d1*M^3 + 2*c*M*e^2
			// dF/de = 2*c*M^2*e + 4*d2*e^3
			four := big.NewInt(4)
			two := big.NewInt(2)
			m2 := new(big.Int).Mul(m, m)
			e2 := new(big.Int).Mul(e, e)
			m3 := new(big.Int).Mul(m2, m)
			e3 := new(big.Int).Mul(e2, e)

			dfDm := new(big.Int).Add(new(big.Int).Mul(four, new(big.Int).Mul(d1, m3)), new(big.Int).Mul(two, new(big.Int).Mul(c, new(big.Int).Mul(m, e2))))
			dfDe := new(big.Int).Add(new(big.Int).Mul(two, new(big.Int).Mul(c, new(big.Int).Mul(m2, e))), new(big.Int).Mul(four, new(big.Int).Mul(d2, e3)))

			if new(big.Int).Mod(dfDm, p).Sign() != 0 || new(big.Int).Mod(dfDe, p).Sign() != 0 {
				return true // smooth root lifts by Hensel's lemma!
			}
			return false
		}

		// Extract p-adic valuation
		val := new(big.Int).Set(f)
		vp := 0
		rem := new(big.Int)
		for {
			rem.Mod(val, p)
			if rem.Sign() == 0 {
				vp++
				val.Div(val, p)
			} else {
				break
			}
		}

		if vp%2 != 0 {
			return false
		}

		// Unit part must be a quadratic residue mod p
		return legendreSymbol(val, p) == 1
	}

	// For p <= 200, test projective pairs mod p
	pInt := p.Int64()
	if pInt <= 200 {
		// Chart 1: e = 1, M in 0..p-1
		for m := int64(0); m < pInt; m++ {
			if testPair(big.NewInt(m), big.NewInt(1)) {
				return true
			}
		}
		// Chart 2: M = 1, e = 0 mod p
		if testPair(big.NewInt(1), big.NewInt(0)) {
			return true
		}

		// If singular mod p, lift to mod p^2
		p2 := pInt * pInt
		if p2 <= 40000 {
			for m := int64(0); m < p2; m++ {
				if testPair(big.NewInt(m), big.NewInt(1)) {
					return true
				}
			}
			for e := int64(0); e < p2; e += pInt {
				if testPair(big.NewInt(1), big.NewInt(e)) {
					return true
				}
			}
		}
		return false
	}

	// For large p, test e=1 with M=0..10 and reciprocal e=0
	for m := int64(0); m <= 50; m++ {
		if testPair(big.NewInt(m), big.NewInt(1)) {
			return true
		}
	}
	if testPair(big.NewInt(1), big.NewInt(0)) {
		return true
	}

	return false
}

// -------------------------------------------------------------------------
// Selmer Group Dimension & Rank Upper Bound
// -------------------------------------------------------------------------

// TwoDescentResult holds Mordell-Weil rank bounds and generators.
type TwoDescentResult struct {
	RankLowerBound int
	RankUpperBound int
	IsExact        bool
	SelmerPhiDim   int
	SelmerPsiDim   int
	Generators     []EllipticPoint
}

// generateSquarefreeDivisors generates all signed squarefree divisor classes mod squares.
func generateSquarefreeDivisors(n *big.Int) []*big.Int {
	absN := new(big.Int).Abs(n)
	factors := factorIntPrimes(absN)
	var primes []*big.Int
	seen := make(map[string]bool)
	for _, f := range factors {
		pStr := f.prime.String()
		if !seen[pStr] {
			seen[pStr] = true
			primes = append(primes, new(big.Int).Set(f.prime))
		}
	}

	var divisors []*big.Int
	k := len(primes)
	total := 1 << k
	for mask := 0; mask < total; mask++ {
		prod := big.NewInt(1)
		for i := 0; i < k; i++ {
			if (mask & (1 << i)) != 0 {
				prod.Mul(prod, primes[i])
			}
		}
		divisors = append(divisors, prod)
		negProd := new(big.Int).Neg(prod)
		divisors = append(divisors, negProd)
	}
	return deduplicateBigInts(divisors)
}

// computeF2Dimension computes the dimension of the subgroup generated by elements in Q*/(Q*)^2
// via Gaussian elimination over F_2.
func computeF2Dimension(elements []*big.Int, primeBasis []*big.Int) int {
	if len(elements) == 0 {
		return 0
	}

	// Matrix columns: column 0 is -1 sign bit, columns 1..k are prime factors
	cols := 1 + len(primeBasis)
	var matrix [][]int

	for _, el := range elements {
		if el.Sign() == 0 {
			continue
		}
		row := make([]int, cols)
		val := new(big.Int).Set(el)
		if val.Sign() < 0 {
			row[0] = 1
			val.Abs(val)
		}
		for i, p := range primeBasis {
			vp := 0
			rem := new(big.Int)
			for {
				rem.Mod(val, p)
				if rem.Sign() == 0 {
					vp++
					val.Div(val, p)
				} else {
					break
				}
			}
			row[1+i] = vp % 2
		}
		// If all zeros (element is 1 mod squares), skip
		isZero := true
		for _, b := range row {
			if b != 0 {
				isZero = false
				break
			}
		}
		if !isZero {
			matrix = append(matrix, row)
		}
	}

	if len(matrix) == 0 {
		return 0
	}

	// Gaussian elimination over F_2
	rows := len(matrix)
	pivotRow := 0
	for col := 0; col < cols && pivotRow < rows; col++ {
		// Find pivot
		sel := -1
		for r := pivotRow; r < rows; r++ {
			if matrix[r][col] == 1 {
				sel = r
				break
			}
		}
		if sel == -1 {
			continue
		}
		// Swap rows
		matrix[pivotRow], matrix[sel] = matrix[sel], matrix[pivotRow]

		// Eliminate
		for r := 0; r < rows; r++ {
			if r != pivotRow && matrix[r][col] == 1 {
				for c := col; c < cols; c++ {
					matrix[r][c] ^= matrix[pivotRow][c]
				}
			}
		}
		pivotRow++
	}

	return pivotRow
}

// ComputeTwoDescentRankBound performs 2-isogeny descent to compute the Selmer rank bounds.
func ComputeTwoDescentRankBound(aRat, bRat *big.Rat, fsm *EllipticDescentLifecycleFSM) (*TwoDescentResult, error) {
	if fsm == nil {
		fsm = NewEllipticDescentLifecycleFSM()
	}

	if err := fsm.TransitionTo(StateDescentNormalized); err != nil {
		return nil, err
	}

	// 1. Find rational roots of x^3 + Ax + B
	roots, err := FindTwoTorsionRationalRoots(aRat, bRat)
	if err != nil {
		_ = fsm.TransitionTo(StateDescentFailed)
		return nil, err
	}
	if len(roots) == 0 {
		_ = fsm.TransitionTo(StateDescentFailed)
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_no_rational_2_torsion"))
	}

	if err := fsm.TransitionTo(StateDescentTwoTorsionFactored); err != nil {
		return nil, err
	}

	// 2. Select rational 2-torsion root (prefer 0 if present)
	chosenRoot := roots[0]
	for _, r := range roots {
		if r.Sign() == 0 {
			chosenRoot = r
			break
		}
	}

	if err := fsm.TransitionTo(StateDescentIsogenyConstructed); err != nil {
		return nil, err
	}

	// 3. Build dual isogenous curve pair
	c, d, cPrime, dPrime, scaleD, err := BuildDualIsogenousPair(aRat, chosenRoot)
	if err != nil {
		_ = fsm.TransitionTo(StateDescentFailed)
		return nil, err
	}

	if err := fsm.TransitionTo(StateDescentDivisorsEnumerated); err != nil {
		return nil, err
	}

	// 4. Bad prime set S = {2} union {p prime : p | 2*d*d'}
	prodDDPrime := new(big.Int).Mul(d, dPrime)
	prodDDPrime.Mul(prodDDPrime, big.NewInt(2))
	factors := factorIntPrimes(new(big.Int).Abs(prodDDPrime))

	var badPrimes []*big.Int
	seenPrimes := make(map[string]bool)
	seenPrimes["2"] = true
	badPrimes = append(badPrimes, big.NewInt(2))

	for _, f := range factors {
		pStr := f.prime.String()
		if !seenPrimes[pStr] {
			seenPrimes[pStr] = true
			badPrimes = append(badPrimes, new(big.Int).Set(f.prime))
		}
	}

	// Sort bad primes
	sort.Slice(badPrimes, func(i, j int) bool {
		return badPrimes[i].Cmp(badPrimes[j]) < 0
	})

	// Generate squarefree divisors of d and d'
	divsD := generateSquarefreeDivisors(d)
	divsDPrime := generateSquarefreeDivisors(dPrime)

	if err := fsm.TransitionTo(StateDescentLocallyFiltered); err != nil {
		return nil, err
	}

	// 5. Test local solvability for phi-Selmer candidates (divisors of d)
	var phiSelmer []*big.Int
	for _, d1 := range divsD {
		d2 := new(big.Int).Div(d, d1)
		// Check real
		if !IsQuarticRealSoluble(d1, c, d2) {
			continue
		}
		// Check 2-adic
		if !IsQuarticTwoAdicSoluble(d1, c, d2) {
			continue
		}
		// Check odd bad primes
		solubleOdd := true
		for _, p := range badPrimes {
			if p.Cmp(big.NewInt(2)) == 0 {
				continue
			}
			if !IsQuarticOddPrimeSoluble(d1, c, d2, p) {
				solubleOdd = false
				break
			}
		}
		if solubleOdd {
			phiSelmer = append(phiSelmer, d1)
		}
	}

	// 6. Test local solvability for psi-Selmer candidates (divisors of d')
	var psiSelmer []*big.Int
	for _, dPrime1 := range divsDPrime {
		dPrime2 := new(big.Int).Div(dPrime, dPrime1)
		// Check real
		if !IsQuarticRealSoluble(dPrime1, cPrime, dPrime2) {
			continue
		}
		// Check 2-adic
		if !IsQuarticTwoAdicSoluble(dPrime1, cPrime, dPrime2) {
			continue
		}
		// Check odd bad primes
		solubleOdd := true
		for _, p := range badPrimes {
			if p.Cmp(big.NewInt(2)) == 0 {
				continue
			}
			if !IsQuarticOddPrimeSoluble(dPrime1, cPrime, dPrime2, p) {
				solubleOdd = false
				break
			}
		}
		if solubleOdd {
			psiSelmer = append(psiSelmer, dPrime1)
		}
	}

	if err := fsm.TransitionTo(StateDescentSelmerSolved); err != nil {
		return nil, err
	}

	// 7. Compute F_2 dimensions
	phiDim := computeF2Dimension(phiSelmer, badPrimes)
	psiDim := computeF2Dimension(psiSelmer, badPrimes)

	// In classical 2-isogeny descent, the rank bound is dim S^(phi) + dim S^(psi) - 2.
	// Note: dim S^(phi) >= 1 (contains image of (0,0) and identity O)
	// and dim S^(psi) >= 1 (contains image of (0,0) and O).
	rankUpper := phiDim + psiDim - 2
	if rankUpper < 0 {
		rankUpper = 0
	}

	if err := fsm.TransitionTo(StateDescentPointsSearched); err != nil {
		return nil, err
	}

	var gens []EllipticPoint
	var rankLower int
	if rankUpper > 0 {
		WithTraceSink(nil, func() {
			var candidates []EllipticPoint
			// 8a. Search phi-Selmer quartics: N^2 = d1*M^4 + c*M^2*e^2 + d2*e^4
			for _, d1 := range phiSelmer {
				d2 := new(big.Int).Div(d, d1)
				pts := SearchQuarticRationalPoints(d1, c, d2, 50)
				for _, pt := range pts {
					origPt := unshiftAndUnscalePoint(pt, chosenRoot, scaleD)
					if IsOnCurve(origPt, aRat, bRat) && !isPointTorsion(origPt, aRat, bRat) {
						candidates = append(candidates, origPt)
					}
				}
			}

			// 8b. Search psi-Selmer quartics: N^2 = dPrime1*M^4 + cPrime*M^2*e^2 + dPrime2*e^4
			for _, dPrime1 := range psiSelmer {
				dPrime2 := new(big.Int).Div(dPrime, dPrime1)
				pts := SearchQuarticRationalPoints(dPrime1, cPrime, dPrime2, 50)
				for _, pt := range pts {
					mappedPt, err := mapDualPointToCurve(pt, dPrime)
					if err != nil {
						continue
					}
					origPt := unshiftAndUnscalePoint(mappedPt, chosenRoot, scaleD)
					if IsOnCurve(origPt, aRat, bRat) && !isPointTorsion(origPt, aRat, bRat) {
						candidates = append(candidates, origPt)
					}
				}
			}

			// 9. Filter linearly independent infinite-order generators
			gens = filterIndependentGenerators(candidates, aRat, bRat, rankUpper)
			rankLower = len(gens)
		})
	}

	if err := fsm.TransitionTo(StateDescentCompleted); err != nil {
		return nil, err
	}

	res := &TwoDescentResult{
		RankLowerBound: rankLower,
		RankUpperBound: rankUpper,
		IsExact:        (rankLower == rankUpper),
		SelmerPhiDim:   phiDim,
		SelmerPsiDim:   psiDim,
		Generators:     gens,
	}

	return res, nil
}

// -------------------------------------------------------------------------
// Quartic Sieve & Rational Point Pullback
// -------------------------------------------------------------------------

// gcdInt64 computes greatest common divisor of two int64 values.
func gcdInt64(a, b int64) int64 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// SearchQuarticRationalPoints searches for coprime integers (M, e) with 1 <= e <= bound, 1 <= M <= max(100, 2*bound)
// such that N^2 = d1*M^4 + c*M^2*e^2 + d2*e^4 has an integer square solution N.
// It maps solutions to points (x', y') = (d1*M^2/e^2, d1*M*N/e^3) on y'^2 = x'^3 + c*x'^2 + d*x'.
func SearchQuarticRationalPoints(d1, c, d2 *big.Int, bound int) []EllipticPoint {
	var points []EllipticPoint
	if bound < 1 {
		bound = 50
	}
	maxM := int64(bound * 2)
	if maxM < 100 {
		maxM = 100
	}

	// Quadratic residues mod 64 mask
	const squareMask64 uint64 = (1 << 0) | (1 << 1) | (1 << 4) | (1 << 9) |
		(1 << 16) | (1 << 17) | (1 << 25) | (1 << 33) |
		(1 << 36) | (1 << 41) | (1 << 49) | (1 << 57)

	for e := int64(1); e <= int64(bound); e++ {
		eBig := big.NewInt(e)
		e2 := new(big.Int).Mul(eBig, eBig)
		e4 := new(big.Int).Mul(e2, e2)
		term3 := new(big.Int).Mul(d2, e4)
		cE2 := new(big.Int).Mul(c, e2)

		for m := int64(1); m <= maxM; m++ {
			if gcdInt64(m, e) != 1 {
				continue
			}

			mBig := big.NewInt(m)
			m2 := new(big.Int).Mul(mBig, mBig)
			m4 := new(big.Int).Mul(m2, m2)

			term1 := new(big.Int).Mul(d1, m4)
			term2 := new(big.Int).Mul(cE2, m2)

			val := new(big.Int).Add(term1, term2)
			val.Add(val, term3)

			if val.Sign() <= 0 {
				continue
			}

			// Bitwise mod 64 test
			mod64 := val.Int64() & 63
			if (squareMask64 & (uint64(1) << mod64)) == 0 {
				continue
			}

			// Exact integer square check
			s := new(big.Int).Sqrt(val)
			s2 := new(big.Int).Mul(s, s)
			if s2.Cmp(val) != 0 {
				continue
			}

			// N = s
			// x' = (d1 * M^2) / e^2
			// y' = (d1 * M * N) / e^3
			numX := new(big.Int).Mul(d1, m2)
			xRat := new(big.Rat).SetFrac(numX, e2)

			numY := new(big.Int).Mul(d1, mBig)
			numY.Mul(numY, s)
			denY := new(big.Int).Mul(e2, eBig)
			yRat := new(big.Rat).SetFrac(numY, denY)

			if yRat.Sign() < 0 {
				yRat.Neg(yRat)
			}

			pt := NewEllipticPoint(xRat, yRat)
			points = append(points, pt)
		}
	}

	return points
}

// mapDualPointToCurve maps a point Q = (X, Y) on dual curve E': Y^2 = X^3 + cPrime*X^2 + dPrime*X
// to point (x', y') on E: y'^2 = x'^3 + c*x'^2 + d*x' via dual isogeny psi.
func mapDualPointToCurve(q EllipticPoint, dPrime *big.Int) (EllipticPoint, error) {
	if q.IsInf || q.X.Sign() == 0 {
		return InfinityPoint(), nil
	}

	// x' = Y^2 / (4 * X^2)
	x2 := new(big.Rat).Mul(q.X, q.X)
	fourX2 := new(big.Rat).Mul(big.NewRat(4, 1), x2)
	y2 := new(big.Rat).Mul(q.Y, q.Y)
	xPrime := new(big.Rat).Quo(y2, fourX2)

	// y' = Y * (X^2 - d') / (8 * X^2)
	dPrimeRat := new(big.Rat).SetInt(dPrime)
	x2MinusD := new(big.Rat).Sub(x2, dPrimeRat)
	numY := new(big.Rat).Mul(q.Y, x2MinusD)
	eightX2 := new(big.Rat).Mul(big.NewRat(8, 1), x2)
	yPrime := new(big.Rat).Quo(numY, eightX2)

	if yPrime.Sign() < 0 {
		yPrime.Neg(yPrime)
	}

	return NewEllipticPoint(xPrime, yPrime), nil
}

// unshiftAndUnscalePoint transforms point from E: y'^2 = x'^3 + c*x'^2 + d*x' back to
// original curve y^2 = x^3 + Ax + B by x = x' / u^2 + x0, y = y' / u^3.
func unshiftAndUnscalePoint(p EllipticPoint, rootRat *big.Rat, u *big.Int) EllipticPoint {
	if p.IsInf {
		return InfinityPoint()
	}

	u2 := new(big.Int).Mul(u, u)
	u3 := new(big.Int).Mul(u2, u)

	u2Rat := new(big.Rat).SetInt(u2)
	u3Rat := new(big.Rat).SetInt(u3)

	xOrig := new(big.Rat).Quo(p.X, u2Rat)
	xOrig.Add(xOrig, rootRat)

	yOrig := new(big.Rat).Quo(p.Y, u3Rat)
	if yOrig.Sign() < 0 {
		yOrig.Neg(yOrig)
	}

	return NewEllipticPoint(xOrig, yOrig)
}

// isPointTorsion checks whether point p has finite order dividing {1..10, 12} using Mazur's theorem.
func isPointTorsion(p EllipticPoint, aRat, bRat *big.Rat) bool {
	if p.IsInf {
		return true
	}
	for m := int64(1); m <= 12; m++ {
		mP, err := ScalarMulEllipticPoint(p, big.NewInt(m), aRat, bRat)
		if err != nil {
			return true
		}
		if mP.IsInf {
			return true
		}
	}
	return false
}

// filterIndependentGenerators selects up to maxRank linearly independent infinite-order generators.
func filterIndependentGenerators(candidates []EllipticPoint, aRat, bRat *big.Rat, maxRank int) []EllipticPoint {
	var gens []EllipticPoint
	for _, cand := range candidates {
		if len(gens) >= maxRank {
			break
		}
		if isPointTorsion(cand, aRat, bRat) {
			continue
		}

		dependent := false
		for _, g := range gens {
			if cand.Equal(g) {
				dependent = true
				break
			}
			candNeg := NewEllipticPoint(cand.X, new(big.Rat).Neg(cand.Y))
			if candNeg.Equal(g) {
				dependent = true
				break
			}

			// Check small multiples
			for k := int64(2); k <= 4; k++ {
				kG, err := ScalarMulEllipticPoint(g, big.NewInt(k), aRat, bRat)
				if err == nil && (cand.Equal(kG) || candNeg.Equal(kG)) {
					dependent = true
					break
				}
				kCand, err := ScalarMulEllipticPoint(cand, big.NewInt(k), aRat, bRat)
				if err == nil && (g.Equal(kCand) || g.Equal(NewEllipticPoint(kCand.X, new(big.Rat).Neg(kCand.Y)))) {
					dependent = true
					break
				}
			}
			if dependent {
				break
			}
		}

		if !dependent {
			gens = append(gens, cand)
		}
	}
	return gens
}

// -------------------------------------------------------------------------
// CLI / Evaluator Handlers & Registration
// -------------------------------------------------------------------------

func init() {
	RegisterHandler("ec_rank_bound", evalEcRankBound)
	RegisterHandler("ec_generators", evalEcGenerators)
}

func evalEcRankBound(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_invalid_args"))
	}
	aRat, err := evalToRat(args[0], env)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_a_parameter_error", err))
	}
	bRat, err := evalToRat(args[1], env)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_b_parameter_error", err))
	}

	res, err := ComputeTwoDescentRankBound(aRat, bRat, nil)
	if err != nil {
		return nil, err
	}

	var exactNode Node
	if res.IsExact {
		exactNode = &VarNode{Name: "true"}
	} else {
		exactNode = &VarNode{Name: "false"}
	}

	resNode := &ListNode{
		Elements: []Node{
			&RationalNode{Val: big.NewRat(int64(res.RankLowerBound), 1)},
			&RationalNode{Val: big.NewRat(int64(res.RankUpperBound), 1)},
			exactNode,
		},
	}

	RecordTraceRewrite(RuleEllipticDescent, &ListNode{Elements: args}, resNode,
		i18n.T("elliptic.trace_elliptic_descent", aRat.RatString(), bRat.RatString(), res.RankLowerBound, res.RankUpperBound, res.SelmerPhiDim, res.SelmerPsiDim))

	return resNode, nil
}

func evalEcGenerators(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_invalid_args"))
	}
	aRat, err := evalToRat(args[0], env)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_a_parameter_error", err))
	}
	bRat, err := evalToRat(args[1], env)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_b_parameter_error", err))
	}

	res, err := ComputeTwoDescentRankBound(aRat, bRat, nil)
	if err != nil {
		return nil, err
	}

	ptNodes := make([]Node, len(res.Generators))
	for i, pt := range res.Generators {
		ptNodes[i] = pointToNodeSafely(pt)
	}

	resNode := &ListNode{Elements: ptNodes}

	RecordTraceRewrite(RuleEllipticDescent, &ListNode{Elements: args}, resNode,
		i18n.T("elliptic.trace_elliptic_generators", len(res.Generators)))

	return resNode, nil
}



