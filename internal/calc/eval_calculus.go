package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"math/big"
)

// -------------------------------------------------------------------------
// Symbolic Calculus (Differentiation, Equation Solving, Series & Sums)
// -------------------------------------------------------------------------
// differentiate computes the exact symbolic derivative of node n with respect to varName.
func differentiate(n Node, varName string) (Node, error) {
	if n == nil {
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_cannot_differentiate_nil_node"))
	}

	// If n does not contain varName, derivative is 0
	if !containsVar(n, varName) {
		return mustRational(0, 1), nil
	}

	switch v := n.(type) {
	case *VarNode:
		if v.Name == varName {
			return mustRational(1, 1), nil
		}
		return mustRational(0, 1), nil

	case *AddNode:
		dTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			dt, err := differentiate(t, varName)
			if err != nil {
				return nil, err
			}
			dTerms[i] = dt
		}
		return simplifyAdd(dTerms)

	case *UnaryOpNode:
		if v.Op == "-" {
			de, err := differentiate(v.Expr, varName)
			if err != nil {
				return nil, err
			}
			return simplifyUnaryOp("-", de)
		}
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_cannot_differentiate_unary_operator", v.Op))

	case *MulNode:
		// Product rule: d(f_1 * ... * f_k) = sum_i (f_i' * prod_{j != i} f_j)
		var sumTerms []Node
		for i, fi := range v.Factors {
			dfi, err := differentiate(fi, varName)
			if err != nil {
				return nil, err
			}
			if isZero(dfi) {
				continue
			}
			var prodFactors []Node
			prodFactors = append(prodFactors, dfi)
			for j, fj := range v.Factors {
				if j != i {
					prodFactors = append(prodFactors, fj)
				}
			}
			p, err := simplifyMul(prodFactors)
			if err != nil {
				return nil, err
			}
			sumTerms = append(sumTerms, p)
		}
		if len(sumTerms) == 0 {
			return mustRational(0, 1), nil
		}
		return simplifyAdd(sumTerms)

	case *PowNode:
		baseHas := containsVar(v.Base, varName)
		expHas := containsVar(v.Exp, varName)

		if baseHas && !expHas {
			// d/dx [ u(x)^n ] = n * u(x)^(n-1) * u'(x)
			du, err := differentiate(v.Base, varName)
			if err != nil {
				return nil, err
			}
			expMinus1, err := simplifyAdd([]Node{v.Exp, mustRational(-1, 1)})
			if err != nil {
				return nil, err
			}
			uPow, err := simplifyPow(v.Base, expMinus1)
			if err != nil {
				return nil, err
			}
			res, err := simplifyMul([]Node{v.Exp, uPow, du})
			if err != nil {
				return nil, err
			}
			RecordTraceRewrite(RuleDiffPower, v, res, fmt.Sprintf("べき乗の微分公式: d/d%s [%s^%s] = %s", varName, Format(v.Base), Format(v.Exp), Format(res)))
			return res, nil
		} else if !baseHas && expHas {
			// d/dx [ a^v(x) ] = a^v(x) * ln(a) * v'(x)
			dv, err := differentiate(v.Exp, varName)
			if err != nil {
				return nil, err
			}
			var lnBase Node
			if c, ok := v.Base.(*ConstNode); ok && c.Name == "e" {
				lnBase = mustRational(1, 1)
			} else {
				lnBase, err = simplifyFunc("ln", []Node{v.Base})
				if err != nil {
					return nil, err
				}
			}
			return simplifyMul([]Node{v, lnBase, dv})
		} else {
			// General form: d/dx [ u^v ] = u^v * ( v' * ln(u) + v * u'/u )
			du, err := differentiate(v.Base, varName)
			if err != nil {
				return nil, err
			}
			dv, err := differentiate(v.Exp, varName)
			if err != nil {
				return nil, err
			}
			lnU, err := simplifyFunc("ln", []Node{v.Base})
			if err != nil {
				return nil, err
			}
			term1, err := simplifyMul([]Node{dv, lnU})
			if err != nil {
				return nil, err
			}
			uInv, err := simplifyPow(v.Base, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			term2, err := simplifyMul([]Node{v.Exp, du, uInv})
			if err != nil {
				return nil, err
			}
			bracket, err := simplifyAdd([]Node{term1, term2})
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{v, bracket})
		}

	case *SqrtNode:
		// sqrt(u) = u^(1/2) -> d/dx = 1/2 * u' * sqrt(u)^-1
		du, err := differentiate(v.Radicand, varName)
		if err != nil {
			return nil, err
		}
		invSqrt, err := simplifyPow(v, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		return simplifyMul([]Node{mustRational(1, 2), du, invSqrt})

	case *FuncNode:
		if len(v.Args) == 1 {
			u := v.Args[0]
			du, err := differentiate(u, varName)
			if err != nil {
				return nil, err
			}
			var dfDu Node
			switch v.Name {
			case "sin":
				dfDu, err = simplifyFunc("cos", []Node{u})
			case "cos":
				sinU, err2 := simplifyFunc("sin", []Node{u})
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyUnaryOp("-", sinU)
			case "tan":
				cosU, err2 := simplifyFunc("cos", []Node{u})
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyPow(cosU, mustRational(-2, 1))
			case "exp":
				dfDu = v
			case "ln":
				dfDu, err = simplifyPow(u, mustRational(-1, 1))
			case "log":
				uInv, err2 := simplifyPow(u, mustRational(-1, 1))
				if err2 != nil {
					return nil, err2
				}
				ln10, err2 := simplifyFunc("ln", []Node{mustRational(10, 1)})
				if err2 != nil {
					return nil, err2
				}
				ln10Inv, err2 := simplifyPow(ln10, mustRational(-1, 1))
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyMul([]Node{uInv, ln10Inv})
			case "asin":
				u2, err2 := simplifyPow(u, mustRational(2, 1))
				if err2 != nil {
					return nil, err2
				}
				negU2, err2 := simplifyUnaryOp("-", u2)
				if err2 != nil {
					return nil, err2
				}
				oneMinusU2, err2 := simplifyAdd([]Node{mustRational(1, 1), negU2})
				if err2 != nil {
					return nil, err2
				}
				sqrtVal, err2 := simplifySqrt(oneMinusU2)
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyPow(sqrtVal, mustRational(-1, 1))
			case "acos":
				u2, err2 := simplifyPow(u, mustRational(2, 1))
				if err2 != nil {
					return nil, err2
				}
				negU2, err2 := simplifyUnaryOp("-", u2)
				if err2 != nil {
					return nil, err2
				}
				oneMinusU2, err2 := simplifyAdd([]Node{mustRational(1, 1), negU2})
				if err2 != nil {
					return nil, err2
				}
				sqrtVal, err2 := simplifySqrt(oneMinusU2)
				if err2 != nil {
					return nil, err2
				}
				posInv, err2 := simplifyPow(sqrtVal, mustRational(-1, 1))
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyUnaryOp("-", posInv)
			case "atan":
				u2, err2 := simplifyPow(u, mustRational(2, 1))
				if err2 != nil {
					return nil, err2
				}
				onePlusU2, err2 := simplifyAdd([]Node{mustRational(1, 1), u2})
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyPow(onePlusU2, mustRational(-1, 1))
			case "cbrt":
				uPow, err2 := simplifyPow(u, mustRational(-2, 3))
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyMul([]Node{mustRational(1, 3), uPow})
			case "abs":
				absU, err2 := simplifyFunc("abs", []Node{u})
				if err2 != nil {
					return nil, err2
				}
				absUInv, err2 := simplifyPow(absU, mustRational(-1, 1))
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyMul([]Node{u, absUInv})
			default:
				return nil, fmt.Errorf("%s", i18n.T("calculus.err_differentiation_of_function", v.Name))
			}
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{dfDu, du})
		} else if v.Name == "log" && len(v.Args) == 2 {
			if containsVar(v.Args[0], varName) {
				return nil, fmt.Errorf("%s", i18n.T("calculus.err_differentiation_of_log_with_variable"))
			}
			u := v.Args[1]
			du, err := differentiate(u, varName)
			if err != nil {
				return nil, err
			}
			uInv, err := simplifyPow(u, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			lnBase, err := simplifyFunc("ln", []Node{v.Args[0]})
			if err != nil {
				return nil, err
			}
			lnBaseInv, err := simplifyPow(lnBase, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{uInv, lnBaseInv, du})
		}
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_cannot_differentiate_function", v.Name, len(v.Args)))

	default:
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_cannot_differentiate_node_of_type", n))
	}
}

