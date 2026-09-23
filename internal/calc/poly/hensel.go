package poly

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// extGCDInt computes the Extended Euclidean Algorithm on a and b,
// returning gcd, x, and y such that a*x + b*y = gcd.
func extGCDInt(a, b *big.Int) (*big.Int, *big.Int, *big.Int) {
	if b.Sign() == 0 {
		return new(big.Int).Set(a), big.NewInt(1), big.NewInt(0)
	}
	g, x1, y1 := extGCDInt(b, new(big.Int).Mod(a, b))
	q := new(big.Int).Div(a, b)
	x := new(big.Int).Set(y1)
	y := new(big.Int).Sub(x1, new(big.Int).Mul(q, y1))
	return g, x, y
}

// invModInt computes the modular inverse of a modulo m: a * x = 1 (mod m).
func invModInt(a, m *big.Int) (*big.Int, error) {
	modA := new(big.Int).Mod(a, m)
	if modA.Sign() < 0 {
		modA.Add(modA, m)
	}
	g, x, _ := extGCDInt(modA, m)
	if g.Cmp(big.NewInt(1)) != 0 {
		return nil, fmt.Errorf("%s", i18n.T("domain.err_no_inverse", a.String()))
	}
	x.Mod(x, m)
	if x.Sign() < 0 {
		x.Add(x, m)
	}
	return x, nil
}

// polyDegree returns the degree of a polynomial represented by []*big.Int.
// Returns -1 for the zero polynomial.
func polyDegree(p []*big.Int) int {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] != nil && p[i].Sign() != 0 {
			return i
		}
	}
	return -1
}

// trimPoly trims trailing zero coefficients.
func trimPoly(p []*big.Int) []*big.Int {
	deg := polyDegree(p)
	if deg < 0 {
		return []*big.Int{big.NewInt(0)}
	}
	res := make([]*big.Int, deg+1)
	for i := 0; i <= deg; i++ {
		if p[i] == nil {
			res[i] = big.NewInt(0)
		} else {
			res[i] = new(big.Int).Set(p[i])
		}
	}
	return res
}

// evalPolyMod evaluates f(x) modulo m using Horner's method.
func evalPolyMod(f []*big.Int, x, m *big.Int) *big.Int {
	deg := polyDegree(f)
	if deg < 0 {
		return big.NewInt(0)
	}
	res := new(big.Int).Mod(f[deg], m)
	for i := deg - 1; i >= 0; i-- {
		res.Mul(res, x)
		if f[i] != nil {
			res.Add(res, f[i])
		}
		res.Mod(res, m)
	}
	if res.Sign() < 0 {
		res.Add(res, m)
	}
	return res
}

// derivativePoly computes the formal derivative f'(x) of a polynomial.
func derivativePoly(f []*big.Int) []*big.Int {
	deg := polyDegree(f)
	if deg <= 0 {
		return []*big.Int{big.NewInt(0)}
	}
	res := make([]*big.Int, deg)
	for i := 1; i <= deg; i++ {
		if f[i] != nil {
			res[i-1] = new(big.Int).Mul(f[i], big.NewInt(int64(i)))
		} else {
			res[i-1] = big.NewInt(0)
		}
	}
	return trimPoly(res)
}

