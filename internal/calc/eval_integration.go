package calc

import (
	"fmt"
	"math/big"
)

// -------------------------------------------------------------------------
// Symbolic Integration (Indefinite & Definite Integration)
// -------------------------------------------------------------------------

// evalIndefiniteIntegral computes the symbolic indefinite integral of expr with respect to varName.
func evalIndefiniteIntegral(expr Node, varName string) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot integrate nil expression")
	}
	F, err := integrateCore(expr, varName)
	if err != nil {
		return nil, err
	}
	res, err := Eval(F)
	if err != nil {
		return nil, err
	}
	RecordTraceRewrite(RuleIntegrate, expr, res, fmt.Sprintf("不定積分: ∫ (%s) d%s = %s", expr.String(), varName, res.String()))
	return res, nil
}

// evalDefiniteIntegral computes the symbolic definite integral of expr from a to b with respect to varName.
func evalDefiniteIntegral(expr Node, varName string, a, b Node) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot integrate nil expression")
	}
	// 1. Compute indefinite integral F(x)
	F, err := integrateCore(expr, varName)
	if err != nil {
		return nil, err
	}

	// 2. Evaluate F(b)
	subB := Substitute(F, varName, b)
	valB, err := Eval(subB)
	if err != nil {
		return nil, fmt.Errorf("integrate: failed to evaluate upper limit %s: %w", b.String(), err)
	}

	// 3. Evaluate F(a)
	subA := Substitute(F, varName, a)
	valA, err := Eval(subA)
	if err != nil {
		return nil, fmt.Errorf("integrate: failed to evaluate lower limit %s: %w", a.String(), err)
	}

	// 4. Compute F(b) - F(a)
	negValA, err := simplifyUnaryOp("-", valA)
	if err != nil {
		return nil, err
	}
	res, err := Eval(NewAdd([]Node{valB, negValA}))
	if err != nil {
		return nil, err
	}
	RecordTraceRewrite(RuleIntegrate, expr, res, fmt.Sprintf("定積分: ∫_%s^%s (%s) d%s = [F(%s)]_%s^%s = %s", a.String(), b.String(), expr.String(), varName, varName, a.String(), b.String(), res.String()))
	return res, nil
}

// isLinear checks if n is of the form a*varName + b with a != 0.
func isLinear(n Node, varName string) (a Node, b Node, ok bool) {
	if !containsVar(n, varName) {
		return nil, nil, false
	}
	coeffs, err := extractPolyCoeffs(expandNode(n), varName)
	if err != nil {
		return nil, nil, false
	}
	maxDeg := 0
	for d := range coeffs {
		if d > maxDeg {
			maxDeg = d
		}
	}
	if maxDeg > 1 {
		return nil, nil, false
	}
	a = coeffs[1]
	if a == nil || isZero(a) {
		return nil, nil, false
	}
	b = coeffs[0]
	if b == nil {
		b = mustRational(0, 1)
	}
	return a, b, true
}

