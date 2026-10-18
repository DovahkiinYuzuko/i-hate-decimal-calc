package calc

import (
	"math/big"
)

// -------------------------------------------------------------------------
// Power / Exponentiation
// -------------------------------------------------------------------------

func simplifyPow(base, exp Node) (Node, error) {
	ratExp, expIsRat := exp.(*RationalNode)
	ratBase, baseIsRat := base.(*RationalNode)

	// 0^exp
	if baseIsRat && ratBase.Val.Sign() == 0 {
		if expIsRat && ratExp.Val.Sign() < 0 {
			return nil, NewZeroDivisionError("division by zero: 0^(negative) is undefined")
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
		expNum := ratExp.Val.Num()

		// 1^n = 1
		if ratBase.Val.Cmp(one) == 0 {
			return mustRational(1, 1), nil
		}

		// (-1)^n = 1 (even) or -1 (odd)
		minusOne := big.NewRat(-1, 1)
		if ratBase.Val.Cmp(minusOne) == 0 {
			if expNum.Bit(0) == 0 {
				return mustRational(1, 1), nil
			}
			return mustRational(-1, 1), nil
		}

		if expNum.Sign() < 0 {
			// Invert base
			if ratBase.Val.Sign() == 0 {
				return nil, NewZeroDivisionError("division by zero: reciprocal of 0")
			}
			inv := new(big.Rat).Inv(ratBase.Val)
			posExp := new(big.Int).Neg(expNum)
			return simplifyPow(&RationalNode{Val: inv}, &RationalNode{Val: new(big.Rat).SetInt(posExp)})
		}

		numPow, err := PowBigIntWithSwap(ratBase.Val.Num(), expNum)
		if err != nil {
			return nil, err
		}
		denomPow, err := PowBigIntWithSwap(ratBase.Val.Denom(), expNum)
		if err != nil {
			return nil, err
		}
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
		expNum := ratExp.Val.Num()
		innerPow, err := simplifyPow(uop.Expr, exp)
		if err != nil {
			return nil, err
		}
		if expNum.Bit(0) == 0 {
			return innerPow, nil
		}
		return simplifyUnaryOp("-", innerPow)
	}

	// Complex^Integer
	if c, ok := base.(*ComplexNode); ok && expIsRat && ratExp.Val.IsInt() {
		expNum := ratExp.Val.Num()
		if !expNum.IsInt64() {
			// 超巨大複素数累乗は未評価記号ノードを返す
			return NewPow(base, exp)
		}
		expInt := expNum.Int64()
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
				return nil, NewZeroDivisionError("division by zero: 1/(0+0i)")
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
			// Exponentiation by squaring: O(log n)
			res := Node(mustRational(1, 1))
			curr := Node(c)
			e := expInt
			for e > 0 {
				if e%2 == 1 {
					var err error
					res, err = simplifyMul([]Node{res, curr})
					if err != nil {
						return nil, err
					}
				}
				if e > 1 {
					var err error
					curr, err = simplifyMul([]Node{curr, curr})
					if err != nil {
						return nil, err
					}
				}
				e /= 2
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
