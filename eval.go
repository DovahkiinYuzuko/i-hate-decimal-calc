package main

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// mustRational creates a RationalNode directly for guaranteed non-zero denominators.
func mustRational(num, denom int64) *RationalNode {
	return &RationalNode{Val: big.NewRat(num, denom)}
}

// Eval evaluates and simplifies an AST node into its canonical exact form.
func Eval(n Node) (Node, error) {
	if n == nil {
		return nil, fmt.Errorf("cannot evaluate nil node")
	}

	switch v := n.(type) {
	case *RationalNode, *ConstNode:
		return v, nil

	case *ComplexNode:
		r, err := Eval(v.Real)
		if err != nil {
			return nil, err
		}
		im, err := Eval(v.Imag)
		if err != nil {
			return nil, err
		}
		return NewComplex(r, im), nil

	case *SqrtNode:
		rad, err := Eval(v.Radicand)
		if err != nil {
			return nil, err
		}
		return simplifySqrt(rad)

	case *FuncNode:
		evaledArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			ea, err := Eval(a)
			if err != nil {
				return nil, err
			}
			evaledArgs[i] = ea
		}
		return simplifyFunc(v.Name, evaledArgs)

	case *UnaryOpNode:
		expr, err := Eval(v.Expr)
		if err != nil {
			return nil, err
		}
		return simplifyUnaryOp(v.Op, expr)

	case *PowNode:
		base, err := Eval(v.Base)
		if err != nil {
			return nil, err
		}
		exp, err := Eval(v.Exp)
		if err != nil {
			return nil, err
		}
		return simplifyPow(base, exp)

	case *AddNode:
		evaledTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			et, err := Eval(t)
			if err != nil {
				return nil, err
			}
			evaledTerms[i] = et
		}
		return simplifyAdd(evaledTerms)

	case *MulNode:
		evaledFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			ef, err := Eval(f)
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

// EvalString parses and evaluates a math expression string.
func EvalString(input string) (Node, error) {
	node, err := Parse(input)
	if err != nil {
		return nil, err
	}
	return Eval(node)
}

// -------------------------------------------------------------------------
// Sqrt Simplification (Square-free reduction & Complex promotion)
// -------------------------------------------------------------------------

func simplifySqrt(radicand Node) (Node, error) {
	rat, ok := radicand.(*RationalNode)
	if !ok {
		// Non-rational radicand: keep as sqrt(expr)
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

	return NewPow(base, exp)
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
				realPart, _ := simplifyAdd([]Node{ac, &UnaryOpNode{Op: "-", Expr: bd}})

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
	for _, f := range otherFactors {
		if pow, ok := f.(*PowNode); ok {
			if rExp, ok := pow.Exp.(*RationalNode); ok && rExp.Val.Cmp(big.NewRat(-1, 1)) == 0 {
				if s, ok := pow.Base.(*SqrtNode); ok {
					if rRad, ok := s.Radicand.(*RationalNode); ok && rRad.Val.IsInt() && rRad.Val.Sign() > 0 {
						// Rationalize! Divide coeff by d, and multiply numerator by sqrt(d)
						d := rRad.Val.Num()
						coeff.Quo(coeff, new(big.Rat).SetInt(d))
						finalFactors = append(finalFactors, s)
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
							continue
						}
					}
				}
			}
		}
		finalFactors = append(finalFactors, f)
	}

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
