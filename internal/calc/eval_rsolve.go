package calc

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterFunction(FunctionSpec{
		Name:            "rsolve",
		MinArgs:         2,
		MaxArgs:         3,
		AllowBareSymbol: true,
		LazyArgs:        true,
		Handler:         handleRSolve,
	})
}

func handleRSolve(args []Node, env *Env) (Node, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "rsolve", 2, 3, len(args)))
	}

	var initsNode Node
	if len(args) == 3 {
		initsNode = args[2]
	}

	return EvalRSolve(args[0], args[1], initsNode, env)
}

// EvalRSolve solves a linear recurrence relation with constant coefficients.
// Syntax: rsolve(eq, a(n), [inits])
// Example: rsolve(a(n+2) == a(n+1) + a(n), a(n), [a(0) == 0, a(1) == 1])
func EvalRSolve(eqNode, fnNode, initsNode Node, env *Env) (Node, error) {
	if eqNode == nil || fnNode == nil {
		return nil, fmt.Errorf("%s", i18n.T("rsolve.err_rsolve_equation_and_sequence_function"))
	}

	// 1. Identify sequence name and index variable from fnNode (e.g. a(n))
	seqName, idxVar, err := extractSequenceTarget(fnNode)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("calc.rsolve_func_required", fnNode.String()))
	}

	// 2. Parse equation into homogeneous coefficients and inhomogeneous term f(n)
	coeffs, fNode, order, err := parseRecurrenceEquation(eqNode, seqName, idxVar)
	if err != nil {
		return nil, err
	}

	if order < 1 {
		return nil, fmt.Errorf("%s", i18n.T("calc.rsolve_linear_constant_coeff", eqNode.String()))
	}

	// 3. Solve homogeneous recurrence part -> characteristic roots and basis functions
	homogBases, err := solveHomogeneousRecurrence(coeffs, idxVar, order)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("calc.rsolve_failed", err))
	}

	// 4. Solve particular recurrence part for inhomogeneous term f(n)
	partSol, err := solveParticularRecurrence(coeffs, fNode, idxVar, order)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("calc.rsolve_failed", err))
	}

	// 5. Parse initial conditions
	inits, err := parseInitialConditions(initsNode, seqName)
	if err != nil {
		return nil, err
	}

	// 6. Fit initial conditions or keep symbolic constants (C1, C2, ...)
	res, err := fitInitialConditions(homogBases, partSol, idxVar, inits)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("calc.rsolve_failed", err))
	}

	return Eval(res)
}

// extractSequenceTarget extracts sequence function name and index variable from target (e.g. a(n)).
func extractSequenceTarget(target Node) (string, string, error) {
	if fn, ok := target.(*FuncNode); ok && len(fn.Args) == 1 {
		if v, isV := fn.Args[0].(*VarNode); isV {
			return fn.Name, v.Name, nil
		}
	}
	if v, ok := target.(*VarNode); ok {
		return v.Name, "n", nil
	}
	return "", "", fmt.Errorf("%s", i18n.T("rsolve.err_expected_sequence_function_like_a", target.String()))
}