// HenselLiftRoot lifts a simple root r0 of f(x) = 0 mod p to a root mod p^targetK
// using the quadratic Newton-Hensel method:
// r_{k+1} = r_k - f(r_k) / f'(r_k) mod p^(2*k).
func HenselLiftRoot(fCoeffs []*big.Int, p *big.Int, r0 *big.Int, targetK int) (*big.Int, error) {
	if targetK < 1 {
		return nil, fmt.Errorf("%s", i18n.T("hensel.err_lift_target_precision"))
	}
	if p.Cmp(big.NewInt(2)) < 0 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_prime_less_than_2"))
	}

	f := trimPoly(fCoeffs)
	deg := polyDegree(f)
	if deg <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("hensel.err_derivative_zero", "deg <= 0"))
	}

	// Verify lc(f) != 0 mod p
	lc := new(big.Int).Mod(f[deg], p)
	if lc.Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("hensel.err_leading_coeff_divisible", f[deg].String(), p.String()))
	}

	fPrime := derivativePoly(f)

	// Verify f'(r0) != 0 mod p
	fPrimeR0 := evalPolyMod(fPrime, r0, p)
	if fPrimeR0.Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("hensel.err_derivative_zero", fPrimeR0.String()))
	}

	// Target modulus p^targetK
	targetMod := new(big.Int).Exp(p, big.NewInt(int64(targetK)), nil)

	r := new(big.Int).Mod(r0, p)
	if r.Sign() < 0 {
		r.Add(r, p)
	}

	currMod := new(big.Int).Set(p)
	currK := 1

	for currK < targetK {
		nextK := currK * 2
		if nextK > targetK {
			nextK = targetK
		}
		nextMod := new(big.Int).Exp(p, big.NewInt(int64(nextK)), nil)

		// f(r) mod nextMod
		fVal := evalPolyMod(f, r, nextMod)

		// invDeriv = f'(r)^(-1) mod currMod
		// (We only need the inverse mod currMod for Newton-Hensel step)
		fPrimeVal := evalPolyMod(fPrime, r, currMod)
		invDeriv, err := invModInt(fPrimeVal, currMod)
		if err != nil {
			return nil, err
		}

		// step = (f(r) * invDeriv) mod nextMod
		step := new(big.Int).Mul(fVal, invDeriv)
		step.Mod(step, nextMod)

		// r = (r - step) mod nextMod
		r.Sub(r, step)
		r.Mod(r, nextMod)
		if r.Sign() < 0 {
			r.Add(r, nextMod)
		}

		currMod = nextMod
		currK = nextK
	}

	r.Mod(r, targetMod)
	if r.Sign() < 0 {
		r.Add(r, targetMod)
	}
	return r, nil
}

// polyMod reduces all coefficients of a polynomial modulo m.
func polyMod(f []*big.Int, m *big.Int) []*big.Int {
	res := make([]*big.Int, len(f))
	for i, c := range f {
		if c == nil {
			res[i] = big.NewInt(0)
		} else {
			mod := new(big.Int).Mod(c, m)
			if mod.Sign() < 0 {
				mod.Add(mod, m)
			}
			res[i] = mod
		}
	}
	return trimPoly(res)
}

// polyAddMod adds two polynomials modulo m.
func polyAddMod(a, b []*big.Int, m *big.Int) []*big.Int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	res := make([]*big.Int, maxLen)
	for i := 0; i < maxLen; i++ {
		sum := big.NewInt(0)
		if i < len(a) && a[i] != nil {
			sum.Add(sum, a[i])
		}
		if i < len(b) && b[i] != nil {
			sum.Add(sum, b[i])
		}
		if m != nil {
			sum.Mod(sum, m)
			if sum.Sign() < 0 {
				sum.Add(sum, m)
			}
		}
		res[i] = sum
	}
	return trimPoly(res)
}

// polySubMod subtracts b from a modulo m.
func polySubMod(a, b []*big.Int, m *big.Int) []*big.Int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	res := make([]*big.Int, maxLen)
	for i := 0; i < maxLen; i++ {
		diff := big.NewInt(0)
		if i < len(a) && a[i] != nil {
			diff.Add(diff, a[i])
		}
		if i < len(b) && b[i] != nil {
			diff.Sub(diff, b[i])
		}
		if m != nil {
			diff.Mod(diff, m)
			if diff.Sign() < 0 {
				diff.Add(diff, m)
			}
		}
		res[i] = diff
	}
	return trimPoly(res)
}

