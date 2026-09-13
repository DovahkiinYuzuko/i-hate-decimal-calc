package calc

import (
	"fmt"
	"math/big"
)

// -------------------------------------------------------------------------
// Unary Operators (- and !)
// -------------------------------------------------------------------------

func simplifyUnaryOp(op string, expr Node) (Node, error) {
	switch op {
	case "-":
		switch v := expr.(type) {
		case *RationalNode:
			neg := new(big.Rat).Neg(v.Val)
			return &RationalNode{Val: neg}, nil
		case *ComplexNode:
			negReal, err := simplifyUnaryOp("-", v.Real)
			if err != nil {
				return nil, err
			}
			negImag, err := simplifyUnaryOp("-", v.Imag)
			if err != nil {
				return nil, err
			}
			return NewComplex(negReal, negImag), nil
		case *AddNode:
			negTerms := make([]Node, len(v.Terms))
			for i, t := range v.Terms {
				nt, err := simplifyUnaryOp("-", t)
				if err != nil {
					return nil, err
				}
				negTerms[i] = nt
			}
			return simplifyAdd(negTerms)
		case *MatrixNode:
			negData := make([][]Node, v.Rows)
			for r := 0; r < v.Rows; r++ {
				negData[r] = make([]Node, v.Cols)
				for c := 0; c < v.Cols; c++ {
					negElem, err := simplifyUnaryOp("-", v.Data[r][c])
					if err != nil {
						return nil, err
					}
					negData[r][c] = negElem
				}
			}
			return NewMatrix(v.Rows, v.Cols, negData)
		default:
			negOne, _ := NewRational(-1, 1)
			return simplifyMul([]Node{negOne, expr})
		}

	case "!":
		rat, ok := expr.(*RationalNode)
		if !ok || !rat.Val.IsInt() || rat.Val.Sign() < 0 {
			return nil, fmt.Errorf("factorial domain error: factorial requires non-negative integer, got %s", expr.String())
		}
		if !rat.Val.Num().IsInt64() || rat.Val.Num().Int64() > 100000 {
			return nil, fmt.Errorf("factorial domain error: factorial is too large to compute, got %s", expr.String())
		}
		n := rat.Val.Num().Int64()
		res := big.NewInt(1)
		for i := int64(2); i <= n; i++ {
			res.Mul(res, big.NewInt(i))
		}
		return &RationalNode{Val: new(big.Rat).SetInt(res)}, nil

	default:
		return nil, fmt.Errorf("unknown unary operator: %s", op)
	}
}

// -------------------------------------------------------------------------
// Power / Exponentiation
// -------------------------------------------------------------------------

