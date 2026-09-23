package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Symbolic Limit Computation (Direct Substitution, Cancellation, L'Hopital)
// -------------------------------------------------------------------------

// EvalLimit computes the exact symbolic limit of expr as varName approaches target.
// It supports direct substitution, rational factor cancellation via the Factor Theorem,
// L'Hopital's rule, and infinite limits with degree comparison.
func EvalLimit(expr Node, varNode Node, target Node, dirNode Node) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.err_limit_expression_cannot_be_nil"))
	}
	vNode, ok := varNode.(*VarNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.err_limit_second_argument_must_be", varNode.String()))
	}
	varName := vNode.Name

	// Determine direction: 0 = two-sided, 1 = right limit (+), -1 = left limit (-)
	dir := 0
	if dirNode != nil {
		if r, ok := dirNode.(*RationalNode); ok {
			sign := r.Val.Sign()
			if sign > 0 {
				dir = 1
			} else if sign < 0 {
				dir = -1
			}
		} else if v, ok := dirNode.(*VarNode); ok {
			switch v.Name {
			case "+", "right":
				dir = 1
			case "-", "left":
				dir = -1
			}
		}
	}

	// Check if target is infinity
	isInf, infSign := isInfiniteTarget(target)
	if isInf {
		res, err := evalInfiniteLimit(expr, varName, infSign, 0)
		if err != nil {
			return nil, err
		}
		RecordTraceRewrite(RuleLimit, expr, res, i18n.T("explain.limit_inf", varName, target.String(), expr.String(), res.String()))
		return res, nil
	}

	// Finite limit
	res, err := evalFiniteLimit(expr, varName, target, dir, 0)
	if err != nil {
		return nil, err
	}
	RecordTraceRewrite(RuleLimit, expr, res, i18n.T("explain.limit_eval", varName, target.String(), expr.String(), res.String()))
	return res, nil
}

// isInfiniteTarget checks if a node represents +inf, -inf, or infinity.
func isInfiniteTarget(target Node) (bool, int) {
	if target == nil {
		return false, 0
	}
	switch v := target.(type) {
	case *VarNode:
		if v.Name == "inf" || v.Name == "infinity" {
			return true, 1
		}
	case *UnaryOpNode:
		if v.Op == "-" {
			if inner, ok := v.Expr.(*VarNode); ok && (inner.Name == "inf" || inner.Name == "infinity") {
				return true, -1
			}
		}
	}
	return false, 0
}

