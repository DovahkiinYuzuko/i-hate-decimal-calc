package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// -------------------------------------------------------------------------
// Polynomial Division & Extended Euclidean Algorithm in Q[x]
// -------------------------------------------------------------------------

// MonicPoly normalizes a polynomial by scaling so its leading coefficient is 1.
func MonicPoly(p *ast.PolyNode) *ast.PolyNode {
	if p == nil || len(p.Terms) == 0 {
		return p
	}
	lc := p.Terms[0].Coeff
	if lc.Cmp(big.NewRat(1, 1)) == 0 {
		return p.Clone()
	}
	invLC := new(big.Rat).Inv(lc)
	scaleMon := ast.Monomial{Coeff: invLC, Exponents: make([]int, len(p.Vars))}
	scalePoly := NewPolyNode(p.Vars, p.Order, []ast.Monomial{scaleMon})
	return MulPoly(p, scalePoly)
}

// PolyDivRem computes quotient and remainder such that dividend = quotient * divisor + remainder,
// where deg(remainder) < deg(divisor) with respect to mainVar in Q[x].
func PolyDivRem(dividend, divisor *ast.PolyNode, mainVar string) (quotient, remainder *ast.PolyNode, err error) {
	if divisor == nil || len(divisor.Terms) == 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("algebra.err_polynomial_division_by_zero"))
	}
	degDivisor := DegreeInVar(divisor, mainVar)
	if degDivisor < 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("algebra.err_polynomial_division_by_zero"))
	}
	degDividend := DegreeInVar(dividend, mainVar)
	if degDividend < degDivisor {
		zeroQ := NewPolyNode(dividend.Vars, dividend.Order, nil)
		return zeroQ, dividend.Clone(), nil
	}

	lcDivisorNode := LeadingCoeffInVar(divisor, mainVar)
	if len(lcDivisorNode.Terms) == 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("algebra.err_divisor_leading_coefficient_is_zero"))
	}
	scalarLC := lcDivisorNode.Terms[0].Coeff

	varIdx := -1
	for idx, v := range dividend.Vars {
		if v == mainVar {
			varIdx = idx
			break
		}
	}

	rem := dividend.Clone()
	quo := NewPolyNode(dividend.Vars, dividend.Order, nil)

	for len(rem.Terms) > 0 && DegreeInVar(rem, mainVar) >= degDivisor {
		remDeg := DegreeInVar(rem, mainVar)
		lcRemNode := LeadingCoeffInVar(rem, mainVar)
		if len(lcRemNode.Terms) == 0 {
			break
		}
		lcRem := lcRemNode.Terms[0].Coeff

		// qCoeff = lcRem / scalarLC
		qCoeff := new(big.Rat).Quo(lcRem, scalarLC)
		qExp := make([]int, len(dividend.Vars))
		if varIdx >= 0 {
			qExp[varIdx] = remDeg - degDivisor
		}

		termPoly := NewPolyNode(dividend.Vars, dividend.Order, []ast.Monomial{
			{Coeff: qCoeff, Exponents: qExp},
		})

		quo = AddPoly(quo, termPoly)
		subPart := MulPoly(termPoly, divisor)
		rem = SubPoly(rem, subPart)
	}

	return quo, rem, nil
}

type polyExtendedGCDResult struct {
	gcd *ast.PolyNode
	u   *ast.PolyNode
	v   *ast.PolyNode
}

