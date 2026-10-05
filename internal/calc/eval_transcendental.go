package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// EvalLambertW evaluates the Lambert W function lambert_w(z [, branch]).
func EvalLambertW(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s", i18n.T("transcendental.err_lambert_w_args"))
	}
	z := args[0]
	branch := int64(0)
	if len(args) == 2 {
		r, ok := args[1].(*RationalNode)
		if !ok || !r.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("transcendental.err_branch_must_be_integer", args[1].String()))
		}
		branch = r.Val.Num().Int64()
	}

	// Immediate special values for principal branch W_0
	if branch == 0 {
		// W_0(0) = 0
		if isZero(z) {
			return mustRational(0, 1), nil
		}
		// W_0(e) = 1
		if c, ok := z.(*ConstNode); ok && c.Name == "e" {
			return mustRational(1, 1), nil
		}
		// Check for -1/e => -1
		if p, ok := z.(*PowNode); ok {
			if c, okC := p.Base.(*ConstNode); okC && c.Name == "e" {
				if r, okR := p.Exp.(*RationalNode); okR && r.Val.Cmp(big.NewRat(-1, 1)) == 0 {
					// W_0(e^-1) is not -1, but -e^-1 is -1
				}
			}
		}
		if u, ok := z.(*UnaryOpNode); ok && u.Op == "-" {
			// -exp(-1) or -1/e
			if p, okP := u.Expr.(*PowNode); okP {
				if c, okC := p.Base.(*ConstNode); okC && c.Name == "e" {
					if r, okR := p.Exp.(*RationalNode); okR && r.Val.Cmp(big.NewRat(-1, 1)) == 0 {
						return mustRational(-1, 1), nil
					}
				}
			}
			if m, okM := u.Expr.(*MulNode); okM {
				// 1/e represented as mul
				_ = m
			}
		}
	}

	return ast.NewFunc("lambert_w", args)
}

// EvalErf evaluates the error function erf(z).
func EvalErf(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("transcendental.err_erf_args"))
	}
	z := args[0]

	// erf(0) = 0
	if isZero(z) {
		return mustRational(0, 1), nil
	}

	// Odd function: erf(-z) = -erf(z)
	if isNegative(z) {
		posZ, err := simplifyUnaryOp("-", z)
		if err == nil {
			innerErf, err2 := ast.NewFunc("erf", []Node{posZ})
			if err2 == nil {
				return simplifyUnaryOp("-", innerErf)
			}
		}
	}

	return ast.NewFunc("erf", args)
}

// DifferentiateLambertW computes d/dx [ W(u(x)) ] = W(u) * u' / (u * (1 + W(u))).
func DifferentiateLambertW(arg Node, varName string) (Node, error) {
	du, err := differentiate(arg, varName)
	if err != nil {
		return nil, err
	}
	if isZero(du) {
		return mustRational(0, 1), nil
	}

	wU, err := ast.NewFunc("lambert_w", []Node{arg})
	if err != nil {
		return nil, err
	}

	// 1 + W(u)
	onePlusW, err := simplifyAdd([]Node{mustRational(1, 1), wU})
	if err != nil {
		return nil, err
	}

	// denom = u * (1 + W(u))
	denom, err := simplifyMul([]Node{arg, onePlusW})
	if err != nil {
		return nil, err
	}

	invDenom, err := simplifyPow(denom, mustRational(-1, 1))
	if err != nil {
		return nil, err
	}

	// num = W(u) * du
	num, err := simplifyMul([]Node{wU, du})
	if err != nil {
		return nil, err
	}

	return simplifyMul([]Node{num, invDenom})
}

// DifferentiateErf computes d/dx [ erf(u(x)) ] = 2 / sqrt(pi) * exp(-u^2) * u'.
func DifferentiateErf(arg Node, varName string) (Node, error) {
	du, err := differentiate(arg, varName)
	if err != nil {
		return nil, err
	}
	if isZero(du) {
		return mustRational(0, 1), nil
	}

	// u^2
	uSq, err := simplifyPow(arg, mustRational(2, 1))
	if err != nil {
		return nil, err
	}

	// -u^2
	negUSq, err := simplifyUnaryOp("-", uSq)
	if err != nil {
		return nil, err
	}

	// exp(-u^2)
	expTerm, err := ast.NewFunc("exp", []Node{negUSq})
	if err != nil {
		return nil, err
	}

	// sqrt(pi)
	sqrtPi, err := simplifySqrt(&ConstNode{Name: "pi"})
	if err != nil {
		return nil, err
	}

	invSqrtPi, err := simplifyPow(sqrtPi, mustRational(-1, 1))
	if err != nil {
		return nil, err
	}

	// 2 / sqrt(pi) * exp(-u^2) * du
	return simplifyMul([]Node{mustRational(2, 1), invSqrtPi, expTerm, du})
}