// evalFiniteLimit evaluates the limit as varName approaches finite target a.
func evalFiniteLimit(expr Node, varName string, a Node, dir int, depth int) (Node, error) {
	if depth > 5 {
		return nil, fmt.Errorf("%s", i18n.T("errors.err_limit_recursion_depth_exceeded_possible"))
	}

	// If expr does not contain varName, limit is expr itself
	if !containsVar(expr, varName) {
		return Eval(expr)
	}

	// Step 1: Separate into rational fraction P(x) / Q(x)
	num, den := toRationalFraction(expr)

	// Evaluate numerator and denominator at x = a
	numSub := Substitute(num, varName, a)
	numVal, errNum := Eval(numSub)

	denSub := Substitute(den, varName, a)
	denVal, errDen := Eval(denSub)

	numIsZero := (errNum == nil && isZero(numVal))
	denIsZero := (errDen == nil && isZero(denVal))

	// Step 2: Non-zero denominator (direct evaluation succeeds)
	if errDen == nil && !denIsZero {
		if errNum == nil {
			invDen, err := simplifyPow(denVal, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			return Eval(NewMul([]Node{numVal, invDen}))
		}
	}

	// Step 3: Indeterminate form 0/0
	if numIsZero && denIsZero {
		// 3.1 Try polynomial division / factor cancellation by (x - a)
		cancelledExpr, cancelled := tryCancelLinearFactor(num, den, varName, a)
		if cancelled {
			RecordTraceRewrite(RuleLimit, expr, cancelledExpr, i18n.T("explain.limit_cancel_factor", varName, a.String()))
			return evalFiniteLimit(cancelledExpr, varName, a, dir, depth+1)
		}

		// 3.2 Try L'Hopital's rule
		dNum, errD1 := differentiate(num, varName)
		dDen, errD2 := differentiate(den, varName)
		if errD1 == nil && errD2 == nil && !isZero(dDen) {
			invDDen, _ := simplifyPow(dDen, mustRational(-1, 1))
			lhopitalExpr, _ := simplifyMul([]Node{dNum, invDDen})
			RecordTraceRewrite(RuleLimit, expr, lhopitalExpr, i18n.T("explain.limit_lhopital", varName, num.String(), varName, den.String(), dNum.String(), dDen.String()))
			return evalFiniteLimit(lhopitalExpr, varName, a, dir, depth+1)
		}
	}

	// Step 4: Nonzero / 0 form (infinite limit / divergence)
	if !numIsZero && denIsZero && errNum == nil {
		return evaluateSingularityLimit(numVal, den, varName, a, dir)
	}

	// Fallback to direct substitution if not already covered
	sub := Substitute(expr, varName, a)
	val, err := Eval(sub)
	if err == nil && isValidLimitValue(val) {
		return val, nil
	}

	return nil, fmt.Errorf("%s", i18n.T("errors.err_limit_unable_to_resolve_limit", a.String(), err))
}

// evalInfiniteLimit evaluates the limit as varName approaches +inf (sign = 1) or -inf (sign = -1).
func evalInfiniteLimit(expr Node, varName string, sign int, depth int) (Node, error) {
	if depth > 5 {
		return nil, fmt.Errorf("%s", i18n.T("errors.err_limit_recursion_depth_exceeded_in"))
	}

	if !containsVar(expr, varName) {
		return Eval(expr)
	}

	num, den := toRationalFraction(expr)

	// Check if both are polynomials in varName
	polyNum, okNum := extractPoly(num, varName)
	polyDen, okDen := extractPoly(den, varName)

	if okNum && okDen {
		degNum := polyNum.degree()
		degDen := polyDen.degree()

		if degNum < degDen {
			return mustRational(0, 1), nil
		}
		if degNum == degDen {
			leadNum := polyNum.leadCoeff()
			leadDen := polyDen.leadCoeff()
			invDen, err := simplifyPow(leadDen, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			return Eval(NewMul([]Node{leadNum, invDen}))
		}
		// degNum > degDen: diverges to +inf or -inf
		leadRatio, err := simplifyMul([]Node{polyNum.leadCoeff(), &PowNode{Base: polyDen.leadCoeff(), Exp: mustRational(-1, 1)}})
		if err != nil {
			return nil, err
		}
		evaluatedRatio, _ := Eval(leadRatio)
		ratioSign := getNodeSign(evaluatedRatio)
		degDiff := degNum - degDen

		var totalSign int
		if sign > 0 {
			totalSign = ratioSign
		} else {
			if degDiff%2 == 0 {
				totalSign = ratioSign
			} else {
				totalSign = -ratioSign
			}
		}

		if totalSign >= 0 {
			return &VarNode{Name: "inf"}, nil
		}
		return simplifyUnaryOp("-", &VarNode{Name: "inf"})
	}

	// If not both polynomials, try L'Hopital if both grow (infinity / infinity)
	dNum, errD1 := differentiate(num, varName)
	dDen, errD2 := differentiate(den, varName)
	if errD1 == nil && errD2 == nil && !isZero(dDen) {
		invDDen, _ := simplifyPow(dDen, mustRational(-1, 1))
		lhopitalExpr, _ := simplifyMul([]Node{dNum, invDDen})
		RecordTraceRewrite(RuleLimit, expr, lhopitalExpr, i18n.T("explain.limit_lhopital_inf", dNum.String(), dDen.String()))
		return evalInfiniteLimit(lhopitalExpr, varName, sign, depth+1)
	}

	return nil, fmt.Errorf("%s", i18n.T("errors.err_limit_unable_to_resolve_limit_1", expr.String()))
}

// toRationalFraction converts any expression expr into a single fraction num / den.
func toRationalFraction(expr Node) (Node, Node) {
	if expr == nil {
		return mustRational(0, 1), mustRational(1, 1)
	}

	switch v := expr.(type) {
	case *AddNode:
		// Extract fraction for each term: T_i = N_i / D_i
		termsNum := make([]Node, len(v.Terms))
		termsDen := make([]Node, len(v.Terms))
		allOne := true
		for i, t := range v.Terms {
			termsNum[i], termsDen[i] = extractFractionFromTerm(t)
			if !isOne(termsDen[i]) {
				allOne = false
			}
		}
		if allOne {
			return expr, mustRational(1, 1)
		}

		// Check if all non-one denominators are identical
		var commonD Node = nil
		sameD := true
		for _, d := range termsDen {
			if !isOne(d) {
				if commonD == nil {
					commonD = d
				} else if !commonD.Equal(d) {
					sameD = false
					break
				}
			}
		}

		if sameD && commonD != nil {
			var newNumerTerms []Node
			for i, d := range termsDen {
				if isOne(d) {
					p, _ := simplifyMul([]Node{termsNum[i], commonD})
					newNumerTerms = append(newNumerTerms, p)
				} else {
					newNumerTerms = append(newNumerTerms, termsNum[i])
				}
			}
			num, _ := simplifyAdd(newNumerTerms)
			return num, commonD
		}

		// General case: D = D_1 * ... * D_k
		D, _ := simplifyMul(termsDen)
		var newNumerTerms []Node
		for i, ni := range termsNum {
			var coFactors []Node
			coFactors = append(coFactors, ni)
			for j, dj := range termsDen {
				if j != i {
					coFactors = append(coFactors, dj)
				}
			}
			termProd, _ := simplifyMul(coFactors)
			newNumerTerms = append(newNumerTerms, termProd)
		}
		num, _ := simplifyAdd(newNumerTerms)
		return num, D

	case *MulNode:
		return extractFractionFromTerm(expr)

	default:
		return extractFractionFromTerm(expr)
	}
}

// extractFractionFromTerm extracts numerator and denominator from a single term.
func extractFractionFromTerm(expr Node) (Node, Node) {
	if expr == nil {
		return mustRational(0, 1), mustRational(1, 1)
	}
	if p, ok := expr.(*PowNode); ok {
		if r, ok := p.Exp.(*RationalNode); ok && r.Val.Sign() < 0 {
			posExp := mustRational(-r.Val.Num().Int64(), r.Val.Denom().Int64())
			if posExp.Val.Cmp(big.NewRat(1, 1)) == 0 {
				return mustRational(1, 1), p.Base
			}
			return mustRational(1, 1), &PowNode{Base: p.Base, Exp: posExp}
		}
	}
	if m, ok := expr.(*MulNode); ok {
		var numFactors []Node
		var denFactors []Node
		for _, f := range m.Factors {
			if p, ok := f.(*PowNode); ok {
				if r, ok := p.Exp.(*RationalNode); ok && r.Val.Sign() < 0 {
					posExp := mustRational(-r.Val.Num().Int64(), r.Val.Denom().Int64())
					if posExp.Val.Cmp(big.NewRat(1, 1)) == 0 {
						denFactors = append(denFactors, p.Base)
					} else {
						denFactors = append(denFactors, &PowNode{Base: p.Base, Exp: posExp})
					}
					continue
				}
			}
			numFactors = append(numFactors, f)
		}
		var num Node = mustRational(1, 1)
		if len(numFactors) > 0 {
			num, _ = simplifyMul(numFactors)
		}
		var den Node = mustRational(1, 1)
		if len(denFactors) > 0 {
			den, _ = simplifyMul(denFactors)
		}
		return num, den
	}
	return expr, mustRational(1, 1)
}

// isOne checks if node represents constant 1.
func isOne(n Node) bool {
	if r, ok := n.(*RationalNode); ok {
		return r.Val.Cmp(big.NewRat(1, 1)) == 0
	}
	return false
}

// tryCancelLinearFactor attempts polynomial division of num and den by (varName - a).
// If both have remainder 0, returns the cancelled fraction and true.
func tryCancelLinearFactor(num, den Node, varName string, a Node) (Node, bool) {
	polyNum, okNum := extractPoly(num, varName)
	polyDen, okDen := extractPoly(den, varName)
	if !okNum || !okDen {
		return nil, false
	}

	// Divisor is x - a: degree 1, coeffs = [-a, 1]
	negA, err := simplifyUnaryOp("-", a)
	if err != nil {
		return nil, false
	}
	negAEval, _ := Eval(negA)
	divisor := &univariatePoly{
		varName: varName,
		coeffs:  []Node{negAEval, mustRational(1, 1)},
	}

	quotNum, remNum, okDiv1 := polyDivide(polyNum, divisor)
	quotDen, remDen, okDiv2 := polyDivide(polyDen, divisor)

	if okDiv1 && okDiv2 && isPolyZero(remNum) && isPolyZero(remDen) {
		newNumNode := quotNum.toNode()
		newDenNode := quotDen.toNode()
		invDen, _ := simplifyPow(newDenNode, mustRational(-1, 1))
		res, _ := simplifyMul([]Node{newNumNode, invDen})
		return res, true
	}

	return nil, false
}

// evaluateSingularityLimit handles the C / 0 case where C != 0.
func evaluateSingularityLimit(numVal Node, den Node, varName string, a Node, dir int) (Node, error) {
	numSign := getNodeSign(numVal)

	// Test sign of denominator approaching from right or left
	switch dir {
	case 1:
		denSign := probeDenSign(den, varName, a, 1)
		totalSign := numSign * denSign
		if totalSign > 0 {
			return &VarNode{Name: "inf"}, nil
		}
		return simplifyUnaryOp("-", &VarNode{Name: "inf"})
	case -1:
		denSign := probeDenSign(den, varName, a, -1)
		totalSign := numSign * denSign
		if totalSign > 0 {
			return &VarNode{Name: "inf"}, nil
		}
		return simplifyUnaryOp("-", &VarNode{Name: "inf"})
	}

	// Two-sided limit: if right and left signs match, limit is inf or -inf, else undefined
	denSignRight := probeDenSign(den, varName, a, 1)
	denSignLeft := probeDenSign(den, varName, a, -1)

	if denSignRight == denSignLeft && denSignRight != 0 {
		totalSign := numSign * denSignRight
		if totalSign > 0 {
			return &VarNode{Name: "inf"}, nil
		}
		return simplifyUnaryOp("-", &VarNode{Name: "inf"})
	}

	return nil, fmt.Errorf("%s", i18n.T("errors.err_limit_two_sided_limit_does"))
}

// probeDenSign checks the algebraic sign of den when varName = a + delta (dir > 0) or a - delta (dir < 0).
func probeDenSign(den Node, varName string, a Node, dir int) int {
	polyDen, ok := extractPoly(den, varName)
	if ok {
		// Division by (x - a)^k
		negA, _ := simplifyUnaryOp("-", a)
		negAEval, _ := Eval(negA)
		divisor := &univariatePoly{
			varName: varName,
			coeffs:  []Node{negAEval, mustRational(1, 1)},
		}

		k := 0
		cur := polyDen
		for {
			quot, rem, okDiv := polyDivide(cur, divisor)
			if okDiv && isPolyZero(rem) {
				k++
				cur = quot
			} else {
				break
			}
		}
		if k > 0 {
			// Sign of cur at x = a
			curNode := cur.toNode()
			subCur := Substitute(curNode, varName, a)
			valCur, err := Eval(subCur)
			if err == nil {
				leadSign := getNodeSign(valCur)
				if dir > 0 {
					return leadSign
				}
				// dir < 0: (-1)^k
				if k%2 == 0 {
					return leadSign
				}
				return -leadSign
			}
		}
	}
	return dir
}

// isValidLimitValue checks if node is a determined numerical/symbolic value.
func isValidLimitValue(n Node) bool {
	if n == nil {
		return false
	}
	if _, ok := n.(*RationalNode); ok {
		return true
	}
	return true
}

// getNodeSign extracts the sign (+1, -1, 0) of a node.
func getNodeSign(n Node) int {
	if n == nil {
		return 1
	}
	if r, ok := n.(*RationalNode); ok {
		return r.Val.Sign()
	}
	if u, ok := n.(*UnaryOpNode); ok && u.Op == "-" {
		return -getNodeSign(u.Expr)
	}
	return 1
}

// univariatePoly represents a single-variable polynomial: sum_{i=0}^n coeffs[i] * varName^i.
type univariatePoly struct {
	varName string
	coeffs  []Node // coeffs[0] is constant, coeffs[n] is lead coeff
}

func (p *univariatePoly) degree() int {
	return len(p.coeffs) - 1
}

func (p *univariatePoly) leadCoeff() Node {
	if len(p.coeffs) == 0 {
		return mustRational(0, 1)
	}
	return p.coeffs[len(p.coeffs)-1]
}

func (p *univariatePoly) toNode() Node {
	if len(p.coeffs) == 0 {
		return mustRational(0, 1)
	}
	var terms []Node
	for i, c := range p.coeffs {
		if isZero(c) {
			continue
		}
		switch i {
		case 0:
			terms = append(terms, c)
		case 1:
			pNode, _ := simplifyMul([]Node{c, &VarNode{Name: p.varName}})
			terms = append(terms, pNode)
		default:
			pNode, _ := simplifyMul([]Node{c, &PowNode{Base: &VarNode{Name: p.varName}, Exp: mustRational(int64(i), 1)}})
			terms = append(terms, pNode)
		}
	}
	if len(terms) == 0 {
		return mustRational(0, 1)
	}
	res, _ := simplifyAdd(terms)
	return res
}

func isPolyZero(p *univariatePoly) bool {
	if p == nil || len(p.coeffs) == 0 {
		return true
	}
	for _, c := range p.coeffs {
		if !isZero(c) {
			return false
		}
	}
	return true
}

// extractPoly extracts univariate polynomial coefficients in varName.
func extractPoly(n Node, varName string) (*univariatePoly, bool) {
	if n == nil {
		return nil, false
	}
	if !containsVar(n, varName) {
		val, err := Eval(n)
		if err != nil {
			return nil, false
		}
		return &univariatePoly{varName: varName, coeffs: []Node{val}}, true
	}

	switch v := n.(type) {
	case *VarNode:
		if v.Name == varName {
			return &univariatePoly{varName: varName, coeffs: []Node{mustRational(0, 1), mustRational(1, 1)}}, true
		}
		return &univariatePoly{varName: varName, coeffs: []Node{v}}, true

	case *RationalNode:
		return &univariatePoly{varName: varName, coeffs: []Node{v}}, true

	case *UnaryOpNode:
		if v.Op == "-" {
			inner, ok := extractPoly(v.Expr, varName)
			if !ok {
				return nil, false
			}
			newCoeffs := make([]Node, len(inner.coeffs))
			for i, c := range inner.coeffs {
				neg, err := simplifyUnaryOp("-", c)
				if err != nil {
					return nil, false
				}
				newCoeffs[i], _ = Eval(neg)
			}
			return &univariatePoly{varName: varName, coeffs: newCoeffs}, true
		}

	case *AddNode:
		var maxDeg int = 0
		var subPolys []*univariatePoly
		for _, t := range v.Terms {
			sp, ok := extractPoly(t, varName)
			if !ok {
				return nil, false
			}
			if sp.degree() > maxDeg {
				maxDeg = sp.degree()
			}
			subPolys = append(subPolys, sp)
		}
		merged := make([]Node, maxDeg+1)
		for i := 0; i <= maxDeg; i++ {
			merged[i] = mustRational(0, 1)
		}
		for _, sp := range subPolys {
			for i, c := range sp.coeffs {
				add, err := simplifyAdd([]Node{merged[i], c})
				if err != nil {
					return nil, false
				}
				merged[i], _ = Eval(add)
			}
		}
		// Trim leading zero coefficients
		return trimPoly(&univariatePoly{varName: varName, coeffs: merged}), true

	case *MulNode:
		var curPoly = &univariatePoly{varName: varName, coeffs: []Node{mustRational(1, 1)}}
		for _, f := range v.Factors {
			fp, ok := extractPoly(f, varName)
			if !ok {
				return nil, false
			}
			// Multiply curPoly by fp
			newDeg := curPoly.degree() + fp.degree()
			newCoeffs := make([]Node, newDeg+1)
			for i := 0; i <= newDeg; i++ {
				newCoeffs[i] = mustRational(0, 1)
			}
			for i, c1 := range curPoly.coeffs {
				for j, c2 := range fp.coeffs {
					prod, err := simplifyMul([]Node{c1, c2})
					if err != nil {
						return nil, false
					}
					prodVal, _ := Eval(prod)
					sum, _ := simplifyAdd([]Node{newCoeffs[i+j], prodVal})
					newCoeffs[i+j], _ = Eval(sum)
				}
			}
			curPoly = trimPoly(&univariatePoly{varName: varName, coeffs: newCoeffs})
		}
		return curPoly, true

	case *PowNode:
		if v.Base != nil && containsVar(v.Base, varName) && !containsVar(v.Exp, varName) {
			if r, ok := v.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() > 0 {
				deg := r.Val.Num().Int64()
				basePoly, ok := extractPoly(v.Base, varName)
				if !ok {
					return nil, false
				}
				resPoly := &univariatePoly{varName: varName, coeffs: []Node{mustRational(1, 1)}}
				for i := int64(0); i < deg; i++ {
					// multiply resPoly by basePoly
					newDeg := resPoly.degree() + basePoly.degree()
					newCoeffs := make([]Node, newDeg+1)
					for k := 0; k <= newDeg; k++ {
						newCoeffs[k] = mustRational(0, 1)
					}
					for rIdx, rCoeff := range resPoly.coeffs {
						for bIdx, bCoeff := range basePoly.coeffs {
							prod, _ := simplifyMul([]Node{rCoeff, bCoeff})
							prodVal, _ := Eval(prod)
							s, _ := simplifyAdd([]Node{newCoeffs[rIdx+bIdx], prodVal})
							newCoeffs[rIdx+bIdx], _ = Eval(s)
						}
					}
					resPoly = trimPoly(&univariatePoly{varName: varName, coeffs: newCoeffs})
				}
				return resPoly, true
			}
		}
	}

	return nil, false
}

func trimPoly(p *univariatePoly) *univariatePoly {
	for len(p.coeffs) > 1 && isZero(p.coeffs[len(p.coeffs)-1]) {
		p.coeffs = p.coeffs[:len(p.coeffs)-1]
	}
	return p
}

// polyDivide performs polynomial division: A = B * Q + R.
func polyDivide(A, B *univariatePoly) (Q, R *univariatePoly, ok bool) {
	if isPolyZero(B) {
		return nil, nil, false
	}
	degA := A.degree()
	degB := B.degree()
	if degA < degB {
		return &univariatePoly{varName: A.varName, coeffs: []Node{mustRational(0, 1)}}, A, true
	}

	// Working copy of A's coefficients
	rCoeffs := make([]Node, len(A.coeffs))
	copy(rCoeffs, A.coeffs)

	qCoeffs := make([]Node, degA-degB+1)
	for i := range qCoeffs {
		qCoeffs[i] = mustRational(0, 1)
	}

	leadB := B.leadCoeff()

	for curDeg := degA; curDeg >= degB; curDeg-- {
		leadR := rCoeffs[curDeg]
		if isZero(leadR) {
			continue
		}
		invLeadB, err := simplifyPow(leadB, mustRational(-1, 1))
		if err != nil {
			return nil, nil, false
		}
		quotCoeff, err := simplifyMul([]Node{leadR, invLeadB})
		if err != nil {
			return nil, nil, false
		}
		quotVal, err := Eval(quotCoeff)
		if err != nil {
			return nil, nil, false
		}

		qIdx := curDeg - degB
		qCoeffs[qIdx] = quotVal

		for bIdx, bVal := range B.coeffs {
			prod, _ := simplifyMul([]Node{quotVal, bVal})
			prodVal, _ := Eval(prod)
			negProd, _ := simplifyUnaryOp("-", prodVal)
			newR, _ := simplifyAdd([]Node{rCoeffs[qIdx+bIdx], negProd})
			rCoeffs[qIdx+bIdx], _ = Eval(newR)
		}
	}

	Q = trimPoly(&univariatePoly{varName: A.varName, coeffs: qCoeffs})
	R = trimPoly(&univariatePoly{varName: A.varName, coeffs: rCoeffs})
	return Q, R, true
}
