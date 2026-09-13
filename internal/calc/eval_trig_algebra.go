package calc

import (
	"fmt"
	"math/big"
	"strings"
)

// -------------------------------------------------------------------------
// Trigonometric Algebra: Expansion & Reduction Engine
// Based on Hongguang Fu et al. (2006) and Joel S. Cohen (2003)
// -------------------------------------------------------------------------

// EvalTrigExpand recursively expands trigonometric functions using angle addition
// and multiple-angle formulas (trig_expand).
func EvalTrigExpand(expr Node, env *Env) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot expand nil expression")
	}

	expanded := expandTrigNode(expr, env)
	// Distribute polynomial products and simplify
	res := expandNode(expanded)
	res, err := EvalWithEnv(res, env)
	if err != nil {
		return nil, err
	}

	RecordTraceRewrite(RuleTrigExpand, expr, res, "三角関数の加法定理・多倍角展開")
	return res, nil
}

// EvalTrigReduce recursively reduces powers and products of trigonometric functions
// to linear sums (product-to-sum, half-angle formulas) and performs harmonic addition.
func EvalTrigReduce(expr Node, env *Env) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot reduce nil expression")
	}

	// 1. First pass: expand products of expressions if necessary to expose trig products
	curr := expandNode(expr)

	// 2. Reduce powers of sin and cos: sin^2(x) -> (1 - cos(2x))/2, etc.
	curr = reduceTrigPowers(curr, env)

	// 3. Product-to-sum reduction: sin(A)*cos(B) -> 1/2*(sin(A+B)+sin(A-B)), etc.
	curr = reduceTrigProducts(curr, env)

	// 4. Harmonic addition: a*sin(x) + b*cos(x) -> R*sin(x + alpha)
	curr = reduceTrigHarmonic(curr)

	// 5. Final algebraic simplification
	res, err := EvalWithEnv(curr, env)
	if err != nil {
		return nil, err
	}

	RecordTraceRewrite(RuleTrigReduce, expr, res, "積和公式・次数下げ・三角関数の合成")
	return res, nil
}

// -------------------------------------------------------------------------
// 1. Trig Expansion (Addition & Multiple Angle Formulas)
// -------------------------------------------------------------------------

func expandTrigNode(n Node, env *Env) Node {
	if n == nil {
		return nil
	}

	switch v := n.(type) {
	case *RationalNode, *ConstNode, *VarNode:
		return v

	case *UnaryOpNode:
		inner := expandTrigNode(v.Expr, env)
		return &UnaryOpNode{Op: v.Op, Expr: inner}

	case *AddNode:
		var terms []Node
		for _, t := range v.Terms {
			terms = append(terms, expandTrigNode(t, env))
		}
		return &AddNode{Terms: terms}

	case *MulNode:
		var factors []Node
		for _, f := range v.Factors {
			factors = append(factors, expandTrigNode(f, env))
		}
		return &MulNode{Factors: factors}

	case *PowNode:
		return &PowNode{
			Base: expandTrigNode(v.Base, env),
			Exp:  expandTrigNode(v.Exp, env),
		}

	case *FuncNode:
		if len(v.Args) == 1 && (v.Name == "sin" || v.Name == "cos" || v.Name == "tan") {
			arg := expandTrigNode(v.Args[0], env)
			return expandTrigFunction(v.Name, arg, env)
		}
		var newArgs []Node
		for _, a := range v.Args {
			newArgs = append(newArgs, expandTrigNode(a, env))
		}
		return &FuncNode{Name: v.Name, Args: newArgs}

	default:
		return n
	}
}

// expandTrigFunction expands sin(arg), cos(arg), or tan(arg).
func expandTrigFunction(fn string, arg Node, env *Env) Node {
	// Check for negative argument: sin(-X) = -sin(X), cos(-X) = cos(X), tan(-X) = -tan(X)
	if isNeg, inner := extractNegativeArg(arg); isNeg {
		switch fn {
		case "sin":
			return expandNode(&MulNode{Factors: []Node{mustRational(-1, 1), expandTrigFunction("sin", inner, env)}})
		case "cos":
			return expandTrigFunction("cos", inner, env)
		case "tan":
			return expandNode(&MulNode{Factors: []Node{mustRational(-1, 1), expandTrigFunction("tan", inner, env)}})
		}
	}

	// 1. Check for addition/subtraction: AddNode [A, B, C, ...]
	if add, ok := arg.(*AddNode); ok && len(add.Terms) >= 2 {
		return expandTrigAddition(fn, add.Terms, env)
	}

	// 2. Check for integer multiple: k * X (where k is integer >= 2)
	if k, base, ok := extractIntMultiple(arg); ok && k >= 2 {
		return expandMultipleAngle(fn, k, base, env)
	}

	return &FuncNode{Name: fn, Args: []Node{arg}}
}