// TryIntegrateGaussian checks if expr has the form k * exp(-a*x^2 + b*x + c) and integrates it using erf.
func TryIntegrateGaussian(expr Node, varName string) (Node, bool) {
	if !containsVar(expr, varName) {
		return nil, false
	}

	// Extract scalar factor k and exp(P(x))
	var k Node = mustRational(1, 1)
	var expArg Node

	switch v := expr.(type) {
	case *FuncNode:
		if v.Name == "exp" && len(v.Args) == 1 {
			expArg = v.Args[0]
		}
	case *PowNode:
		if c, ok := v.Base.(*ConstNode); ok && c.Name == "e" {
			expArg = v.Exp
		}
	case *MulNode:
		var nonVarFactors []Node
		var foundExp bool
		for _, f := range v.Factors {
			if !containsVar(f, varName) {
				nonVarFactors = append(nonVarFactors, f)
				continue
			}
			if !foundExp {
				if fn, ok := f.(*FuncNode); ok && fn.Name == "exp" && len(fn.Args) == 1 {
					expArg = fn.Args[0]
					foundExp = true
					continue
				}
				if pn, ok := f.(*PowNode); ok {
					if c, okC := pn.Base.(*ConstNode); okC && c.Name == "e" {
						expArg = pn.Exp
						foundExp = true
						continue
					}
				}
			}
			// Another variable factor found -> not a pure Gaussian
			return nil, false
		}
		if foundExp {
			if len(nonVarFactors) > 0 {
				mulK, err := simplifyMul(nonVarFactors)
				if err != nil {
					return nil, false
				}
				k = mulK
			}
		}
	case *UnaryOpNode:
		if v.Op == "-" {
			res, ok := TryIntegrateGaussian(v.Expr, varName)
			if ok {
				neg, err := simplifyUnaryOp("-", res)
				if err == nil {
					return neg, true
				}
			}
		}
	}

	if expArg == nil || !containsVar(expArg, varName) {
		return nil, false
	}

	// Extract polynomial coefficients of expArg with respect to varName: expArg = -A*x^2 + B*x + C
	coeffs, err := extractPolyCoeffs(expandNode(expArg), varName)
	if err != nil {
		return nil, false
	}

	deg2Coeff := coeffs[2]
	deg1Coeff := coeffs[1]
	deg0Coeff := coeffs[0]

	if deg2Coeff == nil || isZero(deg2Coeff) {
		return nil, false // Not quadratic in exponent
	}

	// Check if higher degree terms exist
	for d := range coeffs {
		if d > 2 {
			return nil, false
		}
	}

	// Standard Gaussian has negative coefficient: deg2Coeff = -a (a > 0)
	// a = -deg2Coeff
	a, err := simplifyUnaryOp("-", deg2Coeff)
	if err != nil {
		return nil, false
	}
	// Verify that a > 0 if it is rational
	if r, ok := a.(*RationalNode); ok && r.Val.Sign() <= 0 {
		return nil, false // Only positive a gives standard real erf
	}

	var b Node = mustRational(0, 1)
	if deg1Coeff != nil {
		b = deg1Coeff
	}

	var c Node = mustRational(0, 1)
	if deg0Coeff != nil {
		c = deg0Coeff
	}

	// int exp(-a*x^2 + b*x + c) dx
	// Complete square: -a*x^2 + b*x + c = -a*(x - b/(2a))^2 + c + b^2/(4a)
	// Shift factor = exp(c + b^2/(4a))
	// Integral = shift * sqrt(pi)/(2*sqrt(a)) * erf(sqrt(a)*x - b/(2*sqrt(a)))

	sqrtA, err := simplifySqrt(a)
	if err != nil {
		return nil, false
	}
	twoSqrtA, err := simplifyMul([]Node{mustRational(2, 1), sqrtA})
	if err != nil {
		return nil, false
	}
	invTwoSqrtA, err := simplifyPow(twoSqrtA, mustRational(-1, 1))
	if err != nil {
		return nil, false
	}

	sqrtPi, err := simplifySqrt(&ConstNode{Name: "pi"})
	if err != nil {
		return nil, false
	}

	// erf argument: sqrt(a)*x - b / (2*sqrt(a))
	sqrtATimesX, err := simplifyMul([]Node{sqrtA, &VarNode{Name: varName}})
	if err != nil {
		return nil, false
	}

	var erfArg Node = sqrtATimesX
	if !isZero(b) {
		bShift, err := simplifyMul([]Node{b, invTwoSqrtA})
		if err != nil {
			return nil, false
		}
		negBShift, err := simplifyUnaryOp("-", bShift)
		if err != nil {
			return nil, false
		}
		erfArg, err = simplifyAdd([]Node{sqrtATimesX, negBShift})
		if err != nil {
			return nil, false
		}
	}

	erfNode, err := ast.NewFunc("erf", []Node{erfArg})
	if err != nil {
		return nil, false
	}

	// Constant multiplier: k * sqrt(pi) / (2*sqrt(a)) * exp(c + b^2/(4a))
	var shiftMultiplier Node = mustRational(1, 1)
	if !isZero(b) || !isZero(c) {
		// b^2 / (4*a)
		bSq, _ := simplifyPow(b, mustRational(2, 1))
		fourA, _ := simplifyMul([]Node{mustRational(4, 1), a})
		invFourA, _ := simplifyPow(fourA, mustRational(-1, 1))
		bSqOver4A, _ := simplifyMul([]Node{bSq, invFourA})
		exponent, _ := simplifyAdd([]Node{c, bSqOver4A})
		if !isZero(exponent) {
			shiftMultiplier, _ = ast.NewFunc("exp", []Node{exponent})
		}
	}

	finalMul, err := simplifyMul([]Node{k, shiftMultiplier, sqrtPi, invTwoSqrtA, erfNode})
	if err != nil {
		return nil, false
	}

	return finalMul, true
}