// -------------------------------------------------------------------------
// CAS: Polynomial Equation Solving (solve)
// -------------------------------------------------------------------------

// extractPolyCoeffs extracts polynomial coefficients a_k of expr with respect to varName.
func extractPolyCoeffs(expr Node, varName string) (map[int]Node, error) {
	coeffs := make(map[int]Node)

	addCoeff := func(deg int, coeff Node) error {
		if existing, ok := coeffs[deg]; ok {
			sum, err := simplifyAdd([]Node{existing, coeff})
			if err != nil {
				return err
			}
			coeffs[deg] = sum
		} else {
			coeffs[deg] = coeff
		}
		return nil
	}

	var processTerm func(t Node) error
	processTerm = func(t Node) error {
		if !containsVar(t, varName) {
			return addCoeff(0, t)
		}
		if v, ok := t.(*VarNode); ok && v.Name == varName {
			return addCoeff(1, mustRational(1, 1))
		}
		if pow, ok := t.(*PowNode); ok {
			if v, ok := pow.Base.(*VarNode); ok && v.Name == varName {
				if r, ok := pow.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() > 0 {
					return addCoeff(int(r.Val.Num().Int64()), mustRational(1, 1))
				}
			}
			return fmt.Errorf("%s", i18n.T("calculus.err_solve_error_non_polynomial_exponent", t.String()))
		}
		if mul, ok := t.(*MulNode); ok {
			var varFactor Node
			var otherFactors []Node
			varFound := false
			for _, f := range mul.Factors {
				if containsVar(f, varName) {
					if varFound {
						return fmt.Errorf("%s", i18n.T("calculus.err_solve_error_multiple_variable_factors", t.String()))
					}
					varFactor = f
					varFound = true
				} else {
					otherFactors = append(otherFactors, f)
				}
			}
			coeffNode, err := simplifyMul(otherFactors)
			if err != nil {
				return err
			}
			if v, ok := varFactor.(*VarNode); ok && v.Name == varName {
				return addCoeff(1, coeffNode)
			}
			if pow, ok := varFactor.(*PowNode); ok {
				if v, ok := pow.Base.(*VarNode); ok && v.Name == varName {
					if r, ok := pow.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() > 0 {
						return addCoeff(int(r.Val.Num().Int64()), coeffNode)
					}
				}
			}
			return fmt.Errorf("%s", i18n.T("calculus.err_solve_error_non_polynomial_factor", t.String()))
		}
		return fmt.Errorf("%s", i18n.T("calculus.err_solve_error_non_polynomial_term", t.String()))
	}

	if add, ok := expr.(*AddNode); ok {
		for _, t := range add.Terms {
			if err := processTerm(t); err != nil {
				return nil, err
			}
		}
	} else {
		if err := processTerm(expr); err != nil {
			return nil, err
		}
	}

	// Clean zero coefficients
	for d, c := range coeffs {
		if isZero(c) {
			delete(coeffs, d)
		}
	}

	return coeffs, nil
}

