package calc

import (
	"fmt"
	"math/big"
)

// -------------------------------------------------------------------------
// Sqrt Simplification (Square-free reduction & Complex promotion)
// -------------------------------------------------------------------------

func simplifySqrt(radicand Node) (Node, error) {
	rat, ok := radicand.(*RationalNode)
	if !ok {
		// Non-rational radicand: check if it can be denested (Borodin 1985)
		if add, isAdd := radicand.(*AddNode); isAdd {
			if denested, ok := denestSqrtBinomial(add); ok {
				RecordTraceRewrite(RuleDenestRadical, NewSqrt(radicand), denested, "Borodinアルゴリズムによる二重根号の簡約")
				return denested, nil
			}
		}
		return NewSqrt(radicand), nil
	}

	// Negative radicand -> Complex promotion: sqrt(-x) = i * sqrt(x)
	if rat.Val.Sign() < 0 {
		posRat := new(big.Rat).Neg(rat.Val)
		simplifiedPos, err := simplifySqrt(&RationalNode{Val: posRat})
		if err != nil {
			return nil, err
		}
		zero, _ := NewRational(0, 1)
		return NewComplex(zero, simplifiedPos), nil
	}

	// Zero
	if rat.Val.Sign() == 0 {
		return NewRational(0, 1)
	}

	num := rat.Val.Num()
	denom := rat.Val.Denom()

	// sqrt(num / denom) = sqrt(num * denom) / denom
	// Let N = num * denom
	n := new(big.Int).Mul(num, denom)

	// Extract square factors from N: N = out^2 * rem
	outInt, remInt := extractSquareFree(n)

	// Result fraction: outInt / denom
	coeffRat := new(big.Rat).SetFrac(outInt, denom)

	if remInt.Cmp(big.NewInt(1)) == 0 {
		// Fully rational! e.g. sqrt(4) = 2, sqrt(4/9) = 2/3
		return &RationalNode{Val: coeffRat}, nil
	}

	// Radical remains: coeff * sqrt(remInt)
	remRat := new(big.Rat).SetInt(remInt)
	sqrtNode := NewSqrt(&RationalNode{Val: remRat})

	one := big.NewRat(1, 1)
	if coeffRat.Cmp(one) == 0 {
		return sqrtNode, nil
	}

	return NewMul([]Node{&RationalNode{Val: coeffRat}, sqrtNode}), nil
}

// denestSqrtBinomial attempts to denest sqrt(a + b*sqrt(r)) using Borodin (1985) algorithm.
func denestSqrtBinomial(add *AddNode) (Node, bool) {
	if len(add.Terms) != 2 {
		return nil, false
	}

	var aRat *big.Rat
	var bRat *big.Rat
	var rRat *big.Rat

	if r0, ok := add.Terms[0].(*RationalNode); ok {
		aRat = r0.Val
		b, r, okSqrt := extractSqrtTerm(add.Terms[1])
		if !okSqrt {
			return nil, false
		}
		bRat, rRat = b, r
	} else if r1, ok := add.Terms[1].(*RationalNode); ok {
		aRat = r1.Val
		b, r, okSqrt := extractSqrtTerm(add.Terms[0])
		if !okSqrt {
			return nil, false
		}
		bRat, rRat = b, r
	} else {
		return nil, false
	}

	// For real denesting, a must be positive
	if aRat.Sign() <= 0 {
		return nil, false
	}

	// d = a^2 - b^2 * r
	a2 := new(big.Rat).Mul(aRat, aRat)
	b2 := new(big.Rat).Mul(bRat, bRat)
	b2r := new(big.Rat).Mul(b2, rRat)
	d := new(big.Rat).Sub(a2, b2r)

	if d.Sign() <= 0 {
		return nil, false
	}

	sqrtD, okSquare := isRatPerfectSquare(d)
	if !okSquare {
		return nil, false
	}

	// t1 = (a + sqrtD) / 2
	two := big.NewRat(2, 1)
	t1 := new(big.Rat).Add(aRat, sqrtD)
	t1.Quo(t1, two)

	// t2 = (a - sqrtD) / 2
	t2 := new(big.Rat).Sub(aRat, sqrtD)
	t2.Quo(t2, two)

	if t1.Sign() <= 0 || t2.Sign() <= 0 {
		return nil, false
	}

	s1, err1 := simplifySqrt(&RationalNode{Val: t1})
	if err1 != nil {
		return nil, false
	}
	s2, err2 := simplifySqrt(&RationalNode{Val: t2})
	if err2 != nil {
		return nil, false
	}

	if bRat.Sign() > 0 {
		res, err := simplifyAdd([]Node{s1, s2})
		if err != nil {
			return nil, false
		}
		return res, true
	} else {
		negS2, err := simplifyUnaryOp("-", s2)
		if err != nil {
			return nil, false
		}
		res, err := simplifyAdd([]Node{s1, negS2})
		if err != nil {
			return nil, false
		}
		return res, true
	}
}

