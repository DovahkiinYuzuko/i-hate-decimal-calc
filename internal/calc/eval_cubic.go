package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// CubicState refers to the Cubic lifecycle states defined in eval_cubic_fsm.go.

// SolveCubicExact finds all exact algebraic roots of a general cubic equation:
// a3 * x^3 + a2 * x^2 + a1 * x + a0 = 0.
// It integrates the Rational Root Theorem as a fast-path and falls back to
// Cardano's formula (Cardano 1545) with Tschirnhaus transformation for irreducible cubics.
func SolveCubicExact(a3, a2, a1, a0 Node) ([]Node, error) {
	fsm := NewCubicLifecycleFSM()
	return SolveCubicExactWithFSM(a3, a2, a1, a0, fsm)
}

// SolveCubicExactWithFSM finds exact algebraic roots governed by a CubicLifecycleFSM.
func SolveCubicExactWithFSM(a3, a2, a1, a0 Node, fsm *CubicLifecycleFSM) ([]Node, error) {
	if a3 == nil || a2 == nil || a1 == nil || a0 == nil {
		_ = fsm.TransitionTo(CubicStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("cubic.err_nil_coefficients"))
	}

	a3 = EvalOrSelf(a3)
	a2 = EvalOrSelf(a2)
	a1 = EvalOrSelf(a1)
	a0 = EvalOrSelf(a0)

	// Degree check: if a3 is zero, reduce to quadratic equation
	if isZero(a3) {
		_ = fsm.TransitionTo(CubicStateDegreeValidated)
		roots, err := solveQuadraticExact(a2, a1, a0)
		if err != nil {
			_ = fsm.TransitionTo(CubicStateFailed)
			return nil, err
		}
		_ = fsm.TransitionTo(CubicStateRootsConstructed)
		return roots, nil
	}

	_ = fsm.TransitionTo(CubicStateDegreeValidated)

	// 1. Fast-Path: Rational Root Theorem & synthetic division if all coefficients are rational
	r3, ok3 := a3.(*RationalNode)
	r2, ok2 := a2.(*RationalNode)
	r1, ok1 := a1.(*RationalNode)
	r0, ok0 := a0.(*RationalNode)

	if ok3 && ok2 && ok1 && ok0 {
		commonLcm := lcmInt(r3.Val.Denom(), r2.Val.Denom())
		commonLcm = lcmInt(commonLcm, r1.Val.Denom())
		commonLcm = lcmInt(commonLcm, r0.Val.Denom())

		c3 := new(big.Int).Mul(r3.Val.Num(), new(big.Int).Div(commonLcm, r3.Val.Denom()))
		c2 := new(big.Int).Mul(r2.Val.Num(), new(big.Int).Div(commonLcm, r2.Val.Denom()))
		c1 := new(big.Int).Mul(r1.Val.Num(), new(big.Int).Div(commonLcm, r1.Val.Denom()))
		c0 := new(big.Int).Mul(r0.Val.Num(), new(big.Int).Div(commonLcm, r0.Val.Denom()))

		intPoly := []*big.Int{c0, c1, c2, c3}

		// Handle root x = 0 (c0 == 0)
		if c0.Sign() == 0 {
			_ = fsm.TransitionTo(CubicStateRationalDeflated)
			qRoots, err := solveQuadraticExact(&RationalNode{Val: new(big.Rat).SetInt(c3)},
				&RationalNode{Val: new(big.Rat).SetInt(c2)},
				&RationalNode{Val: new(big.Rat).SetInt(c1)})
			if err == nil {
				roots := append([]Node{mustRational(0, 1)}, qRoots...)
				_ = fsm.TransitionTo(CubicStateRootsConstructed)
				return roots, nil
			}
		}

		candRoots := findRationalRoots(intPoly)
		if len(candRoots) > 0 {
			r := candRoots[0]
			quot, ok := syntheticDivide(intPoly, r.num, r.denom)
			if ok && len(quot) == 3 {
				_ = fsm.TransitionTo(CubicStateRationalDeflated)
				rootRat := new(big.Rat).SetFrac(r.num, r.denom)
				qA := &RationalNode{Val: new(big.Rat).SetInt(quot[2])}
				qB := &RationalNode{Val: new(big.Rat).SetInt(quot[1])}
				qC := &RationalNode{Val: new(big.Rat).SetInt(quot[0])}

				qRoots, err := solveQuadraticExact(qA, qB, qC)
				if err == nil {
					roots := append([]Node{&RationalNode{Val: rootRat}}, qRoots...)
					_ = fsm.TransitionTo(CubicStateRootsConstructed)
					return roots, nil
				}
			}
		}
	}

	// 2. Cardano's Formula with Tschirnhaus Transformation
	// Divide by leading coefficient a3 to get monic: x^3 + b*x^2 + c*x + d = 0
	invA3, err := simplifyPow(a3, mustRational(-1, 1))
	if err != nil {
		_ = fsm.TransitionTo(CubicStateFailed)
		return nil, err
	}
	b, _ := simplifyMul([]Node{a2, invA3})
	b = EvalOrSelf(b)
	c, _ := simplifyMul([]Node{a1, invA3})
	c = EvalOrSelf(c)
	d, _ := simplifyMul([]Node{a0, invA3})
	d = EvalOrSelf(d)

	// Tschirnhaus transformation: x = t - b/3
	shift, _ := simplifyMul([]Node{b, mustRational(1, 3)})
	shift = EvalOrSelf(shift)
	negShift, _ := simplifyUnaryOp("-", shift)
	negShift = EvalOrSelf(negShift)

	// Standard depressed cubic: t^3 + p*t + q = 0
	// p = c - b^2 / 3
	bSq, _ := simplifyPow(b, mustRational(2, 1))
	bSqOver3, _ := simplifyMul([]Node{bSq, mustRational(1, 3)})
	negBSqOver3, _ := simplifyUnaryOp("-", bSqOver3)
	pNode, _ := simplifyAdd([]Node{c, negBSqOver3})
	pNode = EvalOrSelf(pNode)

	// q = d - b*c / 3 + 2*b^3 / 27
	bc, _ := simplifyMul([]Node{b, c})
	bcOver3, _ := simplifyMul([]Node{bc, mustRational(1, 3)})
	negBcOver3, _ := simplifyUnaryOp("-", bcOver3)
	bCube, _ := simplifyPow(b, mustRational(3, 1))
	twoBCubeOver27, _ := simplifyMul([]Node{bCube, mustRational(2, 27)})
	qNode, _ := simplifyAdd([]Node{d, negBcOver3, twoBCubeOver27})
	qNode = EvalOrSelf(qNode)

	_ = fsm.TransitionTo(CubicStateDepressedNormalized)

	// Special case A: p == 0 && q == 0 => t^3 = 0 => triple root x = -shift
	if isZero(pNode) && isZero(qNode) {
		_ = fsm.TransitionTo(CubicStateRootsConstructed)
		return []Node{negShift, negShift, negShift}, nil
	}

	// Special case B: q == 0 (and p != 0) => t * (t^2 + p) = 0
	if isZero(qNode) {
		negP, _ := simplifyUnaryOp("-", pNode)
		sqrtNegP, err := simplifySqrt(negP)
		if err == nil {
			negSqrtNegP, _ := simplifyUnaryOp("-", sqrtNegP)
			x1 := negShift
			x2, _ := simplifyAdd([]Node{sqrtNegP, negShift})
			x3, _ := simplifyAdd([]Node{negSqrtNegP, negShift})
			_ = fsm.TransitionTo(CubicStateRootsConstructed)
			return []Node{EvalOrSelf(x1), EvalOrSelf(x2), EvalOrSelf(x3)}, nil
		}
	}

	// Special case C: p == 0 (and q != 0) => t^3 = -q
	if isZero(pNode) {
		negQ, _ := simplifyUnaryOp("-", qNode)
		u, err := simplifyCbrt(negQ)
		if err == nil {
			t1 := u
			omega, omega2 := primitiveCubeRootsOfUnity()
			t2, _ := simplifyMul([]Node{omega, u})
			t3, _ := simplifyMul([]Node{omega2, u})

			x1, _ := simplifyAdd([]Node{t1, negShift})
			x2, _ := simplifyAdd([]Node{t2, negShift})
			x3, _ := simplifyAdd([]Node{t3, negShift})
			_ = fsm.TransitionTo(CubicStateRootsConstructed)
			return []Node{EvalOrSelf(x1), EvalOrSelf(x2), EvalOrSelf(x3)}, nil
		}
	}

	// General Cardano's Formula:
	// Delta = (q/2)^2 + (p/3)^3
	halfQ, _ := simplifyMul([]Node{qNode, mustRational(1, 2)})
	halfQSq, _ := simplifyPow(halfQ, mustRational(2, 1))
	thirdP, _ := simplifyMul([]Node{pNode, mustRational(1, 3)})
	thirdPCube, _ := simplifyPow(thirdP, mustRational(3, 1))
	delta, _ := simplifyAdd([]Node{halfQSq, thirdPCube})
	delta = EvalOrSelf(delta)

	negHalfQ, _ := simplifyUnaryOp("-", halfQ)
	negHalfQ = EvalOrSelf(negHalfQ)

	var sqrtDelta, negSqrtDelta Node

	// Determine sign of Delta for exact radical branch handling
	deltaSign := exactSignEval(delta, 0)
	if deltaSign < 0 {
		// Casus Irreducibilis: Delta < 0 => sqrt(Delta) = i * sqrt(|Delta|)
		absDelta, _ := simplifyUnaryOp("-", delta)
		sqrtAbsDelta, err := simplifySqrt(absDelta)
		if err != nil {
			_ = fsm.TransitionTo(CubicStateFailed)
			return nil, err
		}
		iUnit := &ConstNode{Name: "i"}
		sqrtDelta, _ = simplifyMul([]Node{iUnit, sqrtAbsDelta})
		sqrtDelta = EvalOrSelf(sqrtDelta)
		negSqrtDelta, _ = simplifyUnaryOp("-", sqrtDelta)
		negSqrtDelta = EvalOrSelf(negSqrtDelta)
	} else {
		var err error
		sqrtDelta, err = simplifySqrt(delta)
		if err != nil {
			_ = fsm.TransitionTo(CubicStateFailed)
			return nil, err
		}
		sqrtDelta = EvalOrSelf(sqrtDelta)
		negSqrtDelta, _ = simplifyUnaryOp("-", sqrtDelta)
		negSqrtDelta = EvalOrSelf(negSqrtDelta)
	}

	// u = cbrt(-q/2 + sqrt(Delta))
	// v = cbrt(-q/2 - sqrt(Delta))
	uRad, _ := simplifyAdd([]Node{negHalfQ, sqrtDelta})
	uRad = EvalOrSelf(uRad)
	vRad, _ := simplifyAdd([]Node{negHalfQ, negSqrtDelta})
	vRad = EvalOrSelf(vRad)

	u, err := simplifyCbrt(uRad)
	if err != nil {
		_ = fsm.TransitionTo(CubicStateFailed)
		return nil, err
	}
	v, err := simplifyCbrt(vRad)
	if err != nil {
		_ = fsm.TransitionTo(CubicStateFailed)
		return nil, err
	}

	_ = fsm.TransitionTo(CubicStateCardanoResolved)

	// t1 = u + v
	t1, _ := simplifyAdd([]Node{u, v})
	t1 = EvalOrSelf(t1)

	// t2 = -(u+v)/2 + (u-v)/2 * sqrt(3) * i
	// t3 = -(u+v)/2 - (u-v)/2 * sqrt(3) * i
	negHalfUPlusV, _ := simplifyMul([]Node{mustRational(-1, 2), t1})
	negHalfUPlusV = EvalOrSelf(negHalfUPlusV)

	negV, _ := simplifyUnaryOp("-", v)
	uMinusV, _ := simplifyAdd([]Node{u, negV})
	uMinusV = EvalOrSelf(uMinusV)

	halfUMinusV, _ := simplifyMul([]Node{mustRational(1, 2), uMinusV})
	halfUMinusV = EvalOrSelf(halfUMinusV)

	sqrt3, _ := simplifySqrt(mustRational(3, 1))
	iUnit := &ConstNode{Name: "i"}
	imagPart, _ := simplifyMul([]Node{halfUMinusV, sqrt3, iUnit})
	imagPart = EvalOrSelf(imagPart)
	negImagPart, _ := simplifyUnaryOp("-", imagPart)
	negImagPart = EvalOrSelf(negImagPart)

	t2, _ := simplifyAdd([]Node{negHalfUPlusV, imagPart})
	t2 = EvalOrSelf(t2)
	t3, _ := simplifyAdd([]Node{negHalfUPlusV, negImagPart})
	t3 = EvalOrSelf(t3)

	// Shift back: x_k = t_k - b/3 = t_k + negShift
	x1, _ := simplifyAdd([]Node{t1, negShift})
	x1 = EvalOrSelf(x1)
	x2, _ := simplifyAdd([]Node{t2, negShift})
	x2 = EvalOrSelf(x2)
	x3, _ := simplifyAdd([]Node{t3, negShift})
	x3 = EvalOrSelf(x3)

	_ = fsm.TransitionTo(CubicStateRootsConstructed)
	return []Node{x1, x2, x3}, nil
}

