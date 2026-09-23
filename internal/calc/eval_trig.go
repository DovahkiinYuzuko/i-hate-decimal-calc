package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"math/big"
)

// -------------------------------------------------------------------------
// Trigonometric & Inverse Trigonometric Special Angle Evaluations
// -------------------------------------------------------------------------

// matchPiMultiple checks if node is r * pi or pi / denom, returning r
func matchPiMultiple(n Node) (*big.Rat, bool) {
	if c, ok := n.(*ConstNode); ok && c.Name == "pi" {
		return big.NewRat(1, 1), true
	}
	if mul, ok := n.(*MulNode); ok {
		var rat *big.Rat
		var hasPi bool
		for _, f := range mul.Factors {
			if c, ok := f.(*ConstNode); ok && c.Name == "pi" {
				hasPi = true
			} else if r, ok := f.(*RationalNode); ok {
				if rat == nil {
					rat = new(big.Rat).Set(r.Val)
				} else {
					rat.Mul(rat, r.Val)
				}
			} else if pow, ok := f.(*PowNode); ok {
				// check for /denom -> Pow(denom, -1)
				if rBase, ok := pow.Base.(*RationalNode); ok {
					if rExp, ok := pow.Exp.(*RationalNode); ok && rExp.Val.Cmp(big.NewRat(-1, 1)) == 0 {
						inv := new(big.Rat).Inv(rBase.Val)
						if rat == nil {
							rat = inv
						} else {
							rat.Mul(rat, inv)
						}
					} else {
						return nil, false
					}
				} else {
					return nil, false
				}
			} else {
				return nil, false
			}
		}
		if hasPi {
			if rat == nil {
				return big.NewRat(1, 1), true
			}
			return rat, true
		}
	}
	return nil, false
}

// evalTrigWithAssumptions evaluates trigonometric expressions involving variables with assumptions.
func evalTrigWithAssumptions(fn string, arg Node, env *Env) (Node, bool) {
	if env == nil {
		return nil, false
	}

	// Match: coeff * var * pi or var * pi
	if mul, ok := arg.(*MulNode); ok {
		var hasPi bool
		var varName string
		coeff := big.NewRat(1, 1)

		for _, f := range mul.Factors {
			if c, okConst := f.(*ConstNode); okConst && c.Name == "pi" {
				hasPi = true
			} else if r, okRat := f.(*RationalNode); okRat {
				coeff.Mul(coeff, r.Val)
			} else if v, okVar := f.(*VarNode); okVar {
				if varName == "" {
					varName = v.Name
				} else {
					return nil, false
				}
			} else {
				return nil, false
			}
		}

		if hasPi && varName != "" && coeff.IsInt() {
			if env.IsInteger(varName) {
				k := coeff.Num().Int64()
				isEven := env.IsEven(varName) || k%2 == 0
				isOdd := env.IsOdd(varName) && k%2 != 0

				switch fn {
				case "sin":
					return mustRational(0, 1), true
				case "tan":
					return mustRational(0, 1), true
				case "cos":
					if isEven {
						return mustRational(1, 1), true
					}
					if isOdd {
						return mustRational(-1, 1), true
					}
				}
			}
		}
	}
	return nil, false
}

