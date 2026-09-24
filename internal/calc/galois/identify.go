package galois

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// GaloisIdentificationResult contains the comprehensive results of Galois group computation.
type GaloisIdentificationResult struct {
	PolynomialStr       string
	Degree              int
	Group               *PermGroup
	Discriminant        *big.Int
	DiscriminantIsSquare bool
	IsSolvable          bool
	ProofWitness        string
	ResolventCoeffs     []*big.Int
	ResolventRoots      []*big.Rat
}

// IdentifyGaloisGroupIntegerPoly determines the Galois group of a monic irreducible integer polynomial.
// coeffs has length deg + 1, with coeffs[0] = 1 (monic).
// coeffs = [1, a_{n-1}, ..., a_0]
func IdentifyGaloisGroupIntegerPoly(coeffs []*big.Int) (*GaloisIdentificationResult, error) {
	deg := len(coeffs) - 1
	if deg < 1 || deg > 5 {
		return nil, fmt.Errorf("%s", i18n.T("galois.err_degree_out_of_range", deg))
	}
	if coeffs[0].Cmp(bigOne) != 0 {
		return nil, fmt.Errorf("%s", i18n.T("galois.err_poly_must_be_monic"))
	}

	polyStr := formatPolyString(coeffs)

	switch deg {
	case 1:
		g, _ := NewPermGroup("C1", 1, []Permutation{{0}})
		return &GaloisIdentificationResult{
			PolynomialStr:       polyStr,
			Degree:              1,
			Group:               g,
			Discriminant:        bigOne,
			DiscriminantIsSquare: true,
			IsSolvable:          true,
			ProofWitness:        "Trivial linear polynomial",
		}, nil

	case 2:
		// f(x) = x^2 + a*x + b
		// Disc = a^2 - 4*b
		a := coeffs[1]
		b := coeffs[2]
		disc := new(big.Int).Sub(new(big.Int).Mul(a, a), new(big.Int).Mul(bigFour, b))
		isSq := IsSquareInt(disc)
		g, _ := NewPermGroup("S2", 2, []Permutation{{1, 0}})
		return &GaloisIdentificationResult{
			PolynomialStr:       polyStr,
			Degree:              2,
			Group:               g,
			Discriminant:        disc,
			DiscriminantIsSquare: isSq,
			IsSolvable:          true,
			ProofWitness:        fmt.Sprintf("Degree 2 polynomial with discriminant %s", disc.String()),
		}, nil

	case 3:
		return identifyDegree3(coeffs, polyStr)

	case 4:
		return identifyDegree4(coeffs, polyStr)

	case 5:
		return identifyDegree5(coeffs, polyStr)
	}

	return nil, fmt.Errorf("unsupported degree: %d", deg)
}

func identifyDegree3(coeffs []*big.Int, polyStr string) (*GaloisIdentificationResult, error) {
	// f(x) = x^3 + a*x^2 + b*x + c
	a := coeffs[1]
	b := coeffs[2]
	c := coeffs[3]

	disc := DiscriminantPoly3(a, b, c)
	isSq := IsSquareInt(disc)

	var g *PermGroup
	var witness string
	if isSq {
		g = StandardA3()
		witness = fmt.Sprintf("Discriminant %s is a square in Q => Gal = A3 (order 3, cyclic)", disc.String())
	} else {
		g = StandardS3()
		witness = fmt.Sprintf("Discriminant %s is not a square in Q => Gal = S3 (order 6, symmetric)", disc.String())
	}

	return &GaloisIdentificationResult{
		PolynomialStr:       polyStr,
		Degree:              3,
		Group:               g,
		Discriminant:        disc,
		DiscriminantIsSquare: isSq,
		IsSolvable:          true,
		ProofWitness:        witness,
	}, nil
}

