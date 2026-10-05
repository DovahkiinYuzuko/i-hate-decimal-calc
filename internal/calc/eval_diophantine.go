package calc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// PellSolve calculates the fundamental positive integer solution (x_1, y_1)
// to Pell's equation x^2 - D * y^2 = 1 using the simple continued fraction expansion of sqrt(D).
func PellSolve(d *big.Int) (*big.Int, *big.Int, error) {
	if d == nil || d.Sign() <= 0 {
		valStr := "<nil>"
		if d != nil {
			valStr = d.String()
		}
		return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_pell_d_must_be_positive_integer", valStr))
	}

	a0 := new(big.Int).Sqrt(d)
	a0Sq := new(big.Int).Mul(a0, a0)
	if a0Sq.Cmp(d) == 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_pell_d_is_square", d.String()))
	}

	m := big.NewInt(0)
	denom := big.NewInt(1)
	a := new(big.Int).Set(a0)

	pPrev := big.NewInt(1)
	pCur := new(big.Int).Set(a0)
	qPrev := big.NewInt(0)
	qCur := big.NewInt(1)

	// Continued fraction expansion loop
	for iter := 0; iter < 500000; iter++ {
		pSq := new(big.Int).Mul(pCur, pCur)
		dqSq := new(big.Int).Mul(d, new(big.Int).Mul(qCur, qCur))
		diff := new(big.Int).Sub(pSq, dqSq)

		if diff.Cmp(big.NewInt(1)) == 0 {
			return pCur, qCur, nil
		}

		mNext := new(big.Int).Sub(new(big.Int).Mul(denom, a), m)
		mNextSq := new(big.Int).Mul(mNext, mNext)
		dNext := new(big.Int).Div(new(big.Int).Sub(d, mNextSq), denom)
		aNext := new(big.Int).Div(new(big.Int).Add(a0, mNext), dNext)

		m.Set(mNext)
		denom.Set(dNext)
		a.Set(aNext)

		pNext := new(big.Int).Add(new(big.Int).Mul(a, pCur), pPrev)
		pPrev.Set(pCur)
		pCur.Set(pNext)

		qNext := new(big.Int).Add(new(big.Int).Mul(a, qCur), qPrev)
		qPrev.Set(qCur)
		qCur.Set(qNext)
	}

	return nil, nil, fmt.Errorf("Pell's equation search limit reached for D = %s", d.String())
}

// PellNegativeSolve calculates the fundamental positive integer solution (x_1, y_1)
// to the negative Pell's equation x^2 - D * y^2 = -1 if solvable.
func PellNegativeSolve(d *big.Int) (*big.Int, *big.Int, error) {
	if d == nil || d.Sign() <= 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_pell_d_must_be_positive_integer", d.String()))
	}

	a0 := new(big.Int).Sqrt(d)
	a0Sq := new(big.Int).Mul(a0, a0)
	if a0Sq.Cmp(d) == 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_pell_d_is_square", d.String()))
	}

	m := big.NewInt(0)
	denom := big.NewInt(1)
	a := new(big.Int).Set(a0)

	pPrev := big.NewInt(1)
	pCur := new(big.Int).Set(a0)
	qPrev := big.NewInt(0)
	qCur := big.NewInt(1)

	for iter := 0; iter < 500000; iter++ {
		pSq := new(big.Int).Mul(pCur, pCur)
		dqSq := new(big.Int).Mul(d, new(big.Int).Mul(qCur, qCur))
		diff := new(big.Int).Sub(pSq, dqSq)

		if diff.Cmp(big.NewInt(-1)) == 0 {
			return pCur, qCur, nil
		}
		if diff.Cmp(big.NewInt(1)) == 0 {
			// Fundamental solution to positive Pell encountered first without negative solution:
			// Negative Pell equation has no integer solutions for this D.
			return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_no_integer_solution", fmt.Sprintf("x^2 - %s*y^2 = -1", d.String())))
		}

		mNext := new(big.Int).Sub(new(big.Int).Mul(denom, a), m)
		mNextSq := new(big.Int).Mul(mNext, mNext)
		dNext := new(big.Int).Div(new(big.Int).Sub(d, mNextSq), denom)
		aNext := new(big.Int).Div(new(big.Int).Add(a0, mNext), dNext)

		m.Set(mNext)
		denom.Set(dNext)
		a.Set(aNext)

		pNext := new(big.Int).Add(new(big.Int).Mul(a, pCur), pPrev)
		pPrev.Set(pCur)
		pCur.Set(pNext)

		qNext := new(big.Int).Add(new(big.Int).Mul(a, qCur), qPrev)
		qPrev.Set(qCur)
		qCur.Set(qNext)
	}

	return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_no_integer_solution", fmt.Sprintf("x^2 - %s*y^2 = -1", d.String())))
}