// TryIntegrateLambertW checks if expr has the form W(a*x + b) and integrates it.
func TryIntegrateLambertW(expr Node, varName string) (Node, bool) {
	fn, ok := expr.(*FuncNode)
	if !ok || fn.Name != "lambert_w" || len(fn.Args) == 0 {
		return nil, false
	}
	arg := fn.Args[0]
	a, _, ok := isLinear(arg, varName)
	if !ok {
		return nil, false
	}

	// int W(u) du = u * W(u) - u + u / W(u)
	// dx = du / a
	wU := fn
	u := arg

	// u * W(u)
	uWU, err := simplifyMul([]Node{u, wU})
	if err != nil {
		return nil, false
	}

	// -u
	negU, err := simplifyUnaryOp("-", u)
	if err != nil {
		return nil, false
	}

	// u / W(u)
	invWU, err := simplifyPow(wU, mustRational(-1, 1))
	if err != nil {
		return nil, false
	}
	uOverWU, err := simplifyMul([]Node{u, invWU})
	if err != nil {
		return nil, false
	}

	intU, err := simplifyAdd([]Node{uWU, negU, uOverWU})
	if err != nil {
		return nil, false
	}

	invA, err := simplifyPow(a, mustRational(-1, 1))
	if err != nil {
		return nil, false
	}

	res, err := simplifyMul([]Node{invA, intU})
	if err != nil {
		return nil, false
	}
	return res, true
}

