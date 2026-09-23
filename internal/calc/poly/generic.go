package poly

import (
	"sort"
	"strconv"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/domain"
)

// Term represents a single term in a sparse multivariate polynomial: Coeff * x1^e1 * x2^e2 * ...
type Term[E any] struct {
	Coeff     E
	Exponents []int
}

// Polynomial represents a canonical sparse multivariate polynomial over a coefficient domain.
// Invariant: Terms are strictly sorted in descending order according to Order,
// with no zero coefficients (according to Domain.IsZero) and no duplicate exponent vectors.
type Polynomial[E any] struct {
	Domain domain.Domain[E]
	Vars   []string
	Order  ast.MonomialOrder
	Terms  []Term[E]
}

// CompareExponents compares two exponent slices e1 and e2 according to the specified monomial order.
// Returns 1 if e1 > e2, -1 if e1 < e2, and 0 if e1 == e2.
func CompareExponents(e1, e2 []int, order ast.MonomialOrder) int {
	switch order {
	case ast.OrderGrevLex:
		deg1 := 0
		deg2 := 0
		for _, e := range e1 {
			deg1 += e
		}
		for _, e := range e2 {
			deg2 += e
		}
		if deg1 > deg2 {
			return 1
		}
		if deg1 < deg2 {
			return -1
		}
		// Right to left reverse comparison
		for i := len(e1) - 1; i >= 0; i-- {
			if e1[i] < e2[i] {
				return 1
			}
			if e1[i] > e2[i] {
				return -1
			}
		}
		return 0

	case ast.OrderLex:
		fallthrough
	default:
		for i := 0; i < len(e1) && i < len(e2); i++ {
			if e1[i] > e2[i] {
				return 1
			}
			if e1[i] < e2[i] {
				return -1
			}
		}
		return 0
	}
}

// AreExponentsEqual checks if two exponent slices are identical.
func AreExponentsEqual(e1, e2 []int) bool {
	if len(e1) != len(e2) {
		return false
	}
	for i, v := range e1 {
		if v != e2[i] {
			return false
		}
	}
	return true
}

// NewPolynomial creates a strictly canonical Polynomial from a list of terms.
// It combines duplicate terms, removes zero-coefficient terms, and sorts in descending order.
func NewPolynomial[E any](dom domain.Domain[E], vars []string, order ast.MonomialOrder, terms []Term[E]) *Polynomial[E] {
	vCopy := make([]string, len(vars))
	copy(vCopy, vars)

	if len(terms) == 0 {
		return &Polynomial[E]{
			Domain: dom,
			Vars:   vCopy,
			Order:  order,
			Terms:  nil,
		}
	}

	// Filter out zero-coefficient terms and deep-copy exponents
	validTerms := make([]Term[E], 0, len(terms))
	for _, t := range terms {
		if !dom.IsZero(t.Coeff) {
			expCopy := make([]int, len(vars))
			for i := 0; i < len(vars) && i < len(t.Exponents); i++ {
				expCopy[i] = t.Exponents[i]
			}
			validTerms = append(validTerms, Term[E]{
				Coeff:     dom.Clone(t.Coeff),
				Exponents: expCopy,
			})
		}
	}

	if len(validTerms) == 0 {
		return &Polynomial[E]{
			Domain: dom,
			Vars:   vCopy,
			Order:  order,
			Terms:  nil,
		}
	}

	// Sort terms in descending order
	sort.Slice(validTerms, func(i, j int) bool {
		return CompareExponents(validTerms[i].Exponents, validTerms[j].Exponents, order) > 0
	})

	// Combine like terms
	canonicalTerms := make([]Term[E], 0, len(validTerms))
	current := validTerms[0]

	for i := 1; i < len(validTerms); i++ {
		next := validTerms[i]
		if AreExponentsEqual(current.Exponents, next.Exponents) {
			current.Coeff = dom.Add(current.Coeff, next.Coeff)
		} else {
			if !dom.IsZero(current.Coeff) {
				canonicalTerms = append(canonicalTerms, current)
			}
			current = next
		}
	}
	if !dom.IsZero(current.Coeff) {
		canonicalTerms = append(canonicalTerms, current)
	}

	return &Polynomial[E]{
		Domain: dom,
		Vars:   vCopy,
		Order:  order,
		Terms:  canonicalTerms,
	}
}

