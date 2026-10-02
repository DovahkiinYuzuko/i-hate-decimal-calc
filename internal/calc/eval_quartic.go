package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// QuarticState refers to the Quartic lifecycle states defined in eval_quartic_fsm.go.

// SolveQuarticExact finds all exact algebraic roots of a general quartic equation:
// a4 * x^4 + a3 * x^3 + a2 * x^2 + a1 * x + a0 = 0.
// It integrates the Rational Root Theorem (Fast-Path), Biquadratic direct solving,
// and falls back to Ferrari's method (Ferrari 1540) via Tschirnhaus transformation
// and Resolvent Cubic solving for irreducible quartics.
func SolveQuarticExact(a4, a3, a2, a1, a0 Node) ([]Node, error) {
	fsm := NewQuarticLifecycleFSM()
	return SolveQuarticExactWithFSM(a4, a3, a2, a1, a0, fsm)
}

// SolveQuarticExactWithFSM finds exact algebraic roots governed by a QuarticLifecycleFSM.
func SolveQuarticExactWithFSM(a4, a3, a2, a1, a0 Node, fsm *QuarticLifecycleFSM) ([]Node, error) {
	if a4 == nil || a3 == nil || a2 == nil || a1 == nil || a0 == nil {
		_ = fsm.TransitionTo(QuarticStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("quartic.err_nil_coefficients"))
	}

	a4 = EvalOrSelf(a4)
	a3 = EvalOrSelf(a3)
	a2 = EvalOrSelf(a2)
	a1 = EvalOrSelf(a1)
	a0 = EvalOrSelf(a0)

	// Degree check: if a4 is zero, reduce to cubic equation
	if isZero(a4) {
		_ = fsm.TransitionTo(QuarticStateDegreeValidated)
		roots, err := SolveCubicExact(a3, a2, a1, a0)
		if err != nil {
			_ = fsm.TransitionTo(QuarticStateFailed)
			return nil, err
		}
		_ = fsm.TransitionTo(QuarticStateRootsConstructed)
		return roots, nil
	}

	_ = fsm.TransitionTo(QuarticStateDegreeValidated)

	// 1. Fast-Path: Rational Root Theorem & synthetic division if all coefficients are rational
	r4, ok4 := a4.(*RationalNode)
	r3, ok3 := a3.(*RationalNode)
	r2, ok2 := a2.(*RationalNode)
	r1, ok1 := a1.(*RationalNode)
	r0, ok0 := a0.(*RationalNode)

	if ok4 && ok3 && ok2 && ok1 && ok0 {
		commonLcm := lcmInt(r4.Val.Denom(), r3.Val.Denom())
		commonLcm = lcmInt(commonLcm, r2.Val.Denom())
		commonLcm = lcmInt(commonLcm, r1.Val.Denom())
		commonLcm = lcmInt(commonLcm, r0.Val.Denom())

		c4 := new(big.Int).Mul(r4.Val.Num(), new(big.Int).Div(commonLcm, r4.Val.Denom()))
		c3 := new(big.Int).Mul(r3.Val.Num(), new(big.Int).Div(commonLcm, r3.Val.Denom()))
		c2 := new(big.Int).Mul(r2.Val.Num(), new(big.Int).Div(commonLcm, r2.Val.Denom()))
		c1 := new(big.Int).Mul(r1.Val.Num(), new(big.Int).Div(commonLcm, r1.Val.Denom()))
		c0 := new(big.Int).Mul(r0.Val.Num(), new(big.Int).Div(commonLcm, r0.Val.Denom()))

		intPoly := []*big.Int{c0, c1, c2, c3, c4}

		// Handle root x = 0 (c0 == 0)
		if c0.Sign() == 0 {
			_ = fsm.TransitionTo(QuarticStateRationalDeflated)
			cRoots, err := SolveCubicExact(&RationalNode{Val: new(big.Rat).SetInt(c4)},
				&RationalNode{Val: new(big.Rat).SetInt(c3)},
				&RationalNode{Val: new(big.Rat).SetInt(c2)},
				&RationalNode{Val: new(big.Rat).SetInt(c1)})
			if err == nil {
				roots := append([]Node{mustRational(0, 1)}, cRoots...)
				_ = fsm.TransitionTo(QuarticStateRootsConstructed)
				return roots, nil
			}
		}

		candRoots := findRationalRoots(intPoly)
		if len(candRoots) > 0 {
			r := candRoots[0]
			quot, ok := syntheticDivide(intPoly, r.num, r.denom)
			if ok && len(quot) == 4 {
				_ = fsm.TransitionTo(QuarticStateRationalDeflated)
				rootRat := new(big.Rat).SetFrac(r.num, r.denom)
				qA := &RationalNode{Val: new(big.Rat).SetInt(quot[3])}
				qB := &RationalNode{Val: new(big.Rat).SetInt(quot[2])}
				qC := &RationalNode{Val: new(big.Rat).SetInt(quot[1])}
				qD := &RationalNode{Val: new(big.Rat).SetInt(quot[0])}

				cRoots, err := SolveCubicExact(qA, qB, qC, qD)
				if err == nil {
					roots := append([]Node{&RationalNode{Val: rootRat}}, cRoots...)
					_ = fsm.TransitionTo(QuarticStateRootsConstructed)
					return roots, nil
				}
			}
		}
	}

	// 2. Tschirnhaus Transformation
	// Make monic: x^4 + a*x^3 + b*x^2 + c*x + d = 0
	invA4, err := simplifyPow(a4, mustRational(-1, 1))
	if err != nil {
		_ = fsm.TransitionTo(QuarticStateFailed)
		return nil, err
	}
	a, _ := simplifyMul([]Node{a3, invA4})
	a = EvalOrSelf(a)
	b, _ := simplifyMul([]Node{a2, invA4})
	b = EvalOrSelf(b)
	c, _ := simplifyMul([]Node{a1, invA4})
	c = EvalOrSelf(c)
	d, _ := simplifyMul([]Node{a0, invA4})
	d = EvalOrSelf(d)

	// Shift: x = t - a/4
	shift, _ := simplifyMul([]Node{a, mustRational(1, 4)})
	shift = EvalOrSelf(shift)
	negShift, _ := simplifyUnaryOp("-", shift)
	negShift = EvalOrSelf(negShift)

	// Depressed Quartic: t^4 + p*t^2 + q*t + r = 0
	// p = b - 3*a^2 / 8
	aSq, _ := simplifyPow(a, mustRational(2, 1))
	threeASqOver8, _ := simplifyMul([]Node{aSq, mustRational(3, 8)})
	negThreeASqOver8, _ := simplifyUnaryOp("-", threeASqOver8)
	pNode, _ := simplifyAdd([]Node{b, negThreeASqOver8})
	pNode = EvalOrSelf(pNode)

	// q = c - a*b / 2 + a^3 / 8
	ab, _ := simplifyMul([]Node{a, b})
	halfAB, _ := simplifyMul([]Node{ab, mustRational(1, 2)})
	negHalfAB, _ := simplifyUnaryOp("-", halfAB)
	aCube, _ := simplifyPow(a, mustRational(3, 1))
	aCubeOver8, _ := simplifyMul([]Node{aCube, mustRational(1, 8)})
	qNode, _ := simplifyAdd([]Node{c, negHalfAB, aCubeOver8})
	qNode = EvalOrSelf(qNode)

	// r = d - a*c / 4 + a^2 * b / 16 - 3*a^4 / 256
	ac, _ := simplifyMul([]Node{a, c})
	acOver4, _ := simplifyMul([]Node{ac, mustRational(1, 4)})
	negAcOver4, _ := simplifyUnaryOp("-", acOver4)
	aSqB, _ := simplifyMul([]Node{aSq, b})
	aSqBOver16, _ := simplifyMul([]Node{aSqB, mustRational(1, 16)})
	aQuad, _ := simplifyPow(a, mustRational(4, 1))
	threeAQuadOver256, _ := simplifyMul([]Node{aQuad, mustRational(3, 256)})
	negThreeAQuadOver256, _ := simplifyUnaryOp("-", threeAQuadOver256)
	rNode, _ := simplifyAdd([]Node{d, negAcOver4, aSqBOver16, negThreeAQuadOver256})
	rNode = EvalOrSelf(rNode)

	// 3. Fast-Path: Biquadratic equation (q == 0)
	if isZero(qNode) {
		_ = fsm.TransitionTo(QuarticStateBiquadraticResolved)
		// Solve u^2 + p*u + r = 0 where u = t^2
		uRoots, err := solveQuadraticExact(mustRational(1, 1), pNode, rNode)
		if err == nil && len(uRoots) == 2 {
			var finalRoots []Node
			for _, u := range uRoots {
				sqrtU, err := simplifySqrt(u)
				if err != nil {
					// Fall back to general power if simplifySqrt fails
					sqrtU, _ = simplifyPow(u, mustRational(1, 2))
				}
				sqrtU = EvalOrSelf(sqrtU)
				negSqrtU, _ := simplifyUnaryOp("-", sqrtU)
				negSqrtU = EvalOrSelf(negSqrtU)

				x1, _ := simplifyAdd([]Node{sqrtU, negShift})
				x2, _ := simplifyAdd([]Node{negSqrtU, negShift})
				finalRoots = append(finalRoots, EvalOrSelf(x1), EvalOrSelf(x2))
			}
			if len(finalRoots) == 4 {
				_ = fsm.TransitionTo(QuarticStateRootsConstructed)
				return finalRoots, nil
			}
		}
	}

	// 4. Ferrari's Resolvent Cubic Method (q != 0)
	_ = fsm.TransitionTo(QuarticStateDepressedNormalized)

	// Resolvent Cubic: 8*y^3 - 4*p*y^2 - 8*r*y + (4*p*r - q^2) = 0
	coeffY3 := mustRational(8, 1)

	fourP, _ := simplifyMul([]Node{mustRational(4, 1), pNode})
	coeffY2, _ := simplifyUnaryOp("-", fourP)
	coeffY2 = EvalOrSelf(coeffY2)

	eightR, _ := simplifyMul([]Node{mustRational(8, 1), rNode})
	coeffY1, _ := simplifyUnaryOp("-", eightR)
	coeffY1 = EvalOrSelf(coeffY1)

	fourPR, _ := simplifyMul([]Node{mustRational(4, 1), pNode, rNode})
	qSq, _ := simplifyPow(qNode, mustRational(2, 1))
	negQSq, _ := simplifyUnaryOp("-", qSq)
	coeffY0, _ := simplifyAdd([]Node{fourPR, negQSq})
	coeffY0 = EvalOrSelf(coeffY0)

	cubicRoots, err := SolveCubicExact(coeffY3, coeffY2, coeffY1, coeffY0)
	if err != nil || len(cubicRoots) == 0 {
		_ = fsm.TransitionTo(QuarticStateFailed)
		return nil, fmt.Errorf("%s: %w", i18n.T("quartic.err_cannot_solve_quartic", err), err)
	}

	// Pick a cubic root y such that 2*y - p != 0 (since q != 0, such a root always exists)
	var chosenY Node
	var chosenR Node

	for _, yCand := range cubicRoots {
		twoY, _ := simplifyMul([]Node{mustRational(2, 1), yCand})
		negP, _ := simplifyUnaryOp("-", pNode)
		twoYMinusP, _ := simplifyAdd([]Node{twoY, negP})
		twoYMinusP = EvalOrSelf(twoYMinusP)

		if isZero(twoYMinusP) {
			continue
		}

		rCand, err := simplifySqrt(twoYMinusP)
		if err != nil {
			rCand, _ = simplifyPow(twoYMinusP, mustRational(1, 2))
		}
		rCand = EvalOrSelf(rCand)
		if !isZero(rCand) {
			chosenY = yCand
			chosenR = rCand
			break
		}
	}

	if chosenR == nil {
		_ = fsm.TransitionTo(QuarticStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("quartic.err_cannot_solve_quartic", "degenerate resolvent cubic"))
	}

	_ = fsm.TransitionTo(QuarticStateResolventCubicSolved)

	// Solve the two quadratic equations:
	// Quadratic 1: t^2 + R*t + (y - q / (2*R)) = 0
	// Quadratic 2: t^2 - R*t + (y + q / (2*R)) = 0
	twoR, _ := simplifyMul([]Node{mustRational(2, 1), chosenR})
	invTwoR, _ := simplifyPow(twoR, mustRational(-1, 1))
	qOverTwoR, _ := simplifyMul([]Node{qNode, invTwoR})
	qOverTwoR = EvalOrSelf(qOverTwoR)
	negQOverTwoR, _ := simplifyUnaryOp("-", qOverTwoR)
	negQOverTwoR = EvalOrSelf(negQOverTwoR)

	const1, _ := simplifyAdd([]Node{chosenY, negQOverTwoR})
	const1 = EvalOrSelf(const1)

	const2, _ := simplifyAdd([]Node{chosenY, qOverTwoR})
	const2 = EvalOrSelf(const2)

	negR, _ := simplifyUnaryOp("-", chosenR)
	negR = EvalOrSelf(negR)

	tRoots1, err1 := solveQuadraticExact(mustRational(1, 1), chosenR, const1)
	if err1 != nil {
		_ = fsm.TransitionTo(QuarticStateFailed)
		return nil, fmt.Errorf("%s: %w", i18n.T("quartic.err_cannot_solve_quartic", err1), err1)
	}

	tRoots2, err2 := solveQuadraticExact(mustRational(1, 1), negR, const2)
	if err2 != nil {
		_ = fsm.TransitionTo(QuarticStateFailed)
		return nil, fmt.Errorf("%s: %w", i18n.T("quartic.err_cannot_solve_quartic", err2), err2)
	}

	allTRoots := append(tRoots1, tRoots2...)
	var finalRoots []Node
	for _, t := range allTRoots {
		x, _ := simplifyAdd([]Node{t, negShift})
		finalRoots = append(finalRoots, EvalOrSelf(x))
	}

	_ = fsm.TransitionTo(QuarticStateRootsConstructed)
	return finalRoots, nil
}