// extractNegativeArg returns true and the positive inner expression if expr has a leading negative coefficient.
func extractNegativeArg(expr Node) (bool, Node) {
	if un, ok := expr.(*UnaryOpNode); ok && un.Op == "-" {
		return true, un.Expr
	}
	if mul, ok := expr.(*MulNode); ok && len(mul.Factors) > 0 {
		if r, okRat := mul.Factors[0].(*RationalNode); okRat && r.Val.Sign() < 0 {
			posR := new(big.Rat).Abs(r.Val)
			var rest []Node
			if posR.Cmp(big.NewRat(1, 1)) != 0 || len(mul.Factors) == 1 {
				rest = append(rest, NewRationalFromBigRat(posR))
			}
			rest = append(rest, mul.Factors[1:]...)
			if len(rest) == 1 {
				return true, rest[0]
			}
			return true, &MulNode{Factors: rest}
		}
	}
	return false, nil
}

// expandTrigAddition expands sin/cos/tan of sum of terms: A + B + ...
func expandTrigAddition(fn string, terms []Node, env *Env) Node {
	if len(terms) == 1 {
		return expandTrigFunction(fn, terms[0], env)
	}
	// Split into first term A and rest terms B = terms[1] + ...
	A := terms[0]
	var B Node
	if len(terms) == 2 {
		B = terms[1]
	} else {
		B = &AddNode{Terms: terms[1:]}
	}

	sinA := expandTrigFunction("sin", A, env)
	cosA := expandTrigFunction("cos", A, env)
	sinB := expandTrigFunction("sin", B, env)
	cosB := expandTrigFunction("cos", B, env)

	switch fn {
	case "sin":
		// sin(A + B) = sin(A)*cos(B) + cos(A)*sin(B)
		term1 := &MulNode{Factors: []Node{sinA, cosB}}
		term2 := &MulNode{Factors: []Node{cosA, sinB}}
		return expandNode(&AddNode{Terms: []Node{term1, term2}})

	case "cos":
		// cos(A + B) = cos(A)*cos(B) - sin(A)*sin(B)
		term1 := &MulNode{Factors: []Node{cosA, cosB}}
		term2 := &MulNode{Factors: []Node{mustRational(-1, 1), sinA, sinB}}
		return expandNode(&AddNode{Terms: []Node{term1, term2}})

	case "tan":
		// tan(A + B) = (tan(A) + tan(B)) / (1 - tan(A)*tan(B))
		tanA := expandTrigFunction("tan", A, env)
		tanB := expandTrigFunction("tan", B, env)
		numer := &AddNode{Terms: []Node{tanA, tanB}}
		denom := &AddNode{Terms: []Node{
			mustRational(1, 1),
			&MulNode{Factors: []Node{mustRational(-1, 1), tanA, tanB}},
		}}
		return &MulNode{
			Factors: []Node{
				numer,
				&PowNode{Base: denom, Exp: mustRational(-1, 1)},
			},
		}
	}

	return &FuncNode{Name: fn, Args: []Node{&AddNode{Terms: terms}}}
}

// extractIntMultiple checks if node is integer k * base (k >= 2).
func extractIntMultiple(n Node) (int64, Node, bool) {
	mul, ok := n.(*MulNode)
	if !ok || len(mul.Factors) < 2 {
		return 0, nil, false
	}
	r, okRat := mul.Factors[0].(*RationalNode)
	if !okRat || !r.Val.IsInt() || r.Val.Sign() <= 0 {
		return 0, nil, false
	}
	k := r.Val.Num().Int64()
	if k < 2 {
		return 0, nil, false
	}

	var rest []Node
	rest = append(rest, mul.Factors[1:]...)
	if len(rest) == 1 {
		return k, rest[0], true
	}
	return k, &MulNode{Factors: rest}, true
}

