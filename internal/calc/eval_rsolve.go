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
	// Normalize equation LHS == RHS into LHS - RHS == 0
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
			// Terms not containing a(n+k) belong to inhomogeneous part.
			// Since diffExpr = 0, moving term to RHS negates it.
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

	// Normalize shifts so minimum shift is 0 (e.g. a(n+2) == a(n+1) -> shift 2 and shift 1 -> offset by min)
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

	// If shifts were shifted by minShift, shift index in f(n) accordingly: n -> n - minShift
	if minShift != 0 {
		fNode = substituteVar(fNode, idxVar, &AddNode{Terms: []Node{&VarNode{Name: idxVar}, &RationalNode{Val: big.NewRat(int64(-minShift), 1)}}})
	}

	fNode, _ = Eval(fNode)
	return normalizedCoeffs, fNode, order, nil
}

// matchSequenceTerm checks if a term is c * seq(idxVar + k) and extracts shift k and coefficient c.
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
				// Nonlinear term like a(n)*a(n+1) is not supported
				return 0, nil, false, fmt.Errorf("%s", i18n.T("calc.rsolve_linear_constant_coeff", t.String()))
			}
			seqFunc = fn
		} else {
			// Check if factor contains idxVar or is constant
			if containsVariable(f, idxVar) {
				return 0, nil, false, fmt.Errorf("%s", i18n.T("calc.rsolve_linear_constant_coeff", t.String()))
			}
			// Constant factor - evaluate if possible
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

	// Analyze argument of seqFunc (e.g. n, n+1, n+2, n-1)
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

// solveHomogeneousRecurrence solves the characteristic equation and returns basis functions b_i(n).
func solveHomogeneousRecurrence(coeffs map[int]*big.Rat, idxVar string, order int) ([]Node, error) {
	var bases []Node

	if order == 1 {
		// c1 * a(n+1) + c0 * a(n) = 0 => lambda = -c0 / c1
		c1, ok1 := coeffs[1]
		c0, ok0 := coeffs[0]
		if !ok1 || !ok0 || c1.Sign() == 0 {
			return nil, fmt.Errorf("%s", i18n.T("rsolve.err_invalid_1st_order_recurrence_coefficients"))
		}
		lambda := new(big.Rat).Quo(new(big.Rat).Neg(c0), c1)
		bases = append(bases, makePowerBasis(lambda, idxVar))
		return bases, nil
	}

	if order == 2 {
		// c2 * lambda^2 + c1 * lambda + c0 = 0
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

		// Discriminant D = c1^2 - 4 * c2 * c0
		disc := new(big.Rat).Mul(c1, c1)
		fourAc := new(big.Rat).Mul(big.NewRat(4, 1), new(big.Rat).Mul(c2, c0))
		disc.Sub(disc, fourAc)

		twoA := new(big.Rat).Mul(big.NewRat(2, 1), c2)

		if disc.Sign() == 0 {
			// Single root with multiplicity 2: lambda = -c1 / (2*c2)
			lambda := new(big.Rat).Quo(new(big.Rat).Neg(c1), twoA)
			b1 := makePowerBasis(lambda, idxVar)
			b2 := &MulNode{Factors: []Node{&VarNode{Name: idxVar}, b1}}
			bases = append(bases, b1, b2)
			return bases, nil
		}

		// Check if discriminant is a rational square
		sqrtDiscRat, isSquare := exactRatSqrt(disc)
		if isSquare {
			// Two distinct rational roots: lambda1,2 = (-c1 +- sqrtD) / 2A
			l1 := new(big.Rat).Quo(new(big.Rat).Add(new(big.Rat).Neg(c1), sqrtDiscRat), twoA)
			l2 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Neg(c1), sqrtDiscRat), twoA)
			bases = append(bases,
				makePowerBasis(l1, idxVar),
				makePowerBasis(l2, idxVar),
			)
			return bases, nil
		}

		// Quadratic formula with symbolic square root: (-c1 +- sqrt(disc)) / 2c2
		negC1Div2A := new(big.Rat).Quo(new(big.Rat).Neg(c1), twoA)
		oneDiv2A := new(big.Rat).Quo(big.NewRat(1, 1), twoA)

		sqrtNode := &SqrtNode{Radicand: &RationalNode{Val: disc}}
		coeffSqrt := &RationalNode{Val: oneDiv2A}
		termSqrt := &MulNode{Factors: []Node{coeffSqrt, sqrtNode}}

		root1 := &AddNode{Terms: []Node{&RationalNode{Val: negC1Div2A}, termSqrt}}
		root2 := &AddNode{Terms: []Node{&RationalNode{Val: negC1Div2A}, &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, termSqrt}}}}

		r1Eval, _ := Eval(root1)
		r2Eval, _ := Eval(root2)

		bases = append(bases,
			&PowNode{Base: r1Eval, Exp: &VarNode{Name: idxVar}},
			&PowNode{Base: r2Eval, Exp: &VarNode{Name: idxVar}},
		)
		return bases, nil
	}

	return nil, fmt.Errorf("%s", i18n.T("rsolve.err_recurrence_order", order))
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

	// Decompose f(n) into terms if AddNode
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

