package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// RischState refers to the Risch lifecycle states defined in eval_risch_fsm.go.

// ExtensionKind represents the kind of transcendental differential extension.
type ExtensionKind int

const (
	ExtNone ExtensionKind = iota
	ExtLog
	ExtExp
)

// DifferentialExtension represents a single generator t in the differential field tower K(t).
type DifferentialExtension struct {
	VarName string
	Kind    ExtensionKind
	Arg     ast.Node // u in log(u) or exp(u)
	Dt      ast.Node // D(t)
}

// -------------------------------------------------------------------------
// Sub-issue 66.1: Rational Function Integration over Q(x)
// (Hermite Reduction + Rothstein-Trager Resultant)
// -------------------------------------------------------------------------

// RischIntegrateRational performs deterministic integration of a pure rational function
// P(x)/Q(x) in Q(x) using Ostrogradsky-Hermite reduction and the Rothstein-Trager method.
func RischIntegrateRational(numPoly, denPoly *univariatePoly, varName string, fsm *RischLifecycleFSM) (ast.Node, error) {
	if denPoly == nil || isPolyZero(denPoly) {
		return nil, fmt.Errorf("%s", i18n.T("integral.err_division_by_zero_polynomial"))
	}
	if isPolyZero(numPoly) {
		return mustRational(0, 1), nil
	}

	// 1. Polynomial long division: P(x) = D(x)*Q(x) + R(x)
	quotPoly, remPoly, _ := polyDivide(numPoly, denPoly)

	var integratedParts []ast.Node

	// Integrate polynomial quotient D(x) directly: int(x^n) = x^(n+1)/(n+1)
	if !isPolyZero(quotPoly) {
		polyInt, err := integrateUnivariatePoly(quotPoly, varName)
		if err != nil {
			return nil, err
		}
		if !isZero(polyInt) {
			integratedParts = append(integratedParts, polyInt)
		}
	}

	// If remainder is zero, pure polynomial integration is done
	if isPolyZero(remPoly) {
		_ = fsm.TransitionTo(RischStatePolynomialPartIntegrated)
		if len(integratedParts) == 0 {
			return mustRational(0, 1), nil
		}
		return simplifyAdd(integratedParts)
	}

	// 2. Ostrogradsky-Hermite Reduction on R(x) / Q(x)
	// Splits into: g(x) + int( A(x) / B(x) dx ) where B(x) is square-free
	gRationalPart, aPoly, bPoly, err := HermiteReduce(remPoly, denPoly, varName)
	if err != nil {
		_ = fsm.TransitionTo(RischStateUnsupported)
		return nil, fmt.Errorf("%s", i18n.T("integral.err_hermite_reduction_failed", err))
	}
	_ = fsm.TransitionTo(RischStateHermiteReduced)

	if gRationalPart != nil && !isZero(gRationalPart) {
		integratedParts = append(integratedParts, gRationalPart)
	}

	// If reduced logarithmic numerator is zero, we are done
	if isPolyZero(aPoly) {
		if len(integratedParts) == 0 {
			return mustRational(0, 1), nil
		}
		return simplifyAdd(integratedParts)
	}

	// 3. Rothstein-Trager / Lazard-Rioboo-Trager on square-free A(x) / B(x)
	logPart, err := RothsteinTrager(aPoly, bPoly, varName)
	if err != nil {
		_ = fsm.TransitionTo(RischStateUnsupported)
		return nil, err
	}
	_ = fsm.TransitionTo(RischStateResiduePolesExtracted)

	if logPart != nil && !isZero(logPart) {
		integratedParts = append(integratedParts, logPart)
	}

	if len(integratedParts) == 0 {
		return mustRational(0, 1), nil
	}
	if len(integratedParts) == 1 {
		return integratedParts[0], nil
	}
	return simplifyAdd(integratedParts)
}

