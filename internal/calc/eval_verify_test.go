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