// integrateCore is the recursive symbolic integration engine.
func integrateCore(expr Node, varName string) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot integrate nil expression")
	}

	// 1. Constant with respect to varName: int(c, x) = c * x
	if !containsVar(expr, varName) {
		return simplifyMul([]Node{expr, &VarNode{Name: varName}})
	}

	// 2. Check if expression is a pure polynomial in varName
	expanded := expandNode(expr)
	if coeffs, err := extractPolyCoeffs(expanded, varName); err == nil {
		var integratedTerms []Node
		for deg, coeff := range coeffs {
			if isZero(coeff) {
				continue
			}
			newDeg := deg + 1
			// coeff / (deg + 1) * x^(deg + 1)
			invNewDeg := mustRational(1, int64(newDeg))
			termCoeff, err := simplifyMul([]Node{coeff, invNewDeg})
			if err != nil {
				return nil, err
			}
			var termPower Node
			if newDeg == 1 {
				termPower = &VarNode{Name: varName}
			} else {
				termPower = &PowNode{Base: &VarNode{Name: varName}, Exp: mustRational(int64(newDeg), 1)}
			}
			term, err := simplifyMul([]Node{termCoeff, termPower})
			if err != nil {
				return nil, err
			}
			integratedTerms = append(integratedTerms, term)
		}
		if len(integratedTerms) == 0 {
			return mustRational(0, 1), nil
		}
		if len(integratedTerms) == 1 {
			return integratedTerms[0], nil
		}
		return simplifyAdd(integratedTerms)
	}

	// 3. Pattern match by AST node type
	switch v := expr.(type) {
	case *UnaryOpNode:
		if v.Op == "-" {
			inner, err := integrateCore(v.Expr, varName)
			if err != nil {
				return nil, err
			}
			return simplifyUnaryOp("-", inner)
		}
		return nil, fmt.Errorf("integrate: unsupported unary operator %s", v.Op)

	case *AddNode:
		// Linearity of integration: int(f + g) = int(f) + int(g)
		var terms []Node
		for _, t := range v.Terms {
			it, err := integrateCore(t, varName)
			if err != nil {
				return nil, err
			}
			terms = append(terms, it)
		}
		return simplifyAdd(terms)

	case *PowNode:
		// Power rule: int((ax+b)^n) dx
		// Case A: (ax + b)^n where n is constant
		if !containsVar(v.Exp, varName) {
			if a, _, ok := isLinear(v.Base, varName); ok {
				invA, err := simplifyPow(a, mustRational(-1, 1))
				if err != nil {
					return nil, err
				}
				// If n == -1 => (1/a) * ln(|ax+b|)
				if rExp, ok := v.Exp.(*RationalNode); ok && rExp.Val.Cmp(big.NewRat(-1, 1)) == 0 {
					lnNode := &FuncNode{Name: "ln", Args: []Node{v.Base}}
					return simplifyMul([]Node{invA, lnNode})
				}
				// Otherwise: (1 / (a * (n + 1))) * (ax + b)^(n + 1)
				newExp, err := simplifyAdd([]Node{v.Exp, mustRational(1, 1)})
				if err != nil {
					return nil, err
				}
				if isZero(newExp) {
					lnNode := &FuncNode{Name: "ln", Args: []Node{v.Base}}
					return simplifyMul([]Node{invA, lnNode})
				}
				invNewExp, err := simplifyPow(newExp, mustRational(-1, 1))
				if err != nil {
					return nil, err
				}
				powNode, err := simplifyPow(v.Base, newExp)
				if err != nil {
					return nil, err
				}
				return simplifyMul([]Node{invA, invNewExp, powNode})
			}
		}

		// Case B: Exponential base^exp: int(e^(ax+b)) dx or int(k^(ax+b)) dx
		if !containsVar(v.Base, varName) {
			if a, _, ok := isLinear(v.Exp, varName); ok {
				invA, err := simplifyPow(a, mustRational(-1, 1))
				if err != nil {
					return nil, err
				}
				// Check if base is 'e'
				isE := false
				if cNode, ok := v.Base.(*ConstNode); ok && cNode.Name == "e" {
					isE = true
				} else if vNode, ok := v.Base.(*VarNode); ok && vNode.Name == "e" {
					isE = true
				}

				if isE {
					// int(e^(ax+b)) dx = (1/a) * e^(ax+b)
					return simplifyMul([]Node{invA, v})
				}
				// int(k^(ax+b)) dx = (1 / (a * ln(k))) * k^(ax+b)
				lnBase := &FuncNode{Name: "ln", Args: []Node{v.Base}}
				invLn, err := simplifyPow(lnBase, mustRational(-1, 1))
				if err != nil {
					return nil, err
				}
				return simplifyMul([]Node{invA, invLn, v})
			}
		}

	case *FuncNode:
		// Standard single-argument elementary functions
		if len(v.Args) == 1 {
			arg := v.Args[0]
			if a, _, ok := isLinear(arg, varName); ok {
				invA, err := simplifyPow(a, mustRational(-1, 1))
				if err != nil {
					return nil, err
				}

				switch v.Name {
				case "sin":
					// int(sin(ax+b)) dx = -1/a * cos(ax+b)
					cosNode := &FuncNode{Name: "cos", Args: []Node{arg}}
					negCos, err := simplifyUnaryOp("-", cosNode)
					if err != nil {
						return nil, err
					}
					return simplifyMul([]Node{invA, negCos})

				case "cos":
					// int(cos(ax+b)) dx = 1/a * sin(ax+b)
					sinNode := &FuncNode{Name: "sin", Args: []Node{arg}}
					return simplifyMul([]Node{invA, sinNode})

				case "tan":
					// int(tan(ax+b)) dx = -1/a * ln(cos(ax+b))
					cosNode := &FuncNode{Name: "cos", Args: []Node{arg}}
					lnNode := &FuncNode{Name: "ln", Args: []Node{cosNode}}
					negLn, err := simplifyUnaryOp("-", lnNode)
					if err != nil {
						return nil, err
					}
					return simplifyMul([]Node{invA, negLn})

				case "exp":
					// int(exp(ax+b)) dx = 1/a * exp(ax+b)
					return simplifyMul([]Node{invA, v})

				case "ln", "log":
					// int(ln(ax+b)) dx = 1/a * ((ax+b)*ln(ax+b) - (ax+b))
					lnNode := &FuncNode{Name: "ln", Args: []Node{arg}}
					prod, err := simplifyMul([]Node{arg, lnNode})
					if err != nil {
						return nil, err
					}
					negArg, err := simplifyUnaryOp("-", arg)
					if err != nil {
						return nil, err
					}
					inner, err := simplifyAdd([]Node{prod, negArg})
					if err != nil {
						return nil, err
					}
					return simplifyMul([]Node{invA, inner})
				}
			}
		}

	case *MulNode:
		// 1. Separate constant factors from variable factors
		var constFactors []Node
		var varFactors []Node
		for _, f := range v.Factors {
			if containsVar(f, varName) {
				varFactors = append(varFactors, f)
			} else {
				constFactors = append(constFactors, f)
			}
		}

		if len(constFactors) > 0 {
			cNode, err := simplifyMul(constFactors)
			if err != nil {
				return nil, err
			}
			var remaining Node
			if len(varFactors) == 0 {
				remaining = mustRational(1, 1)
			} else if len(varFactors) == 1 {
				remaining = varFactors[0]
			} else {
				remaining = NewMul(varFactors)
			}
			res, err := integrateCore(remaining, varName)
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{cNode, res})
		}

		// 2. Check for polynomial * elementary function (Integration by Parts)
		// int(u * v') dx = u * v - int(u' * v) dx
		if len(varFactors) == 2 {
			// Try both permutations (f0 as polynomial u, f1 as v', and vice versa)
			f0, f1 := varFactors[0], varFactors[1]
			if res, ok := tryIntegrationByParts(f0, f1, varName); ok {
				return res, nil
			}
			if res, ok := tryIntegrationByParts(f1, f0, varName); ok {
				return res, nil
			}
		}
	}

	return nil, fmt.Errorf("integrate: symbolic integration not supported for %s", expr.String())
}