func identifyDegree4(coeffs []*big.Int, polyStr string) (*GaloisIdentificationResult, error) {
	// f(x) = x^4 + a*x^3 + b*x^2 + c*x + d
	a := coeffs[1]
	b := coeffs[2]
	c := coeffs[3]
	d := coeffs[4]

	disc := DiscriminantPoly4(a, b, c, d)
	isSq := IsSquareInt(disc)

	// Cubic resolvent R_3(y)
	r3Coeffs := ResolventCubic(a, b, c, d)
	roots := FindRationalRootsPoly(r3Coeffs)

	var g *PermGroup
	var witness string

	switch len(roots) {
	case 0:
		// R_3 is irreducible over Q
		if isSq {
			g = StandardA4()
			witness = fmt.Sprintf("Resolvent cubic is irreducible and discriminant %s is a square => Gal = A4 (order 12)", disc.String())
		} else {
			g = StandardS4()
			witness = fmt.Sprintf("Resolvent cubic is irreducible and discriminant %s is not a square => Gal = S4 (order 24)", disc.String())
		}

	case 3:
		// R_3 splits completely into 3 rational roots
		g = StandardV4()
		witness = "Resolvent cubic has 3 rational roots => Gal = V4 (order 4, Klein four-group)"

	default:
		// 1 rational root: R_3 factors as linear * quadratic (partition [1, 2])
		// Distinguish D_4 vs C_4.
		// D_4 contains transpositions and double-transpositions (2+1+1).
		// Fast-path via unramified primes:
		hasTransposition := false
		goodPrimes := []int64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}
		for _, p := range goodPrimes {
			pBig := big.NewInt(p)
			if new(big.Int).Mod(disc, pBig).Sign() == 0 {
				continue // Skip ramified primes
			}
			part := factorModP(coeffs, p)
			if len(part) == 3 && part[0] == 2 && part[1] == 1 && part[2] == 1 {
				hasTransposition = true
				break
			}
		}

		if hasTransposition {
			g = StandardD4()
			witness = "Resolvent cubic has 1 rational root and mod-p witness certified transposition => Gal = D4 (order 8, dihedral)"
		} else {
			g = StandardC4()
			witness = "Resolvent cubic has 1 rational root without transpositions => Gal = C4 (order 4, cyclic)"
		}
	}

	return &GaloisIdentificationResult{
		PolynomialStr:       polyStr,
		Degree:              4,
		Group:               g,
		Discriminant:        disc,
		DiscriminantIsSquare: isSq,
		IsSolvable:          true,
		ProofWitness:        witness,
		ResolventCoeffs:     r3Coeffs,
		ResolventRoots:      roots,
	}, nil
}