// SolveLinearDiophantineSingle computes the general integer solution to a*x + b*y = c.
// Returns [x == x_p + (b/g)*t, y == y_p - (a/g)*t].
func SolveLinearDiophantineSingle(a, b, c *big.Int, varX, varY, paramT string) (Node, Node, error) {
	if a.Sign() == 0 && b.Sign() == 0 {
		if c.Sign() == 0 {
			// 0 = 0: All integers are solutions
			return &RelOpNode{Op: "==", LHS: &VarNode{Name: varX}, RHS: &VarNode{Name: varX}},
				&RelOpNode{Op: "==", LHS: &VarNode{Name: varY}, RHS: &VarNode{Name: varY}}, nil
		}
		return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_no_integer_solution", fmt.Sprintf("0 = %s", c.String())))
	}

	if a.Sign() == 0 {
		// b*y = c
		rem := new(big.Int).Rem(c, b)
		if rem.Sign() != 0 {
			return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_no_integer_solution", fmt.Sprintf("%s*%s = %s", b.String(), varY, c.String())))
		}
		ySol := new(big.Int).Quo(c, b)
		return &RelOpNode{Op: "==", LHS: &VarNode{Name: varX}, RHS: &VarNode{Name: paramT}},
			&RelOpNode{Op: "==", LHS: &VarNode{Name: varY}, RHS: NewRationalFromBigRat(new(big.Rat).SetInt(ySol))}, nil
	}

	if b.Sign() == 0 {
		// a*x = c
		rem := new(big.Int).Rem(c, a)
		if rem.Sign() != 0 {
			return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_no_integer_solution", fmt.Sprintf("%s*%s = %s", a.String(), varX, c.String())))
		}
		xSol := new(big.Int).Quo(c, a)
		return &RelOpNode{Op: "==", LHS: &VarNode{Name: varX}, RHS: NewRationalFromBigRat(new(big.Rat).SetInt(xSol))},
			&RelOpNode{Op: "==", LHS: &VarNode{Name: varY}, RHS: &VarNode{Name: paramT}}, nil
	}

	g := new(big.Int)
	x0 := new(big.Int)
	y0 := new(big.Int)
	g.GCD(x0, y0, a, b)

	rem := new(big.Int).Rem(c, g)
	if rem.Sign() != 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("diophantine.err_no_integer_solution", fmt.Sprintf("%s*%s + %s*%s = %s", a.String(), varX, b.String(), varY, c.String())))
	}

	factor := new(big.Int).Quo(c, g)
	xp := new(big.Int).Mul(x0, factor)
	yp := new(big.Int).Mul(y0, factor)

	stepX := new(big.Int).Quo(b, g)
	stepY := new(big.Int).Quo(a, g)
	stepY.Neg(stepY)

	// Construct x(t) = xp + stepX * t
	var xTerms []Node
	if xp.Sign() != 0 {
		xTerms = append(xTerms, NewRationalFromBigRat(new(big.Rat).SetInt(xp)))
	}
	if stepX.Sign() != 0 {
		if stepX.Cmp(big.NewInt(1)) == 0 {
			xTerms = append(xTerms, &VarNode{Name: paramT})
		} else if stepX.Cmp(big.NewInt(-1)) == 0 {
			negT, _ := simplifyUnaryOp("-", &VarNode{Name: paramT})
			xTerms = append(xTerms, negT)
		} else {
			xTerms = append(xTerms, &MulNode{Factors: []Node{NewRationalFromBigRat(new(big.Rat).SetInt(stepX)), &VarNode{Name: paramT}}})
		}
	}
	var xExpr Node
	if len(xTerms) == 0 {
		xExpr = mustRational(0, 1)
	} else if len(xTerms) == 1 {
		xExpr = xTerms[0]
	} else {
		xExpr, _ = simplifyAdd(xTerms)
	}

	// Construct y(t) = yp + stepY * t
	var yTerms []Node
	if yp.Sign() != 0 {
		yTerms = append(yTerms, NewRationalFromBigRat(new(big.Rat).SetInt(yp)))
	}
	if stepY.Sign() != 0 {
		if stepY.Cmp(big.NewInt(1)) == 0 {
			yTerms = append(yTerms, &VarNode{Name: paramT})
		} else if stepY.Cmp(big.NewInt(-1)) == 0 {
			negT, _ := simplifyUnaryOp("-", &VarNode{Name: paramT})
			yTerms = append(yTerms, negT)
		} else {
			yTerms = append(yTerms, &MulNode{Factors: []Node{NewRationalFromBigRat(new(big.Rat).SetInt(stepY)), &VarNode{Name: paramT}}})
		}
	}
	var yExpr Node
	if len(yTerms) == 0 {
		yExpr = mustRational(0, 1)
	} else if len(yTerms) == 1 {
		yExpr = yTerms[0]
	} else {
		yExpr, _ = simplifyAdd(yTerms)
	}

	return &RelOpNode{Op: "==", LHS: &VarNode{Name: varX}, RHS: xExpr},
		&RelOpNode{Op: "==", LHS: &VarNode{Name: varY}, RHS: yExpr}, nil
}

