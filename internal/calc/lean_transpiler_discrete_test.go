package calc

import (
	"strings"
	"testing"
)

func TestTranspileCertificateIRToLean_Kirchhoff(t *testing.T) {
	// K3 graph Laplacian minor:
	// L = [[2, -1, -1], [-1, 2, -1], [-1, -1, 2]]
	// L00 = [[2, -1], [-1, 2]] -> det = 4 - 1 = 3
	l00 := &MatrixNode{
		Data: [][]Node{
			{mustRational(2, 1), mustRational(-1, 1)},
			{mustRational(-1, 1), mustRational(2, 1)},
		},
	}
	trees := mustRational(3, 1)
	cert := NewKirchhoffCertificate(nil, l00, trees, mustRational(0, 1), true, "Kirchhoff K3")

	code, err := TranspileCertificateIRToLean(cert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "Matrix.det") || !strings.Contains(code, "= 3") {
		t.Errorf("unexpected theorem code: %s", code)
	}
	if !strings.Contains(code, "by norm_num") {
		t.Errorf("expected norm_num tactic, got %s", code)
	}

	imports := DetermineRequiredImports(cert)
	importsStr := strings.Join(imports, " ")
	if !strings.Contains(importsStr, "Mathlib.LinearAlgebra.Matrix.Determinant.Basic") {
		t.Errorf("expected Determinant import, got %s", importsStr)
	}
}

func TestTranspileCertificateIRToLean_Minimax(t *testing.T) {
	// Rock-Paper-Scissors: 3x3
	payoff := &MatrixNode{
		Data: [][]Node{
			{mustRational(0, 1), mustRational(-1, 1), mustRational(1, 1)},
			{mustRational(1, 1), mustRational(0, 1), mustRational(-1, 1)},
			{mustRational(-1, 1), mustRational(1, 1), mustRational(0, 1)},
		},
	}
	p1 := &ListNode{Elements: []Node{mustRational(1, 3), mustRational(1, 3), mustRational(1, 3)}}
	p2 := &ListNode{Elements: []Node{mustRational(1, 3), mustRational(1, 3), mustRational(1, 3)}}
	val := mustRational(0, 1)

	cert := NewMinimaxCertificate(payoff, p1, p2, val, mustRational(0, 1), true, "RPS Minimax")

	code, err := TranspileCertificateIRToLean(cert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "Matrix.vecMul") || !strings.Contains(code, "Matrix.mulVec") {
		t.Errorf("unexpected minimax theorem code: %s", code)
	}
	if !strings.Contains(code, "fin_cases") || !strings.Contains(code, "norm_num") {
		t.Errorf("expected fin_cases and norm_num tactic, got %s", code)
	}
}

func TestTranspileCertificateIRToLean_Markov(t *testing.T) {
	// 2-state Markov chain: P = [[1/2, 1/2], [1/3, 2/3]], pi = [2/5, 3/5]
	trans := &MatrixNode{
		Data: [][]Node{
			{mustRational(1, 2), mustRational(1, 2)},
			{mustRational(1, 3), mustRational(2, 3)},
		},
	}
	dist := &ListNode{Elements: []Node{mustRational(2, 5), mustRational(3, 5)}}

	cert := NewMarkovStationaryCertificate(trans, dist, mustRational(0, 1), true, "Markov 2-state")

	code, err := TranspileCertificateIRToLean(cert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "Matrix.vecMul") {
		t.Errorf("unexpected markov theorem code: %s", code)
	}
	if !strings.Contains(code, "by ext i <;> fin_cases i <;> norm_num") {
		t.Errorf("expected ext/fin_cases/norm_num tactic, got %s", code)
	}
}