func simplifyPow(base, exp Node) (Node, error) {
	ratExp, expIsRat := exp.(*RationalNode)
	ratBase, baseIsRat := base.(*RationalNode)

	// 0^exp
	if baseIsRat && ratBase.Val.Sign() == 0 {
		if expIsRat && ratExp.Val.Sign() < 0 {
			return nil, fmt.Errorf("division by zero: 0^(negative) is undefined")
		}
		if expIsRat && ratExp.Val.Sign() == 0 {
			return mustRational(1, 1), nil // 0^0 = 1 by specification
		}
		return mustRational(0, 1), nil
	}

	// base^0 = 1
	if expIsRat && ratExp.Val.Sign() == 0 {
		return mustRational(1, 1), nil
	}

	// base^1 = base
	one := big.NewRat(1, 1)
	if expIsRat && ratExp.Val.Cmp(one) == 0 {
		return base, nil
	}

	// Rational^Integer
	if baseIsRat && expIsRat && ratExp.Val.IsInt() {
		expInt := ratExp.Val.Num().Int64()
		if expInt < 0 {
			// Invert base
			if ratBase.Val.Sign() == 0 {
				return nil, fmt.Errorf("division by zero: reciprocal of 0")
			}
			inv := new(big.Rat).Inv(ratBase.Val)
			return simplifyPow(&RationalNode{Val: inv}, &RationalNode{Val: big.NewRat(-expInt, 1)})
		}
		numPow := new(big.Int).Exp(ratBase.Val.Num(), big.NewInt(expInt), nil)
		denomPow := new(big.Int).Exp(ratBase.Val.Denom(), big.NewInt(expInt), nil)
		return &RationalNode{Val: new(big.Rat).SetFrac(numPow, denomPow)}, nil
	}

	// base^(1/2) -> sqrt(base)
	half := big.NewRat(1, 2)
	if expIsRat && ratExp.Val.Cmp(half) == 0 {
		return simplifySqrt(base)
	}

	// Sqrt^Integer
	if s, ok := base.(*SqrtNode); ok && expIsRat && ratExp.Val.IsInt() {
		expInt := ratExp.Val.Num().Int64()
		if expInt == 2 {
			return s.Radicand, nil
		}
		if expInt > 0 && expInt%2 == 0 {
			return simplifyPow(s.Radicand, &RationalNode{Val: big.NewRat(expInt/2, 1)})
		}
		if expInt > 2 {
			radPow, err := simplifyPow(s.Radicand, &RationalNode{Val: big.NewRat(expInt/2, 1)})
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{radPow, s})
		}
	}

	// Mul^Integer: (A * B * ...)^n = A^n * B^n * ...
	if mul, ok := base.(*MulNode); ok && expIsRat && ratExp.Val.IsInt() {
		var poweredFactors []Node
		for _, f := range mul.Factors {
			pf, err := simplifyPow(f, exp)
			if err != nil {
				return nil, err
			}
			poweredFactors = append(poweredFactors, pf)
		}
		return simplifyMul(poweredFactors)
	}

	// UnaryOp("-", A)^Integer: (-A)^n
	if uop, ok := base.(*UnaryOpNode); ok && uop.Op == "-" && expIsRat && ratExp.Val.IsInt() {
		expInt := ratExp.Val.Num().Int64()
		innerPow, err := simplifyPow(uop.Expr, exp)
		if err != nil {
			return nil, err
		}
		if expInt%2 == 0 {
			return innerPow, nil
		}
		return simplifyUnaryOp("-", innerPow)
	}

	// Complex^Integer
	if c, ok := base.(*ComplexNode); ok && expIsRat && ratExp.Val.IsInt() {
		expInt := ratExp.Val.Num().Int64()
		if expInt == -1 {
			// 1 / (a + bi) = (a - bi) / (a^2 + b^2)
			a2, err := simplifyPow(c.Real, mustRational(2, 1))
			if err != nil {
				return nil, err
			}
			b2, err := simplifyPow(c.Imag, mustRational(2, 1))
			if err != nil {
				return nil, err
			}
			denom, err := simplifyAdd([]Node{a2, b2})
			if err != nil {
				return nil, err
			}
			if isZero(denom) {
				return nil, fmt.Errorf("division by zero: 1/(0+0i)")
			}
			negImag, err := simplifyUnaryOp("-", c.Imag)
			if err != nil {
				return nil, err
			}
			conj := NewComplex(c.Real, negImag)
			invDenom, err := simplifyPow(denom, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{conj, invDenom})
		}
		if expInt < -1 {
			inv, err := simplifyPow(c, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			return simplifyPow(inv, mustRational(-expInt, 1))
		}
		if expInt > 1 {
			res := Node(c)
			for i := int64(1); i < expInt; i++ {
				var err error
				res, err = simplifyMul([]Node{res, c})
				if err != nil {
					return nil, err
				}
			}
			return res, nil
		}
	}

	// Add^Integer: (a + b*sqrt(r))^-1 -> Rationalize binomial radical denominator
	if add, ok := base.(*AddNode); ok && expIsRat && ratExp.Val.IsInt() {
		expInt := ratExp.Val.Num().Int64()
		if expInt == -1 {
			if inv, ok := rationalizeBinomialDenominator(add); ok {
				return inv, nil
			}
		}
		if expInt < -1 {
			if inv, ok := rationalizeBinomialDenominator(add); ok {
				return simplifyPow(inv, mustRational(-expInt, 1))
			}
		}
	}

	return NewPow(base, exp)
}

// -------------------------------------------------------------------------
// Multiplication Simplification & Distributive Law Expansion
// -------------------------------------------------------------------------

