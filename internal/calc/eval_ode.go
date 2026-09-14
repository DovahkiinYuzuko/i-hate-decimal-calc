package calc

import (
	"fmt"
	"math/big"
)

// odeType represents the recognized classification of an ODE.
type odeType int

const (
	odeUnknown odeType = iota
	odeFirstLinear
	odeSecondLinearConstCoeff
)

// EvalDSolve solves an ordinary differential equation analytically and returns an equation node y = f(x).
// It supports:
// 1. 1st-order linear ODEs: y' + P(x)*y = Q(x) via integrating factor mu(x) = exp(integrate(P, x))
// 2. 1st-order separable ODEs: y' = f(x)*y (as a special case of 1st linear with Q=0)
// 3. 2nd-order linear ODEs with constant coefficients: a*y'' + b*y' + c*y = f(x)
//    - Homogeneous part via characteristic equation a*r^2 + b*r + c = 0
//    - Particular solution via method of undetermined coefficients for polynomials, exponentials, and sines/cosines.
func EvalDSolve(eq Node, yName, xName string, env *Env) (Node, error) {
	if eq == nil {
		return nil, fmt.Errorf("dsolve: equation cannot be nil")
	}

	// 1. Normalize equation to F(x, y, y', y'') = 0
	var expr Node
	if relNode, ok := eq.(*RelOpNode); ok && (relNode.Op == "==" || relNode.Op == "=") {
		negR, err := simplifyUnaryOp("-", relNode.RHS)
		if err != nil {
			return nil, err
		}
		expr, err = simplifyAdd([]Node{relNode.LHS, negR})
		if err != nil {
			return nil, err
		}
	} else {
		expr = eq
	}

	// 2. Infer variable names if not provided
	if yName == "" || xName == "" {
		infY, infX := inferODEVars(expr)
		if yName == "" {
			yName = infY
		}
		if xName == "" {
			xName = infX
		}
	}
	if yName == "" {
		yName = "y"
	}
	if xName == "" {
		xName = "x"
	}

	// 3. Classify the ODE
	typ, coeffs, err := classifyODE(expr, yName, xName)
	if err != nil {
		return nil, fmt.Errorf("dsolve classification error: %w", err)
	}

	var sol Node
	switch typ {
	case odeFirstLinear:
		sol, err = solve1stLinearODE(coeffs.P, coeffs.Q, xName, env)
		if err != nil {
			return nil, fmt.Errorf("dsolve 1st-order error: %w", err)
		}

	case odeSecondLinearConstCoeff:
		sol, err = solve2ndLinearConstCoeffODE(coeffs.a, coeffs.b, coeffs.c, coeffs.f, xName, env)
		if err != nil {
			return nil, fmt.Errorf("dsolve 2nd-order error: %w", err)
		}

	default:
		return nil, fmt.Errorf("dsolve: unable to classify equation into a supported ODE type")
	}

	// Return y == sol
	return NewRelOp(&VarNode{Name: yName}, "==", sol), nil
}

// odeCoeffs holds the extracted coefficient nodes for ODEs.
type odeCoeffs struct {
	// For 1st order: y' + P(x)*y = Q(x)
	P Node
	Q Node

	// For 2nd order: a*y'' + b*y' + c*y = f(x)
	a *big.Rat
	b *big.Rat
	c *big.Rat
	f Node
}

// inferODEVars searches for diff(y, x) inside the expression.
func inferODEVars(n Node) (yName, xName string) {
	var walk func(node Node)
	walk = func(node Node) {
		if node == nil || (yName != "" && xName != "") {
			return
		}
		if fn, ok := node.(*FuncNode); ok && fn.Name == "diff" && len(fn.Args) >= 2 {
			if vy, ok := fn.Args[0].(*VarNode); ok {
				yName = vy.Name
			}
			if vx, ok := fn.Args[1].(*VarNode); ok {
				xName = vx.Name
			}
			return
		}
		switch v := node.(type) {
		case *AddNode:
			for _, t := range v.Terms {
				walk(t)
			}
		case *MulNode:
			for _, f := range v.Factors {
				walk(f)
			}
		case *PowNode:
			walk(v.Base)
			walk(v.Exp)
		case *UnaryOpNode:
			walk(v.Expr)
		case *FuncNode:
			for _, a := range v.Args {
				walk(a)
			}
		case *RelOpNode:
			walk(v.LHS)
			walk(v.RHS)
		}
	}
	walk(n)
	return yName, xName
}