// polyMulMod multiplies two polynomials modulo m.
func polyMulMod(a, b []*big.Int, m *big.Int) []*big.Int {
	degA := polyDegree(a)
	degB := polyDegree(b)
	if degA < 0 || degB < 0 {
		return []*big.Int{big.NewInt(0)}
	}
	res := make([]*big.Int, degA+degB+1)
	for i := range res {
		res[i] = big.NewInt(0)
	}
	for i := 0; i <= degA; i++ {
		if a[i] == nil || a[i].Sign() == 0 {
			continue
		}
		for j := 0; j <= degB; j++ {
			if b[j] == nil || b[j].Sign() == 0 {
				continue
			}
			term := new(big.Int).Mul(a[i], b[j])
			res[i+j].Add(res[i+j], term)
			if m != nil {
				res[i+j].Mod(res[i+j], m)
			}
		}
	}
	if m != nil {
		for i := range res {
			if res[i].Sign() < 0 {
				res[i].Add(res[i], m)
			}
		}
	}
	return trimPoly(res)
}

// polyDivRemMod computes quotient and remainder of a / b in F_p[x].
// Requires b to be non-zero and p to be prime.
func polyDivRemMod(a, b []*big.Int, p *big.Int) (q, r []*big.Int, err error) {
	degA := polyDegree(a)
	degB := polyDegree(b)
	if degB < 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("domain.err_poly_zero_division"))
	}
	if degA < degB {
		return []*big.Int{big.NewInt(0)}, trimPoly(a), nil
	}

	invLeadB, err := invModInt(b[degB], p)
	if err != nil {
		return nil, nil, err
	}

	rem := make([]*big.Int, len(a))
	for i, c := range a {
		rem[i] = new(big.Int).Mod(c, p)
		if rem[i].Sign() < 0 {
			rem[i].Add(rem[i], p)
		}
	}

	degRem := polyDegree(rem)
	quot := make([]*big.Int, degA-degB+1)
	for i := range quot {
		quot[i] = big.NewInt(0)
	}

	for degRem >= degB {
		scale := new(big.Int).Mul(rem[degRem], invLeadB)
		scale.Mod(scale, p)
		shift := degRem - degB
		quot[shift] = scale

		for i := 0; i <= degB; i++ {
			subTerm := new(big.Int).Mul(b[i], scale)
			rem[i+shift].Sub(rem[i+shift], subTerm)
			rem[i+shift].Mod(rem[i+shift], p)
			if rem[i+shift].Sign() < 0 {
				rem[i+shift].Add(rem[i+shift], p)
			}
		}
		degRem = polyDegree(rem)
	}

	return trimPoly(quot), trimPoly(rem), nil
}

// polyExtGCDMod computes extended GCD of a and b in F_p[x]: s*a + t*b = gcd.
// Resulting gcd is monic.
func polyExtGCDMod(a, b []*big.Int, p *big.Int) (gcd, s, t []*big.Int, err error) {
	degB := polyDegree(b)
	if degB < 0 {
		degA := polyDegree(a)
		if degA < 0 {
			return []*big.Int{big.NewInt(0)}, []*big.Int{big.NewInt(0)}, []*big.Int{big.NewInt(0)}, nil
		}
		// Make gcd monic
		leadInv, err := invModInt(a[degA], p)
		if err != nil {
			return nil, nil, nil, err
		}
		monicGcd := make([]*big.Int, degA+1)
		for i := 0; i <= degA; i++ {
			monicGcd[i] = new(big.Int).Mul(a[i], leadInv)
			monicGcd[i].Mod(monicGcd[i], p)
		}
		return monicGcd, []*big.Int{leadInv}, []*big.Int{big.NewInt(0)}, nil
	}

	q, r, err := polyDivRemMod(a, b, p)
	if err != nil {
		return nil, nil, nil, err
	}

	g, x1, y1, err := polyExtGCDMod(b, r, p)
	if err != nil {
		return nil, nil, nil, err
	}

	// s = y1, t = x1 - q * y1
	sRes := y1
	qy1 := polyMulMod(q, y1, p)
	tRes := polySubMod(x1, qy1, p)

	return g, sRes, tRes, nil
}