// IsZero returns true if the polynomial is the zero polynomial.
func (p *Polynomial[E]) IsZero() bool {
	return p == nil || len(p.Terms) == 0
}

// Degree returns the maximum degree of the polynomial with respect to variable at varIndex.
// Returns -1 for zero polynomial.
func (p *Polynomial[E]) Degree(varIndex int) int {
	if p.IsZero() {
		return -1
	}
	maxDeg := -1
	for _, t := range p.Terms {
		if varIndex >= 0 && varIndex < len(t.Exponents) {
			if t.Exponents[varIndex] > maxDeg {
				maxDeg = t.Exponents[varIndex]
			}
		}
	}
	return maxDeg
}

// TotalDegree returns the maximum total degree (sum of exponents in any term).
// Returns -1 for zero polynomial.
func (p *Polynomial[E]) TotalDegree() int {
	if p.IsZero() {
		return -1
	}
	maxDeg := -1
	for _, t := range p.Terms {
		sum := 0
		for _, e := range t.Exponents {
			sum += e
		}
		if sum > maxDeg {
			maxDeg = sum
		}
	}
	return maxDeg
}

// LeadingCoefficient returns the coefficient of the leading term, or Domain.Zero() if zero polynomial.
func (p *Polynomial[E]) LeadingCoefficient() E {
	if p.IsZero() {
		return p.Domain.Zero()
	}
	return p.Domain.Clone(p.Terms[0].Coeff)
}

// Clone creates a deep copy of the polynomial.
func (p *Polynomial[E]) Clone() *Polynomial[E] {
	if p == nil {
		return nil
	}
	vCopy := make([]string, len(p.Vars))
	copy(vCopy, p.Vars)

	termsCopy := make([]Term[E], len(p.Terms))
	for i, t := range p.Terms {
		expCopy := make([]int, len(t.Exponents))
		copy(expCopy, t.Exponents)
		termsCopy[i] = Term[E]{
			Coeff:     p.Domain.Clone(t.Coeff),
			Exponents: expCopy,
		}
	}

	return &Polynomial[E]{
		Domain: p.Domain,
		Vars:   vCopy,
		Order:  p.Order,
		Terms:  termsCopy,
	}
}

// String returns a human-readable representation of the polynomial.
func (p *Polynomial[E]) String() string {
	if p.IsZero() {
		return "0"
	}

	var sb strings.Builder
	for i, t := range p.Terms {
		coeffStr := p.Domain.String(t.Coeff)

		// Check if monomial has variables
		isConst := true
		for _, e := range t.Exponents {
			if e > 0 {
				isConst = false
				break
			}
		}

		if i > 0 {
			sb.WriteString(" + ")
		}

		if isConst {
			sb.WriteString(coeffStr)
			continue
		}

		isOne := p.Domain.IsOne(t.Coeff)
		if !isOne {
			// If coeff contains space or signs, wrap in parentheses
			if strings.ContainsAny(coeffStr, " +-") {
				sb.WriteString("(")
				sb.WriteString(coeffStr)
				sb.WriteString(")*")
			} else {
				sb.WriteString(coeffStr)
				sb.WriteString("*")
			}
		}

		var parts []string
		for vIdx, e := range t.Exponents {
			if e == 0 {
				continue
			}
			vName := "?"
			if vIdx < len(p.Vars) {
				vName = p.Vars[vIdx]
			}
			if e == 1 {
				parts = append(parts, vName)
			} else {
				parts = append(parts, vName+"^"+strconv.Itoa(e))
			}
		}
		sb.WriteString(strings.Join(parts, "*"))
	}

	return sb.String()
}
