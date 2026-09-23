package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
)

// -------------------------------------------------------------------------
// Sylvester Resultant & Algebraic Elimination
// (Sylvester 1840, Cox, Little, O'Shea "Using Algebraic Geometry" Chapter 3)
// -------------------------------------------------------------------------

// EvalResultant computes the Sylvester resultant of polynomials p and q with respect to varName.
// If varName is empty, the main variable is automatically selected.
func EvalResultant(p, q Node, varName string, env *Env) (Node, error) {
	if p == nil || q == nil {
		return nil, fmt.Errorf("%s", i18n.T("resultant.err_resultant_nil_argument"))
	}

	evalP, err := Eval(expandNode(p))
	if err != nil {
		evalP = expandNode(p)
	}
	evalQ, err := Eval(expandNode(q))
	if err != nil {
		evalQ = expandNode(q)
	}

	// Determine variable to eliminate
	v := varName
	if v == "" {
		varsP := collectVariables(evalP)
		varsQ := collectVariables(evalQ)
		common := intersectStrings(varsP, varsQ)
		if len(common) > 0 {
			v = selectMainVariable(common)
		} else {
			allVars := unionStrings(varsP, varsQ)
			if len(allVars) == 0 {
				// Both are constants with respect to any variable
				if isZero(evalP) || isZero(evalQ) {
					return mustRational(0, 1), nil
				}
				return mustRational(1, 1), nil
			}
			v = selectMainVariable(allVars)
		}
	}

	polyP, okP := extractPoly(evalP, v)
	polyQ, okQ := extractPoly(evalQ, v)
	if !okP || !okQ {
		return nil, fmt.Errorf("%s", i18n.T("resultant.err_resultant_failed_to_extract_polynomials", v))
	}

	// Boundary cases
	if isPolyZero(polyP) || isPolyZero(polyQ) {
		return mustRational(0, 1), nil
	}

	m := polyP.degree()
	n := polyQ.degree()

	if m == 0 && n == 0 {
		return mustRational(1, 1), nil
	}

	if m > 0 && n == 0 {
		// Res(P, b0) = b0^m
		b0 := polyQ.coeffs[0]
		powNode, err := simplifyPow(b0, mustRational(int64(m), 1))
		if err != nil {
			return nil, err
		}
		return Eval(expandNode(powNode))
	}

	if m == 0 && n > 0 {
		// Res(a0, Q) = a0^n
		a0 := polyP.coeffs[0]
		powNode, err := simplifyPow(a0, mustRational(int64(n), 1))
		if err != nil {
			return nil, err
		}
		return Eval(expandNode(powNode))
	}

	// Construct Sylvester Matrix of size (m + n) x (m + n)
	sylvesterMat := buildSylvesterMatrix(polyP, polyQ)

	// Compute determinant
	detVal, err := evalDet(sylvesterMat)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("resultant.err_resultant_determinant_calculation_failed", err))
	}

	// Expand and simplify the result
	res, err := Eval(expandNode(detVal))
	if err != nil {
		return detVal, nil
	}
	return res, nil
}

// buildSylvesterMatrix constructs an (m+n) x (m+n) Sylvester matrix from pPoly and qPoly.
// Rows 0 .. n-1 contain shifted coefficients of P: [a_m, a_{m-1}, ..., a_0]
// Rows n .. m+n-1 contain shifted coefficients of Q: [b_n, b_{n-1}, ..., b_0]
func buildSylvesterMatrix(pPoly, qPoly *univariatePoly) *MatrixNode {
	m := pPoly.degree()
	n := qPoly.degree()
	size := m + n

	data := make([][]Node, size)
	for r := 0; r < size; r++ {
		data[r] = make([]Node, size)
		for c := 0; c < size; c++ {
			data[r][c] = mustRational(0, 1)
		}
	}

	// Fill P rows (n rows)
	// pPoly.coeffs has index i for x^i (coeffs[m] is a_m, coeffs[0] is a_0)
	for r := 0; r < n; r++ {
		for deg := m; deg >= 0; deg-- {
			col := r + (m - deg)
			if col < size {
				data[r][col] = pPoly.coeffs[deg]
			}
		}
	}

	// Fill Q rows (m rows)
	// qPoly.coeffs has index j for x^j (coeffs[n] is b_n, coeffs[0] is b_0)
	for s := 0; s < m; s++ {
		row := n + s
		for deg := n; deg >= 0; deg-- {
			col := s + (n - deg)
			if col < size {
				data[row][col] = qPoly.coeffs[deg]
			}
		}
	}

	return &MatrixNode{
		Rows: size,
		Cols: size,
		Data: data,
	}
}

func intersectStrings(a, b []string) []string {
	seen := make(map[string]bool)
	for _, s := range a {
		seen[s] = true
	}
	var res []string
	for _, s := range b {
		if seen[s] {
			res = append(res, s)
		}
	}
	return res
}