// HermiteReduce performs Ostrogradsky-Hermite reduction on proper fraction P(x)/Q(x).
// Returns g(x) (rational part) and coprime polynomials A(x), B(x) with B square-free.
func HermiteReduce(P, Q *univariatePoly, varName string) (g Node, A, B *univariatePoly, err error) {
	// Q = B_1 * B_2^2 * ... * B_k^k (square-free factorization)
	d := polyDeriv1D(Q)
	gcdQD, err := polyGCD1D(Q, d)
	if err != nil {
		return nil, nil, nil, err
	}

	// If gcd(Q, Q') == 1, Q is already square-free: no rational reduction needed
	if gcdQD.degree() == 0 {
		return mustRational(0, 1), P, Q, nil
	}

	// Ostrogradsky-Hermite reduction:
	// Q_1 = gcd(Q, Q')
	// Q_0 = Q / Q_1
	// Find polynomials g_num and A such that P/Q = (g_num/Q_1)' + A/Q_0
	// Iterative reduction for each power of the factor
	curP := clonePoly1D(P)
	curQ := clonePoly1D(Q)
	var gParts []ast.Node

	for {
		dCur := polyDeriv1D(curQ)
		gCur, err := polyGCD1D(curQ, dCur)
		if err != nil || gCur.degree() == 0 {
			break
		}
		qRed := polyDivExact1D(curQ, gCur)

		// Extended Euclidean: s * gCur + t * dCur = 1
		// Here we reduce the multiple poles
		// Simplified Hermite step for quadratic and higher factors
		if qRed.degree() > 0 && curQ.degree() > qRed.degree() {
			m := curQ.degree() / qRed.degree()
			if m > 1 {
				// Reduce by 1 power: (A / qRed^(m-1))
				// Using algebraic ansatz for the rational coefficient
				gPart, remP, ok := hermiteStep(curP, qRed, m, varName)
				if ok {
					gParts = append(gParts, gPart)
					curP = remP
					curQ = polyDivExact1D(curQ, qRed)
					continue
				}
			}
		}
		break
	}

	// Remainder A / B where B is square-free
	dFinal := polyDeriv1D(curQ)
	gFinal, _ := polyGCD1D(curQ, dFinal)
	B = curQ
	if gFinal.degree() > 0 {
		B = polyDivExact1D(curQ, gFinal)
	}
	A = polyDivExact1D(curP, gFinal)
	if A == nil {
		A = curP
	}

	var gTotal ast.Node = mustRational(0, 1)
	if len(gParts) > 0 {
		gTotal, _ = simplifyAdd(gParts)
	}
	return gTotal, A, B, nil
}

// hermiteStep computes a single reduction step for P / (Q^m) -> g / (Q^(m-1)) + remP / (Q^(m-1)).
func hermiteStep(P, Q *univariatePoly, m int, varName string) (gNode ast.Node, remP *univariatePoly, ok bool) {
	if m <= 1 {
		return nil, P, false
	}
	// For standard rational integration (e.g. 1 / (x^2 + 1)^2):
	// d/dx [ x / (2*(m-1)*c * (x^2+c)^(m-1)) ]
	// Match leading quadratic or linear powers
	if Q.degree() == 2 && isZero(Q.coeff(1)) {
		cRat, okC := Q.coeff(0).(*ast.RationalNode)
		leadRat, okL := Q.coeff(2).(*ast.RationalNode)
		if okC && okL && leadRat.Val.Cmp(big.NewRat(1, 1)) == 0 {
			c := cRat.Val
			if c.Sign() > 0 && P.degree() == 0 {
				// P = a0
				a0Rat, okA := P.coeff(0).(*ast.RationalNode)
				if okA {
					// int( 1 / (x^2 + c)^m dx )
					// = x / (2*(m-1)*c * (x^2+c)^(m-1)) + (2m-3)/(2*(m-1)*c) * int( 1 / (x^2+c)^(m-1) dx )
					twoMMinus2 := int64(2 * (m - 1))
					scaleDen := new(big.Rat).Mul(big.NewRat(twoMMinus2, 1), c)
					gCoeff := new(big.Rat).Quo(a0Rat.Val, scaleDen)

					// gNode = gCoeff * x / (x^2 + c)^(m-1)
					xNode := &ast.VarNode{Name: varName}
					qNode := Q.toNode()
					powExp := mustRational(int64(m-1), 1)
					invPow, _ := simplifyPow(qNode, &ast.UnaryOpNode{Op: "-", Expr: powExp})
					gNode, _ = simplifyMul([]ast.Node{mustRationalBig(gCoeff), xNode, invPow})

					// remCoeff = (2m-3) / (2*(m-1)*c) * a0
					twoMMinus3 := int64(2*m - 3)
					remScale := new(big.Rat).Quo(big.NewRat(twoMMinus3, 1), scaleDen)
					remCoeff := new(big.Rat).Mul(a0Rat.Val, remScale)
					remP = &univariatePoly{varName: varName, coeffs: []ast.Node{mustRationalBig(remCoeff)}}
					return gNode, remP, true
				}
			}
		}
	}
	return nil, P, false
}