// SolvePythagorean constructs parametric formulas for x^2 + y^2 = z^2:
// x = k*(m^2 - n^2), y = 2*k*m*n, z = k*(m^2 + n^2).
func SolvePythagorean(varX, varY, varZ string) (Node, error) {
	k := &VarNode{Name: "k"}
	m := &VarNode{Name: "m"}
	n := &VarNode{Name: "n"}

	mSq := &PowNode{Base: m, Exp: mustRational(2, 1)}
	nSq := &PowNode{Base: n, Exp: mustRational(2, 1)}
	negNSq, _ := simplifyUnaryOp("-", nSq)

	mSqMinusNSq, _ := simplifyAdd([]Node{mSq, negNSq})
	mSqPlusNSq, _ := simplifyAdd([]Node{mSq, nSq})

	twoKMN, _ := simplifyMul([]Node{mustRational(2, 1), k, m, n})
	xExpr, _ := simplifyMul([]Node{k, mSqMinusNSq})
	zExpr, _ := simplifyMul([]Node{k, mSqPlusNSq})

	res := &ListNode{Elements: []Node{
		&RelOpNode{Op: "==", LHS: &VarNode{Name: varX}, RHS: xExpr},
		&RelOpNode{Op: "==", LHS: &VarNode{Name: varY}, RHS: twoKMN},
		&RelOpNode{Op: "==", LHS: &VarNode{Name: varZ}, RHS: zExpr},
	}}
	return res, nil
}

