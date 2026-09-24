package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// ReductionWitness records an exact algebraic identity for a single pseudo-division step:
// init^exp * F = Q * G + R
type ReductionWitness struct {
	MainVar  string
	Initial  Node
	Exponent int
	Dividend Node
	Divisor  Node
	Quotient Node
	Residual Node
}

// IdentifyMainVarAndRank finds the leading variable (highest in order) actually present in p,
// along with its degree and initial (leading coefficient polynomial in lower variables).
func IdentifyMainVarAndRank(p Node, order []string) (string, int, Node) {
	evaled, err := Eval(expandNode(p))
	if err != nil {
		evaled = p
	}
	if isZero(evaled) {
		return "", -1, mustRational(0, 1)
	}

	vars := collectVariables(evaled)
	if len(vars) == 0 {
		return "", 0, evaled
	}

	varRank := make(map[string]int)
	for i, v := range order {
		varRank[v] = i
	}

	bestVar := ""
	bestRank := -1
	for _, v := range vars {
		r, ok := varRank[v]
		if !ok {
			r = len(order) // fallback if unlisted
		}
		if r > bestRank {
			bestRank = r
			bestVar = v
		}
	}

	if bestVar == "" {
		return "", 0, evaled
	}

	poly, ok := extractPoly(evaled, bestVar)
	if !ok || isPolyZero(poly) {
		return bestVar, 0, evaled
	}

	deg := poly.degree()
	lead := poly.leadCoeff()
	return bestVar, deg, lead
}

// ContentPrimitive separates polynomial p into content (GCD of coefficients in lower variables)
// and primitive part pp = p / content.
func ContentPrimitive(p Node, mainVar string, order []string) (Node, Node, error) {
	evaled, err := Eval(expandNode(p))
	if err != nil {
		evaled = p
	}
	if isZero(evaled) {
		return mustRational(0, 1), mustRational(0, 1), nil
	}

	if mainVar == "" {
		mvar, _, _ := IdentifyMainVarAndRank(evaled, order)
		mainVar = mvar
	}
	if mainVar == "" {
		// Scalar constant
		return evaled, mustRational(1, 1), nil
	}

	poly, ok := extractPoly(evaled, mainVar)
	if !ok || isPolyZero(poly) || poly.degree() == 0 {
		return mustRational(1, 1), evaled, nil
	}

	cont, err := polyContent(poly)
	if err != nil || isZero(cont) || isOne(cont) {
		return mustRational(1, 1), evaled, nil
	}

	ppPoly, err := polyExactDivideScalar(poly, cont)
	if err != nil {
		return mustRational(1, 1), evaled, nil
	}

	primNode := ppPoly.toNode()
	if ratLead, ok := ppPoly.leadCoeff().(*RationalNode); ok && ratLead.Val.Sign() < 0 {
		neg, err := Eval(&UnaryOpNode{Op: "-", Expr: primNode})
		if err == nil {
			primNode = neg
		}
	}
	return cont, primNode, nil
}

// PseudoDividePrimitive performs Ritt-Wu pseudo-division of F by G with respect to mainVar:
// I^(max(deg(F)-deg(G)+1, 0)) * F = Q * G + R
// and immediately normalizes R to its primitive part to prevent coefficient explosion.
func PseudoDividePrimitive(F, G Node, mainVar string, order []string) (Node, Node, int, Node, error) {
	evalF, err := Eval(expandNode(F))
	if err != nil {
		evalF = F
	}
	evalG, err := Eval(expandNode(G))
	if err != nil {
		evalG = G
	}

	if isZero(evalG) {
		return nil, nil, 0, nil, fmt.Errorf("%s", i18n.T("poly.err_division_by_zero"))
	}
	if isZero(evalF) {
		return mustRational(0, 1), mustRational(0, 1), 0, mustRational(1, 1), nil
	}

	if mainVar == "" {
		mvarG, _, _ := IdentifyMainVarAndRank(evalG, order)
		mainVar = mvarG
	}
	if mainVar == "" {
		return nil, nil, 0, nil, fmt.Errorf("main variable cannot be empty for pseudo-division")
	}

	polyF, okF := extractPoly(evalF, mainVar)
	polyG, okG := extractPoly(evalG, mainVar)
	if !okF || !okG {
		return nil, nil, 0, nil, fmt.Errorf("failed to extract polynomials in variable %s", mainVar)
	}

	degF := polyF.degree()
	degG := polyG.degree()
	initial := polyG.leadCoeff()

	if degF < degG {
		_, primF, _ := ContentPrimitive(evalF, mainVar, order)
		return mustRational(0, 1), primF, 0, initial, nil
	}

	exp := degF - degG + 1
	remPoly, ok := pseudoRemainder(polyF, polyG)
	if !ok {
		return nil, nil, 0, nil, fmt.Errorf("pseudo-remainder calculation failed")
	}

	remNode := remPoly.toNode()
	evalRem, err := Eval(expandNode(remNode))
	if err != nil {
		evalRem = remNode
	}

	// Content removal to enforce primitiveness
	if !isZero(evalRem) {
		_, primRem, err := ContentPrimitive(evalRem, mainVar, order)
		if err == nil && primRem != nil {
			evalRem = primRem
		}
	}

	return mustRational(0, 1), evalRem, exp, initial, nil
}

