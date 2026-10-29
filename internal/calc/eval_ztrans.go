package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// makeDiv creates num / den as num * den^(-1).
func makeDiv(num, den Node) Node {
	negOne := &RationalNode{Val: big.NewRat(-1, 1)}
	return &MulNode{Factors: []Node{num, &PowNode{Base: den, Exp: negOne}}}
}

// EvalZTrans computes the unilateral symbolic Z-transform of f(n).
// X(z) = Z{f[n]} = sum_{n=0}^infty f[n] * z^(-n).
func EvalZTrans(f Node, nName, zName string, env *Env) (Node, error) {
	if f == nil {
		return nil, fmt.Errorf("%s", i18n.T("err_cannot_evaluate_nil_node"))
	}
	if nName == "" {
		nName = "n"
	}
	if zName == "" {
		zName = "z"
	}
	zVar := &VarNode{Name: zName}

	evaledF, err := Eval(f)
	if err == nil && evaledF != nil {
		f = evaledF
	}

	res, err := ztransDispatch(f, nName, zVar, env)
	if err != nil {
		return nil, err
	}
	return Eval(res)
}

func ztransDispatch(f Node, nName string, zVar *VarNode, env *Env) (Node, error) {
	// Linearity: Z{f1 + f2} = Z{f1} + Z{f2}
	if add, ok := f.(*AddNode); ok {
		var terms []Node
		for _, t := range add.Terms {
			zt, err := ztransDispatch(t, nName, zVar, env)
			if err != nil {
				return nil, err
			}
			terms = append(terms, zt)
		}
		return &AddNode{Terms: terms}, nil
	}

	// Constant with respect to n: Z{C} = C * z / (z - 1)
	if !containsVariable(f, nName) {
		oneVal := big.NewRat(1, 1)
		zMinus1 := &AddNode{Terms: []Node{zVar, &RationalNode{Val: big.NewRat(-1, 1)}}}
		stepZ := makeDiv(zVar, zMinus1)
		if r, ok := f.(*RationalNode); ok && r.Val.Sign() == 0 {
			return &RationalNode{Val: big.NewRat(0, 1)}, nil
		}
		if r, ok := f.(*RationalNode); ok && r.Val.Cmp(oneVal) == 0 {
			return stepZ, nil
		}
		return &MulNode{Factors: []Node{f, stepZ}}, nil
	}

	// Handle Multiplication: extract constants
	if mul, ok := f.(*MulNode); ok {
		var constFactors []Node
		var nFactors []Node
		for _, fac := range mul.Factors {
			if containsVariable(fac, nName) {
				nFactors = append(nFactors, fac)
			} else {
				constFactors = append(constFactors, fac)
			}
		}

		if len(constFactors) > 0 {
			var nPart Node
			if len(nFactors) == 0 {
				nPart = &RationalNode{Val: big.NewRat(1, 1)}
			} else if len(nFactors) == 1 {
				nPart = nFactors[0]
			} else {
				nPart = &MulNode{Factors: nFactors}
			}

			zPart, err := ztransDispatch(nPart, nName, zVar, env)
			if err != nil {
				return nil, err
			}
			allFactors := append(constFactors, zPart)
			return &MulNode{Factors: allFactors}, nil
		}
	}

	// Unary minus: Z{-g} = -Z{g}
	if u, ok := f.(*UnaryOpNode); ok && u.Op == "-" {
		zg, err := ztransDispatch(u.Expr, nName, zVar, env)
		if err != nil {
			return nil, err
		}
		return &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, zg}}, nil
	}

	// Single term table lookup / pattern matching
	return ztransTerm(f, nName, zVar, env)
}

