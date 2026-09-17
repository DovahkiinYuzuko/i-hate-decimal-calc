package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Complex Analysis & Symbolic Residue Calculus Pack
// (Cauchy 1825, Laurent 1843, Bronstein 1997)
// -------------------------------------------------------------------------

// EvalResidue computes the exact algebraic residue of expr with respect to z at singularity z0:
// Res(f(z), z0) = a_{-1} (coefficient of (z - z0)^-1 in Laurent series).
func EvalResidue(expr, zNode, z0Node Node, env *Env) (Node, error) {
	if expr == nil || zNode == nil || z0Node == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.residue_var_required", "nil"))
	}

	zVarNode, ok := zNode.(*VarNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.residue_var_required", zNode.String()))
	}
	zVar := zVarNode.Name

	evalZ0, err := EvalWithEnv(z0Node, env)
	if err != nil {
		evalZ0 = z0Node
	}

	evalExpr, err := EvalWithEnv(expr, env)
	if err != nil {
		evalExpr = expr
	}

	// 1. Check if expr even contains the variable
	if !containsVar(evalExpr, zVar) {
		// Constant with respect to z has residue 0
		return mustRational(0, 1), nil
	}

	// 2. Try rational function shortcut via partial fraction decomposition (apart)
	if res, ok := tryRationalResidue(evalExpr, zVar, evalZ0, env); ok {
		return res, nil
	}

	// 3. Variable shift to origin: w = z - z0 => z = z0 + w
	// This standardizes all residue evaluations to w = 0.
	wVar := generateUniqueVarName(evalExpr, "_w")
	shiftExpr := &AddNode{Terms: []Node{evalZ0, &VarNode{Name: wVar}}}
	gw := substVar(evalExpr, zVar, shiftExpr)
	gwEval, err := EvalWithEnv(gw, env)
	if err == nil {
		gw = gwEval
	}

	// 4. Check if z0 is regular or removable singularity
	// Direct substitution:
	substZero := substVar(gw, wVar, mustRational(0, 1))
	valZero, errZero := EvalWithEnv(substZero, env)
	if errZero == nil && !isInfiniteNode(valZero) && !isIndeterminate(valZero) {
		// No singularity at z0
		return mustRational(0, 1), nil
	}

	// Limit at w -> 0:
	limZero, errLimZero := EvalLimit(gw, &VarNode{Name: wVar}, mustRational(0, 1), nil)
	if errLimZero == nil && !isInfiniteNode(limZero) && !isIndeterminate(limZero) {
		// Removable singularity: limit is finite, residue is 0
		return mustRational(0, 1), nil
	}

	// 4. Pole order search and residue computation:
	// Try order m = 1, 2, ..., 10
	maxOrder := 10
	for m := 1; m <= maxOrder; m++ {
		// H_m(w) = w^m * g(w) with pole factor cancellation
		Hw := cancelPoleFactor(gw, wVar, m, env)

		_, denH := extractNumeratorDenominator(Hw)
		if containsVar(denH, wVar) {
			// Denominator still has pole factor at w = 0, so order m is too small
			continue
		}

		// Check if H_m(w) is regular at w = 0 via direct substitution first
		substH := substVar(Hw, wVar, mustRational(0, 1))
		limH, errLim := EvalWithEnv(substH, env)
		if errLim != nil || isInfiniteNode(limH) || isIndeterminate(limH) {
			// Fall back to limit only if direct substitution is indeterminate
			limH, errLim = EvalLimit(Hw, &VarNode{Name: wVar}, mustRational(0, 1), nil)
			if errLim != nil || isInfiniteNode(limH) || isIndeterminate(limH) {
				// Not yet removable at order m, try higher m
				continue
			}
		}

		// At this order m, H_m(w) is regular at w = 0.
		// If m == 1: Res = lim_{w -> 0} [w * g(w)]
		if m == 1 {
			return limH, nil
		}

		// If m >= 2: Cauchy residue formula:
		// Res = (1 / (m-1)!) * (d^{m-1}/dw^{m-1} [w^m * g(w)]) |_{w=0}
		cur := Hw
		diffSuccess := true
		for k := 0; k < m-1; k++ {
			diffNode, errD := differentiate(cur, wVar)
			if errD != nil {
				diffSuccess = false
				break
			}
			curEval, errE := EvalWithEnv(expandNode(diffNode), env)
			if errE != nil {
				cur = expandNode(diffNode)
			} else {
				cur = curEval
			}
		}
		if !diffSuccess {
			continue
		}

		// Since H_m is regular at w = 0, its derivatives are also regular. Direct evaluation is exact & instant!
		substDiff := substVar(cur, wVar, mustRational(0, 1))
		limDiff, errDiff := EvalWithEnv(substDiff, env)
		if errDiff != nil || isInfiniteNode(limDiff) || isIndeterminate(limDiff) {
			limDiff, errDiff = EvalLimit(cur, &VarNode{Name: wVar}, mustRational(0, 1), nil)
			if errDiff != nil || isInfiniteNode(limDiff) || isIndeterminate(limDiff) {
				continue
			}
		}

		fact := factorialInt(m - 1)
		invFact := &RationalNode{Val: new(big.Rat).SetFrac(big.NewInt(1), fact)}
		resMul, errM := simplifyMul([]Node{invFact, limDiff})
		if errM != nil {
			return limDiff, nil
		}
		resFinal, errF := EvalWithEnv(resMul, env)
		if errF != nil {
			return resMul, nil
		}
		return resFinal, nil
	}

	return nil, fmt.Errorf("%s", i18n.T("errors.residue_failed", z0Node.String(), i18n.T("errors.residue_order_exceeded")))
}

