package calc

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"sort"
	"strings"
)

// mustRational creates a RationalNode directly for guaranteed non-zero denominators.
func mustRational(num, denom int64) *RationalNode {
	return &RationalNode{Val: big.NewRat(num, denom)}
}

// Eval evaluates and simplifies an AST node into its canonical exact form using an empty environment.
func Eval(n Node) (Node, error) {
	return EvalWithEnv(n, NewEnv())
}

// EvalWithEnv evaluates and simplifies an AST node in the context of the given environment.
func EvalWithEnv(n Node, env *Env) (Node, error) {
	if n == nil {
		return nil, fmt.Errorf("cannot evaluate nil node")
	}

	switch v := n.(type) {
	case *RationalNode:
		return v, nil

	case *ConstNode:
		if v.Name == "deg" {
			return &MulNode{
				Factors: []Node{
					NewRationalFromBigRat(big.NewRat(1, 180)),
					&ConstNode{Name: "pi"},
				},
			}, nil
		}
		return v, nil

	case *VarNode:
		if env != nil {
			if bound, ok := env.Get(v.Name); ok {
				return EvalWithEnv(bound, env)
			}
		}
		return v, nil

	case *ComplexNode:
		r, err := EvalWithEnv(v.Real, env)
		if err != nil {
			return nil, err
		}
		im, err := EvalWithEnv(v.Imag, env)
		if err != nil {
			return nil, err
		}
		return NewComplex(r, im), nil

	case *SqrtNode:
		rad, err := EvalWithEnv(v.Radicand, env)
		if err != nil {
			return nil, err
		}
		return simplifySqrt(rad)

	case *FuncNode:
		evaledArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			ea, err := EvalWithEnv(a, env)
			if err != nil {
				return nil, err
			}
			evaledArgs[i] = ea
		}
		return simplifyFunc(v.Name, evaledArgs)

	case *UnaryOpNode:
		expr, err := EvalWithEnv(v.Expr, env)
		if err != nil {
			return nil, err
		}
		return simplifyUnaryOp(v.Op, expr)

	case *PowNode:
		base, err := EvalWithEnv(v.Base, env)
		if err != nil {
			return nil, err
		}
		exp, err := EvalWithEnv(v.Exp, env)
		if err != nil {
			return nil, err
		}
		return simplifyPow(base, exp)

	case *AddNode:
		evaledTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			et, err := EvalWithEnv(t, env)
			if err != nil {
				return nil, err
			}
			evaledTerms[i] = et
		}
		return simplifyAdd(evaledTerms)

	case *MulNode:
		evaledFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			ef, err := EvalWithEnv(f, env)
			if err != nil {
				return nil, err
			}
			evaledFactors[i] = ef
		}
		return simplifyMul(evaledFactors)

	default:
		return nil, fmt.Errorf("unknown node type for evaluation: %T", n)
	}
}

// EvalString parses and evaluates a math expression string using an empty environment.
func EvalString(input string) (Node, error) {
	return EvalStringWithEnv(input, NewEnv())
}

// EvalStringWithEnv parses and evaluates a math expression string using the given environment.
func EvalStringWithEnv(input string, env *Env) (Node, error) {
	node, err := Parse(input)
	if err != nil {
		return nil, err
	}
	return EvalWithEnv(node, env)
}