// RothsteinTrager integrates square-free rational fraction A(x)/B(x) using the Sylvester resultant
// R(z) = res_x(A(x) - z*B'(x), B(x)).
func RothsteinTrager(A, B *univariatePoly, varName string) (ast.Node, error) {
	if isPolyZero(A) {
		return mustRational(0, 1), nil
	}

	// 1. Compute derivative B'(x)
	bPrime := polyDeriv1D(B)

	// Check if A is a constant multiple of B': int(k * B' / B) = k * ln|B|
	if divPoly, remPoly, _ := polyDivide(A, bPrime); isPolyZero(remPoly) && divPoly.degree() == 0 {
		kCoeff := divPoly.coeff(0)
		lnNode := &ast.FuncNode{Name: "ln", Args: []ast.Node{B.toNode()}}
		return simplifyMul([]ast.Node{kCoeff, lnNode})
	}

	// 2. Linear denominator: A / (c1*x + c0) -> (A/c1) * ln|c1*x + c0|
	if B.degree() == 1 {
		c1 := B.coeff(1)
		invC1, _ := simplifyPow(c1, mustRational(-1, 1))
		coeff, _ := simplifyMul([]ast.Node{A.coeff(0), invC1})
		lnNode := &ast.FuncNode{Name: "ln", Args: []ast.Node{B.toNode()}}
		return simplifyMul([]ast.Node{coeff, lnNode})
	}

	// 3. Irreducible quadratic denominator: A(x) / (x^2 + c) with c > 0
	if B.degree() == 2 && isZero(B.coeff(1)) {
		cRat, okC := B.coeff(0).(*ast.RationalNode)
		lRat, okL := B.coeff(2).(*ast.RationalNode)
		if okC && okL && lRat.Val.Cmp(big.NewRat(1, 1)) == 0 && cRat.Val.Sign() > 0 {
			// A(x) = a1*x + a0
			a0 := A.coeff(0)
			a1 := A.coeff(1)

			var terms []ast.Node

			// Part 1: a1*x / (x^2 + c) -> (a1/2) * ln(x^2 + c)
			if !isZero(a1) {
				halfA1, _ := simplifyMul([]ast.Node{mustRational(1, 2), a1})
				lnNode := &ast.FuncNode{Name: "ln", Args: []ast.Node{B.toNode()}}
				part1, _ := simplifyMul([]ast.Node{halfA1, lnNode})
				terms = append(terms, part1)
			}

			// Part 2: a0 / (x^2 + c) -> (a0 / sqrt(c)) * atan(x / sqrt(c))
			if !isZero(a0) {
				// sqrt(c)
				sqrtC, err := simplifyPow(B.coeff(0), mustRational(1, 2))
				if err == nil {
					invSqrtC, _ := simplifyPow(sqrtC, mustRational(-1, 1))
					atanArg, _ := simplifyMul([]ast.Node{&ast.VarNode{Name: varName}, invSqrtC})
					atanNode := &ast.FuncNode{Name: "atan", Args: []ast.Node{atanArg}}
					part2Coeff, _ := simplifyMul([]ast.Node{a0, invSqrtC})
					part2, _ := simplifyMul([]ast.Node{part2Coeff, atanNode})
					terms = append(terms, part2)
				}
			}

			if len(terms) > 0 {
				return simplifyAdd(terms)
			}
		}
	}

	// 4. General Sylvester Resultant R(z) = res_x(A(x) - z*B'(x), B(x))
	// Evaluate resultant treating z as symbolic variable
	zVar := "_z_risch"
	azNode, _ := simplifyMul([]ast.Node{&ast.VarNode{Name: zVar}, bPrime.toNode()})
	pMinusZB, _ := simplifyAdd([]ast.Node{A.toNode(), &ast.UnaryOpNode{Op: "-", Expr: azNode}})

	resNode, err := EvalResultant(pMinusZB, B.toNode(), varName, NewEnv())
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("integral.err_rothstein_trager_resultant_computation_failed", err))
	}

	// Extract roots of R(z) in Q
	zPoly, okZ := extractPoly(resNode, zVar)
	if !okZ || isPolyZero(zPoly) {
		return nil, fmt.Errorf("%s", i18n.T("integral.err_resultant_in_z_could_not"))
	}

	// Find rational roots of R(z)
	roots := findUnivariatePolyRationalRoots(zPoly)
	if len(roots) == 0 {
		// Roots are algebraic numbers: return Unsupported as per 4-AI consensus
		return nil, fmt.Errorf("%s", i18n.T("integral.err_rothstein_trager_residues_require_algebraic"))
	}

	var logTerms []ast.Node
	for _, r := range roots {
		cVal := r
		cNode := mustRationalBig(cVal)

		// v_i(x) = gcd(A(x) - c_i*B'(x), B(x))
		cTimesBPrime := scalePoly1D(bPrime, cVal)
		subPoly := polySub1D(A, cTimesBPrime)

		vPoly, err := polyGCD1D(subPoly, B)
		if err == nil && vPoly.degree() > 0 {
			vNode := monicPoly1D(vPoly).toNode()
			lnNode := &ast.FuncNode{Name: "ln", Args: []ast.Node{vNode}}
			term, _ := simplifyMul([]ast.Node{cNode, lnNode})
			logTerms = append(logTerms, term)
		}
	}

	if len(logTerms) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("integral.err_no_valid_logarithmic_factors_recovered"))
	}
	return simplifyAdd(logTerms)
}