// expandMultipleAngle expands sin(k*X), cos(k*X), tan(k*X) for integer k >= 2.
func expandMultipleAngle(fn string, k int64, base Node, env *Env) Node {
	if k == 2 {
		sinX := expandTrigFunction("sin", base, env)
		cosX := expandTrigFunction("cos", base, env)
		switch fn {
		case "sin":
			// sin(2X) = 2*sin(X)*cos(X)
			return &MulNode{Factors: []Node{mustRational(2, 1), sinX, cosX}}
		case "cos":
			// cos(2X) = cos(X)^2 - sin(X)^2
			cos2 := &PowNode{Base: cosX, Exp: mustRational(2, 1)}
			sin2 := &MulNode{Factors: []Node{mustRational(-1, 1), &PowNode{Base: sinX, Exp: mustRational(2, 1)}}}
			return expandNode(&AddNode{Terms: []Node{cos2, sin2}})
		case "tan":
			// tan(2X) = 2*tan(X) / (1 - tan(X)^2)
			tanX := expandTrigFunction("tan", base, env)
			numer := &MulNode{Factors: []Node{mustRational(2, 1), tanX}}
			denom := &AddNode{Terms: []Node{
				mustRational(1, 1),
				&MulNode{Factors: []Node{mustRational(-1, 1), &PowNode{Base: tanX, Exp: mustRational(2, 1)}}},
			}}
			return &MulNode{Factors: []Node{numer, &PowNode{Base: denom, Exp: mustRational(-1, 1)}}}
		}
	}

	if k == 3 {
		sinX := expandTrigFunction("sin", base, env)
		cosX := expandTrigFunction("cos", base, env)
		switch fn {
		case "sin":
			// sin(3X) = 3*cos(X)^2*sin(X) - sin(X)^3
			cos2sin := &MulNode{Factors: []Node{mustRational(3, 1), &PowNode{Base: cosX, Exp: mustRational(2, 1)}, sinX}}
			sin3 := &MulNode{Factors: []Node{mustRational(-1, 1), &PowNode{Base: sinX, Exp: mustRational(3, 1)}}}
			return expandNode(&AddNode{Terms: []Node{cos2sin, sin3}})
		case "cos":
			// cos(3X) = cos(X)^3 - 3*cos(X)*sin(X)^2
			cos3 := &PowNode{Base: cosX, Exp: mustRational(3, 1)}
			cossin2 := &MulNode{Factors: []Node{mustRational(-3, 1), cosX, &PowNode{Base: sinX, Exp: mustRational(2, 1)}}}
			return expandNode(&AddNode{Terms: []Node{cos3, cossin2}})
		case "tan":
			// tan(3X) = (3*tan(X) - tan(X)^3) / (1 - 3*tan(X)^2)
			tanX := expandTrigFunction("tan", base, env)
			numer := &AddNode{Terms: []Node{
				&MulNode{Factors: []Node{mustRational(3, 1), tanX}},
				&MulNode{Factors: []Node{mustRational(-1, 1), &PowNode{Base: tanX, Exp: mustRational(3, 1)}}},
			}}
			denom := &AddNode{Terms: []Node{
				mustRational(1, 1),
				&MulNode{Factors: []Node{mustRational(-3, 1), &PowNode{Base: tanX, Exp: mustRational(2, 1)}}},
			}}
			return &MulNode{Factors: []Node{numer, &PowNode{Base: denom, Exp: mustRational(-1, 1)}}}
		}
	}

	// For k >= 4, use recursive addition: k*X = (k-1)*X + X
	km1Node := &MulNode{Factors: []Node{mustRational(k-1, 1), base}}
	return expandTrigAddition(fn, []Node{km1Node, base}, env)
}

// -------------------------------------------------------------------------
// 2. Trig Reduction (Powers, Products, and Harmonic Addition)
// -------------------------------------------------------------------------

