package calc

import (
	"testing"
)

func TestVerifyDiscrete_SpanningTrees(t *testing.T) {
	// K3 graph
	fn := &FuncNode{
		Name: "spanning_trees",
		Args: []Node{
			&MatrixNode{
				Rows: 3,
				Cols: 3,
				Data: [][]Node{
					{mustRational(0, 1), mustRational(1, 1), mustRational(1, 1)},
					{mustRational(1, 1), mustRational(0, 1), mustRational(1, 1)},
					{mustRational(1, 1), mustRational(1, 1), mustRational(0, 1)},
				},
			},
		},
	}
	result := mustRational(3, 1)
	env := NewEnv()

	cert, err := VerifyComputation(fn, result, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cert.IsVerified {
		t.Errorf("expected verified certificate, got unverified: %s", cert.Details)
	}
	if cert.Domain != DomainGraph {
		t.Errorf("expected DomainGraph, got %v", cert.Domain)
	}
	if _, ok := cert.ToCertificateIR().(*KirchhoffCertificate); !ok {
		t.Errorf("expected *KirchhoffCertificate, got %T", cert.ToCertificateIR())
	}
}

func TestVerifyDiscrete_NashZeroSum(t *testing.T) {
	// Matching pennies
	fn := &FuncNode{
		Name: "nash_zero_sum",
		Args: []Node{
			&MatrixNode{
				Data: [][]Node{
					{mustRational(1, 1), mustRational(-1, 1)},
					{mustRational(-1, 1), mustRational(1, 1)},
				},
			},
		},
	}
	result := &ListNode{
		Elements: []Node{
			mustRational(0, 1),
			&ListNode{Elements: []Node{mustRational(1, 2), mustRational(1, 2)}},
			&ListNode{Elements: []Node{mustRational(1, 2), mustRational(1, 2)}},
		},
	}
	env := NewEnv()

	cert, err := VerifyComputation(fn, result, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cert.IsVerified {
		t.Errorf("expected verified certificate, got unverified: %s", cert.Details)
	}
	if cert.Domain != DomainGame {
		t.Errorf("expected DomainGame, got %v", cert.Domain)
	}
	if _, ok := cert.ToCertificateIR().(*MinimaxCertificate); !ok {
		t.Errorf("expected *MinimaxCertificate, got %T", cert.ToCertificateIR())
	}
}

func TestVerifyDiscrete_MarkovStationary(t *testing.T) {
	// 2-state Markov chain
	fn := &FuncNode{
		Name: "markov_stationary",
		Args: []Node{
			&MatrixNode{
				Data: [][]Node{
					{mustRational(1, 2), mustRational(1, 2)},
					{mustRational(1, 3), mustRational(2, 3)},
				},
			},
		},
	}
	result := &ListNode{
		Elements: []Node{mustRational(2, 5), mustRational(3, 5)},
	}
	env := NewEnv()

	cert, err := VerifyComputation(fn, result, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cert.IsVerified {
		t.Errorf("expected verified certificate, got unverified: %s", cert.Details)
	}
	if cert.Domain != DomainProbability {
		t.Errorf("expected DomainProbability, got %v", cert.Domain)
	}
	if _, ok := cert.ToCertificateIR().(*MarkovStationaryCertificate); !ok {
		t.Errorf("expected *MarkovStationaryCertificate, got %T", cert.ToCertificateIR())
	}
}
