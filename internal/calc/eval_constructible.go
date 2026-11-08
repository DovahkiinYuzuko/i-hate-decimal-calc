package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// ConstructibilityStatus represents the 3-valued constructibility result.
type ConstructibilityStatus int

const (
	ConstructibleFalse ConstructibilityStatus = iota
	ConstructibleTrue
	ConstructibleUndetermined
)

func (s ConstructibilityStatus) String() string {
	switch s {
	case ConstructibleTrue:
		return "True"
	case ConstructibleFalse:
		return "False"
	default:
		return "Undetermined"
	}
}

// ConstructibilityCertificate holds algebraic proof details for constructibility.
type ConstructibilityCertificate struct {
	Status        ConstructibilityStatus
	Degree        int
	ResolventPoly []*big.Int
	RationalRoot  *big.Rat
	Reason        string
}

func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// computeCubicResolventRationalRoots constructs the depressed cubic resolvent
// R3(theta) = theta^3 - p*theta^2 - 4r*theta + (4pr - q^2)
// for quartic poly: a0 + a1*x + a2*x^2 + a3*x^3 + a4*x^4.
func computeCubicResolventRationalRoots(poly []*big.Int) ([]*big.Rat, []*big.Int) {
	if len(poly) < 5 {
		return nil, nil
	}

	// Make monic in rational coefficients: x^4 + c3*x^3 + c2*x^2 + c1*x + c0
	a4 := new(big.Rat).SetInt(poly[4])
	c3 := new(big.Rat).SetFrac(poly[3], poly[4])
	c2 := new(big.Rat).SetFrac(poly[2], poly[4])
	c1 := new(big.Rat).SetFrac(poly[1], poly[4])
	c0 := new(big.Rat).SetFrac(poly[0], poly[4])
	_ = a4

	// Tschirnhaus: x <- y - c3/4
	// p = c2 - 3/8 * c3^2
	c3Sq := new(big.Rat).Mul(c3, c3)
	threeEighths := big.NewRat(3, 8)
	termP := new(big.Rat).Mul(threeEighths, c3Sq)
	p := new(big.Rat).Sub(c2, termP)

	// q = c1 - 1/2 * c2 * c3 + 1/8 * c3^3
	half := big.NewRat(1, 2)
	termQ1 := new(big.Rat).Mul(half, new(big.Rat).Mul(c2, c3))
	oneEighth := big.NewRat(1, 8)
	termQ2 := new(big.Rat).Mul(oneEighth, new(big.Rat).Mul(c3Sq, c3))
	q := new(big.Rat).Sub(c1, termQ1)
	q.Add(q, termQ2)

	// r = c0 - 1/4 * c1*c3 + 1/16 * c2*c3^2 - 3/256 * c3^4
	oneFourth := big.NewRat(1, 4)
	termR1 := new(big.Rat).Mul(oneFourth, new(big.Rat).Mul(c1, c3))
	oneSixteenth := big.NewRat(1, 16)
	termR2 := new(big.Rat).Mul(oneSixteenth, new(big.Rat).Mul(c2, c3Sq))
	three256 := big.NewRat(3, 256)
	c3Pow4 := new(big.Rat).Mul(c3Sq, c3Sq)
	termR3 := new(big.Rat).Mul(three256, c3Pow4)
	r := new(big.Rat).Sub(c0, termR1)
	r.Add(r, termR2)
	r.Sub(r, termR3)

	// Resolvent: theta^3 - p*theta^2 - 4r*theta + (4pr - q^2)
	// Coeff 3: 1
	// Coeff 2: -p
	// Coeff 1: -4r
	// Coeff 0: 4pr - q^2
	rCoeff3 := big.NewRat(1, 1)
	rCoeff2 := new(big.Rat).Neg(p)
	four := big.NewRat(4, 1)
	rCoeff1 := new(big.Rat).Mul(new(big.Rat).Neg(four), r)

	fourPR := new(big.Rat).Mul(four, new(big.Rat).Mul(p, r))
	qSq := new(big.Rat).Mul(q, q)
	rCoeff0 := new(big.Rat).Sub(fourPR, qSq)

	ratCoeffs := []*big.Rat{rCoeff0, rCoeff1, rCoeff2, rCoeff3}

	// Clear denominators to make integer polynomial
	dens := make([]*big.Int, 4)
	nums := make([]*big.Int, 4)
	lcm := big.NewInt(1)
	for i := 0; i < 4; i++ {
		nums[i] = new(big.Int).Set(ratCoeffs[i].Num())
		dens[i] = new(big.Int).Set(ratCoeffs[i].Denom())
		gcd := new(big.Int).GCD(nil, nil, lcm, dens[i])
		lcm.Mul(lcm, dens[i])
		lcm.Div(lcm, gcd)
	}

	intResolvent := make([]*big.Int, 4)
	for i := 0; i < 4; i++ {
		scale := new(big.Int).Div(lcm, dens[i])
		intResolvent[i] = new(big.Int).Mul(nums[i], scale)
	}

	var roots []*big.Rat
	if intResolvent[0] == nil || intResolvent[0].Sign() == 0 {
		roots = append(roots, big.NewRat(0, 1))
	}
	candRoots := findRationalRoots(intResolvent)
	for _, c := range candRoots {
		roots = append(roots, new(big.Rat).SetFrac(c.num, c.denom))
	}
	return roots, intResolvent
}