func extractSqrtTerm(n Node) (*big.Rat, *big.Rat, bool) {
	if s, ok := n.(*SqrtNode); ok {
		if rRat, ok := s.Radicand.(*RationalNode); ok {
			return big.NewRat(1, 1), rRat.Val, true
		}
		return nil, nil, false
	}
	if u, ok := n.(*UnaryOpNode); ok && u.Op == "-" {
		if s, ok := u.Expr.(*SqrtNode); ok {
			if rRat, ok := s.Radicand.(*RationalNode); ok {
				return big.NewRat(-1, 1), rRat.Val, true
			}
		}
		return nil, nil, false
	}
	if m, ok := n.(*MulNode); ok && len(m.Factors) == 2 {
		r1, ok1 := m.Factors[0].(*RationalNode)
		s2, ok2 := m.Factors[1].(*SqrtNode)
		if ok1 && ok2 {
			if rRat, ok := s2.Radicand.(*RationalNode); ok {
				return r1.Val, rRat.Val, true
			}
		}
		s1, ok1 := m.Factors[0].(*SqrtNode)
		r2, ok2 := m.Factors[1].(*RationalNode)
		if ok1 && ok2 {
			if rRat, ok := s1.Radicand.(*RationalNode); ok {
				return r2.Val, rRat.Val, true
			}
		}
	}
	return nil, nil, false
}

func isRatPerfectSquare(r *big.Rat) (*big.Rat, bool) {
	if r.Sign() <= 0 {
		return nil, false
	}
	num := r.Num()
	denom := r.Denom()

	sqrtNum := new(big.Int).Sqrt(num)
	if new(big.Int).Mul(sqrtNum, sqrtNum).Cmp(num) != 0 {
		return nil, false
	}
	sqrtDenom := new(big.Int).Sqrt(denom)
	if new(big.Int).Mul(sqrtDenom, sqrtDenom).Cmp(denom) != 0 {
		return nil, false
	}

	return new(big.Rat).SetFrac(sqrtNum, sqrtDenom), true
}

// extractSquareFree extracts perfect square factors: n = out^2 * rem
func extractSquareFree(n *big.Int) (*big.Int, *big.Int) {
	val := new(big.Int).Set(n)
	out := big.NewInt(1)

	// Factor out 2^2
	two := big.NewInt(2)
	four := big.NewInt(4)
	rem := new(big.Int)
	for {
		rem.Mod(val, four)
		if rem.Sign() == 0 {
			out.Mul(out, two)
			val.Div(val, four)
		} else {
			break
		}
	}

	// Factor out odd squares: 3, 5, 7, 9...
	d := big.NewInt(3)
	d2 := new(big.Int).Mul(d, d)
	for d2.Cmp(val) <= 0 {
		rem.Mod(val, d2)
		if rem.Sign() == 0 {
			out.Mul(out, d)
			val.Div(val, d2)
		} else {
			d.Add(d, two)
			d2.Mul(d, d)
		}
	}

	return out, val
}

// intCbrt calculates floor(n^(1/3)) for a non-negative big.Int using binary search.
func intCbrt(n *big.Int) *big.Int {
	if n.Sign() == 0 {
		return big.NewInt(0)
	}
	one := big.NewInt(1)
	if n.Cmp(one) <= 0 {
		return big.NewInt(1)
	}

	bitLen := n.BitLen()
	highBit := (bitLen + 2) / 3
	high := new(big.Int).Lsh(one, uint(highBit))
	low := big.NewInt(1)

	res := big.NewInt(1)
	mid := new(big.Int)
	midCube := new(big.Int)

	for low.Cmp(high) <= 0 {
		mid.Add(low, high)
		mid.Rsh(mid, 1)

		midCube.Mul(mid, mid)
		midCube.Mul(midCube, mid)

		cmp := midCube.Cmp(n)
		if cmp == 0 {
			return mid
		} else if cmp < 0 {
			res.Set(mid)
			low.Add(mid, one)
		} else {
			high.Sub(mid, one)
		}
	}
	return res
}

