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

	case *ListNode:
		evaledElements := make([]Node, len(v.Elements))
		for i, e := range v.Elements {
			ee, err := EvalWithEnv(e, env)
			if err != nil {
				return nil, err
			}
			evaledElements[i] = ee
		}
		return NewList(evaledElements), nil

	case *MatrixNode:
		evaledData := make([][]Node, v.Rows)
		for r := 0; r < v.Rows; r++ {
			evaledData[r] = make([]Node, v.Cols)
			for c := 0; c < v.Cols; c++ {
				ec, err := EvalWithEnv(v.Data[r][c], env)
				if err != nil {
					return nil, err
				}
				evaledData[r][c] = ec
			}
		}
		return NewMatrix(v.Rows, v.Cols, evaledData)

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
		case *MatrixNode:
			negData := make([][]Node, v.Rows)
			for r := 0; r < v.Rows; r++ {
				negData[r] = make([]Node, v.Cols)
				for c := 0; c < v.Cols; c++ {
					negElem, err := simplifyUnaryOp("-", v.Data[r][c])
					if err != nil {
						return nil, err
					}
					negData[r][c] = negElem
				}
			}
			return NewMatrix(v.Rows, v.Cols, negData)
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

	case "expand":
		return expandNode(args[0]), nil

	case "diff":
		varName := "x"
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("diff error: second argument must be a variable name, got %s", args[1].String())
		}
		return differentiate(args[0], varName)

	case "solve":
		varName := "x"
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("solve error: second argument must be a variable name, got %s", args[1].String())
		}
		return solveEquation(args[0], varName)

	case "det":
		mat, ok := args[0].(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("det error: argument must be a matrix, got %s", args[0].String())
		}
		return evalDet(mat)

	case "inv":
		mat, ok := args[0].(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("inv error: argument must be a matrix, got %s", args[0].String())
		}
		return evalInv(mat)

	case "transpose":
		mat, ok := args[0].(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("transpose error: argument must be a matrix, got %s", args[0].String())
		}
		return evalTranspose(mat), nil

	case "taylor":
		return evalTaylor(args[0], args[1], args[2], args[3])

	case "sum":
		return evalSum(args[0], args[1], args[2], args[3])

	case "dot":
		return evalDot(args[0], args[1])

	case "cross":
		return evalCross(args[0], args[1])

	case "norm":
		return evalNorm(args[0])

	case "grad":
		return evalGrad(args[0], args[1])

	case "div":
		return evalDiv(args[0], args[1])

	case "curl":
		return evalCurl(args[0], args[1])

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

	// 1.5. Check if any factor is a MatrixNode (non-commutative sequential multiplication)
	hasMatrix := false
	for _, f := range flatFactors {
		if _, ok := f.(*MatrixNode); ok {
			hasMatrix = true
			break
		}
	}
	if hasMatrix {
		if len(flatFactors) == 0 {
			return mustRational(1, 1), nil
		}
		current := flatFactors[0]
		for i := 1; i < len(flatFactors); i++ {
			next := flatFactors[i]
			prod, err := mulMatrixOrScalar(current, next)
			if err != nil {
				return nil, err
			}
			current = prod
		}
		return current, nil
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

	// 1.5. Check if any term is a MatrixNode
	var firstMatrix *MatrixNode
	for _, t := range flatTerms {
		if m, ok := t.(*MatrixNode); ok {
			firstMatrix = m
			break
		}
	}
	if firstMatrix != nil {
		resData := make([][]Node, firstMatrix.Rows)
		for r := 0; r < firstMatrix.Rows; r++ {
			resData[r] = make([]Node, firstMatrix.Cols)
			for c := 0; c < firstMatrix.Cols; c++ {
				resData[r][c] = mustRational(0, 1)
			}
		}
		for _, t := range flatTerms {
			m, ok := t.(*MatrixNode)
			if !ok {
				return nil, fmt.Errorf("matrix dimension error: cannot add scalar %s to matrix", t.String())
			}
			if m.Rows != firstMatrix.Rows || m.Cols != firstMatrix.Cols {
				return nil, fmt.Errorf("matrix dimension mismatch: cannot add %dx%d matrix and %dx%d matrix",
					firstMatrix.Rows, firstMatrix.Cols, m.Rows, m.Cols)
			}
			for r := 0; r < m.Rows; r++ {
				for c := 0; c < m.Cols; c++ {
					sum, err := simplifyAdd([]Node{resData[r][c], m.Data[r][c]})
					if err != nil {
						return nil, err
					}
					resData[r][c] = sum
				}
			}
		}
		return NewMatrix(firstMatrix.Rows, firstMatrix.Cols, resData)
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

// -------------------------------------------------------------------------
// CAS: Polynomial Expansion (expand)
// -------------------------------------------------------------------------

// expandNode recursively expands an AST node using distributive laws and binomial expansion.
func expandNode(n Node) Node {
	if n == nil {
		return nil
	}
	switch v := n.(type) {
	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = expandNode(t)
		}
		res, _ := simplifyAdd(newTerms)
		return res

	case *MulNode:
		if len(v.Factors) == 0 {
			return mustRational(1, 1)
		}
		current := expandNode(v.Factors[0])
		for i := 1; i < len(v.Factors); i++ {
			next := expandNode(v.Factors[i])
			current = expandMul2(current, next)
		}
		return current

	case *PowNode:
		base := expandNode(v.Base)
		if r, ok := v.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() >= 0 {
			expInt := r.Val.Num().Int64()
			if expInt == 0 {
				return mustRational(1, 1)
			}
			if expInt == 1 {
				return base
			}
			if expInt <= 10 {
				res := base
				for i := int64(1); i < expInt; i++ {
					res = expandMul2(res, base)
				}
				return res
			}
		}
		res, err := simplifyPow(base, v.Exp)
		if err != nil {
			return &PowNode{Base: base, Exp: v.Exp}
		}
		return res

	case *UnaryOpNode:
		if v.Op == "-" {
			expanded := expandNode(v.Expr)
			res, _ := simplifyUnaryOp("-", expanded)
			return res
		}
		return v

	case *FuncNode:
		newArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			newArgs[i] = expandNode(a)
		}
		res, err := simplifyFunc(v.Name, newArgs)
		if err != nil {
			return &FuncNode{Name: v.Name, Args: newArgs}
		}
		return res

	default:
		return n
	}
}