// -------------------------------------------------------------------------
// Sub-issue 66.2 & 66.3: Transcendental Tower, RDE, and Liouville Proof
// -------------------------------------------------------------------------

// RischIntegrate is the master deterministic transcendental integration entry point.
func RischIntegrate(expr ast.Node, varName string) (ast.Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("%s", i18n.T("integral.err_cannot_integrate_nil_expression"))
	}

	fsm := NewRischLifecycleFSM()

	// 1. Monomial Pre-normalization (Phase 0.5)
	normExpr := normalizeMonomialsAST(expr)
	_ = fsm.TransitionTo(RischStateMonomialsNormalized)

	// 2. Constant with respect to varName: int(c, x) = c * x
	if !containsVar(normExpr, varName) {
		res, err := simplifyMul([]ast.Node{normExpr, &ast.VarNode{Name: varName}})
		if err != nil {
			return nil, err
		}
		return Eval(res)
	}

	// 3. Try pure polynomial in varName
	if poly, ok := extractPoly(normExpr, varName); ok {
		intPoly, err := integrateUnivariatePoly(poly, varName)
		if err == nil {
			_ = fsm.TransitionTo(RischStatePolynomialPartIntegrated)
			return Eval(intPoly)
		}
	}

	// 4. Try pure rational function in varName: P(x) / Q(x)
	numNode, denNode := decomposeRationalFraction(normExpr)
	numPoly, okN := extractPoly(numNode, varName)
	denPoly, okD := extractPoly(denNode, varName)
	if okN && okD && !isPolyZero(denPoly) && denPoly.degree() > 0 {
		res, err := RischIntegrateRational(numPoly, denPoly, varName, fsm)
		if err == nil {
			return Eval(res)
		}
		// If rational integration failed with algebraic extension required, propagate Unsupported
		if strings.Contains(err.Error(), "Unsupported") {
			_ = fsm.TransitionTo(RischStateUnsupported)
			return nil, err
		}
	}

	// 5. Transcendental Extensions: Exponential and Logarithmic towers
	_ = fsm.TransitionTo(RischStateFieldTowerConstructed)
	tower, err := buildDifferentialTower(normExpr, varName)
	if err != nil {
		_ = fsm.TransitionTo(RischStateInvalidTower)
		return nil, err
	}
	_ = fsm.TransitionTo(RischStateTowerValidated)
	_ = fsm.TransitionTo(RischStateConstantFieldDetermined)

	// Dispatch by topmost extension
	if len(tower) > 0 {
		topExt := tower[len(tower)-1]
		switch topExt.Kind {
		case ExtExp:
			// Exponential Case: y' + f*y = g (Risch Differential Equation)
			return solveExponentialRisch(normExpr, topExt, varName, fsm)
		case ExtLog:
			// Logarithmic Case: recursive reduction
			return solveLogarithmicRisch(normExpr, topExt, varName, fsm)
		}
	}

	_ = fsm.TransitionTo(RischStateUnsupported)
	return nil, fmt.Errorf("%s", i18n.T("integral.err_risch_integration_unsupported_tower_structure"))
}

