package calc

import (
	"math/big"
)

// -------------------------------------------------------------------------
// Multiplication Simplification & Distributive Law Expansion
// -------------------------------------------------------------------------

func simplifyMul(factors []Node) (Node, error) {
	// 1. Flatten nested MulNodes
	flatFactors := make([]Node, 0, len(factors))
	for _, f := range factors {
		if mul, ok := f.(*MulNode); ok {
			flatFactors = append(flatFactors, mul.Factors...)
		} else {
			flatFactors = append(flatFactors, f)
		}
	}

	// 1.5. Check if any factor is a MatrixNode (non-commutative sequential multiplication)
	hasMatrix := false
	for _, f := range flatFactors {
		if _, ok := f.(*MatrixNode); ok {
			hasMatrix = true
			break
		}
	}
	if hasMatrix {
		if len(flatFactors) == 0 {
			return mustRational(1, 1), nil
		}
		current := flatFactors[0]
		for i := 1; i < len(flatFactors); i++ {
			next := flatFactors[i]
			prod, err := mulMatrixOrScalar(current, next)
			if err != nil {
				return nil, err
			}
			current = prod
		}
		return current, nil
	}

	// 2. Check for Distributive Law: if any factor is an AddNode, expand!
	// (A + B) * (C + D) -> A*C + A*D + B*C + B*D
	for i, f := range flatFactors {
		if add, ok := f.(*AddNode); ok {
			// Distribute 'add' over remaining factors
			rest := append([]Node{}, flatFactors[:i]...)
			rest = append(rest, flatFactors[i+1:]...)

			var expandedTerms []Node
			for _, term := range add.Terms {
				pair := append([]Node{term}, rest...)
				prod, err := simplifyMul(pair)
				if err != nil {
					return nil, err
				}
				expandedTerms = append(expandedTerms, prod)
			}
			return simplifyAdd(expandedTerms)
		}
	}

	// 3. Multiply factors together (numbers, rads, complexes, etc.)
	coeff := big.NewRat(1, 1)
	radicandProduct := big.NewInt(1)
	hasRadicand := false

	var otherFactors []Node
	complexFactors := []*ComplexNode{}

	for _, f := range flatFactors {
		switch v := f.(type) {
		case *RationalNode:
			coeff.Mul(coeff, v.Val)

		case *SqrtNode:
			if ratRad, ok := v.Radicand.(*RationalNode); ok && ratRad.Val.IsInt() && ratRad.Val.Sign() > 0 {
				hasRadicand = true
				radicandProduct.Mul(radicandProduct, ratRad.Val.Num())
			} else {
				otherFactors = append(otherFactors, v)
			}

		case *ComplexNode:
			complexFactors = append(complexFactors, v)

		case *PowNode:
			// Check for reciprocal: Pow(X, -1) -> handle rationalization later
			otherFactors = append(otherFactors, v)

		default:
			otherFactors = append(otherFactors, v)
		}
	}

	var currentComplex *ComplexNode
	if len(complexFactors) > 0 {
		for _, cf := range complexFactors {
			if currentComplex == nil {
				currentComplex = cf
			} else {
				// Multiply currentComplex by cf
				ac, _ := simplifyMul([]Node{currentComplex.Real, cf.Real})
				bd, _ := simplifyMul([]Node{currentComplex.Imag, cf.Imag})
				negBd, _ := simplifyUnaryOp("-", bd)
				realPart, _ := simplifyAdd([]Node{ac, negBd})

				ad, _ := simplifyMul([]Node{currentComplex.Real, cf.Imag})
				bc, _ := simplifyMul([]Node{currentComplex.Imag, cf.Real})
				imagPart, _ := simplifyAdd([]Node{ad, bc})

				currentComplex = NewComplex(realPart, imagPart)
			}
		}
		if len(otherFactors) == 0 && !hasRadicand {
			// If coeff != 1, scale currentComplex
			if coeff.Cmp(big.NewRat(1, 1)) != 0 {
				scale := &RationalNode{Val: coeff}
				newReal, _ := simplifyMul([]Node{scale, currentComplex.Real})
				newImag, _ := simplifyMul([]Node{scale, currentComplex.Imag})
				currentComplex = NewComplex(newReal, newImag)
			}
			// If imag is 0, reduce to real part
			if isZero(currentComplex.Imag) {
				return currentComplex.Real, nil
			}
			return currentComplex, nil
		}
	}

	// Process radical product
	if hasRadicand {
		sqrtSimp, err := simplifySqrt(&RationalNode{Val: new(big.Rat).SetInt(radicandProduct)})
		if err != nil {
			return nil, err
		}
		switch sv := sqrtSimp.(type) {
		case *RationalNode:
			coeff.Mul(coeff, sv.Val)
		case *MulNode:
			for _, mf := range sv.Factors {
				if r, ok := mf.(*RationalNode); ok {
					coeff.Mul(coeff, r.Val)
				} else {
					otherFactors = append(otherFactors, mf)
				}
			}
		default:
			otherFactors = append(otherFactors, sv)
		}
	}

	// Check for monomial rationalization: e.g. Pow(sqrt(d), -1)
	// coeff * otherFactors * Pow(sqrt(d), -1) -> (coeff / d) * otherFactors * sqrt(d)
	var finalFactors []Node
	didRationalize := false
	for _, f := range otherFactors {
		if pow, ok := f.(*PowNode); ok {
			if rExp, ok := pow.Exp.(*RationalNode); ok && rExp.Val.Cmp(big.NewRat(-1, 1)) == 0 {
				if s, ok := pow.Base.(*SqrtNode); ok {
					if rRad, ok := s.Radicand.(*RationalNode); ok && rRad.Val.IsInt() && rRad.Val.Sign() > 0 {
						// Rationalize! Divide coeff by d, and multiply numerator by sqrt(d)
						d := rRad.Val.Num()
						coeff.Quo(coeff, new(big.Rat).SetInt(d))
						finalFactors = append(finalFactors, s)
						didRationalize = true
						continue
					}
				}
				// 1 / (c * sqrt(d))
				if mul, ok := pow.Base.(*MulNode); ok && len(mul.Factors) == 2 {
					var cRat *big.Rat
					var sNode *SqrtNode
					if r, ok := mul.Factors[0].(*RationalNode); ok {
						cRat = r.Val
						if s, ok := mul.Factors[1].(*SqrtNode); ok {
							sNode = s
						}
					}
					if cRat != nil && sNode != nil {
						if rRad, ok := sNode.Radicand.(*RationalNode); ok && rRad.Val.IsInt() && rRad.Val.Sign() > 0 {
							d := rRad.Val.Num()
							// coeff / (c * d) * sqrt(d)
							cd := new(big.Rat).Mul(cRat, new(big.Rat).SetInt(d))
							coeff.Quo(coeff, cd)
							finalFactors = append(finalFactors, sNode)
							didRationalize = true
							continue
						}
					}
				}
			}
		}
		finalFactors = append(finalFactors, f)
	}

	if didRationalize {
		all := append([]Node{&RationalNode{Val: coeff}}, finalFactors...)
		return simplifyMul(all)
	}

	// Merge exp(...) factors: exp(A) * exp(B) -> exp(A + B)
	var nonExpFactors []Node
	var expArgs []Node
	for _, f := range finalFactors {
		if fn, ok := f.(*FuncNode); ok && fn.Name == "exp" && len(fn.Args) == 1 {
			expArgs = append(expArgs, fn.Args[0])
		} else {
			nonExpFactors = append(nonExpFactors, f)
		}
	}
	if len(expArgs) > 1 {
		sumArg, err := simplifyAdd(expArgs)
		if err == nil {
			if isZero(sumArg) {
				finalFactors = nonExpFactors
			} else {
				mergedExp, err := simplifyFuncWithEnv("exp", []Node{sumArg}, nil)
				if err == nil {
					finalFactors = append(nonExpFactors, mergedExp)
				}
			}
		}
	}

	// Merge powers of identical bases (e.g. pi * pi^-1 -> 1, x^2 * x^3 -> x^5)
	var mergedFactors []Node
	type baseEntry struct {
		base Node
		exp  *big.Rat
	}
	var baseList []*baseEntry

	for _, f := range finalFactors {
		var base Node
		exp := big.NewRat(1, 1)

		if pow, ok := f.(*PowNode); ok {
			base = pow.Base
			if rExp, ok := pow.Exp.(*RationalNode); ok {
				exp = new(big.Rat).Set(rExp.Val)
			} else {
				mergedFactors = append(mergedFactors, f)
				continue
			}
		} else if _, isConst := f.(*ConstNode); isConst {
			base = f
		} else if _, isVar := f.(*VarNode); isVar {
			base = f
		} else if _, isFunc := f.(*FuncNode); isFunc {
			base = f
		} else {
			mergedFactors = append(mergedFactors, f)
			continue
		}

		found := false
		for _, be := range baseList {
			if be.base.Equal(base) {
				be.exp.Add(be.exp, exp)
				found = true
				break
			}
		}
		if !found {
			baseList = append(baseList, &baseEntry{base: base, exp: exp})
		}
	}

	for _, be := range baseList {
		if be.exp.Sign() == 0 {
			// base^0 = 1 (cancels out)
			continue
		}
		one := big.NewRat(1, 1)
		if be.exp.Cmp(one) == 0 {
			mergedFactors = append(mergedFactors, be.base)
		} else {
			powSimp, err := simplifyPow(be.base, &RationalNode{Val: be.exp})
			if err != nil {
				return nil, err
			}
			mergedFactors = append(mergedFactors, powSimp)
		}
	}
	finalFactors = mergedFactors

	if coeff.Sign() == 0 {
		return mustRational(0, 1), nil
	}

	var realScalar Node
	if len(finalFactors) == 0 {
		realScalar = &RationalNode{Val: coeff}
	} else {
		one := big.NewRat(1, 1)
		if coeff.Cmp(one) == 0 {
			if len(finalFactors) == 1 {
				realScalar = finalFactors[0]
			} else {
				realScalar = NewMul(finalFactors)
			}
		} else {
			all := append([]Node{&RationalNode{Val: coeff}}, finalFactors...)
			realScalar = NewMul(all)
		}
	}

	if currentComplex != nil {
		newReal, _ := simplifyMul([]Node{realScalar, currentComplex.Real})
		newImag, _ := simplifyMul([]Node{realScalar, currentComplex.Imag})
		if isZero(newImag) {
			return newReal, nil
		}
		return NewComplex(newReal, newImag), nil
	}

	return realScalar, nil
}