// expandMul2 multiplies two already expanded nodes using the distributive law.
func expandMul2(a, b Node) Node {
	addA, isAddA := a.(*AddNode)
	addB, isAddB := b.(*AddNode)

	if isAddA && isAddB {
		var terms []Node
		for _, ta := range addA.Terms {
			for _, tb := range addB.Terms {
				prod, _ := simplifyMul([]Node{ta, tb})
				terms = append(terms, prod)
			}
		}
		res, _ := simplifyAdd(terms)
		return res
	} else if isAddA {
		var terms []Node
		for _, ta := range addA.Terms {
			prod, _ := simplifyMul([]Node{ta, b})
			terms = append(terms, prod)
		}
		res, _ := simplifyAdd(terms)
		return res
	} else if isAddB {
		var terms []Node
		for _, tb := range addB.Terms {
			prod, _ := simplifyMul([]Node{a, tb})
			terms = append(terms, prod)
		}
		res, _ := simplifyAdd(terms)
		return res
	} else {
		prod, _ := simplifyMul([]Node{a, b})
		return prod
	}
}

// -------------------------------------------------------------------------
// CAS: Symbolic Differentiation (diff)
// -------------------------------------------------------------------------

// containsVar checks if an AST node contains the given variable name.
func containsVar(n Node, varName string) bool {
	if n == nil {
		return false
	}
	switch v := n.(type) {
	case *VarNode:
		return v.Name == varName
	case *AddNode:
		for _, t := range v.Terms {
			if containsVar(t, varName) {
				return true
			}
		}
		return false
	case *MulNode:
		for _, f := range v.Factors {
			if containsVar(f, varName) {
				return true
			}
		}
		return false
	case *PowNode:
		return containsVar(v.Base, varName) || containsVar(v.Exp, varName)
	case *UnaryOpNode:
		return containsVar(v.Expr, varName)
	case *FuncNode:
		for _, a := range v.Args {
			if containsVar(a, varName) {
				return true
			}
		}
		return false
	case *SqrtNode:
		return containsVar(v.Radicand, varName)
	case *ComplexNode:
		return containsVar(v.Real, varName) || containsVar(v.Imag, varName)
	case *ListNode:
		for _, e := range v.Elements {
			if containsVar(e, varName) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// differentiate computes the exact symbolic derivative of node n with respect to varName.
func differentiate(n Node, varName string) (Node, error) {
	if n == nil {
		return nil, fmt.Errorf("cannot differentiate nil node")
	}

	// If n does not contain varName, derivative is 0
	if !containsVar(n, varName) {
		return mustRational(0, 1), nil
	}

	switch v := n.(type) {
	case *VarNode:
		if v.Name == varName {
			return mustRational(1, 1), nil
		}
		return mustRational(0, 1), nil

	case *AddNode:
		dTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			dt, err := differentiate(t, varName)
			if err != nil {
				return nil, err
			}
			dTerms[i] = dt
		}
		return simplifyAdd(dTerms)

	case *UnaryOpNode:
		if v.Op == "-" {
			de, err := differentiate(v.Expr, varName)
			if err != nil {
				return nil, err
			}
			return simplifyUnaryOp("-", de)
		}
		return nil, fmt.Errorf("cannot differentiate unary operator %s", v.Op)

	case *MulNode:
		// Product rule: d(f_1 * ... * f_k) = sum_i (f_i' * prod_{j != i} f_j)
		var sumTerms []Node
		for i, fi := range v.Factors {
			dfi, err := differentiate(fi, varName)
			if err != nil {
				return nil, err
			}
			if isZero(dfi) {
				continue
			}
			var prodFactors []Node
			prodFactors = append(prodFactors, dfi)
			for j, fj := range v.Factors {
				if j != i {
					prodFactors = append(prodFactors, fj)
				}
			}
			p, err := simplifyMul(prodFactors)
			if err != nil {
				return nil, err
			}
			sumTerms = append(sumTerms, p)
		}
		if len(sumTerms) == 0 {
			return mustRational(0, 1), nil
		}
		return simplifyAdd(sumTerms)

	case *PowNode:
		baseHas := containsVar(v.Base, varName)
		expHas := containsVar(v.Exp, varName)

		if baseHas && !expHas {
			// d/dx [ u(x)^n ] = n * u(x)^(n-1) * u'(x)
			du, err := differentiate(v.Base, varName)
			if err != nil {
				return nil, err
			}
			expMinus1, err := simplifyAdd([]Node{v.Exp, mustRational(-1, 1)})
			if err != nil {
				return nil, err
			}
			uPow, err := simplifyPow(v.Base, expMinus1)
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{v.Exp, uPow, du})
		} else if !baseHas && expHas {
			// d/dx [ a^v(x) ] = a^v(x) * ln(a) * v'(x)
			dv, err := differentiate(v.Exp, varName)
			if err != nil {
				return nil, err
			}
			var lnBase Node
			if c, ok := v.Base.(*ConstNode); ok && c.Name == "e" {
				lnBase = mustRational(1, 1)
			} else {
				lnBase, err = simplifyFunc("ln", []Node{v.Base})
				if err != nil {
					return nil, err
				}
			}
			return simplifyMul([]Node{v, lnBase, dv})
		} else {
			// General form: d/dx [ u^v ] = u^v * ( v' * ln(u) + v * u'/u )
			du, err := differentiate(v.Base, varName)
			if err != nil {
				return nil, err
			}
			dv, err := differentiate(v.Exp, varName)
			if err != nil {
				return nil, err
			}
			lnU, err := simplifyFunc("ln", []Node{v.Base})
			if err != nil {
				return nil, err
			}
			term1, err := simplifyMul([]Node{dv, lnU})
			if err != nil {
				return nil, err
			}
			uInv, err := simplifyPow(v.Base, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			term2, err := simplifyMul([]Node{v.Exp, du, uInv})
			if err != nil {
				return nil, err
			}
			bracket, err := simplifyAdd([]Node{term1, term2})
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{v, bracket})
		}

	case *SqrtNode:
		// sqrt(u) = u^(1/2) -> d/dx = 1/2 * u' * sqrt(u)^-1
		du, err := differentiate(v.Radicand, varName)
		if err != nil {
			return nil, err
		}
		invSqrt, err := simplifyPow(v, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		return simplifyMul([]Node{mustRational(1, 2), du, invSqrt})

	case *FuncNode:
		if len(v.Args) == 1 {
			u := v.Args[0]
			du, err := differentiate(u, varName)
			if err != nil {
				return nil, err
			}
			var dfDu Node
			switch v.Name {
			case "sin":
				dfDu, err = simplifyFunc("cos", []Node{u})
			case "cos":
				sinU, err2 := simplifyFunc("sin", []Node{u})
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyUnaryOp("-", sinU)
			case "tan":
				cosU, err2 := simplifyFunc("cos", []Node{u})
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyPow(cosU, mustRational(-2, 1))
			case "ln":
				dfDu, err = simplifyPow(u, mustRational(-1, 1))
			case "log":
				uInv, err2 := simplifyPow(u, mustRational(-1, 1))
				if err2 != nil {
					return nil, err2
				}
				ln10, err2 := simplifyFunc("ln", []Node{mustRational(10, 1)})
				if err2 != nil {
					return nil, err2
				}
				ln10Inv, err2 := simplifyPow(ln10, mustRational(-1, 1))
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyMul([]Node{uInv, ln10Inv})
			case "asin":
				u2, err2 := simplifyPow(u, mustRational(2, 1))
				if err2 != nil {
					return nil, err2
				}
				negU2, err2 := simplifyUnaryOp("-", u2)
				if err2 != nil {
					return nil, err2
				}
				oneMinusU2, err2 := simplifyAdd([]Node{mustRational(1, 1), negU2})
				if err2 != nil {
					return nil, err2
				}
				sqrtVal, err2 := simplifySqrt(oneMinusU2)
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyPow(sqrtVal, mustRational(-1, 1))
			case "acos":
				u2, err2 := simplifyPow(u, mustRational(2, 1))
				if err2 != nil {
					return nil, err2
				}
				negU2, err2 := simplifyUnaryOp("-", u2)
				if err2 != nil {
					return nil, err2
				}
				oneMinusU2, err2 := simplifyAdd([]Node{mustRational(1, 1), negU2})
				if err2 != nil {
					return nil, err2
				}
				sqrtVal, err2 := simplifySqrt(oneMinusU2)
				if err2 != nil {
					return nil, err2
				}
				posInv, err2 := simplifyPow(sqrtVal, mustRational(-1, 1))
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyUnaryOp("-", posInv)
			case "atan":
				u2, err2 := simplifyPow(u, mustRational(2, 1))
				if err2 != nil {
					return nil, err2
				}
				onePlusU2, err2 := simplifyAdd([]Node{mustRational(1, 1), u2})
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyPow(onePlusU2, mustRational(-1, 1))
			case "cbrt":
				uPow, err2 := simplifyPow(u, mustRational(-2, 3))
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyMul([]Node{mustRational(1, 3), uPow})
			case "abs":
				absU, err2 := simplifyFunc("abs", []Node{u})
				if err2 != nil {
					return nil, err2
				}
				absUInv, err2 := simplifyPow(absU, mustRational(-1, 1))
				if err2 != nil {
					return nil, err2
				}
				dfDu, err = simplifyMul([]Node{u, absUInv})
			default:
				return nil, fmt.Errorf("differentiation of function %s is not supported", v.Name)
			}
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{dfDu, du})
		} else if v.Name == "log" && len(v.Args) == 2 {
			if containsVar(v.Args[0], varName) {
				return nil, fmt.Errorf("differentiation of log with variable base is not supported")
			}
			u := v.Args[1]
			du, err := differentiate(u, varName)
			if err != nil {
				return nil, err
			}
			uInv, err := simplifyPow(u, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			lnBase, err := simplifyFunc("ln", []Node{v.Args[0]})
			if err != nil {
				return nil, err
			}
			lnBaseInv, err := simplifyPow(lnBase, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{uInv, lnBaseInv, du})
		}
		return nil, fmt.Errorf("cannot differentiate function %s with %d arguments", v.Name, len(v.Args))

	default:
		return nil, fmt.Errorf("cannot differentiate node of type %T", n)
	}
}

// -------------------------------------------------------------------------
// CAS: Polynomial Equation Solving (solve)
// -------------------------------------------------------------------------

// extractPolyCoeffs extracts polynomial coefficients a_k of expr with respect to varName.
func extractPolyCoeffs(expr Node, varName string) (map[int]Node, error) {
	coeffs := make(map[int]Node)

	addCoeff := func(deg int, coeff Node) error {
		if existing, ok := coeffs[deg]; ok {
			sum, err := simplifyAdd([]Node{existing, coeff})
			if err != nil {
				return err
			}
			coeffs[deg] = sum
		} else {
			coeffs[deg] = coeff
		}
		return nil
	}

	var processTerm func(t Node) error
	processTerm = func(t Node) error {
		if !containsVar(t, varName) {
			return addCoeff(0, t)
		}
		if v, ok := t.(*VarNode); ok && v.Name == varName {
			return addCoeff(1, mustRational(1, 1))
		}
		if pow, ok := t.(*PowNode); ok {
			if v, ok := pow.Base.(*VarNode); ok && v.Name == varName {
				if r, ok := pow.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() > 0 {
					return addCoeff(int(r.Val.Num().Int64()), mustRational(1, 1))
				}
			}
			return fmt.Errorf("solve error: non-polynomial exponent in %s", t.String())
		}
		if mul, ok := t.(*MulNode); ok {
			var varFactor Node
			var otherFactors []Node
			varFound := false
			for _, f := range mul.Factors {
				if containsVar(f, varName) {
					if varFound {
						return fmt.Errorf("solve error: multiple variable factors in term %s", t.String())
					}
					varFactor = f
					varFound = true
				} else {
					otherFactors = append(otherFactors, f)
				}
			}
			coeffNode, err := simplifyMul(otherFactors)
			if err != nil {
				return err
			}
			if v, ok := varFactor.(*VarNode); ok && v.Name == varName {
				return addCoeff(1, coeffNode)
			}
			if pow, ok := varFactor.(*PowNode); ok {
				if v, ok := pow.Base.(*VarNode); ok && v.Name == varName {
					if r, ok := pow.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() > 0 {
						return addCoeff(int(r.Val.Num().Int64()), coeffNode)
					}
				}
			}
			return fmt.Errorf("solve error: non-polynomial factor in term %s", t.String())
		}
		return fmt.Errorf("solve error: non-polynomial term %s", t.String())
	}

	if add, ok := expr.(*AddNode); ok {
		for _, t := range add.Terms {
			if err := processTerm(t); err != nil {
				return nil, err
			}
		}
	} else {
		if err := processTerm(expr); err != nil {
			return nil, err
		}
	}

	// Clean zero coefficients
	for d, c := range coeffs {
		if isZero(c) {
			delete(coeffs, d)
		}
	}

	return coeffs, nil
}

// solveEquation algebraically solves expr = 0 for varName.
func solveEquation(expr Node, varName string) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot solve nil equation")
	}

	// 1. Expand the expression to standard form
	expanded := expandNode(expr)

	// Check if varName is in the equation
	if !containsVar(expanded, varName) {
		if isZero(expanded) {
			return nil, fmt.Errorf("solve: identity equation (infinite solutions)")
		}
		return nil, fmt.Errorf("solve: equation has no solution (contradiction: %s = 0)", expanded.String())
	}

	// 2. Extract polynomial coefficients
	coeffs, err := extractPolyCoeffs(expanded, varName)
	if err != nil {
		return nil, err
	}

	maxDeg := 0
	for d := range coeffs {
		if d > maxDeg {
			maxDeg = d
		}
	}

	if maxDeg == 0 {
		return nil, fmt.Errorf("solve: equation contains no degree of %s", varName)
	}

	a0 := coeffs[0]
	if a0 == nil {
		a0 = mustRational(0, 1)
	}

	if maxDeg == 1 {
		// Linear equation: a1 * x + a0 = 0  =>  x = -a0 / a1
		a1 := coeffs[1]
		negA0, err := simplifyUnaryOp("-", a0)
		if err != nil {
			return nil, err
		}
		invA1, err := simplifyPow(a1, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		root, err := simplifyMul([]Node{negA0, invA1})
		if err != nil {
			return nil, err
		}
		return NewList([]Node{root}), nil
	}

	if maxDeg == 2 {
		// Quadratic equation: a2 * x^2 + a1 * x + a0 = 0
		a2 := coeffs[2]
		a1 := coeffs[1]
		if a1 == nil {
			a1 = mustRational(0, 1)
		}

		// Discriminant D = a1^2 - 4 * a2 * a0
		a1Sq, err := simplifyPow(a1, mustRational(2, 1))
		if err != nil {
			return nil, err
		}
		fourA2A0, err := simplifyMul([]Node{mustRational(4, 1), a2, a0})
		if err != nil {
			return nil, err
		}
		negFour, err := simplifyUnaryOp("-", fourA2A0)
		if err != nil {
			return nil, err
		}
		d, err := simplifyAdd([]Node{a1Sq, negFour})
		if err != nil {
			return nil, err
		}

		negA1, err := simplifyUnaryOp("-", a1)
		if err != nil {
			return nil, err
		}
		twoA2, err := simplifyMul([]Node{mustRational(2, 1), a2})
		if err != nil {
			return nil, err
		}
		invTwoA2, err := simplifyPow(twoA2, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}

		if isZero(d) {
			// Single repeated root: x = -a1 / (2*a2)
			root, err := simplifyMul([]Node{negA1, invTwoA2})
			if err != nil {
				return nil, err
			}
			return NewList([]Node{root}), nil
		}

		sqrtD, err := simplifySqrt(d)
		if err != nil {
			return nil, err
		}
		negSqrtD, err := simplifyUnaryOp("-", sqrtD)
		if err != nil {
			return nil, err
		}

		// root1 = (-a1 - sqrt(D)) / (2*a2)
		// root2 = (-a1 + sqrt(D)) / (2*a2)
		num1, err := simplifyAdd([]Node{negA1, negSqrtD})
		if err != nil {
			return nil, err
		}
		root1, err := simplifyMul([]Node{num1, invTwoA2})
		if err != nil {
			return nil, err
		}

		num2, err := simplifyAdd([]Node{negA1, sqrtD})
		if err != nil {
			return nil, err
		}
		root2, err := simplifyMul([]Node{num2, invTwoA2})
		if err != nil {
			return nil, err
		}

		return NewList([]Node{root1, root2}), nil
	}

	return nil, fmt.Errorf("solve error: polynomial degree %d is not currently supported (only linear and quadratic equations)", maxDeg)
}

// -------------------------------------------------------------------------
// CAS: Matrix Operations (mulMatrixOrScalar, det, inv, transpose)
// -------------------------------------------------------------------------

func mulMatrixOrScalar(a, b Node) (Node, error) {
	matA, isMatA := a.(*MatrixNode)
	matB, isMatB := b.(*MatrixNode)

	if isMatA && isMatB {
		if matA.Cols != matB.Rows {
			return nil, fmt.Errorf("matrix dimension mismatch: cannot multiply %dx%d matrix by %dx%d matrix",
				matA.Rows, matA.Cols, matB.Rows, matB.Cols)
		}
		resData := make([][]Node, matA.Rows)
		for r := 0; r < matA.Rows; r++ {
			resData[r] = make([]Node, matB.Cols)
			for c := 0; c < matB.Cols; c++ {
				var dotTerms []Node
				for k := 0; k < matA.Cols; k++ {
					prod, err := simplifyMul([]Node{matA.Data[r][k], matB.Data[k][c]})
					if err != nil {
						return nil, err
					}
					dotTerms = append(dotTerms, prod)
				}
				sum, err := simplifyAdd(dotTerms)
				if err != nil {
					return nil, err
				}
				resData[r][c] = sum
			}
		}
		return NewMatrix(matA.Rows, matB.Cols, resData)
	}

	if isMatA {
		// Matrix * Scalar
		resData := make([][]Node, matA.Rows)
		for r := 0; r < matA.Rows; r++ {
			resData[r] = make([]Node, matA.Cols)
			for c := 0; c < matA.Cols; c++ {
				prod, err := simplifyMul([]Node{matA.Data[r][c], b})
				if err != nil {
					return nil, err
				}
				resData[r][c] = prod
			}
		}
		return NewMatrix(matA.Rows, matA.Cols, resData)
	}

	if isMatB {
		// Scalar * Matrix
		resData := make([][]Node, matB.Rows)
		for r := 0; r < matB.Rows; r++ {
			resData[r] = make([]Node, matB.Cols)
			for c := 0; c < matB.Cols; c++ {
				prod, err := simplifyMul([]Node{a, matB.Data[r][c]})
				if err != nil {
					return nil, err
				}
				resData[r][c] = prod
			}
		}
		return NewMatrix(matB.Rows, matB.Cols, resData)
	}

	return simplifyMul([]Node{a, b})
}

// submatrix returns a copy of m without dropR row and dropC col.
func submatrix(m *MatrixNode, dropR, dropC int) *MatrixNode {
	var data [][]Node
	for r := 0; r < m.Rows; r++ {
		if r == dropR {
			continue
		}
		var row []Node
		for c := 0; c < m.Cols; c++ {
			if c == dropC {
				continue
			}
			row = append(row, m.Data[r][c])
		}
		data = append(data, row)
	}
	res, _ := NewMatrix(m.Rows-1, m.Cols-1, data)
	return res
}

// evalDet calculates the exact determinant using division-free Laplace expansion.
func evalDet(m *MatrixNode) (Node, error) {
	if m.Rows != m.Cols {
		return nil, fmt.Errorf("matrix dimension error: det requires square matrix, got %dx%d", m.Rows, m.Cols)
	}

	n := m.Rows
	if n == 1 {
		return m.Data[0][0], nil
	}

	if n == 2 {
		// ad - bc
		ad, err := simplifyMul([]Node{m.Data[0][0], m.Data[1][1]})
		if err != nil {
			return nil, err
		}
		bc, err := simplifyMul([]Node{m.Data[0][1], m.Data[1][0]})
		if err != nil {
			return nil, err
		}
		negBC, err := simplifyUnaryOp("-", bc)
		if err != nil {
			return nil, err
		}
		return simplifyAdd([]Node{ad, negBC})
	}

	// n >= 3: Laplace expansion along row 0
	var sumTerms []Node
	for c := 0; c < n; c++ {
		elem := m.Data[0][c]
		if isZero(elem) {
			continue
		}
		sub := submatrix(m, 0, c)
		subDet, err := evalDet(sub)
		if err != nil {
			return nil, err
		}
		term, err := simplifyMul([]Node{elem, subDet})
		if err != nil {
			return nil, err
		}
		if c%2 == 1 {
			negTerm, err := simplifyUnaryOp("-", term)
			if err != nil {
				return nil, err
			}
			sumTerms = append(sumTerms, negTerm)
		} else {
			sumTerms = append(sumTerms, term)
		}
	}

	if len(sumTerms) == 0 {
		return mustRational(0, 1), nil
	}
	return simplifyAdd(sumTerms)
}

// evalInv calculates the exact matrix inverse using the adjugate matrix method.
func evalInv(m *MatrixNode) (*MatrixNode, error) {
	if m.Rows != m.Cols {
		return nil, fmt.Errorf("matrix dimension error: inv requires square matrix, got %dx%d", m.Rows, m.Cols)
	}

	d, err := evalDet(m)
	if err != nil {
		return nil, err
	}
	if isZero(d) {
		return nil, fmt.Errorf("math error: singular matrix (det = 0), inverse does not exist")
	}

	invDet, err := simplifyPow(d, mustRational(-1, 1))
	if err != nil {
		return nil, err
	}

	n := m.Rows
	if n == 1 {
		invElem, err := simplifyPow(m.Data[0][0], mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		return NewMatrix(1, 1, [][]Node{{invElem}})
	}

	// Adjugate matrix: adj(A)[r][c] = (-1)^(r+c) * det(submatrix(m, c, r)) (note transposed c, r!)
	invData := make([][]Node, n)
	for r := 0; r < n; r++ {
		invData[r] = make([]Node, n)
		for c := 0; c < n; c++ {
			sub := submatrix(m, c, r)
			subDet, err := evalDet(sub)
			if err != nil {
				return nil, err
			}
			var cofactor Node
			if (r+c)%2 == 1 {
				cofactor, err = simplifyUnaryOp("-", subDet)
				if err != nil {
					return nil, err
				}
			} else {
				cofactor = subDet
			}

			// elem = cofactor * (1/det)
			elem, err := simplifyMul([]Node{cofactor, invDet})
			if err != nil {
				return nil, err
			}
			invData[r][c] = elem
		}
	}

	return NewMatrix(n, n, invData)
}

// evalTranspose returns the transpose of matrix m.
func evalTranspose(m *MatrixNode) *MatrixNode {
	tData := make([][]Node, m.Cols)
	for c := 0; c < m.Cols; c++ {
		tData[c] = make([]Node, m.Rows)
		for r := 0; r < m.Rows; r++ {
			tData[c][r] = m.Data[r][c]
		}
	}
	res, _ := NewMatrix(m.Cols, m.Rows, tData)
	return res
}

// -------------------------------------------------------------------------
// Taylor / Maclaurin Series Expansion
// -------------------------------------------------------------------------

func isZeroNode(n Node) bool {
	if n == nil {
		return false
	}
	switch v := n.(type) {
	case *RationalNode:
		return v.Val.Sign() == 0
	case *MulNode:
		for _, f := range v.Factors {
			if isZeroNode(f) {
				return true
			}
		}
	}
	return false
}

func evalTaylor(f Node, varNode Node, center Node, orderNode Node) (Node, error) {
	v, ok := varNode.(*VarNode)
	if !ok {
		return nil, fmt.Errorf("taylor error: second argument must be a variable, got %s", varNode.String())
	}
	varName := v.Name

	rOrder, ok := orderNode.(*RationalNode)
	if !ok || !rOrder.Val.IsInt() || rOrder.Val.Sign() < 0 {
		return nil, fmt.Errorf("taylor error: order must be non-negative integer, got %s", orderNode.String())
	}
	n := rOrder.Val.Num().Int64()

	subEnv := NewEnv()
	subEnv.Set(varName, center)

	currentDeriv := f
	var terms []Node

	kFact := big.NewInt(1)

	for k := int64(0); k <= n; k++ {
		if k > 0 {
			kFact.Mul(kFact, big.NewInt(k))
			d, err := differentiate(currentDeriv, varName)
			if err != nil {
				return nil, fmt.Errorf("taylor error in %d-th derivative: %w", k, err)
			}
			currentDeriv = d
		}

		// evaluate f^(k)(center)
		fVal, err := EvalWithEnv(currentDeriv, subEnv)
		if err != nil {
			return nil, fmt.Errorf("taylor error: cannot evaluate derivative at center: %w", err)
		}

		if isZeroNode(fVal) {
			continue
		}

		// coeff = fVal / k!
		factRat := new(big.Rat).SetInt(kFact)
		invFact := new(big.Rat).Inv(factRat)
		coeff, err := simplifyMul([]Node{fVal, NewRationalFromBigRat(invFact)})
		if err != nil {
			return nil, err
		}
		if isZeroNode(coeff) {
			continue
		}

		// term = coeff * (x - center)^k
		var powerNode Node
		if isZeroNode(center) {
			switch k {
			case 0:
				powerNode = mustRational(1, 1)
			case 1:
				powerNode = v
			default:
				powerNode = &PowNode{Base: v, Exp: mustRational(k, 1)}
			}
		} else {
			diffTerm, err := simplifyAdd([]Node{v, &MulNode{Factors: []Node{mustRational(-1, 1), center}}})
			if err != nil {
				return nil, err
			}
			switch k {
			case 0:
				powerNode = mustRational(1, 1)
			case 1:
				powerNode = diffTerm
			default:
				powerNode = expandNode(&PowNode{Base: diffTerm, Exp: mustRational(k, 1)})
			}
		}

		term, err := simplifyMul([]Node{coeff, powerNode})
		if err != nil {
			return nil, err
		}
		term = expandNode(term)
		terms = append(terms, term)
	}

	if len(terms) == 0 {
		return mustRational(0, 1), nil
	}
	res, err := simplifyAdd(terms)
	if err != nil {
		return nil, err
	}
	return expandNode(res), nil
}

// -------------------------------------------------------------------------
// Discrete Series & Summation Engine (Faulhaber & Bernoulli)
// -------------------------------------------------------------------------

func computeBernoulli(m int) []*big.Rat {
	B := make([]*big.Rat, m+1)
	for i := 0; i <= m; i++ {
		B[i] = big.NewRat(0, 1)
	}
	B[0] = big.NewRat(1, 1)

	for i := 1; i <= m; i++ {
		sum := big.NewRat(0, 1)
		for j := 0; j < i; j++ {
			c := big.NewInt(0).Binomial(int64(i+1), int64(j))
			term := new(big.Rat).Mul(new(big.Rat).SetInt(c), B[j])
			sum.Add(sum, term)
		}
		denom := big.NewRat(int64(i+1), 1)
		B[i].Quo(new(big.Rat).Neg(sum), denom)
	}

	if m >= 1 {
		B[1] = big.NewRat(1, 2)
	}
	return B
}

func faulhaberSum(p int64, nVar Node) (Node, error) {
	if p == 0 {
		return nVar, nil
	}
	B := computeBernoulli(int(p))
	var terms []Node

	pPlus1 := p + 1
	pPlus1Rat := big.NewRat(pPlus1, 1)

	for j := int64(0); j <= p; j++ {
		c := big.NewInt(0).Binomial(pPlus1, j)
		coeffRat := new(big.Rat).Mul(new(big.Rat).SetInt(c), B[j])
		coeffRat.Quo(coeffRat, pPlus1Rat)

		if coeffRat.Sign() == 0 {
			continue
		}

		expVal := pPlus1 - j
		var nPow Node
		if expVal == 1 {
			nPow = nVar
		} else {
			nPow = &PowNode{Base: nVar, Exp: mustRational(expVal, 1)}
		}

		term, err := simplifyMul([]Node{NewRationalFromBigRat(coeffRat), nPow})
		if err != nil {
			return nil, err
		}
		terms = append(terms, term)
	}

	res, err := simplifyAdd(terms)
	if err != nil {
		return nil, err
	}
	return expandNode(res), nil
}

func extractPowerOfK(term Node, kVar string) (Node, int64, error) {
	if !containsVar(term, kVar) {
		return term, 0, nil
	}
	if v, ok := term.(*VarNode); ok && v.Name == kVar {
		return mustRational(1, 1), 1, nil
	}
	if pow, ok := term.(*PowNode); ok {
		if v, ok := pow.Base.(*VarNode); ok && v.Name == kVar {
			if rExp, ok := pow.Exp.(*RationalNode); ok && rExp.Val.IsInt() && rExp.Val.Sign() >= 0 {
				return mustRational(1, 1), rExp.Val.Num().Int64(), nil
			}
		}
	}
	if mul, ok := term.(*MulNode); ok {
		var otherFactors []Node
		totalPow := int64(0)
		for _, f := range mul.Factors {
			if containsVar(f, kVar) {
				subCoeff, subPow, err := extractPowerOfK(f, kVar)
				if err != nil {
					return nil, 0, err
				}
				if !isOneRat(subCoeff) {
					otherFactors = append(otherFactors, subCoeff)
				}
				totalPow += subPow
			} else {
				otherFactors = append(otherFactors, f)
			}
		}
		coeff, err := simplifyMul(otherFactors)
		if err != nil {
			return nil, 0, err
		}
		return coeff, totalPow, nil
	}
	return nil, 0, fmt.Errorf("sum error: cannot handle term %s in symbolic sum", term.String())
}

func isOneRat(n Node) bool {
	if r, ok := n.(*RationalNode); ok && r.Val.Cmp(big.NewRat(1, 1)) == 0 {
		return true
	}
	return false
}

func evalSum(expr Node, kVarNode Node, startNode Node, endNode Node) (Node, error) {
	v, ok := kVarNode.(*VarNode)
	if !ok {
		return nil, fmt.Errorf("sum error: second argument must be a variable, got %s", kVarNode.String())
	}
	kName := v.Name

	rStart, okStart := startNode.(*RationalNode)
	rEnd, okEnd := endNode.(*RationalNode)

	if okStart && okEnd && rStart.Val.IsInt() && rEnd.Val.IsInt() {
		startVal := rStart.Val.Num().Int64()
		endVal := rEnd.Val.Num().Int64()
		if startVal > endVal {
			return nil, fmt.Errorf("sum error: start value exceeds end value: %d > %d", startVal, endVal)
		}

		var sumTerms []Node
		for k := startVal; k <= endVal; k++ {
			subEnv := NewEnv()
			subEnv.Set(kName, mustRational(k, 1))
			val, err := EvalWithEnv(expr, subEnv)
			if err != nil {
				return nil, fmt.Errorf("sum error at %s=%d: %w", kName, k, err)
			}
			sumTerms = append(sumTerms, val)
		}
		if len(sumTerms) == 0 {
			return mustRational(0, 1), nil
		}
		res, err := simplifyAdd(sumTerms)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	if okStart && rStart.Val.Cmp(big.NewRat(1, 1)) == 0 {
		expanded := expandNode(expr)
		var terms []Node
		if add, ok := expanded.(*AddNode); ok {
			terms = add.Terms
		} else {
			terms = []Node{expanded}
		}

		var resultTerms []Node
		for _, t := range terms {
			coeff, pow, err := extractPowerOfK(t, kName)
			if err != nil {
				return nil, err
			}
			sNode, err := faulhaberSum(pow, endNode)
			if err != nil {
				return nil, err
			}
			termProd, err := simplifyMul([]Node{coeff, sNode})
			if err != nil {
				return nil, err
			}
			resultTerms = append(resultTerms, expandNode(termProd))
		}

		res, err := simplifyAdd(resultTerms)
		if err != nil {
			return nil, err
		}
		return expandNode(res), nil
	}

	return nil, fmt.Errorf("sum error: unsupported bounds %s to %s", startNode.String(), endNode.String())
}

// -------------------------------------------------------------------------
// 3D Vector Calculus (dot, cross, norm, grad, div, curl)
// -------------------------------------------------------------------------

func toVectorElements(n Node) ([]Node, error) {
	if n == nil {
		return nil, fmt.Errorf("vector cannot be nil")
	}
	switch v := n.(type) {
	case *ListNode:
		return v.Elements, nil
	case *MatrixNode:
		if v.Rows == 1 {
			return v.Data[0], nil
		}
		if v.Cols == 1 {
			elems := make([]Node, v.Rows)
			for r := 0; r < v.Rows; r++ {
				elems[r] = v.Data[r][0]
			}
			return elems, nil
		}
		return nil, fmt.Errorf("argument must be a 1D vector or 1xN/Nx1 matrix, got %dx%d matrix", v.Rows, v.Cols)
	default:
		return nil, fmt.Errorf("argument must be a vector (list or 1D matrix), got %s", n.String())
	}
}

func evalDot(uNode, vNode Node) (Node, error) {
	u, err := toVectorElements(uNode)
	if err != nil {
		return nil, fmt.Errorf("dot error: %w", err)
	}
	v, err := toVectorElements(vNode)
	if err != nil {
		return nil, fmt.Errorf("dot error: %w", err)
	}
	if len(u) != len(v) {
		return nil, fmt.Errorf("dot error: dimension mismatch: %d and %d", len(u), len(v))
	}
	if len(u) == 0 {
		return mustRational(0, 1), nil
	}

	var prods []Node
	for i := range u {
		p, err := simplifyMul([]Node{u[i], v[i]})
		if err != nil {
			return nil, err
		}
		prods = append(prods, p)
	}
	res, err := simplifyAdd(prods)
	if err != nil {
		return nil, err
	}
	return expandNode(res), nil
}

func evalCross(uNode, vNode Node) (*ListNode, error) {
	u, err := toVectorElements(uNode)
	if err != nil {
		return nil, fmt.Errorf("cross error: %w", err)
	}
	v, err := toVectorElements(vNode)
	if err != nil {
		return nil, fmt.Errorf("cross error: %w", err)
	}
	if len(u) != 3 || len(v) != 3 {
		return nil, fmt.Errorf("cross error: cross product requires 3-dimensional vectors, got %d and %d", len(u), len(v))
	}

	w1p1, err := simplifyMul([]Node{u[1], v[2]})
	if err != nil {
		return nil, err
	}
	w1p2, err := simplifyMul([]Node{mustRational(-1, 1), u[2], v[1]})
	if err != nil {
		return nil, err
	}
	w1, err := simplifyAdd([]Node{w1p1, w1p2})
	if err != nil {
		return nil, err
	}

	w2p1, err := simplifyMul([]Node{u[2], v[0]})
	if err != nil {
		return nil, err
	}
	w2p2, err := simplifyMul([]Node{mustRational(-1, 1), u[0], v[2]})
	if err != nil {
		return nil, err
	}
	w2, err := simplifyAdd([]Node{w2p1, w2p2})
	if err != nil {
		return nil, err
	}

	w3p1, err := simplifyMul([]Node{u[0], v[1]})
	if err != nil {
		return nil, err
	}
	w3p2, err := simplifyMul([]Node{mustRational(-1, 1), u[1], v[0]})
	if err != nil {
		return nil, err
	}
	w3, err := simplifyAdd([]Node{w3p1, w3p2})
	if err != nil {
		return nil, err
	}

	return &ListNode{Elements: []Node{expandNode(w1), expandNode(w2), expandNode(w3)}}, nil
}

func evalNorm(vNode Node) (Node, error) {
	v, err := toVectorElements(vNode)
	if err != nil {
		return nil, fmt.Errorf("norm error: %w", err)
	}
	if len(v) == 0 {
		return mustRational(0, 1), nil
	}

	var squares []Node
	for _, elem := range v {
		sq, err := simplifyMul([]Node{elem, elem})
		if err != nil {
			return nil, err
		}
		squares = append(squares, sq)
	}
	sumSq, err := simplifyAdd(squares)
	if err != nil {
		return nil, err
	}
	return simplifySqrt(sumSq)
}

func evalGrad(fNode Node, varsNode Node) (*ListNode, error) {
	vars, err := toVectorElements(varsNode)
	if err != nil {
		return nil, fmt.Errorf("grad error: %w", err)
	}
	if len(vars) == 0 {
		return nil, fmt.Errorf("grad error: coordinate variables vector cannot be empty")
	}

	results := make([]Node, len(vars))
	for i, vn := range vars {
		v, ok := vn.(*VarNode)
		if !ok {
			return nil, fmt.Errorf("grad error: element %d of coordinates must be a variable, got %s", i+1, vn.String())
		}
		d, err := differentiate(fNode, v.Name)
		if err != nil {
			return nil, fmt.Errorf("grad error in d/d%s: %w", v.Name, err)
		}
		results[i] = expandNode(d)
	}
	return &ListNode{Elements: results}, nil
}

func evalDiv(FNode Node, varsNode Node) (Node, error) {
	F, err := toVectorElements(FNode)
	if err != nil {
		return nil, fmt.Errorf("div error: %w", err)
	}
	vars, err := toVectorElements(varsNode)
	if err != nil {
		return nil, fmt.Errorf("div error: %w", err)
	}
	if len(F) != len(vars) {
		return nil, fmt.Errorf("div error: vector field and coordinates dimension mismatch: %d and %d", len(F), len(vars))
	}

	var terms []Node
	for i := range F {
		v, ok := vars[i].(*VarNode)
		if !ok {
			return nil, fmt.Errorf("div error: coordinate %d must be a variable, got %s", i+1, vars[i].String())
		}
		d, err := differentiate(F[i], v.Name)
		if err != nil {
			return nil, fmt.Errorf("div error in d/d%s: %w", v.Name, err)
		}
		terms = append(terms, d)
	}
	res, err := simplifyAdd(terms)
	if err != nil {
		return nil, err
	}
	return expandNode(res), nil
}

func evalCurl(FNode Node, varsNode Node) (*ListNode, error) {
	F, err := toVectorElements(FNode)
	if err != nil {
		return nil, fmt.Errorf("curl error: %w", err)
	}
	vars, err := toVectorElements(varsNode)
	if err != nil {
		return nil, fmt.Errorf("curl error: %w", err)
	}
	if len(F) != 3 || len(vars) != 3 {
		return nil, fmt.Errorf("curl error: curl requires 3-dimensional vectors, got %d and %d", len(F), len(vars))
	}

	vx, okX := vars[0].(*VarNode)
	vy, okY := vars[1].(*VarNode)
	vz, okZ := vars[2].(*VarNode)
	if !okX || !okY || !okZ {
		return nil, fmt.Errorf("curl error: all coordinate elements must be variables")
	}

	dF3dy, err := differentiate(F[2], vy.Name)
	if err != nil {
		return nil, err
	}
	dF2dz, err := differentiate(F[1], vz.Name)
	if err != nil {
		return nil, err
	}
	negDF2dz, err := simplifyMul([]Node{mustRational(-1, 1), dF2dz})
	if err != nil {
		return nil, err
	}
	c1, err := simplifyAdd([]Node{dF3dy, negDF2dz})
	if err != nil {
		return nil, err
	}

	dF1dz, err := differentiate(F[0], vz.Name)
	if err != nil {
		return nil, err
	}
	dF3dx, err := differentiate(F[2], vx.Name)
	if err != nil {
		return nil, err
	}
	negDF3dx, err := simplifyMul([]Node{mustRational(-1, 1), dF3dx})
	if err != nil {
		return nil, err
	}
	c2, err := simplifyAdd([]Node{dF1dz, negDF3dx})
	if err != nil {
		return nil, err
	}

	dF2dx, err := differentiate(F[1], vx.Name)
	if err != nil {
		return nil, err
	}
	dF1dy, err := differentiate(F[0], vy.Name)
	if err != nil {
		return nil, err
	}
	negDF1dy, err := simplifyMul([]Node{mustRational(-1, 1), dF1dy})
	if err != nil {
		return nil, err
	}
	c3, err := simplifyAdd([]Node{dF2dx, negDF1dy})
	if err != nil {
		return nil, err
	}

	return &ListNode{Elements: []Node{expandNode(c1), expandNode(c2), expandNode(c3)}}, nil
}

// Suppress unused imports
var _ = sort.Strings
var _ = strings.Join