// HenselLiftFactors lifts a coprime factorization f = g0 * h0 mod p to f = g * h mod p^targetK
// where h is monic, deg(h) = deg(h0), and lc(g) = lc(f).
// Reference: Joachim von zur Gathen & Jürgen Gerhard "Modern Computer Algebra" Algorithm 15.10.
func HenselLiftFactors(fCoeffs, g0Coeffs, h0Coeffs []*big.Int, p *big.Int, targetK int) ([]*big.Int, []*big.Int, error) {
	if targetK < 1 {
		return nil, nil, fmt.Errorf("%s", i18n.T("hensel.err_lift_target_precision"))
	}

	f := trimPoly(fCoeffs)
	g := polyMod(g0Coeffs, p)
	h := polyMod(h0Coeffs, p)

	degF := polyDegree(f)
	if degF <= 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("hensel.err_leading_coeff_divisible", "deg <= 0", p.String()))
	}

	// Verify lc(f) mod p != 0
	lcF := new(big.Int).Mod(f[degF], p)
	if lcF.Sign() == 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("hensel.err_leading_coeff_divisible", f[degF].String(), p.String()))
	}

	// Initial Bezout coefficients: s * g + t * h = 1 mod p
	gcdPoly, s, t, err := polyExtGCDMod(g, h, p)
	if err != nil {
		return nil, nil, err
	}
	if polyDegree(gcdPoly) != 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("hensel.err_not_coprime"))
	}
	// Normalize if gcd is a constant != 1
	if gcdPoly[0].Cmp(big.NewInt(1)) != 0 {
		cInv, err := invModInt(gcdPoly[0], p)
		if err != nil {
			return nil, nil, err
		}
		s = polyMulMod(s, []*big.Int{cInv}, p)
		t = polyMulMod(t, []*big.Int{cInv}, p)
	}

	// Linear Hensel lifting step from p^k to p^(k+1)
	pPow := new(big.Int).Set(p)
	for k := 1; k < targetK; k++ {
		nextPPow := new(big.Int).Mul(pPow, p)

		// error polynomial e = (f - g * h) mod nextPPow
		gh := polyMulMod(g, h, nextPPow)
		diff := polySubMod(f, gh, nextPPow)

		// e / p^k mod p
		e := make([]*big.Int, len(diff))
		for i, c := range diff {
			div := new(big.Int).Div(c, pPow)
			div.Mod(div, p)
			if div.Sign() < 0 {
				div.Add(div, p)
			}
			e[i] = div
		}
		e = trimPoly(e)

		// s * e = q * h + tau mod p with deg(tau) < deg(h)
		se := polyMulMod(s, e, p)
		q, tau, err := polyDivRemMod(se, h, p)
		if err != nil {
			return nil, nil, err
		}

		// sigma = t * e + q * g mod p
		te := polyMulMod(t, e, p)
		qg := polyMulMod(q, g, p)
		sigma := polyAddMod(te, qg, p)

		// g = g + p^k * sigma mod nextPPow
		scaledSigma := make([]*big.Int, len(sigma))
		for i, c := range sigma {
			scaledSigma[i] = new(big.Int).Mul(c, pPow)
		}
		g = polyAddMod(g, scaledSigma, nextPPow)

		// h = h + p^k * tau mod nextPPow
		scaledTau := make([]*big.Int, len(tau))
		for i, c := range tau {
			scaledTau[i] = new(big.Int).Mul(c, pPow)
		}
		h = polyAddMod(h, scaledTau, nextPPow)

		pPow = nextPPow
	}

	return g, h, nil
}

