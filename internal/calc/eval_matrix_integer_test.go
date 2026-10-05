package calc

import (
	"math/big"
	"testing"
)

func TestComputeHNF(t *testing.T) {
	// A = [[2, 4], [0, 3]]
	// H should have diagonal [2, 3] and top-right entry 4 mod 3 = 1 => [[2, 1], [0, 3]]
	mat := [][]big.Int{
		{*big.NewInt(2), *big.NewInt(4)},
		{*big.NewInt(0), *big.NewInt(3)},
	}

	H, U, err := ComputeHNF(mat)
	if err != nil {
		t.Fatalf("ComputeHNF failed: %v", err)
	}

	if H[0][0].Int64() != 2 || H[0][1].Int64() != 1 || H[1][0].Int64() != 0 || H[1][1].Int64() != 3 {
		t.Errorf("expected H = [[2, 1], [0, 3]], got [[%s, %s], [%s, %s]]",
			H[0][0].String(), H[0][1].String(), H[1][0].String(), H[1][1].String())
	}

	// Verify U * A == H
	// U = [[1, -1], [0, 1]]
	// [[1, -1], [0, 1]] * [[2, 4], [0, 3]] = [[2, 4 - 3], [0, 3]] = [[2, 1], [0, 3]]
	detU := new(big.Int).Sub(
		new(big.Int).Mul(&U[0][0], &U[1][1]),
		new(big.Int).Mul(&U[0][1], &U[1][0]),
	)
	if new(big.Int).Abs(detU).Int64() != 1 {
		t.Errorf("expected |det(U)| == 1, got %s", detU.String())
	}
}

func TestComputeSNF(t *testing.T) {
	// A = [[2, 4], [4, 2]]
	// det(A) = 4 - 16 = -12
	// SNF invariant factors: gcd of entries = 2, determinant magnitude = 12 => diag(2, 6)
	mat := [][]big.Int{
		{*big.NewInt(2), *big.NewInt(4)},
		{*big.NewInt(4), *big.NewInt(2)},
	}

	D, U, V, err := ComputeSNF(mat)
	if err != nil {
		t.Fatalf("ComputeSNF failed: %v", err)
	}

	if D[0][0].Int64() != 2 || D[0][1].Int64() != 0 || D[1][0].Int64() != 0 || D[1][1].Int64() != 6 {
		t.Errorf("expected D = [[2, 0], [0, 6]], got [[%s, %s], [%s, %s]]",
			D[0][0].String(), D[0][1].String(), D[1][0].String(), D[1][1].String())
	}

	// Verify |det(U)| == 1 and |det(V)| == 1
	detU := new(big.Int).Sub(
		new(big.Int).Mul(&U[0][0], &U[1][1]),
		new(big.Int).Mul(&U[0][1], &U[1][0]),
	)
	if new(big.Int).Abs(detU).Int64() != 1 {
		t.Errorf("expected |det(U)| == 1, got %s", detU.String())
	}

	detV := new(big.Int).Sub(
		new(big.Int).Mul(&V[0][0], &V[1][1]),
		new(big.Int).Mul(&V[0][1], &V[1][0]),
	)
	if new(big.Int).Abs(detV).Int64() != 1 {
		t.Errorf("expected |det(V)| == 1, got %s", detV.String())
	}

	// Verify U * A * V == D
	// 1. UA = U * mat
	UA := make([][]big.Int, 2)
	for r := 0; r < 2; r++ {
		UA[r] = make([]big.Int, 2)
		for c := 0; c < 2; c++ {
			term0 := new(big.Int).Mul(&U[r][0], &mat[0][c])
			term1 := new(big.Int).Mul(&U[r][1], &mat[1][c])
			UA[r][c].Add(term0, term1)
		}
	}
	// 2. UAV = UA * V
	UAV := make([][]big.Int, 2)
	for r := 0; r < 2; r++ {
		UAV[r] = make([]big.Int, 2)
		for c := 0; c < 2; c++ {
			term0 := new(big.Int).Mul(&UA[r][0], &V[0][c])
			term1 := new(big.Int).Mul(&UA[r][1], &V[1][c])
			UAV[r][c].Add(term0, term1)
		}
	}

	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			if UAV[r][c].Cmp(&D[r][c]) != 0 {
				t.Errorf("mismatch at (%d, %d): UAV = %s, D = %s", r, c, UAV[r][c].String(), D[r][c].String())
			}
		}
	}
}