// solveEquation algebraically solves expr = 0 for varName.
func solveEquation(expr Node, varName string) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_cannot_solve_nil_equation"))
	}

	// 1. Expand the expression to standard form
	expanded := expandNode(expr)

	// Check if varName is in the equation
	if !containsVar(expanded, varName) {
		if isZero(expanded) {
			return nil, fmt.Errorf("%s", i18n.T("calculus.err_solve_identity_equation_infinite_solutions"))
		}
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_solve_equation_has_no_solution", expanded.String()))
	}

	// 2. Extract polynomial coefficients
	coeffs, err := extractPolyCoeffs(expanded, varName)
	if err != nil {
		return nil, err
	}

	maxDeg := 0
	for d := range coeffs {
		if d > maxDeg {
			maxDeg = d
		}
	}

	if maxDeg == 0 {
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_solve_equation_contains_no_degree", varName))
	}

	a0 := coeffs[0]
	if a0 == nil {
		a0 = mustRational(0, 1)
	}

	if maxDeg == 1 {
		// Linear equation: a1 * x + a0 = 0  =>  x = -a0 / a1
		a1 := coeffs[1]
		negA0, err := simplifyUnaryOp("-", a0)
		if err != nil {
			return nil, err
		}
		invA1, err := simplifyPow(a1, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		root, err := simplifyMul([]Node{negA0, invA1})
		if err != nil {
			return nil, err
		}
		res := NewList([]Node{root})
		RecordTraceRewrite(RuleSolveLinear, expr, res, fmt.Sprintf("1次方程式 %s = 0 を移項・除算して解を導出", Format(expanded)))
		return res, nil
	}

	if maxDeg == 2 {
		// Quadratic equation: a2 * x^2 + a1 * x + a0 = 0
		a2 := coeffs[2]
		a1 := coeffs[1]
		if a1 == nil {
			a1 = mustRational(0, 1)
		}

		// Discriminant D = a1^2 - 4 * a2 * a0
		a1Sq, err := simplifyPow(a1, mustRational(2, 1))
		if err != nil {
			return nil, err
		}
		fourA2A0, err := simplifyMul([]Node{mustRational(4, 1), a2, a0})
		if err != nil {
			return nil, err
		}
		negFour, err := simplifyUnaryOp("-", fourA2A0)
		if err != nil {
			return nil, err
		}
		d, err := simplifyAdd([]Node{a1Sq, negFour})
		if err != nil {
			return nil, err
		}

		negA1, err := simplifyUnaryOp("-", a1)
		if err != nil {
			return nil, err
		}
		twoA2, err := simplifyMul([]Node{mustRational(2, 1), a2})
		if err != nil {
			return nil, err
		}
		invTwoA2, err := simplifyPow(twoA2, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}

		if isZero(d) {
			// Single repeated root: x = -a1 / (2*a2)
			root, err := simplifyMul([]Node{negA1, invTwoA2})
			if err != nil {
				return nil, err
			}
			res := NewList([]Node{root})
			RecordTraceRewrite(RuleSolveQuadratic, expr, res, fmt.Sprintf("2次方程式の重解: 判別式 D = 0 より %s = -b / (2a)", varName))
			return res, nil
		}

		sqrtD, err := simplifySqrt(d)
		if err != nil {
			return nil, err
		}
		negSqrtD, err := simplifyUnaryOp("-", sqrtD)
		if err != nil {
			return nil, err
		}

		// root1 = (-a1 - sqrt(D)) / (2*a2)
		// root2 = (-a1 + sqrt(D)) / (2*a2)
		num1, err := simplifyAdd([]Node{negA1, negSqrtD})
		if err != nil {
			return nil, err
		}
		root1, err := simplifyMul([]Node{num1, invTwoA2})
		if err != nil {
			return nil, err
		}

		num2, err := simplifyAdd([]Node{negA1, sqrtD})
		if err != nil {
			return nil, err
		}
		root2, err := simplifyMul([]Node{num2, invTwoA2})
		if err != nil {
			return nil, err
		}

		resList := NewList([]Node{root1, root2})
		RecordTraceRewrite(RuleSolveQuadratic, expr, resList, fmt.Sprintf("2次方程式の解の公式: 判別式 D = b^2 - 4ac = %s, %s = (-b ± √D) / (2a)", Format(d), varName))
		return resList, nil
	}

	return nil, fmt.Errorf("%s", i18n.T("calculus.err_solve_error_polynomial_degree_only", maxDeg))
}

