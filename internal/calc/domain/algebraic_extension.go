package domain

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// AlgExtensionDomain represents an algebraic number field Q(alpha) = Q[x] / <m(x)>.
// Elements are represented in canonical dense form as []*big.Rat (coeffs of 1, alpha, alpha^2, ...).
// Invariant: len(element) <= deg(m), trimmed so that the last element is non-zero (or empty for 0).
type AlgExtensionDomain struct {
	symbol string      // e.g., "alpha" or "sqrt2"
	minPoly []*big.Rat // Monic minimal polynomial: m[0] + m[1]*x + ... + x^d
	deg     int        // deg(m) >= 1
}

// NewAlgExtensionDomain creates a new Q(alpha) domain given generator symbol and monic minimal polynomial coeffs.
// minPoly must have deg >= 1 and leading coefficient 1.
func NewAlgExtensionDomain(symbol string, minPoly []*big.Rat) (*AlgExtensionDomain, error) {
	if len(minPoly) < 2 {
		return nil, errors.New(i18n.T("domain.err_modulus_positive"))
	}
	// Trim trailing zeros
	d := len(minPoly) - 1
	for d >= 0 && (minPoly[d] == nil || minPoly[d].Sign() == 0) {
		d--
	}
	if d < 1 {
		return nil, errors.New(i18n.T("domain.err_modulus_positive"))
	}

	// Verify monic (leading coefficient == 1)
	lc := minPoly[d]
	if lc.Cmp(big.NewRat(1, 1)) != 0 {
		return nil, errors.New(i18n.T("domain.err_monic_required"))
	}

	polyCopy := make([]*big.Rat, d+1)
	for i := 0; i <= d; i++ {
		if minPoly[i] == nil {
			polyCopy[i] = big.NewRat(0, 1)
		} else {
			polyCopy[i] = new(big.Rat).Set(minPoly[i])
		}
	}

	return &AlgExtensionDomain{
		symbol:  symbol,
		minPoly: polyCopy,
		deg:     d,
	}, nil
}

func (d *AlgExtensionDomain) Name() string {
	return fmt.Sprintf("Q(%s)", d.symbol)
}

func (d *AlgExtensionDomain) Zero() []*big.Rat {
	return nil
}

func (d *AlgExtensionDomain) One() []*big.Rat {
	return []*big.Rat{big.NewRat(1, 1)}
}

// trim removes trailing zero coefficients.
func trimDense(a []*big.Rat) []*big.Rat {
	n := len(a)
	for n > 0 && (a[n-1] == nil || a[n-1].Sign() == 0) {
		n--
	}
	if n == 0 {
		return nil
	}
	return a[:n]
}

func (d *AlgExtensionDomain) reduce(a []*big.Rat) []*big.Rat {
	a = trimDense(a)
	if len(a) <= d.deg {
		return a
	}

	// Polynomial division by minPoly
	rem := make([]*big.Rat, len(a))
	for i, c := range a {
		if c == nil {
			rem[i] = big.NewRat(0, 1)
		} else {
			rem[i] = new(big.Rat).Set(c)
		}
	}

	mDeg := d.deg
	for len(rem) > mDeg {
		curDeg := len(rem) - 1
		lcRem := rem[curDeg]
		if lcRem.Sign() != 0 {
			diff := curDeg - mDeg
			for i := 0; i <= mDeg; i++ {
				sub := new(big.Rat).Mul(lcRem, d.minPoly[i])
				rem[i+diff].Sub(rem[i+diff], sub)
			}
		}
		rem = trimDense(rem)
	}

	return rem
}

func (d *AlgExtensionDomain) IsZero(a []*big.Rat) bool {
	norm := d.reduce(a)
	return len(norm) == 0
}

func (d *AlgExtensionDomain) IsOne(a []*big.Rat) bool {
	norm := d.reduce(a)
	if len(norm) != 1 {
		return false
	}
	return norm[0].Cmp(big.NewRat(1, 1)) == 0
}

func (d *AlgExtensionDomain) Equals(a, b []*big.Rat) bool {
	diff := d.Sub(a, b)
	return d.IsZero(diff)
}

func (d *AlgExtensionDomain) Add(a, b []*big.Rat) []*big.Rat {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	res := make([]*big.Rat, maxLen)
	for i := 0; i < maxLen; i++ {
		res[i] = big.NewRat(0, 1)
		if i < len(a) && a[i] != nil {
			res[i].Add(res[i], a[i])
		}
		if i < len(b) && b[i] != nil {
			res[i].Add(res[i], b[i])
		}
	}
	return d.reduce(res)
}

func (d *AlgExtensionDomain) Sub(a, b []*big.Rat) []*big.Rat {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	res := make([]*big.Rat, maxLen)
	for i := 0; i < maxLen; i++ {
		res[i] = big.NewRat(0, 1)
		if i < len(a) && a[i] != nil {
			res[i].Add(res[i], a[i])
		}
		if i < len(b) && b[i] != nil {
			res[i].Sub(res[i], b[i])
		}
	}
	return d.reduce(res)
}

func (d *AlgExtensionDomain) Mul(a, b []*big.Rat) []*big.Rat {
	a = trimDense(a)
	b = trimDense(b)
	if len(a) == 0 || len(b) == 0 {
		return nil
	}

	prod := make([]*big.Rat, len(a)+len(b)-1)
	for i := range prod {
		prod[i] = big.NewRat(0, 1)
	}

	for i, ca := range a {
		if ca == nil || ca.Sign() == 0 {
			continue
		}
		for j, cb := range b {
			if cb == nil || cb.Sign() == 0 {
				continue
			}
			term := new(big.Rat).Mul(ca, cb)
			prod[i+j].Add(prod[i+j], term)
		}
	}

	return d.reduce(prod)
}