// isDeriv1 returns true if n is diff(y, x)
func isDeriv1(n Node, yName, xName string) bool {
	fn, ok := n.(*FuncNode)
	if !ok || fn.Name != "diff" {
		return false
	}
	if len(fn.Args) < 2 {
		return false
	}
	vy, okY := fn.Args[0].(*VarNode)
	vx, okX := fn.Args[1].(*VarNode)
	if !okY || !okX || vy.Name != yName || vx.Name != xName {
		return false
	}
	if len(fn.Args) == 2 {
		return true
	}
	if len(fn.Args) == 3 {
		if r, ok := fn.Args[2].(*RationalNode); ok && r.Val.Cmp(big.NewRat(1, 1)) == 0 {
			return true
		}
	}
	return false
}

// isDeriv2 returns true if n is diff(y, x, 2) or diff(diff(y, x), x)
func isDeriv2(n Node, yName, xName string) bool {
	fn, ok := n.(*FuncNode)
	if !ok || fn.Name != "diff" {
		return false
	}
	if len(fn.Args) == 3 {
		vy, okY := fn.Args[0].(*VarNode)
		vx, okX := fn.Args[1].(*VarNode)
		if okY && okX && vy.Name == yName && vx.Name == xName {
			if r, ok := fn.Args[2].(*RationalNode); ok && r.Val.Cmp(big.NewRat(2, 1)) == 0 {
				return true
			}
		}
	}
	if len(fn.Args) == 2 {
		vx, okX := fn.Args[1].(*VarNode)
		if okX && vx.Name == xName {
			if isDeriv1(fn.Args[0], yName, xName) {
				return true
			}
		}
	}
	return false
}

// isDepVar returns true if n is y or y(x)
func isDepVar(n Node, yName, xName string) bool {
	if v, ok := n.(*VarNode); ok && v.Name == yName {
		return true
	}
	if fn, ok := n.(*FuncNode); ok && fn.Name == yName && len(fn.Args) == 1 {
		if vx, ok := fn.Args[0].(*VarNode); ok && vx.Name == xName {
			return true
		}
	}
	return false
}

// extractLinearCoeff checks if term contains a factor matching pred(factor).
// If so, returns the remaining factors (coeff) and true.
func extractLinearCoeff(term Node, pred func(Node) bool) (Node, bool) {
	if pred(term) {
		return mustRational(1, 1), true
	}
	if u, ok := term.(*UnaryOpNode); ok && u.Op == "-" {
		if subCoeff, found := extractLinearCoeff(u.Expr, pred); found {
			negCoeff, _ := simplifyUnaryOp("-", subCoeff)
			return negCoeff, true
		}
		return nil, false
	}
	if m, ok := term.(*MulNode); ok {
		var matchedIndex = -1
		for i, f := range m.Factors {
			if pred(f) {
				if matchedIndex != -1 {
					// Non-linear! (e.g. y * y')
					return nil, false
				}
				matchedIndex = i
			}
		}
		if matchedIndex != -1 {
			var remaining []Node
			for i, f := range m.Factors {
				if i != matchedIndex {
					remaining = append(remaining, f)
				}
			}
			coeff, err := simplifyMul(remaining)
			if err != nil {
				return nil, false
			}
			return coeff, true
		}
	}
	return nil, false
}