// -------------------------------------------------------------------------
// CAS: Matrix Operations (mulMatrixOrScalar, det, inv, transpose)
// -------------------------------------------------------------------------

func evalTaylor(f Node, varNode Node, center Node, orderNode Node) (Node, error) {
	v, ok := varNode.(*VarNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_taylor_error_second_argument_must", varNode.String()))
	}
	varName := v.Name

	rOrder, ok := orderNode.(*RationalNode)
	if !ok || !rOrder.Val.IsInt() || rOrder.Val.Sign() < 0 {
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_taylor_error_order_must_be", orderNode.String()))
	}
	n := rOrder.Val.Num().Int64()

	subEnv := NewEnv()
	subEnv.Set(varName, center)

	currentDeriv := f
	var terms []Node

	kFact := big.NewInt(1)

	for k := int64(0); k <= n; k++ {
		if k > 0 {
			kFact.Mul(kFact, big.NewInt(k))
			d, err := differentiate(currentDeriv, varName)
			if err != nil {
				return nil, fmt.Errorf("%s", i18n.T("calculus.err_taylor_error_in", k, err))
			}
			currentDeriv = d
		}

		// evaluate f^(k)(center)
		fVal, err := EvalWithEnv(currentDeriv, subEnv)
		if err != nil {
			return nil, fmt.Errorf("%s", i18n.T("calculus.err_taylor_error_cannot_evaluate_derivative", err))
		}

		if isZeroNode(fVal) {
			continue
		}

		// coeff = fVal / k!
		factRat := new(big.Rat).SetInt(kFact)
		invFact := new(big.Rat).Inv(factRat)
		coeff, err := simplifyMul([]Node{fVal, NewRationalFromBigRat(invFact)})
		if err != nil {
			return nil, err
		}
		if isZeroNode(coeff) {
			continue
		}

		// term = coeff * (x - center)^k
		var powerNode Node
		if isZeroNode(center) {
			switch k {
			case 0:
				powerNode = mustRational(1, 1)
			case 1:
				powerNode = v
			default:
				powerNode = &PowNode{Base: v, Exp: mustRational(k, 1)}
			}
		} else {
			diffTerm, err := simplifyAdd([]Node{v, &MulNode{Factors: []Node{mustRational(-1, 1), center}}})
			if err != nil {
				return nil, err
			}
			switch k {
			case 0:
				powerNode = mustRational(1, 1)
			case 1:
				powerNode = diffTerm
			default:
				powerNode = expandNode(&PowNode{Base: diffTerm, Exp: mustRational(k, 1)})
			}
		}

		term, err := simplifyMul([]Node{coeff, powerNode})
		if err != nil {
			return nil, err
		}
		term = expandNode(term)
		terms = append(terms, term)
	}

	if len(terms) == 0 {
		return mustRational(0, 1), nil
	}
	res, err := simplifyAdd(terms)
	if err != nil {
		return nil, err
	}
	return expandNode(res), nil
}