// canonicalPolyExtendedGCD computes u*a + v*b = gcd in Q[x] where gcd is monic.
func canonicalPolyExtendedGCD(a, b *ast.PolyNode, mainVar string) (*polyExtendedGCDResult, error) {
	vars := a.Vars
	order := a.Order

	r0 := a.Clone()
	r1 := b.Clone()

	s0 := NewPolyNode(vars, order, []ast.Monomial{{Coeff: big.NewRat(1, 1), Exponents: make([]int, len(vars))}})
	s1 := NewPolyNode(vars, order, nil)

	t0 := NewPolyNode(vars, order, nil)
	t1 := NewPolyNode(vars, order, []ast.Monomial{{Coeff: big.NewRat(1, 1), Exponents: make([]int, len(vars))}})

	for len(r1.Terms) > 0 {
		q, rem, err := PolyDivRem(r0, r1, mainVar)
		if err != nil {
			return nil, err
		}
		r0 = r1
		r1 = rem

		nextS := SubPoly(s0, MulPoly(q, s1))
		s0 = s1
		s1 = nextS

		nextT := SubPoly(t0, MulPoly(q, t1))
		t0 = t1
		t1 = nextT
	}

	// Monicize r0 and scale s0, t0 accordingly
	if len(r0.Terms) > 0 {
		lc := r0.Terms[0].Coeff
		invLC := new(big.Rat).Inv(lc)
		scaleMon := ast.Monomial{Coeff: invLC, Exponents: make([]int, len(vars))}
		scalePoly := NewPolyNode(vars, order, []ast.Monomial{scaleMon})

		r0 = MulPoly(r0, scalePoly)
		s0 = MulPoly(s0, scalePoly)
		t0 = MulPoly(t0, scalePoly)
	}

	return &polyExtendedGCDResult{gcd: r0, u: s0, v: t0}, nil
}

// -------------------------------------------------------------------------
// Algebraic Number Node Operations
// -------------------------------------------------------------------------

// NewAlgebraicNumber constructs a canonical AlgebraicNumberNode in Q(alpha) modulo minPoly(alpha) = 0.
// Invariant deg(repPoly) < deg(minPoly) is guaranteed by polynomial reduction.
func NewAlgebraicNumber(minPoly, repPoly *ast.PolyNode, symbol string) (*ast.AlgebraicNumberNode, error) {
	if minPoly == nil || len(minPoly.Terms) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_minpoly_cannot_be_nil_or"))
	}
	if symbol == "" {
		symbol = "alpha"
	}
	monicMin := MonicPoly(minPoly)
	if len(monicMin.Vars) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_minpoly_must_have_at_least"))
	}
	mainVar := monicMin.Vars[0]

	var normRep *ast.PolyNode
	if repPoly == nil || len(repPoly.Terms) == 0 {
		normRep = NewPolyNode(monicMin.Vars, monicMin.Order, nil)
	} else {
		// Align vars if necessary
		targetRep := repPoly
		if len(targetRep.Vars) != len(monicMin.Vars) || targetRep.Vars[0] != mainVar {
			targetRep = NewPolyNode(monicMin.Vars, monicMin.Order, repPoly.Terms)
		}
		_, rem, err := PolyDivRem(targetRep, monicMin, mainVar)
		if err != nil {
			return nil, fmt.Errorf("%s", i18n.T("algebra.err_reduction_modulo_minimal_polynomial_failed", err))
		}
		normRep = rem
	}

	return &ast.AlgebraicNumberNode{
		MinPoly: monicMin,
		RepPoly: normRep,
		Symbol:  symbol,
	}, nil
}

// IsZeroAlg returns true if the algebraic number is mathematically zero.
func IsZeroAlg(a *ast.AlgebraicNumberNode) bool {
	if a == nil || a.RepPoly == nil || len(a.RepPoly.Terms) == 0 {
		return true
	}
	return false
}

// checkSameField verifies that two algebraic numbers belong to the same algebraic number field.
func checkSameField(a, b *ast.AlgebraicNumberNode) error {
	if a == nil || b == nil {
		return fmt.Errorf("%s", i18n.T("algebra.err_nil_algebraic_number_operand"))
	}
	if !a.MinPoly.Equal(b.MinPoly) {
		return fmt.Errorf("%s", i18n.T("algebra.err_algebraic_operations_across_different_number", a.MinPoly.String(), b.MinPoly.String()))
	}
	return nil
}