// extractPolynomialCoeffs expands the expression and extracts coefficients for monomials in terms of target variables.
func extractPolynomialCoeffs(expr Node, vars []string) (map[string]*big.Int, *big.Int, error) {
	expanded := expandNode(expr)
	if expanded == nil {
		expanded = expr
	}

	var terms []Node
	if addNode, ok := expanded.(*AddNode); ok {
		terms = addNode.Terms
	} else {
		terms = []Node{expanded}
	}

	monomials := make(map[string]*big.Int)
	constTerm := big.NewInt(0)

	for _, term := range terms {
		t := EvalOrSelf(term)
		coeff := big.NewInt(1)
		degMap := make(map[string]int)

		factors := []Node{t}
		if mulNode, ok := t.(*MulNode); ok {
			factors = mulNode.Factors
		} else if unOp, ok := t.(*UnaryOpNode); ok && unOp.Op == "-" {
			coeff.SetInt64(-1)
			factors = []Node{unOp.Expr}
			if subMul, isMul := unOp.Expr.(*MulNode); isMul {
				factors = subMul.Factors
			}
		}

		for _, f := range factors {
			fEval := EvalOrSelf(f)
			if rat, ok := fEval.(*RationalNode); ok {
				if !rat.Val.IsInt() {
					return nil, nil, fmt.Errorf("non-integer coefficient: %s", rat.String())
				}
				coeff.Mul(coeff, rat.Val.Num())
			} else if sym, ok := fEval.(*VarNode); ok {
				degMap[sym.Name]++
			} else if pow, ok := fEval.(*PowNode); ok {
				if sym, isSym := pow.Base.(*VarNode); isSym {
					if expRat, isExp := pow.Exp.(*RationalNode); isExp && expRat.Val.IsInt() {
						degMap[sym.Name] += int(expRat.Val.Num().Int64())
					} else {
						return nil, nil, fmt.Errorf("non-integer exponent in variable %s", sym.Name)
					}
				} else {
					return nil, nil, fmt.Errorf("unsupported power base: %s", pow.Base.String())
				}
			} else if unOp, ok := fEval.(*UnaryOpNode); ok && unOp.Op == "-" {
				coeff.Neg(coeff)
				if sym, isSym := unOp.Expr.(*VarNode); isSym {
					degMap[sym.Name]++
				}
			}
		}

		var parts []string
		for _, v := range vars {
			deg := degMap[v]
			if deg > 0 {
				if deg == 1 {
					parts = append(parts, v)
				} else {
					parts = append(parts, fmt.Sprintf("%s^%d", v, deg))
				}
			}
		}

		if len(parts) == 0 {
			constTerm.Add(constTerm, coeff)
		} else {
			key := strings.Join(parts, "*")
			if monomials[key] == nil {
				monomials[key] = big.NewInt(0)
			}
			monomials[key].Add(monomials[key], coeff)
		}
	}

	return monomials, constTerm, nil
}

// EvalPellSolve implements the `pell_solve(D)` CAS function.
func EvalPellSolve(args []Node) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("diophantine.err_pell_solve_args"))
	}

	evaled := EvalOrSelf(args[0])
	rat, ok := evaled.(*RationalNode)
	if !ok || !rat.Val.IsInt() || rat.Val.Num().Sign() <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("diophantine.err_pell_d_must_be_positive_integer", evaled.String()))
	}

	d := rat.Val.Num()
	x1, y1, err := PellSolve(d)
	if err != nil {
		return nil, err
	}

	xNode := NewRationalFromBigRat(new(big.Rat).SetInt(x1))
	yNode := NewRationalFromBigRat(new(big.Rat).SetInt(y1))

	return &ListNode{Elements: []Node{xNode, yNode}}, nil
}