// reduceTrigPowers reduces powers of sin(x)^n and cos(x)^n using half-angle formulas.
func reduceTrigPowers(n Node, env *Env) Node {
	if n == nil {
		return nil
	}

	switch v := n.(type) {
	case *PowNode:
		base := reduceTrigPowers(v.Base, env)
		exp := reduceTrigPowers(v.Exp, env)

		rExp, okExp := exp.(*RationalNode)
		fn, okFn := base.(*FuncNode)
		if okExp && rExp.Val.IsInt() && rExp.Val.Sign() > 0 && okFn && len(fn.Args) == 1 {
			k := rExp.Val.Num().Int64()
			arg := fn.Args[0]
			twoArg := &MulNode{Factors: []Node{mustRational(2, 1), arg}}

			switch fn.Name {
			case "sin":
				if k == 2 {
					// sin^2(x) = 1/2 - 1/2*cos(2x)
					half := mustRational(1, 2)
					mHalf := mustRational(-1, 2)
					cos2 := &FuncNode{Name: "cos", Args: []Node{twoArg}}
					return &AddNode{Terms: []Node{half, &MulNode{Factors: []Node{mHalf, cos2}}}}
				}
				if k > 2 {
					// sin^k(x) = sin^2(x) * sin^(k-2)(x)
					sin2 := reduceTrigPowers(&PowNode{Base: base, Exp: mustRational(2, 1)}, env)
					sinRest := reduceTrigPowers(&PowNode{Base: base, Exp: mustRational(k-2, 1)}, env)
					prod := expandNode(&MulNode{Factors: []Node{sin2, sinRest}})
					return reduceTrigProducts(prod, env)
				}
			case "cos":
				if k == 2 {
					// cos^2(x) = 1/2 + 1/2*cos(2x)
					half := mustRational(1, 2)
					cos2 := &FuncNode{Name: "cos", Args: []Node{twoArg}}
					return &AddNode{Terms: []Node{half, &MulNode{Factors: []Node{half, cos2}}}}
				}
				if k > 2 {
					// cos^k(x) = cos^2(x) * cos^(k-2)(x)
					cos2 := reduceTrigPowers(&PowNode{Base: base, Exp: mustRational(2, 1)}, env)
					cosRest := reduceTrigPowers(&PowNode{Base: base, Exp: mustRational(k-2, 1)}, env)
					prod := expandNode(&MulNode{Factors: []Node{cos2, cosRest}})
					return reduceTrigProducts(prod, env)
				}
			}
		}
		return &PowNode{Base: base, Exp: exp}

	case *AddNode:
		var terms []Node
		for _, t := range v.Terms {
			terms = append(terms, reduceTrigPowers(t, env))
		}
		return &AddNode{Terms: terms}

	case *MulNode:
		var factors []Node
		for _, f := range v.Factors {
			factors = append(factors, reduceTrigPowers(f, env))
		}
		return &MulNode{Factors: factors}

	case *UnaryOpNode:
		return &UnaryOpNode{Op: v.Op, Expr: reduceTrigPowers(v.Expr, env)}

	default:
		return n
	}
}

// reduceTrigProducts converts products of sin and cos into sums using product-to-sum formulas.
func reduceTrigProducts(n Node, env *Env) Node {
	if n == nil {
		return nil
	}

	switch v := n.(type) {
	case *AddNode:
		var terms []Node
		for _, t := range v.Terms {
			terms = append(terms, reduceTrigProducts(t, env))
		}
		return &AddNode{Terms: terms}

	case *MulNode:
		// Separate trigonometric factors from other factors
		var trigFactors []*FuncNode
		var otherFactors []Node

		for _, f := range v.Factors {
			if fn, ok := f.(*FuncNode); ok && (fn.Name == "sin" || fn.Name == "cos") && len(fn.Args) == 1 {
				trigFactors = append(trigFactors, fn)
			} else {
				otherFactors = append(otherFactors, reduceTrigProducts(f, env))
			}
		}

		if len(trigFactors) >= 2 {
			// Take first two trig factors and combine via product-to-sum
			f1 := trigFactors[0]
			f2 := trigFactors[1]
			combined := productToSum(f1, f2)

			var remFactors []Node
			remFactors = append(remFactors, otherFactors...)
			for _, rf := range trigFactors[2:] {
				remFactors = append(remFactors, rf)
			}
			remFactors = append(remFactors, combined)

			prod := expandNode(&MulNode{Factors: remFactors})
			return reduceTrigProducts(prod, env)
		}

		var newFactors []Node
		newFactors = append(newFactors, otherFactors...)
		for _, tf := range trigFactors {
			newFactors = append(newFactors, tf)
		}
		if len(newFactors) == 1 {
			return newFactors[0]
		}
		return &MulNode{Factors: newFactors}

	case *UnaryOpNode:
		return &UnaryOpNode{Op: v.Op, Expr: reduceTrigProducts(v.Expr, env)}

	default:
		return n
	}
}

