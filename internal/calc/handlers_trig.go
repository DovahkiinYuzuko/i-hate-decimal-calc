package calc

import (
	"math/big"
)

func init() {
	RegisterHandler("sin", handleSin)
	RegisterHandler("cos", handleCos)
	RegisterHandler("tan", handleTan)
	RegisterHandler("log", handleLog)
	RegisterHandler("exp", handleExp)
	RegisterHandler("ln", handleLn)
	RegisterHandler("abs", handleAbs)
	RegisterHandler("cbrt", handleCbrt)
}

func handleSin(args []Node, env *Env) (Node, error) {
	arg := args[0]
	if r, ok := matchPiMultiple(arg); ok {
		if val, ok := evalTrigPi("sin", r); ok {
			return val, nil
		}
	}
	if val, ok := evalTrigWithAssumptions("sin", arg, env); ok {
		return val, nil
	}
	if rat, ok := arg.(*RationalNode); ok && rat.Val.Sign() == 0 {
		return mustRational(0, 1), nil
	}
	return NewFunc("sin", args)
}

func handleCos(args []Node, env *Env) (Node, error) {
	arg := args[0]
	if r, ok := matchPiMultiple(arg); ok {
		if val, ok := evalTrigPi("cos", r); ok {
			return val, nil
		}
	}
	if val, ok := evalTrigWithAssumptions("cos", arg, env); ok {
		return val, nil
	}
	if rat, ok := arg.(*RationalNode); ok && rat.Val.Sign() == 0 {
		return mustRational(1, 1), nil
	}
	return NewFunc("cos", args)
}

func handleTan(args []Node, env *Env) (Node, error) {
	arg := args[0]
	if r, ok := matchPiMultiple(arg); ok {
		rMinusHalf := new(big.Rat).Sub(r, big.NewRat(1, 2))
		if rMinusHalf.IsInt() {
			return nil, NewZeroDivisionError("math error: tan(%s) is undefined (division by zero)", arg.String())
		}
		if val, ok := evalTrigPi("tan", r); ok {
			return val, nil
		}
	}
	if val, ok := evalTrigWithAssumptions("tan", arg, env); ok {
		return val, nil
	}
	if rat, ok := arg.(*RationalNode); ok && rat.Val.Sign() == 0 {
		return mustRational(0, 1), nil
	}
	return NewFunc("tan", args)
}

func init() {
	RegisterHandler("asin", func(args []Node, env *Env) (Node, error) {
		if val, ok, err := evalInverseTrig("asin", args[0]); err != nil {
			return nil, err
		} else if ok {
			return val, nil
		}
		return NewFunc("asin", args)
	})
	RegisterHandler("acos", func(args []Node, env *Env) (Node, error) {
		if val, ok, err := evalInverseTrig("acos", args[0]); err != nil {
			return nil, err
		} else if ok {
			return val, nil
		}
		return NewFunc("acos", args)
	})
	RegisterHandler("atan", func(args []Node, env *Env) (Node, error) {
		if val, ok, err := evalInverseTrig("atan", args[0]); err != nil {
			return nil, err
		} else if ok {
			return val, nil
		}
		return NewFunc("atan", args)
	})
}

func handleLog(args []Node, env *Env) (Node, error) {
	if len(args) == 1 {
		// log10(x)
		rat, ok := args[0].(*RationalNode)
		if !ok {
			return NewFunc("log", args)
		}
		if rat.Val.Sign() <= 0 {
			return nil, NewDomainError("errors.domain_log_arg", "log domain error: argument must be positive, got %s", rat.String())
		}
		if rat.Val.IsInt() {
			n := rat.Val.Num().Int64()
			power := 0
			for n > 1 && n%10 == 0 {
				n /= 10
				power++
			}
			if n == 1 {
				return mustRational(int64(power), 1), nil
			}
		}
		return NewFunc("log", args)
	}

	// log(base, x)
	baseRat, baseOk := args[0].(*RationalNode)
	xRat, xOk := args[1].(*RationalNode)
	if baseOk {
		if baseRat.Val.Sign() <= 0 {
			return nil, NewDomainError("errors.domain_log_base", "log base error: base must be positive, got %s", baseRat.String())
		}
		if baseRat.Val.Cmp(big.NewRat(1, 1)) == 0 {
			return nil, NewDomainError("errors.domain_log_base_one", "log base error: base cannot be 1")
		}
	}
	if xOk && xRat.Val.Sign() <= 0 {
		return nil, NewDomainError("errors.domain_log_arg", "log domain error: argument must be positive, got %s", xRat.String())
	}
	if baseOk && xOk && baseRat.Val.IsInt() && xRat.Val.IsInt() {
		b := baseRat.Val.Num().Int64()
		x := xRat.Val.Num().Int64()
		power := 0
		for x > 1 && x%b == 0 {
			x /= b
			power++
		}
		if x == 1 {
			return mustRational(int64(power), 1), nil
		}
	}
	return NewFunc("log", args)
}