// EvalSolveDiophantine implements the `solve_diophantine(eq [, [vars]])` CAS function.
func EvalSolveDiophantine(args []Node) (Node, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s", i18n.T("diophantine.err_solve_diophantine_args"))
	}

	eqNode := EvalOrSelf(args[0])
	var diffExpr Node

	if rel, ok := eqNode.(*RelOpNode); ok && (rel.Op == "==" || rel.Op == "=") {
		negRHS, err := simplifyUnaryOp("-", rel.RHS)
		if err != nil {
			return nil, err
		}
		diffExpr, err = simplifyAdd([]Node{rel.LHS, negRHS})
		if err != nil {
			return nil, err
		}
	} else {
		diffExpr = eqNode
	}

	var targetVars []string
	if len(args) == 2 {
		varsNode := EvalOrSelf(args[1])
		if list, ok := varsNode.(*ListNode); ok {
			for _, elem := range list.Elements {
				if sym, isSym := elem.(*VarNode); isSym {
					targetVars = append(targetVars, sym.Name)
				}
			}
		} else if sym, ok := varsNode.(*VarNode); ok {
			targetVars = []string{sym.Name}
		}
	} else {
		free := CollectFreeVariables(diffExpr)
		for _, v := range free {
			targetVars = append(targetVars, v)
		}
	}

	fsm := NewDiophantineSolverFSM()
	_ = fsm.Transition(DiophantineStateClassify)

	coeffs, constTerm, err := extractPolynomialCoeffs(diffExpr, targetVars)
	if err != nil {
		_ = fsm.Transition(DiophantineStateFailure)
		return nil, fmt.Errorf("%s: %v", i18n.T("diophantine.err_unsupported_equation_type", eqNode.String()), err)
	}

	// 1. Check for Linear Diophantine Equation in 2 variables: a*x + b*y + c = 0
	if len(targetVars) == 2 {
		v1 := targetVars[0]
		v2 := targetVars[1]
		c1 := coeffs[v1]
		c2 := coeffs[v2]

		isLinear := true
		for k := range coeffs {
			if k != v1 && k != v2 {
				isLinear = false
				break
			}
		}

		if isLinear && (c1 != nil || c2 != nil) {
			_ = fsm.Transition(DiophantineStateSolveLinear)
			aCoeff := big.NewInt(0)
			bCoeff := big.NewInt(0)
			if c1 != nil {
				aCoeff.Set(c1)
			}
			if c2 != nil {
				bCoeff.Set(c2)
			}
			// equation is a*x + b*y + constTerm = 0  => a*x + b*y = -constTerm
			cVal := new(big.Int).Neg(constTerm)

			xSol, ySol, err := SolveLinearDiophantineSingle(aCoeff, bCoeff, cVal, v1, v2, "t")
			if err != nil {
				_ = fsm.Transition(DiophantineStateFailure)
				return nil, err
			}
			_ = fsm.Transition(DiophantineStateSuccess)
			return &ListNode{Elements: []Node{xSol, ySol}}, nil
		}

		// 2. Check for Pell's Equation: x^2 - D*y^2 - 1 = 0  or  x^2 - D*y^2 + 1 = 0
		xSqKey := fmt.Sprintf("%s^2", v1)
		ySqKey := fmt.Sprintf("%s^2", v2)
		cXSq := coeffs[xSqKey]
		cYSq := coeffs[ySqKey]

		if cXSq == nil || cYSq == nil {
			// Try reversed order: v2^2 - D*v1^2
			v1, v2 = v2, v1
			xSqKey = fmt.Sprintf("%s^2", v1)
			ySqKey = fmt.Sprintf("%s^2", v2)
			cXSq = coeffs[xSqKey]
			cYSq = coeffs[ySqKey]
		}

		isPellCandidate := cXSq != nil && cYSq != nil && len(coeffs) == 2
		if isPellCandidate {
			// Normal form: cXSq * x^2 + cYSq * y^2 + constTerm = 0
			// Pell: x^2 - D*y^2 = 1  =>  x^2 - D*y^2 - 1 = 0 (cXSq=1, cYSq=-D, constTerm=-1)
			// Negative Pell: x^2 - D*y^2 = -1 => x^2 - D*y^2 + 1 = 0 (cXSq=1, cYSq=-D, constTerm=1)
			if cXSq.Cmp(big.NewInt(-1)) == 0 && cYSq.Sign() > 0 {
				// -x^2 + D*y^2 + const = 0  => Multiply by -1
				cXSq = big.NewInt(1)
				cYSq = new(big.Int).Neg(cYSq)
				constTerm = new(big.Int).Neg(constTerm)
			}

			if cXSq.Cmp(big.NewInt(1)) == 0 && cYSq.Sign() < 0 {
				_ = fsm.Transition(DiophantineStateSolvePell)
				dVal := new(big.Int).Neg(cYSq)

				if constTerm.Cmp(big.NewInt(-1)) == 0 {
					// x^2 - D*y^2 = 1
					x1, y1, err := PellSolve(dVal)
					if err != nil {
						_ = fsm.Transition(DiophantineStateFailure)
						return nil, err
					}
					_ = fsm.Transition(DiophantineStateSuccess)
					return &ListNode{Elements: []Node{
						&RelOpNode{Op: "==", LHS: &VarNode{Name: v1}, RHS: NewRationalFromBigRat(new(big.Rat).SetInt(x1))},
						&RelOpNode{Op: "==", LHS: &VarNode{Name: v2}, RHS: NewRationalFromBigRat(new(big.Rat).SetInt(y1))},
					}}, nil
				} else if constTerm.Cmp(big.NewInt(1)) == 0 {
					// x^2 - D*y^2 = -1
					x1, y1, err := PellNegativeSolve(dVal)
					if err != nil {
						_ = fsm.Transition(DiophantineStateFailure)
						return nil, err
					}
					_ = fsm.Transition(DiophantineStateSuccess)
					return &ListNode{Elements: []Node{
						&RelOpNode{Op: "==", LHS: &VarNode{Name: v1}, RHS: NewRationalFromBigRat(new(big.Rat).SetInt(x1))},
						&RelOpNode{Op: "==", LHS: &VarNode{Name: v2}, RHS: NewRationalFromBigRat(new(big.Rat).SetInt(y1))},
					}}, nil
				}
			}
		}
	}

	// 3. Check for Pythagorean Equation in 3 variables: x^2 + y^2 - z^2 = 0
	if len(targetVars) == 3 && constTerm.Sign() == 0 && len(coeffs) == 3 {
		v1, v2, v3 := targetVars[0], targetVars[1], targetVars[2]
		k1 := fmt.Sprintf("%s^2", v1)
		k2 := fmt.Sprintf("%s^2", v2)
		k3 := fmt.Sprintf("%s^2", v3)

		c1, c2, c3 := coeffs[k1], coeffs[k2], coeffs[k3]
		if c1 != nil && c2 != nil && c3 != nil {
			// Find which term has opposite sign
			if c1.Cmp(big.NewInt(1)) == 0 && c2.Cmp(big.NewInt(1)) == 0 && c3.Cmp(big.NewInt(-1)) == 0 {
				_ = fsm.Transition(DiophantineStateSolvePythagorean)
				sol, err := SolvePythagorean(v1, v2, v3)
				if err != nil {
					_ = fsm.Transition(DiophantineStateFailure)
					return nil, err
				}
				_ = fsm.Transition(DiophantineStateSuccess)
				return sol, nil
			} else if c1.Cmp(big.NewInt(1)) == 0 && c3.Cmp(big.NewInt(1)) == 0 && c2.Cmp(big.NewInt(-1)) == 0 {
				_ = fsm.Transition(DiophantineStateSolvePythagorean)
				sol, err := SolvePythagorean(v1, v3, v2)
				if err != nil {
					_ = fsm.Transition(DiophantineStateFailure)
					return nil, err
				}
				_ = fsm.Transition(DiophantineStateSuccess)
				return sol, nil
			} else if c2.Cmp(big.NewInt(1)) == 0 && c3.Cmp(big.NewInt(1)) == 0 && c1.Cmp(big.NewInt(-1)) == 0 {
				_ = fsm.Transition(DiophantineStateSolvePythagorean)
				sol, err := SolvePythagorean(v2, v3, v1)
				if err != nil {
					_ = fsm.Transition(DiophantineStateFailure)
					return nil, err
				}
				_ = fsm.Transition(DiophantineStateSuccess)
				return sol, nil
			}
		}
	}

	_ = fsm.Transition(DiophantineStateFailure)
	return nil, fmt.Errorf("%s", i18n.T("diophantine.err_unsupported_equation_type", eqNode.String()))
}

func init() {
	RegisterHandler("solve_diophantine", func(args []Node, env *Env) (Node, error) {
		return EvalSolveDiophantine(args)
	})
	RegisterHandler("pell_solve", func(args []Node, env *Env) (Node, error) {
		return EvalPellSolve(args)
	})
}