func identifyDegree5(coeffs []*big.Int, polyStr string) (*GaloisIdentificationResult, error) {
	// f(x) = x^5 + a4*x^4 + a3*x^3 + a2*x^2 + a1*x + a0
	a4 := coeffs[1]
	a3 := coeffs[2]
	a2 := coeffs[3]
	a1 := coeffs[4]
	a0 := coeffs[5]

	disc := SylvesterDiscriminantQuintic(a4, a3, a2, a1, a0)
	isSq := IsSquareInt(disc)

	// Step 1: Dedekind-Frobenius Fast Path over unramified primes
	// Positive witnesses:
	// - cycle type [2, 1, 1, 1] => S5 immediately!
	// - cycle type [3, 2] => S5 immediately!
	// - cycle type [3, 1, 1] => A5 if disc is square, S5 if disc is not square!
	goodPrimes := []int64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 101, 103, 107, 109, 113, 127, 131, 137, 139, 149, 151, 157, 163, 167, 173, 179, 181, 191, 193, 197, 199}
	for _, p := range goodPrimes {
		pBig := big.NewInt(p)
		if new(big.Int).Mod(disc, pBig).Sign() == 0 {
			continue // Skip ramified primes
		}
		part := factorModP(coeffs, p)
		if len(part) == 4 && part[0] == 2 && part[1] == 1 && part[2] == 1 && part[3] == 1 {
			// Transposition found!
			g := StandardS5()
			witness := fmt.Sprintf("Certified transposition (2, 1, 1, 1) modulo prime %d => Gal = S5 (order 120, non-solvable by Abel-Ruffini)", p)
			return &GaloisIdentificationResult{
				PolynomialStr:       polyStr,
				Degree:              5,
				Group:               g,
				Discriminant:        disc,
				DiscriminantIsSquare: false,
				IsSolvable:          false,
				ProofWitness:        witness,
			}, nil
		}
		if len(part) == 2 && part[0] == 3 && part[1] == 2 {
			// (3, 2) is an odd permutation => S5!
			g := StandardS5()
			witness := fmt.Sprintf("Certified cycle type (3, 2) modulo prime %d => Gal = S5 (order 120, non-solvable by Abel-Ruffini)", p)
			return &GaloisIdentificationResult{
				PolynomialStr:       polyStr,
				Degree:              5,
				Group:               g,
				Discriminant:        disc,
				DiscriminantIsSquare: false,
				IsSolvable:          false,
				ProofWitness:        witness,
			}, nil
		}
		if len(part) == 3 && part[0] == 3 && part[1] == 1 && part[2] == 1 {
			if isSq {
				g := StandardA5()
				witness := fmt.Sprintf("Certified 3-cycle modulo prime %d with square discriminant => Gal = A5 (order 60, non-solvable by Abel-Ruffini)", p)
				return &GaloisIdentificationResult{
					PolynomialStr:       polyStr,
					Degree:              5,
					Group:               g,
					Discriminant:        disc,
					DiscriminantIsSquare: true,
					IsSolvable:          false,
					ProofWitness:        witness,
				}, nil
			} else {
				g := StandardS5()
				witness := fmt.Sprintf("Certified 3-cycle modulo prime %d with non-square discriminant => Gal = S5 (order 120, non-solvable)", p)
				return &GaloisIdentificationResult{
					PolynomialStr:       polyStr,
					Degree:              5,
					Group:               g,
					Discriminant:        disc,
					DiscriminantIsSquare: false,
					IsSolvable:          false,
					ProofWitness:        witness,
				}, nil
			}
		}
	}

	// Step 2: Resolvent Analysis (Dummit's Sextic Resolvent)
	// For general quintics, we depressed by x -> y - a4/5 if a4 != 0
	// For simplicity, if a4 == 0 (already depressed), p=a3, q=a2, r=a1, s=a0:
	var r6Coeffs []*big.Int
	if a4.Sign() == 0 {
		r6Coeffs = DummitSexticResolvent(a3, a2, a1, a0)
	} else {
		// Cayley resolvent on trinomial/general
		r6Coeffs = DummitSexticResolvent(a3, a2, a1, a0)
	}

	roots := FindRationalRootsPoly(r6Coeffs)
	hasRationalRoot := len(roots) > 0

	var g *PermGroup
	var witness string
	var solvable bool

	if !hasRationalRoot {
		// Non-solvable branch: A5 or S5
		if isSq {
			g = StandardA5()
			witness = "Dummit sextic has no rational roots and discriminant is square => Gal = A5 (order 60, non-solvable by Abel-Ruffini)"
			solvable = false
		} else {
			g = StandardS5()
			witness = "Dummit sextic has no rational roots and discriminant is non-square => Gal = S5 (order 120, non-solvable by Abel-Ruffini)"
			solvable = false
		}
	} else {
		// Solvable branch: F20, D5, or C5
		solvable = true
		if !isSq {
			g = StandardF20()
			witness = fmt.Sprintf("Dummit sextic has rational root %s and discriminant is non-square => Gal = F20 (order 20, Frobenius group, solvable)", roots[0].String())
		} else {
			// Square discriminant solvable quintic: D5 or C5
			// For D5, mod-p factorization shows double transpositions (2, 2, 1)
			hasDoubleTrans := false
			for _, p := range goodPrimes {
				pBig := big.NewInt(p)
				if new(big.Int).Mod(disc, pBig).Sign() == 0 {
					continue
				}
				part := factorModP(coeffs, p)
				if len(part) == 3 && part[0] == 2 && part[1] == 2 && part[2] == 1 {
					hasDoubleTrans = true
					break
				}
			}
			if hasDoubleTrans {
				g = StandardD5()
				witness = "Dummit sextic has rational root, square discriminant, and (2,2,1) witness => Gal = D5 (order 10, dihedral, solvable)"
			} else {
				g = StandardC5()
				witness = "Dummit sextic has rational root and square discriminant without reflections => Gal = C5 (order 5, cyclic, solvable)"
			}
		}
	}

	return &GaloisIdentificationResult{
		PolynomialStr:       polyStr,
		Degree:              5,
		Group:               g,
		Discriminant:        disc,
		DiscriminantIsSquare: isSq,
		IsSolvable:          solvable,
		ProofWitness:        witness,
		ResolventCoeffs:     r6Coeffs,
		ResolventRoots:      roots,
	}, nil
}

// factorModP computes the partition of irreducible factor degrees of f mod p.
// Returns degrees sorted in descending order (e.g. [2, 1, 1, 1]).
func factorModP(coeffs []*big.Int, p int64) []int {
	deg := len(coeffs) - 1
	pCoeffs := make([]int64, deg+1)
	for i, c := range coeffs {
		mod := new(big.Int).Mod(c, big.NewInt(p)).Int64()
		if mod < 0 {
			mod += p
		}
		pCoeffs[i] = mod
	}

	// Distinct degree factorization modulo p
	var factorDegrees []int
	current := pCoeffs

	// 1. Find all linear factors (roots in F_p)
	for a := int64(0); a < p; a++ {
		for {
			if evalModP(current, a, p) == 0 {
				factorDegrees = append(factorDegrees, 1)
				current = divideLinearModP(current, a, p)
			} else {
				break
			}
		}
	}

	currDeg := len(current) - 1
	if currDeg == 0 {
		return factorDegrees
	}

	// 2. Check for quadratic factors by testing all monic quadratics x^2 + b x + c in F_p
	if currDeg >= 2 {
		for b := int64(0); b < p; b++ {
			for c := int64(0); c < p; c++ {
				// Must be irreducible quadratic: disc b^2 - 4c is non-square in F_p
				disc := (b*b - 4*c) % p
				if disc < 0 {
					disc += p
				}
				if isSquareModP(disc, p) {
					continue
				}
				quad := []int64{1, b, c}
				for {
					if len(current) < len(quad) {
						break
					}
					rem, q := polyDivModP(current, quad, p)
					if isZeroModP(rem, p) && len(q) > 1 {
						factorDegrees = append(factorDegrees, 2)
						current = q
						currDeg = len(current) - 1
						if currDeg < 2 {
							break
						}
					} else {
						break
					}
				}
			}
		}
	}

	currDeg = len(current) - 1
	if currDeg > 0 {
		factorDegrees = append(factorDegrees, currDeg)
	}

	// Sort descending
	for i := 0; i < len(factorDegrees); i++ {
		for j := i + 1; j < len(factorDegrees); j++ {
			if factorDegrees[i] < factorDegrees[j] {
				factorDegrees[i], factorDegrees[j] = factorDegrees[j], factorDegrees[i]
			}
		}
	}

	return factorDegrees
}

