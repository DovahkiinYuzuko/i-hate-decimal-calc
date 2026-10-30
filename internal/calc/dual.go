package calc

import (
	"fmt"
	"math/big"
)

// DualRat represents a dual number over the rational field: Real + Dual * epsilon (where epsilon^2 = 0).
type DualRat struct {
	Real *big.Rat
	Dual *big.Rat
}

// NewDualRat creates a new DualRat number with defensive copies.
func NewDualRat(real, dual *big.Rat) DualRat {
	r := new(big.Rat)
	if real != nil {
		r.Set(real)
	}
	d := new(big.Rat)
	if dual != nil {
		d.Set(dual)
	}
	return DualRat{Real: r, Dual: d}
}

// NewDualConst creates a constant dual number (derivative = 0).
func NewDualConst(val *big.Rat) DualRat {
	return NewDualRat(val, new(big.Rat))
}

// NewDualVar creates a variable dual number (derivative = 1).
func NewDualVar(val *big.Rat) DualRat {
	return NewDualRat(val, big.NewRat(1, 1))
}

// DualAdd computes (a + b*eps) + (c + d*eps) = (a + c) + (b + d)*eps.
func DualAdd(x, y DualRat) DualRat {
	r := new(big.Rat).Add(x.Real, y.Real)
	d := new(big.Rat).Add(x.Dual, y.Dual)
	return DualRat{Real: r, Dual: d}
}

// DualSub computes (a + b*eps) - (c + d*eps) = (a - c) + (b - d)*eps.
func DualSub(x, y DualRat) DualRat {
	r := new(big.Rat).Sub(x.Real, y.Real)
	d := new(big.Rat).Sub(x.Dual, y.Dual)
	return DualRat{Real: r, Dual: d}
}

// DualMul computes (a + b*eps) * (c + d*eps) = a*c + (a*d + b*c)*eps.
func DualMul(x, y DualRat) DualRat {
	r := new(big.Rat).Mul(x.Real, y.Real)
	t1 := new(big.Rat).Mul(x.Real, y.Dual)
	t2 := new(big.Rat).Mul(x.Dual, y.Real)
	d := new(big.Rat).Add(t1, t2)
	return DualRat{Real: r, Dual: d}
}

// DualQuo computes (a + b*eps) / (c + d*eps) = a/c + (b*c - a*d)/(c^2)*eps.
func DualQuo(x, y DualRat) (DualRat, error) {
	if y.Real.Sign() == 0 {
		return DualRat{}, fmt.Errorf("division by zero in dual number arithmetic")
	}
	r := new(big.Rat).Quo(x.Real, y.Real)

	bc := new(big.Rat).Mul(x.Dual, y.Real)
	ad := new(big.Rat).Mul(x.Real, y.Dual)
	num := new(big.Rat).Sub(bc, ad)
	c2 := new(big.Rat).Mul(y.Real, y.Real)
	d := new(big.Rat).Quo(num, c2)

	return DualRat{Real: r, Dual: d}, nil
}

// DualPowInt computes (a + b*eps)^n = a^n + n * a^(n-1) * b * eps.
func DualPowInt(x DualRat, n int) (DualRat, error) {
	if n == 0 {
		return NewDualConst(big.NewRat(1, 1)), nil
	}
	if n < 0 {
		pos, err := DualPowInt(x, -n)
		if err != nil {
			return DualRat{}, err
		}
		one := NewDualConst(big.NewRat(1, 1))
		return DualQuo(one, pos)
	}

	r := new(big.Rat).Set(x.Real)
	for i := 1; i < n; i++ {
		r.Mul(r, x.Real)
	}

	if n == 1 {
		return DualRat{Real: r, Dual: new(big.Rat).Set(x.Dual)}, nil
	}

	an1 := new(big.Rat).Set(x.Real)
	for i := 1; i < n-1; i++ {
		an1.Mul(an1, x.Real)
	}
	nRat := big.NewRat(int64(n), 1)
	d := new(big.Rat).Mul(nRat, an1)
	d.Mul(d, x.Dual)

	return DualRat{Real: r, Dual: d}, nil
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
		// Any variable other than varName is treated as constant at 1 or given value
		return NewDualConst(big.NewRat(1, 1)), nil

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
		// If exponent is an integer constant
		if rn, ok := v.Exp.(*RationalNode); ok && rn.Val.IsInt() {
			nInt := int(rn.Val.Num().Int64())
			return DualPowInt(baseVal, nInt)
		}
		return DualRat{}, fmt.Errorf("non-integer exponent in dual evaluation")

	default:
		return DualRat{}, fmt.Errorf("unsupported node type %T in dual evaluation", n)
	}
}
