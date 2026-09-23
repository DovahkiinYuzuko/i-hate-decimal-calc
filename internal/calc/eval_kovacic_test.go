package calc

import (
	"strings"
	"testing"
)

func TestKovacic_NormalFormReduction(t *testing.T) {
	// y'' + 4*x*y' + (4*x^2 + 2)*y = 0
	// P(x) = 4*x, Q(x) = 4*x^2 + 2
	// r(x) = Q - P^2/4 - P'/2 = (4*x^2 + 2) - 4*x^2 - 2 = 0
	pNode, err := Parse("4*x")
	if err != nil {
		t.Fatalf("failed to parse P(x): %v", err)
	}
	qNode, err := Parse("4*x^2 + 2")
	if err != nil {
		t.Fatalf("failed to parse Q(x): %v", err)
	}

	rNode, xiNode, err := NormalizeSecondOrderODE(pNode, qNode, "x")
	if err != nil {
		t.Fatalf("NormalizeSecondOrderODE failed: %v", err)
	}

	// r(x) should simplify to 0
	if rNode.String() != "0" {
		t.Errorf("expected r(x) == 0, got %s", rNode.String())
	}

	// xi(x) should be exp(-x^2)
	xiStr := xiNode.String()
	if !strings.Contains(xiStr, "exp") || !strings.Contains(xiStr, "x") {
		t.Errorf("expected xi(x) containing exp(-x^2), got %s", xiStr)
	}
}

func TestKovacic_Case1_Gaussian(t *testing.T) {
	// y'' + 4*x*y' + (4*x^2 + 2)*y = 0
	// Has solution basis y1 = exp(-x^2), y2 = x*exp(-x^2)
	pNode, _ := Parse("4*x")
	qNode, _ := Parse("4*x^2 + 2")

	res, err := SolveKovacicExact(pNode, qNode, "x")
	if err != nil {
		t.Fatalf("SolveKovacicExact failed: %v", err)
	}

	if !res.IsLiouvillian {
		t.Fatalf("expected Liouvillian solution, but got false. Proof: %s", res.ProofMessage)
	}
	if res.Case != 1 {
		t.Errorf("expected Case 1, got %d", res.Case)
	}
	if len(res.Basis) != 2 {
		t.Fatalf("expected 2 basis solutions, got %d", len(res.Basis))
	}

	solStr := res.GeneralSol.String()
	if !strings.Contains(solStr, "exp") {
		t.Errorf("expected exponential term in general solution, got: %s", solStr)
	}
	if !strings.Contains(solStr, "C_1") || !strings.Contains(solStr, "C_2") {
		t.Errorf("expected integration constants C_1, C_2 in general solution, got: %s", solStr)
	}
}

func TestKovacic_Case4_Airy(t *testing.T) {
	// Airy equation: y'' - x*y = 0 => P(x) = 0, Q(x) = -x => r(x) = -x
	// Non-Liouvillian: proved that Galois group is SL(2, C)
	pNode := mustRational(0, 1)
	qNode, _ := Parse("-x")

	res, err := SolveKovacicExact(pNode, qNode, "x")
	if err != nil {
		t.Fatalf("SolveKovacicExact failed unexpectedly: %v", err)
	}

	if res.IsLiouvillian {
		t.Fatalf("Airy equation should have no Liouvillian solutions, but got IsLiouvillian=true")
	}
	if res.Case != 4 {
		t.Errorf("expected Case 4 (non-Liouvillian proof), got %d", res.Case)
	}
	if !strings.Contains(res.ProofMessage, "Differential Galois") && !strings.Contains(res.ProofMessage, "ガロア") {
		t.Errorf("expected proof message mentioning differential Galois group, got: %s", res.ProofMessage)
	}
}

func TestKovacic_FSM_Transitions(t *testing.T) {
	// 1. Fast-path: Gaussian equation (r(x) constant 0)
	fsm1 := NewKovacicLifecycleFSM()
	pNode1, _ := Parse("4*x")
	qNode1, _ := Parse("4*x^2 + 2")

	res1, err := SolveKovacicExactWithFSM(pNode1, qNode1, "x", fsm1)
	if err != nil {
		t.Fatalf("SolveKovacicExactWithFSM (Gaussian) failed: %v", err)
	}
	if res1.FSM.CurrentState() != KovacicStateSolutionConstructed {
		t.Errorf("expected final state %s, got %s", KovacicStateSolutionConstructed, res1.FSM.CurrentState())
	}
	history1 := res1.FSM.History()
	if len(history1) != 3 {
		t.Errorf("expected 3 state transitions for constant-r fast path, got %d: %v", len(history1), history1)
	}

	// 2. Full pipeline: Airy equation y'' - x*y = 0 (r(x) = -x)
	fsm2 := NewKovacicLifecycleFSM()
	pNode2 := mustRational(0, 1)
	qNode2, _ := Parse("-x")

	res2, err := SolveKovacicExactWithFSM(pNode2, qNode2, "x", fsm2)
	if err != nil {
		t.Fatalf("SolveKovacicExactWithFSM (Airy) failed: %v", err)
	}
	if res2.FSM.CurrentState() != KovacicStateNonLiouvillianProven {
		t.Errorf("expected final state %s, got %s", KovacicStateNonLiouvillianProven, res2.FSM.CurrentState())
	}
	history2 := res2.FSM.History()
	if len(history2) != 6 {
		t.Errorf("expected 6 state transitions for full non-liouvillian proof, got %d: %v", len(history2), history2)
	}
}

func TestEvalDSolve_KovacicIntegration(t *testing.T) {
	env := NewEnv()
	// diff(y, x, 2) + 4*x*diff(y, x) + (4*x^2 + 2)*y = 0
	eq, err := Parse("diff(y, x, 2) + 4*x*diff(y, x) + (4*x^2 + 2)*y = 0")
	if err != nil {
		t.Fatalf("failed to parse ODE equation: %v", err)
	}

	sol, err := EvalDSolve(eq, "y", "x", env)
	if err != nil {
		t.Fatalf("EvalDSolve with Kovacic failed: %v", err)
	}

	rel, ok := sol.(*RelOpNode)
	if !ok {
		t.Fatalf("expected RelOpNode, got %T", sol)
	}
	if rel.LHS.String() != "y" {
		t.Errorf("expected LHS == y, got %s", rel.LHS.String())
	}

	rhsStr := rel.RHS.String()
	if !strings.Contains(rhsStr, "exp") {
		t.Errorf("expected exp in RHS, got %s", rhsStr)
	}
}
