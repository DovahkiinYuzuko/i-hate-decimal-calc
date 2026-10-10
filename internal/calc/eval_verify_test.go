package calc

import (
	"strings"
	"testing"
)

func TestVerifyFSM_Transitions(t *testing.T) {
	fsm := NewVerifyLifecycleFSM()
	if fsm.CurrentState() != VerifyStateIdle {
		t.Fatalf("expected state Idle, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(VerifyStateTargetClassified); err != nil {
		t.Fatalf("failed to transition to TargetClassified: %v", err)
	}
	if fsm.CurrentState() != VerifyStateTargetClassified {
		t.Fatalf("expected state TargetClassified, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(VerifyStateResidualConstructed); err != nil {
		t.Fatalf("failed to transition to ResidualConstructed: %v", err)
	}
	if fsm.CurrentState() != VerifyStateResidualConstructed {
		t.Fatalf("expected state ResidualConstructed, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(VerifyStateSimplificationEvaluated); err != nil {
		t.Fatalf("failed to transition to SimplificationEvaluated: %v", err)
	}
	if fsm.CurrentState() != VerifyStateSimplificationEvaluated {
		t.Fatalf("expected state SimplificationEvaluated, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(VerifyStateCertified); err != nil {
		t.Fatalf("failed to transition to Certified: %v", err)
	}
	if fsm.CurrentState() != VerifyStateCertified {
		t.Fatalf("expected state Certified, got %v", fsm.CurrentState())
	}
}

func TestVerify_IndefiniteIntegral(t *testing.T) {
	env := NewEnv()
	// integrate(x^2, x)
	expr, err := Parse("integrate(x^2, x)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, evaled, env)
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected verified certificate, got: %s", cert.String())
	}
	if cert.Domain != DomainIntegral {
		t.Errorf("unexpected domain: %s", cert.Domain)
	}
}

func TestVerify_Factor(t *testing.T) {
	env := NewEnv()
	// factor(x^2 - 1)
	expr, err := Parse("factor(x^2 - 1)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, evaled, env)
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected verified certificate, got: %s", cert.String())
	}
	if cert.Domain != DomainFactor {
		t.Errorf("unexpected domain: %s", cert.Domain)
	}
}

func TestVerify_MatrixInverse(t *testing.T) {
	env := NewEnv()
	// inv([[1, 2], [3, 4]])
	expr, err := Parse("inv([[1, 2], [3, 4]])")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, evaled, env)
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected verified certificate, got: %s", cert.String())
	}
	if cert.Domain != DomainMatrix {
		t.Errorf("unexpected domain: %s", cert.Domain)
	}
}

func TestVerify_MatrixDecompositions(t *testing.T) {
	env := NewEnv()

	// LU
	exprLU, err := Parse("lu([[2, 1], [6, 8]])")
	if err != nil {
		t.Fatalf("parse LU failed: %v", err)
	}
	evalLU, err := EvalWithEnv(exprLU, env)
	if err != nil {
		t.Fatalf("eval LU failed: %v", err)
	}
	certLU, err := VerifyComputation(exprLU, evalLU, env)
	if err != nil {
		t.Fatalf("verification LU error: %v", err)
	}
	if !certLU.IsVerified {
		t.Fatalf("expected verified LU certificate, got: %s", certLU.String())
	}

	// QR
	exprQR, err := Parse("qr([[1, 0], [1, 1]])")
	if err != nil {
		t.Fatalf("parse QR failed: %v", err)
	}
	evalQR, err := EvalWithEnv(exprQR, env)
	if err != nil {
		t.Fatalf("eval QR failed: %v", err)
	}
	certQR, err := VerifyComputation(exprQR, evalQR, env)
	if err != nil {
		t.Fatalf("verification QR error: %v", err)
	}
	if !certQR.IsVerified {
		t.Fatalf("expected verified QR certificate, got: %s", certQR.String())
	}
}

func TestVerify_GeneralEquivalence(t *testing.T) {
	env := NewEnv()
	// (x + 1)^2 == x^2 + 2*x + 1
	expr, err := Parse("(x + 1)^2 == x^2 + 2*x + 1")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, evaled, env)
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected verified certificate for algebraic identity, got: %s", cert.String())
	}
}

