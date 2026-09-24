package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// VerificationDomain defines the mathematical domain of the verified operation.
type VerificationDomain string

const (
	DomainIntegral      VerificationDomain = "integral"
	DomainODE           VerificationDomain = "ode"
	DomainFactor        VerificationDomain = "factor"
	DomainSolve         VerificationDomain = "solve"
	DomainRationalApart VerificationDomain = "rational"
	DomainMatrix        VerificationDomain = "matrix"
	DomainWZ            VerificationDomain = "wz"
	DomainGeneral       VerificationDomain = "general"
)

// VerificationCertificate represents a certified mathematical proof of correctness.
type VerificationCertificate struct {
	Domain     VerificationDomain
	Equation   string
	Residual   Node
	IsVerified bool
	Details    string
	State      VerifyState
	CertIR     Certificate
}

// ToCertificateIR returns the underlying structured certificate IR.
// If not already set, it dynamically constructs a BaseCertificate wrapper.
func (vc *VerificationCertificate) ToCertificateIR() Certificate {
	if vc == nil {
		return nil
	}
	if vc.CertIR != nil {
		return vc.CertIR
	}
	return &BaseCertificate{
		DomainVal:    CertificateDomain(vc.Domain),
		Verified:     vc.IsVerified,
		ResidualNode: vc.Residual,
		DetailsMsg:   vc.Details,
		EquationStr:  vc.Equation,
	}
}

// String returns the formatted certificate line for CLI display.
func (vc *VerificationCertificate) String() string {
	if vc == nil {
		return ""
	}
	if vc.IsVerified {
		return fmt.Sprintf("[VERIFIED: %s]", vc.Equation)
	}
	return fmt.Sprintf("[FAILED VERIFICATION: %s] %s", vc.Equation, vc.Details)
}

// VerifyComputation inspects an input expression and its evaluated result,
// classifies its domain, and executes an independent reverse-verification checker.
func VerifyComputation(expr, result Node, env *Env) (*VerificationCertificate, error) {
	if expr == nil || result == nil {
		return nil, fmt.Errorf("%s", i18n.T("verify.err_cannot_verify_nil_expression_or"))
	}

	fsm := NewVerifyLifecycleFSM()

	// 1. Classify operation domain by inspecting input AST
	switch fn := expr.(type) {
	case *FuncNode:
		switch fn.Name {
		case "integrate", "risch_integrate":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyIntegral(fn, result, env, fsm)

		case "factor":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyFactor(fn, result, env, fsm)

		case "solve":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifySolve(fn, result, env, fsm)

		case "apart":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyApart(fn, result, env, fsm)

		case "inv":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyMatrixInv(fn, result, env, fsm)

		case "lu", "qr", "cholesky", "ldlt", "pinv":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyMatrixDecomp(fn, result, env, fsm)

		case "dsolve":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyODE(fn, result, env, fsm)

		case "wz_cert":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyWZ(fn, result, env, fsm)
		}

	case *RelOpNode:
		if fn.Op == "==" {
			// Equation verification: verify both sides are equivalent
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return VerifyAlgebraicEquivalence(fn.LHS, fn.RHS, env)
		}
	}

	// Default: General Algebraic Equivalence / Self-Consistency
	_ = fsm.TransitionTo(VerifyStateTargetClassified)
	return verifyGeneral(expr, result, env, fsm)
}

// -------------------------------------------------------------------------
// Specialized Reverse-Checkers (Layer 1: Asymmetric Verifiers)
// -------------------------------------------------------------------------