// -------------------------------------------------------------------------
// Helper Functions for Residue Calculus
// -------------------------------------------------------------------------

// substVar substitutes all occurrences of variable varName with replacement node.
func substVar(n Node, varName string, replacement Node) Node {
	if n == nil {
		return nil
	}
	switch v := n.(type) {
	case *VarNode:
		if v.Name == varName {
			return replacement
		}
		return v
	case *UnaryOpNode:
		return &UnaryOpNode{Op: v.Op, Expr: substVar(v.Expr, varName, replacement)}
	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = substVar(t, varName, replacement)
		}
		return &AddNode{Terms: newTerms}
	case *MulNode:
		newFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			newFactors[i] = substVar(f, varName, replacement)
		}
		return &MulNode{Factors: newFactors}
	case *PowNode:
		return &PowNode{
			Base: substVar(v.Base, varName, replacement),
			Exp:  substVar(v.Exp, varName, replacement),
		}
	case *FuncNode:
		newArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			newArgs[i] = substVar(a, varName, replacement)
		}
		return &FuncNode{Name: v.Name, Args: newArgs}
	case *ListNode:
		newElems := make([]Node, len(v.Elements))
		for i, e := range v.Elements {
			newElems[i] = substVar(e, varName, replacement)
		}
		return &ListNode{Elements: newElems}
	default:
		return n
	}
}

// isInfiniteNode checks if n is infinity or -infinity.
func isInfiniteNode(n Node) bool {
	if n == nil {
		return false
	}
	isInf, _ := isInfiniteTarget(n)
	return isInf
}

// generateUniqueVarName generates a variable name not present in expr.
func generateUniqueVarName(n Node, prefix string) string {
	usedVars := collectVariables(n)
	usedMap := make(map[string]bool)
	for _, v := range usedVars {
		usedMap[v] = true
	}

	candidate := prefix
	idx := 1
	for usedMap[candidate] {
		candidate = fmt.Sprintf("%s%d", prefix, idx)
		idx++
	}
	return candidate
}

// factorialInt returns n! as *big.Int.
func factorialInt(n int) *big.Int {
	if n <= 1 {
		return big.NewInt(1)
	}
	res := big.NewInt(1)
	for i := 2; i <= n; i++ {
		res.Mul(res, big.NewInt(int64(i)))
	}
	return res
}

