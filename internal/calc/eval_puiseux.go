package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// NewtonPoint represents an exponent pair (i, j) in F(x, y) = sum a_{i,j} x^i y^j.
type NewtonPoint struct {
	X     int  // Exponent of x (i)
	Y     int  // Exponent of y (j)
	Coeff Node // Non-zero coefficient node
}

// NewtonEdge represents an edge on the lower convex hull of the Newton polygon.
type NewtonEdge struct {
	P1               NewtonPoint // Start point (higher y)
	P2               NewtonPoint // End point (lower y)
	SlopeNumerator   int         // p in lambda = p / q
	SlopeDenominator int         // q in lambda = p / q (q >= 1)
	MinDegree        *big.Rat    // gamma = i + lambda * j
	Points           []NewtonPoint
}

// PuiseuxTerm represents a single fractional power term c * x^(p/q).
type PuiseuxTerm struct {
	Coeff  Node
	ExpNum int
	ExpDen int
}

// PuiseuxBranch represents one local solution branch y(x) = sum c_k x^(p_k/q_k).
type PuiseuxBranch struct {
	Terms             []PuiseuxTerm
	RamificationIndex int
	TruncationOrder   int
}

func init() {
	RegisterHandler("puiseux", func(args []Node, env *Env) (Node, error) {
		return EvalPuiseux(args)
	})
}

// EvalPuiseux computes the exact Puiseux series solutions y(x) for F(x, y) = 0 around x -> 0.
// Arguments: puiseux(F, y, x [, order])
func EvalPuiseux(args []Node) (Node, error) {
	if len(args) < 3 || len(args) > 4 {
		return nil, fmt.Errorf("%s", i18n.T("puiseux.err_puiseux_args"))
	}

	fsm := NewPuiseuxLifecycleFSM()

	expr := args[0]
	// Handle equation L == R -> L - R
	if rel, ok := expr.(*RelOpNode); ok && (rel.Op == "==" || rel.Op == "=") {
		negRHS, _ := simplifyUnaryOp("-", rel.RHS)
		diff, err := simplifyAdd([]Node{rel.LHS, negRHS})
		if err != nil {
			_ = fsm.TransitionTo(PuiseuxStateFailed)
			return nil, err
		}
		expr = diff
	}

	yVarNode, okY := args[1].(*VarNode)
	if !okY {
		_ = fsm.TransitionTo(PuiseuxStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("puiseux.err_arg_must_be_variable", args[1].String()))
	}
	yVar := yVarNode.Name

	xVarNode, okX := args[2].(*VarNode)
	if !okX {
		_ = fsm.TransitionTo(PuiseuxStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("puiseux.err_arg_must_be_variable", args[2].String()))
	}
	xVar := xVarNode.Name

	order := 3 // default truncation order
	if len(args) == 4 {
		orderRat, okOrd := args[3].(*RationalNode)
		if !okOrd || !orderRat.Val.IsInt() || orderRat.Val.Sign() < 0 {
			_ = fsm.TransitionTo(PuiseuxStateFailed)
			return nil, fmt.Errorf("%s", i18n.T("puiseux.err_order_must_be_positive_int", args[3].String()))
		}
		order = int(orderRat.Val.Num().Int64())
	}

	// Simplify expr before extracting support
	simplifiedExpr, err := Eval(expr)
	if err != nil {
		simplifiedExpr = expr
	}
	if isZero(simplifiedExpr) {
		_ = fsm.TransitionTo(PuiseuxStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("puiseux.err_identity_equation"))
	}

	// Verify expr contains yVar
	if !containsVar(simplifiedExpr, yVar) {
		_ = fsm.TransitionTo(PuiseuxStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("puiseux.err_equation_contains_no_var", yVar))
	}

	// 1. Extract bivariate support points (i, j)
	points, err := extractBivariatePoints(simplifiedExpr, yVar, xVar)
	if err != nil {
		_ = fsm.TransitionTo(PuiseuxStateFailed)
		return nil, err
	}
	if len(points) == 0 {
		_ = fsm.TransitionTo(PuiseuxStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("puiseux.err_equation_contains_no_var", yVar))
	}

	if err := fsm.TransitionTo(PuiseuxStateSupportExtracted); err != nil {
		return nil, err
	}

	// 2. Build Newton Polygon (Lower Convex Hull)
	edges, err := BuildNewtonPolygon(points)
	if err != nil {
		_ = fsm.TransitionTo(PuiseuxStateFailed)
		return nil, err
	}

	if len(edges) == 0 {
		_ = fsm.TransitionTo(PuiseuxStateBranchesConstructed)
		return NewList([]Node{}), nil
	}

	if err := fsm.TransitionTo(PuiseuxStateHullConstructed); err != nil {
		return nil, err
	}

	// 3. For each edge, compute characteristic roots and expand branches
	var allBranchNodes []Node
	seenBranches := make(map[string]bool)

	for _, edge := range edges {
		_ = fsm.TransitionTo(PuiseuxStateEdgeSelected)

		// Characteristic roots
		cRoots, err := SolveCharacteristicPolynomial(edge)
		if err != nil {
			_ = fsm.TransitionTo(PuiseuxStateFailed)
			return nil, err
		}
		if len(cRoots) == 0 {
			continue
		}

		_ = fsm.TransitionTo(PuiseuxStateCharPolySolved)

		p := edge.SlopeNumerator
		q := edge.SlopeDenominator

		for _, c0 := range cRoots {
			branchTerms := []PuiseuxTerm{
				{Coeff: c0, ExpNum: p, ExpDen: q},
			}

			// If higher order terms are needed, perform shift and recursion
			_ = fsm.TransitionTo(PuiseuxStateRamifiedAndShifted)
			expandedTerms := expandHigherTerms(simplifiedExpr, yVar, xVar, branchTerms, order, 1)
			_ = fsm.TransitionTo(PuiseuxStateRecursiveExpanded)

			branchNode := constructSeriesNode(expandedTerms, xVar)
			str := Format(branchNode)
			if !seenBranches[str] {
				seenBranches[str] = true
				allBranchNodes = append(allBranchNodes, branchNode)
			}
		}
	}

	_ = fsm.TransitionTo(PuiseuxStateBranchesConstructed)
	return NewList(allBranchNodes), nil
}