// SubresultantPrem computes the pseudo-remainder using Collins Subresultant algorithm
// when F and G share the same main variable and deg(F) >= deg(G).
func SubresultantPrem(F, G Node, mainVar string, order []string) (Node, error) {
	evalF, _ := Eval(expandNode(F))
	evalG, _ := Eval(expandNode(G))
	if isZero(evalG) {
		return nil, fmt.Errorf("%s", i18n.T("poly.err_division_by_zero"))
	}
	if isZero(evalF) {
		return mustRational(0, 1), nil
	}

	polyF, okF := extractPoly(evalF, mainVar)
	polyG, okG := extractPoly(evalG, mainVar)
	if !okF || !okG {
		return nil, fmt.Errorf("failed to extract polynomials in %s", mainVar)
	}

	if polyF.degree() < polyG.degree() {
		return evalF, nil
	}

	r, ok := pseudoRemainder(polyF, polyG)
	if !ok {
		return nil, fmt.Errorf("pseudoRemainder failed in subresultant step")
	}

	remNode := r.toNode()
	_, primRem, err := ContentPrimitive(remNode, mainVar, order)
	if err == nil && primRem != nil {
		return primRem, nil
	}
	return remNode, nil
}

// SuccessiveReduce successively reduces conclusion G by an ascending chain in descending rank order.
func SuccessiveReduce(conclusion Node, chain []Node, order []string) (Node, []Node, []ReductionWitness, error) {
	cur := conclusion
	var initials []Node
	var witnesses []ReductionWitness

	// Reduce in descending order (highest main variable first)
	for i := len(chain) - 1; i >= 0; i-- {
		A := chain[i]
		if isZero(cur) {
			break
		}

		mvarA, degA, _ := IdentifyMainVarAndRank(A, order)
		if mvarA == "" || degA <= 0 {
			continue
		}

		// Check if cur contains mvarA with degree >= degA
		mvarCur, _, _ := IdentifyMainVarAndRank(cur, order)
		if mvarCur != mvarA && !containsVar(cur, mvarA) {
			continue
		}

		polyCur, okCur := extractPoly(cur, mvarA)
		if !okCur || polyCur.degree() < degA {
			continue
		}

		_, rem, exp, initial, err := PseudoDividePrimitive(cur, A, mvarA, order)
		if err != nil {
			return nil, nil, nil, err
		}

		if initial != nil && !isZero(initial) && !isOne(initial) {
			initials = append(initials, initial)
		}

		witnesses = append(witnesses, ReductionWitness{
			MainVar:  mvarA,
			Initial:  initial,
			Exponent: exp,
			Dividend: cur,
			Divisor:  A,
			Residual: rem,
		})

		cur = rem
	}

	evalFinal, err := Eval(expandNode(cur))
	if err != nil {
		evalFinal = cur
	}
	if !isZero(evalFinal) {
		vars := collectVariables(evalFinal)
		if len(vars) > 0 {
			polyNode, err := NodeToPoly(evalFinal, vars, ast.OrderGrevLex)
			if err == nil {
				evalFinal = PolyToNode(polyNode)
			}
		}
	}

	// De-duplicate initials
	var uniqueInitials []Node
	seen := make(map[string]bool)
	for _, initNode := range initials {
		str := initNode.String()
		if !seen[str] && !isOne(initNode) && !isZero(initNode) {
			seen[str] = true
			uniqueInitials = append(uniqueInitials, initNode)
		}
	}

	return evalFinal, uniqueInitials, witnesses, nil
}