// isIndeterminate checks if a node represents an indeterminate form or undefined error node.
func isIndeterminate(n Node) bool {
	if n == nil {
		return true
	}
	if v, ok := n.(*VarNode); ok {
		return v.Name == "NaN" || v.Name == "undefined"
	}
	return false
}

// tryRationalResidue attempts to compute the residue of a rational function P(z)/Q(z).
// If expr is a ratio of polynomials in zVar, it shifts by z = z0 + w and uses
// the formal Laurent/Taylor power series expansion of H(w) = w^m * f(w) to directly
// extract the residue without any limit evaluations or L'Hôpital cycles.
func tryRationalResidue(expr Node, zVar string, z0 Node, env *Env) (Node, bool) {
	numNode, denNode := extractNumeratorDenominator(expr)
	_, okNum := extractPoly(numNode, zVar)
	polyDen, okDen := extractPoly(denNode, zVar)
	if !okNum || !okDen || isPolyZero(polyDen) {
		// Not a pure polynomial rational function in zVar (e.g. contains exp, sin, etc.)
		return nil, false
	}

	evalEnv := env
	if evalEnv == nil {
		evalEnv = NewEnv()
	} else {
		evalEnv = evalEnv.Clone()
	}
	if _, ok := evalEnv.Get("i"); !ok {
		evalEnv.Set("i", NewComplex(mustRational(0, 1), mustRational(1, 1)))
	}

	// 1. Shift to origin: w = z - z0 => z = z0 + w
	wVar := generateUniqueVarName(expr, "_w")
	z0Ast := complexToAst(z0)
	shiftExpr := &AddNode{Terms: []Node{z0Ast, &VarNode{Name: wVar}}}
	PwNode := substVar(numNode, zVar, shiftExpr)
	QwNode := substVar(denNode, zVar, shiftExpr)

	PwEval, errP := EvalWithEnv(expandNode(PwNode), evalEnv)
	if errP != nil {
		PwEval = expandNode(PwNode)
	}
	QwEval, errQ := EvalWithEnv(expandNode(QwNode), evalEnv)
	if errQ != nil {
		QwEval = expandNode(QwNode)
	}

	PwEval = expandComplexToAst(PwEval, wVar)
	QwEval = expandComplexToAst(QwEval, wVar)

	polyPw, okPw := extractPoly(PwEval, wVar)
	polyQw, okQw := extractPoly(QwEval, wVar)
	if !okPw || !okQw || isPolyZero(polyQw) {
		return nil, false
	}

	// 2. Find valuation of w in Qw and Pw
	valP := polyValuation(polyPw, evalEnv)
	valQ := polyValuation(polyQw, evalEnv)

	if valQ <= valP {
		// Regular or removable singularity at w = 0 (z = z0)
		return mustRational(0, 1), true
	}

	// Pole order m = valQ - valP
	m := valQ - valP
	if m <= 0 {
		return mustRational(0, 1), true
	}

	// Reduced polynomials: P_red = Pw / w^valP, Q_red = Qw / w^valQ
	// Both have non-zero constant term (q0 != 0, p0 != 0)
	polyP_red := polyShiftDegree(polyPw, -valP)
	polyQ_red := polyShiftDegree(polyQw, -valQ)

	q0 := polyQ_red.coeffs[0]

	// Compute Taylor coefficients a_0, ..., a_{m-1} of P_red(w) / Q_red(w)
	// a_k = (p_k - sum_{j=0}^{k-1} a_j * q_{k-j}) / q_0
	a := make([]Node, m)
	for k := 0; k < m; k++ {
		var pk Node = mustRational(0, 1)
		if k < len(polyP_red.coeffs) {
			pk = polyP_red.coeffs[k]
		}

		var sumTerms []Node
		for j := 0; j < k; j++ {
			qIdx := k - j
			if qIdx < len(polyQ_red.coeffs) {
				prod, errMul := simplifyMul([]Node{a[j], polyQ_red.coeffs[qIdx]})
				if errMul == nil {
					sumTerms = append(sumTerms, prod)
				}
			}
		}

		var numerator Node = pk
		if len(sumTerms) > 0 {
			convSum, _ := simplifyAdd(sumTerms)
			negSum, _ := simplifyUnaryOp("-", convSum)
			diffNode, _ := simplifyAdd([]Node{pk, negSum})
			numerator = diffNode
		}

		// Divide by q0: numerator / q0 = numerator * (q0)^-1
		invQ0 := &PowNode{Base: q0, Exp: mustRational(-1, 1)}
		akNode, errAk := simplifyMul([]Node{numerator, invQ0})
		if errAk != nil {
			return nil, false
		}
		akEval, errEval := EvalWithEnv(akNode, evalEnv)
		if errEval != nil {
			a[k] = akNode
		} else {
			a[k] = akEval
		}
	}

	// The residue is exactly the coefficient of w^(m-1), which is a[m-1]
	res := a[m-1]
	finalRes, errF := EvalWithEnv(res, evalEnv)
	if errF != nil {
		return res, true
	}
	return finalRes, true
}

