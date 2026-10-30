package calc

import (
	"fmt"
	"math/big"
)

// DualRat represents a dual number over big.Rat: a + b*eps where eps^2 = 0.
type DualRat struct {
	Real *big.Rat
	Dual *big.Rat
}

// NewDualConst creates a constant dual number with zero dual component: a + 0*eps.
func NewDualConst(val *big.Rat) DualRat {
	return DualRat{
		Real: new(big.Rat).Set(val),
		Dual: new(big.Rat),
	}
}

// NewDualVar creates an independent variable dual number with unit dual component: x + 1*eps.
func NewDualVar(val *big.Rat) DualRat {
	return DualRat{
		Real: new(big.Rat).Set(val),
		Dual: big.NewRat(1, 1),
	}
}

// DualAdd computes (a + b*eps) + (c + d*eps) = (a+c) + (b+d)*eps.
func DualAdd(x, y DualRat) DualRat {
	return DualRat{
		Real: new(big.Rat).Add(x.Real, y.Real),
		Dual: new(big.Rat).Add(x.Dual, y.Dual),
	}
}

// DualSub computes (a + b*eps) - (c + d*eps) = (a-c) + (b-d)*eps.
func DualSub(x, y DualRat) DualRat {
	return DualRat{
		Real: new(big.Rat).Sub(x.Real, y.Real),
		Dual: new(big.Rat).Sub(x.Dual, y.Dual),
	}
}

// DualMul computes (a + b*eps) * (c + d*eps) = (a*c) + (a*d + b*c)*eps.
func DualMul(x, y DualRat) DualRat {
	r := new(big.Rat).Mul(x.Real, y.Real)
	ad := new(big.Rat).Mul(x.Real, y.Dual)
	bc := new(big.Rat).Mul(x.Dual, y.Real)
	d := new(big.Rat).Add(ad, bc)
	return DualRat{Real: r, Dual: d}
}

// DualQuo computes (a + b*eps) / (c + d*eps) = (a/c) + ((b*c - a*d)/c^2)*eps.
func DualQuo(x, y DualRat) (DualRat, error) {
	if y.Real.Sign() == 0 {
		return DualRat{}, fmt.Errorf("division by zero in dual division")
	}

	r := new(big.Rat).Quo(x.Real, y.Real)

	bc := new(big.Rat).Mul(x.Dual, y.Real)
	ad := new(big.Rat).Mul(x.Real, y.Dual)
	num := new(big.Rat).Sub(bc, ad)
	c2 := new(big.Rat).Mul(y.Real, y.Real)
	d := new(big.Rat).Quo(num, c2)

	return DualRat{Real: r, Dual: d}, nil
}

// DualPowBigInt computes (a + b*eps)^n = a^n + n * a^(n-1) * b * eps using binary exponentiation.
// It accepts arbitrary precision *big.Int to prevent overflow and MinInt wrap-around.
func DualPowBigInt(x DualRat, n *big.Int) (DualRat, error) {
	if n.Sign() == 0 {
		return NewDualConst(big.NewRat(1, 1)), nil
	}

	if n.Sign() < 0 {
		absN := new(big.Int).Abs(n)
		pos, err := DualPowBigInt(x, absN)
		if err != nil {
			return DualRat{}, err
		}
		one := NewDualConst(big.NewRat(1, 1))
		return DualQuo(one, pos)
	}

	// For n > 0, compute a^n and a^(n-1) using binary exponentiation
	powReal := func(base *big.Rat, p *big.Int) (*big.Rat, error) {
		res := big.NewRat(1, 1)
		cur := new(big.Rat).Set(base)
		for i := 0; i < p.BitLen(); i++ {
			if p.Bit(i) == 1 {
				res.Mul(res, cur)
				if res.Num().BitLen()+res.Denom().BitLen() > 100000 {
					return nil, fmt.Errorf("bit size budget exceeded in dual power")
				}
			}
			if i+1 < p.BitLen() {
				cur.Mul(cur, cur)
				if cur.Num().BitLen()+cur.Denom().BitLen() > 100000 {
					return nil, fmt.Errorf("bit size budget exceeded in dual power")
				}
			}
		}
		return res, nil
	}

	rn, err := powReal(x.Real, n)
	if err != nil {
		return DualRat{}, err
	}

	nMinus1 := new(big.Int).Sub(n, big.NewInt(1))
	ran1, err := powReal(x.Real, nMinus1)
	if err != nil {
		return DualRat{}, err
	}

	nRat := new(big.Rat).SetInt(n)
	d := new(big.Rat).Mul(nRat, ran1)
	d.Mul(d, x.Dual)

	return DualRat{Real: rn, Dual: d}, nil
}

// DualPowInt is a convenience wrapper for DualPowBigInt accepting int.
func DualPowInt(x DualRat, n int) (DualRat, error) {
	return DualPowBigInt(x, big.NewInt(int64(n)))
}

// EvalNodeDual evaluates an AST node into a DualRat for forward automatic differentiation.
// It is completely independent of differentiate() and simplify_core().
func EvalNodeDual(n Node, varName string, point *big.Rat) (DualRat, error) {
	if n == nil {
		return DualRat{}, fmt.Errorf("nil node in dual evaluation")
	}

	switch v := n.(type) {
	case *RationalNode:
		return NewDualConst(v.Val), nil

	case *VarNode:
		if v.Name == varName {
			return NewDualVar(point), nil
		}
		return DualRat{}, fmt.Errorf("unsupported: multiple variables in dual evaluation (%s)", v.Name)

	case *UnaryOpNode:
		inner, err := EvalNodeDual(v.Expr, varName, point)
		if err != nil {
			return DualRat{}, err
		}
		if v.Op == "-" {
			return DualRat{Real: new(big.Rat).Neg(inner.Real), Dual: new(big.Rat).Neg(inner.Dual)}, nil
		}
		return DualRat{}, fmt.Errorf("unsupported unary operator %s in dual evaluation", v.Op)

	case *AddNode:
		res := NewDualConst(new(big.Rat))
		for _, term := range v.Terms {
			tVal, err := EvalNodeDual(term, varName, point)
			if err != nil {
				return DualRat{}, err
			}
			res = DualAdd(res, tVal)
		}
		return res, nil

	case *MulNode:
		res := NewDualConst(big.NewRat(1, 1))
		for _, factor := range v.Factors {
			fVal, err := EvalNodeDual(factor, varName, point)
			if err != nil {
				return DualRat{}, err
			}
			res = DualMul(res, fVal)
		}
		return res, nil

	case *PowNode:
		baseVal, err := EvalNodeDual(v.Base, varName, point)
		if err != nil {
			return DualRat{}, err
		}
		if rn, ok := v.Exp.(*RationalNode); ok && rn.Val.IsInt() {
			return DualPowBigInt(baseVal, rn.Val.Num())
		}
		return DualRat{}, fmt.Errorf("non-integer exponent in dual evaluation")

	default:
		return DualRat{}, fmt.Errorf("unsupported node type %T in dual evaluation", n)
	}
}
