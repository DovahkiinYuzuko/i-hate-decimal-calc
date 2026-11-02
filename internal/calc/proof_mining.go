package calc

import (
	"fmt"
	"math/big"
)

// VerifyIdealMembership verifies whether f == sum_{i=1}^m h_i * f_i algebraically.
// It expands sum h_i * f_i - f using multivariate polynomial arithmetic and tests whether the residual strictly vanishes.
func VerifyIdealMembership(target Node, gens []Node, cofactors []Node, vars []string, order MonomialOrder) (*MPoly, bool, error) {
	n := len(vars)
	varMap := make(map[string]int, n)
	for i, v := range vars {
		varMap[v] = i
	}

	targetPoly, err := nodeToMPoly(target, varMap, n, order)
	if err != nil {
		return nil, false, fmt.Errorf("target nodeToMPoly: %w", err)
	}

	sumPoly := NewZeroMPoly(n)
	for i, gen := range gens {
		if i >= len(cofactors) {
			break
		}
		genPoly, err := nodeToMPoly(gen, varMap, n, order)
		if err != nil {
			return nil, false, fmt.Errorf("gen[%d] nodeToMPoly: %w", i, err)
		}
		cofPoly, err := nodeToMPoly(cofactors[i], varMap, n, order)
		if err != nil {
			return nil, false, fmt.Errorf("cofactor[%d] nodeToMPoly: %w", i, err)
		}
		termPoly := cofPoly.Mul(genPoly, order)
		sumPoly = sumPoly.Add(termPoly, order)
	}

	residual := sumPoly.Sub(targetPoly, order)
	return residual, residual.IsZero(), nil
}

// VerifySOSDecomposition verifies whether P(x) == sum_{i=1}^k c_i * q_i(x)^2 algebraically.
// It expands sum c_i * q_i^2 - P using multivariate polynomial arithmetic, ensuring all weights are strictly positive.
func VerifySOSDecomposition(target Node, terms []SOSCertificateTerm, vars []string, order MonomialOrder) (*MPoly, bool, error) {
	n := len(vars)
	varMap := make(map[string]int, n)
	for i, v := range vars {
		varMap[v] = i
	}

	targetPoly, err := nodeToMPoly(target, varMap, n, order)
	if err != nil {
		return nil, false, fmt.Errorf("target nodeToMPoly: %w", err)
	}

	sumPoly := NewZeroMPoly(n)
	for i, term := range terms {
		if term.Weight == nil || term.Weight.Sign() <= 0 {
			return nil, false, fmt.Errorf("SOS term %d has invalid or non-positive weight %v", i, term.Weight)
		}
		if term.Poly == nil {
			return nil, false, fmt.Errorf("SOS term %d has nil polynomial", i)
		}
		qPoly, err := nodeToMPoly(term.Poly, varMap, n, order)
		if err != nil {
			return nil, false, fmt.Errorf("SOS term[%d] nodeToMPoly: %w", i, err)
		}

		qSq := qPoly.Mul(qPoly, order)
		for j := range qSq.Terms {
			qSq.Terms[j].Coeff = new(big.Rat).Mul(qSq.Terms[j].Coeff, term.Weight)
		}
		sumPoly = sumPoly.Add(qSq, order)
	}

	residual := sumPoly.Sub(targetPoly, order)
	return residual, residual.IsZero(), nil
}
