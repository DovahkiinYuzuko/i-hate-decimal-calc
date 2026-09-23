package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterLazyHandler("qe", func(args []Node, env *Env) (Node, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("%s", i18n.T("qe.err_invalid_args"))
		}
		arg := args[0]
		if fn, ok := arg.(*FuncNode); ok {
			if fn.Name == "forall" || fn.Name == "exists" {
				q, err := parseQuantifierFromFunc(fn)
				if err != nil {
					return nil, err
				}
				arg = q
			}
		}
		return EvalQE(arg, env)
	})

	RegisterLazyHandler("forall", func(args []Node, env *Env) (Node, error) {
		return buildQuantifierFromArgs(QuantifierForall, args)
	})

	RegisterLazyHandler("exists", func(args []Node, env *Env) (Node, error) {
		return buildQuantifierFromArgs(QuantifierExists, args)
	})
}

func parseQuantifierFromFunc(fn *FuncNode) (*QuantifierNode, error) {
	kind := QuantifierForall
	if fn.Name == "exists" {
		kind = QuantifierExists
	}
	return buildQuantifierFromArgs(kind, fn.Args)
}

func buildQuantifierFromArgs(kind QuantifierKind, args []Node) (*QuantifierNode, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("qe.err_invalid_args"))
	}
	var vars []string
	if list, ok := args[0].(*ListNode); ok {
		for _, elem := range list.Elements {
			if v, ok := elem.(*VarNode); ok {
				vars = append(vars, v.Name)
			} else {
				return nil, fmt.Errorf("%s: %s", i18n.T("qe.err_expected_var_list"), elem)
			}
		}
	} else if v, ok := args[0].(*VarNode); ok {
		vars = append(vars, v.Name)
	} else {
		return nil, fmt.Errorf("%s: %s", i18n.T("qe.err_expected_var_list"), args[0])
	}
	return NewQuantifier(kind, vars, args[1]), nil
}

// EvalQE evaluates the quantifier elimination of a first-order formula.
// Syntax: qe(formula)
func EvalQE(formula Node, env *Env) (Node, error) {
	if formula == nil {
		return nil, fmt.Errorf("%s", i18n.T("qe.err_nil_formula"))
	}

	// If argument is a FuncNode (forall/exists), convert it
	if fn, ok := formula.(*FuncNode); ok {
		if fn.Name == "forall" || fn.Name == "exists" {
			q, err := parseQuantifierFromFunc(fn)
			if err != nil {
				return nil, err
			}
			formula = q
		}
	}

	fsm := NewQeLifecycleFSM()

	// 1. Check if formula is a QuantifierNode
	qNode, ok := formula.(*QuantifierNode)
	if !ok {
		// If formula is already quantifier-free, just simplify/evaluate it
		return EvalWithEnv(formula, env)
	}

	_ = fsm.TransitionTo(QeStatePrenexNormalized)
	return EliminateQuantifiers(qNode, env, fsm)
}

// EliminateQuantifiers is the core QE pipeline implementing CAD-based elimination (Collins 1975, Hong 1992).
func EliminateQuantifiers(q *QuantifierNode, env *Env, fsm *QeLifecycleFSM) (Node, error) {
	if q == nil || len(q.Vars) == 0 {
		_ = fsm.TransitionTo(QeStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("qe.err_no_variables"))
	}

	// 1. Identify free variables (parameters)
	freeVars := ExtractFreeVariables(q)

	_ = fsm.TransitionTo(QeStateVariableOrdered)

	// Case A: Closed formula (no free variables / parameters)
	if len(freeVars) == 0 {
		res, err := eliminateClosedQuantifier(q, env)
		if err == nil {
			if fsm.CurrentState() == QeStateVariableOrdered {
				_ = fsm.TransitionTo(QeStateCadDecomposed)
				_ = fsm.TransitionTo(QeStateTruthEvaluated)
				_ = fsm.TransitionTo(QeStateFormulaConstructed)
			}
			return res, nil
		}
		// Fallback to general CAD QE for closed formulas (handles multi-var, compound, nested)
		return eliminateQuantifiersCAD(q, freeVars, env, fsm)
	}

	// Case B: Parametric formula (1 or more free variables)
	res, err := eliminateParametricQuantifier(q, freeVars, env, fsm)
	if err != nil {
		_ = fsm.TransitionTo(QeStateFailed)
		return nil, err
	}
	if fsm.CurrentState() == QeStateTruthEvaluated {
		_ = fsm.TransitionTo(QeStateFormulaConstructed)
	}
	return res, nil
}