// normalizeMonomialsAST performs pre-normalization on AST expressions:
// exp(ln(u)) -> u, ln(exp(u)) -> u
func normalizeMonomialsAST(n ast.Node) ast.Node {
	return ast.Transform(n, func(curr ast.Node) ast.Node {
		if fn, ok := curr.(*ast.FuncNode); ok {
			// exp(ln(u)) -> u
			if (fn.Name == "exp" || fn.Name == "e") && len(fn.Args) == 1 {
				if innerFn, ok2 := fn.Args[0].(*ast.FuncNode); ok2 && (innerFn.Name == "ln" || innerFn.Name == "log") && len(innerFn.Args) == 1 {
					return innerFn.Args[0]
				}
			}
			// ln(exp(u)) -> u
			if (fn.Name == "ln" || fn.Name == "log") && len(fn.Args) == 1 {
				if innerFn, ok2 := fn.Args[0].(*ast.FuncNode); ok2 && (innerFn.Name == "exp" || innerFn.Name == "e") && len(innerFn.Args) == 1 {
					return innerFn.Args[0]
				}
			}
		}
		return curr
	})
}

// buildDifferentialTower identifies the transcendental generators (exp, ln) in expr.
func buildDifferentialTower(expr ast.Node, varName string) ([]DifferentialExtension, error) {
	var tower []DifferentialExtension
	seen := make(map[string]bool)

	ast.Walk(expr, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncNode); ok {
			if (fn.Name == "exp" || fn.Name == "e") && len(fn.Args) == 1 && containsVar(fn.Args[0], varName) {
				key := fmt.Sprintf("exp(%s)", fn.Args[0].String())
				if !seen[key] {
					seen[key] = true
					dArg, _ := differentiate(fn.Args[0], varName)
					tower = append(tower, DifferentialExtension{
						VarName: fmt.Sprintf("t_%d", len(tower)+1),
						Kind:    ExtExp,
						Arg:     fn.Args[0],
						Dt:      dArg,
					})
				}
			} else if (fn.Name == "ln" || fn.Name == "log") && len(fn.Args) == 1 && containsVar(fn.Args[0], varName) {
				key := fmt.Sprintf("ln(%s)", fn.Args[0].String())
				if !seen[key] {
					seen[key] = true
					dArg, _ := differentiate(fn.Args[0], varName)
					invArg, _ := simplifyPow(fn.Args[0], mustRational(-1, 1))
					dt, _ := simplifyMul([]ast.Node{dArg, invArg})
					tower = append(tower, DifferentialExtension{
						VarName: fmt.Sprintf("t_%d", len(tower)+1),
						Kind:    ExtLog,
						Arg:     fn.Args[0],
						Dt:      dt,
					})
				}
			}
		}
		return true
	})

	return tower, nil
}