func ztransTerm(f Node, nName string, zVar *VarNode, _ *Env) (Node, error) {
	one := &RationalNode{Val: big.NewRat(1, 1)}
	negOne := &RationalNode{Val: big.NewRat(-1, 1)}

	// 1. Variable itself: n -> z / (z - 1)^2
	if v, ok := f.(*VarNode); ok && v.Name == nName {
		zMinus1 := &AddNode{Terms: []Node{zVar, negOne}}
		denom := &PowNode{Base: zMinus1, Exp: &RationalNode{Val: big.NewRat(2, 1)}}
		return makeDiv(zVar, denom), nil
	}

	// 2. Kronecker impulse: delta(n) or dirac(n) -> 1
	if fn, ok := f.(*FuncNode); ok && (fn.Name == "delta" || fn.Name == "dirac") && len(fn.Args) == 1 {
		if v, isV := fn.Args[0].(*VarNode); isV && v.Name == nName {
			return one, nil
		}
		// delta(n - k) -> z^(-k)
		shift, err := parseShiftArg(fn.Args[0], nName)
		if err == nil {
			exp := &RationalNode{Val: big.NewRat(int64(shift), 1)}
			return &PowNode{Base: zVar, Exp: exp}, nil
		}
	}

	// 3. Step function: heaviside(n) or step(n) -> z / (z - 1)
	if fn, ok := f.(*FuncNode); ok && (fn.Name == "heaviside" || fn.Name == "step" || fn.Name == "u") && len(fn.Args) == 1 {
		if containsVariable(fn.Args[0], nName) {
			zMinus1 := &AddNode{Terms: []Node{zVar, negOne}}
			return makeDiv(zVar, zMinus1), nil
		}
	}

	// 4. Power of n: n^k
	if pow, ok := f.(*PowNode); ok {
		if v, isV := pow.Base.(*VarNode); isV && v.Name == nName {
			if r, isR := pow.Exp.(*RationalNode); isR && r.Val.IsInt() {
				k := r.Val.Num().Int64()
				if k == 1 {
					zMinus1 := &AddNode{Terms: []Node{zVar, negOne}}
					return makeDiv(zVar, &PowNode{Base: zMinus1, Exp: &RationalNode{Val: big.NewRat(2, 1)}}), nil
				}
				if k == 2 {
					// n^2 -> z*(z + 1) / (z - 1)^3
					num := &MulNode{Factors: []Node{zVar, &AddNode{Terms: []Node{zVar, one}}}}
					den := &PowNode{Base: &AddNode{Terms: []Node{zVar, negOne}}, Exp: &RationalNode{Val: big.NewRat(3, 1)}}
					return makeDiv(num, den), nil
				}
				if k == 3 {
					// n^3 -> z*(z^2 + 4z + 1) / (z - 1)^4
					z2 := &PowNode{Base: zVar, Exp: &RationalNode{Val: big.NewRat(2, 1)}}
					fourZ := &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(4, 1)}, zVar}}
					num := &MulNode{Factors: []Node{zVar, &AddNode{Terms: []Node{z2, fourZ, one}}}}
					den := &PowNode{Base: &AddNode{Terms: []Node{zVar, negOne}}, Exp: &RationalNode{Val: big.NewRat(4, 1)}}
					return makeDiv(num, den), nil
				}
			}
		}

		// Geometric sequence: a^n -> z / (z - a)
		if v, isV := pow.Exp.(*VarNode); isV && v.Name == nName {
			if !containsVariable(pow.Base, nName) {
				a := pow.Base
				zMinusA := &AddNode{Terms: []Node{zVar, &MulNode{Factors: []Node{negOne, a}}}}
				return makeDiv(zVar, zMinusA), nil
			}
		}
	}

	// 5. Exponential: exp(c * n) -> z / (z - exp(c))
	if fn, ok := f.(*FuncNode); ok && fn.Name == "exp" && len(fn.Args) == 1 {
		arg := fn.Args[0]
		if isLinearInVar(arg, nName) {
			c := extractVarCoeff(arg, nName)
			var base Node
			if c.Cmp(big.NewRat(1, 1)) == 0 {
				base = &FuncNode{Name: "exp", Args: []Node{one}}
			} else {
				base = &FuncNode{Name: "exp", Args: []Node{&RationalNode{Val: c}}}
			}
			zMinusBase := &AddNode{Terms: []Node{zVar, &MulNode{Factors: []Node{negOne, base}}}}
			return makeDiv(zVar, zMinusBase), nil
		}
	}

	// 6. Trigonometric functions: sin(w * n) and cos(w * n)
	if fn, ok := f.(*FuncNode); ok && (fn.Name == "sin" || fn.Name == "cos") && len(fn.Args) == 1 {
		arg := fn.Args[0]
		w, okW := extractTrigFrequency(arg, nName)
		if okW {
			z2 := &PowNode{Base: zVar, Exp: &RationalNode{Val: big.NewRat(2, 1)}}
			cosW := &FuncNode{Name: "cos", Args: []Node{w}}
			sinW := &FuncNode{Name: "sin", Args: []Node{w}}
			twoZCosW := &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-2, 1)}, zVar, cosW}}
			den := &AddNode{Terms: []Node{z2, twoZCosW, one}}

			if fn.Name == "sin" {
				// sin(w*n) -> z*sin(w) / (z^2 - 2z*cos(w) + 1)
				num := &MulNode{Factors: []Node{zVar, sinW}}
				return makeDiv(num, den), nil
			}
			// cos(w*n) -> z*(z - cos(w)) / (z^2 - 2z*cos(w) + 1)
			zMinusCosW := &AddNode{Terms: []Node{zVar, &MulNode{Factors: []Node{negOne, cosW}}}}
			num := &MulNode{Factors: []Node{zVar, zMinusCosW}}
			return makeDiv(num, den), nil
		}
	}

	// 7. Product combinations: n * a^n, n^2 * a^n, a^n * sin(w*n), a^n * cos(w*n)
	if mul, ok := f.(*MulNode); ok {
		var aBase Node
		var nPolyPower int64 = -1
		var trigFunc *FuncNode

		for _, factor := range mul.Factors {
			if pow, isPow := factor.(*PowNode); isPow {
				if v, isV := pow.Exp.(*VarNode); isV && v.Name == nName && !containsVariable(pow.Base, nName) {
					aBase = pow.Base
					continue
				}
				if v, isV := pow.Base.(*VarNode); isV && v.Name == nName {
					if r, isR := pow.Exp.(*RationalNode); isR && r.Val.IsInt() {
						nPolyPower = r.Val.Num().Int64()
						continue
					}
				}
			}
			if v, isV := factor.(*VarNode); isV && v.Name == nName {
				nPolyPower = 1
				continue
			}
			if fn, isFn := factor.(*FuncNode); isFn && (fn.Name == "sin" || fn.Name == "cos") && len(fn.Args) == 1 {
				trigFunc = fn
				continue
			}
		}

		// Case: n * a^n -> a*z / (z - a)^2
		if aBase != nil && nPolyPower == 1 && trigFunc == nil {
			zMinusA := &AddNode{Terms: []Node{zVar, &MulNode{Factors: []Node{negOne, aBase}}}}
			num := &MulNode{Factors: []Node{aBase, zVar}}
			den := &PowNode{Base: zMinusA, Exp: &RationalNode{Val: big.NewRat(2, 1)}}
			return makeDiv(num, den), nil
		}

		// Case: n^2 * a^n -> a*z*(z + a) / (z - a)^3
		if aBase != nil && nPolyPower == 2 && trigFunc == nil {
			zMinusA := &AddNode{Terms: []Node{zVar, &MulNode{Factors: []Node{negOne, aBase}}}}
			zPlusA := &AddNode{Terms: []Node{zVar, aBase}}
			num := &MulNode{Factors: []Node{aBase, zVar, zPlusA}}
			den := &PowNode{Base: zMinusA, Exp: &RationalNode{Val: big.NewRat(3, 1)}}
			return makeDiv(num, den), nil
		}

		// Case: a^n * sin(w*n) or a^n * cos(w*n)
		if aBase != nil && trigFunc != nil && nPolyPower == -1 {
			w, okW := extractTrigFrequency(trigFunc.Args[0], nName)
			if okW {
				z2 := &PowNode{Base: zVar, Exp: &RationalNode{Val: big.NewRat(2, 1)}}
				a2 := &PowNode{Base: aBase, Exp: &RationalNode{Val: big.NewRat(2, 1)}}
				cosW := &FuncNode{Name: "cos", Args: []Node{w}}
				sinW := &FuncNode{Name: "sin", Args: []Node{w}}
				twoAzCosW := &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-2, 1)}, aBase, zVar, cosW}}
				den := &AddNode{Terms: []Node{z2, twoAzCosW, a2}}

				if trigFunc.Name == "sin" {
					num := &MulNode{Factors: []Node{aBase, zVar, sinW}}
					return makeDiv(num, den), nil
				}
				aCosW := &MulNode{Factors: []Node{aBase, cosW}}
				zMinusACosW := &AddNode{Terms: []Node{zVar, &MulNode{Factors: []Node{negOne, aCosW}}}}
				num := &MulNode{Factors: []Node{zVar, zMinusACosW}}
				return makeDiv(num, den), nil
			}
		}
	}

	// 8. General sequence function shift: y(n + k) or y(n - k)
	if fn, ok := f.(*FuncNode); ok && len(fn.Args) == 1 {
		shift, err := parseShiftArg(fn.Args[0], nName)
		if err == nil {
			return ztransSequenceShift(fn.Name, shift, zVar)
		}
	}

	return nil, fmt.Errorf("%s", i18n.T("ztrans.err_cannot_transform", f.String()))
}

