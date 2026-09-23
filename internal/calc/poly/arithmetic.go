package poly

import (
	"errors"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// areVariablesCompatible checks if two polynomials have matching variables and ordering.
func areVariablesCompatible[E any](p, q *Polynomial[E]) bool {
	if p.Order != q.Order {
		return false
	}
	if len(p.Vars) != len(q.Vars) {
		return false
	}
	for i, v := range p.Vars {
		if v != q.Vars[i] {
			return false
		}
	}
	return true
}

// Neg returns the negation -p.
func (p *Polynomial[E]) Neg() *Polynomial[E] {
	if p.IsZero() {
		return p.Clone()
	}
	resTerms := make([]Term[E], len(p.Terms))
	for i, t := range p.Terms {
		expCopy := make([]int, len(t.Exponents))
		copy(expCopy, t.Exponents)
		resTerms[i] = Term[E]{
			Coeff:     p.Domain.Neg(t.Coeff),
			Exponents: expCopy,
		}
	}
	return &Polynomial[E]{
		Domain: p.Domain,
		Vars:   p.Vars,
		Order:  p.Order,
		Terms:  resTerms,
	}
}

// ScalarMul multiplies every term of p by scalar s.
func (p *Polynomial[E]) ScalarMul(s E) *Polynomial[E] {
	if p.IsZero() || p.Domain.IsZero(s) {
		return NewPolynomial(p.Domain, p.Vars, p.Order, nil)
	}
	if p.Domain.IsOne(s) {
		return p.Clone()
	}

	resTerms := make([]Term[E], 0, len(p.Terms))
	for _, t := range p.Terms {
		c := p.Domain.Mul(t.Coeff, s)
		if !p.Domain.IsZero(c) {
			expCopy := make([]int, len(t.Exponents))
			copy(expCopy, t.Exponents)
			resTerms = append(resTerms, Term[E]{
				Coeff:     c,
				Exponents: expCopy,
			})
		}
	}
	return &Polynomial[E]{
		Domain: p.Domain,
		Vars:   p.Vars,
		Order:  p.Order,
		Terms:  resTerms,
	}
}

// Add returns p + q. Both polynomials must share the same variables and order.
func (p *Polynomial[E]) Add(q *Polynomial[E]) (*Polynomial[E], error) {
	if p.IsZero() {
		return q.Clone(), nil
	}
	if q.IsZero() {
		return p.Clone(), nil
	}
	if !areVariablesCompatible(p, q) {
		return nil, errors.New(i18n.T("domain.err_variable_mismatch"))
	}

	dom := p.Domain
	merged := make([]Term[E], 0, len(p.Terms)+len(q.Terms))
	i, j := 0, 0

	for i < len(p.Terms) && j < len(q.Terms) {
		cmp := CompareExponents(p.Terms[i].Exponents, q.Terms[j].Exponents, p.Order)
		if cmp > 0 {
			expCopy := make([]int, len(p.Terms[i].Exponents))
			copy(expCopy, p.Terms[i].Exponents)
			merged = append(merged, Term[E]{
				Coeff:     dom.Clone(p.Terms[i].Coeff),
				Exponents: expCopy,
			})
			i++
		} else if cmp < 0 {
			expCopy := make([]int, len(q.Terms[j].Exponents))
			copy(expCopy, q.Terms[j].Exponents)
			merged = append(merged, Term[E]{
				Coeff:     dom.Clone(q.Terms[j].Coeff),
				Exponents: expCopy,
			})
			j++
		} else {
			// Same monomial: add coefficients
			sumCoeff := dom.Add(p.Terms[i].Coeff, q.Terms[j].Coeff)
			if !dom.IsZero(sumCoeff) {
				expCopy := make([]int, len(p.Terms[i].Exponents))
				copy(expCopy, p.Terms[i].Exponents)
				merged = append(merged, Term[E]{
					Coeff:     sumCoeff,
					Exponents: expCopy,
				})
			}
			i++
			j++
		}
	}

	for i < len(p.Terms) {
		expCopy := make([]int, len(p.Terms[i].Exponents))
		copy(expCopy, p.Terms[i].Exponents)
		merged = append(merged, Term[E]{
			Coeff:     dom.Clone(p.Terms[i].Coeff),
			Exponents: expCopy,
		})
		i++
	}

	for j < len(q.Terms) {
		expCopy := make([]int, len(q.Terms[j].Exponents))
		copy(expCopy, q.Terms[j].Exponents)
		merged = append(merged, Term[E]{
			Coeff:     dom.Clone(q.Terms[j].Coeff),
			Exponents: expCopy,
		})
		j++
	}

	return &Polynomial[E]{
		Domain: dom,
		Vars:   p.Vars,
		Order:  p.Order,
		Terms:  merged,
	}, nil
}

// Sub returns p - q.
func (p *Polynomial[E]) Sub(q *Polynomial[E]) (*Polynomial[E], error) {
	return p.Add(q.Neg())
}

// Mul returns p * q.
func (p *Polynomial[E]) Mul(q *Polynomial[E]) (*Polynomial[E], error) {
	if p.IsZero() || q.IsZero() {
		return NewPolynomial(p.Domain, p.Vars, p.Order, nil), nil
	}
	if !areVariablesCompatible(p, q) {
		return nil, errors.New(i18n.T("domain.err_variable_mismatch"))
	}

	dom := p.Domain
	rawTerms := make([]Term[E], 0, len(p.Terms)*len(q.Terms))

	for _, pt := range p.Terms {
		for _, qt := range q.Terms {
			coeff := dom.Mul(pt.Coeff, qt.Coeff)
			if dom.IsZero(coeff) {
				continue
			}
			exp := make([]int, len(pt.Exponents))
			for k := range exp {
				exp[k] = pt.Exponents[k] + qt.Exponents[k]
			}
			rawTerms = append(rawTerms, Term[E]{
				Coeff:     coeff,
				Exponents: exp,
			})
		}
	}

	return NewPolynomial(dom, p.Vars, p.Order, rawTerms), nil
}

// Pow computes p^n for non-negative integer n using binary exponentiation.
func (p *Polynomial[E]) Pow(n int) (*Polynomial[E], error) {
	if n < 0 {
		return nil, errors.New(i18n.T("domain.err_modulus_positive"))
	}

	dom := p.Domain
	if n == 0 {
		// Constant 1 polynomial
		constOne := Term[E]{
			Coeff:     dom.One(),
			Exponents: make([]int, len(p.Vars)),
		}
		return NewPolynomial(dom, p.Vars, p.Order, []Term[E]{constOne}), nil
	}

	result := NewPolynomial(dom, p.Vars, p.Order, []Term[E]{{
		Coeff:     dom.One(),
		Exponents: make([]int, len(p.Vars)),
	}})
	base := p.Clone()

	for n > 0 {
		if n%2 == 1 {
			var err error
			result, err = result.Mul(base)
			if err != nil {
				return nil, err
			}
		}
		if n > 1 {
			var err error
			base, err = base.Mul(base)
			if err != nil {
				return nil, err
			}
		}
		n /= 2
	}

	return result, nil
}