// productToSum converts f1 * f2 into a sum.
func productToSum(f1, f2 *FuncNode) Node {
	A := f1.Args[0]
	B := f2.Args[0]

	// Check if A and B are identical: A == B
	if A.String() == B.String() {
		twoA := &MulNode{Factors: []Node{mustRational(2, 1), A}}
		if f1.Name == "sin" && f2.Name == "cos" || f1.Name == "cos" && f2.Name == "sin" {
			// sin(A)*cos(A) = 1/2 * sin(2A)
			return &MulNode{Factors: []Node{mustRational(1, 2), &FuncNode{Name: "sin", Args: []Node{twoA}}}}
		}
		if f1.Name == "sin" && f2.Name == "sin" {
			// sin(A)^2 = 1/2 - 1/2*cos(2A)
			return &AddNode{Terms: []Node{
				mustRational(1, 2),
				&MulNode{Factors: []Node{mustRational(-1, 2), &FuncNode{Name: "cos", Args: []Node{twoA}}}},
			}}
		}
		if f1.Name == "cos" && f2.Name == "cos" {
			// cos(A)^2 = 1/2 + 1/2*cos(2A)
			return &AddNode{Terms: []Node{
				mustRational(1, 2),
				&MulNode{Factors: []Node{mustRational(1, 2), &FuncNode{Name: "cos", Args: []Node{twoA}}}},
			}}
		}
	}

	// General product-to-sum:
	// A+B and A-B
	ApB := &AddNode{Terms: []Node{A, B}}
	AmB := &AddNode{Terms: []Node{A, &MulNode{Factors: []Node{mustRational(-1, 1), B}}}}

	half := mustRational(1, 2)
	mHalf := mustRational(-1, 2)

	if f1.Name == "sin" && f2.Name == "cos" {
		// sin(A)*cos(B) = 1/2 * (sin(A+B) + sin(A-B))
		s1 := &FuncNode{Name: "sin", Args: []Node{ApB}}
		s2 := &FuncNode{Name: "sin", Args: []Node{AmB}}
		return &AddNode{Terms: []Node{
			&MulNode{Factors: []Node{half, s1}},
			&MulNode{Factors: []Node{half, s2}},
		}}
	} else if f1.Name == "cos" && f2.Name == "sin" {
		// cos(A)*sin(B) = 1/2 * (sin(A+B) - sin(A-B))
		s1 := &FuncNode{Name: "sin", Args: []Node{ApB}}
		s2 := &FuncNode{Name: "sin", Args: []Node{AmB}}
		return &AddNode{Terms: []Node{
			&MulNode{Factors: []Node{half, s1}},
			&MulNode{Factors: []Node{mHalf, s2}},
		}}
	} else if f1.Name == "cos" && f2.Name == "cos" {
		// cos(A)*cos(B) = 1/2 * (cos(A+B) + cos(A-B))
		c1 := &FuncNode{Name: "cos", Args: []Node{ApB}}
		c2 := &FuncNode{Name: "cos", Args: []Node{AmB}}
		return &AddNode{Terms: []Node{
			&MulNode{Factors: []Node{half, c1}},
			&MulNode{Factors: []Node{half, c2}},
		}}
	} else if f1.Name == "sin" && f2.Name == "sin" {
		// sin(A)*sin(B) = 1/2 * (cos(A-B) - cos(A+B))
		c1 := &FuncNode{Name: "cos", Args: []Node{AmB}}
		c2 := &FuncNode{Name: "cos", Args: []Node{ApB}}
		return &AddNode{Terms: []Node{
			&MulNode{Factors: []Node{half, c1}},
			&MulNode{Factors: []Node{mHalf, c2}},
		}}
	}

	return &MulNode{Factors: []Node{f1, f2}}
}