// simplifyCbrt simplifies cube root of a node.
// If n is a perfect cube rational, it extracts the exact rational root using intCbrt.
func simplifyCbrt(n Node) (Node, error) {
	n = EvalOrSelf(n)
	if isZero(n) {
		return mustRational(0, 1), nil
	}

	if r, ok := n.(*RationalNode); ok {
		if r.Val.Sign() < 0 {
			pos := new(big.Rat).Neg(r.Val)
			inner, err := simplifyCbrt(&RationalNode{Val: pos})
			if err != nil {
				return nil, err
			}
			return simplifyUnaryOp("-", inner)
		}

		numCbrt := intCbrt(r.Val.Num())
		denomCbrt := intCbrt(r.Val.Denom())

		numCube := new(big.Int).Mul(numCbrt, new(big.Int).Mul(numCbrt, numCbrt))
		denomCube := new(big.Int).Mul(denomCbrt, new(big.Int).Mul(denomCbrt, denomCbrt))

		if numCube.Cmp(r.Val.Num()) == 0 && denomCube.Cmp(r.Val.Denom()) == 0 {
			return &RationalNode{Val: new(big.Rat).SetFrac(numCbrt, denomCbrt)}, nil
		}
	}

	return simplifyPow(n, mustRational(1, 3))
}

// primitiveCubeRootsOfUnity returns omega and omega^2:
// omega = -1/2 + sqrt(3)/2 * i
// omega^2 = -1/2 - sqrt(3)/2 * i
func primitiveCubeRootsOfUnity() (Node, Node) {
	sqrt3, _ := simplifySqrt(mustRational(3, 1))
	halfSqrt3, _ := simplifyMul([]Node{mustRational(1, 2), sqrt3})
	iUnit := &ConstNode{Name: "i"}
	imag, _ := simplifyMul([]Node{halfSqrt3, iUnit})
	negImag, _ := simplifyUnaryOp("-", imag)

	omega, _ := simplifyAdd([]Node{mustRational(-1, 2), imag})
	omega2, _ := simplifyAdd([]Node{mustRational(-1, 2), negImag})
	return EvalOrSelf(omega), EvalOrSelf(omega2)
}

// EvalOrSelf evaluates a node or returns it if evaluation fails.
func EvalOrSelf(n Node) Node {
	if n == nil {
		return nil
	}
	evaled, err := Eval(n)
	if err == nil {
		return evaled
	}
	return n
}