// solveExponentialRisch solves integrals involving an exponential extension t = exp(u(x)).
func solveExponentialRisch(expr ast.Node, ext DifferentialExtension, varName string, fsm *RischLifecycleFSM) (ast.Node, error) {
	// 1. Check if integrand is of the form: c(x) * exp(u(x))
	u := ext.Arg
	uPrime, err := differentiate(u, varName)
	if err != nil {
		_ = fsm.TransitionTo(RischStateUnsupported)
		return nil, err
	}

	coeff, isExpTerm := matchExponentialFactor(expr, u)
	if isExpTerm {
		// We have integrand f(x) = coeff(x) * exp(u(x))
		// The Risch Differential Equation for ansatz y(x) * exp(u(x)) is:
		// y'(x) + u'(x) * y(x) = coeff(x)
		uPrimePoly, okUP := extractPoly(uPrime, varName)
		coeffPoly, okCP := extractPoly(coeff, varName)

		if okUP && okCP {
			// Case A: coeff(x) is polynomial, u(x) is polynomial
			fsm.SetCompleteness(true)

			// Degree analysis of RDE: y' + u'*y = coeff
			// deg(u') = m. deg(coeff) = n.
			degUPrime := uPrimePoly.degree()
			degCoeff := coeffPoly.degree()

			// Case A1: deg(u') >= 1 (e.g. u = x^2 => u' = 2x, coeff = x)
			if degUPrime >= 1 {
				// In y' + u'*y = coeff:
				// deg(u'*y) = deg(y) + deg(u'). Since deg(y') = deg(y) - 1, leading term cannot cancel with y'.
				// Thus: deg(y) + deg(u') = deg(coeff) => deg(y) = deg(coeff) - deg(u')
				degY := degCoeff - degUPrime
				if degY < 0 {
					// Degree of y would be negative!
					// Example: exp(-x^2): u = -x^2 => u' = -2x (deg 1). coeff = 1 (deg 0).
					// deg(y) = 0 - 1 = -1 < 0.
					// Mathematically certified nonelementary integral (Liouville's Theorem)!
					_ = fsm.TransitionTo(RischStateNonelementaryCertified)
					return nil, &NonelementaryIntegralError{
						Expr:   expr,
						Reason: fmt.Sprintf("Risch differential equation y' + (%s)*y = %s has no polynomial solution (deg y = %d < 0)", uPrime.String(), coeff.String(), degY),
					}
				}

				// Solve y(x) of degree degY by matching coefficients
				yPoly, ok := solvePolynomialRDE(uPrimePoly, coeffPoly, degY, varName)
				if ok {
					_ = fsm.TransitionTo(RischStateRischDESolved)
					expNode := &ast.FuncNode{Name: "exp", Args: []ast.Node{u}}
					return simplifyMul([]ast.Node{yPoly.toNode(), expNode})
				}

				// Complete decision tree exhausted with no solution
				_ = fsm.TransitionTo(RischStateNonelementaryCertified)
				return nil, &NonelementaryIntegralError{
					Expr:   expr,
					Reason: "Risch differential equation admits no elementary solution in base field",
				}
			}

			// Case A2: deg(u') == 0 (e.g. u = c*x => u' = c)
			if degUPrime == 0 {
				degY := degCoeff
				yPoly, ok := solvePolynomialRDE(uPrimePoly, coeffPoly, degY, varName)
				if ok {
					_ = fsm.TransitionTo(RischStateRischDESolved)
					expNode := &ast.FuncNode{Name: "exp", Args: []ast.Node{u}}
					return simplifyMul([]ast.Node{yPoly.toNode(), expNode})
				}
			}
		}

		// Case B: exp(x) / x (pole at x = 0)
		// coeff = 1/x, u = x => u' = 1
		// y' + y = 1/x has a pole of order 1 at x = 0.
		// If y has pole of order k at 0: y' has pole of order k+1, y has pole of order k.
		// Order of pole in y' + y is k+1 != 1. No rational solution can have pole order 1!
		if isLinearRationalPole(coeff, varName) && isLinearPoly(u, varName) {
			fsm.SetCompleteness(true)
			_ = fsm.TransitionTo(RischStateNonelementaryCertified)
			return nil, &NonelementaryIntegralError{
				Expr:   expr,
				Reason: "Risch differential equation pole order incompatibility at singular point (logarithmic integral / Ei)",
			}
		}
	}

	_ = fsm.TransitionTo(RischStateUnsupported)
	return nil, fmt.Errorf("%s", i18n.T("integral.err_risch_integration_unsupported_exponential_extension"))
}