// SolveTranscendental attempts to solve expr = 0 for varName using Lambert W function.
func SolveTranscendental(expr Node, varName string) (Node, error) {
	fsm := NewTranscendentalSolverFSM()

	// Step 1: Classification
	if err := fsm.TransitionTo(TranscendentalStateClassified); err != nil {
		return nil, err
	}

	expanded := expandNode(expr)

	// Pattern A: A * x * exp(B * x) + C = 0  =>  A * x * exp(B * x) = -C
	// Also handles x * exp(x) - C = 0
	if root, ok := solveExponentialLinearPattern(expanded, varName); ok {
		_ = fsm.TransitionTo(TranscendentalStateNormalized)
		_ = fsm.TransitionTo(TranscendentalStateLambertInverted)
		_ = fsm.TransitionTo(TranscendentalStateRootIsolated)
		_ = fsm.TransitionTo(TranscendentalStateVerified)
		res := ast.NewList([]Node{root})
		RecordTraceRewrite(RuleSolveLinear, expr, res, i18n.T("transcendental.trace_solve_lambert", Format(expr), Format(res)))
		return res, nil
	}

	// Pattern B: A * x + B * ln(x) + C = 0  =>  x + ln(x) = C
	if root, ok := solveLogLinearPattern(expanded, varName); ok {
		_ = fsm.TransitionTo(TranscendentalStateNormalized)
		_ = fsm.TransitionTo(TranscendentalStateLambertInverted)
		_ = fsm.TransitionTo(TranscendentalStateRootIsolated)
		_ = fsm.TransitionTo(TranscendentalStateVerified)
		res := ast.NewList([]Node{root})
		RecordTraceRewrite(RuleSolveLinear, expr, res, i18n.T("transcendental.trace_solve_lambert", Format(expr), Format(res)))
		return res, nil
	}

	// Pattern C: x^x - A = 0  =>  x = exp(W(ln(A)))
	if root, ok := solvePowerTowerPattern(expanded, varName); ok {
		_ = fsm.TransitionTo(TranscendentalStateNormalized)
		_ = fsm.TransitionTo(TranscendentalStateLambertInverted)
		_ = fsm.TransitionTo(TranscendentalStateRootIsolated)
		_ = fsm.TransitionTo(TranscendentalStateVerified)
		res := ast.NewList([]Node{root})
		RecordTraceRewrite(RuleSolveLinear, expr, res, i18n.T("transcendental.trace_solve_lambert", Format(expr), Format(res)))
		return res, nil
	}

	_ = fsm.TransitionTo(TranscendentalStateFailed)
	return nil, fmt.Errorf("%s", i18n.T("transcendental.err_transcendental_not_solvable", expr.String()))
}

// solveExponentialLinearPattern detects (A*x + B)*exp(C*x) + D = 0 or A*x*exp(B*x) = C.
func solveExponentialLinearPattern(expr Node, varName string) (Node, bool) {
	// Separate terms into variable-containing terms and constant terms: T_var + D = 0 => T_var = -D
	var varTerms []Node
	var constTerms []Node

	if add, ok := expr.(*AddNode); ok {
		for _, t := range add.Terms {
			if containsVar(t, varName) {
				varTerms = append(varTerms, t)
			} else {
				constTerms = append(constTerms, t)
			}
		}
	} else if containsVar(expr, varName) {
		varTerms = []Node{expr}
	} else {
		return nil, false
	}

	if len(varTerms) == 0 {
		return nil, false
	}

	// D = sum(constTerms)
	var d Node = mustRational(0, 1)
	if len(constTerms) > 0 {
		sumC, err := simplifyAdd(constTerms)
		if err != nil {
			return nil, false
		}
		d = sumC
	}
	negD, err := simplifyUnaryOp("-", d)
	if err != nil {
		return nil, false
	}

	// Inspect varTerms. Case 1: single product term P(x) * exp(Q(x))
	varTerm, err := simplifyAdd(varTerms)
	if err != nil {
		return nil, false
	}

	mul, ok := varTerm.(*MulNode)
	if !ok {
		return nil, false
	}

	var polyFactor Node
	var expArg Node
	for _, f := range mul.Factors {
		if !containsVar(f, varName) {
			continue
		}
		if fn, ok := f.(*FuncNode); ok && fn.Name == "exp" && len(fn.Args) == 1 {
			expArg = fn.Args[0]
			continue
		}
		if pn, ok := f.(*PowNode); ok {
			if c, okC := pn.Base.(*ConstNode); okC && c.Name == "e" {
				expArg = pn.Exp
				continue
			}
		}
		if polyFactor == nil {
			polyFactor = f
		} else {
			// Multiple variable non-exp factors
			polyFactor, _ = simplifyMul([]Node{polyFactor, f})
		}
	}

	if polyFactor == nil || expArg == nil {
		return nil, false
	}

	// Also extract non-variable scalar in mul: mul = k * polyFactor * exp(expArg)
	var scalars []Node
	for _, f := range mul.Factors {
		if !containsVar(f, varName) {
			scalars = append(scalars, f)
		}
	}
	var k Node = mustRational(1, 1)
	if len(scalars) > 0 {
		k, _ = simplifyMul(scalars)
	}

	// Equation is: k * (A*x + B) * exp(C*x) = negD
	// polyFactor = A*x + B, expArg = C*x
	a, b, okA := isLinear(polyFactor, varName)
	if !okA {
		return nil, false
	}
	c, dLinear, okC := isLinear(expArg, varName)
	if !okC {
		return nil, false
	}

	// If expArg has constant offset: exp(C*x + dLinear) = exp(dLinear) * exp(C*x)
	if !isZero(dLinear) {
		expD, _ := ast.NewFunc("exp", []Node{dLinear})
		k, _ = simplifyMul([]Node{k, expD})
	}

	// Now: k * (A*x + B) * exp(C*x) = negD  =>  (A*x + B) * exp(C*x) = negD / k
	invK, err := simplifyPow(k, mustRational(-1, 1))
	if err != nil {
		return nil, false
	}
	rhs, err := simplifyMul([]Node{negD, invK})
	if err != nil {
		return nil, false
	}

	// Subcase 1: b == 0  =>  A * x * exp(C * x) = rhs  =>  C * x * exp(C * x) = rhs * C / A
	if isZero(b) {
		cOverA, err := simplifyMul([]Node{c, mustPow(a, -1)})
		if err != nil {
			return nil, false
		}
		wArg, err := simplifyMul([]Node{rhs, cOverA})
		if err != nil {
			return nil, false
		}
		lambertW, err := simplifyFuncWithEnv("lambert_w", []Node{wArg}, nil)
		if err != nil {
			return nil, false
		}
		// x = W(wArg) / c
		invC, err := simplifyPow(c, mustRational(-1, 1))
		if err != nil {
			return nil, false
		}
		root, err := simplifyMul([]Node{lambertW, invC})
		if err != nil {
			return nil, false
		}
		return root, true
	}

	// Subcase 2: (A*x + B)*exp(C*x) = rhs
	// (C/A)*(A*x + B)*exp(C*x) = rhs * C / A
	// (C*x + B*C/A) * exp(C*x + B*C/A) = (rhs * C / A) * exp(B*C/A)
	bCOverA, err := simplifyMul([]Node{b, c, mustPow(a, -1)})
	if err != nil {
		return nil, false
	}
	expBCOverA, err := ast.NewFunc("exp", []Node{bCOverA})
	if err != nil {
		return nil, false
	}
	cOverA, err := simplifyMul([]Node{c, mustPow(a, -1)})
	if err != nil {
		return nil, false
	}
	wArg, err := simplifyMul([]Node{rhs, cOverA, expBCOverA})
	if err != nil {
		return nil, false
	}
	lambertW, err := simplifyFuncWithEnv("lambert_w", []Node{wArg}, nil)
	if err != nil {
		return nil, false
	}
	// C*x + bCOverA = W => C*x = W - bCOverA => x = (W - bCOverA) / C
	negBCOverA, err := simplifyUnaryOp("-", bCOverA)
	if err != nil {
		return nil, false
	}
	bracket, err := simplifyAdd([]Node{lambertW, negBCOverA})
	if err != nil {
		return nil, false
	}
	invC, err := simplifyPow(c, mustRational(-1, 1))
	if err != nil {
		return nil, false
	}
	root, err := simplifyMul([]Node{bracket, invC})
	if err != nil {
		return nil, false
	}
	return root, true
}

