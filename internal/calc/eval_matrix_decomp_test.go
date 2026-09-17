package calc

import (
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// Helper to evaluate an expression string into an AST Node
func evalDecompExpr(t *testing.T, exprStr string) ast.Node {
	t.Helper()
	parsed, err := Parse(exprStr)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", exprStr, err)
	}
	node, err := EvalWithEnv(parsed, NewEnv())
	if err != nil {
		t.Fatalf("EvalWithEnv(%q) failed: %v", exprStr, err)
	}
	return node
}

func evalDecompExprErr(exprStr string) (ast.Node, error) {
	parsed, err := Parse(exprStr)
	if err != nil {
		return nil, err
	}
	return EvalWithEnv(parsed, NewEnv())
}

// Helper to check exact equality between two MatrixNodes
func assertMatrixEqual(t *testing.T, name string, got, want *ast.MatrixNode) {
	t.Helper()
	if got.Rows != want.Rows || got.Cols != want.Cols {
		t.Fatalf("%s dimension mismatch: got %dx%d, want %dx%d", name, got.Rows, got.Cols, want.Rows, want.Cols)
	}
	for r := 0; r < got.Rows; r++ {
		for c := 0; c < got.Cols; c++ {
			diff, err := simplifyAdd([]ast.Node{got.Data[r][c], mustNegate(want.Data[r][c])})
			if err != nil || !isZero(diff) {
				t.Fatalf("%s mismatch at [%d][%d]: got %s, want %s (diff=%v)",
					name, r, c, got.Data[r][c].String(), want.Data[r][c].String(), diff)
			}
		}
	}
}

// TestLUDecomposition verifies P*A == L*U for square and rectangular matrices.
func TestLUDecomposition(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{
			name: "2x2 invertable without row swap",
			expr: "lu([[4, 3], [6, 3]])",
		},
		{
			name: "2x2 requiring row swap (zero pivot)",
			expr: "lu([[0, 2], [3, 1]])",
		},
		{
			name: "3x3 general invertible",
			expr: "lu([[2, 1, 1], [4, -6, 0], [-2, 7, 2]])",
		},
		{
			name: "3x2 rectangular m > n",
			expr: "lu([[1, 2], [3, 4], [5, 6]])",
		},
		{
			name: "2x3 rectangular m < n",
			expr: "lu([[1, 2, 3], [4, 5, 6]])",
		},
		{
			name: "Singular 3x3 matrix",
			expr: "lu([[1, 2, 3], [2, 4, 6], [1, 1, 1]])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := evalDecompExpr(t, tt.expr)
			listNode, ok := res.(*ast.ListNode)
			if !ok || len(listNode.Elements) != 3 {
				t.Fatalf("expected 3-element list [P, L, U], got %T (%s)", res, res.String())
			}

			P := listNode.Elements[0].(*ast.MatrixNode)
			L := listNode.Elements[1].(*ast.MatrixNode)
			U := listNode.Elements[2].(*ast.MatrixNode)

			// Compute L * U
			lu, err := matMul(L, U)
			if err != nil {
				t.Fatalf("matMul(L, U) failed: %v", err)
			}

			// Compute P * A (extract A from original expression)
			// Alternatively, P is orthogonal so P^T * (L * U) == A
			pt := evalTranspose(P)
			reconstructedA, err := matMul(pt, lu)
			if err != nil {
				t.Fatalf("matMul(P^T, L*U) failed: %v", err)
			}

			// Evaluate original matrix A
			origExpr := tt.expr[3 : len(tt.expr)-1] // extract inside lu(...)
			origA := evalDecompExpr(t, origExpr).(*ast.MatrixNode)

			assertMatrixEqual(t, "P^T * L * U == A", reconstructedA, origA)
		})
	}
}