// ztransSequenceShift handles Z{y[n+k]} or Z{y[n-k]}
func ztransSequenceShift(seqName string, shift int, zVar *VarNode) (Node, error) {
	capitalY := &FuncNode{Name: seqName, Args: []Node{zVar}}
	if shift == 0 {
		return capitalY, nil
	}

	if shift > 0 {
		// Advance shift: Z{y[n+k]} = z^k * Y(z) - sum_{j=0}^{k-1} z^(k-j) * y(j)
		zk := &PowNode{Base: zVar, Exp: &RationalNode{Val: big.NewRat(int64(shift), 1)}}
		mainTerm := &MulNode{Factors: []Node{zk, capitalY}}
		var subTerms []Node
		subTerms = append(subTerms, mainTerm)

		for j := 0; j < shift; j++ {
			p := shift - j
			var zPow Node
			if p == 1 {
				zPow = zVar
			} else {
				zPow = &PowNode{Base: zVar, Exp: &RationalNode{Val: big.NewRat(int64(p), 1)}}
			}
			yInit := &FuncNode{Name: seqName, Args: []Node{&RationalNode{Val: big.NewRat(int64(j), 1)}}}
			negInit := &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, zPow, yInit}}
			subTerms = append(subTerms, negInit)
		}
		return &AddNode{Terms: subTerms}, nil
	}

	// Delay shift (causal): Z{y[n - k]} = z^(-k) * Y(z)
	delayExp := &RationalNode{Val: big.NewRat(int64(shift), 1)}
	zDelay := &PowNode{Base: zVar, Exp: delayExp}
	return &MulNode{Factors: []Node{zDelay, capitalY}}, nil
}

func extractTrigFrequency(arg Node, nName string) (Node, bool) {
	if v, isV := arg.(*VarNode); isV && v.Name == nName {
		return &RationalNode{Val: big.NewRat(1, 1)}, true
	}
	if mul, isMul := arg.(*MulNode); isMul {
		var otherFactors []Node
		hasN := false
		for _, fac := range mul.Factors {
			if v, isV := fac.(*VarNode); isV && v.Name == nName {
				hasN = true
			} else if !containsVariable(fac, nName) {
				otherFactors = append(otherFactors, fac)
			}
		}
		if hasN && len(otherFactors) > 0 {
			if len(otherFactors) == 1 {
				return otherFactors[0], true
			}
			return &MulNode{Factors: otherFactors}, true
		}
	}
	if isLinearInVar(arg, nName) {
		coeff := extractVarCoeff(arg, nName)
		return &RationalNode{Val: coeff}, true
	}
	return nil, false
}