// parseRecurrenceEquation converts eq into homogeneous shift coefficients and non-homogeneous f(n).
func parseRecurrenceEquation(eqNode Node, seqName, idxVar string) (map[int]*big.Rat, Node, int, error) {
	var diffExpr Node
	if rel, ok := eqNode.(*RelOpNode); ok && (rel.Op == "==" || rel.Op == "=") {
		negRHS, err := simplifyUnaryOp("-", rel.RHS)
		if err != nil {
			return nil, nil, 0, err
		}
		diffExpr, err = simplifyAdd([]Node{rel.LHS, negRHS})
		if err != nil {
			return nil, nil, 0, err
		}
	} else {
		return nil, nil, 0, fmt.Errorf("%s", i18n.T("calc.rsolve_equation_required", eqNode.String()))
	}

	expanded := expandNode(diffExpr)
	terms := collectAddTerms(expanded)
	shiftCoeffs := make(map[int]*big.Rat)
	var inhomeTerms []Node

	for _, t := range terms {
		shift, coeff, isSeq, err := matchSequenceTerm(t, seqName, idxVar)
		if err != nil {
			return nil, nil, 0, err
		}
		if isSeq {
			if curr, exists := shiftCoeffs[shift]; exists {
				curr.Add(curr, coeff)
			} else {
				shiftCoeffs[shift] = new(big.Rat).Set(coeff)
			}
		} else {
			negT, err := simplifyUnaryOp("-", t)
			if err != nil {
				negT = &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, t}}
			}
			inhomeTerms = append(inhomeTerms, negT)
		}
	}

	if len(shiftCoeffs) == 0 {
		return nil, nil, 0, fmt.Errorf("%s", i18n.T("rsolve.err_no_terms_of_sequence_found", seqName, idxVar))
	}

	minShift := 1 << 30
	maxShift := -(1 << 30)
	for s := range shiftCoeffs {
		if s < minShift {
			minShift = s
		}
		if s > maxShift {
			maxShift = s
		}
	}

	order := maxShift - minShift
	normalizedCoeffs := make(map[int]*big.Rat)
	for s, c := range shiftCoeffs {
		if c.Sign() != 0 {
			normalizedCoeffs[s-minShift] = c
		}
	}

	var fNode Node
	if len(inhomeTerms) == 0 {
		fNode = &RationalNode{Val: big.NewRat(0, 1)}
	} else if len(inhomeTerms) == 1 {
		fNode = inhomeTerms[0]
	} else {
		fNode = &AddNode{Terms: inhomeTerms}
	}

	if minShift != 0 {
		fNode = substituteVar(fNode, idxVar, &AddNode{Terms: []Node{&VarNode{Name: idxVar}, &RationalNode{Val: big.NewRat(int64(-minShift), 1)}}})
	}

	fNode, _ = Eval(fNode)
	return normalizedCoeffs, fNode, order, nil
}

func matchSequenceTerm(t Node, seqName, idxVar string) (shift int, coeff *big.Rat, isSeq bool, err error) {
	if !containsSequenceFunction(t, seqName) {
		return 0, nil, false, nil
	}

	factors := collectMulFactors(t)
	coeff = big.NewRat(1, 1)
	var seqFunc *FuncNode

	for _, f := range factors {
		if r, ok := f.(*RationalNode); ok {
			coeff.Mul(coeff, r.Val)
		} else if u, ok := f.(*UnaryOpNode); ok && u.Op == "-" {
			if r, ok := u.Expr.(*RationalNode); ok {
				coeff.Mul(coeff, new(big.Rat).Neg(r.Val))
			} else {
				coeff.Mul(coeff, big.NewRat(-1, 1))
				factors = append(factors, u.Expr)
			}
		} else if fn, ok := f.(*FuncNode); ok && fn.Name == seqName && len(fn.Args) == 1 {
			if seqFunc != nil {
				return 0, nil, false, fmt.Errorf("%s", i18n.T("calc.rsolve_linear_constant_coeff", t.String()))
			}
			seqFunc = fn
		} else {
			if containsVariable(f, idxVar) {
				return 0, nil, false, fmt.Errorf("%s", i18n.T("calc.rsolve_linear_constant_coeff", t.String()))
			}
			evaled, err := Eval(f)
			if err == nil {
				if r, ok := evaled.(*RationalNode); ok {
					coeff.Mul(coeff, r.Val)
					continue
				}
			}
			return 0, nil, false, fmt.Errorf("%s", i18n.T("calc.rsolve_linear_constant_coeff", t.String()))
		}
	}

	if seqFunc == nil {
		return 0, nil, false, nil
	}

	arg := seqFunc.Args[0]
	shift, err = parseShiftArg(arg, idxVar)
	if err != nil {
		return 0, nil, false, err
	}

	return shift, coeff, true, nil
}

func parseShiftArg(arg Node, idxVar string) (int, error) {
	if v, ok := arg.(*VarNode); ok && v.Name == idxVar {
		return 0, nil
	}
	if add, ok := arg.(*AddNode); ok {
		hasVar := false
		shift := int64(0)
		for _, t := range add.Terms {
			if v, ok := t.(*VarNode); ok && v.Name == idxVar {
				hasVar = true
			} else if r, ok := t.(*RationalNode); ok && r.Val.IsInt() {
				shift += r.Val.Num().Int64()
			} else if u, ok := t.(*UnaryOpNode); ok && u.Op == "-" {
				if r, ok := u.Expr.(*RationalNode); ok && r.Val.IsInt() {
					shift -= r.Val.Num().Int64()
				}
			} else {
				return 0, fmt.Errorf("%s", i18n.T("rsolve.err_unsupported_index_expression_in_recurrence", arg.String()))
			}
		}
		if hasVar {
			return int(shift), nil
		}
	}
	return 0, fmt.Errorf("%s", i18n.T("rsolve.err_unsupported_index_expression_in_recurrence", arg.String()))
}