// solveSingleParticularTerm handles constant, polynomial in n, or geometric r^n.
func solveSingleParticularTerm(coeffs map[int]*big.Rat, term Node, idxVar string, order int) (Node, error) {
	// Case 1: Constant term C
	if r, ok := term.(*RationalNode); ok {
		// Sum of coefficients P(1) = sum c_j * 1^j
		p1 := big.NewRat(0, 1)
		for _, c := range coeffs {
			p1.Add(p1, c)
		}
		if p1.Sign() != 0 {
			// a_p = C / P(1)
			a := new(big.Rat).Quo(r.Val, p1)
			return &RationalNode{Val: a}, nil
		}
		// If 1 is a root of characteristic equation: try A * n
		// sum c_j * (n + j) = n * sum c_j + sum (j * c_j) = sum (j * c_j)
		pPrime1 := big.NewRat(0, 1)
		for j, c := range coeffs {
			pPrime1.Add(pPrime1, new(big.Rat).Mul(big.NewRat(int64(j), 1), c))
		}
		if pPrime1.Sign() != 0 {
			a := new(big.Rat).Quo(r.Val, pPrime1)
			return &MulNode{Factors: []Node{&RationalNode{Val: a}, &VarNode{Name: idxVar}}}, nil
		}
		return nil, fmt.Errorf("%s", i18n.T("rsolve.err_multiplicity_of_root_1_exceeds"))
	}

	// Case 2: Linear term c * n
	if isLinearInVar(term, idxVar) {
		kCoeff := extractVarCoeff(term, idxVar)
		// Try A * n^2 + B * n or A * n + B
		p1 := big.NewRat(0, 1)
		for _, c := range coeffs {
			p1.Add(p1, c)
		}

		if p1.Sign() == 0 {
			// 1 is a root of characteristic equation (e.g. a(n+1) = a(n) + n)
			// Try A * n^2 + B * n
			// c_1 * (A*(n+1)^2 + B*(n+1)) + c_0 * (A*n^2 + B*n) = k * n
			// For a(n+1) - a(n) = n => A(2n+1) + B = 2A*n + (A+B) = k*n
			// => 2A = k => A = k/2, A+B = 0 => B = -k/2
			if order == 1 {
				// c1 * (A(n+1)^2 + B(n+1)) + c0 * (An^2 + Bn)
				// = n^2(c1*A + c0*A) + n(2*c1*A + c1*B + c0*B) + (c1*A + c1*B)
				// Since c1+c0 = 0 (p1=0), n^2 term vanishes.
				// n term: 2*c1*A = k => A = k / (2*c1)
				// const term: c1*A + c1*B = 0 => B = -A
				c1 := coeffs[1]
				twoC1 := new(big.Rat).Mul(big.NewRat(2, 1), c1)
				a := new(big.Rat).Quo(kCoeff, twoC1)
				b := new(big.Rat).Neg(a)

				res := &AddNode{Terms: []Node{
					&MulNode{Factors: []Node{&RationalNode{Val: a}, &PowNode{Base: &VarNode{Name: idxVar}, Exp: &RationalNode{Val: big.NewRat(2, 1)}}}},
					&MulNode{Factors: []Node{&RationalNode{Val: b}, &VarNode{Name: idxVar}}},
				}}
				return Eval(res)
			}
		} else {
			// 1 is NOT a root: Try A * n + B
			// sum c_j (A(n+j) + B) = n (A sum c_j) + (A sum j*c_j + B sum c_j) = k * n
			// A = k / p1
			// B = - A * (sum j*c_j) / p1
			a := new(big.Rat).Quo(kCoeff, p1)
			jCj := big.NewRat(0, 1)
			for j, c := range coeffs {
				jCj.Add(jCj, new(big.Rat).Mul(big.NewRat(int64(j), 1), c))
			}
			b := new(big.Rat).Quo(new(big.Rat).Mul(new(big.Rat).Neg(a), jCj), p1)

			res := &AddNode{Terms: []Node{
				&MulNode{Factors: []Node{&RationalNode{Val: a}, &VarNode{Name: idxVar}}},
				&RationalNode{Val: b},
			}}
			return Eval(res)
		}
	}

	// Case 3: Exponential term C * r^n
	rBase, cMult, isExp := matchExponentialTerm(term, idxVar)
	if isExp {
		// Evaluate characteristic polynomial at r: P(r) = sum c_j * r^j
		pr := big.NewRat(0, 1)
		for j, c := range coeffs {
			rPow := ratPow(rBase, j)
			pr.Add(pr, new(big.Rat).Mul(c, rPow))
		}
		if pr.Sign() != 0 {
			// A = C / P(r)
			a := new(big.Rat).Quo(cMult, pr)
			res := &MulNode{Factors: []Node{
				&RationalNode{Val: a},
				&PowNode{Base: &RationalNode{Val: rBase}, Exp: &VarNode{Name: idxVar}},
			}}
			return Eval(res)
		}
	}

	return nil, fmt.Errorf("%s", i18n.T("rsolve.err_inhomogeneous_term_not_yet_supported", term.String()))
}