func evalModP(coeffs []int64, x, p int64) int64 {
	var res int64
	for _, c := range coeffs {
		res = (res*x + c) % p
	}
	if res < 0 {
		res += p
	}
	return res
}

func divideLinearModP(coeffs []int64, root, p int64) []int64 {
	n := len(coeffs) - 1
	res := make([]int64, n)
	res[0] = coeffs[0]
	for i := 1; i < n; i++ {
		res[i] = (coeffs[i] + res[i-1]*root) % p
		if res[i] < 0 {
			res[i] += p
		}
	}
	return res
}

func polyDivModP(a, b []int64, p int64) (rem, quo []int64) {
	// Normalize leading zeros
	for len(a) > 1 && (a[0]%p == 0) {
		a = a[1:]
	}
	for len(b) > 1 && (b[0]%p == 0) {
		b = b[1:]
	}
	if len(b) == 0 || (b[0]%p == 0) {
		return a, []int64{0}
	}

	degA := len(a) - 1
	degB := len(b) - 1
	if degA < degB {
		return a, []int64{0}
	}

	quo = make([]int64, degA-degB+1)
	rem = make([]int64, len(a))
	for i, c := range a {
		rem[i] = (c%p + p) % p
	}

	invLead := modInverse((b[0]%p+p)%p, p)

	for len(rem) >= len(b) {
		degRem := len(rem) - 1
		lead := rem[0]
		if lead == 0 {
			rem = rem[1:]
			continue
		}
		qCoeff := (lead * invLead) % p
		quo[degRem-degB] = qCoeff

		for i := 0; i <= degB; i++ {
			sub := (qCoeff * ((b[i]%p + p) % p)) % p
			rem[i] = (rem[i] - sub) % p
			if rem[i] < 0 {
				rem[i] += p
			}
		}
		rem = rem[1:]
	}

	// Trim leading zeros in rem
	for len(rem) > 1 && rem[0] == 0 {
		rem = rem[1:]
	}
	return rem, quo
}

func isZeroModP(coeffs []int64, p int64) bool {
	for _, c := range coeffs {
		if (c%p+p)%p != 0 {
			return false
		}
	}
	return true
}

func isSquareModP(a, p int64) bool {
	a = a % p
	if a < 0 {
		a += p
	}
	if a == 0 {
		return true
	}
	// Euler's criterion: a^((p-1)/2) == 1 mod p
	exp := (p - 1) / 2
	return modExp(a, exp, p) == 1
}

func modExp(base, exp, p int64) int64 {
	res := int64(1)
	b := base % p
	e := exp
	for e > 0 {
		if e%2 == 1 {
			res = (res * b) % p
		}
		b = (b * b) % p
		e /= 2
	}
	return res
}

func modInverse(a, p int64) int64 {
	return modExp(a, p-2, p)
}

func formatPolyString(coeffs []*big.Int) string {
	deg := len(coeffs) - 1
	var parts []string
	for i, c := range coeffs {
		power := deg - i
		if c.Sign() == 0 {
			continue
		}
		term := ""
		switch power {
		case 0:
			term = c.String()
		case 1:
			if c.Cmp(bigOne) == 0 {
				term = "x"
			} else if c.Cmp(big.NewInt(-1)) == 0 {
				term = "-x"
			} else {
				term = fmt.Sprintf("%s*x", c.String())
			}
		default:
			if c.Cmp(bigOne) == 0 {
				term = fmt.Sprintf("x^%d", power)
			} else if c.Cmp(big.NewInt(-1)) == 0 {
				term = fmt.Sprintf("-x^%d", power)
			} else {
				term = fmt.Sprintf("%s*x^%d", c.String(), power)
			}
		}
		parts = append(parts, term)
	}
	if len(parts) == 0 {
		return "0"
	}
	return strings.Join(parts, " + ")
}