func evalTrigPi(fn string, r *big.Rat) (Node, bool) {
	// Normalize r to [0, 2): r = r mod 2
	two := big.NewRat(2, 1)
	rMod := new(big.Rat).Set(r)
	div := new(big.Rat).Quo(rMod, two)
	floorInt := new(big.Int).Div(div.Num(), div.Denom())
	rMod.Sub(rMod, new(big.Rat).Mul(two, new(big.Rat).SetInt(floorInt)))
	if rMod.Sign() < 0 {
		rMod.Add(rMod, two)
	}

	sqrt2over2 := func() Node {
		// sqrt(2) / 2
		half, _ := NewRational(1, 2)
		s2 := NewSqrt(&RationalNode{Val: big.NewRat(2, 1)})
		return NewMul([]Node{half, s2})
	}
	sqrt3over2 := func() Node {
		half, _ := NewRational(1, 2)
		s3 := NewSqrt(&RationalNode{Val: big.NewRat(3, 1)})
		return NewMul([]Node{half, s3})
	}
	sqrt3over3 := func() Node {
		third, _ := NewRational(1, 3)
		s3 := NewSqrt(&RationalNode{Val: big.NewRat(3, 1)})
		return NewMul([]Node{third, s3})
	}
	sqrt3 := func() Node {
		return NewSqrt(&RationalNode{Val: big.NewRat(3, 1)})
	}

	rKey := rMod.String()
	if rMod.IsInt() {
		rKey = rMod.Num().String()
	}

	// Match key angles: 0, 1/6, 1/4, 1/3, 1/2, 2/3, 3/4, 5/6, 1...
	switch fn {
	case "sin":
		switch rKey {
		case "0", "1":
			return mustRational(0, 1), true
		case "1/6", "5/6":
			return mustRational(1, 2), true
		case "1/4", "3/4":
			return sqrt2over2(), true
		case "1/3", "2/3":
			return sqrt3over2(), true
		case "1/2":
			return mustRational(1, 1), true
		case "7/6", "11/6":
			return mustRational(-1, 2), true
		case "5/4", "7/4":
			res, _ := simplifyUnaryOp("-", sqrt2over2())
			return res, true
		case "4/3", "5/3":
			res, _ := simplifyUnaryOp("-", sqrt3over2())
			return res, true
		case "3/2":
			return mustRational(-1, 1), true
		}

	case "cos":
		switch rKey {
		case "0":
			return mustRational(1, 1), true
		case "1/6", "11/6":
			return sqrt3over2(), true
		case "1/4", "7/4":
			return sqrt2over2(), true
		case "1/3", "5/3":
			return mustRational(1, 2), true
		case "1/2", "3/2":
			return mustRational(0, 1), true
		case "2/3", "4/3":
			return mustRational(-1, 2), true
		case "3/4", "5/4":
			res, _ := simplifyUnaryOp("-", sqrt2over2())
			return res, true
		case "5/6", "7/6":
			res, _ := simplifyUnaryOp("-", sqrt3over2())
			return res, true
		case "1":
			return mustRational(-1, 1), true
		}

	case "tan":
		switch rKey {
		case "0", "1":
			return mustRational(0, 1), true
		case "1/6", "7/6":
			return sqrt3over3(), true
		case "1/4", "5/4":
			return mustRational(1, 1), true
		case "1/3", "4/3":
			return sqrt3(), true
		case "2/3", "5/3":
			res, _ := simplifyUnaryOp("-", sqrt3())
			return res, true
		case "3/4", "7/4":
			return mustRational(-1, 1), true
		case "5/6", "11/6":
			res, _ := simplifyUnaryOp("-", sqrt3over3())
			return res, true
		}
	}

	return nil, false
}

func piMultiple(num, denom int64) Node {
	if num == 0 {
		return mustRational(0, 1)
	}
	r := NewRationalFromBigRat(big.NewRat(num, denom))
	pi := &ConstNode{Name: "pi"}
	if r.Val.Cmp(big.NewRat(1, 1)) == 0 {
		return pi
	}
	return &MulNode{Factors: []Node{r, pi}}
}