func ratPow(base *big.Rat, exp int) *big.Rat {
	res := big.NewRat(1, 1)
	for i := 0; i < exp; i++ {
		res.Mul(res, base)
	}
	return res
}

func isLinearInVar(n Node, varName string) bool {
	factors := collectMulFactors(n)
	count := 0
	for _, f := range factors {
		if v, ok := f.(*VarNode); ok && v.Name == varName {
			count++
		} else if containsVariable(f, varName) {
			return false
		}
	}
	return count == 1
}

func extractVarCoeff(n Node, varName string) *big.Rat {
	factors := collectMulFactors(n)
	coeff := big.NewRat(1, 1)
	for _, f := range factors {
		if v, ok := f.(*VarNode); ok && v.Name == varName {
			continue
		}
		if r, ok := f.(*RationalNode); ok {
			coeff.Mul(coeff, r.Val)
		}
	}
	return coeff
}

func matchExponentialTerm(n Node, varName string) (*big.Rat, *big.Rat, bool) {
	factors := collectMulFactors(n)
	coeff := big.NewRat(1, 1)
	var expBase *big.Rat

	for _, f := range factors {
		if r, ok := f.(*RationalNode); ok {
			coeff.Mul(coeff, r.Val)
		} else if pow, ok := f.(*PowNode); ok {
			if v, ok := pow.Exp.(*VarNode); ok && v.Name == varName {
				if rb, ok := pow.Base.(*RationalNode); ok {
					expBase = rb.Val
				}
			}
		}
	}
	if expBase != nil {
		return expBase, coeff, true
	}
	return nil, nil, false
}