// -------------------------------------------------------------------------
// 3. Harmonic Addition: a*sin(x) + b*cos(x) -> R*sin(x + alpha)
// -------------------------------------------------------------------------

type trigTermInfo struct {
	fn    string // "sin" or "cos"
	arg   Node
	coeff Node
}

func reduceTrigHarmonic(n Node) Node {
	add, ok := n.(*AddNode)
	if !ok {
		return n
	}

	// Inspect each term in AddNode
	var trigTerms []trigTermInfo
	var otherTerms []Node

	for _, t := range add.Terms {
		if info, ok := matchTrigTerm(t); ok {
			trigTerms = append(trigTerms, info)
		} else {
			otherTerms = append(otherTerms, t)
		}
	}

	if len(trigTerms) < 2 {
		return n
	}

	// Look for pairs with identical argument: one sin and one cos
	used := make([]bool, len(trigTerms))
	var combinedTerms []Node

	for i := 0; i < len(trigTerms); i++ {
		if used[i] {
			continue
		}
		t1 := trigTerms[i]
		paired := false
		for j := i + 1; j < len(trigTerms); j++ {
			if used[j] {
				continue
			}
			t2 := trigTerms[j]
			if t1.arg.String() == t2.arg.String() && ((t1.fn == "sin" && t2.fn == "cos") || (t1.fn == "cos" && t2.fn == "sin")) {
				var a, b Node
				if t1.fn == "sin" {
					a, b = t1.coeff, t2.coeff
				} else {
					a, b = t2.coeff, t1.coeff
				}

				if R, alpha, ok := matchSpecialHarmonic(a, b); ok {
					// R * sin(x + alpha)
					var angle Node
					if alpha == nil {
						angle = t1.arg
					} else {
						angle = &AddNode{Terms: []Node{t1.arg, alpha}}
					}
					sinTerm := &FuncNode{Name: "sin", Args: []Node{angle}}
					var combined Node
					if R == nil {
						combined = sinTerm
					} else {
						combined = &MulNode{Factors: []Node{R, sinTerm}}
					}
					combinedTerms = append(combinedTerms, combined)
					used[i] = true
					used[j] = true
					paired = true
					break
				}
			}
		}
		if !paired {
			// Put back unchanged
			combinedTerms = append(combinedTerms, reconstructTrigTerm(t1))
			used[i] = true
		}
	}

	var allTerms []Node
	allTerms = append(allTerms, otherTerms...)
	allTerms = append(allTerms, combinedTerms...)

	if len(allTerms) == 1 {
		return allTerms[0]
	}
	return &AddNode{Terms: allTerms}
}

func matchTrigTerm(t Node) (trigTermInfo, bool) {
	// sin(x) or cos(x)
	if fn, ok := t.(*FuncNode); ok && (fn.Name == "sin" || fn.Name == "cos") && len(fn.Args) == 1 {
		return trigTermInfo{fn: fn.Name, arg: fn.Args[0], coeff: mustRational(1, 1)}, true
	}
	// -sin(x) or -cos(x)
	if un, ok := t.(*UnaryOpNode); ok && un.Op == "-" {
		if fn, okFn := un.Expr.(*FuncNode); okFn && (fn.Name == "sin" || fn.Name == "cos") && len(fn.Args) == 1 {
			return trigTermInfo{fn: fn.Name, arg: fn.Args[0], coeff: mustRational(-1, 1)}, true
		}
	}
	// c * sin(x) or c * cos(x)
	if mul, ok := t.(*MulNode); ok && len(mul.Factors) >= 2 {
		var trigPart *FuncNode
		var coeffFactors []Node
		for _, f := range mul.Factors {
			if fn, okFn := f.(*FuncNode); okFn && (fn.Name == "sin" || fn.Name == "cos") && len(fn.Args) == 1 && trigPart == nil {
				trigPart = fn
			} else {
				coeffFactors = append(coeffFactors, f)
			}
		}
		if trigPart != nil {
			var coeff Node
			if len(coeffFactors) == 1 {
				coeff = coeffFactors[0]
			} else {
				coeff, _ = simplifyMul(coeffFactors)
			}
			if evaledCoeff, err := Eval(coeff); err == nil {
				coeff = evaledCoeff
			}
			return trigTermInfo{fn: trigPart.Name, arg: trigPart.Args[0], coeff: coeff}, true
		}
	}
	return trigTermInfo{}, false
}