// extractBivariatePoints extracts non-zero points (i, j) where F = sum a_{i,j} x^i y^j.
func extractBivariatePoints(expr Node, yVar, xVar string) ([]NewtonPoint, error) {
	yCoeffs, err := extractPolyCoeffs(expr, yVar)
	if err != nil {
		return nil, err
	}

	var points []NewtonPoint
	for j, aJ := range yCoeffs {
		if isZero(aJ) {
			continue
		}
		// If aJ does not contain xVar, it's x^0
		if !containsVar(aJ, xVar) {
			evalCoeff, _ := Eval(aJ)
			if !isZero(evalCoeff) {
				points = append(points, NewtonPoint{X: 0, Y: j, Coeff: evalCoeff})
			}
			continue
		}

		xCoeffs, errX := extractPolyCoeffs(aJ, xVar)
		if errX != nil {
			// fallback: treat as x^0 if error
			evalCoeff, _ := Eval(aJ)
			if !isZero(evalCoeff) {
				points = append(points, NewtonPoint{X: 0, Y: j, Coeff: evalCoeff})
			}
			continue
		}

		for i, aIJ := range xCoeffs {
			evalCoeff, _ := Eval(aIJ)
			if !isZero(evalCoeff) {
				points = append(points, NewtonPoint{X: i, Y: j, Coeff: evalCoeff})
			}
		}
	}

	return points, nil
}