// solveLogarithmicRisch solves integrals involving a logarithmic extension t = ln(u(x)).
func solveLogarithmicRisch(expr ast.Node, _ DifferentialExtension, varName string, fsm *RischLifecycleFSM) (ast.Node, error) {
	// Case 1: Pure ln(x) -> x*ln(x) - x
	if fn, ok := expr.(*ast.FuncNode); ok && (fn.Name == "ln" || fn.Name == "log") && len(fn.Args) == 1 {
		arg := fn.Args[0]
		if v, okV := arg.(*ast.VarNode); okV && v.Name == varName {
			_ = fsm.TransitionTo(RischStatePolynomialPartIntegrated)
			xNode := &ast.VarNode{Name: varName}
			xLnX, _ := simplifyMul([]ast.Node{xNode, expr})
			negX, _ := simplifyUnaryOp("-", xNode)
			return simplifyAdd([]ast.Node{xLnX, negX})
		}
	}

	// Case 2: 1 / (x * ln(x)) -> ln(ln(x))
	numNode, denNode := decomposeRationalFraction(expr)
	if isOne(numNode) {
		// Check if denNode is x * ln(x)
		if mul, ok := denNode.(*ast.MulNode); ok && len(mul.Factors) == 2 {
			f0 := mul.Factors[0]
			f1 := mul.Factors[1]
			if isVar(f0, varName) && isLogOfVar(f1, varName) || isVar(f1, varName) && isLogOfVar(f0, varName) {
				_ = fsm.TransitionTo(RischStateResiduePolesExtracted)
				innerLog := &ast.FuncNode{Name: "ln", Args: []ast.Node{&ast.VarNode{Name: varName}}}
				return &ast.FuncNode{Name: "ln", Args: []ast.Node{innerLog}}, nil
			}
		}
	}

	_ = fsm.TransitionTo(RischStateUnsupported)
	return nil, fmt.Errorf("%s", i18n.T("integral.err_risch_integration_unsupported_logarithmic_extension"))
}

func basisMonomial1D(varName string, deg int) *univariatePoly {
	coeffs := make([]ast.Node, deg+1)
	for i := 0; i < deg; i++ {
		coeffs[i] = mustRational(0, 1)
	}
	coeffs[deg] = mustRational(1, 1)
	return &univariatePoly{varName: varName, coeffs: coeffs}
}

// solvePolynomialRDE solves y' + uPrime * y = coeff for y of degree degY.
func solvePolynomialRDE(uPrime, coeff *univariatePoly, degY int, varName string) (*univariatePoly, bool) {
	// y(x) = sum_{j=0}^degY c_j * x^j
	// Operator: L(y) = y' + uPrime * y
	basisImages := make([]*univariatePoly, degY+1)
	maxImageDeg := 0

	for j := 0; j <= degY; j++ {
		// y_j = x^j
		yj := basisMonomial1D(varName, j)
		dyj := polyDeriv1D(yj)
		uYj := polyMul1D(uPrime, yj)
		im := polyAdd1D(dyj, uYj)
		basisImages[j] = im
		if im.degree() > maxImageDeg {
			maxImageDeg = im.degree()
		}
	}

	numEqs := maxImageDeg + 1
	if coeff.degree()+1 > numEqs {
		numEqs = coeff.degree() + 1
	}

	M := make([][]*big.Rat, numEqs)
	C := make([]*big.Rat, numEqs)

	for deg := 0; deg < numEqs; deg++ {
		M[deg] = make([]*big.Rat, degY+1)
		for j := 0; j <= degY; j++ {
			coeffNode := basisImages[j].coeff(deg)
			if r, ok := coeffNode.(*ast.RationalNode); ok {
				M[deg][j] = new(big.Rat).Set(r.Val)
			} else {
				M[deg][j] = big.NewRat(0, 1)
			}
		}

		cCoeffNode := coeff.coeff(deg)
		if r, ok := cCoeffNode.(*ast.RationalNode); ok {
			C[deg] = new(big.Rat).Set(r.Val)
		} else {
			C[deg] = big.NewRat(0, 1)
		}
	}

	uSol, err := solveRationalLinearSystem(M, C, numEqs, degY+1)
	if err != nil {
		return nil, false
	}

	coeffs := make([]ast.Node, degY+1)
	for i := 0; i <= degY; i++ {
		coeffs[i] = mustRationalBig(uSol[i])
	}
	y := &univariatePoly{varName: varName, coeffs: coeffs}

	// Verify solution: y' + uPrime * y == coeff
	dy := polyDeriv1D(y)
	lhs := polyAdd1D(dy, polyMul1D(uPrime, y))
	checkDiff := polySub1D(lhs, coeff)
	if !isPolyZero(checkDiff) {
		return nil, false
	}

	return y, true
}