func simplifyMul(factors []Node) (Node, error) {
	// 1. Flatten nested MulNodes
	var flatFactors []Node
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

// -------------------------------------------------------------------------
// Addition Simplification & Like-term Collection (Coeff × Base Model)
// -------------------------------------------------------------------------

type termEntry struct {
	coeff *big.Rat
	base  string
	node  Node
}

func simplifyAdd(terms []Node) (Node, error) {
	// 1. Flatten nested AddNodes
	var flatTerms []Node
	for _, t := range terms {
		if add, ok := t.(*AddNode); ok {
			flatTerms = append(flatTerms, add.Terms...)
		} else {
			flatTerms = append(flatTerms, t)
		}
	}

	// 1.5. Check if any term is a MatrixNode
	var firstMatrix *MatrixNode
	for _, t := range flatTerms {
		if m, ok := t.(*MatrixNode); ok {
			firstMatrix = m
			break
		}
	}
	if firstMatrix != nil {
		resData := make([][]Node, firstMatrix.Rows)
		for r := 0; r < firstMatrix.Rows; r++ {
			resData[r] = make([]Node, firstMatrix.Cols)
			for c := 0; c < firstMatrix.Cols; c++ {
				resData[r][c] = mustRational(0, 1)
			}
		}
		for _, t := range flatTerms {
			m, ok := t.(*MatrixNode)
			if !ok {
				return nil, fmt.Errorf("matrix dimension error: cannot add scalar %s to matrix", t.String())
			}
			if m.Rows != firstMatrix.Rows || m.Cols != firstMatrix.Cols {
				return nil, fmt.Errorf("matrix dimension mismatch: cannot add %dx%d matrix and %dx%d matrix",
					firstMatrix.Rows, firstMatrix.Cols, m.Rows, m.Cols)
			}
			for r := 0; r < m.Rows; r++ {
				for c := 0; c < m.Cols; c++ {
					sum, err := simplifyAdd([]Node{resData[r][c], m.Data[r][c]})
					if err != nil {
						return nil, err
					}
					resData[r][c] = sum
				}
			}
		}
		return NewMatrix(firstMatrix.Rows, firstMatrix.Cols, resData)
	}

	// 2. Separate into Coeff × Base
	ratSum := big.NewRat(0, 1)
	imagSum := big.NewRat(0, 1)
	hasComplex := false
	termMap := make(map[string]*termEntry)
	var termOrder []string

	for _, t := range flatTerms {
		switch v := t.(type) {
		case *RationalNode:
			ratSum.Add(ratSum, v.Val)

		case *ComplexNode:
			hasComplex = true
			if rRat, ok := v.Real.(*RationalNode); ok {
				ratSum.Add(ratSum, rRat.Val)
			} else if !isZero(v.Real) {
				baseKey := v.Real.String()
				if entry, exists := termMap[baseKey]; exists {
					entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
				} else {
					termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v.Real}
					termOrder = append(termOrder, baseKey)
				}
			}
			if iRat, ok := v.Imag.(*RationalNode); ok {
				imagSum.Add(imagSum, iRat.Val)
			} else if !isZero(v.Imag) {
				baseKey := fmt.Sprintf("(%s)*i", v.Imag.String())
				if entry, exists := termMap[baseKey]; exists {
					entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
				} else {
					termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
					termOrder = append(termOrder, baseKey)
				}
			}

		case *SqrtNode:
			baseKey := v.String()
			if entry, exists := termMap[baseKey]; exists {
				entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
			} else {
				termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
				termOrder = append(termOrder, baseKey)
			}

		case *MulNode:
			// Check if v is Coeff * Base
			if len(v.Factors) >= 2 {
				if r, ok := v.Factors[0].(*RationalNode); ok {
					var rest []Node
					rest = append(rest, v.Factors[1:]...)
					var baseNode Node
					if len(rest) == 1 {
						baseNode = rest[0]
					} else {
						baseNode = NewMul(rest)
					}
					baseKey := baseNode.String()
					if entry, exists := termMap[baseKey]; exists {
						entry.coeff.Add(entry.coeff, r.Val)
					} else {
						termMap[baseKey] = &termEntry{coeff: new(big.Rat).Set(r.Val), base: baseKey, node: baseNode}
						termOrder = append(termOrder, baseKey)
					}
					continue
				}
			}
			baseKey := v.String()
			if entry, exists := termMap[baseKey]; exists {
				entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
			} else {
				termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
				termOrder = append(termOrder, baseKey)
			}

		default:
			baseKey := v.String()
			if entry, exists := termMap[baseKey]; exists {
				entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
			} else {
				termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
				termOrder = append(termOrder, baseKey)
			}
		}
	}

	// 3. Assemble combined terms
	var resultTerms []Node
	if ratSum.Sign() != 0 {
		resultTerms = append(resultTerms, &RationalNode{Val: ratSum})
	}

	for _, key := range termOrder {
		entry := termMap[key]
		if entry.coeff.Sign() == 0 {
			continue // collected to 0
		}
		one := big.NewRat(1, 1)
		if entry.coeff.Cmp(one) == 0 {
			resultTerms = append(resultTerms, entry.node)
		} else {
			resultTerms = append(resultTerms, NewMul([]Node{&RationalNode{Val: entry.coeff}, entry.node}))
		}
	}

	if hasComplex && imagSum.Sign() != 0 {
		var realPart Node
		if len(resultTerms) == 0 {
			realPart = mustRational(0, 1)
		} else if len(resultTerms) == 1 {
			realPart = resultTerms[0]
		} else {
			realPart = NewAdd(resultTerms)
		}
		return NewComplex(realPart, &RationalNode{Val: imagSum}), nil
	}

	if len(resultTerms) == 0 {
		return mustRational(0, 1), nil
	}
	if len(resultTerms) == 1 {
		return resultTerms[0], nil
	}

	return NewAdd(resultTerms), nil
}