// AddAlg adds two algebraic numbers belonging to the same field.
func AddAlg(a, b *ast.AlgebraicNumberNode) (*ast.AlgebraicNumberNode, error) {
	if err := checkSameField(a, b); err != nil {
		return nil, err
	}
	rep := AddPoly(a.RepPoly, b.RepPoly)
	return NewAlgebraicNumber(a.MinPoly, rep, a.Symbol)
}

// SubAlg subtracts algebraic number b from a (a - b).
func SubAlg(a, b *ast.AlgebraicNumberNode) (*ast.AlgebraicNumberNode, error) {
	if err := checkSameField(a, b); err != nil {
		return nil, err
	}
	rep := SubPoly(a.RepPoly, b.RepPoly)
	return NewAlgebraicNumber(a.MinPoly, rep, a.Symbol)
}

// MulAlg multiplies two algebraic numbers belonging to the same field.
func MulAlg(a, b *ast.AlgebraicNumberNode) (*ast.AlgebraicNumberNode, error) {
	if err := checkSameField(a, b); err != nil {
		return nil, err
	}
	prod := MulPoly(a.RepPoly, b.RepPoly)
	return NewAlgebraicNumber(a.MinPoly, prod, a.Symbol)
}

// InvAlg computes the multiplicative inverse a(alpha)^(-1) in Q(alpha) using extended Euclidean algorithm.
func InvAlg(a *ast.AlgebraicNumberNode) (*ast.AlgebraicNumberNode, error) {
	if a == nil {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_nil_algebraic_number"))
	}
	if IsZeroAlg(a) {
		return nil, NewZeroDivisionError("division by zero in algebraic number field")
	}

	mainVar := a.MinPoly.Vars[0]
	res, err := canonicalPolyExtendedGCD(a.RepPoly, a.MinPoly, mainVar)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_extended_gcd_failed", err))
	}

	// In an algebraic field, gcd(rep, minPoly) must be 1 (degree 0) since minPoly is irreducible
	if len(res.gcd.Terms) == 0 || DegreeInVar(res.gcd, mainVar) != 0 {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_element_is_not_invertible_modulo", a.MinPoly.String()))
	}

	return NewAlgebraicNumber(a.MinPoly, res.u, a.Symbol)
}

// MinPolySum computes the minimal/elimination polynomial of alpha + beta where m1(alpha) = 0 and m2(beta) = 0,
// using Sylvester resultant Res_x(m1(y - x), m2(x)).
func MinPolySum(m1, m2 *ast.PolyNode, varName string) (*ast.PolyNode, error) {
	if m1 == nil || len(m1.Terms) == 0 || m2 == nil || len(m2.Terms) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_minimal_polynomials_cannot_be_nil"))
	}
	if varName == "" {
		varName = "y"
	}

	m1Node := PolyToNode(m1)
	m2Node := PolyToNode(m2)

	xVar := "x"
	if varName == "x" {
		xVar = "_x"
	}

	// Substitute var in m1 with (varName - xVar)
	substM1 := ast.Substitute(m1Node, m1.Vars[0], &ast.AddNode{
		Terms: []ast.Node{
			ast.NewVar(varName),
			&ast.UnaryOpNode{Op: "-", Expr: ast.NewVar(xVar)},
		},
	})

	// Substitute var in m2 with xVar
	substM2 := ast.Substitute(m2Node, m2.Vars[0], ast.NewVar(xVar))

	// Compute Sylvester Resultant with respect to xVar
	env := NewEnv()
	resNode, err := EvalResultant(substM1, substM2, xVar, env)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_resultant_elimination_failed", err))
	}

	// Convert resultant back to PolyNode in varName
	resPoly, err := NodeToPoly(resNode, []string{varName}, ast.OrderLex)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("algebra.err_conversion_of_elimination_polynomial_failed", err))
	}

	return MonicPoly(resPoly), nil
}