// isRatPerfectCube checks if rational r is a perfect cube: r = (a/b)^3
func isRatPerfectCube(r *big.Rat) (*big.Rat, bool) {
	if r.Sign() == 0 {
		return big.NewRat(0, 1), true
	}
	num := r.Num()
	denom := r.Denom()

	rootNum := intCbrt(num)
	rootDenom := intCbrt(denom)

	testNum := new(big.Int).Mul(rootNum, rootNum)
	testNum.Mul(testNum, rootNum)

	testDenom := new(big.Int).Mul(rootDenom, rootDenom)
	testDenom.Mul(testDenom, rootDenom)

	if testNum.Cmp(num) == 0 && testDenom.Cmp(denom) == 0 {
		res := new(big.Rat).SetFrac(rootNum, rootDenom)
		return res, true
	}
	return nil, false
}

// extractCubeFree extracts perfect cube factors: n = out^3 * rem
func extractCubeFree(n *big.Int) (*big.Int, *big.Int) {
	val := new(big.Int).Set(n)
	out := big.NewInt(1)

	two := big.NewInt(2)
	eight := big.NewInt(8)
	rem := new(big.Int)

	// Factor out 2^3
	for {
		rem.Mod(val, eight)
		if rem.Sign() == 0 {
			out.Mul(out, two)
			val.Div(val, eight)
		} else {
			break
		}
	}

	// Factor out odd cubes: 3, 5, 7, 9...
	d := big.NewInt(3)
	d3 := new(big.Int).Mul(d, d)
	d3.Mul(d3, d)

	for d3.Cmp(val) <= 0 {
		rem.Mod(val, d3)
		if rem.Sign() == 0 {
			out.Mul(out, d)
			val.Div(val, d3)
		} else {
			d.Add(d, two)
			d3.Mul(d, d)
			d3.Mul(d3, d)
		}
	}

	return out, val
}

func extractRadicalSquare(n Node) (*big.Rat, bool) {
	switch v := n.(type) {
	case *RationalNode:
		sq := new(big.Rat).Mul(v.Val, v.Val)
		return sq, true
	case *SqrtNode:
		if r, ok := v.Radicand.(*RationalNode); ok && r.Val.Sign() >= 0 {
			return r.Val, true
		}
	case *UnaryOpNode:
		if v.Op == "-" {
			return extractRadicalSquare(v.Expr)
		}
	case *MulNode:
		if len(v.Factors) == 2 {
			r1, ok1 := v.Factors[0].(*RationalNode)
			s2, ok2 := v.Factors[1].(*SqrtNode)
			if ok1 && ok2 {
				if rRat, ok := s2.Radicand.(*RationalNode); ok && rRat.Val.Sign() >= 0 {
					k2 := new(big.Rat).Mul(r1.Val, r1.Val)
					return new(big.Rat).Mul(k2, rRat.Val), true
				}
			}
			s1, ok1 := v.Factors[0].(*SqrtNode)
			r2, ok2 := v.Factors[1].(*RationalNode)
			if ok1 && ok2 {
				if rRat, ok := s1.Radicand.(*RationalNode); ok && rRat.Val.Sign() >= 0 {
					k2 := new(big.Rat).Mul(r2.Val, r2.Val)
					return new(big.Rat).Mul(k2, rRat.Val), true
				}
			}
		}
	}
	return nil, false
}

func rationalizeBinomialDenominator(add *AddNode) (Node, bool) {
	if len(add.Terms) != 2 {
		return nil, false
	}
	t1 := add.Terms[0]
	t2 := add.Terms[1]

	sq1, ok1 := extractRadicalSquare(t1)
	sq2, ok2 := extractRadicalSquare(t2)
	if !ok1 || !ok2 {
		return nil, false
	}

	norm := new(big.Rat).Sub(sq1, sq2)
	if norm.Sign() == 0 {
		return nil, false
	}

	// Conjugate: t1 - t2 = t1 + (-t2)
	negT2, err := simplifyUnaryOp("-", t2)
	if err != nil {
		return nil, false
	}
	conj, err := simplifyAdd([]Node{t1, negT2})
	if err != nil {
		return nil, false
	}

	// 1 / D = (1 / norm) * conj
	invNorm := new(big.Rat).Inv(norm)
	invCoeff := &RationalNode{Val: invNorm}

	res, err := simplifyMul([]Node{invCoeff, conj})
	if err != nil {
		return nil, false
	}
	RecordTraceRewrite(RuleRationalize, &PowNode{Base: add, Exp: mustRational(-1, 1)}, res, fmt.Sprintf("分母に共役式 (%s) を乗算して有理化", Format(conj)))
	return res, true
}