// LandauMignotteBound computes the upper bound on the absolute value of the coefficients
// of any factor of polynomial f:
// B = binom(deg(f), factorDeg) * ||f||_2
// where ||f||_2 is the Euclidean norm.
func LandauMignotteBound(fCoeffs []*big.Int, factorDeg int) *big.Int {
	f := trimPoly(fCoeffs)
	n := polyDegree(f)
	if n <= 0 || factorDeg <= 0 || factorDeg >= n {
		return big.NewInt(1)
	}

	// 2-norm: sum of squares
	sumSquares := big.NewInt(0)
	for _, c := range f {
		if c != nil {
			sq := new(big.Int).Mul(c, c)
			sumSquares.Add(sumSquares, sq)
		}
	}
	// Integer square root ceiling
	norm2 := new(big.Int).Sqrt(sumSquares)
	norm2.Add(norm2, big.NewInt(1)) // safe upper bound

	// Binomial coefficient binom(n, factorDeg)
	binom := big.NewInt(1)
	k := factorDeg
	if n-factorDeg < k {
		k = n - factorDeg
	}
	for i := 1; i <= k; i++ {
		binom.Mul(binom, big.NewInt(int64(n-i+1)))
		binom.Div(binom, big.NewInt(int64(i)))
	}

	bound := new(big.Int).Mul(binom, norm2)
	return bound
}

// CenteredModulo converts a polynomial modulo m to the centered range [-m/2, m/2).
func CenteredModulo(poly []*big.Int, m *big.Int) []*big.Int {
	halfM := new(big.Int).Div(m, big.NewInt(2))
	res := make([]*big.Int, len(poly))
	for i, c := range poly {
		if c == nil {
			res[i] = big.NewInt(0)
			continue
		}
		val := new(big.Int).Mod(c, m)
		if val.Cmp(halfM) > 0 {
			val.Sub(val, m)
		}
		res[i] = val
	}
	return trimPoly(res)
}

// FactorHenselDegree2 attempts to factor a monic polynomial of degree >= 4 into two true factors in Z[x]
// using Hensel lifting over small prime fields. Returns (g, h, true) if a true non-trivial factor is found.
func FactorHenselDegree2(fCoeffs []*big.Int) ([]*big.Int, []*big.Int, bool) {
	f := trimPoly(fCoeffs)
	degF := polyDegree(f)
	if degF < 4 {
		return nil, nil, false
	}
	if f[degF].Cmp(big.NewInt(1)) != 0 {
		return nil, nil, false
	}

	bound := LandauMignotteBound(f, 2)
	twoBound := new(big.Int).Mul(bound, big.NewInt(2))

	primes := []int64{3, 5, 7, 11, 13, 17, 19, 23}
	for _, pInt := range primes {
		p := big.NewInt(pInt)

		// Calculate required k such that p^k > 2*Bound
		k := 1
		pPow := new(big.Int).Set(p)
		for pPow.Cmp(twoBound) <= 0 {
			pPow.Mul(pPow, p)
			k++
		}

		// Try quadratic monic factor g0 = x^2 + a*x + b mod p
		for b := int64(0); b < pInt; b++ {
			for a := int64(0); a < pInt; a++ {
				g0 := []*big.Int{big.NewInt(b), big.NewInt(a), big.NewInt(1)}
				q0, r0, err := polyDivRemMod(f, g0, p)
				if err != nil || polyDegree(r0) >= 0 || polyDegree(q0) < 2 {
					continue
				}

				// Check gcd(g0, q0) == 1 mod p
				gcdPoly, _, _, err := polyExtGCDMod(g0, q0, p)
				if err != nil || polyDegree(gcdPoly) != 0 {
					continue
				}

				// Lift factors to p^k
				g, h, err := HenselLiftFactors(f, g0, q0, p, k)
				if err != nil {
					continue
				}

				// Centered modulo
				gZ := CenteredModulo(g, pPow)
				hZ := CenteredModulo(h, pPow)

				// Verify in Z[x]
				prod := polyMulMod(gZ, hZ, nil)
				if polyDegree(prod) != degF {
					continue
				}
				match := true
				for i := 0; i <= degF; i++ {
					if prod[i].Cmp(f[i]) != 0 {
						match = false
						break
					}
				}
				if match {
					return gZ, hZ, true
				}
			}
		}
	}
	return nil, nil, false
}