// solveLogLinearPattern detects A*x + B*ln(x) + C = 0.
func solveLogLinearPattern(expr Node, varName string) (Node, bool) {
	add, ok := expr.(*AddNode)
	if !ok {
		return nil, false
	}

	var linearPart Node
	var lnCoeff Node
	var constTerms []Node

	for _, t := range add.Terms {
		if !containsVar(t, varName) {
			constTerms = append(constTerms, t)
			continue
		}
		// Check for ln(x) or k*ln(x)
		if fn, ok := t.(*FuncNode); ok && fn.Name == "ln" && len(fn.Args) == 1 {
			if v, okV := fn.Args[0].(*VarNode); okV && v.Name == varName {
				lnCoeff = mustRational(1, 1)
				continue
			}
		}
		if mul, ok := t.(*MulNode); ok {
			var nonLnFactors []Node
			var isLn bool
			for _, f := range mul.Factors {
				if fn, okF := f.(*FuncNode); okF && fn.Name == "ln" && len(fn.Args) == 1 {
					if v, okV := fn.Args[0].(*VarNode); okV && v.Name == varName {
						isLn = true
						continue
					}
				}
				nonLnFactors = append(nonLnFactors, f)
			}
			if isLn {
				coeff, _ := simplifyMul(nonLnFactors)
				lnCoeff = coeff
				continue
			}
		}

		// Otherwise should be linear term A*x
		if linearPart == nil {
			linearPart = t
		} else {
			linearPart, _ = simplifyAdd([]Node{linearPart, t})
		}
	}

	if linearPart == nil || lnCoeff == nil {
		return nil, false
	}

	// linearPart should be a*x
	a, b, ok := isLinear(linearPart, varName)
	if !ok {
		return nil, false
	}

	// C = sum(constTerms) + b
	allConsts := constTerms
	if !isZero(b) {
		allConsts = append(allConsts, b)
	}
	var c Node = mustRational(0, 1)
	if len(allConsts) > 0 {
		c, _ = simplifyAdd(allConsts)
	}

	// a*x + B*ln(x) + C = 0  =>  a*x/B + ln(x) = -C/B
	// exp(a*x/B + ln(x)) = exp(-C/B)  =>  x * exp(a*x/B) = exp(-C/B)
	// (a/B)*x * exp(a*x/B) = (a/B) * exp(-C/B)
	// (a/B)*x = W( (a/B) * exp(-C/B) )
	// x = (B/a) * W( (a/B) * exp(-C/B) )
	bCoeff := lnCoeff
	invB, err := simplifyPow(bCoeff, mustRational(-1, 1))
	if err != nil {
		return nil, false
	}
	aOverB, err := simplifyMul([]Node{a, invB})
	if err != nil {
		return nil, false
	}
	bOverA, err := simplifyPow(aOverB, mustRational(-1, 1))
	if err != nil {
		return nil, false
	}

	negCOverB, err := simplifyMul([]Node{mustRational(-1, 1), c, invB})
	if err != nil {
		return nil, false
	}

	var expNegCOverB Node = mustRational(1, 1)
	if !isZero(negCOverB) {
		expVal, err := simplifyFuncWithEnv("exp", []Node{negCOverB}, nil)
		if err != nil {
			return nil, false
		}
		expNegCOverB = expVal
	}

	wArg, err := simplifyMul([]Node{aOverB, expNegCOverB})
	if err != nil {
		return nil, false
	}

	lambertW, err := simplifyFuncWithEnv("lambert_w", []Node{wArg}, nil)
	if err != nil {
		return nil, false
	}

	root, err := simplifyMul([]Node{bOverA, lambertW})
	if err != nil {
		return nil, false
	}
	return root, true
}