// -------------------------------------------------------------------------
// Discrete Series & Summation Engine (Faulhaber & Bernoulli)
// -------------------------------------------------------------------------

func computeBernoulli(m int) []*big.Rat {
	B := make([]*big.Rat, m+1)
	for i := 0; i <= m; i++ {
		B[i] = big.NewRat(0, 1)
	}
	B[0] = big.NewRat(1, 1)

	for i := 1; i <= m; i++ {
		sum := big.NewRat(0, 1)
		for j := 0; j < i; j++ {
			c := big.NewInt(0).Binomial(int64(i+1), int64(j))
			term := new(big.Rat).Mul(new(big.Rat).SetInt(c), B[j])
			sum.Add(sum, term)
		}
		denom := big.NewRat(int64(i+1), 1)
		B[i].Quo(new(big.Rat).Neg(sum), denom)
	}

	if m >= 1 {
		B[1] = big.NewRat(1, 2)
	}
	return B
}

func faulhaberSum(p int64, nVar Node) (Node, error) {
	if p == 0 {
		return nVar, nil
	}
	B := computeBernoulli(int(p))
	var terms []Node

	pPlus1 := p + 1
	pPlus1Rat := big.NewRat(pPlus1, 1)

	for j := int64(0); j <= p; j++ {
		c := big.NewInt(0).Binomial(pPlus1, j)
		coeffRat := new(big.Rat).Mul(new(big.Rat).SetInt(c), B[j])
		coeffRat.Quo(coeffRat, pPlus1Rat)

		if coeffRat.Sign() == 0 {
			continue
		}

		expVal := pPlus1 - j
		var nPow Node
		if expVal == 1 {
			nPow = nVar
		} else {
			nPow = &PowNode{Base: nVar, Exp: mustRational(expVal, 1)}
		}

		term, err := simplifyMul([]Node{NewRationalFromBigRat(coeffRat), nPow})
		if err != nil {
			return nil, err
		}
		terms = append(terms, term)
	}

	res, err := simplifyAdd(terms)
	if err != nil {
		return nil, err
	}
	return expandNode(res), nil
}