// ApplyDegreeMode transforms trigonometric and inverse trigonometric function calls in the AST
// to work with degrees rather than radians.
// sin(x), cos(x), tan(x) -> sin(x * deg), cos(x * deg), tan(x * deg)
// asin(x), acos(x), atan(x) -> asin(x) / deg, acos(x) / deg, atan(x) / deg
func ApplyDegreeMode(n Node) Node {
	if n == nil {
		return nil
	}

	degNode := &ConstNode{Name: "deg"}

	switch v := n.(type) {
	case *FuncNode:
		newArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			newArgs[i] = ApplyDegreeMode(a)
		}

		switch v.Name {
		case "sin", "cos", "tan":
			if len(newArgs) == 1 {
				wrappedArg := &MulNode{
					Factors: []Node{newArgs[0], degNode},
				}
				return &FuncNode{Name: v.Name, Args: []Node{wrappedArg}}
			}
			return &FuncNode{Name: v.Name, Args: newArgs}

		case "asin", "acos", "atan":
			fn := &FuncNode{Name: v.Name, Args: newArgs}
			degInv := &PowNode{
				Base: degNode,
				Exp:  NewRationalFromBigRat(big.NewRat(-1, 1)),
			}
			return &MulNode{
				Factors: []Node{fn, degInv},
			}

		default:
			return &FuncNode{Name: v.Name, Args: newArgs}
		}

	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = ApplyDegreeMode(t)
		}
		return &AddNode{Terms: newTerms}

	case *MulNode:
		newFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			newFactors[i] = ApplyDegreeMode(f)
		}
		return &MulNode{Factors: newFactors}

	case *PowNode:
		return &PowNode{
			Base: ApplyDegreeMode(v.Base),
			Exp:  ApplyDegreeMode(v.Exp),
		}

	case *UnaryOpNode:
		return &UnaryOpNode{
			Op:   v.Op,
			Expr: ApplyDegreeMode(v.Expr),
		}

	case *ComplexNode:
		return &ComplexNode{
			Real: ApplyDegreeMode(v.Real),
			Imag: ApplyDegreeMode(v.Imag),
		}

	case *SqrtNode:
		return &SqrtNode{
			Radicand: ApplyDegreeMode(v.Radicand),
		}

	default:
		return n
	}
}

// ApplyDegreeModeToStatement applies degree mode to expressions within statements or plain expression nodes.
func ApplyDegreeModeToStatement(stmt interface{}) interface{} {
	switch s := stmt.(type) {
	case *AssignStmt:
		return &AssignStmt{
			Name:  s.Name,
			Value: ApplyDegreeMode(s.Value),
		}
	case Node:
		return ApplyDegreeMode(s)
	default:
		return stmt
	}
}

// -------------------------------------------------------------------------
// Sqrt Simplification (Square-free reduction & Complex promotion)
// -------------------------------------------------------------------------