// CheckConstructibility checks ruler and compass constructibility for polynomial poly.
func CheckConstructibility(poly []*big.Int) *ConstructibilityCertificate {
	// Trim leading zeros
	deg := len(poly) - 1
	for deg >= 0 && (poly[deg] == nil || poly[deg].Sign() == 0) {
		deg--
	}
	if deg <= 0 {
		return &ConstructibilityCertificate{
			Status: ConstructibleTrue,
			Degree: 0,
			Reason: "constructible.degree_le_2",
		}
	}

	// 1. Wantzel Necessary Condition: degree must be power of 2
	if !isPowerOfTwo(deg) {
		return &ConstructibilityCertificate{
			Status: ConstructibleFalse,
			Degree: deg,
			Reason: "constructible.degree_not_power_of_two",
		}
	}

	// 2. Degrees 1 and 2 are always constructible
	if deg <= 2 {
		return &ConstructibilityCertificate{
			Status: ConstructibleTrue,
			Degree: deg,
			Reason: "constructible.degree_le_2",
		}
	}

	// 3. Degree 4: Cubic Resolvent criterion (Galois 2-group condition)
	if deg == 4 {
		roots, resolventPoly := computeCubicResolventRationalRoots(poly)
		if len(roots) > 0 {
			return &ConstructibilityCertificate{
				Status:        ConstructibleTrue,
				Degree:        4,
				ResolventPoly: resolventPoly,
				RationalRoot:  roots[0],
				Reason:        "constructible.quartic_resolvent_rational_root",
			}
		}
		return &ConstructibilityCertificate{
			Status:        ConstructibleFalse,
			Degree:        4,
			ResolventPoly: resolventPoly,
			Reason:        "constructible.quartic_resolvent_no_rational_root",
		}
	}

	// 4. Degrees 8, 16, ... are undetermined without full Galois solver
	return &ConstructibilityCertificate{
		Status: ConstructibleUndetermined,
		Degree: deg,
		Reason: "constructible.degree_high_undetermined",
	}
}

// EvalIsConstructible evaluates constructibility of expression node.
func EvalIsConstructible(p Node, varName string, env *Env) (Node, error) {
	if p == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.constructible_poly_required", "is_constructible"))
	}

	evalP, err := Eval(expandNode(p))
	if err != nil {
		evalP = expandNode(p)
	}

	v := varName
	if v == "" {
		vars := collectVariables(evalP)
		if len(vars) == 0 {
			return &VarNode{Name: "true"}, nil
		}
		if len(vars) > 1 {
			return nil, fmt.Errorf("%s", i18n.T("errors.constructible_poly_required", "is_constructible"))
		}
		v = vars[0]
	}

	polyP, ok := extractPoly(evalP, v)
	if !ok || !isRationalPoly(polyP) {
		return nil, fmt.Errorf("%s", i18n.T("errors.constructible_poly_required", "is_constructible"))
	}

	// Convert to []*big.Int
	intCoeffs := univariatePolyToIntCoeffs(polyP)
	cert := CheckConstructibility(intCoeffs)

	switch cert.Status {
	case ConstructibleTrue:
		return &VarNode{Name: "true"}, nil
	case ConstructibleFalse:
		return &VarNode{Name: "false"}, nil
	default:
		return &VarNode{Name: "undetermined"}, nil
	}
}