// solveHomogeneousRecurrence solves the characteristic equation for order >= 1.
func solveHomogeneousRecurrence(coeffs map[int]*big.Rat, idxVar string, order int) ([]Node, error) {
	if order == 1 {
		c1, ok1 := coeffs[1]
		c0, ok0 := coeffs[0]
		if !ok1 || !ok0 || c1.Sign() == 0 {
			return nil, fmt.Errorf("%s", i18n.T("rsolve.err_invalid_1st_order_recurrence_coefficients"))
		}
		lambda := new(big.Rat).Quo(new(big.Rat).Neg(c0), c1)
		return []Node{makePowerBasis(lambda, idxVar)}, nil
	}

	if order == 2 {
		c2 := coeffs[2]
		c1 := coeffs[1]
		c0 := coeffs[0]
		if c2 == nil || c2.Sign() == 0 {
			return nil, fmt.Errorf("%s", i18n.T("rsolve.err_invalid_2nd_order_recurrence_coefficients"))
		}
		if c1 == nil {
			c1 = big.NewRat(0, 1)
		}
		if c0 == nil {
			c0 = big.NewRat(0, 1)
		}

		disc := new(big.Rat).Mul(c1, c1)
		fourAc := new(big.Rat).Mul(big.NewRat(4, 1), new(big.Rat).Mul(c2, c0))
		disc.Sub(disc, fourAc)
		twoA := new(big.Rat).Mul(big.NewRat(2, 1), c2)

		if disc.Sign() == 0 {
			lambda := new(big.Rat).Quo(new(big.Rat).Neg(c1), twoA)
			b1 := makePowerBasis(lambda, idxVar)
			b2 := &MulNode{Factors: []Node{&VarNode{Name: idxVar}, b1}}
			return []Node{b1, b2}, nil
		}

		sqrtDiscRat, isSquare := exactRatSqrt(disc)
		if isSquare {
			l1 := new(big.Rat).Quo(new(big.Rat).Add(new(big.Rat).Neg(c1), sqrtDiscRat), twoA)
			l2 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Neg(c1), sqrtDiscRat), twoA)
			return []Node{makePowerBasis(l1, idxVar), makePowerBasis(l2, idxVar)}, nil
		}

		if disc.Sign() > 0 {
			// Real irrational roots: (-c1 +- sqrt(disc)) / 2c2
			negC1Div2A := new(big.Rat).Quo(new(big.Rat).Neg(c1), twoA)
			oneDiv2A := new(big.Rat).Quo(big.NewRat(1, 1), twoA)
			sqrtNode := &SqrtNode{Radicand: &RationalNode{Val: disc}}
			termSqrt := &MulNode{Factors: []Node{&RationalNode{Val: oneDiv2A}, sqrtNode}}

			root1, _ := Eval(&AddNode{Terms: []Node{&RationalNode{Val: negC1Div2A}, termSqrt}})
			root2, _ := Eval(&AddNode{Terms: []Node{&RationalNode{Val: negC1Div2A}, &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, termSqrt}}}})

			return []Node{
				&PowNode{Base: root1, Exp: &VarNode{Name: idxVar}},
				&PowNode{Base: root2, Exp: &VarNode{Name: idxVar}},
			}, nil
		}

		// Complex conjugate roots: disc < 0
		posDisc := new(big.Rat).Neg(disc)
		sqrtPosDisc, isSquarePos := exactRatSqrt(posDisc)
		alpha := new(big.Rat).Quo(new(big.Rat).Neg(c1), twoA)

		var betaNode Node
		if isSquarePos {
			betaRat := new(big.Rat).Quo(sqrtPosDisc, twoA)
			if betaRat.Sign() < 0 {
				betaRat.Neg(betaRat)
			}
			betaNode = &RationalNode{Val: betaRat}
		} else {
			sqrtPart := &SqrtNode{Radicand: &RationalNode{Val: posDisc}}
			oneDiv2A := new(big.Rat).Quo(big.NewRat(1, 1), twoA)
			if oneDiv2A.Sign() < 0 {
				oneDiv2A.Neg(oneDiv2A)
			}
			betaNode = &MulNode{Factors: []Node{&RationalNode{Val: oneDiv2A}, sqrtPart}}
		}

		c1Node := &ComplexNode{Real: &RationalNode{Val: alpha}, Imag: betaNode}
		c2Node := &ComplexNode{Real: &RationalNode{Val: alpha}, Imag: &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, betaNode}}}

		r1Eval, _ := Eval(c1Node)
		r2Eval, _ := Eval(c2Node)

		return []Node{
			&PowNode{Base: r1Eval, Exp: &VarNode{Name: idxVar}},
			&PowNode{Base: r2Eval, Exp: &VarNode{Name: idxVar}},
		}, nil
	}

	// For order >= 3: Find rational roots using factor.go machinery
	cPoly := make([]*big.Rat, order+1)
	for j := 0; j <= order; j++ {
		if c, ok := coeffs[j]; ok {
			cPoly[j] = new(big.Rat).Set(c)
		} else {
			cPoly[j] = big.NewRat(0, 1)
		}
	}

	roots, remPoly, err := factorRecurrenceRationalRoots(cPoly, order)
	if err != nil {
		return nil, err
	}

	var bases []Node
	for _, r := range roots {
		for j := 0; j < r.Multiplicity; j++ {
			b := makePowerBasis(r.Val, idxVar)
			if j == 0 {
				bases = append(bases, b)
			} else if j == 1 {
				bases = append(bases, &MulNode{Factors: []Node{&VarNode{Name: idxVar}, b}})
			} else {
				jPow := &PowNode{Base: &VarNode{Name: idxVar}, Exp: &RationalNode{Val: big.NewRat(int64(j), 1)}}
				bases = append(bases, &MulNode{Factors: []Node{jPow, b}})
			}
		}
	}

	remDeg := len(remPoly) - 1
	for remDeg >= 0 && remPoly[remDeg].Sign() == 0 {
		remDeg--
	}

	if remDeg == 2 {
		remMap := map[int]*big.Rat{
			2: remPoly[2],
			1: remPoly[1],
			0: remPoly[0],
		}
		remBases, err := solveHomogeneousRecurrence(remMap, idxVar, 2)
		if err != nil {
			return nil, err
		}
		bases = append(bases, remBases...)
	} else if remDeg > 2 {
		return nil, fmt.Errorf("%s", i18n.T("rsolve.err_recurrence_order", order))
	}

	if len(bases) != order {
		return nil, fmt.Errorf("%s", i18n.T("rsolve.err_recurrence_order", order))
	}

	return bases, nil
}