// isAlgebraicZero checks if a node evaluates algebraically to zero.
func isAlgebraicZero(n Node, env *Env) bool {
	if n == nil || isZero(n) {
		return true
	}
	evalEnv := env
	if evalEnv == nil {
		evalEnv = NewEnv()
	} else {
		evalEnv = evalEnv.Clone()
	}
	if _, ok := evalEnv.Get("i"); !ok {
		evalEnv.Set("i", NewComplex(mustRational(0, 1), mustRational(1, 1)))
	}

	evalN, err := EvalWithEnv(n, evalEnv)
	if err == nil && isZero(evalN) {
		return true
	}
	simp, err := Eval(n)
	if err == nil && isZero(simp) {
		return true
	}
	return false
}

// polyValuation returns the lowest degree k such that coeffs[k] is non-zero.
func polyValuation(p *univariatePoly, env *Env) int {
	if p == nil {
		return 0
	}
	for i, c := range p.coeffs {
		if !isAlgebraicZero(c, env) {
			return i
		}
	}
	return len(p.coeffs)
}

// polyShiftDegree shifts the polynomial degree by shift (e.g. shift = -k drops lowest k terms).
func polyShiftDegree(p *univariatePoly, shift int) *univariatePoly {
	if p == nil || isPolyZero(p) {
		return &univariatePoly{varName: p.varName, coeffs: []Node{mustRational(0, 1)}}
	}
	if shift < 0 {
		drop := -shift
		if drop >= len(p.coeffs) {
			return &univariatePoly{varName: p.varName, coeffs: []Node{mustRational(0, 1)}}
		}
		return &univariatePoly{varName: p.varName, coeffs: p.coeffs[drop:]}
	}
	if shift > 0 {
		newCoeffs := make([]Node, shift+len(p.coeffs))
		for i := 0; i < shift; i++ {
			newCoeffs[i] = mustRational(0, 1)
		}
		copy(newCoeffs[shift:], p.coeffs)
		return &univariatePoly{varName: p.varName, coeffs: newCoeffs}
	}
	return p
}

// expandComplexToAst unpacks any ComplexNode containing varName into Real + Imag * i.
// This allows extractPoly and polynomial algebra to process complex variable expressions
// as standard polynomials with imaginary unit coefficients.
func expandComplexToAst(n Node, varName string) Node {
	if n == nil {
		return nil
	}
	switch v := n.(type) {
	case *ComplexNode:
		realNode := expandComplexToAst(v.Real, varName)
		imagNode := expandComplexToAst(v.Imag, varName)
		iVar := &VarNode{Name: "i"}
		return &AddNode{Terms: []Node{realNode, &MulNode{Factors: []Node{imagNode, iVar}}}}
	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = expandComplexToAst(t, varName)
		}
		return &AddNode{Terms: newTerms}
	case *MulNode:
		newFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			newFactors[i] = expandComplexToAst(f, varName)
		}
		return &MulNode{Factors: newFactors}
	case *PowNode:
		return &PowNode{
			Base: expandComplexToAst(v.Base, varName),
			Exp:  expandComplexToAst(v.Exp, varName),
		}
	default:
		return n
	}
}