// classifyODE decomposes the expression into terms of y'', y', y, and independent terms f(x).
func classifyODE(expr Node, yName, xName string) (odeType, *odeCoeffs, error) {
	var terms []Node
	if add, ok := expr.(*AddNode); ok {
		terms = add.Terms
	} else {
		terms = []Node{expr}
	}

	var d2Terms []Node
	var d1Terms []Node
	var yTerms []Node
	var fTerms []Node

	for _, t := range terms {
		if c, ok := extractLinearCoeff(t, func(n Node) bool { return isDeriv2(n, yName, xName) }); ok {
			d2Terms = append(d2Terms, c)
		} else if c, ok := extractLinearCoeff(t, func(n Node) bool { return isDeriv1(n, yName, xName) }); ok {
			d1Terms = append(d1Terms, c)
		} else if c, ok := extractLinearCoeff(t, func(n Node) bool { return isDepVar(n, yName, xName) }); ok {
			yTerms = append(yTerms, c)
		} else {
			// Terms without y, y', y'' move to the RHS (so negate them: F = 0 => RHS = -t)
			negT, err := simplifyUnaryOp("-", t)
			if err != nil {
				return odeUnknown, nil, err
			}
			fTerms = append(fTerms, negT)
		}
	}

	rhsNode, err := simplifyAdd(fTerms)
	if err != nil {
		return odeUnknown, nil, err
	}

	// Case 1: 2nd-order ODE
	if len(d2Terms) > 0 {
		coeffA, err := simplifyAdd(d2Terms)
		if err != nil {
			return odeUnknown, nil, err
		}
		var coeffB Node = mustRational(0, 1)
		if len(d1Terms) > 0 {
			coeffB, err = simplifyAdd(d1Terms)
			if err != nil {
				return odeUnknown, nil, err
			}
		}
		var coeffC Node = mustRational(0, 1)
		if len(yTerms) > 0 {
			coeffC, err = simplifyAdd(yTerms)
			if err != nil {
				return odeUnknown, nil, err
			}
		}

		// Check if a, b, c are constants (RationalNode)
		ra, okA := coeffA.(*RationalNode)
		rb, okB := coeffB.(*RationalNode)
		rc, okC := coeffC.(*RationalNode)
		if okA && okB && okC {
			if ra.Val.Sign() == 0 {
				return odeUnknown, nil, fmt.Errorf("coefficient of y'' is zero")
			}
			return odeSecondLinearConstCoeff, &odeCoeffs{
				a: new(big.Rat).Set(ra.Val),
				b: new(big.Rat).Set(rb.Val),
				c: new(big.Rat).Set(rc.Val),
				f: rhsNode,
			}, nil
		}
		return odeUnknown, nil, fmt.Errorf("2nd-order ODE has non-constant coefficients")
	}

	// Case 2: 1st-order linear ODE
	if len(d1Terms) > 0 {
		coeffA, err := simplifyAdd(d1Terms)
		if err != nil {
			return odeUnknown, nil, err
		}
		var coeffB Node = mustRational(0, 1)
		if len(yTerms) > 0 {
			coeffB, err = simplifyAdd(yTerms)
			if err != nil {
				return odeUnknown, nil, err
			}
		}

		// y' + (B/A)*y = RHS / A
		// P(x) = B(x) / A(x)
		invA, err := simplifyPow(coeffA, mustRational(-1, 1))
		if err != nil {
			return odeUnknown, nil, err
		}
		pNode, err := simplifyMul([]Node{coeffB, invA})
		if err != nil {
			return odeUnknown, nil, err
		}
		qNode, err := simplifyMul([]Node{rhsNode, invA})
		if err != nil {
			return odeUnknown, nil, err
		}

		return odeFirstLinear, &odeCoeffs{
			P: pNode,
			Q: qNode,
		}, nil
	}

	return odeUnknown, nil, fmt.Errorf("no derivative terms y' or y'' found in equation")
}