type ratRootWithMult struct {
	Val          *big.Rat
	Multiplicity int
}

// factorRecurrenceRationalRoots converts rational coefficients to integer polynomial
// and uses factor.go's findRationalRoots and syntheticDivide.
func factorRecurrenceRationalRoots(poly []*big.Rat, deg int) ([]ratRootWithMult, []*big.Rat, error) {
	// Find common denominator to convert to []*big.Int
	lcmDenom := big.NewInt(1)
	for _, c := range poly {
		if c.Sign() != 0 {
			lcmDenom = lcmInt(lcmDenom, c.Denom())
		}
	}

	intPoly := make([]*big.Int, len(poly))
	for i, c := range poly {
		num := new(big.Int).Mul(c.Num(), new(big.Int).Div(lcmDenom, c.Denom()))
		intPoly[i] = num
	}

	var roots []ratRootWithMult

	for len(intPoly)-1 >= 3 {
		curDeg := len(intPoly) - 1
		if intPoly[curDeg].Sign() == 0 {
			intPoly = intPoly[:curDeg]
			continue
		}
		if intPoly[0].Sign() == 0 {
			roots = append(roots, ratRootWithMult{Val: big.NewRat(0, 1), Multiplicity: 1})
			intPoly = intPoly[1:]
			continue
		}

		cands := findRationalRoots(intPoly)
		if len(cands) == 0 {
			break
		}

		found := false
		for _, cand := range cands {
			quot, ok := syntheticDivide(intPoly, cand.num, cand.denom)
			if ok {
				ratVal := new(big.Rat).SetFrac(cand.num, cand.denom)
				mult := 1
				intPoly = quot
				// Check for higher multiplicity
				for len(intPoly)-1 >= 1 {
					nextQuot, nextOk := syntheticDivide(intPoly, cand.num, cand.denom)
					if nextOk {
						mult++
						intPoly = nextQuot
					} else {
						break
					}
				}
				roots = append(roots, ratRootWithMult{Val: ratVal, Multiplicity: mult})
				found = true
				break
			}
		}

		if !found {
			break
		}
	}

	// Convert remaining integer polynomial back to []*big.Rat
	remPoly := make([]*big.Rat, len(intPoly))
	for i, c := range intPoly {
		remPoly[i] = new(big.Rat).SetInt(c)
	}

	return roots, remPoly, nil
}

