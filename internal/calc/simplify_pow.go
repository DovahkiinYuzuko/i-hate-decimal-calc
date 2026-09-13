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
		expInt := ratExp.Val.Num().Int64()
		if expInt < 0 {
			// Invert base
			if ratBase.Val.Sign() == 0 {
				return nil, NewZeroDivisionError("division by zero: reciprocal of 0")
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
