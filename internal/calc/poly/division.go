package poly

import (
	"errors"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/domain"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// Monic converts a non-zero polynomial over a field into a monic polynomial (leading coefficient = 1).
func (p *Polynomial[E]) Monic() (*Polynomial[E], error) {
	if p.IsZero() {
		return p.Clone(), nil
	}
	fld, ok := p.Domain.(domain.FieldDomain[E])
	if !ok {
		return nil, errors.New(i18n.T("domain.err_no_inverse", p.Domain.Name()))
	}

	lc := p.LeadingCoefficient()
	invLC, err := fld.Inv(lc)
	if err != nil {
		return nil, err
	}

	return p.ScalarMul(invLC), nil
}

// DivRem performs univariate Euclidean polynomial division f = q * g + r over a field domain.
// varIndex specifies the variable of division (default: 0 for univariate).
// Requires that deg(r, varIndex) < deg(g, varIndex).
func DivRem[E any](f, g *Polynomial[E], varIndex int) (q, r *Polynomial[E], err error) {
	if g.IsZero() {
		return nil, nil, errors.New(i18n.T("domain.err_poly_zero_division"))
	}
	if !areVariablesCompatible(f, g) {
		return nil, nil, errors.New(i18n.T("domain.err_variable_mismatch"))
	}

	fld, ok := f.Domain.(domain.FieldDomain[E])
	if !ok {
		return nil, nil, errors.New(i18n.T("domain.err_no_inverse", f.Domain.Name()))
	}

	dom := f.Domain
	quotTerms := make([]Term[E], 0)
	rem := f.Clone()

	degG := g.Degree(varIndex)
	lcG := g.LeadingCoefficient()
	invLcG, err := fld.Inv(lcG)
	if err != nil {
		return nil, nil, err
	}

	for !rem.IsZero() && rem.Degree(varIndex) >= degG {
		ltRem := rem.Terms[0]
		degDiff := ltRem.Exponents[varIndex] - g.Terms[0].Exponents[varIndex]
		if degDiff < 0 {
			break
		}

		// Check if other exponents are >= g's exponents
		canDivide := true
		newExp := make([]int, len(ltRem.Exponents))
		for k := range newExp {
			newExp[k] = ltRem.Exponents[k] - g.Terms[0].Exponents[k]
			if newExp[k] < 0 {
				canDivide = false
				break
			}
		}
		if !canDivide {
			break
		}

		coeff := dom.Mul(ltRem.Coeff, invLcG)
		term := Term[E]{
			Coeff:     coeff,
			Exponents: newExp,
		}
		quotTerms = append(quotTerms, term)

		// Subtract term * g from rem
		termPoly := NewPolynomial(dom, f.Vars, f.Order, []Term[E]{term})
		prod, mulErr := termPoly.Mul(g)
		if mulErr != nil {
			return nil, nil, mulErr
		}
		rem, mulErr = rem.Sub(prod)
		if mulErr != nil {
			return nil, nil, mulErr
		}
	}

	quot := NewPolynomial(dom, f.Vars, f.Order, quotTerms)
	return quot, rem, nil
}

// PseudoDivRem performs pseudo-division in R[x] over an integral domain:
// lc(g)^(deg(f) - deg(g) + 1) * f = q * g + r where deg(r) < deg(g).
// This guarantees exact quotient and remainder without requiring division in the coefficient domain.
func PseudoDivRem[E any](f, g *Polynomial[E], varIndex int) (q, r *Polynomial[E], delta int, err error) {
	if g.IsZero() {
		return nil, nil, 0, errors.New(i18n.T("domain.err_poly_zero_division"))
	}
	if !areVariablesCompatible(f, g) {
		return nil, nil, 0, errors.New(i18n.T("domain.err_variable_mismatch"))
	}

	degF := f.Degree(varIndex)
	degG := g.Degree(varIndex)

	if degF < degG {
		return NewPolynomial(f.Domain, f.Vars, f.Order, nil), f.Clone(), 0, nil
	}

	dom := f.Domain
	delta = degF - degG + 1
	lcG := g.LeadingCoefficient()

	rem := f.Clone()
	quot := NewPolynomial(dom, f.Vars, f.Order, nil)

	for iter := 0; iter < delta && !rem.IsZero() && rem.Degree(varIndex) >= degG; iter++ {
		newExp := make([]int, len(rem.Terms[0].Exponents))
		for k := range newExp {
			newExp[k] = rem.Terms[0].Exponents[k] - g.Terms[0].Exponents[k]
		}

		t := Term[E]{
			Coeff:     rem.Terms[0].Coeff,
			Exponents: newExp,
		}

		// rem = lcG * rem - t * g
		scaledRem := rem.ScalarMul(lcG)
		termPoly := NewPolynomial(dom, f.Vars, f.Order, []Term[E]{t})
		prod, err := termPoly.Mul(g)
		if err != nil {
			return nil, nil, 0, err
		}
		rem, err = scaledRem.Sub(prod)
		if err != nil {
			return nil, nil, 0, err
		}

		// quot = lcG * quot + t
		scaledQuot := quot.ScalarMul(lcG)
		addQuot, err := scaledQuot.Add(termPoly)
		if err != nil {
			return nil, nil, 0, err
		}
		quot = addQuot
	}

	return quot, rem, delta, nil
}