func makePowerBasis(lambda *big.Rat, idxVar string) Node {
	one := big.NewRat(1, 1)
	if lambda.Cmp(one) == 0 {
		return &RationalNode{Val: big.NewRat(1, 1)}
	}
	return &PowNode{Base: &RationalNode{Val: lambda}, Exp: &VarNode{Name: idxVar}}
}

// solveParticularRecurrence finds a particular solution a_p(n) for non-homogeneous term f(n).
func solveParticularRecurrence(coeffs map[int]*big.Rat, fNode Node, idxVar string, order int) (Node, error) {
	if fNode == nil {
		return &RationalNode{Val: big.NewRat(0, 1)}, nil
	}
	fEval, _ := Eval(fNode)
	if r, ok := fEval.(*RationalNode); ok && r.Val.Sign() == 0 {
		return &RationalNode{Val: big.NewRat(0, 1)}, nil
	}

	var subTerms []Node
	if add, ok := fEval.(*AddNode); ok {
		subTerms = add.Terms
	} else {
		subTerms = []Node{fEval}
	}

	var partParts []Node
	for _, st := range subTerms {
		p, err := solveSingleParticularTerm(coeffs, st, idxVar, order)
		if err != nil {
			return nil, err
		}
		partParts = append(partParts, p)
	}

	if len(partParts) == 1 {
		return partParts[0], nil
	}
	return &AddNode{Terms: partParts}, nil
}

