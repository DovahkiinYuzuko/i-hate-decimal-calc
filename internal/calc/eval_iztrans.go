package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// EvalInvZTrans computes the causal inverse Z-transform: x[n] = Z^-1{X(z)}.
func EvalInvZTrans(F Node, zName, nName string, env *Env) (Node, error) {
	if F == nil {
		return nil, fmt.Errorf("%s", i18n.T("err_cannot_evaluate_nil_node"))
	}
	if zName == "" {
		zName = "z"
	}
	if nName == "" {
		nName = "n"
	}
	nVar := &VarNode{Name: nName}

	evaledF, err := Eval(F)
	if err == nil && evaledF != nil {
		F = evaledF
	}

	// Constant with respect to z: Z^-1{C} = C * delta(n)
	if !containsVariable(F, zName) {
		deltaNode := &FuncNode{Name: "delta", Args: []Node{nVar}}
		if r, ok := F.(*RationalNode); ok && r.Val.Sign() == 0 {
			return &RationalNode{Val: big.NewRat(0, 1)}, nil
		}
		oneVal := big.NewRat(1, 1)
		if r, ok := F.(*RationalNode); ok && r.Val.Cmp(oneVal) == 0 {
			return deltaNode, nil
		}
		return &MulNode{Factors: []Node{F, deltaNode}}, nil
	}

	// Linearity: Z^-1{F1 + F2} = Z^-1{F1} + Z^-1{F2}
	if add, ok := F.(*AddNode); ok {
		var terms []Node
		for _, t := range add.Terms {
			it, err := EvalInvZTrans(t, zName, nName, env)
			if err != nil {
				return nil, err
			}
			terms = append(terms, it)
		}
		return simplifyAdd(terms)
	}

	// Multiplicative constant extraction
	if mul, ok := F.(*MulNode); ok {
		var constFactors []Node
		var zFactors []Node
		for _, fac := range mul.Factors {
			if containsVariable(fac, zName) {
				zFactors = append(zFactors, fac)
			} else {
				constFactors = append(constFactors, fac)
			}
		}

		if len(constFactors) > 0 {
			var zPart Node
			if len(zFactors) == 0 {
				zPart = &RationalNode{Val: big.NewRat(1, 1)}
			} else if len(zFactors) == 1 {
				zPart = zFactors[0]
			} else {
				zPart = &MulNode{Factors: zFactors}
			}

			izPart, err := EvalInvZTrans(zPart, zName, nName, env)
			if err != nil {
				return nil, err
			}
			allFactors := append(constFactors, izPart)
			return simplifyMul(allFactors)
		}
	}

	// Handle standard PFD of X(z)/z
	return invZTransRational(F, zName, nVar, env)
}

func invZTransRational(F Node, zName string, nVar *VarNode, env *Env) (Node, error) {
	numNode, denNode := extractNumeratorDenominator(F)
	zVar := &VarNode{Name: zName}

	// If expression is already a power of z: z^(-k) -> delta(n - k)
	if v, ok := F.(*VarNode); ok && v.Name == zName {
		// F is z^1 -> non-causal term
		return nil, fmt.Errorf("%s", i18n.T("iztrans.err_non_causal_term", Format(F)))
	}
	if pow, ok := F.(*PowNode); ok {
		if v, isV := pow.Base.(*VarNode); isV && v.Name == zName {
			if r, isR := pow.Exp.(*RationalNode); isR && r.Val.IsInt() {
				k := r.Val.Num().Int64()
				if k <= 0 {
					delay := -k
					if delay == 0 {
						return &FuncNode{Name: "delta", Args: []Node{nVar}}, nil
					}
					arg := &AddNode{Terms: []Node{nVar, &RationalNode{Val: big.NewRat(-delay, 1)}}}
					return &FuncNode{Name: "delta", Args: []Node{arg}}, nil
				}
				// z^k (k > 0) is non-causal
				return nil, fmt.Errorf("%s", i18n.T("iztrans.err_non_causal_term", Format(F)))
			}
		}
	}

	// Try polynomial conversion for polynomial long division
	numPoly, okNum := extractPoly(numNode, zName)
	denPoly, okDen := extractPoly(denNode, zName)

	var deltaTerms []Node
	properNumNode := numNode
	properDenNode := denNode

	if okNum && okDen && denPoly.degree() >= 0 {
		// If numerator degree >= denominator degree: perform polynomial long division
		if numPoly.degree() >= denPoly.degree() {
			quot, rem, ok := polyDivide(numPoly, denPoly)
			if ok {
				// quot is a polynomial in z: sum c_k z^k
				for k, coeff := range quot.coeffs {
					if isZero(coeff) {
						continue
					}
					if k > 0 {
						// Advanced term z^k (k > 0) is non-causal
						return nil, fmt.Errorf("%s", i18n.T("iztrans.err_non_causal_term", Format(F)))
					}
					// k == 0 is constant: c_0 * delta(n)
					dNode := &FuncNode{Name: "delta", Args: []Node{nVar}}
					term, err := simplifyMul([]Node{coeff, dNode})
					if err != nil {
						term = &MulNode{Factors: []Node{coeff, dNode}}
					}
					deltaTerms = append(deltaTerms, term)
				}

				if isPolyZero(rem) {
					// Pure polynomial: return delta terms directly
					if len(deltaTerms) == 0 {
						return &RationalNode{Val: big.NewRat(0, 1)}, nil
					}
					return simplifyAdd(deltaTerms)
				}
				properNumNode = rem.toNode()
				properDenNode = denPoly.toNode()
			}
		}
	}

	// Now compute X_proper(z) / z for partial fraction decomposition
	// F_div_z = properNumNode / (properDenNode * z)
	denTimesZ, err := simplifyMul([]Node{properDenNode, zVar})
	if err != nil {
		denTimesZ = &MulNode{Factors: []Node{properDenNode, zVar}}
	}
	fDivZ := makeDiv(properNumNode, denTimesZ)

	// Perform partial fraction expansion
	pfdNode, err := EvalApart(fDivZ, zName)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("iztrans.err_cannot_invert", F.String()))
	}

	var pfdTerms []Node
	if add, ok := pfdNode.(*AddNode); ok {
		pfdTerms = add.Terms
	} else {
		pfdTerms = []Node{pfdNode}
	}

	var invTerms []Node
	invTerms = append(invTerms, deltaTerms...)

	for _, term := range pfdTerms {
		it, err := invZTransTerm(term, zName, nVar, env)
		if err != nil {
			return nil, err
		}
		invTerms = append(invTerms, it)
	}

	return simplifyAdd(invTerms)
}

