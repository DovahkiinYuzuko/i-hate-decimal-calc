package calc

import (
	"fmt"
	"math/big"
)

// EvalLaplace computes the symbolic Laplace transform of f(t): L[f(t)](s) = int_0^inf exp(-st)*f(t) dt.
func EvalLaplace(f Node, tName, sName string, env *Env) (Node, error) {
	if f == nil {
		return nil, fmt.Errorf("laplace: input expression cannot be nil")
	}

	if tName == "" {
		// Infer tName: prefer "t", then "x", then any variable in f
		if ContainsVar(f, "t") {
			tName = "t"
		} else if ContainsVar(f, "x") {
			tName = "x"
		} else {
			vars := ExtractFreeVariables(f)
			if len(vars) > 0 {
				tName = vars[0]
			} else {
				tName = "t"
			}
		}
	}

	if sName == "" {
		if tName == "s" {
			sName = "p"
		} else {
			sName = "s"
		}
	}

	sVar := &VarNode{Name: sName}

	// Expand f linearly: (A + B + ...)
	expanded := expandNode(f)
	var terms []Node
	if add, ok := expanded.(*AddNode); ok {
		terms = add.Terms
	} else {
		terms = []Node{expanded}
	}

	var laplaceTerms []Node
	for _, term := range terms {
		lTerm, err := laplaceTerm(term, tName, sVar, env)
		if err != nil {
			return nil, err
		}
		laplaceTerms = append(laplaceTerms, lTerm)
	}

	return simplifyAdd(laplaceTerms)
}