// solveSingleParticularTerm handles constant, polynomial in n, or geometric r^n with resonance support.
func solveSingleParticularTerm(coeffs map[int]*big.Rat, term Node, idxVar string, order int) (Node, error) {
	// Case 1: Constant term C
	if r, ok := term.(*RationalNode); ok {
		p1 := big.NewRat(0, 1)
		for _, c := range coeffs {
			p1.Add(p1, c)
		}
		if p1.Sign() != 0 {
			a := new(big.Rat).Quo(r.Val, p1)
			return &RationalNode{Val: a}, nil
		}
		// 1 is a characteristic root: try A * n
		pPrime1 := big.NewRat(0, 1)
		for j, c := range coeffs {
			pPrime1.Add(pPrime1, new(big.Rat).Mul(big.NewRat(int64(j), 1), c))
		}
		if pPrime1.Sign() != 0 {
			a := new(big.Rat).Quo(r.Val, pPrime1)
			return &MulNode{Factors: []Node{&RationalNode{Val: a}, &VarNode{Name: idxVar}}}, nil
		}
		// 1 is a double root: try A * n^2
		pDoublePrime1 := big.NewRat(0, 1)
		for j, c := range coeffs {
			jInt := int64(j)
			termVal := new(big.Rat).Mul(big.NewRat(jInt*(jInt-1), 1), c)
			pDoublePrime1.Add(pDoublePrime1, termVal)
		}
		if pDoublePrime1.Sign() != 0 {
			two := big.NewRat(2, 1)
			denom := new(big.Rat).Quo(pDoublePrime1, two)
			a := new(big.Rat).Quo(r.Val, denom)
			n2 := &PowNode{Base: &VarNode{Name: idxVar}, Exp: &RationalNode{Val: two}}
			return &MulNode{Factors: []Node{&RationalNode{Val: a}, n2}}, nil
		}
		return nil, fmt.Errorf("%s", i18n.T("rsolve.err_multiplicity_of_root_1_exceeds"))
	}

	// Case 2: Linear term c * n
	if isLinearInVar(term, idxVar) {
		kCoeff := extractVarCoeff(term, idxVar)
		p1 := big.NewRat(0, 1)
		for _, c := range coeffs {
			p1.Add(p1, c)
		}

		if p1.Sign() == 0 {
			// 1 is a root of characteristic equation (resonance)
			pPrime1 := big.NewRat(0, 1)
			for j, c := range coeffs {
				pPrime1.Add(pPrime1, new(big.Rat).Mul(big.NewRat(int64(j), 1), c))
			}
			if pPrime1.Sign() != 0 {
				two := big.NewRat(2, 1)
				twoPPrime := new(big.Rat).Mul(two, pPrime1)
				a := new(big.Rat).Quo(kCoeff, twoPPrime)

				// yp(n) = A*n^2 + B*n
				// L(A*n^2 + B*n) = A*(2*P'(1)*n + sum(j^2 * c_j)) + B*P'(1) == k*n
				// Constant term must be zero: A * sum(j^2 * c_j) + B * P'(1) = 0
				// => B = - (A * sum(j^2 * c_j)) / P'(1)
				sumJ2C := big.NewRat(0, 1)
				for j, c := range coeffs {
					jInt := int64(j)
					sumJ2C.Add(sumJ2C, new(big.Rat).Mul(big.NewRat(jInt*jInt, 1), c))
				}
				b := new(big.Rat).Quo(new(big.Rat).Mul(new(big.Rat).Neg(a), sumJ2C), pPrime1)

				n2 := &PowNode{Base: &VarNode{Name: idxVar}, Exp: &RationalNode{Val: two}}
				terms := []Node{
					&MulNode{Factors: []Node{&RationalNode{Val: a}, n2}},
				}
				if b.Sign() != 0 {
					terms = append(terms, &MulNode{Factors: []Node{&RationalNode{Val: b}, &VarNode{Name: idxVar}}})
				}
				if len(terms) == 1 {
					return terms[0], nil
				}
				return &AddNode{Terms: terms}, nil
			}
		} else {
			a := new(big.Rat).Quo(kCoeff, p1)
			pPrime1 := big.NewRat(0, 1)
			for j, c := range coeffs {
				pPrime1.Add(pPrime1, new(big.Rat).Mul(big.NewRat(int64(j), 1), c))
			}
			b := new(big.Rat).Quo(new(big.Rat).Mul(new(big.Rat).Neg(a), pPrime1), p1)
			return &AddNode{Terms: []Node{
				&MulNode{Factors: []Node{&RationalNode{Val: a}, &VarNode{Name: idxVar}}},
				&RationalNode{Val: b},
			}}, nil
		}
	}

	// Case 3: Geometric term c * r^n
	coeffR, baseR, isGeom := matchExponentialTerm(term, idxVar)
	if isGeom {
		pAtR := big.NewRat(0, 1)
		for j, c := range coeffs {
			rPowJ := ratPow(baseR, j)
			pAtR.Add(pAtR, new(big.Rat).Mul(c, rPowJ))
		}

		if pAtR.Sign() != 0 {
			a := new(big.Rat).Quo(coeffR, pAtR)
			rPowN := &PowNode{Base: &RationalNode{Val: baseR}, Exp: &VarNode{Name: idxVar}}
			if a.Cmp(big.NewRat(1, 1)) == 0 {
				return rPowN, nil
			}
			return &MulNode{Factors: []Node{&RationalNode{Val: a}, rPowN}}, nil
		}

		// Resonant: baseR is a characteristic root!
		pPrimeAtR := big.NewRat(0, 1)
		for j, c := range coeffs {
			if j > 0 {
				jRat := big.NewRat(int64(j), 1)
				rPowJMinus1 := ratPow(baseR, j-1)
				termVal := new(big.Rat).Mul(new(big.Rat).Mul(jRat, c), rPowJMinus1)
				pPrimeAtR.Add(pPrimeAtR, termVal)
			}
		}

		if pPrimeAtR.Sign() != 0 {
			rTimesPPrime := new(big.Rat).Mul(baseR, pPrimeAtR)
			a := new(big.Rat).Quo(coeffR, rTimesPPrime)
			rPowN := &PowNode{Base: &RationalNode{Val: baseR}, Exp: &VarNode{Name: idxVar}}
			return &MulNode{Factors: []Node{&RationalNode{Val: a}, &VarNode{Name: idxVar}, rPowN}}, nil
		}

		// Double resonance: P''(r)
		pDoublePrimeAtR := big.NewRat(0, 1)
		for j, c := range coeffs {
			if j > 1 {
				jRat := big.NewRat(int64(j*(j-1)), 1)
				rPowJMinus2 := ratPow(baseR, j-2)
				termVal := new(big.Rat).Mul(new(big.Rat).Mul(jRat, c), rPowJMinus2)
				pDoublePrimeAtR.Add(pDoublePrimeAtR, termVal)
			}
		}

		if pDoublePrimeAtR.Sign() != 0 {
			two := big.NewRat(2, 1)
			r2 := new(big.Rat).Mul(baseR, baseR)
			denom := new(big.Rat).Mul(new(big.Rat).Quo(pDoublePrimeAtR, two), r2)
			a := new(big.Rat).Quo(coeffR, denom)
			n2 := &PowNode{Base: &VarNode{Name: idxVar}, Exp: &RationalNode{Val: two}}
			rPowN := &PowNode{Base: &RationalNode{Val: baseR}, Exp: &VarNode{Name: idxVar}}
			return &MulNode{Factors: []Node{&RationalNode{Val: a}, n2, rPowN}}, nil
		}
	}

	return nil, fmt.Errorf("%s", i18n.T("rsolve.err_inhomogeneous_term_not_yet_supported", term.String()))
}