// invZTransTerm transforms a single partial fraction term of X(z)/z to time domain.
// Because the term is from X(z)/z, we multiply by z in the transform domain:
// A/(z-p)^k in X(z)/z  -->  A * z / (z-p)^k in X(z).
func invZTransTerm(term Node, zName string, nVar *VarNode, env *Env) (Node, error) {
	if !containsVariable(term, zName) {
		return &RationalNode{Val: big.NewRat(0, 1)}, nil
	}

	// Extract pole: C / (z - a)^k
	coeff, aRat, order, ok := extractLinearPole(term, zName)
	if ok && order >= 1 {
		// Case 1: Pole at origin a = 0 (z = 0)
		// A * z / z^k = A * z^(1-k) -> A * delta(n - (k - 1))
		if aRat.Sign() == 0 {
			shift := order - 1
			var deltaArg Node
			if shift == 0 {
				deltaArg = nVar
			} else {
				deltaArg = &AddNode{Terms: []Node{nVar, &RationalNode{Val: big.NewRat(-shift, 1)}}}
			}
			dNode := &FuncNode{Name: "delta", Args: []Node{deltaArg}}
			if coeff.Cmp(big.NewRat(1, 1)) == 0 {
				return dNode, nil
			}
			return &MulNode{Factors: []Node{&RationalNode{Val: coeff}, dNode}}, nil
		}

		// Case 2: Pole at a != 0
		// A * z / (z - a)^k -> A * binom(n, k-1) * a^(n - (k - 1))
		var factors []Node
		if coeff.Cmp(big.NewRat(1, 1)) != 0 {
			factors = append(factors, &RationalNode{Val: coeff})
		}

		kMinus1 := order - 1
		if kMinus1 > 0 {
			binomPoly := makeBinomialPoly(nVar, kMinus1)
			factors = append(factors, binomPoly)
		}

		// Power term: a^(n - kMinus1)
		var expNode Node
		if kMinus1 == 0 {
			expNode = nVar
		} else {
			expNode = &AddNode{Terms: []Node{nVar, &RationalNode{Val: big.NewRat(-kMinus1, 1)}}}
		}

		aNode := &RationalNode{Val: aRat}
		if aRat.Cmp(big.NewRat(1, 1)) != 0 {
			factors = append(factors, &PowNode{Base: aNode, Exp: expNode})
		}

		if len(factors) == 0 {
			return &RationalNode{Val: big.NewRat(1, 1)}, nil
		}
		if len(factors) == 1 {
			return factors[0], nil
		}
		return &MulNode{Factors: factors}, nil
	}

	return nil, fmt.Errorf("%s", i18n.T("iztrans.err_cannot_invert", term.String()))
}

// makeBinomialPoly generates binom(n, k) = n*(n-1)*...*(n-k+1) / k!
func makeBinomialPoly(nVar *VarNode, k int64) Node {
	if k == 1 {
		return nVar
	}
	fact := factorialBigInt(k)
	factRat := new(big.Rat).SetInt(fact)

	var numFactors []Node
	for j := int64(0); j < k; j++ {
		if j == 0 {
			numFactors = append(numFactors, nVar)
		} else {
			term := &AddNode{Terms: []Node{nVar, &RationalNode{Val: big.NewRat(-j, 1)}}}
			numFactors = append(numFactors, term)
		}
	}

	numNode := &MulNode{Factors: numFactors}
	invFact := new(big.Rat).Inv(factRat)
	return &MulNode{Factors: []Node{&RationalNode{Val: invFact}, numNode}}
}