// solve1stLinearODE solves y' + P(x)*y = Q(x)
// Integrating factor: mu(x) = exp(integrate(P, x))
// Solution: y = (1/mu) * (integrate(mu * Q, x) + C_1)
func solve1stLinearODE(pNode, qNode Node, xName string, env *Env) (Node, error) {
	xVar := &VarNode{Name: xName}
	c1 := &VarNode{Name: "C_1"}

	// 1. Integrate P(x) dx
	intP, err := evalIndefiniteIntegral(pNode, xName)
	if err != nil {
		return nil, fmt.Errorf("failed to integrate P(x): %w", err)
	}

	// 2. Integrating factor mu(x) = exp(intP)
	mu, err := simplifyFuncWithEnv("exp", []Node{intP}, env)
	if err != nil {
		return nil, fmt.Errorf("failed to compute integrating factor: %w", err)
	}

	// If Q(x) == 0 (separable / homogeneous), solution is simply C_1 / mu = C_1 * exp(-intP)
	if isZero(qNode) {
		negIntP, err := simplifyUnaryOp("-", intP)
		if err != nil {
			return nil, err
		}
		expNegIntP, err := simplifyFuncWithEnv("exp", []Node{negIntP}, env)
		if err != nil {
			return nil, err
		}
		return simplifyMul([]Node{c1, expNegIntP})
	}

	// 3. Integrate mu * Q dx
	muTimesQ, err := simplifyMul([]Node{mu, qNode})
	if err != nil {
		return nil, err
	}
	intMuQ, err := evalIndefiniteIntegral(muTimesQ, xName)
	if err != nil {
		return nil, fmt.Errorf("failed to integrate mu(x)*Q(x): %w", err)
	}

	// 4. (intMuQ + C_1) / mu
	sumWithConst, err := simplifyAdd([]Node{intMuQ, c1})
	if err != nil {
		return nil, err
	}

	// If mu is exp(u), 1/mu is exp(-u)
	var invMu Node
	if fn, ok := mu.(*FuncNode); ok && fn.Name == "exp" && len(fn.Args) == 1 {
		negArg, err := simplifyUnaryOp("-", fn.Args[0])
		if err == nil {
			invMu, _ = simplifyFuncWithEnv("exp", []Node{negArg}, env)
		}
	}
	if invMu == nil {
		invMu, err = simplifyPow(mu, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
	}

	sol, err := simplifyMul([]Node{sumWithConst, invMu})
	if err != nil {
		return nil, err
	}
	_ = xVar
	return sol, nil
}

// solve2ndLinearConstCoeffODE solves a*y'' + b*y' + c*y = f(x)
func solve2ndLinearConstCoeffODE(a, b, c *big.Rat, fNode Node, xName string, env *Env) (Node, error) {
	c1 := &VarNode{Name: "C_1"}
	c2 := &VarNode{Name: "C_2"}
	xVar := &VarNode{Name: xName}

	// Characteristic equation: a*r^2 + b*r + c = 0
	// D = b^2 - 4ac
	b2 := new(big.Rat).Mul(b, b)
	fourAC := new(big.Rat).Mul(big.NewRat(4, 1), new(big.Rat).Mul(a, c))
	disc := new(big.Rat).Sub(b2, fourAC)

	twoA := new(big.Rat).Mul(big.NewRat(2, 1), a)
	minusB := new(big.Rat).Neg(b)

	var yh Node
	discSign := disc.Sign()

	if discSign == 0 {
		// Single repeated real root: r = -b / (2a)
		r := new(big.Rat).Quo(minusB, twoA)
		rNode := &RationalNode{Val: r}

		// (C_1 + C_2 * x) * exp(r * x)
		c2x, err := simplifyMul([]Node{c2, xVar})
		if err != nil {
			return nil, err
		}
		cPart, err := simplifyAdd([]Node{c1, c2x})
		if err != nil {
			return nil, err
		}

		var expPart Node
		if r.Sign() == 0 {
			expPart = mustRational(1, 1)
		} else {
			rx, err := simplifyMul([]Node{rNode, xVar})
			if err != nil {
				return nil, err
			}
			expPart, err = simplifyFuncWithEnv("exp", []Node{rx}, env)
			if err != nil {
				return nil, err
			}
		}
		yh, err = simplifyMul([]Node{cPart, expPart})
		if err != nil {
			return nil, err
		}

	} else if discSign > 0 {
		// Two distinct real roots: r1, r2 = (-b +/- sqrt(D)) / (2a)
		sqrtD, isPerf := exactRatSqrt(disc)
		if isPerf {
			r1Rat := new(big.Rat).Quo(new(big.Rat).Add(minusB, sqrtD), twoA)
			r2Rat := new(big.Rat).Quo(new(big.Rat).Sub(minusB, sqrtD), twoA)

			// C_1 * exp(r1*x) + C_2 * exp(r2*x)
			term1, err := makeExpTerm(c1, r1Rat, xVar, env)
			if err != nil {
				return nil, err
			}
			term2, err := makeExpTerm(c2, r2Rat, xVar, env)
			if err != nil {
				return nil, err
			}
			yh, err = simplifyAdd([]Node{term1, term2})
			if err != nil {
				return nil, err
			}
		} else {
			// Symbolic sqrt: r1,2 = (-b / 2a) +/- sqrt(D) / 2a
			alpha := new(big.Rat).Quo(minusB, twoA)
			betaNode := &MulNode{
				Factors: []Node{
					&PowNode{Base: &RationalNode{Val: twoA}, Exp: mustRational(-1, 1)},
					&SqrtNode{Radicand: &RationalNode{Val: disc}},
				},
			}
			alphaNode := &RationalNode{Val: alpha}
			r1, err := simplifyAdd([]Node{alphaNode, betaNode})
			if err != nil {
				return nil, err
			}
			negBeta, _ := simplifyUnaryOp("-", betaNode)
			r2, err := simplifyAdd([]Node{alphaNode, negBeta})
			if err != nil {
				return nil, err
			}

			r1x, _ := simplifyMul([]Node{r1, xVar})
			r2x, _ := simplifyMul([]Node{r2, xVar})
			exp1, _ := simplifyFuncWithEnv("exp", []Node{r1x}, env)
			exp2, _ := simplifyFuncWithEnv("exp", []Node{r2x}, env)

			term1, _ := simplifyMul([]Node{c1, exp1})
			term2, _ := simplifyMul([]Node{c2, exp2})
			yh, err = simplifyAdd([]Node{term1, term2})
			if err != nil {
				return nil, err
			}
		}

	} else {
		// Complex conjugate roots: alpha +/- i * beta
		// alpha = -b / (2a), beta = sqrt(-D) / (2a)
		negDisc := new(big.Rat).Neg(disc)
		alpha := new(big.Rat).Quo(minusB, twoA)
		alphaNode := &RationalNode{Val: alpha}

		sqrtNegD, isPerf := exactRatSqrt(negDisc)
		var betaNode Node
		if isPerf {
			betaRat := new(big.Rat).Quo(sqrtNegD, twoA)
			betaRat.Abs(betaRat)
			betaNode = &RationalNode{Val: betaRat}
		} else {
			betaNode = &MulNode{
				Factors: []Node{
					&PowNode{Base: &RationalNode{Val: twoA}, Exp: mustRational(-1, 1)},
					&SqrtNode{Radicand: &RationalNode{Val: negDisc}},
				},
			}
		}

		// yh = exp(alpha*x) * (C_1 * cos(beta*x) + C_2 * sin(beta*x))
		betax, err := simplifyMul([]Node{betaNode, xVar})
		if err != nil {
			return nil, err
		}
		cosPart, err := simplifyFuncWithEnv("cos", []Node{betax}, env)
		if err != nil {
			return nil, err
		}
		sinPart, err := simplifyFuncWithEnv("sin", []Node{betax}, env)
		if err != nil {
			return nil, err
		}

		c1Cos, err := simplifyMul([]Node{c1, cosPart})
		if err != nil {
			return nil, err
		}
		c2Sin, err := simplifyMul([]Node{c2, sinPart})
		if err != nil {
			return nil, err
		}

		trigSum, err := simplifyAdd([]Node{c1Cos, c2Sin})
		if err != nil {
			return nil, err
		}

		if alpha.Sign() == 0 {
			yh = trigSum
		} else {
			alphax, err := simplifyMul([]Node{alphaNode, xVar})
			if err != nil {
				return nil, err
			}
			expAlpha, err := simplifyFuncWithEnv("exp", []Node{alphax}, env)
			if err != nil {
				return nil, err
			}
			yh, err = simplifyMul([]Node{expAlpha, trigSum})
			if err != nil {
				return nil, err
			}
		}
	}

	// Particular solution for non-homogeneous term f(x)
	if isZero(fNode) {
		return yh, nil
	}

	yp, err := solveUndeterminedCoefficients(a, b, c, fNode, xName, env)
	if err != nil {
		// Fall back to returning homogeneous solution with warning or error
		return nil, fmt.Errorf("unable to find particular solution for RHS %s: %w", fNode.String(), err)
	}

	return simplifyAdd([]Node{yh, yp})
}

// makeExpTerm constructs C * exp(r * x)
func makeExpTerm(constNode Node, r *big.Rat, xVar Node, env *Env) (Node, error) {
	if r.Sign() == 0 {
		return constNode, nil
	}
	rx, err := simplifyMul([]Node{&RationalNode{Val: r}, xVar})
	if err != nil {
		return nil, err
	}
	expr, err := simplifyFuncWithEnv("exp", []Node{rx}, env)
	if err != nil {
		return nil, err
	}
	return simplifyMul([]Node{constNode, expr})
}

// exactRatSqrt returns (sqrt(r), true) if r is a perfect rational square.
func exactRatSqrt(r *big.Rat) (*big.Rat, bool) {
	if r.Sign() < 0 {
		return nil, false
	}
	if r.Sign() == 0 {
		return big.NewRat(0, 1), true
	}
	num := r.Num()
	den := r.Denom()

	sqrtNum := new(big.Int).Sqrt(num)
	if new(big.Int).Mul(sqrtNum, sqrtNum).Cmp(num) != 0 {
		return nil, false
	}
	sqrtDen := new(big.Int).Sqrt(den)
	if new(big.Int).Mul(sqrtDen, sqrtDen).Cmp(den) != 0 {
		return nil, false
	}
	res := new(big.Rat).SetFrac(sqrtNum, sqrtDen)
	return res, true
}

// solveUndeterminedCoefficients finds a particular solution yp for a*y'' + b*y' + c*y = f(x).
func solveUndeterminedCoefficients(a, b, c *big.Rat, fNode Node, xName string, env *Env) (Node, error) {
	xVar := &VarNode{Name: xName}

	// Case 1: Constant RHS f(x) = K
	if r, ok := fNode.(*RationalNode); ok {
		if c.Sign() != 0 {
			// a*0 + b*0 + c*yp = K => yp = K / c
			ypVal := new(big.Rat).Quo(r.Val, c)
			return &RationalNode{Val: ypVal}, nil
		} else if b.Sign() != 0 {
			// a*0 + b*A + c*0 = K => A = K / b => yp = (K/b) * x
			ypVal := new(big.Rat).Quo(r.Val, b)
			return simplifyMul([]Node{&RationalNode{Val: ypVal}, xVar})
		} else {
			// a * 2A = K => A = K / (2a) => yp = (K/2a) * x^2
			twoA := new(big.Rat).Mul(big.NewRat(2, 1), a)
			ypVal := new(big.Rat).Quo(r.Val, twoA)
			x2, err := simplifyPow(xVar, mustRational(2, 1))
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{&RationalNode{Val: ypVal}, x2})
		}
	}

	// Case 2: Exponential RHS f(x) = k * exp(lambda * x)
	// Check if fNode is exp(lambda * x) or k * exp(lambda * x)
	kRat, lambdaRat, isExp := extractExpForm(fNode, xName)
	if isExp {
		// Characteristic polynomial P(lambda) = a*lambda^2 + b*lambda + c
		lam2 := new(big.Rat).Mul(lambdaRat, lambdaRat)
		pLam := new(big.Rat).Add(
			new(big.Rat).Mul(a, lam2),
			new(big.Rat).Add(new(big.Rat).Mul(b, lambdaRat), c),
		)

		if pLam.Sign() != 0 {
			// yp = (k / P(lambda)) * exp(lambda * x)
			coeff := new(big.Rat).Quo(kRat, pLam)
			return makeExpTerm(&RationalNode{Val: coeff}, lambdaRat, xVar, env)
		}

		// Derivative P'(lambda) = 2*a*lambda + b
		pPrimeLam := new(big.Rat).Add(
			new(big.Rat).Mul(big.NewRat(2, 1), new(big.Rat).Mul(a, lambdaRat)),
			b,
		)
		if pPrimeLam.Sign() != 0 {
			// yp = (k / P'(lambda)) * x * exp(lambda * x)
			coeff := new(big.Rat).Quo(kRat, pPrimeLam)
			expPart, err := makeExpTerm(&RationalNode{Val: coeff}, lambdaRat, xVar, env)
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{xVar, expPart})
		}

		// Double root: P''(lambda) = 2*a => yp = (k / 2a) * x^2 * exp(lambda * x)
		twoA := new(big.Rat).Mul(big.NewRat(2, 1), a)
		coeff := new(big.Rat).Quo(kRat, twoA)
		x2, err := simplifyPow(xVar, mustRational(2, 1))
		if err != nil {
			return nil, err
		}
		expPart, err := makeExpTerm(&RationalNode{Val: coeff}, lambdaRat, xVar, env)
		if err != nil {
			return nil, err
		}
		return simplifyMul([]Node{x2, expPart})
	}

	// Case 3: Sine / Cosine RHS: f(x) = k1 * sin(w*x) + k2 * cos(w*x)
	kSin, kCos, omegaRat, isTrig := extractTrigForm(fNode, xName)
	if isTrig {
		// yp = A * cos(w*x) + B * sin(w*x)
		// y' = -A*w*sin(w*x) + B*w*cos(w*x)
		// y'' = -A*w^2*cos(w*x) - B*w^2*sin(w*x)
		// a*y'' + b*y' + c*y = (c - a*w^2)*[A cos + B sin] + b*w*[-A sin + B cos]
		//                    = [(c - a*w^2)*A + b*w*B] * cos(w*x) + [(c - a*w^2)*B - b*w*A] * sin(w*x)
		// Let u = c - a*w^2, v = b*w
		// Coeff of cos: u*A + v*B = kCos
		// Coeff of sin: -v*A + u*B = kSin
		// Matrix: [ u   v ] [ A ] = [ kCos ]
		//         [ -v  u ] [ B ]   [ kSin ]
		// Determinant: Delta = u^2 + v^2
		w2 := new(big.Rat).Mul(omegaRat, omegaRat)
		aw2 := new(big.Rat).Mul(a, w2)
		u := new(big.Rat).Sub(c, aw2)
		v := new(big.Rat).Mul(b, omegaRat)

		delta := new(big.Rat).Add(new(big.Rat).Mul(u, u), new(big.Rat).Mul(v, v))
		if delta.Sign() != 0 {
			// A = (u * kCos - v * kSin) / Delta
			// B = (v * kCos + u * kSin) / Delta
			uKcos := new(big.Rat).Mul(u, kCos)
			vKsin := new(big.Rat).Mul(v, kSin)
			aNum := new(big.Rat).Sub(uKcos, vKsin)
			aRat := new(big.Rat).Quo(aNum, delta)

			vKcos := new(big.Rat).Mul(v, kCos)
			uKsin := new(big.Rat).Mul(u, kSin)
			bNum := new(big.Rat).Add(vKcos, uKsin)
			bRat := new(big.Rat).Quo(bNum, delta)

			wx, err := simplifyMul([]Node{&RationalNode{Val: omegaRat}, xVar})
			if err != nil {
				return nil, err
			}
			cosPart, err := simplifyFuncWithEnv("cos", []Node{wx}, env)
			if err != nil {
				return nil, err
			}
			sinPart, err := simplifyFuncWithEnv("sin", []Node{wx}, env)
			if err != nil {
				return nil, err
			}

			termA, err := simplifyMul([]Node{&RationalNode{Val: aRat}, cosPart})
			if err != nil {
				return nil, err
			}
			termB, err := simplifyMul([]Node{&RationalNode{Val: bRat}, sinPart})
			if err != nil {
				return nil, err
			}
			return simplifyAdd([]Node{termA, termB})
		}
		// Resonance (delta == 0): yp = x * (A * cos + B * sin)
		// For resonance in a*y'' + c*y = k*cos(w*x) with b=0, c = a*w^2:
		// y'' + w^2 y = (k/a)*cos(w*x) => yp = (k / (2*a*w)) * x * sin(w*x)
		if b.Sign() == 0 {
			twoAW := new(big.Rat).Mul(big.NewRat(2, 1), new(big.Rat).Mul(a, omegaRat))
			wx, _ := simplifyMul([]Node{&RationalNode{Val: omegaRat}, xVar})
			var ypTerms []Node
			if kCos.Sign() != 0 {
				// kCos * cos(w*x) => (kCos / (2aw)) * x * sin(w*x)
				coeff := new(big.Rat).Quo(kCos, twoAW)
				sinPart, _ := simplifyFuncWithEnv("sin", []Node{wx}, env)
				t, _ := simplifyMul([]Node{&RationalNode{Val: coeff}, xVar, sinPart})
				ypTerms = append(ypTerms, t)
			}
			if kSin.Sign() != 0 {
				// kSin * sin(w*x) => -(kSin / (2aw)) * x * cos(w*x)
				coeff := new(big.Rat).Quo(kSin, twoAW)
				coeff.Neg(coeff)
				cosPart, _ := simplifyFuncWithEnv("cos", []Node{wx}, env)
				t, _ := simplifyMul([]Node{&RationalNode{Val: coeff}, xVar, cosPart})
				ypTerms = append(ypTerms, t)
			}
			if len(ypTerms) > 0 {
				return simplifyAdd(ypTerms)
			}
		}
	}

	return nil, fmt.Errorf("undetermined coefficients: unsupported RHS pattern")
}