func ratPow(base *big.Rat, exp int) *big.Rat {
	if exp == 0 {
		return big.NewRat(1, 1)
	}
	res := big.NewRat(1, 1)
	for i := 0; i < exp; i++ {
		res.Mul(res, base)
	}
	return res
}

func isLinearInVar(n Node, varName string) bool {
	if v, ok := n.(*VarNode); ok && v.Name == varName {
		return true
	}
	if mul, ok := n.(*MulNode); ok {
		hasVar := false
		for _, f := range mul.Factors {
			if containsVariable(f, varName) {
				if hasVar {
					return false
				}
				if v, ok := f.(*VarNode); ok && v.Name == varName {
					hasVar = true
				} else {
					return false
				}
			}
		}
		return hasVar
	}
	return false
}

func extractVarCoeff(n Node, varName string) *big.Rat {
	if v, ok := n.(*VarNode); ok && v.Name == varName {
		return big.NewRat(1, 1)
	}
	if mul, ok := n.(*MulNode); ok {
		c := big.NewRat(1, 1)
		for _, f := range mul.Factors {
			if r, ok := f.(*RationalNode); ok {
				c.Mul(c, r.Val)
			}
		}
		return c
	}
	return big.NewRat(1, 1)
}

func matchExponentialTerm(n Node, varName string) (*big.Rat, *big.Rat, bool) {
	coeff := big.NewRat(1, 1)
	var base *big.Rat
	factors := collectMulFactors(n)

	for _, f := range factors {
		if r, ok := f.(*RationalNode); ok {
			coeff.Mul(coeff, r.Val)
		} else if u, ok := f.(*UnaryOpNode); ok && u.Op == "-" {
			if r, ok := u.Expr.(*RationalNode); ok {
				coeff.Mul(coeff, new(big.Rat).Neg(r.Val))
			} else {
				coeff.Mul(coeff, big.NewRat(-1, 1))
			}
		} else if pow, ok := f.(*PowNode); ok {
			if v, isV := pow.Exp.(*VarNode); isV && v.Name == varName {
				if rBase, isR := pow.Base.(*RationalNode); isR {
					base = new(big.Rat).Set(rBase.Val)
				}
			}
		}
	}

	if base != nil {
		return coeff, base, true
	}
	return nil, nil, false
}

type initialCondition struct {
	index int
	val   Node
}

func parseInitialConditions(initsNode Node, seqName string) ([]initialCondition, error) {
	if initsNode == nil {
		return nil, nil
	}
	var initEqs []Node
	if list, ok := initsNode.(*ListNode); ok {
		initEqs = list.Elements
	} else if mat, ok := initsNode.(*MatrixNode); ok {
		for _, row := range mat.Data {
			initEqs = append(initEqs, row...)
		}
	} else {
		initEqs = []Node{initsNode}
	}

	var inits []initialCondition
	for _, ieq := range initEqs {
		rel, ok := ieq.(*RelOpNode)
		if !ok || (rel.Op != "==" && rel.Op != "=") {
			continue
		}
		if fn, ok := rel.LHS.(*FuncNode); ok && fn.Name == seqName && len(fn.Args) == 1 {
			if r, ok := fn.Args[0].(*RationalNode); ok && r.Val.IsInt() {
				idx := int(r.Val.Num().Int64())
				val, _ := Eval(rel.RHS)
				inits = append(inits, initialCondition{index: idx, val: val})
			}
		}
	}

	sort.Slice(inits, func(i, j int) bool {
		return inits[i].index < inits[j].index
	})
	return inits, nil
}