// TestQRDecomposition verifies A == Q*R and Q^T*Q == I using Two-Stage Gram-Schmidt.
func TestQRDecomposition(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{
			name: "2x2 orthogonal basis",
			expr: "qr([[1, 1], [1, 0]])",
		},
		{
			name: "2x2 Pythagorean 3-4",
			expr: "qr([[3, 1], [4, 2]])",
		},
		{
			name: "3x2 rectangular full column rank",
			expr: "qr([[1, 0], [1, 1], [0, 1]])",
		},
		{
			name: "3x3 full rank matrix",
			expr: "qr([[12, -51, 4], [6, 167, -68], [-4, 24, -41]])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := evalDecompExpr(t, tt.expr)
			listNode, ok := res.(*ast.ListNode)
			if !ok || len(listNode.Elements) != 2 {
				t.Fatalf("expected 2-element list [Q, R], got %T (%s)", res, res.String())
			}

			Q := listNode.Elements[0].(*ast.MatrixNode)
			R := listNode.Elements[1].(*ast.MatrixNode)

			// 1. Verify A == Q * R
			qr, err := matMul(Q, R)
			if err != nil {
				t.Fatalf("matMul(Q, R) failed: %v", err)
			}

			origExpr := tt.expr[3 : len(tt.expr)-1]
			origA := evalDecompExpr(t, origExpr).(*ast.MatrixNode)

			assertMatrixEqual(t, "Q * R == A", qr, origA)

			// 2. Verify Q^T * Q == I_n
			qt := evalTranspose(Q)
			qtq, err := matMul(qt, Q)
			if err != nil {
				t.Fatalf("matMul(Q^T, Q) failed: %v", err)
			}

			// Build Identity matrix of size n
			n := Q.Cols
			idData := make([][]ast.Node, n)
			for r := 0; r < n; r++ {
				idData[r] = make([]ast.Node, n)
				for c := 0; c < n; c++ {
					if r == c {
						idData[r][c] = mustRational(1, 1)
					} else {
						idData[r][c] = mustRational(0, 1)
					}
				}
			}
			idMat, _ := ast.NewMatrix(n, n, idData)

			assertMatrixEqual(t, "Q^T * Q == I", qtq, idMat)
		})
	}

	// Test rank deficiency error
	t.Run("linearly dependent columns error", func(t *testing.T) {
		_, err := evalDecompExprErr("qr([[1, 2], [2, 4]])")
		if err == nil {
			t.Fatalf("expected rank deficiency error on qr([[1, 2], [2, 4]]), got nil")
		}
	})
}

// TestCholeskyAndLDLT verifies A == L * D * L^T and A == L * L^T.
func TestCholeskyAndLDLT(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{
			name: "2x2 SPD matrix",
			expr: "[[4, 2], [2, 3]]",
		},
		{
			name: "3x3 symmetric tridiagonal SPD",
			expr: "[[2, -1, 0], [-1, 2, -1], [0, -1, 2]]",
		},
		{
			name: "3x3 Pascal matrix SPD",
			expr: "[[1, 1, 1], [1, 2, 3], [1, 3, 6]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_LDLT", func(t *testing.T) {
			res := evalDecompExpr(t, "ldlt("+tt.expr+")")
			listNode, ok := res.(*ast.ListNode)
			if !ok || len(listNode.Elements) != 2 {
				t.Fatalf("expected 2-element list [L, D], got %T", res)
			}

			L := listNode.Elements[0].(*ast.MatrixNode)
			D := listNode.Elements[1].(*ast.MatrixNode)

			// Compute L * D * L^T
			ld, err := matMul(L, D)
			if err != nil {
				t.Fatalf("matMul(L, D) failed: %v", err)
			}
			lt := evalTranspose(L)
			ldlt, err := matMul(ld, lt)
			if err != nil {
				t.Fatalf("matMul(L*D, L^T) failed: %v", err)
			}

			origA := evalDecompExpr(t, tt.expr).(*ast.MatrixNode)
			assertMatrixEqual(t, "L * D * L^T == A", ldlt, origA)
		})

		t.Run(tt.name+"_Cholesky", func(t *testing.T) {
			res := evalDecompExpr(t, "cholesky("+tt.expr+")")
			L, ok := res.(*ast.MatrixNode)
			if !ok {
				t.Fatalf("expected MatrixNode, got %T", res)
			}

			// Compute L * L^T
			lt := evalTranspose(L)
			llt, err := matMul(L, lt)
			if err != nil {
				t.Fatalf("matMul(L, L^T) failed: %v", err)
			}

			origA := evalDecompExpr(t, tt.expr).(*ast.MatrixNode)
			assertMatrixEqual(t, "L * L^T == A", llt, origA)
		})
	}

	// Test non-symmetric error
	t.Run("non-symmetric matrix error", func(t *testing.T) {
		_, err := evalDecompExprErr("cholesky([[1, 2], [3, 4]])")
		if err == nil {
			t.Fatalf("expected non-symmetric error, got nil")
		}
	})

	// Test non-positive-definite error
	t.Run("indefinite matrix error", func(t *testing.T) {
		_, err := evalDecompExprErr("cholesky([[1, 2], [2, 1]])")
		if err == nil {
			t.Fatalf("expected non-positive-definite error, got nil")
		}
	})
}