// extractExpForm tests if node is k * exp(lambda * x) or exp(lambda * x)
func extractExpForm(n Node, xName string) (k, lambda *big.Rat, ok bool) {
	k = big.NewRat(1, 1)

	target := n
	if mul, isMul := n.(*MulNode); isMul {
		var expNode Node
		for _, f := range mul.Factors {
			if r, isR := f.(*RationalNode); isR {
				k.Mul(k, r.Val)
			} else if expNode == nil && isExpNode(f) {
				expNode = f
			} else {
				return nil, nil, false
			}
		}
		if expNode == nil {
			return nil, nil, false
		}
		target = expNode
	} else if u, isU := n.(*UnaryOpNode); isU && u.Op == "-" {
		subK, subLam, subOk := extractExpForm(u.Expr, xName)
		if subOk {
			subK.Neg(subK)
			return subK, subLam, true
		}
		return nil, nil, false
	}

	fn, isFn := target.(*FuncNode)
	if !isFn || fn.Name != "exp" || len(fn.Args) != 1 {
		return nil, nil, false
	}

	arg := fn.Args[0]
	// arg should be lambda * x or x
	if vx, isVx := arg.(*VarNode); isVx && vx.Name == xName {
		return k, big.NewRat(1, 1), true
	}
	if u, isU := arg.(*UnaryOpNode); isU && u.Op == "-" {
		if vx, isVx := u.Expr.(*VarNode); isVx && vx.Name == xName {
			return k, big.NewRat(-1, 1), true
		}
	}
	if m, isM := arg.(*MulNode); isM {
		var lam = big.NewRat(1, 1)
		foundX := false
		for _, f := range m.Factors {
			if vx, isVx := f.(*VarNode); isVx && vx.Name == xName {
				foundX = true
			} else if r, isR := f.(*RationalNode); isR {
				lam.Mul(lam, r.Val)
			} else {
				return nil, nil, false
			}
		}
		if foundX {
			return k, lam, true
		}
	}

	return nil, nil, false
}