// eliminateClosedQuantifier decides sentence truth when there are no free variables.
func eliminateClosedQuantifier(q *QuantifierNode, env *Env) (Node, error) {
	// If multi-variable quantifier (e.g. forall([x, y], ...)), process outermost first
	body := q.Body
	if len(q.Vars) > 1 {
		inner := NewQuantifier(q.Kind, q.Vars[1:], body)
		q = NewQuantifier(q.Kind, []string{q.Vars[0]}, inner)
	}

	v := q.Vars[0]

	// Check if body is an equation or inequality
	relOp, ok := q.Body.(*RelOpNode)
	if !ok {
		// Try evaluating body
		evaled, err := EvalWithEnv(q.Body, env)
		if err == nil {
			if r, ok := evaled.(*RelOpNode); ok {
				relOp = r
			}
		}
	}

	if relOp == nil {
		return nil, fmt.Errorf("%s: %s", i18n.T("qe.err_unsupported_body"), q.Body)
	}

	// Solve the relation for variable v
	if relOp.Op == "==" {
		// Equation: f(v) == 0
		negR, _ := simplifyUnaryOp("-", relOp.RHS)
		zeroExpr, err := simplifyAdd([]Node{relOp.LHS, negR})
		if err != nil {
			return nil, err
		}
		roots, err := solveExactRoots(zeroExpr, v)
		if err != nil {
			// Fallback to Sturm real root isolation
			if rootsSturm, errSturm := EvalIsolateRoots(zeroExpr, v, nil, nil, env); errSturm == nil {
				if rList, ok := rootsSturm.(*ListNode); ok {
					hasRealRoot := len(rList.Elements) > 0
					if q.Kind == QuantifierExists {
						return &VarNode{Name: boolToString(hasRealRoot)}, nil
					}
					// Forall x (f(x) == 0) is false unless f is identically zero
					isZeroPoly := isZero(zeroExpr)
					return &VarNode{Name: boolToString(isZeroPoly)}, nil
				}
			}
			return nil, err
		}

		hasRealRoot := len(roots) > 0
		if q.Kind == QuantifierExists {
			return &VarNode{Name: boolToString(hasRealRoot)}, nil
		}
		// Forall x (f(x) == 0)
		return &VarNode{Name: "false"}, nil
	}

	// Inequality: <, <=, >, >=, !=
	sol, err := SolveInequality(relOp, v, env)
	if err != nil {
		return nil, err
	}

	if vNode, ok := sol.(*VarNode); ok {
		if vNode.Name == "true" {
			// Solution is all of R
			if q.Kind == QuantifierForall || q.Kind == QuantifierExists {
				return &VarNode{Name: "true"}, nil
			}
		}
		if vNode.Name == "false" {
			// Solution is empty
			return &VarNode{Name: "false"}, nil
		}
	}

	// Check intervals for full R: [[-inf, inf]]
	if list, ok := sol.(*ListNode); ok {
		if isAllRealLine(list) {
			return &VarNode{Name: "true"}, nil
		}
		if len(list.Elements) > 0 {
			if q.Kind == QuantifierExists {
				return &VarNode{Name: "true"}, nil
			}
			// Forall requires ALL of R, but we only have a subset
			return &VarNode{Name: "false"}, nil
		}
	}

	if q.Kind == QuantifierForall {
		return &VarNode{Name: "false"}, nil
	}
	return &VarNode{Name: "false"}, nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func isAllRealLine(list *ListNode) bool {
	if list == nil || len(list.Elements) == 0 {
		return false
	}
	if len(list.Elements) == 1 {
		pair, ok := list.Elements[0].(*ListNode)
		if !ok || len(pair.Elements) != 2 {
			return false
		}
		l, okL := pair.Elements[0].(*VarNode)
		r, okR := pair.Elements[1].(*VarNode)
		return okL && okR && l.Name == "-inf" && r.Name == "inf"
	}
	// Check if chained intervals cover [-inf, inf] (e.g. [[-inf, 0], [0, inf]])
	firstPair, ok1 := list.Elements[0].(*ListNode)
	lastPair, ok2 := list.Elements[len(list.Elements)-1].(*ListNode)
	if !ok1 || !ok2 || len(firstPair.Elements) != 2 || len(lastPair.Elements) != 2 {
		return false
	}
	l0, okL0 := firstPair.Elements[0].(*VarNode)
	rLast, okRLast := lastPair.Elements[1].(*VarNode)
	if !okL0 || !okRLast || l0.Name != "-inf" || rLast.Name != "inf" {
		return false
	}
	// Check continuity between intervals
	for i := 0; i < len(list.Elements)-1; i++ {
		p1, okP1 := list.Elements[i].(*ListNode)
		p2, okP2 := list.Elements[i+1].(*ListNode)
		if !okP1 || !okP2 || len(p1.Elements) != 2 || len(p2.Elements) != 2 {
			return false
		}
		if !p1.Elements[1].Equal(p2.Elements[0]) {
			return false
		}
	}
	return true
}

func makeRelOp(lhs Node, op string, rhs Node, env *Env) Node {
	rel := NewRelOp(lhs, op, rhs)
	evaled, err := EvalWithEnv(rel, env)
	if err == nil {
		return evaled
	}
	return rel
}

// eliminateParametricQuantifier derives quantifier-free conditions on free variables.
func eliminateParametricQuantifier(q *QuantifierNode, freeVars []string, env *Env, fsm *QeLifecycleFSM) (Node, error) {
	if len(q.Vars) > 1 {
		return eliminateQuantifiersCAD(q, freeVars, env, fsm)
	}

	relOp, ok := q.Body.(*RelOpNode)
	if !ok {
		evaled, err := EvalWithEnv(q.Body, env)
		if err == nil {
			if r, ok := evaled.(*RelOpNode); ok {
				relOp = r
			}
		}
	}
	if relOp == nil {
		return eliminateQuantifiersCAD(q, freeVars, env, fsm)
	}

	// Move everything to LHS: f(x, params) op 0
	negR, err := simplifyUnaryOp("-", relOp.RHS)
	if err != nil {
		return nil, err
	}
	zeroExpr, err := simplifyAdd([]Node{relOp.LHS, negR})
	if err != nil {
		return nil, err
	}
	zeroExpr = expandNode(zeroExpr)

	boundVar := q.Vars[0]

	// 1. Polynomial check in boundVar
	p, ok := extractPoly(zeroExpr, boundVar)
	if !ok {
		return eliminateQuantifiersCAD(q, freeVars, env, fsm)
	}

	_ = fsm.TransitionTo(QeStateCadDecomposed)

	// Hong (1992) / Brown (2001) Boundary Polynomial Analysis:
	// Degree 1: A*x + B op 0
	if p.degree() == 1 {
		lead := p.leadCoeff()   // A
		constant := p.coeffs[0] // B

		if q.Kind == QuantifierForall {
			// For all x, A*x + B > 0 (or >= 0) is impossible unless A == 0
			// If A == 0: B op 0
			condA := NewRelOp(lead, "==", mustRational(0, 1))
			condB := NewRelOp(constant, relOp.Op, mustRational(0, 1))
			_ = fsm.TransitionTo(QeStateTruthEvaluated)
			return mustFunc("and", condA, condB), nil
		}
		if q.Kind == QuantifierExists {
			// Exists x, A*x + B op 0 is true if A != 0, or (A == 0 and B op 0)
			condA := NewRelOp(lead, "!=", mustRational(0, 1))
			condA0 := NewRelOp(lead, "==", mustRational(0, 1))
			condB := NewRelOp(constant, relOp.Op, mustRational(0, 1))
			_ = fsm.TransitionTo(QeStateTruthEvaluated)
			return mustFunc("or", condA, mustFunc("and", condA0, condB)), nil
		}
	}

	// Degree 2: A*x^2 + B*x + C op 0
	if p.degree() == 2 {
		lead := p.leadCoeff() // A
		var bCoeff Node = mustRational(0, 1)
		if len(p.coeffs) > 1 {
			bCoeff = p.coeffs[1] // B
		}
		cCoeff := p.coeffs[0] // C

		// Compute Discriminant: D = B^2 - 4*A*C
		bSquared, err := simplifyPow(bCoeff, mustRational(2, 1))
		if err != nil {
			return nil, err
		}
		fourAC, err := simplifyMul([]Node{mustRational(4, 1), lead, cCoeff})
		if err != nil {
			return nil, err
		}
		negFourAC, err := simplifyUnaryOp("-", fourAC)
		if err != nil {
			return nil, err
		}
		disc, err := simplifyAdd([]Node{bSquared, negFourAC})
		if err != nil {
			return nil, err
		}
		disc = expandNode(disc)
		if evaledDisc, err := EvalWithEnv(disc, env); err == nil {
			disc = evaledDisc
		}

		_ = fsm.TransitionTo(QeStateTruthEvaluated)

		// Forall x, A*x^2 + B*x + C > 0
		// Condition: (A > 0 and D < 0)
		if q.Kind == QuantifierForall {
			switch relOp.Op {
			case ">":
				condD := makeRelOp(disc, "<", mustRational(0, 1), env)
				if isPositiveConst(lead) {
					return condD, nil
				}
				condA := makeRelOp(lead, ">", mustRational(0, 1), env)
				return mustFunc("and", condA, condD), nil
			case ">=":
				condD := makeRelOp(disc, "<=", mustRational(0, 1), env)
				if isPositiveConst(lead) {
					return condD, nil
				}
				condA := makeRelOp(lead, ">", mustRational(0, 1), env)
				return mustFunc("and", condA, condD), nil
			case "<":
				condD := makeRelOp(disc, "<", mustRational(0, 1), env)
				if isNegativeConst(lead) {
					return condD, nil
				}
				condA := makeRelOp(lead, "<", mustRational(0, 1), env)
				return mustFunc("and", condA, condD), nil
			case "<=":
				condD := makeRelOp(disc, "<=", mustRational(0, 1), env)
				if isNegativeConst(lead) {
					return condD, nil
				}
				condA := makeRelOp(lead, "<", mustRational(0, 1), env)
				return mustFunc("and", condA, condD), nil
			}
		}

		// Exists x, A*x^2 + B*x + C == 0
		// Condition: (A != 0 and D >= 0) or (A == 0 and B != 0) or (A == 0 and B == 0 and C == 0)
		if q.Kind == QuantifierExists {
			switch relOp.Op {
			case "==":
				condD := makeRelOp(disc, ">=", mustRational(0, 1), env)
				if isNonZeroConst(lead) {
					return condD, nil
				}
				condA := makeRelOp(lead, "!=", mustRational(0, 1), env)
				case1 := mustFunc("and", condA, condD)
				case2 := mustFunc("and",
					makeRelOp(lead, "==", mustRational(0, 1), env),
					makeRelOp(bCoeff, "!=", mustRational(0, 1), env),
				)
				return mustFunc("or", case1, case2), nil
			case ">", ">=":
				// Exists x (f(x) > 0) is true unless (A < 0 and D <= 0) or identically <= 0
				condD := makeRelOp(disc, ">", mustRational(0, 1), env)
				if isPositiveConst(lead) {
					return &VarNode{Name: "true"}, nil
				}
				return condD, nil
			}
		}
	}

	// Higher degree or general CAD fallback
	return eliminateQuantifiersCAD(q, freeVars, env, fsm)
}

type quantifierBlock struct {
	kind QuantifierKind
	vars []string
}

// flattenQuantifierBlocks decomposes nested and multi-variable quantifiers into an ordered list of blocks.
func flattenQuantifierBlocks(q *QuantifierNode) ([]quantifierBlock, Node) {
	var blocks []quantifierBlock
	curr := q
	for {
		blocks = append(blocks, quantifierBlock{
			kind: curr.Kind,
			vars: curr.Vars,
		})
		if nextQ, ok := curr.Body.(*QuantifierNode); ok {
			curr = nextQ
			continue
		}
		if fn, ok := curr.Body.(*FuncNode); ok && (fn.Name == "forall" || fn.Name == "exists") {
			if parsedQ, err := parseQuantifierFromFunc(fn); err == nil {
				curr = parsedQ
				continue
			}
		}
		return blocks, curr.Body
	}
}

// eliminateQuantifiersCAD performs general Quantifier Elimination via Cylindrical Algebraic Decomposition
// (Collins 1975, Hong 1992, Brown 2001).
func eliminateQuantifiersCAD(q *QuantifierNode, freeVars []string, env *Env, fsm *QeLifecycleFSM) (Node, error) {
	blocks, body := flattenQuantifierBlocks(q)

	var allBoundVars []string
	varQuantifier := make(map[string]QuantifierKind)
	for _, b := range blocks {
		for _, v := range b.vars {
			allBoundVars = append(allBoundVars, v)
			varQuantifier[v] = b.kind
		}
	}

	cadVars := append(append([]string{}, freeVars...), allBoundVars...)
	m := len(freeVars)
	n := len(cadVars)

	// Extract atomic polynomials
	_, polys, err := extractAtomicRelationsAndPolys(body, env)
	if err != nil {
		return nil, err
	}

	if len(polys) == 0 {
		evaled, err := EvalWithEnv(body, env)
		if err == nil {
			if b, ok := evaled.(*VarNode); ok && (b.Name == "true" || b.Name == "false") {
				if fsm.CurrentState() == QeStateVariableOrdered {
					_ = fsm.TransitionTo(QeStateCadDecomposed)
					_ = fsm.TransitionTo(QeStateTruthEvaluated)
					_ = fsm.TransitionTo(QeStateFormulaConstructed)
				}
				return b, nil
			}
		}
	}

	if fsm.CurrentState() == QeStateVariableOrdered {
		_ = fsm.TransitionTo(QeStateCadDecomposed)
	}

	cells, err := CADDecomposeCells(polys, cadVars, env)
	if err != nil {
		return nil, err
	}

	if fsm.CurrentState() == QeStateCadDecomposed {
		_ = fsm.TransitionTo(QeStateTruthEvaluated)
	}

	// Evaluate formula truth on leaf cells
	for i := range cells {
		sat, err := evaluateFormulaOnCell(body, &cells[i], cadVars, env)
		if err != nil {
			return nil, err
		}
		cells[i].Satisfied = sat
	}

	// Cylinder Truth Reduction from level n down to level m
	currentCells := make([]*CadCell, len(cells))
	for i := range cells {
		currentCells[i] = &cells[i]
	}

	for level := n; level > m; level-- {
		boundVar := cadVars[level-1]
		qKind := varQuantifier[boundVar]

		if level == 1 {
			// Closed sentence with 1 bound variable (Parent == nil)
			if qKind == QuantifierExists {
				for _, c := range currentCells {
					if c.Satisfied {
						if fsm.CurrentState() == QeStateTruthEvaluated {
							_ = fsm.TransitionTo(QeStateFormulaConstructed)
						}
						return &VarNode{Name: "true"}, nil
					}
				}
				if fsm.CurrentState() == QeStateTruthEvaluated {
					_ = fsm.TransitionTo(QeStateFormulaConstructed)
				}
				return &VarNode{Name: "false"}, nil
			}
			// QuantifierForall
			for _, c := range currentCells {
				if !c.Satisfied {
					if fsm.CurrentState() == QeStateTruthEvaluated {
						_ = fsm.TransitionTo(QeStateFormulaConstructed)
					}
					return &VarNode{Name: "false"}, nil
				}
			}
			if fsm.CurrentState() == QeStateTruthEvaluated {
				_ = fsm.TransitionTo(QeStateFormulaConstructed)
			}
			return &VarNode{Name: "true"}, nil
		}

		var parentList []*CadCell
		childrenMap := make(map[*CadCell][]*CadCell)
		for _, c := range currentCells {
			p := c.Parent
			if _, exists := childrenMap[p]; !exists {
				parentList = append(parentList, p)
			}
			childrenMap[p] = append(childrenMap[p], c)
		}

		for _, p := range parentList {
			children := childrenMap[p]
			if qKind == QuantifierExists {
				p.Satisfied = false
				for _, child := range children {
					if child.Satisfied {
						p.Satisfied = true
						break
					}
				}
			} else {
				p.Satisfied = true
				for _, child := range children {
					if !child.Satisfied {
						p.Satisfied = false
						break
					}
				}
			}
		}

		currentCells = parentList
	}

	if m == 0 {
		return &VarNode{Name: "true"}, nil
	}

	if m == 1 {
		res, err := reconstruct1DQuantifierFreeFormula(cadVars[0], currentCells)
		if err != nil {
			return nil, err
		}
		if fsm.CurrentState() == QeStateTruthEvaluated {
			_ = fsm.TransitionTo(QeStateFormulaConstructed)
		}
		return res, nil
	}

	allSat := true
	noneSat := true
	var samples []Node
	for _, c := range currentCells {
		if c.Satisfied {
			noneSat = false
			samples = append(samples, &ListNode{Elements: c.SamplePoint})
		} else {
			allSat = false
		}
	}

	if fsm.CurrentState() == QeStateTruthEvaluated {
		_ = fsm.TransitionTo(QeStateFormulaConstructed)
	}

	if allSat {
		return &VarNode{Name: "true"}, nil
	}
	if noneSat {
		return &VarNode{Name: "false"}, nil
	}
	return &ListNode{Elements: samples}, nil
}

// reconstruct1DQuantifierFreeFormula constructs a quantifier-free formula for a single parameter
// from the truth values of the base cylindrical cells.
func reconstruct1DQuantifierFreeFormula(varName string, baseCells []*CadCell) (Node, error) {
	if len(baseCells) == 0 {
		return &VarNode{Name: "false"}, nil
	}

	allSatisfied := true
	noneSatisfied := true
	for _, c := range baseCells {
		if c.Satisfied {
			noneSatisfied = false
		} else {
			allSatisfied = false
		}
	}

	if allSatisfied {
		return &VarNode{Name: "true"}, nil
	}
	if noneSatisfied {
		return &VarNode{Name: "false"}, nil
	}

	var intervals [][]int
	i := 0
	for i < len(baseCells) {
		if !baseCells[i].Satisfied {
			i++
			continue
		}
		start := i
		for i < len(baseCells) && baseCells[i].Satisfied {
			i++
		}
		end := i - 1
		intervals = append(intervals, []int{start, end})
	}

	mCells := len(baseCells)
	var intervalConds []Node
	yVar := &VarNode{Name: varName}

	for _, interval := range intervals {
		start := interval[0]
		end := interval[1]

		var leftCond Node
		if start > 0 {
			if baseCells[start].IsSection {
				leftBound := baseCells[start].SamplePoint[0]
				leftCond = NewRelOp(yVar, ">=", leftBound)
			} else {
				leftBound := baseCells[start-1].SamplePoint[0]
				leftCond = NewRelOp(yVar, ">", leftBound)
			}
		}

		var rightCond Node
		if end < mCells-1 {
			if baseCells[end].IsSection {
				rightBound := baseCells[end].SamplePoint[0]
				rightCond = NewRelOp(yVar, "<=", rightBound)
			} else {
				rightBound := baseCells[end+1].SamplePoint[0]
				rightCond = NewRelOp(yVar, "<", rightBound)
			}
		}

		var cond Node
		if leftCond == nil && rightCond == nil {
			cond = &VarNode{Name: "true"}
		} else if leftCond == nil {
			cond = rightCond
		} else if rightCond == nil {
			cond = leftCond
		} else {
			if start == end && baseCells[start].IsSection {
				cond = NewRelOp(yVar, "==", baseCells[start].SamplePoint[0])
			} else {
				cond = mustFunc("and", leftCond, rightCond)
			}
		}
		intervalConds = append(intervalConds, cond)
	}

	if len(intervalConds) == 0 {
		return &VarNode{Name: "false"}, nil
	}
	if len(intervalConds) == 1 {
		return intervalConds[0], nil
	}
	return mustFunc("or", intervalConds...), nil
}

func isPositiveConst(n Node) bool {
	if r, ok := n.(*RationalNode); ok {
		return r.Val.Sign() > 0
	}
	return false
}

func isNegativeConst(n Node) bool {
	if r, ok := n.(*RationalNode); ok {
		return r.Val.Sign() < 0
	}
	return false
}

func isNonZeroConst(n Node) bool {
	if r, ok := n.(*RationalNode); ok {
		return r.Val.Sign() != 0
	}
	return false
}

func mustFunc(name string, args ...Node) *FuncNode {
	return &FuncNode{Name: name, Args: args}
}