// BuildNewtonPolygon computes the lower convex hull of the points,
// returning segments from highest y to lowest y.
func BuildNewtonPolygon(points []NewtonPoint) ([]NewtonEdge, error) {
	if len(points) == 0 {
		return nil, nil
	}

	// Filter distinct (X, Y)
	pointMap := make(map[string]NewtonPoint)
	for _, pt := range points {
		key := fmt.Sprintf("%d,%d", pt.X, pt.Y)
		pointMap[key] = pt
	}
	var pts []NewtonPoint
	for _, pt := range pointMap {
		pts = append(pts, pt)
	}

	// Find maxY and minY
	maxY := pts[0].Y
	minY := pts[0].Y
	for _, pt := range pts {
		if pt.Y > maxY {
			maxY = pt.Y
		}
		if pt.Y < minY {
			minY = pt.Y
		}
	}

	if maxY == minY {
		return nil, nil
	}

	// Start from point with y = maxY (if multiple, minimum x)
	var current NewtonPoint
	first := true
	for _, pt := range pts {
		if pt.Y == maxY {
			if first || pt.X < current.X {
				current = pt
				first = false
			}
		}
	}

	var edges []NewtonEdge

	// Trace lower convex hull from current.Y down to minY
	for current.Y > minY {
		// Find next vertex P such that P.Y < current.Y and
		// the slope lambda = -(P.X - current.X)/(P.Y - current.Y) is minimal,
		// and all points satisfy Q.X + lambda * Q.Y >= current.X + lambda * current.Y.
		var bestTarget NewtonPoint
		var bestLambda *big.Rat
		foundTarget := false

		for _, cand := range pts {
			if cand.Y >= current.Y {
				continue
			}
			dx := int64(cand.X - current.X)
			dy := int64(cand.Y - current.Y) // negative
			// lambda = - dx / dy = dx / (-dy)
			candLambda := new(big.Rat).SetFrac(big.NewInt(dx), big.NewInt(-dy))

			// Check if all points lie above or on the supporting line
			// Q.X + lambda * Q.Y >= current.X + lambda * current.Y
			// <=> Q.X - current.X + lambda * (Q.Y - current.Y) >= 0
			valid := true
			for _, q := range pts {
				qDx := new(big.Rat).SetInt64(int64(q.X - current.X))
				qDy := new(big.Rat).SetInt64(int64(q.Y - current.Y))
				term := new(big.Rat).Mul(candLambda, qDy)
				val := new(big.Rat).Add(qDx, term)
				if val.Sign() < 0 {
					valid = false
					break
				}
			}

			if valid {
				if !foundTarget || candLambda.Cmp(bestLambda) < 0 || (candLambda.Cmp(bestLambda) == 0 && cand.Y < bestTarget.Y) {
					bestTarget = cand
					bestLambda = candLambda
					foundTarget = true
				}
			}
		}

		if !foundTarget {
			break
		}

		// Find all points lying on this segment
		var onEdge []NewtonPoint
		minGamma := new(big.Rat).Add(new(big.Rat).SetInt64(int64(current.X)), new(big.Rat).Mul(bestLambda, new(big.Rat).SetInt64(int64(current.Y))))
		for _, q := range pts {
			gammaQ := new(big.Rat).Add(new(big.Rat).SetInt64(int64(q.X)), new(big.Rat).Mul(bestLambda, new(big.Rat).SetInt64(int64(q.Y))))
			if gammaQ.Cmp(minGamma) == 0 {
				onEdge = append(onEdge, q)
			}
		}

		pInt := int(bestLambda.Num().Int64())
		qInt := int(bestLambda.Denom().Int64())

		edge := NewtonEdge{
			P1:               current,
			P2:               bestTarget,
			SlopeNumerator:   pInt,
			SlopeDenominator: qInt,
			MinDegree:        minGamma,
			Points:           onEdge,
		}
		edges = append(edges, edge)
		current = bestTarget
	}

	return edges, nil
}

// SolveCharacteristicPolynomial solves P_E(c) = sum_{(i,j) in E} a_{i,j} c^j = 0 for non-zero c.
func SolveCharacteristicPolynomial(edge NewtonEdge) ([]Node, error) {
	if len(edge.Points) == 0 {
		return nil, nil
	}

	minY := edge.Points[0].Y
	for _, pt := range edge.Points {
		if pt.Y < minY {
			minY = pt.Y
		}
	}

	// Construct polynomial in variable "c": sum a_{i, j} * c^(j - minY)
	var terms []Node
	for _, pt := range edge.Points {
		deg := pt.Y - minY
		var term Node
		switch deg {
		case 0:
			term = pt.Coeff
		case 1:
			term, _ = simplifyMul([]Node{pt.Coeff, NewVar("c")})
		default:
			pow := &PowNode{Base: NewVar("c"), Exp: mustRational(int64(deg), 1)}
			term, _ = simplifyMul([]Node{pt.Coeff, pow})
		}
		terms = append(terms, term)
	}

	charPoly, err := simplifyAdd(terms)
	if err != nil {
		return nil, err
	}
	charPoly, _ = Eval(charPoly)

	// If characteristic polynomial is linear or pure power, solve directly
	// e.g. a*c^2 + b = 0 or a*c - b = 0
	roots, err := solveEquation(charPoly, "c")
	if err != nil {
		// Try factoring or fallback
		return nil, fmt.Errorf("%s: %w", i18n.T("puiseux.err_cannot_expand", charPoly.String()), err)
	}

	listNode, ok := roots.(*ListNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("puiseux.err_no_characteristic_roots"))
	}

	var nonZeroRoots []Node
	seen := make(map[string]bool)
	for _, r := range listNode.Elements {
		evalR, _ := Eval(r)
		if isZero(evalR) {
			continue
		}
		s := Format(evalR)
		if !seen[s] {
			seen[s] = true
			nonZeroRoots = append(nonZeroRoots, evalR)
		}
	}

	return nonZeroRoots, nil
}