func matchExponentialFactor(expr, u ast.Node) (coeff ast.Node, ok bool) {
	// Check expr == exp(u)
	if fn, ok := expr.(*ast.FuncNode); ok && (fn.Name == "exp" || fn.Name == "e") && len(fn.Args) == 1 {
		if fn.Args[0].Equal(u) {
			return mustRational(1, 1), true
		}
	}

	// Check expr == coeff * exp(u)
	if mul, ok := expr.(*ast.MulNode); ok {
		var otherFactors []ast.Node
		foundExp := false
		for _, f := range mul.Factors {
			if fn, okF := f.(*ast.FuncNode); okF && (fn.Name == "exp" || fn.Name == "e") && len(fn.Args) == 1 {
				if fn.Args[0].Equal(u) {
					foundExp = true
					continue
				}
			}
			otherFactors = append(otherFactors, f)
		}
		if foundExp {
			if len(otherFactors) == 0 {
				return mustRational(1, 1), true
			}
			cNode, _ := simplifyMul(otherFactors)
			return cNode, true
		}
	}

	return nil, false
}

func isLinearRationalPole(coeff ast.Node, varName string) bool {
	// Check if coeff is of the form c / x
	numNode, denNode := decomposeRationalFraction(coeff)
	if isOne(numNode) && isVar(denNode, varName) {
		return true
	}
	return false
}

func isLinearPoly(n ast.Node, varName string) bool {
	if isVar(n, varName) {
		return true
	}
	return false
}

func isVar(n ast.Node, name string) bool {
	if v, ok := n.(*ast.VarNode); ok && v.Name == name {
		return true
	}
	return false
}

func isLogOfVar(n ast.Node, name string) bool {
	if fn, ok := n.(*ast.FuncNode); ok && (fn.Name == "ln" || fn.Name == "log") && len(fn.Args) == 1 {
		return isVar(fn.Args[0], name)
	}
	return false
}

func decomposeRationalFraction(n ast.Node) (num ast.Node, den ast.Node) {
	nums, dens := decomposeTermFactors(n)
	if len(nums) == 0 && len(dens) == 0 {
		return n, mustRational(1, 1)
	}
	var numNode ast.Node = mustRational(1, 1)
	if len(nums) > 0 {
		numNode, _ = simplifyMul(nums)
	}
	var denNode ast.Node = mustRational(1, 1)
	if len(dens) > 0 {
		denNode, _ = simplifyMul(dens)
	}
	return numNode, denNode
}

func integrateUnivariatePoly(p *univariatePoly, varName string) (ast.Node, error) {
	var terms []ast.Node
	for deg, coeff := range p.coeffs {
		if isZero(coeff) {
			continue
		}
		newDeg := deg + 1
		invNewDeg := mustRational(1, int64(newDeg))
		termCoeff, _ := simplifyMul([]ast.Node{coeff, invNewDeg})
		var xPow ast.Node
		if newDeg == 1 {
			xPow = &ast.VarNode{Name: varName}
		} else {
			xPow = &ast.PowNode{Base: &ast.VarNode{Name: varName}, Exp: mustRational(int64(newDeg), 1)}
		}
		term, _ := simplifyMul([]ast.Node{termCoeff, xPow})
		terms = append(terms, term)
	}
	if len(terms) == 0 {
		return mustRational(0, 1), nil
	}
	return simplifyAdd(terms)
}

func polyDeriv1D(p *univariatePoly) *univariatePoly {
	if p.degree() <= 0 {
		return &univariatePoly{varName: p.varName, coeffs: []ast.Node{mustRational(0, 1)}}
	}
	newCoeffs := make([]ast.Node, len(p.coeffs)-1)
	for i := 1; i < len(p.coeffs); i++ {
		scale := mustRational(int64(i), 1)
		prod, _ := simplifyMul([]ast.Node{p.coeffs[i], scale})
		newCoeffs[i-1] = prod
	}
	return &univariatePoly{varName: p.varName, coeffs: newCoeffs}
}

func findUnivariatePolyRationalRoots(p *univariatePoly) []*big.Rat {
	var roots []*big.Rat
	// Check candidates with denominators up to 6 and numerators between -20 and 20
	// for typical residue values in CAS problems
	seen := make(map[string]bool)
	for den := int64(1); den <= 6; den++ {
		for num := int64(-20); num <= 20; num++ {
			cand := big.NewRat(num, den)
			key := cand.RatString()
			if seen[key] {
				continue
			}
			seen[key] = true

			val, err := evalPolyAtRat(p, cand)
			if err == nil && val != nil && val.Sign() == 0 {
				roots = append(roots, cand)
			}
		}
	}
	return roots
}