// fitInitialConditions fits arbitrary constants C_1, ..., C_K using initial conditions.
// Uses exact Gaussian elimination via linear_solver.go for arbitrary order d >= 1.
func fitInitialConditions(homogBases []Node, partSol Node, idxVar string, inits []initialCondition) (Node, error) {
	order := len(homogBases)
	if len(inits) < order {
		var terms []Node
		for i, b := range homogBases {
			cVar := &VarNode{Name: fmt.Sprintf("C%d", i+1)}
			terms = append(terms, &MulNode{Factors: []Node{cVar, b}})
		}
		if partSol != nil {
			terms = append(terms, partSol)
		}
		return Eval(&AddNode{Terms: terms})
	}

	// Build linear system A * x = b of size order x order
	matData := make([][]Node, order)
	bData := make([]Node, order)

	for i := 0; i < order; i++ {
		initVal := inits[i]
		matData[i] = make([]Node, order)
		for j := 0; j < order; j++ {
			matData[i][j] = evalBaseAt(homogBases[j], idxVar, initVal.index)
		}
		pVal := evalBaseAt(partSol, idxVar, initVal.index)
		negP, _ := simplifyUnaryOp("-", pVal)
		rhs, _ := simplifyAdd([]Node{initVal.val, negP})
		bData[i], _ = Eval(rhs)
	}

	A := &MatrixNode{Rows: order, Cols: order, Data: matData}
	b := &ListNode{Elements: bData}

	solNode, err := evalSolveLinear(A, b)
	if err != nil {
		return nil, err
	}

	var cVals []Node
	if solList, ok := solNode.(*ListNode); ok {
		cVals = solList.Elements
	} else if solMat, ok := solNode.(*MatrixNode); ok {
		for r := 0; r < solMat.Rows; r++ {
			cVals = append(cVals, solMat.Data[r][0])
		}
	} else {
		return nil, fmt.Errorf("%s", fmt.Sprintf(i18n.T("rsolve.err_unexpected_linear_solver_result"), solNode))
	}

	var finalTerms []Node
	for j := 0; j < order; j++ {
		term, _ := simplifyMul([]Node{cVals[j], homogBases[j]})
		finalTerms = append(finalTerms, term)
	}
	if partSol != nil {
		finalTerms = append(finalTerms, partSol)
	}

	return Eval(&AddNode{Terms: finalTerms})
}

func evalBaseAt(b Node, varName string, val int) Node {
	sub := substituteVar(b, varName, &RationalNode{Val: big.NewRat(int64(val), 1)})
	evaled, err := Eval(sub)
	if err != nil {
		return sub
	}
	return evaled
}

func collectAddTerms(n Node) []Node {
	if add, ok := n.(*AddNode); ok {
		return add.Terms
	}
	return []Node{n}
}

func collectMulFactors(n Node) []Node {
	if mul, ok := n.(*MulNode); ok {
		return mul.Factors
	}
	return []Node{n}
}

func containsVariable(n Node, varName string) bool {
	if v, ok := n.(*VarNode); ok && v.Name == varName {
		return true
	}
	if fn, ok := n.(*FuncNode); ok {
		for _, a := range fn.Args {
			if containsVariable(a, varName) {
				return true
			}
		}
	}
	if add, ok := n.(*AddNode); ok {
		for _, t := range add.Terms {
			if containsVariable(t, varName) {
				return true
			}
		}
	}
	if mul, ok := n.(*MulNode); ok {
		for _, f := range mul.Factors {
			if containsVariable(f, varName) {
				return true
			}
		}
	}
	if pow, ok := n.(*PowNode); ok {
		return containsVariable(pow.Base, varName) || containsVariable(pow.Exp, varName)
	}
	if u, ok := n.(*UnaryOpNode); ok {
		return containsVariable(u.Expr, varName)
	}
	if rel, ok := n.(*RelOpNode); ok {
		return containsVariable(rel.LHS, varName) || containsVariable(rel.RHS, varName)
	}
	return false
}

func containsSequenceFunction(n Node, seqName string) bool {
	if fn, ok := n.(*FuncNode); ok && fn.Name == seqName {
		return true
	}
	if add, ok := n.(*AddNode); ok {
		for _, t := range add.Terms {
			if containsSequenceFunction(t, seqName) {
				return true
			}
		}
	}
	if mul, ok := n.(*MulNode); ok {
		for _, f := range mul.Factors {
			if containsSequenceFunction(f, seqName) {
				return true
			}
		}
	}
	if pow, ok := n.(*PowNode); ok {
		return containsSequenceFunction(pow.Base, seqName) || containsSequenceFunction(pow.Exp, seqName)
	}
	if u, ok := n.(*UnaryOpNode); ok {
		return containsSequenceFunction(u.Expr, seqName)
	}
	return false
}