// expandHigherTerms performs Tschirnhausen shift y = c0 * t^p + y1 and recurses up to maxOrder.
func expandHigherTerms(expr Node, yVar, xVar string, currentTerms []PuiseuxTerm, maxOrder int, depth int) []PuiseuxTerm {
	if depth > maxOrder || len(currentTerms) == 0 {
		return currentTerms
	}

	lastTerm := currentTerms[len(currentTerms)-1]
	// Check if current order already satisfies maxOrder
	currDeg := float64(lastTerm.ExpNum) / float64(lastTerm.ExpDen)
	if currDeg >= float64(maxOrder) {
		return currentTerms
	}

	// Substitute x = t^q into expr, where q = lastTerm.ExpDen
	q := lastTerm.ExpDen
	p := lastTerm.ExpNum
	c0 := lastTerm.Coeff

	tVar := "__t"
	y1Var := "__y1"

	// Construct xSub = t^q
	var xSub Node = NewVar(tVar)
	if q != 1 {
		xSub = &PowNode{Base: NewVar(tVar), Exp: mustRational(int64(q), 1)}
	}

	// Construct ySub = c0 * t^p + y1
	var leadingTerm Node
	switch p {
	case 0:
		leadingTerm = c0
	case 1:
		leadingTerm, _ = simplifyMul([]Node{c0, NewVar(tVar)})
	default:
		pow := &PowNode{Base: NewVar(tVar), Exp: mustRational(int64(p), 1)}
		leadingTerm, _ = simplifyMul([]Node{c0, pow})
	}
	ySub, _ := simplifyAdd([]Node{leadingTerm, NewVar(y1Var)})

	// Substitute into expr
	sub1 := Substitute(expr, xVar, xSub)
	sub2 := Substitute(sub1, yVar, ySub)
	evalF1, err := Eval(sub2)
	if err != nil {
		return currentTerms
	}

	// Check if y1 = 0 is an exact root (i.e. evalF1 at y1 = 0 is identically zero)
	zeroTest := Substitute(evalF1, y1Var, mustRational(0, 1))
	evalZero, _ := Eval(zeroTest)
	if isZero(evalZero) {
		return currentTerms // exact closed form
	}

	// Extract points of F1(t, y1)
	pts1, err := extractBivariatePoints(evalF1, y1Var, tVar)
	if err != nil || len(pts1) == 0 {
		return currentTerms
	}

	// Find edges of F1
	edges1, err := BuildNewtonPolygon(pts1)
	if err != nil || len(edges1) == 0 {
		return currentTerms
	}

	// Select the edge with positive slope (higher order correction)
	for _, edge := range edges1 {
		p1 := edge.SlopeNumerator
		q1 := edge.SlopeDenominator
		if p1 <= 0 {
			continue
		}

		cRoots, err := SolveCharacteristicPolynomial(edge)
		if err != nil || len(cRoots) == 0 {
			continue
		}

		// First valid root
		c1 := cRoots[0]

		// Effective total exponent: original exponent (p/q) + correction (p1 / (q * q1))
		// Total exp in original variable x = (p * q1 + p1) / (q * q1)
		newNum := int64(p*q1 + p1)
		newDen := int64(q * q1)
		gcdVal := new(big.Int).GCD(nil, nil, big.NewInt(newNum), big.NewInt(newDen)).Int64()
		newNum /= gcdVal
		newDen /= gcdVal

		newTerm := PuiseuxTerm{
			Coeff:  c1,
			ExpNum: int(newNum),
			ExpDen: int(newDen),
		}

		nextTerms := append(currentTerms, newTerm)
		return expandHigherTerms(expr, yVar, xVar, nextTerms, maxOrder, depth+1)
	}

	return currentTerms
}

// constructSeriesNode turns a slice of PuiseuxTerms into a readable AddNode or single Node.
func constructSeriesNode(terms []PuiseuxTerm, xVar string) Node {
	if len(terms) == 0 {
		return mustRational(0, 1)
	}

	var nodeTerms []Node
	for _, term := range terms {
		c := term.Coeff
		p := term.ExpNum
		q := term.ExpDen

		if isZero(c) {
			continue
		}

		if p == 0 {
			nodeTerms = append(nodeTerms, c)
			continue
		}

		var xPow Node
		if q == 1 {
			if p == 1 {
				xPow = NewVar(xVar)
			} else {
				xPow = &PowNode{Base: NewVar(xVar), Exp: mustRational(int64(p), 1)}
			}
		} else {
			xPow = &PowNode{Base: NewVar(xVar), Exp: mustRational(int64(p), int64(q))}
		}

		if isOne(c) {
			nodeTerms = append(nodeTerms, xPow)
		} else if isMinusOne(c) {
			negPow, _ := simplifyUnaryOp("-", xPow)
			nodeTerms = append(nodeTerms, negPow)
		} else {
			mul, err := simplifyMul([]Node{c, xPow})
			if err != nil {
				mul = &MulNode{Factors: []Node{c, xPow}}
			}
			nodeTerms = append(nodeTerms, mul)
		}
	}

	if len(nodeTerms) == 0 {
		return mustRational(0, 1)
	}
	if len(nodeTerms) == 1 {
		return nodeTerms[0]
	}

	res, err := simplifyAdd(nodeTerms)
	if err != nil {
		return &AddNode{Terms: nodeTerms}
	}
	return res
}

func isMinusOne(n Node) bool {
	if r, ok := n.(*RationalNode); ok {
		return r.Val.Cmp(big.NewRat(-1, 1)) == 0
	}
	return false
}