func simplifySqrt(radicand Node) (Node, error) {
	rat, ok := radicand.(*RationalNode)
	if !ok {
		// Non-rational radicand: check if it can be denested (Borodin 1985)
		if add, isAdd := radicand.(*AddNode); isAdd {
			if denested, ok := denestSqrtBinomial(add); ok {
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

// -------------------------------------------------------------------------
// Unary Operators (- and !)
// -------------------------------------------------------------------------

func simplifyUnaryOp(op string, expr Node) (Node, error) {
	switch op {
	case "-":
		switch v := expr.(type) {
		case *RationalNode:
			neg := new(big.Rat).Neg(v.Val)
			return &RationalNode{Val: neg}, nil
		case *ComplexNode:
			negReal, err := simplifyUnaryOp("-", v.Real)
			if err != nil {
				return nil, err
			}
			negImag, err := simplifyUnaryOp("-", v.Imag)
			if err != nil {
				return nil, err
			}
			return NewComplex(negReal, negImag), nil
		case *AddNode:
			negTerms := make([]Node, len(v.Terms))
			for i, t := range v.Terms {
				nt, err := simplifyUnaryOp("-", t)
				if err != nil {
					return nil, err
				}
				negTerms[i] = nt
			}
			return simplifyAdd(negTerms)
		default:
			negOne, _ := NewRational(-1, 1)
			return simplifyMul([]Node{negOne, expr})
		}

	case "!":
		rat, ok := expr.(*RationalNode)
		if !ok || !rat.Val.IsInt() || rat.Val.Sign() < 0 {
			return nil, fmt.Errorf("factorial domain error: factorial requires non-negative integer, got %s", expr.String())
		}
		n := rat.Val.Num().Int64()
		res := big.NewInt(1)
		for i := int64(2); i <= n; i++ {
			res.Mul(res, big.NewInt(i))
		}
		return &RationalNode{Val: new(big.Rat).SetInt(res)}, nil

	default:
		return nil, fmt.Errorf("unknown unary operator: %s", op)
	}
}

// -------------------------------------------------------------------------
// Power / Exponentiation
// -------------------------------------------------------------------------

func simplifyPow(base, exp Node) (Node, error) {
	ratExp, expIsRat := exp.(*RationalNode)
	ratBase, baseIsRat := base.(*RationalNode)

	// 0^exp
	if baseIsRat && ratBase.Val.Sign() == 0 {
		if expIsRat && ratExp.Val.Sign() < 0 {
			return nil, fmt.Errorf("division by zero: 0^(negative) is undefined")
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
				return nil, fmt.Errorf("division by zero: reciprocal of 0")
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
				return nil, fmt.Errorf("division by zero: 1/(0+0i)")
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
	return res, true
}

// -------------------------------------------------------------------------
// Functions (sin, cos, tan, log, ln)
// -------------------------------------------------------------------------

func simplifyFunc(name string, args []Node) (Node, error) {
	switch name {
	case "sin":
		arg := args[0]
		if r, ok := matchPiMultiple(arg); ok {
			// sin(pi * r)
			if val, ok := evalTrigPi("sin", r); ok {
				return val, nil
			}
		}
		if rat, ok := arg.(*RationalNode); ok && rat.Val.Sign() == 0 {
			return mustRational(0, 1), nil
		}
		return NewFunc(name, args)

	case "cos":
		arg := args[0]
		if r, ok := matchPiMultiple(arg); ok {
			if val, ok := evalTrigPi("cos", r); ok {
				return val, nil
			}
		}
		if rat, ok := arg.(*RationalNode); ok && rat.Val.Sign() == 0 {
			return mustRational(1, 1), nil
		}
		return NewFunc(name, args)

	case "tan":
		arg := args[0]
		if r, ok := matchPiMultiple(arg); ok {
			// Check for undefined values: pi/2 + k*pi (where r = 1/2, 3/2, etc.)
			// r - 1/2 is integer?
			rMinusHalf := new(big.Rat).Sub(r, big.NewRat(1, 2))
			if rMinusHalf.IsInt() {
				return nil, fmt.Errorf("math error: tan(%s) is undefined (division by zero)", arg.String())
			}
			if val, ok := evalTrigPi("tan", r); ok {
				return val, nil
			}
		}
		if rat, ok := arg.(*RationalNode); ok && rat.Val.Sign() == 0 {
			return mustRational(0, 1), nil
		}
		return NewFunc(name, args)

	case "asin", "acos", "atan":
		arg := args[0]
		if val, ok, err := evalInverseTrig(name, arg); err != nil {
			return nil, err
		} else if ok {
			return val, nil
		}
		return NewFunc(name, args)

	case "log":
		if len(args) == 1 {
			// log10(x)
			rat, ok := args[0].(*RationalNode)
			if !ok {
				return NewFunc(name, args)
			}
			if rat.Val.Sign() <= 0 {
				return nil, fmt.Errorf("log domain error: argument must be positive, got %s", rat.String())
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
			return NewFunc(name, args)
		}
		// log(base, x)
		baseRat, baseOk := args[0].(*RationalNode)
		xRat, xOk := args[1].(*RationalNode)
		if baseOk {
			if baseRat.Val.Sign() <= 0 {
				return nil, fmt.Errorf("log base error: base must be positive, got %s", baseRat.String())
			}
			if baseRat.Val.Cmp(big.NewRat(1, 1)) == 0 {
				return nil, fmt.Errorf("log base error: base cannot be 1")
			}
		}
		if xOk && xRat.Val.Sign() <= 0 {
			return nil, fmt.Errorf("log domain error: argument must be positive, got %s", xRat.String())
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
		return NewFunc(name, args)

	case "ln":
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
			return NewFunc(name, args)
		}
		if rat.Val.Sign() <= 0 {
			return nil, fmt.Errorf("ln domain error: argument must be positive, got %s", rat.String())
		}
		if rat.Val.Cmp(big.NewRat(1, 1)) == 0 {
			return mustRational(0, 1), nil
		}
		return NewFunc(name, args)

	case "abs":
		arg := args[0]
		switch v := arg.(type) {
		case *RationalNode:
			newRat := new(big.Rat).Abs(v.Val)
			return NewRationalFromBigRat(newRat), nil
		case *ComplexNode:
			// |a + bi| = sqrt(a^2 + b^2)
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
		return NewFunc(name, args)

	case "cbrt":
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
		return NewFunc(name, args)

	case "gcd":
		aRat, aOk := args[0].(*RationalNode)
		bRat, bOk := args[1].(*RationalNode)
		if aOk && bOk {
			if !aRat.Val.IsInt() || !bRat.Val.IsInt() {
				return nil, fmt.Errorf("gcd domain error: arguments must be integers, got %s and %s", aRat.String(), bRat.String())
			}
			g := new(big.Int).GCD(nil, nil, aRat.Val.Num(), bRat.Val.Num())
			return NewRationalFromBigRat(new(big.Rat).SetInt(g)), nil
		}
		return NewFunc(name, args)

	case "lcm":
		aRat, aOk := args[0].(*RationalNode)
		bRat, bOk := args[1].(*RationalNode)
		if aOk && bOk {
			if !aRat.Val.IsInt() || !bRat.Val.IsInt() {
				return nil, fmt.Errorf("lcm domain error: arguments must be integers, got %s and %s", aRat.String(), bRat.String())
			}
			if aRat.Val.Sign() == 0 || bRat.Val.Sign() == 0 {
				return mustRational(0, 1), nil
			}
			aInt := aRat.Val.Num()
			bInt := bRat.Val.Num()
			g := new(big.Int).GCD(nil, nil, aInt, bInt)
			div := new(big.Int).Div(aInt, g)
			l := new(big.Int).Mul(div, bInt)
			l.Abs(l)
			return NewRationalFromBigRat(new(big.Rat).SetInt(l)), nil
		}
		return NewFunc(name, args)

	case "mod":
		aRat, aOk := args[0].(*RationalNode)
		bRat, bOk := args[1].(*RationalNode)
		if aOk && bOk {
			if !aRat.Val.IsInt() || !bRat.Val.IsInt() {
				return nil, fmt.Errorf("mod domain error: arguments must be integers, got %s and %s", aRat.String(), bRat.String())
			}
			if bRat.Val.Sign() == 0 {
				return nil, fmt.Errorf("division by zero in mod")
			}
			m := new(big.Int).Mod(aRat.Val.Num(), bRat.Val.Num())
			return NewRationalFromBigRat(new(big.Rat).SetInt(m)), nil
		}
		return NewFunc(name, args)

	case "perm":
		nRat, nOk := args[0].(*RationalNode)
		rRat, rOk := args[1].(*RationalNode)
		if nOk && rOk {
			if !nRat.Val.IsInt() || !rRat.Val.IsInt() {
				return nil, fmt.Errorf("perm domain error: arguments must be integers, got %s and %s", nRat.String(), rRat.String())
			}
			if nRat.Val.Sign() < 0 || rRat.Val.Sign() < 0 {
				return nil, fmt.Errorf("perm domain error: arguments must be non-negative, got %s and %s", nRat.String(), rRat.String())
			}
			nVal := nRat.Val.Num()
			rVal := rRat.Val.Num()
			if rVal.Cmp(nVal) > 0 {
				return mustRational(0, 1), nil
			}
			res := big.NewInt(1)
			curr := new(big.Int).Set(nVal)
			count := rVal.Int64()
			one := big.NewInt(1)
			for i := int64(0); i < count; i++ {
				res.Mul(res, curr)
				curr.Sub(curr, one)
			}
			return NewRationalFromBigRat(new(big.Rat).SetInt(res)), nil
		}
		return NewFunc(name, args)

	case "comb":
		nRat, nOk := args[0].(*RationalNode)
		rRat, rOk := args[1].(*RationalNode)
		if nOk && rOk {
			if !nRat.Val.IsInt() || !rRat.Val.IsInt() {
				return nil, fmt.Errorf("comb domain error: arguments must be integers, got %s and %s", nRat.String(), rRat.String())
			}
			if nRat.Val.Sign() < 0 || rRat.Val.Sign() < 0 {
				return nil, fmt.Errorf("comb domain error: arguments must be non-negative, got %s and %s", nRat.String(), rRat.String())
			}
			nVal := nRat.Val.Num()
			rVal := rRat.Val.Num()
			if rVal.Cmp(nVal) > 0 {
				return mustRational(0, 1), nil
			}
			res := new(big.Int).Binomial(nVal.Int64(), rVal.Int64())
			return NewRationalFromBigRat(new(big.Rat).SetInt(res)), nil
		}
		return NewFunc(name, args)

	case "rand":
		if len(args) == 1 {
			maxRat, ok := args[0].(*RationalNode)
			if !ok || !maxRat.Val.IsInt() {
				return nil, fmt.Errorf("rand domain error: max must be an integer, got %s", args[0].String())
			}
			maxVal := maxRat.Val.Num().Int64()
			if maxVal < 0 {
				return nil, fmt.Errorf("rand domain error: max must be non-negative, got %d", maxVal)
			}
			if maxVal == 0 {
				return mustRational(0, 1), nil
			}
			val := rand.Int64N(maxVal + 1)
			return mustRational(val, 1), nil
		} else if len(args) == 2 {
			minRat, minOk := args[0].(*RationalNode)
			maxRat, maxOk := args[1].(*RationalNode)
			if !minOk || !maxOk || !minRat.Val.IsInt() || !maxRat.Val.IsInt() {
				return nil, fmt.Errorf("rand domain error: min and max must be integers, got %s and %s", args[0].String(), args[1].String())
			}
			minVal := minRat.Val.Num().Int64()
			maxVal := maxRat.Val.Num().Int64()
			if minVal > maxVal {
				return nil, fmt.Errorf("rand domain error: min cannot be greater than max, got %d > %d", minVal, maxVal)
			}
			delta := maxVal - minVal + 1
			val := minVal + rand.Int64N(delta)
			return mustRational(val, 1), nil
		} else if len(args) == 3 {
			seedRat, sOk := args[0].(*RationalNode)
			minRat, minOk := args[1].(*RationalNode)
			maxRat, maxOk := args[2].(*RationalNode)
			if !sOk || !minOk || !maxOk || !seedRat.Val.IsInt() || !minRat.Val.IsInt() || !maxRat.Val.IsInt() {
				return nil, fmt.Errorf("rand domain error: seed, min, and max must be integers")
			}
			seedVal := seedRat.Val.Num().Int64()
			minVal := minRat.Val.Num().Int64()
			maxVal := maxRat.Val.Num().Int64()
			if minVal > maxVal {
				return nil, fmt.Errorf("rand domain error: min cannot be greater than max, got %d > %d", minVal, maxVal)
			}
			delta := maxVal - minVal + 1
			rng := rand.New(rand.NewPCG(uint64(seedVal), 1))
			val := minVal + rng.Int64N(delta)
			return mustRational(val, 1), nil
		}
		return NewFunc(name, args)

	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

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
					}
				}
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

	// Match key angles: 0, 1/6, 1/4, 1/3, 1/2, 2/3, 3/4, 5/6, 1...
	switch fn {
	case "sin":
		switch rMod.String() {
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
		switch rMod.String() {
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
		switch rMod.String() {
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
				return nil, false, fmt.Errorf("%s domain error: argument must be in [-1, 1], got %s", fn, rat.String())
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

// -------------------------------------------------------------------------
// Multiplication Simplification & Distributive Law Expansion
// -------------------------------------------------------------------------

func simplifyMul(factors []Node) (Node, error) {
	// 1. Flatten nested MulNodes
	var flatFactors []Node
	for _, f := range factors {
		if mul, ok := f.(*MulNode); ok {
			flatFactors = append(flatFactors, mul.Factors...)
		} else {
			flatFactors = append(flatFactors, f)
		}
	}

	// 2. Check for Distributive Law: if any factor is an AddNode, expand!
	// (A + B) * (C + D) -> A*C + A*D + B*C + B*D
	for i, f := range flatFactors {
		if add, ok := f.(*AddNode); ok {
			// Distribute 'add' over remaining factors
			rest := append([]Node{}, flatFactors[:i]...)
			rest = append(rest, flatFactors[i+1:]...)

			var expandedTerms []Node
			for _, term := range add.Terms {
				pair := append([]Node{term}, rest...)
				prod, err := simplifyMul(pair)
				if err != nil {
					return nil, err
				}
				expandedTerms = append(expandedTerms, prod)
			}
			return simplifyAdd(expandedTerms)
		}
	}

	// 3. Multiply factors together (numbers, rads, complexes, etc.)
	coeff := big.NewRat(1, 1)
	radicandProduct := big.NewInt(1)
	hasRadicand := false

	var otherFactors []Node
	complexFactors := []*ComplexNode{}

	for _, f := range flatFactors {
		switch v := f.(type) {
		case *RationalNode:
			coeff.Mul(coeff, v.Val)

		case *SqrtNode:
			if ratRad, ok := v.Radicand.(*RationalNode); ok && ratRad.Val.IsInt() && ratRad.Val.Sign() > 0 {
				hasRadicand = true
				radicandProduct.Mul(radicandProduct, ratRad.Val.Num())
			} else {
				otherFactors = append(otherFactors, v)
			}

		case *ComplexNode:
			complexFactors = append(complexFactors, v)

		case *PowNode:
			// Check for reciprocal: Pow(X, -1) -> handle rationalization later
			otherFactors = append(otherFactors, v)

		default:
			otherFactors = append(otherFactors, v)
		}
	}

	// Multiply complex factors: (a + bi)*(c + di) = (ac - bd) + (ad + bc)i
	if len(complexFactors) > 0 {
		var currentComplex *ComplexNode
		for _, cf := range complexFactors {
			if currentComplex == nil {
				currentComplex = cf
			} else {
				// Multiply currentComplex by cf
				ac, _ := simplifyMul([]Node{currentComplex.Real, cf.Real})
				bd, _ := simplifyMul([]Node{currentComplex.Imag, cf.Imag})
				negBd, _ := simplifyUnaryOp("-", bd)
				realPart, _ := simplifyAdd([]Node{ac, negBd})

				ad, _ := simplifyMul([]Node{currentComplex.Real, cf.Imag})
				bc, _ := simplifyMul([]Node{currentComplex.Imag, cf.Real})
				imagPart, _ := simplifyAdd([]Node{ad, bc})

				currentComplex = NewComplex(realPart, imagPart)
			}
		}
		// If coeff != 1, scale currentComplex
		if coeff.Cmp(big.NewRat(1, 1)) != 0 {
			scale := &RationalNode{Val: coeff}
			newReal, _ := simplifyMul([]Node{scale, currentComplex.Real})
			newImag, _ := simplifyMul([]Node{scale, currentComplex.Imag})
			currentComplex = NewComplex(newReal, newImag)
		}
		// If imag is 0, reduce to real part
		if isZero(currentComplex.Imag) {
			return currentComplex.Real, nil
		}
		if len(otherFactors) == 0 && !hasRadicand {
			return currentComplex, nil
		}
	}

	// Process radical product
	if hasRadicand {
		sqrtSimp, err := simplifySqrt(&RationalNode{Val: new(big.Rat).SetInt(radicandProduct)})
		if err != nil {
			return nil, err
		}
		switch sv := sqrtSimp.(type) {
		case *RationalNode:
			coeff.Mul(coeff, sv.Val)
		case *MulNode:
			for _, mf := range sv.Factors {
				if r, ok := mf.(*RationalNode); ok {
					coeff.Mul(coeff, r.Val)
				} else {
					otherFactors = append(otherFactors, mf)
				}
			}
		default:
			otherFactors = append(otherFactors, sv)
		}
	}

	// Check for monomial rationalization: e.g. Pow(sqrt(d), -1)
	// coeff * otherFactors * Pow(sqrt(d), -1) -> (coeff / d) * otherFactors * sqrt(d)
	var finalFactors []Node
	didRationalize := false
	for _, f := range otherFactors {
		if pow, ok := f.(*PowNode); ok {
			if rExp, ok := pow.Exp.(*RationalNode); ok && rExp.Val.Cmp(big.NewRat(-1, 1)) == 0 {
				if s, ok := pow.Base.(*SqrtNode); ok {
					if rRad, ok := s.Radicand.(*RationalNode); ok && rRad.Val.IsInt() && rRad.Val.Sign() > 0 {
						// Rationalize! Divide coeff by d, and multiply numerator by sqrt(d)
						d := rRad.Val.Num()
						coeff.Quo(coeff, new(big.Rat).SetInt(d))
						finalFactors = append(finalFactors, s)
						didRationalize = true
						continue
					}
				}
				// 1 / (c * sqrt(d))
				if mul, ok := pow.Base.(*MulNode); ok && len(mul.Factors) == 2 {
					var cRat *big.Rat
					var sNode *SqrtNode
					if r, ok := mul.Factors[0].(*RationalNode); ok {
						cRat = r.Val
						if s, ok := mul.Factors[1].(*SqrtNode); ok {
							sNode = s
						}
					}
					if cRat != nil && sNode != nil {
						if rRad, ok := sNode.Radicand.(*RationalNode); ok && rRad.Val.IsInt() && rRad.Val.Sign() > 0 {
							d := rRad.Val.Num()
							// coeff / (c * d) * sqrt(d)
							cd := new(big.Rat).Mul(cRat, new(big.Rat).SetInt(d))
							coeff.Quo(coeff, cd)
							finalFactors = append(finalFactors, sNode)
							didRationalize = true
							continue
						}
					}
				}
			}
		}
		finalFactors = append(finalFactors, f)
	}

	if didRationalize {
		all := append([]Node{&RationalNode{Val: coeff}}, finalFactors...)
		return simplifyMul(all)
	}

	// Merge powers of identical bases (e.g. pi * pi^-1 -> 1, x^2 * x^3 -> x^5)
	var mergedFactors []Node
	type baseEntry struct {
		base Node
		exp  *big.Rat
	}
	var baseList []*baseEntry

	for _, f := range finalFactors {
		var base Node
		exp := big.NewRat(1, 1)

		if pow, ok := f.(*PowNode); ok {
			base = pow.Base
			if rExp, ok := pow.Exp.(*RationalNode); ok {
				exp = new(big.Rat).Set(rExp.Val)
			} else {
				mergedFactors = append(mergedFactors, f)
				continue
			}
		} else if _, isConst := f.(*ConstNode); isConst {
			base = f
		} else if _, isVar := f.(*VarNode); isVar {
			base = f
		} else {
			mergedFactors = append(mergedFactors, f)
			continue
		}

		found := false
		for _, be := range baseList {
			if be.base.Equal(base) {
				be.exp.Add(be.exp, exp)
				found = true
				break
			}
		}
		if !found {
			baseList = append(baseList, &baseEntry{base: base, exp: exp})
		}
	}

	for _, be := range baseList {
		if be.exp.Sign() == 0 {
			// base^0 = 1 (cancels out)
			continue
		}
		one := big.NewRat(1, 1)
		if be.exp.Cmp(one) == 0 {
			mergedFactors = append(mergedFactors, be.base)
		} else {
			powSimp, err := simplifyPow(be.base, &RationalNode{Val: be.exp})
			if err != nil {
				return nil, err
			}
			mergedFactors = append(mergedFactors, powSimp)
		}
	}
	finalFactors = mergedFactors

	if coeff.Sign() == 0 {
		return mustRational(0, 1), nil
	}

	if len(finalFactors) == 0 {
		return &RationalNode{Val: coeff}, nil
	}

	one := big.NewRat(1, 1)
	if coeff.Cmp(one) == 0 {
		if len(finalFactors) == 1 {
			return finalFactors[0], nil
		}
		return NewMul(finalFactors), nil
	}

	all := append([]Node{&RationalNode{Val: coeff}}, finalFactors...)
	return NewMul(all), nil
}

// -------------------------------------------------------------------------
// Addition Simplification & Like-term Collection (Coeff × Base Model)
// -------------------------------------------------------------------------

type termEntry struct {
	coeff *big.Rat
	base  string
	node  Node
}

func simplifyAdd(terms []Node) (Node, error) {
	// 1. Flatten nested AddNodes
	var flatTerms []Node
	for _, t := range terms {
		if add, ok := t.(*AddNode); ok {
			flatTerms = append(flatTerms, add.Terms...)
		} else {
			flatTerms = append(flatTerms, t)
		}
	}

	// 2. Separate into Coeff × Base
	ratSum := big.NewRat(0, 1)
	imagSum := big.NewRat(0, 1)
	hasComplex := false
	termMap := make(map[string]*termEntry)
	var termOrder []string

	for _, t := range flatTerms {
		switch v := t.(type) {
		case *RationalNode:
			ratSum.Add(ratSum, v.Val)

		case *ComplexNode:
			hasComplex = true
			if rRat, ok := v.Real.(*RationalNode); ok {
				ratSum.Add(ratSum, rRat.Val)
			} else if !isZero(v.Real) {
				baseKey := v.Real.String()
				if entry, exists := termMap[baseKey]; exists {
					entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
				} else {
					termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v.Real}
					termOrder = append(termOrder, baseKey)
				}
			}
			if iRat, ok := v.Imag.(*RationalNode); ok {
				imagSum.Add(imagSum, iRat.Val)
			} else if !isZero(v.Imag) {
				baseKey := fmt.Sprintf("(%s)*i", v.Imag.String())
				if entry, exists := termMap[baseKey]; exists {
					entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
				} else {
					termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
					termOrder = append(termOrder, baseKey)
				}
			}

		case *SqrtNode:
			baseKey := v.String()
			if entry, exists := termMap[baseKey]; exists {
				entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
			} else {
				termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
				termOrder = append(termOrder, baseKey)
			}

		case *MulNode:
			// Check if v is Coeff * Base
			if len(v.Factors) >= 2 {
				if r, ok := v.Factors[0].(*RationalNode); ok {
					var rest []Node
					rest = append(rest, v.Factors[1:]...)
					var baseNode Node
					if len(rest) == 1 {
						baseNode = rest[0]
					} else {
						baseNode = NewMul(rest)
					}
					baseKey := baseNode.String()
					if entry, exists := termMap[baseKey]; exists {
						entry.coeff.Add(entry.coeff, r.Val)
					} else {
						termMap[baseKey] = &termEntry{coeff: new(big.Rat).Set(r.Val), base: baseKey, node: baseNode}
						termOrder = append(termOrder, baseKey)
					}
					continue
				}
			}
			baseKey := v.String()
			if entry, exists := termMap[baseKey]; exists {
				entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
			} else {
				termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
				termOrder = append(termOrder, baseKey)
			}

		default:
			baseKey := v.String()
			if entry, exists := termMap[baseKey]; exists {
				entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
			} else {
				termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
				termOrder = append(termOrder, baseKey)
			}
		}
	}

	// 3. Assemble combined terms
	var resultTerms []Node
	if ratSum.Sign() != 0 {
		resultTerms = append(resultTerms, &RationalNode{Val: ratSum})
	}

	for _, key := range termOrder {
		entry := termMap[key]
		if entry.coeff.Sign() == 0 {
			continue // collected to 0
		}
		one := big.NewRat(1, 1)
		if entry.coeff.Cmp(one) == 0 {
			resultTerms = append(resultTerms, entry.node)
		} else {
			resultTerms = append(resultTerms, NewMul([]Node{&RationalNode{Val: entry.coeff}, entry.node}))
		}
	}

	if hasComplex && imagSum.Sign() != 0 {
		var realPart Node
		if len(resultTerms) == 0 {
			realPart = mustRational(0, 1)
		} else if len(resultTerms) == 1 {
			realPart = resultTerms[0]
		} else {
			realPart = NewAdd(resultTerms)
		}
		return NewComplex(realPart, &RationalNode{Val: imagSum}), nil
	}

	if len(resultTerms) == 0 {
		return mustRational(0, 1), nil
	}
	if len(resultTerms) == 1 {
		return resultTerms[0], nil
	}

	return NewAdd(resultTerms), nil
}

func isZero(n Node) bool {
	if r, ok := n.(*RationalNode); ok && r.Val.Sign() == 0 {
		return true
	}
	return false
}

// Suppress unused imports
var _ = sort.Strings
var _ = strings.Join