func classifyTrigVal(n Node) (string, bool) {
	neg := false
	cur := n
	if isNegative(cur) {
		neg = true
		if negated, err := simplifyUnaryOp("-", cur); err == nil {
			cur = negated
		}
	}

	if rat, ok := cur.(*RationalNode); ok {
		if rat.Val.Sign() == 0 {
			return "0", false
		}
		if rat.Val.Cmp(big.NewRat(1, 2)) == 0 {
			return "1/2", neg
		}
		if rat.Val.Cmp(big.NewRat(1, 1)) == 0 {
			return "1", neg
		}
		return "", neg
	}

	if s, ok := cur.(*SqrtNode); ok {
		if r, ok := s.Radicand.(*RationalNode); ok && r.Val.Cmp(big.NewRat(3, 1)) == 0 {
			return "sqrt(3)", neg
		}
	}

	if mul, ok := cur.(*MulNode); ok {
		var rat *big.Rat
		var sqrtN int64
		for _, f := range mul.Factors {
			if r, ok := f.(*RationalNode); ok {
				if rat == nil {
					rat = new(big.Rat).Set(r.Val)
				} else {
					rat.Mul(rat, r.Val)
				}
			} else if s, ok := f.(*SqrtNode); ok {
				if r, ok := s.Radicand.(*RationalNode); ok && r.Val.IsInt() {
					sqrtN = r.Val.Num().Int64()
				}
			} else if pow, ok := f.(*PowNode); ok {
				if rBase, ok := pow.Base.(*RationalNode); ok {
					if rExp, ok := pow.Exp.(*RationalNode); ok && rExp.Val.Cmp(big.NewRat(-1, 1)) == 0 {
						inv := new(big.Rat).Inv(rBase.Val)
						if rat == nil {
							rat = inv
						} else {
							rat.Mul(rat, inv)
						}
					}
				}
			}
		}
		if rat != nil && sqrtN > 0 {
			if sqrtN == 2 && rat.Cmp(big.NewRat(1, 2)) == 0 {
				return "sqrt(2)/2", neg
			}
			if sqrtN == 3 && rat.Cmp(big.NewRat(1, 2)) == 0 {
				return "sqrt(3)/2", neg
			}
			if sqrtN == 3 && rat.Cmp(big.NewRat(1, 3)) == 0 {
				return "sqrt(3)/3", neg
			}
		}
	}

	return "", neg
}

func evalInverseTrig(fn string, arg Node) (Node, bool, error) {
	// Domain check for rational arguments
	if rat, ok := arg.(*RationalNode); ok {
		if fn == "asin" || fn == "acos" {
			one := big.NewRat(1, 1)
			negOne := big.NewRat(-1, 1)
			if rat.Val.Cmp(one) > 0 || rat.Val.Cmp(negOne) < 0 {
				return nil, false, fmt.Errorf("%s", i18n.T("errors.err_argument_must_be_in_1", fn, rat.String()))
			}
		}
	}

	vType, isNeg := classifyTrigVal(arg)
	if vType == "" {
		return nil, false, nil
	}

	switch fn {
	case "asin":
		var posRes Node
		switch vType {
		case "0":
			return mustRational(0, 1), true, nil
		case "1/2":
			posRes = piMultiple(1, 6)
		case "sqrt(2)/2":
			posRes = piMultiple(1, 4)
		case "sqrt(3)/2":
			posRes = piMultiple(1, 3)
		case "1":
			posRes = piMultiple(1, 2)
		default:
			return nil, false, nil
		}
		if isNeg {
			res, err := simplifyUnaryOp("-", posRes)
			return res, true, err
		}
		return posRes, true, nil

	case "acos":
		switch vType {
		case "0":
			return piMultiple(1, 2), true, nil
		case "1/2":
			if isNeg {
				return piMultiple(2, 3), true, nil
			}
			return piMultiple(1, 3), true, nil
		case "sqrt(2)/2":
			if isNeg {
				return piMultiple(3, 4), true, nil
			}
			return piMultiple(1, 4), true, nil
		case "sqrt(3)/2":
			if isNeg {
				return piMultiple(5, 6), true, nil
			}
			return piMultiple(1, 6), true, nil
		case "1":
			if isNeg {
				return piMultiple(1, 1), true, nil
			}
			return mustRational(0, 1), true, nil
		default:
			return nil, false, nil
		}

	case "atan":
		var posRes Node
		switch vType {
		case "0":
			return mustRational(0, 1), true, nil
		case "sqrt(3)/3":
			posRes = piMultiple(1, 6)
		case "1":
			posRes = piMultiple(1, 4)
		case "sqrt(3)":
			posRes = piMultiple(1, 3)
		default:
			return nil, false, nil
		}
		if isNeg {
			res, err := simplifyUnaryOp("-", posRes)
			return res, true, err
		}
		return posRes, true, nil

	default:
		return nil, false, nil
	}
}