// verifyIntegral verifies: diff(F, x) - f == 0
func verifyIntegral(fn *FuncNode, F Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_integrate_requires_at_least_1"))
	}
	f := fn.Args[0]
	varName := "x"
	if len(fn.Args) >= 2 {
		if v, ok := fn.Args[1].(*VarNode); ok {
			varName = v.Name
		}
	}

	// Compute derivative F'(x) = d/dx (F)
	fPrime, err := differentiate(F, varName)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_verify_failed_to_differentiate_result", err))
	}

	// Residual = F'(x) - f(x)
	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	negF, err := simplifyUnaryOp("-", f)
	if err != nil {
		return nil, err
	}
	residualNode := NewAdd([]Node{fPrime, negF})

	// Evaluate residual to 0
	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	isZeroRes := checkIsZeroAlgebraically(residualNode, env)

	eqStr := fmt.Sprintf("diff(F, %s) - f == 0", varName)
	if isZeroRes {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := fmt.Sprintf(i18n.T("verify.integral_holds"), varName)
		certIR := NewDerivCertificate(f, F, mustRational(0, 1), varName, true, details)
		return &VerificationCertificate{
			Domain:     DomainIntegral,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	evalRes, _ := Eval(residualNode)
	details := fmt.Sprintf(i18n.T("verify.refuted_residual"), evalRes.String())
	certIR := NewDerivCertificate(f, F, evalRes, varName, false, details)
	return &VerificationCertificate{
		Domain:     DomainIntegral,
		Equation:   eqStr,
		Residual:   evalRes,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// verifyFactor verifies: expand(result) - expand(original) == 0
func verifyFactor(fn *FuncNode, factorResult Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_factor_requires_at_least_1"))
	}
	orig := fn.Args[0]

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	expFactors := expandNode(factorResult)
	expOrig := expandNode(orig)

	negOrig, _ := simplifyUnaryOp("-", expOrig)
	residual := NewAdd([]Node{expFactors, negOrig})

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	isZeroRes := checkIsZeroAlgebraically(residual, env)

	eqStr := "expand(factors) - P == 0"
	if isZeroRes {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.factor_holds")
		certIR := NewIdentityCertificate(IdentityKindFactor, orig, factorResult, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainFactor,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	evalRes, _ := Eval(residual)
	details := fmt.Sprintf(i18n.T("verify.refuted_residual"), evalRes.String())
	certIR := NewIdentityCertificate(IdentityKindFactor, orig, factorResult, evalRes, false, details)
	return &VerificationCertificate{
		Domain:     DomainFactor,
		Equation:   eqStr,
		Residual:   evalRes,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// verifySolve verifies: substituting each root into equation yields 0 == 0
func verifySolve(fn *FuncNode, rootsResult Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_solve_requires_at_least_1"))
	}
	eq := fn.Args[0]
	varName := "x"
	if len(fn.Args) >= 2 {
		if v, ok := fn.Args[1].(*VarNode); ok {
			varName = v.Name
		}
	}

	// Form lhs - rhs = 0
	var zeroExpr Node
	if bin, ok := eq.(*RelOpNode); ok && bin.Op == "==" {
		negR, _ := simplifyUnaryOp("-", bin.RHS)
		zeroExpr = NewAdd([]Node{bin.LHS, negR})
	} else {
		zeroExpr = eq
	}

	// Extract roots list
	var roots []Node
	if list, ok := rootsResult.(*ListNode); ok {
		roots = list.Elements
	} else {
		roots = []Node{rootsResult}
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	allRootsSatisfied := true
	var failedRoot Node
	var lastResidual Node

	for _, r := range roots {
		subbed := Substitute(zeroExpr, varName, r)
		if !checkIsZeroAlgebraically(subbed, env) {
			allRootsSatisfied = false
			failedRoot = r
			lastResidual, _ = Eval(subbed)
			break
		}
	}

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	eqStr := fmt.Sprintf("eq[%s -> roots] == 0", varName)
	if allRootsSatisfied {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.solve_holds")
		certIR := NewIdentityCertificate(IdentityKindEquivalence, zeroExpr, rootsResult, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainSolve,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	details := fmt.Sprintf("root %s did not satisfy equation (residual = %s)", failedRoot.String(), lastResidual.String())
	certIR := NewIdentityCertificate(IdentityKindEquivalence, zeroExpr, rootsResult, lastResidual, false, details)
	return &VerificationCertificate{
		Domain:     DomainSolve,
		Equation:   eqStr,
		Residual:   lastResidual,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// verifyApart verifies: together(apartResult) - original == 0
func verifyApart(fn *FuncNode, apartResult Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_apart_requires_at_least_1"))
	}
	orig := fn.Args[0]

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	togRes, err := EvalTogether(apartResult)
	if err != nil {
		togRes = apartResult
	}

	negOrig, _ := simplifyUnaryOp("-", orig)
	diff := NewAdd([]Node{togRes, negOrig})

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	isZeroRes := checkIsZeroAlgebraically(diff, env)

	eqStr := "together(apart) - f == 0"
	if isZeroRes {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.rational_apart_holds")
		certIR := NewIdentityCertificate(IdentityKindApart, orig, apartResult, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainRationalApart,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	evalRes, _ := Eval(diff)
	details := fmt.Sprintf(i18n.T("verify.refuted_residual"), evalRes.String())
	certIR := NewIdentityCertificate(IdentityKindApart, orig, apartResult, evalRes, false, details)
	return &VerificationCertificate{
		Domain:     DomainRationalApart,
		Equation:   eqStr,
		Residual:   evalRes,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// verifyMatrixInv verifies: A * A^-1 - I == O
func verifyMatrixInv(fn *FuncNode, invResult Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_inv_requires_at_least_1"))
	}
	A, okA := fn.Args[0].(*MatrixNode)
	AInv, okInv := invResult.(*MatrixNode)
	if !okA || !okInv {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_matrix_inversion_verification_requires_matrixnode"))
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	prod, err := evalMatrixMulVerified(A, AInv)
	if err != nil {
		return nil, err
	}

	// Check if prod == Identity matrix
	isIdent := true
	for r := 0; r < prod.Rows; r++ {
		for c := 0; c < prod.Cols; c++ {
			elem := prod.Data[r][c]
			if r == c {
				negOne := mustRational(-1, 1)
				diff := NewAdd([]Node{elem, negOne})
				if !checkIsZeroAlgebraically(diff, env) {
					isIdent = false
					break
				}
			} else {
				if !checkIsZeroAlgebraically(elem, env) {
					isIdent = false
					break
				}
			}
		}
		if !isIdent {
			break
		}
	}

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	eqStr := "A * inv(A) - I == O"
	if isIdent {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.matrix_inv_holds")
		certIR := NewInvertibilityCertificate(A, AInv, nil, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainMatrix,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	details := i18n.T("verify.refuted_residual")
	certIR := NewInvertibilityCertificate(A, AInv, nil, prod, false, details)
	return &VerificationCertificate{
		Domain:     DomainMatrix,
		Equation:   eqStr,
		Residual:   prod,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// verifyMatrixDecomp verifies: LU (PA == LU), QR (Q^T Q == I && QR == A), Cholesky (L L^T == A), etc.
func verifyMatrixDecomp(fn *FuncNode, decompResult Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_matrix_decomposition_requires_at_least"))
	}
	A, okA := fn.Args[0].(*MatrixNode)
	if !okA {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_argument_must_be_a_matrix"))
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	switch fn.Name {
	case "cholesky":
		// result is L -> check A - L * L^T == O
		L, okL := decompResult.(*MatrixNode)
		if !okL {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			return nil, fmt.Errorf("%s", i18n.T("verify.err_cholesky_output_must_be_a"))
		}
		LT := evalTranspose(L)
		LLT, err := evalMatrixMulVerified(L, LT)
		if err != nil {
			return nil, err
		}
		isEqual := checkMatrixEquality(A, LLT, env)
		_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
		eqStr := "A - L * L^T == O"
		if isEqual {
			_ = fsm.TransitionTo(VerifyStateCertified)
			details := i18n.T("verify.matrix_cholesky_holds")
			certIR := NewDecompositionCertificate(DecompCholesky, A, []Node{L, LT}, mustRational(0, 1), true, details)
			return &VerificationCertificate{
				Domain:     DomainMatrix,
				Equation:   eqStr,
				Residual:   mustRational(0, 1),
				IsVerified: true,
				Details:    details,
				State:      VerifyStateCertified,
				CertIR:     certIR,
			}, nil
		}
		_ = fsm.TransitionTo(VerifyStateRefuted)
		details := i18n.T("verify.refuted_residual")
		certIR := NewDecompositionCertificate(DecompCholesky, A, []Node{L, LT}, LLT, false, details)
		return &VerificationCertificate{
			Domain:     DomainMatrix,
			Equation:   eqStr,
			Residual:   LLT,
			IsVerified: false,
			Details:    details,
			State:      VerifyStateRefuted,
			CertIR:     certIR,
		}, nil

	case "lu":
		// result is [P, L, U] -> check P * A == L * U
		list, okList := decompResult.(*ListNode)
		if !okList || len(list.Elements) != 3 {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			return nil, fmt.Errorf("%s", i18n.T("verify.err_lu_output_must_be_a"))
		}
		P, okP := list.Elements[0].(*MatrixNode)
		L, okL := list.Elements[1].(*MatrixNode)
		U, okU := list.Elements[2].(*MatrixNode)
		if !okP || !okL || !okU {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			return nil, fmt.Errorf("%s", i18n.T("verify.err_lu_elements_must_be_matrixnode"))
		}
		PA, err1 := evalMatrixMulVerified(P, A)
		LU, err2 := evalMatrixMulVerified(L, U)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("%s", i18n.T("verify.err_matrix_mul_error_in_lu"))
		}
		isEqual := checkMatrixEquality(PA, LU, env)
		_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
		eqStr := "P * A - L * U == O"
		if isEqual {
			_ = fsm.TransitionTo(VerifyStateCertified)
			details := i18n.T("verify.matrix_lu_holds")
			certIR := NewDecompositionCertificate(DecompLU, A, []Node{P, L, U}, mustRational(0, 1), true, details)
			return &VerificationCertificate{
				Domain:     DomainMatrix,
				Equation:   eqStr,
				Residual:   mustRational(0, 1),
				IsVerified: true,
				Details:    details,
				State:      VerifyStateCertified,
				CertIR:     certIR,
			}, nil
		}
		_ = fsm.TransitionTo(VerifyStateRefuted)
		details := i18n.T("verify.refuted_residual")
		certIR := NewDecompositionCertificate(DecompLU, A, []Node{P, L, U}, PA, false, details)
		return &VerificationCertificate{
			Domain:     DomainMatrix,
			Equation:   eqStr,
			Residual:   PA,
			IsVerified: false,
			Details:    details,
			State:      VerifyStateRefuted,
			CertIR:     certIR,
		}, nil

	case "qr":
		// result is [Q, R] -> check Q^T * Q == I && Q * R == A
		list, okList := decompResult.(*ListNode)
		if !okList || len(list.Elements) != 2 {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			return nil, fmt.Errorf("%s", i18n.T("verify.err_qr_output_must_be_a"))
		}
		Q, okQ := list.Elements[0].(*MatrixNode)
		R, okR := list.Elements[1].(*MatrixNode)
		if !okQ || !okR {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			return nil, fmt.Errorf("%s", i18n.T("verify.err_qr_elements_must_be_matrixnode"))
		}
		QT := evalTranspose(Q)
		QTQ, err1 := evalMatrixMulVerified(QT, Q)
		QR, err2 := evalMatrixMulVerified(Q, R)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("%s", i18n.T("verify.err_matrix_mul_error_in_qr"))
		}
		// QTQ must be Identity
		isOrthogonal := true
		for r := 0; r < QTQ.Rows; r++ {
			for c := 0; c < QTQ.Cols; c++ {
				elem := QTQ.Data[r][c]
				if r == c {
					diff := NewAdd([]Node{elem, mustRational(-1, 1)})
					if !checkIsZeroAlgebraically(diff, env) {
						isOrthogonal = false
						break
					}
				} else {
					if !checkIsZeroAlgebraically(elem, env) {
						isOrthogonal = false
						break
					}
				}
			}
		}
		isFactored := checkMatrixEquality(QR, A, env)
		_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
		eqStr := "Q^T * Q == I and Q * R == A"
		if isOrthogonal && isFactored {
			_ = fsm.TransitionTo(VerifyStateCertified)
			details := i18n.T("verify.matrix_qr_holds")
			certIR := NewDecompositionCertificate(DecompQR, A, []Node{Q, R}, mustRational(0, 1), true, details)
			return &VerificationCertificate{
				Domain:     DomainMatrix,
				Equation:   eqStr,
				Residual:   mustRational(0, 1),
				IsVerified: true,
				Details:    details,
				State:      VerifyStateCertified,
				CertIR:     certIR,
			}, nil
		}
		_ = fsm.TransitionTo(VerifyStateRefuted)
		details := i18n.T("verify.refuted_residual")
		certIR := NewDecompositionCertificate(DecompQR, A, []Node{Q, R}, QR, false, details)
		return &VerificationCertificate{
			Domain:     DomainMatrix,
			Equation:   eqStr,
			Residual:   QR,
			IsVerified: false,
			Details:    details,
			State:      VerifyStateRefuted,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
	return nil, fmt.Errorf("%s", i18n.T("verify.err_unsupported_matrix_decomposition_verification", fn.Name))
}

// verifyODE verifies: substituting y(x) into differential equation yields 0
func verifyODE(fn *FuncNode, ySol Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_dsolve_requires_at_least_1"))
	}
	eq := fn.Args[0]
	xName := "x"
	if len(fn.Args) >= 3 {
		if vx, ok := fn.Args[2].(*VarNode); ok {
			xName = vx.Name
		}
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	// Extract LHS and RHS of ODE
	var diffEq Node
	if bin, ok := eq.(*RelOpNode); ok && bin.Op == "==" {
		negR, _ := simplifyUnaryOp("-", bin.RHS)
		diffEq = NewAdd([]Node{bin.LHS, negR})
	} else {
		diffEq = eq
	}

	// Substitute y, y', y'' into diffEq
	// In i-hate-decimal-calc ODE, y is typically represented as a variable or function
	// Compute y' = diff(ySol, x) and y'' = diff(y', x)
	dySol, err := differentiate(ySol, xName)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, err
	}
	d2ySol, err := differentiate(dySol, xName)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, err
	}

	// Replace derivatives and y in diffEq
	// We do AST replacement for y'', y', y
	substituted := substituteODE(diffEq, ySol, dySol, d2ySol)

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	isZeroRes := checkIsZeroAlgebraically(substituted, env)

	eqStr := "ODE_Residual[y -> y_sol] == 0"
	if isZeroRes {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.ode_holds")
		certIR := NewODECertificate(diffEq, ySol, mustRational(0, 1), xName, "y", true, details)
		return &VerificationCertificate{
			Domain:     DomainODE,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	evalRes, _ := Eval(substituted)
	details := fmt.Sprintf(i18n.T("verify.refuted_residual"), evalRes.String())
	certIR := NewODECertificate(diffEq, ySol, evalRes, xName, "y", false, details)
	return &VerificationCertificate{
		Domain:     DomainODE,
		Equation:   eqStr,
		Residual:   evalRes,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// verifyWZ verifies Wilf-Zeilberger certificate: F(n+1, k) - F(n, k) == G(n, k+1) - G(n, k)
func verifyWZ(fn *FuncNode, certResult Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 3 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("verify.err_wz_cert_requires_3_arguments"))
	}
	F := fn.Args[0]
	nVar := "n"
	if vn, ok := fn.Args[1].(*VarNode); ok {
		nVar = vn.Name
	}
	kVar := "k"
	if vk, ok := fn.Args[2].(*VarNode); ok {
		kVar = vk.Name
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	// G(n, k) = certResult * F(n, k)
	G, err := simplifyMul([]Node{certResult, F})
	if err != nil {
		return nil, err
	}

	// Delta_n F = F(n+1, k) - F(n, k)
	fNPlus1 := Substitute(F, nVar, NewAdd([]Node{&VarNode{Name: nVar}, mustRational(1, 1)}))
	negF, _ := simplifyUnaryOp("-", F)
	deltaNF := NewAdd([]Node{fNPlus1, negF})

	// Delta_k G = G(n, k+1) - G(n, k)
	gKPlus1 := Substitute(G, kVar, NewAdd([]Node{&VarNode{Name: kVar}, mustRational(1, 1)}))
	negG, _ := simplifyUnaryOp("-", G)
	deltaKG := NewAdd([]Node{gKPlus1, negG})

	// Residual = Delta_n F - Delta_k G
	negDeltaKG, _ := simplifyUnaryOp("-", deltaKG)
	residual := NewAdd([]Node{deltaNF, negDeltaKG})

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	isZeroRes := checkIsZeroAlgebraically(residual, env)

	eqStr := "Delta_n(F) == Delta_k(R * F)"
	if isZeroRes {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.wz_holds")
		certIR := NewWZCertificate(F, certResult, mustRational(0, 1), nVar, kVar, true, details)
		return &VerificationCertificate{
			Domain:     DomainWZ,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	evalRes, _ := Eval(residual)
	details := fmt.Sprintf(i18n.T("verify.refuted_residual"), evalRes.String())
	certIR := NewWZCertificate(F, certResult, evalRes, nVar, kVar, false, details)
	return &VerificationCertificate{
		Domain:     DomainWZ,
		Equation:   eqStr,
		Residual:   evalRes,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// -------------------------------------------------------------------------
// General Algebraic Equivalence Verifier (Layer 3)
// -------------------------------------------------------------------------

// VerifyAlgebraicEquivalence checks if lhs and rhs are mathematically identical: expand(lhs - rhs) == 0.
func VerifyAlgebraicEquivalence(lhs, rhs Node, env *Env) (*VerificationCertificate, error) {
	fsm := NewVerifyLifecycleFSM()
	_ = fsm.TransitionTo(VerifyStateTargetClassified)

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	negR, _ := simplifyUnaryOp("-", rhs)
	diff := NewAdd([]Node{lhs, negR})

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	isZeroRes := checkIsZeroAlgebraically(diff, env)

	eqStr := fmt.Sprintf("%s == %s", lhs.String(), rhs.String())
	if isZeroRes {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.general_holds")
		certIR := NewIdentityCertificate(IdentityKindEquivalence, lhs, rhs, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainGeneral,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	evalDiff, _ := Eval(diff)
	details := fmt.Sprintf(i18n.T("verify.refuted_residual"), evalDiff.String())
	certIR := NewIdentityCertificate(IdentityKindEquivalence, lhs, rhs, evalDiff, false, details)
	return &VerificationCertificate{
		Domain:     DomainGeneral,
		Equation:   eqStr,
		Residual:   evalDiff,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// verifyGeneral verifies if expr simplifies to result: expr - result == 0.
func verifyGeneral(expr, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	negRes, _ := simplifyUnaryOp("-", result)
	diff := NewAdd([]Node{expr, negRes})

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	isZeroRes := checkIsZeroAlgebraically(diff, env)

	eqStr := fmt.Sprintf("expr == %s", result.String())
	if isZeroRes {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.general_holds")
		certIR := NewIdentityCertificate(IdentityKindGeneral, expr, result, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainGeneral,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	evalDiff, _ := Eval(diff)
	details := fmt.Sprintf(i18n.T("verify.refuted_residual"), evalDiff.String())
	certIR := NewIdentityCertificate(IdentityKindGeneral, expr, result, evalDiff, false, details)
	return &VerificationCertificate{
		Domain:     DomainGeneral,
		Equation:   eqStr,
		Residual:   evalDiff,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// -------------------------------------------------------------------------
// Algebraic Zero Check & Matrix Helpers
// -------------------------------------------------------------------------

// checkIsZeroAlgebraically thoroughly verifies if node simplifies to 0 using multiple CAS techniques.
func checkIsZeroAlgebraically(node Node, env *Env) bool {
	if node == nil {
		return true
	}
	if isZero(node) {
		return true
	}

	// 1. Direct Eval
	if ev, err := Eval(node); err == nil && isZero(ev) {
		return true
	}

	// 2. Expand terms
	expanded := expandNode(node)
	if isZero(expanded) {
		return true
	}
	if ev, err := Eval(expanded); err == nil && isZero(ev) {
		return true
	}

	// 3. Rational recombination (Together)
	if tog, err := EvalTogether(node); err == nil {
		if isZero(tog) {
			return true
		}
		if ev, err := Eval(tog); err == nil && isZero(ev) {
			return true
		}
	}

	// 4. Trig reduce
	if trig, err := EvalTrigReduce(node, env); err == nil {
		if isZero(trig) {
			return true
		}
		if ev, err := Eval(trig); err == nil && isZero(ev) {
			return true
		}
	}

	return false
}

// checkMatrixEquality verifies if two matrices A and B have identical elements modulo algebra.
func checkMatrixEquality(A, B *MatrixNode, env *Env) bool {
	if A.Rows != B.Rows || A.Cols != B.Cols {
		return false
	}
	for r := 0; r < A.Rows; r++ {
		for c := 0; c < A.Cols; c++ {
			elemA := A.Data[r][c]
			elemB := B.Data[r][c]
			negB, _ := simplifyUnaryOp("-", elemB)
			diff := NewAdd([]Node{elemA, negB})
			if !checkIsZeroAlgebraically(diff, env) {
				return false
			}
		}
	}
	return true
}

// substituteODE replaces y'', y', and y in an expression AST with their respective candidate solutions.
func substituteODE(node Node, y, dy, d2y Node) Node {
	if node == nil {
		return nil
	}

	// Match derivative nodes or function calls like diff(y, x) or diff(y, x, 2)
	if fn, ok := node.(*FuncNode); ok {
		if fn.Name == "diff" && len(fn.Args) >= 2 {
			order := int64(1)
			if len(fn.Args) >= 3 {
				if r, ok := fn.Args[2].(*RationalNode); ok && r.Val.IsInt() {
					order = r.Val.Num().Int64()
				}
			}
			if order == 2 {
				return d2y
			}
			if order == 1 {
				return dy
			}
		}
	}

	// Match variable y
	if v, ok := node.(*VarNode); ok && v.Name == "y" {
		return y
	}

	// Recurse into children
	switch v := node.(type) {
	case *UnaryOpNode:
		return &UnaryOpNode{Op: v.Op, Expr: substituteODE(v.Expr, y, dy, d2y)}
	case *RelOpNode:
		return &RelOpNode{Op: v.Op, LHS: substituteODE(v.LHS, y, dy, d2y), RHS: substituteODE(v.RHS, y, dy, d2y)}
	case *AddNode:
		terms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			terms[i] = substituteODE(t, y, dy, d2y)
		}
		return &AddNode{Terms: terms}
	case *MulNode:
		factors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			factors[i] = substituteODE(f, y, dy, d2y)
		}
		return &MulNode{Factors: factors}
	case *PowNode:
		return &PowNode{Base: substituteODE(v.Base, y, dy, d2y), Exp: substituteODE(v.Exp, y, dy, d2y)}
	case *FuncNode:
		args := make([]Node, len(v.Args))
		for i, a := range v.Args {
			args[i] = substituteODE(a, y, dy, d2y)
		}
		return &FuncNode{Name: v.Name, Args: args}
	}

	return node
}

func evalMatrixMulVerified(a, b Node) (*MatrixNode, error) {
	res, err := mulMatrixOrScalar(a, b)
	if err != nil {
		return nil, err
	}
	mat, ok := res.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("verify.err_result_is_not_a_matrix"))
	}
	return mat, nil
}