// complexToAst converts any ComplexNode in a constant expression to symbolic AddNode/MulNode with VarNode("i").
func complexToAst(n Node) Node {
	if n == nil {
		return nil
	}
	switch v := n.(type) {
	case *ComplexNode:
		realNode := complexToAst(v.Real)
		imagNode := complexToAst(v.Imag)
		iVar := &VarNode{Name: "i"}
		if isZero(realNode) {
			if isOne(imagNode) {
				return iVar
			}
			prod, _ := simplifyMul([]Node{imagNode, iVar})
			return prod
		}
		prod, _ := simplifyMul([]Node{imagNode, iVar})
		add, _ := simplifyAdd([]Node{realNode, prod})
		return add
	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = complexToAst(t)
		}
		return &AddNode{Terms: newTerms}
	case *MulNode:
		newFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			newFactors[i] = complexToAst(f)
		}
		return &MulNode{Factors: newFactors}
	default:
		return n
	}
}

// extractVarPower returns the integer power k if n is equivalent to varName^k.
func extractVarPower(n Node, varName string) (int, bool) {
	if n == nil {
		return 0, false
	}
	if v, ok := n.(*VarNode); ok && v.Name == varName {
		return 1, true
	}
	if p, ok := n.(*PowNode); ok {
		if r, okR := p.Exp.(*RationalNode); okR && r.Val.IsInt() && r.Val.Sign() > 0 {
			expInt := int(r.Val.Num().Int64())
			if innerK, okInner := extractVarPower(p.Base, varName); okInner {
				return innerK * expInt, true
			}
		}
	}
	return 0, false
}

// cancelPoleFactor constructs w^m * gw while canceling exact power factors of w in the denominator.
// This guarantees that H_m(w) does not retain uncancelled w^m / w^k indeterminate forms.
func cancelPoleFactor(gw Node, wVar string, m int, env *Env) Node {
	num, den := extractNumeratorDenominator(gw)

	// Extract power of w from denominator: den = w^k * denRest
	k := 0
	var denRestFactors []Node

	if power, ok := extractVarPower(den, wVar); ok {
		k = power
	} else if denMul, ok := den.(*MulNode); ok {
		for _, f := range denMul.Factors {
			if power, okP := extractVarPower(f, wVar); okP {
				k += power
			} else {
				denRestFactors = append(denRestFactors, f)
			}
		}
	} else {
		denRestFactors = []Node{den}
	}

	if k == 0 {
		// No pure w^k factor isolated, fall back to general product
		wm := &PowNode{Base: &VarNode{Name: wVar}, Exp: mustRational(int64(m), 1)}
		prod, _ := simplifyMul([]Node{wm, gw})
		evalP, err := EvalWithEnv(expandNode(prod), env)
		if err == nil {
			return evalP
		}
		return prod
	}

	var newDen Node = mustRational(1, 1)
	if len(denRestFactors) > 0 {
		newDen, _ = simplifyMul(denRestFactors)
	}

	var newNum Node = num
	if m > k {
		diffExp := m - k
		wPow := &PowNode{Base: &VarNode{Name: wVar}, Exp: mustRational(int64(diffExp), 1)}
		newNum, _ = simplifyMul([]Node{wPow, num})
	} else if m < k {
		diffExp := k - m
		wPow := &PowNode{Base: &VarNode{Name: wVar}, Exp: mustRational(int64(diffExp), 1)}
		newDen, _ = simplifyMul([]Node{wPow, newDen})
	}

	if isOne(newDen) {
		evalN, err := EvalWithEnv(newNum, env)
		if err == nil {
			return evalN
		}
		return newNum
	}

	invDen := &PowNode{Base: newDen, Exp: mustRational(-1, 1)}
	prod, _ := simplifyMul([]Node{newNum, invDen})
	evalP, err := EvalWithEnv(prod, env)
	if err == nil {
		return evalP
	}
	return prod
}