// tryIntegrationByParts attempts integration by parts with u = poly, and vPrime = elementary func.
func tryIntegrationByParts(uCandidate, vPrimeCandidate Node, varName string) (Node, bool) {
	// uCandidate must be a polynomial of degree >= 1
	coeffs, err := extractPolyCoeffs(expandNode(uCandidate), varName)
	if err != nil {
		return nil, false
	}
	maxDeg := 0
	for d := range coeffs {
		if d > maxDeg {
			maxDeg = d
		}
	}
	if maxDeg < 1 {
		return nil, false
	}

	u := uCandidate

	// vPrimeCandidate must be integrable without triggering an infinite loop
	// It should be sin, cos, exp, or pow (like e^(ax+b))
	var v Node
	switch vp := vPrimeCandidate.(type) {
	case *FuncNode:
		switch vp.Name {
		case "sin", "cos", "exp":
			v, err = integrateCore(vp, varName)
			if err != nil {
				return nil, false
			}
		case "ln", "log":
			// For ln: choose u = ln, v' = poly!
			// This case is handled when candidates are swapped, but let's be explicit:
			return nil, false
		}
	case *PowNode:
		if !containsVar(vp.Base, varName) {
			v, err = integrateCore(vp, varName)
			if err != nil {
				return nil, false
			}
		}
	default:
		return nil, false
	}

	if v == nil {
		return nil, false
	}

	// 1. u * v
	uv, err := simplifyMul([]Node{u, v})
	if err != nil {
		return nil, false
	}

	// 2. u' = d(u)/dx
	uPrime, err := differentiate(u, varName)
	if err != nil {
		return nil, false
	}

	// 3. int(u' * v) dx
	uPrimeV, err := simplifyMul([]Node{uPrime, v})
	if err != nil {
		return nil, false
	}

	intUPrimeV, err := integrateCore(uPrimeV, varName)
	if err != nil {
		return nil, false
	}

	// 4. u * v - int(u' * v)
	negInt, err := simplifyUnaryOp("-", intUPrimeV)
	if err != nil {
		return nil, false
	}

	res, err := simplifyAdd([]Node{uv, negInt})
	if err != nil {
		return nil, false
	}
	return res, true
}