func TestEvalAbelianGroupStructure(t *testing.T) {
	// A = [[2, 4], [4, 2]] => Z_2 + Z_6
	res, err := EvalString("abelian_group_structure([[2, 4], [4, 2]])")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Format(res) != "Z_2 + Z_6" {
		t.Errorf("expected Z_2 + Z_6, got %s", Format(res))
	}

	// Matrix with free rank: [[2, 0], [0, 0]] => Z_2 + Z
	resFree, err := EvalString("abelian_group_structure([[2, 0], [0, 0]])")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Format(resFree) != "Z_2 + Z" {
		t.Errorf("expected Z_2 + Z, got %s", Format(resFree))
	}
}

func TestSmithNormalFormFSM(t *testing.T) {
	fsm := NewSmithNormalFormFSM()
	if fsm.CurrentState() != SNFStateInit {
		t.Errorf("expected INIT, got %s", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(SNFStateSelectPivot); err != nil {
		t.Errorf("transition to SELECT_PIVOT failed: %v", err)
	}
	if err := fsm.TransitionTo(SNFStateEliminateRowCol); err != nil {
		t.Errorf("transition to ELIMINATE_ROW_COL failed: %v", err)
	}
	if err := fsm.TransitionTo(SNFStateCheckDivisibility); err != nil {
		t.Errorf("transition to CHECK_DIVISIBILITY failed: %v", err)
	}
	if err := fsm.TransitionTo(SNFStateNextBlock); err != nil {
		t.Errorf("transition to NEXT_BLOCK failed: %v", err)
	}
	if err := fsm.TransitionTo(SNFStateSuccess); err != nil {
		t.Errorf("transition to SUCCESS failed: %v", err)
	}

	// Invalid transition from SUCCESS to INIT
	if err := fsm.TransitionTo(SNFStateInit); err == nil {
		t.Errorf("expected error for invalid transition SUCCESS -> INIT, got nil")
	}
}

func TestRectangularAndDegenerateMatrices(t *testing.T) {
	// Rectangular 2x3 matrix: [[1, 2, 3], [4, 5, 6]]
	// SNF should be [[1, 0, 0], [0, 3, 0]]
	resSNF, err := EvalString("snf([[1, 2, 3], [4, 5, 6]])")
	if err != nil {
		t.Fatalf("snf on 2x3 failed: %v", err)
	}
	if Format(resSNF) != "[[1, 0, 0], [0, 3, 0]]" {
		t.Errorf("expected [[1, 0, 0], [0, 3, 0]], got %s", Format(resSNF))
	}

	// Invariant factors of 2x3: [1, 3]
	resInv, err := EvalString("invariant_factors([[1, 2, 3], [4, 5, 6]])")
	if err != nil {
		t.Fatalf("invariant_factors on 2x3 failed: %v", err)
	}
	if Format(resInv) != "[1, 3]" {
		t.Errorf("expected [1, 3], got %s", Format(resInv))
	}

	// 3x2 matrix: [[1, 4], [2, 5], [3, 6]]
	// SNF should be [[1, 0], [0, 3], [0, 0]]
	resSNF32, err := EvalString("snf([[1, 4], [2, 5], [3, 6]])")
	if err != nil {
		t.Fatalf("snf on 3x2 failed: %v", err)
	}
	if Format(resSNF32) != "[[1, 0], [0, 3], [0, 0]]" {
		t.Errorf("expected [[1, 0], [0, 3], [0, 0]], got %s", Format(resSNF32))
	}

	// Zero matrix 2x2: [[0, 0], [0, 0]]
	resZero, err := EvalString("snf([[0, 0], [0, 0]])")
	if err != nil {
		t.Fatalf("snf on zero matrix failed: %v", err)
	}
	if Format(resZero) != "[[0, 0], [0, 0]]" {
		t.Errorf("expected [[0, 0], [0, 0]], got %s", Format(resZero))
	}

	// HNF on rectangular 2x3 matrix: [[1, 2, 3], [4, 5, 6]]
	resHNF, err := EvalString("hnf([[1, 2, 3], [4, 5, 6]])")
	if err != nil {
		t.Fatalf("hnf on 2x3 failed: %v", err)
	}
	if Format(resHNF) != "[[1, 2, 3], [0, 3, 6]]" {
		t.Errorf("expected [[1, 2, 3], [0, 3, 6]], got %s", Format(resHNF))
	}
}