func TestVerify_Refuted(t *testing.T) {
	env := NewEnv()
	// 1 == 2
	expr, err := Parse("1 == 2")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, evaled, env)
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}
	if cert.IsVerified {
		t.Fatalf("expected false identity to NOT be verified, got: %s", cert.String())
	}
	if cert.State != VerifyStateRefuted {
		t.Errorf("expected State Refuted, got: %v", cert.State)
	}
}

func TestVerify_BuiltinFunction(t *testing.T) {
	env := NewEnv()
	// verify(x^2 - 1 == (x - 1)*(x + 1))
	expr, err := Parse("verify(x^2 - 1 == (x - 1)*(x + 1))")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	str := evaled.String()
	if !strings.Contains(str, "VERIFIED") {
		t.Fatalf("expected VERIFIED certificate, got %v", evaled)
	}

	// verify(1 == 2) -> FAILED
	exprFalse, err := Parse("verify(1 == 2)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evalFalse, err := EvalWithEnv(exprFalse, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	strFalse := evalFalse.String()
	if !strings.Contains(strFalse, "FAILED") {
		t.Fatalf("expected FAILED certificate, got %v", evalFalse)
	}
}

func TestVerify_SNF_and_HNF(t *testing.T) {
	env := NewEnv()

	// 1. SNF verification: snf([[2, 4], [4, 2]])
	snfExpr, err := Parse("snf([[2, 4], [4, 2]])")
	if err != nil {
		t.Fatalf("parse snf failed: %v", err)
	}
	snfEvaled, err := EvalWithEnv(snfExpr, env)
	if err != nil {
		t.Fatalf("eval snf failed: %v", err)
	}
	snfCert, err := VerifyComputation(snfExpr, snfEvaled, env)
	if err != nil {
		t.Fatalf("verify snf failed: %v", err)
	}
	if !snfCert.IsVerified {
		t.Fatalf("expected SNF to be verified, got: %s", snfCert.String())
	}
	if snfCert.Domain != DomainSNF {
		t.Errorf("expected DomainSNF, got: %s", snfCert.Domain)
	}
	if snfCert.CertIR == nil {
		t.Fatalf("expected CertIR to be populated")
	}
	if _, ok := snfCert.CertIR.(*SmithNormalFormCertificate); !ok {
		t.Errorf("expected SmithNormalFormCertificate, got %T", snfCert.CertIR)
	}

	// 2. HNF verification: hnf([[2, 4], [4, 2]])
	hnfExpr, err := Parse("hnf([[2, 4], [4, 2]])")
	if err != nil {
		t.Fatalf("parse hnf failed: %v", err)
	}
	hnfEvaled, err := EvalWithEnv(hnfExpr, env)
	if err != nil {
		t.Fatalf("eval hnf failed: %v", err)
	}
	hnfCert, err := VerifyComputation(hnfExpr, hnfEvaled, env)
	if err != nil {
		t.Fatalf("verify hnf failed: %v", err)
	}
	if !hnfCert.IsVerified {
		t.Fatalf("expected HNF to be verified, got: %s", hnfCert.String())
	}
	if hnfCert.Domain != DomainHNF {
		t.Errorf("expected DomainHNF, got: %s", hnfCert.Domain)
	}
	if hnfCert.CertIR == nil {
		t.Fatalf("expected CertIR to be populated")
	}
	if _, ok := hnfCert.CertIR.(*HermiteNormalFormCertificate); !ok {
		t.Errorf("expected HermiteNormalFormCertificate, got %T", hnfCert.CertIR)
	}
}

func TestVerificationCertificate_ToProofTrace(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("integrate(x^2, x)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, evaled, env)
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}

	trace, err := cert.ToProofTrace()
	if err != nil {
		t.Fatalf("failed to convert VerificationCertificate to ProofTrace: %v", err)
	}
	if trace == nil {
		t.Fatalf("expected non-nil ProofTrace")
	}
	if trace.StepCount() == 0 {
		t.Fatalf("expected at least 1 step in ProofTrace")
	}
}

func TestVerificationCertificate_VerifyTraceAndExplain(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("factor(x^2 - 1)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, evaled, env)
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}

	ok, err := cert.VerifyTrace(env)
	if err != nil {
		t.Fatalf("failed to verify trace: %v", err)
	}
	if !ok {
		t.Fatalf("expected trace verification to succeed")
	}

	explain := cert.RenderTraceExplain("ja")
	if explain == "" {
		t.Fatalf("expected non-empty trace explain")
	}
}

func TestVerifySolve_IncompleteRootsRejected(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("solve(x^2 - 1 == 0, x)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// 1. Partial root list [1] (missing -1)
	partialRoots, err := Parse("[1]")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	certPartial, err := VerifyComputation(expr, partialRoots, env)
	if err != nil {
		t.Fatalf("verification returned unexpected error: %v", err)
	}
	if certPartial.IsVerified {
		t.Fatalf("expected partial roots [1] for x^2 - 1 to be REJECTED, but got IsVerified=true")
	}

	// 2. Empty root list []
	emptyRoots, err := Parse("[]")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	certEmpty, err := VerifyComputation(expr, emptyRoots, env)
	if err != nil {
		t.Fatalf("verification returned unexpected error: %v", err)
	}
	if certEmpty.IsVerified {
		t.Fatalf("expected empty roots [] for x^2 - 1 to be REJECTED, but got IsVerified=true")
	}
}

func TestVerifyMinimalPolynomial_ReducibleOrNonMonicRejected(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("minimal_polynomial(sqrt(2))")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// 1. Non-monic polynomial 2*x^2 - 4 (leading coeff is 2 != 1)
	nonMonic, err := Parse("2*x^2 - 4")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	certNonMonic, err := VerifyComputation(expr, nonMonic, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if certNonMonic.IsVerified {
		t.Fatalf("expected non-monic polynomial to be REJECTED, but got IsVerified=true")
	}

	// 2. Reducible polynomial x^4 - 4 = (x^2 - 2)(x^2 + 2) (degree is not minimal)
	reducible, err := Parse("x^4 - 4")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	certReducible, err := VerifyComputation(expr, reducible, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if certReducible.IsVerified {
		t.Fatalf("expected reducible polynomial x^4 - 4 to be REJECTED, but got IsVerified=true")
	}
}

func TestVerifyEllipticAdd_WrongPointOnCurveRejected(t *testing.T) {
	env := NewEnv()
	// Curve: y^2 = x^3 - x (a = -1, b = 0)
	// Point addition: ec_add(-1, 0, [0, 0], [1, 0])
	// In the group law on y^2 = x^3 - x, (0, 0) + (1, 0) = (-1, 0).
	expr, err := Parse("ec_add(-1, 0, [0, 0], [1, 0])")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Adversarial forged point: [0, 0]
	// Lies on the curve 0^2 == 0^3 - 0, but is NOT the sum of (0,0) and (1,0)!
	forgedPoint, err := Parse("[0, 0]")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cert, err := VerifyComputation(expr, forgedPoint, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert.IsVerified {
		t.Fatalf("expected forged point on curve to be REJECTED by group law, but got IsVerified=true")
	}
}

func TestVerifySNF_ForgedResultMatricesRejected(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("snf([[1, 2], [3, 4]])")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Forged result where U * A * V != D (e.g. Identity D = [[99, 0], [0, 99]])
	forgedResult, err := Parse("[[[99, 0], [0, 99]], [[1, 0], [0, 1]], [[1, 0], [0, 1]]]")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cert, err := VerifyComputation(expr, forgedResult, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert.IsVerified {
		t.Fatalf("expected forged SNF result to be REJECTED, but got IsVerified=true")
	}
}

func TestVerifyHNF_ForgedResultMatricesRejected(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("hnf([[1, 2], [3, 4]])")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Forged result where U * A != H
	forgedResult, err := Parse("[[[99, 99], [0, 99]], [[1, 0], [0, 1]]]")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cert, err := VerifyComputation(expr, forgedResult, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert.IsVerified {
		t.Fatalf("expected forged HNF result to be REJECTED, but got IsVerified=true")
	}
}