func extractPowerOfK(term Node, kVar string) (Node, int64, error) {
	if !containsVar(term, kVar) {
		return term, 0, nil
	}
	if v, ok := term.(*VarNode); ok && v.Name == kVar {
		return mustRational(1, 1), 1, nil
	}
	if pow, ok := term.(*PowNode); ok {
		if v, ok := pow.Base.(*VarNode); ok && v.Name == kVar {
			if rExp, ok := pow.Exp.(*RationalNode); ok && rExp.Val.IsInt() && rExp.Val.Sign() >= 0 {
				return mustRational(1, 1), rExp.Val.Num().Int64(), nil
			}
		}
	}
	if mul, ok := term.(*MulNode); ok {
		var otherFactors []Node
		totalPow := int64(0)
		for _, f := range mul.Factors {
			if containsVar(f, kVar) {
				subCoeff, subPow, err := extractPowerOfK(f, kVar)
				if err != nil {
					return nil, 0, err
				}
				if !isOneRat(subCoeff) {
					otherFactors = append(otherFactors, subCoeff)
				}
				totalPow += subPow
			} else {
				otherFactors = append(otherFactors, f)
			}
		}
		coeff, err := simplifyMul(otherFactors)
		if err != nil {
			return nil, 0, err
		}
		return coeff, totalPow, nil
	}
	return nil, 0, fmt.Errorf("%s", i18n.T("calculus.err_sum_error_cannot_handle_term", term.String()))
}

func isOneRat(n Node) bool {
	if r, ok := n.(*RationalNode); ok && r.Val.Cmp(big.NewRat(1, 1)) == 0 {
		return true
	}
	return false
}

func evalSum(expr Node, kVarNode Node, startNode Node, endNode Node) (Node, error) {
	v, ok := kVarNode.(*VarNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("calculus.err_sum_error_second_argument_must", kVarNode.String()))
	}
	kName := v.Name

	rStart, okStart := startNode.(*RationalNode)
	rEnd, okEnd := endNode.(*RationalNode)

	if okStart && okEnd && rStart.Val.IsInt() && rEnd.Val.IsInt() {
		startVal := rStart.Val.Num().Int64()
		endVal := rEnd.Val.Num().Int64()
		if startVal > endVal {
			return nil, fmt.Errorf("%s", i18n.T("calculus.err_sum_error_start_value_exceeds", startVal, endVal))
		}

		var sumTerms []Node
		for k := startVal; k <= endVal; k++ {
			subEnv := NewEnv()
			subEnv.Set(kName, mustRational(k, 1))
			val, err := EvalWithEnv(expr, subEnv)
			if err != nil {
				return nil, fmt.Errorf("%s", i18n.T("calculus.err_sum_error_at", kName, k, err))
			}
			sumTerms = append(sumTerms, val)
		}
		if len(sumTerms) == 0 {
			return mustRational(0, 1), nil
		}
		res, err := simplifyAdd(sumTerms)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	if okStart && rStart.Val.Cmp(big.NewRat(1, 1)) == 0 {
		expanded := expandNode(expr)
		var terms []Node
		if add, ok := expanded.(*AddNode); ok {
			terms = add.Terms
		} else {
			terms = []Node{expanded}
		}

		allFaulhaber := true
		var resultTerms []Node
		for _, t := range terms {
			coeff, pow, err := extractPowerOfK(t, kName)
			if err != nil {
				allFaulhaber = false
				break
			}
			sNode, err := faulhaberSum(pow, endNode)
			if err != nil {
				allFaulhaber = false
				break
			}
			termProd, err := simplifyMul([]Node{coeff, sNode})
			if err != nil {
				allFaulhaber = false
				break
			}
			resultTerms = append(resultTerms, expandNode(termProd))
		}

		if allFaulhaber {
			res, err := simplifyAdd(resultTerms)
			if err != nil {
				return nil, err
			}
			return expandNode(res), nil
		}
	}

	// Try Gosper's algorithm for hypergeometric summation
	gosperRes, err := GosperDefiniteSum(expr, kName, startNode, endNode)
	if err == nil {
		return gosperRes, nil
	}

	return nil, fmt.Errorf("%s", i18n.T("calculus.err_sum_error_unsupported_bounds_and", startNode.String(), endNode.String(), err))
}

// -------------------------------------------------------------------------
// 3D Vector Calculus (dot, cross, norm, grad, div, curl)
// -------------------------------------------------------------------------