// initialCondition stores a(n_0) = val.
type initialCondition struct {
	index int
	val   Node
}

func parseInitialConditions(initsNode Node, seqName string) ([]initialCondition, error) {
	if initsNode == nil {
		return nil, nil
	}
	var condNodes []Node
	if list, ok := initsNode.(*ListNode); ok {
		condNodes = append(condNodes, list.Elements...)
	} else if matrix, ok := initsNode.(*MatrixNode); ok {
		for _, row := range matrix.Data {
			condNodes = append(condNodes, row...)
		}
	} else if rel, ok := initsNode.(*RelOpNode); ok {
		condNodes = []Node{rel}
	}

	var inits []initialCondition
	for _, cn := range condNodes {
		rel, ok := cn.(*RelOpNode)
		if !ok || (rel.Op != "==" && rel.Op != "=") {
			continue
		}
		// LHS should be a(k)
		fn, ok := rel.LHS.(*FuncNode)
		if !ok || fn.Name != seqName || len(fn.Args) != 1 {
			continue
		}
		r, ok := fn.Args[0].(*RationalNode)
		if !ok || !r.Val.IsInt() {
			continue
		}
		idx := int(r.Val.Num().Int64())
		val, _ := Eval(rel.RHS)
		inits = append(inits, initialCondition{index: idx, val: val})
	}

	sort.Slice(inits, func(i, j int) bool {
		return inits[i].index < inits[j].index
	})
	return inits, nil
}

// fitInitialConditions fits arbitrary constants C_1, ..., C_K using initial conditions.
func fitInitialConditions(homogBases []Node, partSol Node, idxVar string, inits []initialCondition) (Node, error) {
	order := len(homogBases)
	if len(inits) < order {
		// Not enough initial conditions -> return general solution with symbolic C1, C2...
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

	// We have at least 'order' conditions. Build linear system:
	// sum_j C_j * b_j(n_i) = val_i - a_p(n_i)
	// For order = 1:
	if order == 1 {
		init0 := inits[0]
		b0 := substituteVar(homogBases[0], idxVar, &RationalNode{Val: big.NewRat(int64(init0.index), 1)})
		b0Eval, err := Eval(b0)
		if err != nil {
			return nil, err
		}
		p0 := substituteVar(partSol, idxVar, &RationalNode{Val: big.NewRat(int64(init0.index), 1)})
		p0Eval, _ := Eval(p0)

		// RHS = val_0 - p0Eval
		negP0, _ := simplifyUnaryOp("-", p0Eval)
		rhs, err := simplifyAdd([]Node{init0.val, negP0})
		if err != nil {
			return nil, err
		}
		rhsEval, _ := Eval(rhs)

		// C1 = RHS / b0Eval
		invB0, err := simplifyPow(b0Eval, &RationalNode{Val: big.NewRat(-1, 1)})
		if err != nil {
			return nil, err
		}
		c1, err := simplifyMul([]Node{rhsEval, invB0})
		if err != nil {
			return nil, err
		}
		c1Eval, _ := Eval(c1)

		finalSol := &AddNode{Terms: []Node{
			&MulNode{Factors: []Node{c1Eval, homogBases[0]}},
			partSol,
		}}
		return Eval(finalSol)
	}

	// For order = 2:
	if order == 2 {
		i0 := inits[0]
		i1 := inits[1]

		// b_j at n0 and n1
		b00 := evalBaseAt(homogBases[0], idxVar, i0.index)
		b01 := evalBaseAt(homogBases[1], idxVar, i0.index)
		b10 := evalBaseAt(homogBases[0], idxVar, i1.index)
		b11 := evalBaseAt(homogBases[1], idxVar, i1.index)

		p0 := evalBaseAt(partSol, idxVar, i0.index)
		p1 := evalBaseAt(partSol, idxVar, i1.index)

		rhs0, _ := Eval(&AddNode{Terms: []Node{i0.val, &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, p0}}}})
		rhs1, _ := Eval(&AddNode{Terms: []Node{i1.val, &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, p1}}}})

		// Determinant Det = b00 * b11 - b01 * b10
		t1, _ := Eval(&MulNode{Factors: []Node{b00, b11}})
		t2, _ := Eval(&MulNode{Factors: []Node{b01, b10}})
		det, err := Eval(&AddNode{Terms: []Node{t1, &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, t2}}}})
		if err != nil {
			return nil, err
		}

		invDet, err := simplifyPow(det, &RationalNode{Val: big.NewRat(-1, 1)})
		if err != nil {
			return nil, err
		}

		// C1 = (rhs0 * b11 - b01 * rhs1) / Det
		num1Part1, _ := Eval(&MulNode{Factors: []Node{rhs0, b11}})
		num1Part2, _ := Eval(&MulNode{Factors: []Node{b01, rhs1}})
		num1, _ := Eval(&AddNode{Terms: []Node{num1Part1, &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, num1Part2}}}})
		c1, _ := Eval(&MulNode{Factors: []Node{num1, invDet}})

		// C2 = (b00 * rhs1 - rhs0 * b10) / Det
		num2Part1, _ := Eval(&MulNode{Factors: []Node{b00, rhs1}})
		num2Part2, _ := Eval(&MulNode{Factors: []Node{rhs0, b10}})
		num2, _ := Eval(&AddNode{Terms: []Node{num2Part1, &MulNode{Factors: []Node{&RationalNode{Val: big.NewRat(-1, 1)}, num2Part2}}}})
		c2, _ := Eval(&MulNode{Factors: []Node{num2, invDet}})

		finalSol := &AddNode{Terms: []Node{
			&MulNode{Factors: []Node{c1, homogBases[0]}},
			&MulNode{Factors: []Node{c2, homogBases[1]}},
			partSol,
		}}
		return Eval(finalSol)
	}

	return nil, fmt.Errorf("%s", i18n.T("rsolve.err_initial_condition_fitting_for_order", order))
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
	if a, ok := n.(*AddNode); ok {
		return a.Terms
	}
	return []Node{n}
}