func (d *AlgExtensionDomain) Neg(a []*big.Rat) []*big.Rat {
	a = trimDense(a)
	if len(a) == 0 {
		return nil
	}
	res := make([]*big.Rat, len(a))
	for i, c := range a {
		if c == nil {
			res[i] = big.NewRat(0, 1)
		} else {
			res[i] = new(big.Rat).Neg(c)
		}
	}
	return res
}

// polyDivDense performs univariate polynomial division over Q: a = q*b + r.
func polyDivDense(a, b []*big.Rat) (q, r []*big.Rat) {
	a = trimDense(a)
	b = trimDense(b)
	if len(b) == 0 {
		return nil, nil
	}
	if len(a) < len(b) {
		return nil, a
	}

	degB := len(b) - 1
	lcB := b[degB]

	rem := make([]*big.Rat, len(a))
	for i, c := range a {
		rem[i] = new(big.Rat).Set(c)
	}

	degQ := len(a) - degB
	quot := make([]*big.Rat, degQ+1)
	for i := range quot {
		quot[i] = big.NewRat(0, 1)
	}

	for len(rem) >= len(b) {
		curDeg := len(rem) - 1
		degDiff := curDeg - degB
		lcRem := rem[curDeg]

		coeff := new(big.Rat).Quo(lcRem, lcB)
		quot[degDiff] = coeff

		for i := 0; i <= degB; i++ {
			sub := new(big.Rat).Mul(coeff, b[i])
			rem[i+degDiff].Sub(rem[i+degDiff], sub)
		}
		rem = trimDense(rem)
	}

	return trimDense(quot), trimDense(rem)
}

// Inv computes the multiplicative inverse a^(-1) mod minPoly using Extended Euclidean Algorithm.
func (d *AlgExtensionDomain) Inv(a []*big.Rat) ([]*big.Rat, error) {
	a = d.reduce(a)
	if len(a) == 0 {
		return nil, errors.New(i18n.T("domain.err_zero_division"))
	}

	// Extended Euclidean Algorithm for polynomials:
	// s*a + t*minPoly = gcd (which is 1 because minPoly is irreducible).
	// Then s mod minPoly is the inverse.
	r0 := d.minPoly
	r1 := a
	s0 := []*big.Rat{big.NewRat(0, 1)}
	s1 := []*big.Rat{big.NewRat(1, 1)}

	for len(r1) > 0 {
		q, rem := polyDivDense(r0, r1)
		r0 = r1
		r1 = rem

		// nextS = s0 - q * s1
		qTimesS1 := make([]*big.Rat, 0)
		if len(q) > 0 && len(s1) > 0 {
			qTimesS1 = make([]*big.Rat, len(q)+len(s1)-1)
			for i := range qTimesS1 {
				qTimesS1[i] = big.NewRat(0, 1)
			}
			for i, cq := range q {
				for j, cs := range s1 {
					term := new(big.Rat).Mul(cq, cs)
					qTimesS1[i+j].Add(qTimesS1[i+j], term)
				}
			}
		}

		// s0 - qTimesS1
		maxLen := len(s0)
		if len(qTimesS1) > maxLen {
			maxLen = len(qTimesS1)
		}
		newS := make([]*big.Rat, maxLen)
		for i := 0; i < maxLen; i++ {
			newS[i] = big.NewRat(0, 1)
			if i < len(s0) {
				newS[i].Add(newS[i], s0[i])
			}
			if i < len(qTimesS1) {
				newS[i].Sub(newS[i], qTimesS1[i])
			}
		}

		s0 = s1
		s1 = trimDense(newS)
	}

	// r0 is gcd (degree 0 constant)
	if len(r0) == 0 || r0[0].Sign() == 0 {
		return nil, errors.New(i18n.T("domain.err_no_inverse", d.String(a)))
	}

	// Normalize s0 by dividing by r0[0]
	invConst := new(big.Rat).Inv(r0[0])
	for i := range s0 {
		s0[i].Mul(s0[i], invConst)
	}

	return d.reduce(s0), nil
}

func (d *AlgExtensionDomain) Div(a, b []*big.Rat) ([]*big.Rat, error) {
	invB, err := d.Inv(b)
	if err != nil {
		return nil, err
	}
	return d.Mul(a, invB), nil
}

func (d *AlgExtensionDomain) String(a []*big.Rat) string {
	a = d.reduce(a)
	if len(a) == 0 {
		return "0"
	}
	var sb strings.Builder
	for i, c := range a {
		if c.Sign() == 0 {
			continue
		}
		if sb.Len() > 0 && c.Sign() > 0 {
			sb.WriteString(" + ")
		} else if sb.Len() > 0 && c.Sign() < 0 {
			sb.WriteString(" - ")
		} else if c.Sign() < 0 {
			sb.WriteString("-")
		}

		absVal := new(big.Rat).Abs(c)
		isOne := absVal.Cmp(big.NewRat(1, 1)) == 0

		if i == 0 {
			sb.WriteString(absVal.RatString())
		} else {
			if !isOne {
				sb.WriteString(absVal.RatString())
				sb.WriteString("*")
			}
			if i == 1 {
				sb.WriteString(d.symbol)
			} else {
				sb.WriteString(fmt.Sprintf("%s^%d", d.symbol, i))
			}
		}
	}
	return sb.String()
}

func (d *AlgExtensionDomain) Clone(a []*big.Rat) []*big.Rat {
	a = trimDense(a)
	if len(a) == 0 {
		return nil
	}
	res := make([]*big.Rat, len(a))
	for i, c := range a {
		res[i] = new(big.Rat).Set(c)
	}
	return res
}
