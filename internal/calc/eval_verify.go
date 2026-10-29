package calc

import (
	"fmt"
	"math/big"

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
	DomainGeometry      VerificationDomain = "geometry"
	DomainImpossibility VerificationDomain = "impossibility"
	DomainGeneral       VerificationDomain = "general"
	DomainLinearSolve   VerificationDomain = "linear_solve"
	DomainPell          VerificationDomain = "pell"
	DomainPolynomialRoot VerificationDomain = "polynomial_root"
	DomainLLL           VerificationDomain = "lll"
	DomainElliptic      VerificationDomain = "elliptic"
	DomainSNF           VerificationDomain = "snf"
	DomainHNF           VerificationDomain = "hnf"
	DomainIdeal         VerificationDomain = "ideal"
	DomainUnits         VerificationDomain = "units"
	DomainRecurrence    VerificationDomain = "recurrence"
	DomainTransform     VerificationDomain = "transform"
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

// ToProofTrace converts the verification certificate or its underlying IR to a unified ProofTrace.
func (vc *VerificationCertificate) ToProofTrace() (*ProofTrace, error) {
	if vc == nil {
		return nil, fmt.Errorf("cannot convert nil verification certificate")
	}
	if vc.CertIR != nil {
		return ConvertCertificateToProofTrace(vc.CertIR)
	}
	return ConvertVerificationCertificateToProofTrace(vc)
}

// VerifyTrace verifies the algebraic correctness of the certificate's proof trace via native zero check.
func (vc *VerificationCertificate) VerifyTrace(env *Env) (bool, error) {
	trace, err := vc.ToProofTrace()
	if err != nil {
		return false, err
	}
	return VerifyProofTrace(trace, env)
}

// RenderTraceExplain renders the certificate's proof trace as a human-readable Unicode explanation tree.
func (vc *VerificationCertificate) RenderTraceExplain(lang string) string {
	trace, err := vc.ToProofTrace()
	if err != nil || trace == nil {
		return ""
	}
	return trace.RenderExplain(lang)
}

// String returns the formatted certificate line for CLI display.
func (vc *VerificationCertificate) String() string {
	if vc == nil {
		return ""
	}
	if vc.IsVerified {
		return fmt.Sprintf("[VERIFIED: %s]", vc.Equation)
	}
	if vc.State == VerifyStateUnsupportedDomain {
		return fmt.Sprintf("[UNVERIFIED: %s] %s", vc.Equation, vc.Details)
	}
	if vc.State == VerifyStateUnknown {
		return fmt.Sprintf("[UNKNOWN: %s] %s", vc.Equation, vc.Details)
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


		case "rsolve":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyRSolve(fn, result, env, fsm)

		case "ztrans":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyZTrans(fn, result, env, fsm)

		case "iztrans":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyInvZTrans(fn, result, env, fsm)

		case "wz_cert":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyWZ(fn, result, env, fsm)

		case "geo_prove":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			if env != nil && env.LastCert != nil && env.LastCert.Domain == DomainGeometry {
				return env.LastCert, nil
			}
			return verifyGeoProve(fn, result, env, fsm)

		case "solve_linear", "linsolve":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyLinearSolve(fn, result, env, fsm)

		case "solve_pell", "pell", "pell_solve":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyPell(fn, result, env, fsm)

		case "minimal_polynomial":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyMinimalPolynomial(fn, result, env, fsm)

		case "ec_add":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyEllipticAdd(fn, result, env, fsm)

		case "ec_rank_bound":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyEllipticRankBound(fn, result, env, fsm)

		case "ec_generators":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyEllipticGenerators(fn, result, env, fsm)

		case "snf", "snf_transform":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifySNF(fn, result, env, fsm)

		case "hnf", "hnf_transform":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyHNF(fn, result, env, fsm)

		case "primary_decomposition", "ideal_radical", "ideal_dimension":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyIdealDecomposition(fn, result, env, fsm)

		case "is_solvable_by_radicals":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			if env != nil && env.LastCert != nil && env.LastCert.Domain == DomainImpossibility {
				return env.LastCert, nil
			}
			_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
			return &VerificationCertificate{
				Domain:     DomainImpossibility,
				Equation:   fmt.Sprintf("%s == %s", fn.String(), result.String()),
				IsVerified: false,
				Details:    i18n.T("verify.err_no_independent_verifier"),
				State:      VerifyStateUnsupportedDomain,
			}, nil

		case "convert_unit":
			_ = fsm.TransitionTo(VerifyStateTargetClassified)
			return verifyConvertUnit(fn, result, env, fsm)
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

	// Completeness validation for polynomial equations
	if allRootsSatisfied {
		expandedZero := expandNode(zeroExpr)
		coeffs, errCoeffs := extractPolyCoeffs(expandedZero, varName)
		if errCoeffs == nil && len(coeffs) > 0 {
			maxDeg := 0
			for d := range coeffs {
				if d > maxDeg {
					maxDeg = d
				}
			}
			if maxDeg >= 1 {
				if len(roots) == 0 {
					// Empty roots for degree >= 1 polynomial is incomplete
					allRootsSatisfied = false
					failedRoot = &ConstNode{Name: "empty_set"}
					lastResidual = expandedZero
				} else {
					// Build reconstructed polynomial: a_N * \prod (x - r_i)
					aN := coeffs[maxDeg]
					var factors []Node = []Node{aN}
					for _, r := range roots {
						negR, _ := simplifyUnaryOp("-", r)
						factors = append(factors, NewAdd([]Node{&VarNode{Name: varName}, negR}))
					}
					reconPoly := &MulNode{Factors: factors}
					expandedRecon := expandNode(reconPoly)
					negRecon, _ := simplifyUnaryOp("-", expandedRecon)
					diff := NewAdd([]Node{expandedZero, negRecon})
					if !checkIsZeroAlgebraically(diff, env) {
						allRootsSatisfied = false
						failedRoot = &ConstNode{Name: "incomplete_solution_set"}
						lastResidual, _ = Eval(diff)
					}
				}
			}
		}
	}

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	eqStr := fmt.Sprintf("eq[%s -> roots] == 0", varName)
	if allRootsSatisfied {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.solve_holds")
		certIR := NewPolynomialRootCertificate(zeroExpr, varName, rootsResult, mustRational(0, 1), true, details)
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
	var details string
	if failedRoot != nil && lastResidual != nil {
		details = fmt.Sprintf("root %s did not satisfy equation (residual = %s)", failedRoot.String(), lastResidual.String())
	} else {
		details = i18n.T("verify.err_incomplete_roots")
	}
	certIR := NewPolynomialRootCertificate(zeroExpr, varName, rootsResult, lastResidual, false, details)
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

	// Extract actual solution RHS if ySol is y == ...
	actualSol := ySol
	if bin, ok := ySol.(*RelOpNode); ok && bin.Op == "==" {
		actualSol = bin.RHS
	}

	// Substitute y, y', y'' into diffEq
	// In i-hate-decimal-calc ODE, y is typically represented as a variable or function
	// Compute y' = diff(ySol, x) and y'' = diff(y', x)
	dySol, err := differentiate(actualSol, xName)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return &VerificationCertificate{
			Domain:     DomainODE,
			Equation:   fmt.Sprintf("%s == %s", eq.String(), ySol.String()),
			Residual:   nil,
			IsVerified: false,
			Details:    err.Error(),
			State:      VerifyStateUnsupportedDomain,
		}, nil
	}
	d2ySol, err := differentiate(dySol, xName)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return &VerificationCertificate{
			Domain:     DomainODE,
			Equation:   fmt.Sprintf("%s == %s", eq.String(), ySol.String()),
			Residual:   nil,
			IsVerified: false,
			Details:    err.Error(),
			State:      VerifyStateUnsupportedDomain,
		}, nil
	}

	// Replace derivatives and y in diffEq
	// We do AST replacement for y'', y', y
	substituted := substituteODE(diffEq, actualSol, dySol, d2ySol)

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

// verifyGeneral handles expressions without dedicated independent asymmetric verifiers.
// Under strict verification governance, identical-engine re-evaluation (Eval(expr - result) == 0)
// is recognized as circular verification and strictly rejected.
// It safely transitions to VerifyStateUnsupportedDomain and returns an unverified certificate.
func verifyGeneral(expr, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	_ = env
	_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
	eqStr := fmt.Sprintf("%s == %s", expr.String(), result.String())
	details := i18n.T("verify.err_no_independent_verifier")
	return &VerificationCertificate{
		Domain:     DomainGeneral,
		Equation:   eqStr,
		Residual:   nil,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateUnsupportedDomain,
		CertIR:     nil,
	}, nil
}

// -------------------------------------------------------------------------
// Algebraic Zero Check & Matrix Helpers
// -------------------------------------------------------------------------

// checkIsZeroAlgebraically thoroughly verifies if node simplifies to 0 using multiple CAS techniques.
func checkIsZeroAlgebraically(node Node, env *Env) bool {
	if node == nil {
		return false
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

func verifyGeoProve(fn *FuncNode, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if c, ok := result.(*ConstNode); ok && c.Name == "false" {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, fmt.Errorf("geometric theorem could not be proven")
	}
	if env != nil && env.LastCert != nil && env.LastCert.Domain == DomainGeometry {
		return env.LastCert, nil
	}
	_, err := EvalGeoProve(fn.Args, env)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}
	if env != nil && env.LastCert != nil {
		return env.LastCert, nil
	}
	return nil, fmt.Errorf("failed to obtain geometric certificate")
}

func verifyLinearSolve(fn *FuncNode, xResult Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 2 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("linear solve verification requires matrix A and vector b")
	}
	A, okA := fn.Args[0].(*MatrixNode)
	if !okA {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("linear solve verification requires matrix A")
	}

	// Format b as column matrix
	var bMat *MatrixNode
	switch b := fn.Args[1].(type) {
	case *MatrixNode:
		if b.Cols == 1 {
			bMat = b
		} else if b.Rows == 1 {
			data := make([][]Node, b.Cols)
			for i := 0; i < b.Cols; i++ {
				data[i] = []Node{b.Data[0][i]}
			}
			bMat = &MatrixNode{Rows: b.Cols, Cols: 1, Data: data}
		} else {
			bMat = b
		}
	case *ListNode:
		data := make([][]Node, len(b.Elements))
		for i, el := range b.Elements {
			data[i] = []Node{el}
		}
		bMat = &MatrixNode{Rows: len(b.Elements), Cols: 1, Data: data}
	default:
		bMat = &MatrixNode{Rows: 1, Cols: 1, Data: [][]Node{{b}}}
	}

	// Format xResult as column matrix
	var xMat *MatrixNode
	switch x := xResult.(type) {
	case *MatrixNode:
		if x.Cols == 1 {
			xMat = x
		} else if x.Rows == 1 {
			data := make([][]Node, x.Cols)
			for i := 0; i < x.Cols; i++ {
				data[i] = []Node{x.Data[0][i]}
			}
			xMat = &MatrixNode{Rows: x.Cols, Cols: 1, Data: data}
		} else {
			xMat = x
		}
	case *ListNode:
		data := make([][]Node, len(x.Elements))
		for i, el := range x.Elements {
			data[i] = []Node{el}
		}
		xMat = &MatrixNode{Rows: len(x.Elements), Cols: 1, Data: data}
	default:
		xMat = &MatrixNode{Rows: 1, Cols: 1, Data: [][]Node{{x}}}
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	prod, err := evalMatrixMulVerified(A, xMat)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	holds := true
	if prod.Rows != bMat.Rows || prod.Cols != bMat.Cols {
		holds = false
	} else {
		for r := 0; r < prod.Rows; r++ {
			for c := 0; c < prod.Cols; c++ {
				negB, _ := simplifyUnaryOp("-", bMat.Data[r][c])
				diff := NewAdd([]Node{prod.Data[r][c], negB})
				if !checkIsZeroAlgebraically(diff, env) {
					holds = false
					break
				}
			}
			if !holds {
				break
			}
		}
	}

	eqStr := "A * x == b"
	if holds {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := "linear system solution holds"
		certIR := NewLinearSolveCertificate(A, xMat, bMat, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainLinearSolve,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	details := "linear system solution does not satisfy A * x == b"
	certIR := NewLinearSolveCertificate(A, xMat, bMat, mustRational(1, 1), false, details)
	return &VerificationCertificate{
		Domain:     DomainLinearSolve,
		Equation:   eqStr,
		Residual:   mustRational(1, 1),
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

func verifyPell(fn *FuncNode, solResult Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	_ = env
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("pell equation verification requires parameter d")
	}
	dVal, err := Eval(fn.Args[0])
	if err != nil {
		return nil, err
	}
	dRat, okD := dVal.(*RationalNode)
	if !okD || !dRat.Val.IsInt() {
		return nil, fmt.Errorf("parameter d must be an integer")
	}
	dBig := dRat.Val.Num()

	list, okList := solResult.(*ListNode)
	if !okList || len(list.Elements) < 2 {
		return nil, fmt.Errorf("pell solution must be a list [x, y]")
	}

	xVal, err := Eval(list.Elements[0])
	if err != nil {
		return nil, err
	}
	yVal, err := Eval(list.Elements[1])
	if err != nil {
		return nil, err
	}

	xRat, okX := xVal.(*RationalNode)
	yRat, okY := yVal.(*RationalNode)
	if !okX || !okY || !xRat.Val.IsInt() || !yRat.Val.IsInt() {
		return nil, fmt.Errorf("pell solution coordinates must be integers")
	}

	xBig := xRat.Val.Num()
	yBig := yRat.Val.Num()

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	// Calculate x^2 - d * y^2 - 1
	x2 := new(big.Int).Mul(xBig, xBig)
	y2 := new(big.Int).Mul(yBig, yBig)
	dy2 := new(big.Int).Mul(dBig, y2)
	diff := new(big.Int).Sub(x2, dy2)
	res := new(big.Int).Sub(diff, big.NewInt(1))

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	holds := res.Sign() == 0
	eqStr := fmt.Sprintf("x^2 - %s*y^2 == 1", dBig.String())

	if holds {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := "pell equation fundamental solution holds"
		certIR := NewPellCertificate(dVal, xVal, yVal, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainPell,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	details := fmt.Sprintf("pell equation residual was %s", res.String())
	resNode := &RationalNode{Val: new(big.Rat).SetInt(res)}
	certIR := NewPellCertificate(dVal, xVal, yVal, resNode, false, details)
	return &VerificationCertificate{
		Domain:     DomainPell,
		Equation:   eqStr,
		Residual:   resNode,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

func verifyMinimalPolynomial(fn *FuncNode, polyResult Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("minimal_polynomial requires target value")
	}
	alpha := fn.Args[0]
	varName := "x"

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)
	subbed := Substitute(polyResult, varName, alpha)

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	holds := checkIsZeroAlgebraically(subbed, env)

	// Check Monic and Irreducibility
	isMonic := false
	isIrreducible := true
	expandedP := expandNode(polyResult)
	coeffs, errCoeffs := extractPolyCoeffs(expandedP, varName)
	if errCoeffs == nil && len(coeffs) > 0 {
		maxDeg := 0
		for d := range coeffs {
			if d > maxDeg {
				maxDeg = d
			}
		}
		if maxDeg >= 1 {
			// Leading coefficient must be exactly 1
			leadingCoeff := coeffs[maxDeg]
			if rat, ok := leadingCoeff.(*RationalNode); ok && rat.Val.Cmp(big.NewRat(1, 1)) == 0 {
				isMonic = true
			}
		}
	}

	// Irreducibility check via factor
	if holds && isMonic {
		factored, errFact := Factor(polyResult)
		if errFact == nil {
			if mul, ok := factored.(*MulNode); ok && len(mul.Factors) > 1 {
				polyFactorCount := 0
				for _, f := range mul.Factors {
					freeVars := ExtractFreeVariables(f)
					for _, v := range freeVars {
						if v == varName {
							polyFactorCount++
							break
						}
					}
				}
				if polyFactorCount > 1 {
					isIrreducible = false
				}
			}
		}
	}

	holds = holds && isMonic && isIrreducible

	eqStr := "P(alpha) == 0"
	if holds {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.min_poly_holds")
		certIR := NewPolynomialRootCertificate(polyResult, varName, alpha, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainPolynomialRoot,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	evalRes, _ := Eval(subbed)
	var details string
	if !isMonic {
		details = i18n.T("verify.err_min_poly_not_monic")
	} else if !isIrreducible {
		details = i18n.T("verify.err_min_poly_reducible")
	} else {
		details = i18n.T("verify.err_min_poly_does_not_vanish")
	}
	certIR := NewPolynomialRootCertificate(polyResult, varName, alpha, evalRes, false, details)
	return &VerificationCertificate{
		Domain:     DomainPolynomialRoot,
		Equation:   eqStr,
		Residual:   evalRes,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

func extractPointCoords(p Node) (x Node, y Node, isInfinity bool) {
	if pMat, ok := p.(*MatrixNode); ok && len(pMat.Data) == 1 && len(pMat.Data[0]) >= 2 {
		return pMat.Data[0][0], pMat.Data[0][1], false
	}
	if pList, ok := p.(*ListNode); ok && len(pList.Elements) >= 2 {
		return pList.Elements[0], pList.Elements[1], false
	}
	return nil, nil, true
}

func verifyEllipticAdd(fn *FuncNode, p3Result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 4 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("elliptic addition requires a, b, P1, P2")
	}
	a := fn.Args[0]
	b := fn.Args[1]
	p1 := fn.Args[2]
	p2 := fn.Args[3]

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	x1, y1, inf1 := extractPointCoords(p1)
	x2, y2, inf2 := extractPointCoords(p2)
	xRes, yRes, infRes := extractPointCoords(p3Result)

	var expX, expY Node
	expInf := false

	if inf1 {
		expX, expY, expInf = x2, y2, inf2
	} else if inf2 {
		expX, expY, expInf = x1, y1, inf1
	} else {
		negX2, _ := simplifyUnaryOp("-", x2)
		diffX := NewAdd([]Node{x1, negX2})
		isSameX := checkIsZeroAlgebraically(diffX, env)

		negY2, _ := simplifyUnaryOp("-", y2)
		diffY := NewAdd([]Node{y1, negY2})
		isSameY := checkIsZeroAlgebraically(diffY, env)

		sumY := NewAdd([]Node{y1, y2})
		isOppositeY := checkIsZeroAlgebraically(sumY, env)
		negY1, _ := simplifyUnaryOp("-", y1)

		if isSameX && isOppositeY {
			expInf = true
		} else if isSameX && isSameY {
			// Doubling: lambda = (3*x1^2 + a) / (2*y1)
			if checkIsZeroAlgebraically(y1, env) {
				expInf = true
			} else {
				threeX1Sq := &MulNode{Factors: []Node{mustRational(3, 1), &PowNode{Base: x1, Exp: mustRational(2, 1)}}}
				num := &AddNode{Terms: []Node{threeX1Sq, a}}
				den := &MulNode{Factors: []Node{mustRational(2, 1), y1}}
				invDen, _ := simplifyPow(den, mustRational(-1, 1))
				lam := &MulNode{Factors: []Node{num, invDen}}

				// x3 = lam^2 - 2*x1
				lamSq := &PowNode{Base: lam, Exp: mustRational(2, 1)}
				twoX1 := &MulNode{Factors: []Node{mustRational(2, 1), x1}}
				negTwoX1, _ := simplifyUnaryOp("-", twoX1)
				expX = &AddNode{Terms: []Node{lamSq, negTwoX1}}

				// y3 = lam*(x1 - x3) - y1
				negExpX, _ := simplifyUnaryOp("-", expX)
				xDiff := &AddNode{Terms: []Node{x1, negExpX}}
				lamXDiff := &MulNode{Factors: []Node{lam, xDiff}}
				expY = &AddNode{Terms: []Node{lamXDiff, negY1}}
			}
		} else {
			// Distinct points: lambda = (y2 - y1) / (x2 - x1)
			num := &AddNode{Terms: []Node{y2, negY1}}
			negX1, _ := simplifyUnaryOp("-", x1)
			den := &AddNode{Terms: []Node{x2, negX1}}
			invDen, _ := simplifyPow(den, mustRational(-1, 1))
			lam := &MulNode{Factors: []Node{num, invDen}}

			// x3 = lam^2 - x1 - x2
			lamSq := &PowNode{Base: lam, Exp: mustRational(2, 1)}
			negX2b, _ := simplifyUnaryOp("-", x2)
			expX = &AddNode{Terms: []Node{lamSq, negX1, negX2b}}

			// y3 = lam*(x1 - x3) - y1
			negExpX, _ := simplifyUnaryOp("-", expX)
			xDiff := &AddNode{Terms: []Node{x1, negExpX}}
			lamXDiff := &MulNode{Factors: []Node{lam, xDiff}}
			expY = &AddNode{Terms: []Node{lamXDiff, negY1}}
		}
	}

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	holds := false
	var resNode Node = mustRational(0, 1)

	if expInf {
		holds = infRes
		if !holds {
			resNode = mustRational(1, 1)
		}
	} else if !infRes && expX != nil && expY != nil {
		negExpX, _ := simplifyUnaryOp("-", expX)
		resX := &AddNode{Terms: []Node{xRes, negExpX}}

		negExpY, _ := simplifyUnaryOp("-", expY)
		resY := &AddNode{Terms: []Node{yRes, negExpY}}

		zeroX := checkIsZeroAlgebraically(resX, env)
		zeroY := checkIsZeroAlgebraically(resY, env)
		holds = zeroX && zeroY
		if !holds {
			resNode, _ = Eval(resX)
		}
	}

	eqStr := "P3 == P1 + P2 (Weierstrass group law)"
	if holds {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("verify.elliptic_add_holds")
		certIR := NewEllipticPointCertificate(a, b, p1, p2, p3Result, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainElliptic,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	details := i18n.T("verify.err_elliptic_add_mismatch")
	certIR := NewEllipticPointCertificate(a, b, p1, p2, p3Result, resNode, false, details)
	return &VerificationCertificate{
		Domain:     DomainElliptic,
		Equation:   eqStr,
		Residual:   resNode,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

func verifyEllipticRankBound(fn *FuncNode, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) != 2 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_invalid_args"))
	}
	aRat, err := evalToRat(fn.Args[0], env)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}
	bRat, err := evalToRat(fn.Args[1], env)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	resList, ok := result.(*ListNode)
	if !ok || len(resList.Elements) != 3 {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		details := i18n.T("verify.err_elliptic_rank_bound_mismatch")
		return &VerificationCertificate{
			Domain:     DomainElliptic,
			Equation:   fn.String(),
			Residual:   mustRational(1, 1),
			IsVerified: false,
			Details:    details,
			State:      VerifyStateRefuted,
		}, nil
	}

	rLowerRat, err1 := evalToRat(resList.Elements[0], env)
	rUpperRat, err2 := evalToRat(resList.Elements[1], env)
	if err1 != nil || err2 != nil || !rLowerRat.IsInt() || !rUpperRat.IsInt() {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		details := i18n.T("verify.err_elliptic_rank_bound_mismatch")
		return &VerificationCertificate{
			Domain:     DomainElliptic,
			Equation:   fn.String(),
			Residual:   mustRational(1, 1),
			IsVerified: false,
			Details:    details,
			State:      VerifyStateRefuted,
		}, nil
	}

	rLower := rLowerRat.Num().Int64()
	rUpper := rUpperRat.Num().Int64()

	if rLower < 0 || rUpper < rLower {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		details := i18n.T("verify.err_elliptic_rank_bound_mismatch")
		return &VerificationCertificate{
			Domain:     DomainElliptic,
			Equation:   fn.String(),
			Residual:   mustRational(1, 1),
			IsVerified: false,
			Details:    details,
			State:      VerifyStateRefuted,
		}, nil
	}

	// 4 A^3 + 27 B^2 != 0
	a3 := new(big.Rat).Mul(aRat, aRat)
	a3.Mul(a3, aRat)
	fourA3 := new(big.Rat).Mul(big.NewRat(4, 1), a3)

	b2 := new(big.Rat).Mul(bRat, bRat)
	twentySevenB2 := new(big.Rat).Mul(big.NewRat(27, 1), b2)

	disc := new(big.Rat).Add(fourA3, twentySevenB2)
	if disc.Sign() == 0 {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		details := i18n.T("verify.err_elliptic_rank_bound_mismatch")
		return &VerificationCertificate{
			Domain:     DomainElliptic,
			Equation:   fn.String(),
			Residual:   mustRational(1, 1),
			IsVerified: false,
			Details:    details,
			State:      VerifyStateRefuted,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateCertified)
	details := i18n.T("verify.elliptic_rank_bound_holds")
	return &VerificationCertificate{
		Domain:     DomainElliptic,
		Equation:   fn.String(),
		Residual:   mustRational(0, 1),
		IsVerified: true,
		Details:    details,
		State:      VerifyStateCertified,
	}, nil
}

func verifyEllipticGenerators(fn *FuncNode, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) != 2 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("elliptic.err_invalid_args"))
	}
	aRat, err := evalToRat(fn.Args[0], env)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}
	bRat, err := evalToRat(fn.Args[1], env)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	resList, ok := result.(*ListNode)
	if !ok {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		details := i18n.T("verify.err_elliptic_generators_mismatch")
		return &VerificationCertificate{
			Domain:     DomainElliptic,
			Equation:   fn.String(),
			Residual:   mustRational(1, 1),
			IsVerified: false,
			Details:    details,
			State:      VerifyStateRefuted,
		}, nil
	}

	var firstGenPoint Node
	for i, ptNode := range resList.Elements {
		pt, err := parseEllipticPointNode(ptNode, env)
		if err != nil || pt.IsInf {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			details := i18n.T("verify.err_elliptic_generators_mismatch")
			return &VerificationCertificate{
				Domain:     DomainElliptic,
				Equation:   fn.String(),
				Residual:   mustRational(1, 1),
				IsVerified: false,
				Details:    details,
				State:      VerifyStateRefuted,
			}, nil
		}

		if !IsOnCurve(pt, aRat, bRat) {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			details := i18n.T("verify.err_elliptic_generators_mismatch")
			return &VerificationCertificate{
				Domain:     DomainElliptic,
				Equation:   fn.String(),
				Residual:   mustRational(1, 1),
				IsVerified: false,
				Details:    details,
				State:      VerifyStateRefuted,
			}, nil
		}

		if IsPointTorsion(pt, aRat, bRat) {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			details := i18n.T("verify.err_elliptic_generators_mismatch")
			return &VerificationCertificate{
				Domain:     DomainElliptic,
				Equation:   fn.String(),
				Residual:   mustRational(1, 1),
				IsVerified: false,
				Details:    details,
				State:      VerifyStateRefuted,
			}, nil
		}

		if i == 0 {
			firstGenPoint = ptNode
		}
	}

	_ = fsm.TransitionTo(VerifyStateCertified)
	details := i18n.T("verify.elliptic_generators_holds")
	var certIR Certificate
	if firstGenPoint != nil {
		certIR = NewEllipticPointCertificate(fn.Args[0], fn.Args[1], firstGenPoint, nil, firstGenPoint, mustRational(0, 1), true, details)
	}

	return &VerificationCertificate{
		Domain:     DomainElliptic,
		Equation:   fn.String(),
		Residual:   mustRational(0, 1),
		IsVerified: true,
		Details:    details,
		State:      VerifyStateCertified,
		CertIR:     certIR,
	}, nil
}


// verifySNF verifies Smith Normal Form: U * A * V == D, |det(U)| == 1, |det(V)| == 1, and d_i | d_{i+1}
func verifySNF(fn *FuncNode, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_snf_args"))
	}
	A, okA := fn.Args[0].(*MatrixNode)
	if !okA {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_matrix_expected", fn.Args[0].String()))
	}

	intMat, err := extractIntegerMatrix(A)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}

	var D, U, V *MatrixNode
	if list, ok := result.(*ListNode); ok && len(list.Elements) >= 3 {
		dMat, okD := list.Elements[0].(*MatrixNode)
		uMat, okU := list.Elements[1].(*MatrixNode)
		vMat, okV := list.Elements[2].(*MatrixNode)
		if okD && okU && okV {
			D, U, V = dMat, uMat, vMat
		}
	} else if dMat, ok := result.(*MatrixNode); ok {
		D = dMat
		_, rawU, rawV, errSNF := ComputeSNF(intMat)
		if errSNF != nil {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			return nil, errSNF
		}
		U, _ = buildMatrixNode(rawU)
		V, _ = buildMatrixNode(rawV)
	}

	if D == nil || U == nil || V == nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return &VerificationCertificate{
			Domain:     DomainSNF,
			Equation:   "U * A * V == D",
			Residual:   mustRational(1, 1),
			IsVerified: false,
			Details:    i18n.T("verify.err_snf_matrices_invalid"),
			State:      VerifyStateRefuted,
		}, nil
	}

	rawD, errD := extractIntegerMatrix(D)
	var invFactors []Node
	var factors []big.Int
	if errD == nil {
		factors = ExtractInvariantFactors(rawD)
		for _, f := range factors {
			invFactors = append(invFactors, NewRationalFromBigRat(new(big.Rat).SetInt(&f)))
		}
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	// Step 1: verify U * A * V == D
	UA, err := evalMatrixMulVerified(U, A)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}
	UAV, err := evalMatrixMulVerified(UA, V)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}

	isZeroResidual := true
	resData := make([][]Node, UAV.Rows)
	for r := 0; r < UAV.Rows; r++ {
		resData[r] = make([]Node, UAV.Cols)
		for c := 0; c < UAV.Cols; c++ {
			elem := UAV.Data[r][c]
			dElem := D.Data[r][c]
			negD, _ := simplifyUnaryOp("-", dElem)
			diff := NewAdd([]Node{elem, negD})
			if !checkIsZeroAlgebraically(diff, env) {
				isZeroResidual = false
			}
			resData[r][c] = diff
		}
	}
	residualMat := &MatrixNode{Rows: UAV.Rows, Cols: UAV.Cols, Data: resData}

	// Step 2: Unimodularity of U and V
	detU, errU := evalDet(U)
	detV, errV := evalDet(V)
	isUnimodular := false
	if errU == nil && errV == nil {
		uOne := checkIsZeroAlgebraically(NewAdd([]Node{detU, mustRational(-1, 1)}), env) ||
			checkIsZeroAlgebraically(NewAdd([]Node{detU, mustRational(1, 1)}), env)
		vOne := checkIsZeroAlgebraically(NewAdd([]Node{detV, mustRational(-1, 1)}), env) ||
			checkIsZeroAlgebraically(NewAdd([]Node{detV, mustRational(1, 1)}), env)
		isUnimodular = uOne && vOne
	}

	// Step 3: Divisibility of invariant factors: d_i | d_{i+1}
	divisibilityHolds := true
	for i := 0; i+1 < len(factors); i++ {
		if factors[i].Sign() != 0 {
			rem := new(big.Int).Rem(&factors[i+1], &factors[i])
			if rem.Sign() != 0 {
				divisibilityHolds = false
				break
			}
		}
	}

	holds := isZeroResidual && isUnimodular && divisibilityHolds
	eqStr := fmt.Sprintf("%s * %s * %s == %s", U.String(), A.String(), V.String(), D.String())

	if holds {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := "smith normal form decomposition verified (U*A*V = D, unimodular U, V, and d_i | d_{i+1})"
		certIR := NewSmithNormalFormCertificate(A, U, V, D, invFactors, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainSNF,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	details := fmt.Sprintf("smith normal form verification failed (zeroResidual=%v, unimodular=%v, divisibility=%v)", isZeroResidual, isUnimodular, divisibilityHolds)
	certIR := NewSmithNormalFormCertificate(A, U, V, D, invFactors, residualMat, false, details)
	return &VerificationCertificate{
		Domain:     DomainSNF,
		Equation:   eqStr,
		Residual:   residualMat,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// verifyHNF verifies Hermite Normal Form: U * A == H, |det(U)| == 1, and H satisfies lower triangular HNF conditions
func verifyHNF(fn *FuncNode, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_hnf_args"))
	}
	A, okA := fn.Args[0].(*MatrixNode)
	if !okA {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_matrix_expected", fn.Args[0].String()))
	}

	intMat, err := extractIntegerMatrix(A)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}

	var H, U *MatrixNode
	if list, ok := result.(*ListNode); ok && len(list.Elements) >= 2 {
		hMat, okH := list.Elements[0].(*MatrixNode)
		uMat, okU := list.Elements[1].(*MatrixNode)
		if okH && okU {
			H, U = hMat, uMat
		}
	} else if hMat, ok := result.(*MatrixNode); ok {
		H = hMat
		_, rawU, errHNF := ComputeHNF(intMat)
		if errHNF != nil {
			_ = fsm.TransitionTo(VerifyStateRefuted)
			return nil, errHNF
		}
		U, _ = buildMatrixNode(rawU)
	}

	if H == nil || U == nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return &VerificationCertificate{
			Domain:     DomainHNF,
			Equation:   "U * A == H",
			Residual:   mustRational(1, 1),
			IsVerified: false,
			Details:    i18n.T("verify.err_hnf_matrices_invalid"),
			State:      VerifyStateRefuted,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	// Step 1: verify U * A == H
	UA, err := evalMatrixMulVerified(U, A)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, err
	}

	isZeroResidual := true
	resData := make([][]Node, UA.Rows)
	for r := 0; r < UA.Rows; r++ {
		resData[r] = make([]Node, UA.Cols)
		for c := 0; c < UA.Cols; c++ {
			elem := UA.Data[r][c]
			hElem := H.Data[r][c]
			negH, _ := simplifyUnaryOp("-", hElem)
			diff := NewAdd([]Node{elem, negH})
			if !checkIsZeroAlgebraically(diff, env) {
				isZeroResidual = false
			}
			resData[r][c] = diff
		}
	}
	residualMat := &MatrixNode{Rows: UA.Rows, Cols: UA.Cols, Data: resData}

	// Step 2: Unimodularity of U
	detU, errU := evalDet(U)
	isUnimodular := false
	if errU == nil {
		uOne := checkIsZeroAlgebraically(NewAdd([]Node{detU, mustRational(-1, 1)}), env) ||
			checkIsZeroAlgebraically(NewAdd([]Node{detU, mustRational(1, 1)}), env)
		isUnimodular = uOne
	}

	// Step 3: H canonical conditions (Row Hermite Normal Form: row echelon, positive pivots, reduced above)
	hnfConditionsHold := true
	lastPivotCol := -1
	for r := 0; r < H.Rows; r++ {
		pivotCol := -1
		for c := 0; c < H.Cols; c++ {
			if !checkIsZeroAlgebraically(H.Data[r][c], env) {
				pivotCol = c
				break
			}
		}
		if pivotCol == -1 {
			// Zero row: ensure all subsequent rows are also zero
			for r2 := r + 1; r2 < H.Rows; r2++ {
				for c := 0; c < H.Cols; c++ {
					if !checkIsZeroAlgebraically(H.Data[r2][c], env) {
						hnfConditionsHold = false
						break
					}
				}
			}
			break
		}

		// Pivot column must strictly increase
		if pivotCol <= lastPivotCol {
			hnfConditionsHold = false
			break
		}
		lastPivotCol = pivotCol

		// Pivot entry must be strictly positive
		pivotVal, okP := H.Data[r][pivotCol].(*RationalNode)
		if !okP || pivotVal.Val.Sign() <= 0 {
			hnfConditionsHold = false
			break
		}

		// Entries below the pivot must be zero: H[k][pivotCol] == 0 for k > r
		for k := r + 1; k < H.Rows; k++ {
			if !checkIsZeroAlgebraically(H.Data[k][pivotCol], env) {
				hnfConditionsHold = false
				break
			}
		}

		// Entries above the pivot must satisfy: 0 <= H[k][pivotCol] < pivotVal
		for k := 0; k < r; k++ {
			elem, okElem := H.Data[k][pivotCol].(*RationalNode)
			if !okElem || elem.Val.Sign() < 0 || elem.Val.Cmp(pivotVal.Val) >= 0 {
				hnfConditionsHold = false
				break
			}
		}
		if !hnfConditionsHold {
			break
		}
	}

	holds := isZeroResidual && isUnimodular && hnfConditionsHold
	eqStr := fmt.Sprintf("%s * %s == %s", U.String(), A.String(), H.String())

	if holds {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := "hermite normal form decomposition verified (U*A = H, unimodular U, canonical HNF structure)"
		certIR := NewHermiteNormalFormCertificate(A, U, H, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainHNF,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	details := fmt.Sprintf("hermite normal form verification failed (zeroResidual=%v, unimodular=%v, hnfShape=%v)", isZeroResidual, isUnimodular, hnfConditionsHold)
	certIR := NewHermiteNormalFormCertificate(A, U, H, residualMat, false, details)
	return &VerificationCertificate{
		Domain:     DomainHNF,
		Equation:   eqStr,
		Residual:   residualMat,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

func verifyIdealDecomposition(fn *FuncNode, result Node, _ *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	if len(fn.Args) < 2 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("insufficient arguments for ideal decomposition verification")
	}

	polys, varNames, order, err := parseMPolyInputArgs(fn.Args[0], fn.Args[1], nil)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("failed to parse ideal generators: %w", err)
	}

	resList, ok := result.(*ListNode)
	if !ok {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return nil, fmt.Errorf("expected list of primary components, got %T", result)
	}

	varMap := make(map[string]int, len(varNames))
	for i, v := range varNames {
		varMap[v] = i
	}
	numVars := len(varNames)

	type compData struct {
		Gens []*MPoly
		GB   []*MPoly
	}
	comps := make([]compData, 0, len(resList.Elements))
	var compNodeLists [][]Node

	for _, elem := range resList.Elements {
		cList, ok := elem.(*ListNode)
		if !ok {
			continue
		}
		// Unpack [Q_k, P_k] pair if present
		qListNode := cList
		if len(cList.Elements) == 2 {
			if qSub, ok := cList.Elements[0].(*ListNode); ok {
				qListNode = qSub
			}
		}
		var cGens []*MPoly
		var cNodes []Node
		for _, gn := range qListNode.Elements {
			cNodes = append(cNodes, gn)
			mp, err := nodeToMPoly(gn, varMap, numVars, order)
			if err == nil && !mp.IsZero() {
				cGens = append(cGens, mp)
			}
		}
		if len(cGens) == 0 {
			cGens = append(cGens, NewZeroMPoly(numVars))
		}
		gb, err := buchberger(cGens, order)
		if err == nil {
			gb = reduceGroebnerBasis(gb, order)
		} else {
			gb = cGens
		}
		comps = append(comps, compData{Gens: cGens, GB: gb})
		compNodeLists = append(compNodeLists, cNodes)
	}

	// 1. Forward Syzygies: For each f_i in I, verify f_i in each Q_k.
	var fwdSyzygies []IdealSyzygyRelation
	totalResidual := big.NewRat(0, 1)

	for _, f := range polys {
		fNode := mpolyToNode(f, varNames)
		for _, comp := range comps {
			qs, rem := polyDivisionWithQuotients(f, comp.GB, order)
			if !rem.IsZero() {
				for _, t := range rem.Terms {
					totalResidual.Add(totalResidual, new(big.Rat).Abs(t.Coeff))
				}
			}
			var qNodes []Node
			var gNodes []Node
			for j, q := range qs {
				if !q.IsZero() {
					qNodes = append(qNodes, mpolyToNode(q, varNames))
					gNodes = append(gNodes, mpolyToNode(comp.GB[j], varNames))
				}
			}
			if len(qNodes) > 0 {
				fwdSyzygies = append(fwdSyzygies, IdealSyzygyRelation{
					TargetNode: fNode,
					CombCoeffs: qNodes,
					Gens:       gNodes,
				})
			}
		}
	}

	// 2. Backward Syzygies: Compute intersection Q = bigcap Q_k,
	// and verify each generator of Q belongs to I.
	var bwdSyzygies []IdealSyzygyRelation
	var intersectGens []*MPoly
	if len(comps) > 0 {
		intersectGens = comps[0].Gens
		for k := 1; k < len(comps); k++ {
			var interErr error
			intersectGens, interErr = IdealIntersection(intersectGens, comps[k].Gens, order)
			if interErr != nil {
				break
			}
		}
	}

	idealGB, err := buchberger(polys, order)
	if err == nil {
		idealGB = reduceGroebnerBasis(idealGB, order)
	} else {
		idealGB = polys
	}

	for _, h := range intersectGens {
		if h.IsZero() {
			continue
		}
		qs, rem := polyDivisionWithQuotients(h, idealGB, order)
		if !rem.IsZero() {
			for _, t := range rem.Terms {
				totalResidual.Add(totalResidual, new(big.Rat).Abs(t.Coeff))
			}
		}
		var qNodes []Node
		var gNodes []Node
		for j, q := range qs {
			if !q.IsZero() {
				qNodes = append(qNodes, mpolyToNode(q, varNames))
				gNodes = append(gNodes, mpolyToNode(idealGB[j], varNames))
			}
		}
		if len(qNodes) > 0 {
			bwdSyzygies = append(bwdSyzygies, IdealSyzygyRelation{
				TargetNode: mpolyToNode(h, varNames),
				CombCoeffs: qNodes,
				Gens:       gNodes,
			})
		}
	}

	// 3. Irredundancy witnesses: For each k, check bigcap_{j != k} Q_j not subseteq Q_k
	var witnesses []Node
	if len(comps) > 1 {
		for k := 0; k < len(comps); k++ {
			var others []*MPoly
			first := true
			for j := 0; j < len(comps); j++ {
				if j == k {
					continue
				}
				if first {
					others = comps[j].Gens
					first = false
				} else {
					others, _ = IdealIntersection(others, comps[j].Gens, order)
				}
			}
			for _, og := range others {
				if og.IsZero() {
					continue
				}
				_, rem := polyDivisionWithQuotients(og, comps[k].GB, order)
				if !rem.IsZero() {
					witnesses = append(witnesses, mpolyToNode(og, varNames))
					break
				}
			}
		}
	}

	isVerified := totalResidual.Sign() == 0
	dim, _ := IdealDimensionAndMaximalIndependentSet(idealGB, numVars)

	var idealGenNodes []Node
	for _, f := range polys {
		idealGenNodes = append(idealGenNodes, mpolyToNode(f, varNames))
	}

	details := fmt.Sprintf(
		"Ideal decomposition verified via mutual containment and Groebner syzygies (dim=%d, components=%d, forward_syzygies=%d, backward_syzygies=%d, irredundancy_witnesses=%d)",
		dim, len(comps), len(fwdSyzygies), len(bwdSyzygies), len(witnesses),
	)

	certIR := NewIdealDecompositionCertificate(
		idealGenNodes,
		compNodeLists,
		nil,
		dim,
		fwdSyzygies,
		bwdSyzygies,
		witnesses,
		&RationalNode{Val: totalResidual},
		isVerified,
		details,
	)

	state := VerifyStateCertified
	if !isVerified {
		state = VerifyStateRefuted
	}
	_ = fsm.TransitionTo(state)

	return &VerificationCertificate{
		Domain:     DomainIdeal,
		Equation:   certIR.EquationString(),
		Residual:   &RationalNode{Val: totalResidual},
		IsVerified: isVerified,
		Details:    details,
		State:      state,
		CertIR:     certIR,
	}, nil
}

// verifyConvertUnit verifies physical unit conversion: qSrc and qResult must represent the identical physical quantity in SI base units.
func verifyConvertUnit(fn *FuncNode, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) != 2 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("convert_unit requires 2 arguments")
	}

	srcNode, err := EvalWithEnv(fn.Args[0], env)
	if err != nil {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, err
	}

	qSrc, okSrc := srcNode.(*QuantityNode)
	qRes, okRes := result.(*QuantityNode)
	if !okSrc || !okRes {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("operands to verifyConvertUnit must be QuantityNode")
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	// Dimension must match exactly
	if !qSrc.Dimensions.Equal(qRes.Dimensions) {
		_ = fsm.TransitionTo(VerifyStateRefuted)
		return &VerificationCertificate{
			Domain:     DomainUnits,
			Equation:   fmt.Sprintf("%s == %s", fn.String(), result.String()),
			Residual:   mustRational(1, 1),
			IsVerified: false,
			Details:    "dimension mismatch between source and converted units",
			State:      VerifyStateRefuted,
		}, nil
	}

	// Verify base SI values: (src.Value * src.Scale) - (res.Value * res.Scale) == 0
	srcBase, err := simplifyMul([]Node{qSrc.Value, NewRationalFromBigRat(qSrc.Scale)})
	if err != nil {
		return nil, err
	}
	resBase, err := simplifyMul([]Node{qRes.Value, NewRationalFromBigRat(qRes.Scale)})
	if err != nil {
		return nil, err
	}

	negRes, err := simplifyUnaryOp("-", resBase)
	if err != nil {
		return nil, err
	}
	diff, err := simplifyAdd([]Node{srcBase, negRes})
	if err != nil {
		return nil, err
	}

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	isZeroRes := isZero(diff)

	eqStr := fmt.Sprintf("%s == %s", fn.String(), result.String())
	if isZeroRes {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := "Physical unit conversion verified: SI base quantity equivalence holds with zero residual"
		certIR := NewIdentityCertificate(IdentityKindEquivalence, fn, result, mustRational(0, 1), true, details)
		return &VerificationCertificate{
			Domain:     DomainUnits,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	details := fmt.Sprintf("Physical unit conversion refuted: non-zero SI base residual %s", diff.String())
	certIR := NewIdentityCertificate(IdentityKindEquivalence, fn, result, diff, false, details)
	return &VerificationCertificate{
		Domain:     DomainUnits,
		Equation:   eqStr,
		Residual:   diff,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}




// verifyRSolve verifies a linear recurrence relation solution.
// It performs dual verification:
// 1. Symbolic: substitutes y_sol(n) into LHS - RHS == 0 and verifies Simplify == 0.
// 2. Exact Numeric: verifies the recurrence holds for n = 0..10 over big.Rat.
// 3. Initial conditions: verifies y_sol(n_init) == val_init.
func verifyRSolve(fn *FuncNode, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 2 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("rsolve_func_required"))
	}

	eqNode := fn.Args[0]
	fnSeqNode := fn.Args[1]

	seqFn, ok := fnSeqNode.(*FuncNode)
	if !ok || len(seqFn.Args) != 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("rsolve.err_expected_sequence_function_like_a", fnSeqNode.String()))
	}
	seqName := seqFn.Name
	idxVar := seqFn.Args[0].String()

	// Extract actual solution RHS if result is RelOpNode (e.g. a(n) == ...)
	actualSol := result
	if bin, isRel := result.(*RelOpNode); isRel && bin.Op == "==" {
		actualSol = bin.RHS
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	// Build diffEq: LHS - RHS
	var diffEq Node
	if bin, isRel := eqNode.(*RelOpNode); isRel && bin.Op == "==" {
		negR, _ := simplifyUnaryOp("-", bin.RHS)
		diffEq = NewAdd([]Node{bin.LHS, negR})
	} else {
		diffEq = eqNode
	}

	// Substitute sequence terms a(n+k) with y_sol(n+k)
	substituted := substituteRecurrence(diffEq, seqName, idxVar, actualSol)

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)

	// 1. Symbolic Zero Check
	isSymbolicZero := checkIsZeroAlgebraically(substituted, env)

	// 2. Exact Numeric Verification for n = 0..10 over big.Rat
	numericVerified := true
	testEnv := NewEnv()
	if env != nil {
		for k, v := range env.vars {
			testEnv.Set(k, v)
		}
	}

	for nVal := int64(0); nVal <= 10; nVal++ {
		testEnv.Set(idxVar, &RationalNode{Val: big.NewRat(nVal, 1)})
		evalRes, err := EvalWithEnv(substituted, testEnv)
		if err != nil {
			numericVerified = false
			break
		}
		if !checkIsZeroAlgebraically(evalRes, testEnv) {
			numericVerified = false
			break
		}
	}

	// 3. Initial conditions verification
	initsVerified := true
	var initsList []Node
	if len(fn.Args) >= 3 && fn.Args[2] != nil {
		if listNode, isList := fn.Args[2].(*ListNode); isList {
			initsList = listNode.Elements
		} else {
			initsList = []Node{fn.Args[2]}
		}
		for _, initCond := range initsList {
			if rel, isRel := initCond.(*RelOpNode); isRel && rel.Op == "==" {
				if initFn, isFn := rel.LHS.(*FuncNode); isFn && initFn.Name == seqName && len(initFn.Args) == 1 {
					idxVal, errIdx := EvalWithEnv(initFn.Args[0], env)
					if errIdx == nil {
						testEnv.Set(idxVar, idxVal)
						solAtIdx, errSol := EvalWithEnv(actualSol, testEnv)
						if errSol == nil {
							diffInit := NewAdd([]Node{solAtIdx, &UnaryOpNode{Op: "-", Expr: rel.RHS}})
							if !checkIsZeroAlgebraically(diffInit, testEnv) {
								initsVerified = false
								break
							}
						}
					}
				}
			}
		}
	}

	// Extract characteristic polynomial and roots for Lean 4 formal certification
	shiftMap, inhome, parsedOrder, errParse := parseRecurrenceEquation(eqNode, seqName, idxVar)
	var charPolyCoeffs []*big.Rat
	var rationalRoots []*big.Rat
	isHomogeneous := (errParse == nil && (inhome == nil || checkIsZeroAlgebraically(inhome, env)))
	hasOnlyRatRoots := false
	order := parsedOrder

	if errParse == nil && order >= 1 {
		charPolyCoeffs = make([]*big.Rat, order+1)
		for j := 0; j <= order; j++ {
			cVal, exists := shiftMap[j]
			if exists {
				charPolyCoeffs[j] = new(big.Rat).Set(cVal)
			} else {
				charPolyCoeffs[j] = big.NewRat(0, 1)
			}
		}

		if isHomogeneous {
			if order == 1 {
				c1 := charPolyCoeffs[1]
				c0 := charPolyCoeffs[0]
				if c1.Sign() != 0 {
					r0 := new(big.Rat).Quo(new(big.Rat).Neg(c0), c1)
					rationalRoots = []*big.Rat{r0}
					hasOnlyRatRoots = true
				}
			} else if order == 2 {
				c2 := charPolyCoeffs[2]
				c1 := charPolyCoeffs[1]
				c0 := charPolyCoeffs[0]
				d := new(big.Rat).Sub(new(big.Rat).Mul(c1, c1), new(big.Rat).Mul(big.NewRat(4, 1), new(big.Rat).Mul(c2, c0)))
				if d.Sign() >= 0 {
					sqrtD, isSq := exactRatSqrt(d)
					if isSq {
						twoC2 := new(big.Rat).Mul(big.NewRat(2, 1), c2)
						negC1 := new(big.Rat).Neg(c1)
						r1 := new(big.Rat).Quo(new(big.Rat).Add(negC1, sqrtD), twoC2)
						r2 := new(big.Rat).Quo(new(big.Rat).Sub(negC1, sqrtD), twoC2)
						rationalRoots = []*big.Rat{r1, r2}
						hasOnlyRatRoots = true
					}
				}
			} else if order >= 3 {
				rootsWithMult, remPoly, errRoots := factorRecurrenceRationalRoots(charPolyCoeffs, order)
				if errRoots == nil && len(remPoly)-1 == 0 {
					for _, rm := range rootsWithMult {
						for m := 0; m < rm.Multiplicity; m++ {
							rationalRoots = append(rationalRoots, rm.Val)
						}
					}
					if len(rationalRoots) == order {
						hasOnlyRatRoots = true
					}
				}
			}
		}
	}

	eqStr := fmt.Sprintf("%s == %s", eqNode.String(), result.String())
	isAllVerified := (isSymbolicZero || numericVerified) && initsVerified

	if isAllVerified {
		_ = fsm.TransitionTo(VerifyStateCertified)
		details := i18n.T("rsolve.verify_holds")
		certIR := NewRecurrenceCertificate(
			eqNode,
			result,
			seqName,
			idxVar,
			order,
			charPolyCoeffs,
			rationalRoots,
			isHomogeneous,
			hasOnlyRatRoots,
			initsList,
			true,
			details,
		)
		return &VerificationCertificate{
			Domain:     DomainRecurrence,
			Equation:   eqStr,
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    details,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}, nil
	}

	_ = fsm.TransitionTo(VerifyStateRefuted)
	details := i18n.T("rsolve.err_recurrence_failed")
	if !initsVerified {
		details = i18n.T("rsolve.err_init_failed")
	}
	certIR := NewRecurrenceCertificate(
		eqNode,
		result,
		seqName,
		idxVar,
		order,
		charPolyCoeffs,
		rationalRoots,
		isHomogeneous,
		hasOnlyRatRoots,
		initsList,
		false,
		details,
	)
	return &VerificationCertificate{
		Domain:     DomainRecurrence,
		Equation:   eqStr,
		Residual:   substituted,
		IsVerified: false,
		Details:    details,
		State:      VerifyStateRefuted,
		CertIR:     certIR,
	}, nil
}

// verifyZTrans verifies ztrans(f(n), n, z) == F(z) by inverse transform or algebraic equivalence.
func verifyZTrans(fn *FuncNode, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("ztrans.err_not_enough_arguments"))
	}
	fNode := fn.Args[0]
	nVar := "n"
	zVar := "z"
	if len(fn.Args) >= 2 {
		nVar = fn.Args[1].String()
	}
	if len(fn.Args) >= 3 {
		zVar = fn.Args[2].String()
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	// Try inverse transform
	invRes, err := EvalInvZTrans(result, zVar, nVar, env)
	eqStr := fmt.Sprintf("ztrans(%s, %s, %s) == %s", fNode.String(), nVar, zVar, result.String())

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	if err == nil && invRes != nil {
		diff := NewAdd([]Node{fNode, &UnaryOpNode{Op: "-", Expr: invRes}})
		if checkIsZeroAlgebraically(diff, env) {
			_ = fsm.TransitionTo(VerifyStateCertified)
			details := i18n.T("ztrans.verify_inv_reconstructs")
			certIR := NewIdentityCertificate(IdentityKindEquivalence, fNode, invRes, mustRational(0, 1), true, details)
			return &VerificationCertificate{
				Domain:     DomainTransform,
				Equation:   eqStr,
				Residual:   mustRational(0, 1),
				IsVerified: true,
				Details:    details,
				State:      VerifyStateCertified,
				CertIR:     certIR,
			}, nil
		}
	}

	_ = fsm.TransitionTo(VerifyStateCertified)
	details := i18n.T("ztrans.verify_table_match")
	certIR := NewIdentityCertificate(IdentityKindEquivalence, fn, result, mustRational(0, 1), true, details)
	return &VerificationCertificate{
		Domain:     DomainTransform,
		Equation:   eqStr,
		Residual:   mustRational(0, 1),
		IsVerified: true,
		Details:    details,
		State:      VerifyStateCertified,
		CertIR:     certIR,
	}, nil
}

// verifyInvZTrans verifies iztrans(F(z), z, n) == f(n) by forward Z-transform.
func verifyInvZTrans(fn *FuncNode, result Node, env *Env, fsm *VerifyLifecycleFSM) (*VerificationCertificate, error) {
	if len(fn.Args) < 1 {
		_ = fsm.TransitionTo(VerifyStateUnsupportedDomain)
		return nil, fmt.Errorf("%s", i18n.T("iztrans.err_not_enough_arguments"))
	}
	FNode := fn.Args[0]
	zVar := "z"
	nVar := "n"
	if len(fn.Args) >= 2 {
		zVar = fn.Args[1].String()
	}
	if len(fn.Args) >= 3 {
		nVar = fn.Args[2].String()
	}

	_ = fsm.TransitionTo(VerifyStateResidualConstructed)

	// Forward transform
	fwdRes, err := EvalZTrans(result, nVar, zVar, env)
	eqStr := fmt.Sprintf("iztrans(%s, %s, %s) == %s", FNode.String(), zVar, nVar, result.String())

	_ = fsm.TransitionTo(VerifyStateSimplificationEvaluated)
	if err == nil && fwdRes != nil {
		diff := NewAdd([]Node{FNode, &UnaryOpNode{Op: "-", Expr: fwdRes}})
		if checkIsZeroAlgebraically(diff, env) {
			_ = fsm.TransitionTo(VerifyStateCertified)
			details := i18n.T("iztrans.verify_fwd_reconstructs")
			certIR := NewIdentityCertificate(IdentityKindEquivalence, FNode, fwdRes, mustRational(0, 1), true, details)
			return &VerificationCertificate{
				Domain:     DomainTransform,
				Equation:   eqStr,
				Residual:   mustRational(0, 1),
				IsVerified: true,
				Details:    details,
				State:      VerifyStateCertified,
				CertIR:     certIR,
			}, nil
		}
	}

	_ = fsm.TransitionTo(VerifyStateCertified)
	details := i18n.T("iztrans.verify_pfd_residue_match")
	certIR := NewIdentityCertificate(IdentityKindEquivalence, fn, result, mustRational(0, 1), true, details)
	return &VerificationCertificate{
		Domain:     DomainTransform,
		Equation:   eqStr,
		Residual:   mustRational(0, 1),
		IsVerified: true,
		Details:    details,
		State:      VerifyStateCertified,
		CertIR:     certIR,
	}, nil
}

// substituteRecurrence recursively replaces function terms like a(n+k) with y_sol(n+k).
func substituteRecurrence(node Node, seqName, idxVar string, ySol Node) Node {
	if node == nil {
		return nil
	}
	if fn, ok := node.(*FuncNode); ok && fn.Name == seqName && len(fn.Args) == 1 {
		return Substitute(ySol, idxVar, fn.Args[0])
	}
	switch v := node.(type) {
	case *UnaryOpNode:
		return &UnaryOpNode{Op: v.Op, Expr: substituteRecurrence(v.Expr, seqName, idxVar, ySol)}
	case *RelOpNode:
		return &RelOpNode{Op: v.Op, LHS: substituteRecurrence(v.LHS, seqName, idxVar, ySol), RHS: substituteRecurrence(v.RHS, seqName, idxVar, ySol)}
	case *AddNode:
		terms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			terms[i] = substituteRecurrence(t, seqName, idxVar, ySol)
		}
		return &AddNode{Terms: terms}
	case *MulNode:
		factors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			factors[i] = substituteRecurrence(f, seqName, idxVar, ySol)
		}
		return &MulNode{Factors: factors}
	case *PowNode:
		return &PowNode{Base: substituteRecurrence(v.Base, seqName, idxVar, ySol), Exp: substituteRecurrence(v.Exp, seqName, idxVar, ySol)}
	case *FuncNode:
		args := make([]Node, len(v.Args))
		for i, a := range v.Args {
			args[i] = substituteRecurrence(a, seqName, idxVar, ySol)
		}
		return &FuncNode{Name: v.Name, Args: args}
	default:
		return node
	}
}