func collectMulFactors(n Node) []Node {
	if m, ok := n.(*MulNode); ok {
		return m.Factors
	}
	return []Node{n}
}

func containsVariable(n Node, varName string) bool {
	if n == nil {
		return false
	}
	found := false
	var walk func(curr Node)
	walk = func(curr Node) {
		if curr == nil || found {
			return
		}
		switch v := curr.(type) {
		case *VarNode:
			if v.Name == varName {
				found = true
			}
		case *AddNode:
			for _, t := range v.Terms {
				walk(t)
			}
		case *MulNode:
			for _, f := range v.Factors {
				walk(f)
			}
		case *PowNode:
			walk(v.Base)
			walk(v.Exp)
		case *UnaryOpNode:
			walk(v.Expr)
		case *FuncNode:
			for _, a := range v.Args {
				walk(a)
			}
		case *SqrtNode:
			walk(v.Radicand)
		}
	}
	walk(n)
	return found
}

func containsSequenceFunction(n Node, seqName string) bool {
	if n == nil {
		return false
	}
	found := false
	var walk func(curr Node)
	walk = func(curr Node) {
		if curr == nil || found {
			return
		}
		switch v := curr.(type) {
		case *FuncNode:
			if v.Name == seqName {
				found = true
				return
			}
			for _, a := range v.Args {
				walk(a)
			}
		case *AddNode:
			for _, t := range v.Terms {
				walk(t)
			}
		case *MulNode:
			for _, f := range v.Factors {
				walk(f)
			}
		case *PowNode:
			walk(v.Base)
			walk(v.Exp)
		case *UnaryOpNode:
			walk(v.Expr)
		case *SqrtNode:
			walk(v.Radicand)
		}
	}
	walk(n)
	return found
}