func handleExp(args []Node, env *Env) (Node, error) {
	if isZero(args[0]) {
		return mustRational(1, 1), nil
	}
	return NewFunc("exp", args)
}

func handleLn(args []Node, env *Env) (Node, error) {
	arg := args[0]
	if c, ok := arg.(*ConstNode); ok && c.Name == "e" {
		return mustRational(1, 1), nil
	}
	if pow, ok := arg.(*PowNode); ok {
		if c, ok := pow.Base.(*ConstNode); ok && c.Name == "e" {
			return pow.Exp, nil // ln(e^x) = x
		}
	}
	rat, ok := arg.(*RationalNode)
	if !ok {
		return NewFunc("ln", args)
	}
	if rat.Val.Sign() <= 0 {
		return nil, NewDomainError("errors.domain_ln_arg", "ln domain error: argument must be positive, got %s", rat.String())
	}
	if rat.Val.Cmp(big.NewRat(1, 1)) == 0 {
		return mustRational(0, 1), nil
	}
	return NewFunc("ln", args)
}

func handleAbs(args []Node, env *Env) (Node, error) {
	arg := args[0]
	switch v := arg.(type) {
	case *RationalNode:
		newRat := new(big.Rat).Abs(v.Val)
		return NewRationalFromBigRat(newRat), nil
	case *ComplexNode:
		aSq := &PowNode{Base: v.Real, Exp: mustRational(2, 1)}
		bSq := &PowNode{Base: v.Imag, Exp: mustRational(2, 1)}
		sumNode := NewAdd([]Node{aSq, bSq})
		evaledSum, err := Eval(sumNode)
		if err != nil {
			return nil, err
		}
		return simplifySqrt(evaledSum)
	case *UnaryOpNode:
		if v.Op == "-" {
			return simplifyFunc("abs", []Node{v.Expr})
		}
	case *SqrtNode:
		return v, nil
	case *MulNode:
		if len(v.Factors) > 0 {
			if r, ok := v.Factors[0].(*RationalNode); ok && r.Val.Sign() < 0 {
				posR := NewRationalFromBigRat(new(big.Rat).Abs(r.Val))
				newFactors := make([]Node, len(v.Factors))
				copy(newFactors, v.Factors)
				newFactors[0] = posR
				return simplifyMul(newFactors)
			}
		}
	}
	if isNegative(arg) {
		neg, err := simplifyUnaryOp("-", arg)
		if err == nil {
			return simplifyFunc("abs", []Node{neg})
		}
	}
	return NewFunc("abs", args)
}

func handleCbrt(args []Node, env *Env) (Node, error) {
	arg := args[0]
	if u, ok := arg.(*UnaryOpNode); ok && u.Op == "-" {
		sub, err := simplifyFunc("cbrt", []Node{u.Expr})
		if err != nil {
			return nil, err
		}
		return simplifyUnaryOp("-", sub)
	}
	if rat, ok := arg.(*RationalNode); ok {
		if rat.Val.Sign() < 0 {
			posRat := new(big.Rat).Abs(rat.Val)
			sub, err := simplifyFunc("cbrt", []Node{NewRationalFromBigRat(posRat)})
			if err != nil {
				return nil, err
			}
			return simplifyUnaryOp("-", sub)
		}
		if rat.Val.Sign() == 0 {
			return mustRational(0, 1), nil
		}
		if root, ok := isRatPerfectCube(rat.Val); ok {
			return NewRationalFromBigRat(root), nil
		}
		numOut, numRem := extractCubeFree(rat.Val.Num())
		denomOut, denomRem := extractCubeFree(rat.Val.Denom())
		one := big.NewInt(1)
		if numOut.Cmp(one) > 0 || denomOut.Cmp(one) > 0 {
			coeff := new(big.Rat).SetFrac(numOut, denomOut)
			rem := new(big.Rat).SetFrac(numRem, denomRem)
			cbrtRem, _ := NewFunc("cbrt", []Node{NewRationalFromBigRat(rem)})
			return simplifyMul([]Node{NewRationalFromBigRat(coeff), cbrtRem})
		}
	}
	return NewFunc("cbrt", args)
}