func reconstructTrigTerm(info trigTermInfo) Node {
	fn := &FuncNode{Name: info.fn, Args: []Node{info.arg}}
	if r, ok := info.coeff.(*RationalNode); ok && r.Val.Cmp(big.NewRat(1, 1)) == 0 {
		return fn
	}
	return &MulNode{Factors: []Node{info.coeff, fn}}
}

// matchSpecialHarmonic finds exact R and alpha for a*sin(x) + b*cos(x).
func normalizeRadicalStr(s string) string {
	s = strings.ReplaceAll(s, "sqrt(3)", "√3")
	s = strings.ReplaceAll(s, "3^(1/2)", "√3")
	s = strings.ReplaceAll(s, "sqrt(2)", "√2")
	s = strings.ReplaceAll(s, "2^(1/2)", "√2")
	s = strings.ReplaceAll(s, "-1*", "-")
	return s
}

// matchSpecialHarmonic finds exact R and alpha for a*sin(x) + b*cos(x).
// R*sin(x + alpha) where R = sqrt(a^2 + b^2), tan(alpha) = b / a.
func matchSpecialHarmonic(a, b Node) (R Node, alpha Node, ok bool) {
	aStr := normalizeRadicalStr(Format(a))
	bStr := normalizeRadicalStr(Format(b))

	piNode := &ConstNode{Name: "pi"}
	sqrt2 := &SqrtNode{Radicand: mustRational(2, 1)}
	sqrt3 := &SqrtNode{Radicand: mustRational(3, 1)}

	// Table of well-known special angles
	// (a, b) -> (R, alpha)
	type specialCase struct {
		aStr, bStr string
		R          Node
		alpha      Node
	}

	cases := []specialCase{
		// 1 : 1 ratio (pi/4 family)
		{"1", "1", sqrt2, &MulNode{Factors: []Node{mustRational(1, 4), piNode}}},
		{"1", "-1", sqrt2, &MulNode{Factors: []Node{mustRational(-1, 4), piNode}}},
		{"-1", "1", sqrt2, &MulNode{Factors: []Node{mustRational(3, 4), piNode}}},
		{"-1", "-1", sqrt2, &MulNode{Factors: []Node{mustRational(-3, 4), piNode}}},

		// √3 : 1 ratio (pi/6 family: a=√3, b=1 -> alpha=pi/6, R=2)
		{"√3", "1", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(1, 6), piNode}}},
		{"√3", "-1", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(-1, 6), piNode}}},
		{"-√3", "1", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(5, 6), piNode}}},
		{"-√3", "-1", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(-5, 6), piNode}}},

		// 1 : √3 ratio (pi/3 family: a=1, b=√3 -> alpha=pi/3, R=2)
		{"1", "√3", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(1, 3), piNode}}},
		{"1", "-√3", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(-1, 3), piNode}}},
		{"-1", "√3", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(2, 3), piNode}}},
		{"-1", "-√3", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(-2, 3), piNode}}},

		// Scaled ratios
		{"√2", "√2", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(1, 4), piNode}}},
		{"√2", "-√2", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(-1, 4), piNode}}},
		{"-√2", "√2", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(3, 4), piNode}}},
		{"-√2", "-√2", mustRational(2, 1), &MulNode{Factors: []Node{mustRational(-3, 4), piNode}}},

		// Scaled √3: √3*sin(x) + 3*cos(x) -> 2*√3*sin(x + pi/3)
		{"√3", "3", &MulNode{Factors: []Node{mustRational(2, 1), sqrt3}}, &MulNode{Factors: []Node{mustRational(1, 3), piNode}}},
		{"3", "√3", &MulNode{Factors: []Node{mustRational(2, 1), sqrt3}}, &MulNode{Factors: []Node{mustRational(1, 6), piNode}}},
	}

	for _, c := range cases {
		if aStr == c.aStr && bStr == c.bStr {
			return c.R, c.alpha, true
		}
	}

	return nil, nil, false
}