func isZero(n Node) bool {
	if r, ok := n.(*RationalNode); ok && r.Val.Sign() == 0 {
		return true
	}
	return false
}

func isZeroNode(n Node) bool {
	if n == nil {
		return false
	}
	switch v := n.(type) {
	case *RationalNode:
		return v.Val.Sign() == 0
	case *MulNode:
		for _, f := range v.Factors {
			if isZeroNode(f) {
				return true
			}
		}
	}
	return false
}

// -------------------------------------------------------------------------
// CAS: Polynomial Expansion (expand)
// -------------------------------------------------------------------------

// expandNode recursively expands an AST node using distributive laws and binomial expansion.
func expandNode(n Node) Node {
	if n == nil {
		return nil
	}
	switch v := n.(type) {
	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = expandNode(t)
		}
		res, _ := simplifyAdd(newTerms)
		return res

	case *MulNode:
		if len(v.Factors) == 0 {
			return mustRational(1, 1)
		}
		current := expandNode(v.Factors[0])
		for i := 1; i < len(v.Factors); i++ {
			next := expandNode(v.Factors[i])
			current = expandMul2(current, next)
		}
		return current

	case *PowNode:
		base := expandNode(v.Base)
		if r, ok := v.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() >= 0 {
			expInt := r.Val.Num().Int64()
			if expInt == 0 {
				return mustRational(1, 1)
			}
			if expInt == 1 {
				return base
			}
			if expInt <= 10 {
				res := base
				for i := int64(1); i < expInt; i++ {
					res = expandMul2(res, base)
				}
				return res
			}
		}
		res, err := simplifyPow(base, v.Exp)
		if err != nil {
			return &PowNode{Base: base, Exp: v.Exp}
		}
		return res

	case *UnaryOpNode:
		if v.Op == "-" {
			expanded := expandNode(v.Expr)
			res, _ := simplifyUnaryOp("-", expanded)
			return res
		}
		return v

	case *FuncNode:
		newArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			newArgs[i] = expandNode(a)
		}
		res, err := simplifyFunc(v.Name, newArgs)
		if err != nil {
			return &FuncNode{Name: v.Name, Args: newArgs}
		}
		return res

	default:
		return n
	}
}

// expandMul2 multiplies two already expanded nodes using the distributive law.
func expandMul2(a, b Node) Node {
	addA, isAddA := a.(*AddNode)
	addB, isAddB := b.(*AddNode)

	if isAddA && isAddB {
		var terms []Node
		for _, ta := range addA.Terms {
			for _, tb := range addB.Terms {
				prod, _ := simplifyMul([]Node{ta, tb})
				terms = append(terms, prod)
			}
		}
		res, _ := simplifyAdd(terms)
		return res
	} else if isAddA {
		var terms []Node
		for _, ta := range addA.Terms {
			prod, _ := simplifyMul([]Node{ta, b})
			terms = append(terms, prod)
		}
		res, _ := simplifyAdd(terms)
		return res
	} else if isAddB {
		var terms []Node
		for _, tb := range addB.Terms {
			prod, _ := simplifyMul([]Node{a, tb})
			terms = append(terms, prod)
		}
		res, _ := simplifyAdd(terms)
		return res
	} else {
		prod, _ := simplifyMul([]Node{a, b})
		return prod
	}
}

// -------------------------------------------------------------------------
// CAS: Symbolic Differentiation (diff)
// -------------------------------------------------------------------------

// containsVar checks if an AST node contains the given variable name.
func containsVar(n Node, varName string) bool {
	return ContainsVar(n, varName)
}