// laplaceTerm transforms a single term (or product of factors)
func laplaceTerm(term Node, tName string, sVar *VarNode, env *Env) (Node, error) {
	// If term does not contain t, L[C] = C / s
	if !ContainsVar(term, tName) {
		invS, err := simplifyPow(sVar, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		return simplifyMul([]Node{term, invS})
	}

	// Separate constant factors and t-dependent factors: term = coeff * f(t)
	coeff, fNode, err := extractConstantCoeff(term, tName)
	if err != nil {
		return nil, err
	}

	// Now match fNode against standard elementary forms:
	// 1. t^n (n >= 1 integer, with n=1 when fNode is VarNode(t))
	if n, ok := extractPowerOfVar(fNode, tName); ok && n >= 1 {
		// L[t^n] = n! / s^(n+1)
		factN := factorialBigInt(n)
		factRat := new(big.Rat).SetInt(factN)
		coeffTotal := new(big.Rat).Mul(coeff, factRat)

		negNPlus1 := mustRational(-(n+1), 1)
		invSPow, err := simplifyPow(sVar, negNPlus1)
		if err != nil {
			return nil, err
		}
		return simplifyMul([]Node{&RationalNode{Val: coeffTotal}, invSPow})
	}

	// 2. exp(a * t)
	if a, ok := extractExpCoeff(fNode, tName); ok {
		// L[e^(at)] = 1 / (s - a)
		sMinusA, err := makeSMinusA(sVar, a)
		if err != nil {
			return nil, err
		}
		invSMA, err := simplifyPow(sMinusA, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		return simplifyMul([]Node{&RationalNode{Val: coeff}, invSMA})
	}

	// 3. sin(omega * t) or cos(omega * t)
	if omega, isSin, ok := extractTrigCoeff(fNode, tName); ok {
		// L[sin(w*t)] = w / (s^2 + w^2)
		// L[cos(w*t)] = s / (s^2 + w^2)
		den, err := makeS2PlusW2(sVar, omega)
		if err != nil {
			return nil, err
		}
		invDen, err := simplifyPow(den, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}

		var num Node
		if isSin {
			numVal := new(big.Rat).Mul(coeff, omega)
			num = &RationalNode{Val: numVal}
		} else {
			num, err = simplifyMul([]Node{&RationalNode{Val: coeff}, sVar})
			if err != nil {
				return nil, err
			}
		}
		return simplifyMul([]Node{num, invDen})
	}

	// 4. Products: e^(at) * t^n, e^(at) * sin(wt), e^(at) * cos(wt)
	if mul, ok := fNode.(*MulNode); ok {
		var aRat *big.Rat
		var remaining []Node
		for _, factor := range mul.Factors {
			if a, isExp := extractExpCoeff(factor, tName); isExp && aRat == nil {
				aRat = a
			} else {
				remaining = append(remaining, factor)
			}
		}

		if aRat != nil && len(remaining) > 0 {
			remNode, err := simplifyMul(remaining)
			if err != nil {
				return nil, err
			}

			// L[e^(at) * t^n] = n! / (s - a)^(n+1)
			if n, isPow := extractPowerOfVar(remNode, tName); isPow && n >= 1 {
				factN := factorialBigInt(n)
				factRat := new(big.Rat).SetInt(factN)
				coeffTotal := new(big.Rat).Mul(coeff, factRat)

				sMinusA, err := makeSMinusA(sVar, aRat)
				if err != nil {
					return nil, err
				}
				negNPlus1 := mustRational(-(n+1), 1)
				invSPow, err := simplifyPow(sMinusA, negNPlus1)
				if err != nil {
					return nil, err
				}
				return simplifyMul([]Node{&RationalNode{Val: coeffTotal}, invSPow})
			}

			// L[e^(at) * sin(wt)] = w / ((s - a)^2 + w^2)
			// L[e^(at) * cos(wt)] = (s - a) / ((s - a)^2 + w^2)
			if omega, isSin, isTrig := extractTrigCoeff(remNode, tName); isTrig {
				sMinusA, err := makeSMinusA(sVar, aRat)
				if err != nil {
					return nil, err
				}
				den, err := makeS2PlusW2(sMinusA, omega)
				if err != nil {
					return nil, err
				}
				invDen, err := simplifyPow(den, mustRational(-1, 1))
				if err != nil {
					return nil, err
				}

				var num Node
				if isSin {
					numVal := new(big.Rat).Mul(coeff, omega)
					num = &RationalNode{Val: numVal}
				} else {
					num, err = simplifyMul([]Node{&RationalNode{Val: coeff}, sMinusA})
					if err != nil {
						return nil, err
					}
				}
				return simplifyMul([]Node{num, invDen})
			}
		}
	}

	return nil, fmt.Errorf("laplace: cannot transform unsupported term %s", term.String())
}

// EvalInvLaplace computes the symbolic inverse Laplace transform: L^-1[F(s)](t).
func EvalInvLaplace(F Node, sName, tName string, env *Env) (Node, error) {
	if F == nil {
		return nil, fmt.Errorf("inv_laplace: input expression cannot be nil")
	}

	if sName == "" {
		if ContainsVar(F, "s") {
			sName = "s"
		} else if ContainsVar(F, "p") {
			sName = "p"
		} else {
			vars := ExtractFreeVariables(F)
			if len(vars) > 0 {
				sName = vars[0]
			} else {
				sName = "s"
			}
		}
	}

	if tName == "" {
		if sName == "t" {
			tName = "tau"
		} else {
			tName = "t"
		}
	}

	tVar := &VarNode{Name: tName}

	// 1. Partial fraction decomposition over sName
	decomp, err := EvalApart(F, sName)
	if err != nil {
		// If apart fails, fall back to analyzing F directly
		decomp = F
	}

	var terms []Node
	if add, ok := decomp.(*AddNode); ok {
		terms = add.Terms
	} else {
		terms = []Node{decomp}
	}

	var invTerms []Node
	for _, term := range terms {
		it, err := invLaplaceTerm(term, sName, tVar, env)
		if err != nil {
			return nil, err
		}
		invTerms = append(invTerms, it)
	}

	return simplifyAdd(invTerms)
}

// invLaplaceTerm transforms a single partial fraction component to the time domain.
func invLaplaceTerm(term Node, sName string, tVar *VarNode, env *Env) (Node, error) {
	// If term does not contain s, its inverse Laplace transform involves Dirac delta delta(t)
	if !ContainsVar(term, sName) {
		deltaNode := &FuncNode{Name: "delta", Args: []Node{tVar}}
		return simplifyMul([]Node{term, deltaNode})
	}

	// Pattern 1: C / (s - a)^n
	// Format in AST: C * (s - a)^(-n) or (s - a)^(-n) or C / (s - a)
	coeff, aRat, n, ok1 := extractLinearPole(term, sName)
	if ok1 && n >= 1 {
		// L^-1[ C / (s - a)^n ] = C * t^(n-1) / (n-1)! * exp(at)
		factNMinus1 := factorialBigInt(n - 1)
		factRat := new(big.Rat).SetInt(factNMinus1)
		overallCoeff := new(big.Rat).Quo(coeff, factRat)

		var factors []Node
		if overallCoeff.Cmp(big.NewRat(1, 1)) != 0 || n == 1 && aRat.Sign() == 0 {
			factors = append(factors, &RationalNode{Val: overallCoeff})
		}

		if n > 1 {
			if n == 2 {
				factors = append(factors, tVar)
			} else {
				tPow, err := simplifyPow(tVar, mustRational(n-1, 1))
				if err != nil {
					return nil, err
				}
				factors = append(factors, tPow)
			}
		}

		if aRat.Sign() != 0 {
			at, err := simplifyMul([]Node{&RationalNode{Val: aRat}, tVar})
			if err != nil {
				return nil, err
			}
			expPart, err := simplifyFuncWithEnv("exp", []Node{at}, env)
			if err != nil {
				return nil, err
			}
			factors = append(factors, expPart)
		}

		if len(factors) == 0 {
			return mustRational(1, 1), nil
		}
		return simplifyMul(factors)
	}

	// Pattern 2: Quadratic denominator: (A*s + B) / ((s - a)^2 + w^2) or (A*s + B) / (s^2 + w^2)
	A, B, aQuad, wQuad, ok2 := extractQuadraticPole(term, sName)
	if ok2 && wQuad.Sign() > 0 {
		// L^-1[ (A*s + B) / ((s - a)^2 + w^2) ]
		// = L^-1[ A*(s - a) / ((s - a)^2 + w^2) + (B + A*a)/w * w / ((s - a)^2 + w^2) ]
		// = exp(at) * [ A*cos(wt) + ((B + A*a)/w)*sin(wt) ]
		wt, err := simplifyMul([]Node{&RationalNode{Val: wQuad}, tVar})
		if err != nil {
			return nil, err
		}
		cosPart, err := simplifyFuncWithEnv("cos", []Node{wt}, env)
		if err != nil {
			return nil, err
		}
		sinPart, err := simplifyFuncWithEnv("sin", []Node{wt}, env)
		if err != nil {
			return nil, err
		}

		var trigParts []Node
		if A.Sign() != 0 {
			tA, err := simplifyMul([]Node{&RationalNode{Val: A}, cosPart})
			if err != nil {
				return nil, err
			}
			trigParts = append(trigParts, tA)
		}

		// Coeff of sin: (B + A*a) / w
		bPlusAa := new(big.Rat).Add(B, new(big.Rat).Mul(A, aQuad))
		sinCoeff := new(big.Rat).Quo(bPlusAa, wQuad)
		if sinCoeff.Sign() != 0 {
			tB, err := simplifyMul([]Node{&RationalNode{Val: sinCoeff}, sinPart})
			if err != nil {
				return nil, err
			}
			trigParts = append(trigParts, tB)
		}

		trigSum, err := simplifyAdd(trigParts)
		if err != nil {
			return nil, err
		}

		if aQuad.Sign() != 0 {
			at, err := simplifyMul([]Node{&RationalNode{Val: aQuad}, tVar})
			if err != nil {
				return nil, err
			}
			expPart, err := simplifyFuncWithEnv("exp", []Node{at}, env)
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{expPart, trigSum})
		}
		return trigSum, nil
	}

	return nil, fmt.Errorf("inv_laplace: unsupported term %s", term.String())
}

// -------------------------------------------------------------------------
// Helpers for pattern matching
// -------------------------------------------------------------------------

func extractConstantCoeff(term Node, tName string) (*big.Rat, Node, error) {
	coeff := big.NewRat(1, 1)

	if u, ok := term.(*UnaryOpNode); ok && u.Op == "-" {
		subCoeff, subF, err := extractConstantCoeff(u.Expr, tName)
		if err != nil {
			return nil, nil, err
		}
		subCoeff.Neg(subCoeff)
		return subCoeff, subF, nil
	}

	if mul, ok := term.(*MulNode); ok {
		var nonConst []Node
		for _, f := range mul.Factors {
			if !ContainsVar(f, tName) {
				if r, ok := f.(*RationalNode); ok {
					coeff.Mul(coeff, r.Val)
				} else {
					return nil, nil, fmt.Errorf("non-rational constant factor %s", f.String())
				}
			} else {
				nonConst = append(nonConst, f)
			}
		}
		if len(nonConst) == 0 {
			return coeff, mustRational(1, 1), nil
		}
		res, err := simplifyMul(nonConst)
		return coeff, res, err
	}

	return coeff, term, nil
}

func extractPowerOfVar(n Node, varName string) (int64, bool) {
	if v, ok := n.(*VarNode); ok && v.Name == varName {
		return 1, true
	}
	if pow, ok := n.(*PowNode); ok {
		if v, ok := pow.Base.(*VarNode); ok && v.Name == varName {
			if r, ok := pow.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() > 0 {
				return r.Val.Num().Int64(), true
			}
		}
	}
	return 0, false
}

func extractExpCoeff(n Node, tName string) (*big.Rat, bool) {
	fn, ok := n.(*FuncNode)
	if !ok || fn.Name != "exp" || len(fn.Args) != 1 {
		return nil, false
	}
	arg := fn.Args[0]
	// arg should be a * t or t or -t
	if v, ok := arg.(*VarNode); ok && v.Name == tName {
		return big.NewRat(1, 1), true
	}
	if u, ok := arg.(*UnaryOpNode); ok && u.Op == "-" {
		if v, ok := u.Expr.(*VarNode); ok && v.Name == tName {
			return big.NewRat(-1, 1), true
		}
	}
	if mul, ok := arg.(*MulNode); ok {
		var coeff = big.NewRat(1, 1)
		foundT := false
		for _, f := range mul.Factors {
			if v, ok := f.(*VarNode); ok && v.Name == tName {
				foundT = true
			} else if r, ok := f.(*RationalNode); ok {
				coeff.Mul(coeff, r.Val)
			} else {
				return nil, false
			}
		}
		if foundT {
			return coeff, true
		}
	}
	return nil, false
}

func extractTrigCoeff(n Node, tName string) (omega *big.Rat, isSin bool, ok bool) {
	fn, isFn := n.(*FuncNode)
	if !isFn || len(fn.Args) != 1 {
		return nil, false, false
	}
	if fn.Name != "sin" && fn.Name != "cos" {
		return nil, false, false
	}
	isSin = (fn.Name == "sin")

	arg := fn.Args[0]
	omega = big.NewRat(1, 1)
	if v, ok := arg.(*VarNode); ok && v.Name == tName {
		return omega, isSin, true
	}
	if mul, ok := arg.(*MulNode); ok {
		foundT := false
		for _, f := range mul.Factors {
			if v, ok := f.(*VarNode); ok && v.Name == tName {
				foundT = true
			} else if r, ok := f.(*RationalNode); ok {
				omega.Mul(omega, r.Val)
			} else {
				return nil, false, false
			}
		}
		if foundT {
			return omega, isSin, true
		}
	}
	return nil, false, false
}

func makeSMinusA(sVar *VarNode, a *big.Rat) (Node, error) {
	if a.Sign() == 0 {
		return sVar, nil
	}
	negA := new(big.Rat).Neg(a)
	return simplifyAdd([]Node{sVar, &RationalNode{Val: negA}})
}

func makeS2PlusW2(sTerm Node, omega *big.Rat) (Node, error) {
	s2, err := simplifyPow(sTerm, mustRational(2, 1))
	if err != nil {
		return nil, err
	}
	w2 := new(big.Rat).Mul(omega, omega)
	return simplifyAdd([]Node{s2, &RationalNode{Val: w2}})
}

func factorialBigInt(n int64) *big.Int {
	res := big.NewInt(1)
	for i := int64(2); i <= n; i++ {
		res.Mul(res, big.NewInt(i))
	}
	return res
}

// extractLinearPole recognizes C / (s - a)^n or C * (s - a)^(-n)
func extractLinearPole(n Node, sName string) (coeff, a *big.Rat, order int64, ok bool) {
	coeff = big.NewRat(1, 1)

	target := n
	if u, okU := n.(*UnaryOpNode); okU && u.Op == "-" {
		subC, subA, subO, okSub := extractLinearPole(u.Expr, sName)
		if okSub {
			subC.Neg(subC)
			return subC, subA, subO, true
		}
		return nil, nil, 0, false
	}

	if mul, okM := n.(*MulNode); okM {
		var powFactor Node
		for _, f := range mul.Factors {
			if r, okR := f.(*RationalNode); okR {
				coeff.Mul(coeff, r.Val)
			} else if powFactor == nil {
				powFactor = f
			} else {
				return nil, nil, 0, false
			}
		}
		if powFactor == nil {
			return nil, nil, 0, false
		}
		target = powFactor
	}

	pow, okP := target.(*PowNode)
	if !okP {
		return nil, nil, 0, false
	}

	rExp, okExp := pow.Exp.(*RationalNode)
	if !okExp || !rExp.Val.IsInt() || rExp.Val.Sign() >= 0 {
		return nil, nil, 0, false
	}
	order = -rExp.Val.Num().Int64()

	base := pow.Base
	if innerPow, okInner := base.(*PowNode); okInner {
		if innerExp, okInnerExp := innerPow.Exp.(*RationalNode); okInnerExp && innerExp.Val.IsInt() && innerExp.Val.Sign() > 0 {
			order = order * innerExp.Val.Num().Int64()
			base = innerPow.Base
		}
	}

	// Base should be s - a or s or a*s + b
	// e.g. s - a: AddNode{Terms: [s, -a]} or VarNode{s}
	if v, okV := base.(*VarNode); okV && v.Name == sName {
		return coeff, big.NewRat(0, 1), order, true
	}

	if add, okA := base.(*AddNode); okA {
		var constVal = big.NewRat(0, 1)
		var sCoeff = big.NewRat(0, 1)
		for _, t := range add.Terms {
			if v, okV := t.(*VarNode); okV && v.Name == sName {
				sCoeff.Add(sCoeff, big.NewRat(1, 1))
			} else if u, okU := t.(*UnaryOpNode); okU && u.Op == "-" {
				if v, okV := u.Expr.(*VarNode); okV && v.Name == sName {
					sCoeff.Sub(sCoeff, big.NewRat(1, 1))
				} else if r, okR := u.Expr.(*RationalNode); okR {
					constVal.Sub(constVal, r.Val)
				} else {
					return nil, nil, 0, false
				}
			} else if r, okR := t.(*RationalNode); okR {
				constVal.Add(constVal, r.Val)
			} else if m, okM := t.(*MulNode); okM {
				subC, subOk := extractLinearCoeff(m, func(node Node) bool {
					vx, ok := node.(*VarNode)
					return ok && vx.Name == sName
				})
				if subOk {
					if r, okR := subC.(*RationalNode); okR {
						sCoeff.Add(sCoeff, r.Val)
					} else {
						return nil, nil, 0, false
					}
				} else {
					return nil, nil, 0, false
				}
			} else {
				return nil, nil, 0, false
			}
		}

		if sCoeff.Sign() == 0 {
			return nil, nil, 0, false
		}

		// Normalize to (s - a):
		// (sCoeff * s + constVal) = sCoeff * (s + constVal/sCoeff)
		// denominator is (sCoeff)^order * (s - a)^order
		// a = -constVal / sCoeff
		a = new(big.Rat).Quo(constVal, sCoeff)
		a.Neg(a)

		sCoeffPow := new(big.Int).Exp(sCoeff.Num(), big.NewInt(order), nil)
		sCoeffDenPow := new(big.Int).Exp(sCoeff.Denom(), big.NewInt(order), nil)
		scale := new(big.Rat).SetFrac(sCoeffPow, sCoeffDenPow)
		coeff.Quo(coeff, scale)

		return coeff, a, order, true
	}

	return nil, nil, 0, false
}

// extractQuadraticPole recognizes (A*s + B) / ((s - a)^2 + w^2) or (A*s + B) * (s^2 + w^2)^(-1)
func extractQuadraticPole(n Node, sName string) (A, B, a, w *big.Rat, ok bool) {
	A = big.NewRat(0, 1)
	B = big.NewRat(0, 1)
	a = big.NewRat(0, 1)
	w = big.NewRat(1, 1)

	var num Node
	var den Node

	if pow, okP := n.(*PowNode); okP {
		if r, okR := pow.Exp.(*RationalNode); okR && r.Val.Cmp(big.NewRat(-1, 1)) == 0 {
			num = mustRational(1, 1)
			den = pow.Base
		}
	} else if mul, okM := n.(*MulNode); okM {
		var powFactor Node
		var numFactors []Node
		for _, f := range mul.Factors {
			if pow, okP := f.(*PowNode); okP {
				if r, okR := pow.Exp.(*RationalNode); okR && r.Val.Cmp(big.NewRat(-1, 1)) == 0 {
					powFactor = pow
					continue
				}
			}
			numFactors = append(numFactors, f)
		}
		if powFactor != nil {
			den = powFactor.(*PowNode).Base
			var err error
			num, err = simplifyMul(numFactors)
			if err != nil {
				return nil, nil, nil, nil, false
			}
		}
	}

	if den == nil {
		return nil, nil, nil, nil, false
	}

	// 1. Analyze numerator: A*s + B
	numExpanded := expandNode(num)
	var numTerms []Node
	if add, ok := numExpanded.(*AddNode); ok {
		numTerms = add.Terms
	} else {
		numTerms = []Node{numExpanded}
	}

	for _, t := range numTerms {
		if c, okS := extractLinearCoeff(t, func(node Node) bool {
			vx, ok := node.(*VarNode)
			return ok && vx.Name == sName
		}); okS {
			if r, okR := c.(*RationalNode); okR {
				A.Add(A, r.Val)
			} else {
				return nil, nil, nil, nil, false
			}
		} else if !ContainsVar(t, sName) {
			if r, okR := t.(*RationalNode); okR {
				B.Add(B, r.Val)
			} else {
				return nil, nil, nil, nil, false
			}
		} else {
			return nil, nil, nil, nil, false
		}
	}

	// 2. Analyze denominator: as^2 + bs + c => monic: s^2 + (b/a)s + (c/a)
	// completing the square: (s + b/2a)^2 + (c/a - (b/2a)^2)
	denExpanded := expandNode(den)
	var denTerms []Node
	if add, ok := denExpanded.(*AddNode); ok {
		denTerms = add.Terms
	} else {
		denTerms = []Node{denExpanded}
	}

	coeffA := big.NewRat(0, 1)
	coeffB := big.NewRat(0, 1)
	coeffC := big.NewRat(0, 1)

	for _, t := range denTerms {
		if c, okS2 := extractLinearCoeff(t, func(node Node) bool {
			p, okP := node.(*PowNode)
			if !okP {
				return false
			}
			vx, okV := p.Base.(*VarNode)
			r, okR := p.Exp.(*RationalNode)
			return okV && vx.Name == sName && okR && r.Val.Cmp(big.NewRat(2, 1)) == 0
		}); okS2 {
			if r, okR := c.(*RationalNode); okR {
				coeffA.Add(coeffA, r.Val)
			} else {
				return nil, nil, nil, nil, false
			}
		} else if c, okS := extractLinearCoeff(t, func(node Node) bool {
			vx, ok := node.(*VarNode)
			return ok && vx.Name == sName
		}); okS {
			if r, okR := c.(*RationalNode); okR {
				coeffB.Add(coeffB, r.Val)
			} else {
				return nil, nil, nil, nil, false
			}
		} else if !ContainsVar(t, sName) {
			if r, okR := t.(*RationalNode); okR {
				coeffC.Add(coeffC, r.Val)
			} else {
				return nil, nil, nil, nil, false
			}
		} else {
			return nil, nil, nil, nil, false
		}
	}

	if coeffA.Sign() == 0 {
		return nil, nil, nil, nil, false
	}

	// Normalize by coeffA
	if coeffA.Cmp(big.NewRat(1, 1)) != 0 {
		coeffB.Quo(coeffB, coeffA)
		coeffC.Quo(coeffC, coeffA)
		A.Quo(A, coeffA)
		B.Quo(B, coeffA)
	}

	// Completing the square: (s - a)^2 + w^2
	// s^2 + coeffB*s + coeffC = (s + coeffB/2)^2 + (coeffC - (coeffB/2)^2)
	// So a = -coeffB / 2
	bOver2 := new(big.Rat).Quo(coeffB, big.NewRat(2, 1))
	a = new(big.Rat).Neg(bOver2)

	w2 := new(big.Rat).Sub(coeffC, new(big.Rat).Mul(bOver2, bOver2))
	if w2.Sign() <= 0 {
		// Not irreducible over R
		return nil, nil, nil, nil, false
	}

	sqrtW2, isPerf := exactRatSqrt(w2)
	if !isPerf {
		return nil, nil, nil, nil, false
	}
	w = sqrtW2

	return A, B, a, w, true
}