func isExpNode(n Node) bool {
	fn, ok := n.(*FuncNode)
	return ok && fn.Name == "exp" && len(fn.Args) == 1
}

// extractTrigForm tests if node is a sum of k1 * sin(w*x) + k2 * cos(w*x)
func extractTrigForm(n Node, xName string) (kSin, kCos, omega *big.Rat, ok bool) {
	kSin = big.NewRat(0, 1)
	kCos = big.NewRat(0, 1)

	var terms []Node
	if add, isAdd := n.(*AddNode); isAdd {
		terms = add.Terms
	} else {
		terms = []Node{n}
	}

	for _, t := range terms {
		k, w, isS, isC := parseSingleTrigTerm(t, xName)
		if !isS && !isC {
			return nil, nil, nil, false
		}
		if omega == nil {
			omega = w
		} else if omega.Cmp(w) != 0 {
			// Different frequencies not supported in single pass
			return nil, nil, nil, false
		}
		if isS {
			kSin.Add(kSin, k)
		}
		if isC {
			kCos.Add(kCos, k)
		}
	}

	if omega == nil {
		return nil, nil, nil, false
	}
	return kSin, kCos, omega, true
}

func parseSingleTrigTerm(t Node, xName string) (k, omega *big.Rat, isSin, isCos bool) {
	k = big.NewRat(1, 1)
	target := t

	if u, isU := t.(*UnaryOpNode); isU && u.Op == "-" {
		subK, subW, s, c := parseSingleTrigTerm(u.Expr, xName)
		if s || c {
			subK.Neg(subK)
			return subK, subW, s, c
		}
		return nil, nil, false, false
	}

	if m, isM := t.(*MulNode); isM {
		var trigNode Node
		for _, f := range m.Factors {
			if r, isR := f.(*RationalNode); isR {
				k.Mul(k, r.Val)
			} else if trigNode == nil && isSinOrCos(f) {
				trigNode = f
			} else {
				return nil, nil, false, false
			}
		}
		if trigNode == nil {
			return nil, nil, false, false
		}
		target = trigNode
	}

	fn, isFn := target.(*FuncNode)
	if !isFn || len(fn.Args) != 1 {
		return nil, nil, false, false
	}

	if fn.Name != "sin" && fn.Name != "cos" {
		return nil, nil, false, false
	}

	// Extract omega from arg (omega * x or x)
	arg := fn.Args[0]
	omega = big.NewRat(1, 1)
	if vx, isVx := arg.(*VarNode); isVx && vx.Name == xName {
		return k, omega, fn.Name == "sin", fn.Name == "cos"
	}
	if m, isM := arg.(*MulNode); isM {
		foundX := false
		for _, f := range m.Factors {
			if vx, isVx := f.(*VarNode); isVx && vx.Name == xName {
				foundX = true
			} else if r, isR := f.(*RationalNode); isR {
				omega.Mul(omega, r.Val)
			} else {
				return nil, nil, false, false
			}
		}
		if foundX {
			return k, omega, fn.Name == "sin", fn.Name == "cos"
		}
	}

	return nil, nil, false, false
}

func isSinOrCos(n Node) bool {
	fn, ok := n.(*FuncNode)
	return ok && (fn.Name == "sin" || fn.Name == "cos") && len(fn.Args) == 1
}