// TestPinvDecomposition verifies Moore-Penrose 4 conditions for exact pseudoinverse.
func TestPinvDecomposition(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{
			name: "1x1 non-zero",
			expr: "[[5]]",
		},
		{
			name: "2x2 invertible",
			expr: "[[1, 2], [3, 4]]",
		},
		{
			name: "3x2 full column rank",
			expr: "[[1, 0], [0, 1], [1, 1]]",
		},
		{
			name: "2x3 full row rank",
			expr: "[[1, 0, 1], [0, 1, 1]]",
		},
		{
			name: "2x2 rank 1 deficient matrix",
			expr: "[[1, 2], [2, 4]]",
		},
		{
			name: "3x3 rank 2 matrix",
			expr: "[[1, 2, 3], [4, 5, 6], [5, 7, 9]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origA := evalDecompExpr(t, tt.expr).(*ast.MatrixNode)
			res := evalDecompExpr(t, "pinv("+tt.expr+")")
			pinvA, ok := res.(*ast.MatrixNode)
			if !ok {
				t.Fatalf("expected MatrixNode, got %T", res)
			}

			// Condition 1: A * A^+ * A == A
			aPinv, err := matMul(origA, pinvA)
			if err != nil {
				t.Fatalf("matMul(A, A^+) failed: %v", err)
			}
			c1, err := matMul(aPinv, origA)
			if err != nil {
				t.Fatalf("matMul(A * A^+, A) failed: %v", err)
			}
			assertMatrixEqual(t, "Condition 1 (A * A^+ * A == A)", c1, origA)

			// Condition 2: A^+ * A * A^+ == A^+
			pinvA_A, err := matMul(pinvA, origA)
			if err != nil {
				t.Fatalf("matMul(A^+, A) failed: %v", err)
			}
			c2, err := matMul(pinvA_A, pinvA)
			if err != nil {
				t.Fatalf("matMul(A^+ * A, A^+) failed: %v", err)
			}
			assertMatrixEqual(t, "Condition 2 (A^+ * A * A^+ == A^+)", c2, pinvA)

			// Condition 3: (A * A^+)^T == A * A^+
			c3 := evalTranspose(aPinv)
			assertMatrixEqual(t, "Condition 3 ((A * A^+)^T == A * A^+)", c3, aPinv)

			// Condition 4: (A^+ * A)^T == A^+ * A
			c4 := evalTranspose(pinvA_A)
			assertMatrixEqual(t, "Condition 4 ((A^+ * A)^T == A^+ * A)", c4, pinvA_A)
		})
	}

	t.Run("zero matrix pinv", func(t *testing.T) {
		res := evalDecompExpr(t, "pinv([[0, 0, 0], [0, 0, 0]])")
		mat, ok := res.(*ast.MatrixNode)
		if !ok || mat.Rows != 3 || mat.Cols != 2 {
			t.Fatalf("expected 3x2 zero matrix, got %v", res)
		}
		for r := 0; r < 3; r++ {
			for c := 0; c < 2; c++ {
				if !isZero(mat.Data[r][c]) {
					t.Fatalf("expected zero at [%d][%d], got %v", r, c, mat.Data[r][c])
				}
			}
		}
	})
}