// solvePowerTowerPattern detects x^x - A = 0  =>  x = exp(W(ln(A))) = ln(A) / W(ln(A)).
func solvePowerTowerPattern(expr Node, varName string) (Node, bool) {
	add, ok := expr.(*AddNode)
	if !ok {
		return nil, false
	}

	var powerTowerTerm Node
	var constTerms []Node

	for _, t := range add.Terms {
		if !containsVar(t, varName) {
			constTerms = append(constTerms, t)
			continue
		}
		if pn, ok := t.(*PowNode); ok {
			if vB, okB := pn.Base.(*VarNode); okB && vB.Name == varName {
				if vE, okE := pn.Exp.(*VarNode); okE && vE.Name == varName {
					powerTowerTerm = t
					continue
				}
			}
		}
		// Negated x^x
		if u, ok := t.(*UnaryOpNode); ok && u.Op == "-" {
			if pn, okP := u.Expr.(*PowNode); okP {
				if vB, okB := pn.Base.(*VarNode); okB && vB.Name == varName {
					if vE, okE := pn.Exp.(*VarNode); okE && vE.Name == varName {
						powerTowerTerm = t
						continue
					}
				}
			}
		}
	}

	if powerTowerTerm == nil || len(constTerms) == 0 {
		return nil, false
	}

	c, err := simplifyAdd(constTerms)
	if err != nil {
		return nil, false
	}

	// If x^x + C = 0 => x^x = -C = A
	negC, err := simplifyUnaryOp("-", c)
	if err != nil {
		return nil, false
	}
	a := negC
	if _, ok := powerTowerTerm.(*UnaryOpNode); ok {
		// -x^x + C = 0 => x^x = C
		a = c
	}

	// x^x = A  =>  x * ln(x) = ln(A)  =>  ln(x) * exp(ln(x)) = ln(A)
	// ln(x) = W(ln(A))  =>  x = exp(W(ln(A)))
	lnA, err := ast.NewFunc("ln", []Node{a})
	if err != nil {
		return nil, false
	}
	lambertW, err := ast.NewFunc("lambert_w", []Node{lnA})
	if err != nil {
		return nil, false
	}
	root, err := ast.NewFunc("exp", []Node{lambertW})
	if err != nil {
		return nil, false
	}
	return root, true
}

func mustPow(base Node, exp int64) Node {
	res, err := simplifyPow(base, mustRational(exp, 1))
	if err != nil {
		return &PowNode{Base: base, Exp: mustRational(exp, 1)}
	}
	return res
}
