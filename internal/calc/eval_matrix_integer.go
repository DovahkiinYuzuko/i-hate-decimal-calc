package calc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// extractIntegerMatrix validates that the node is a MatrixNode containing only integer elements,
// returning a 2D slice of big.Int.
func extractIntegerMatrix(node Node) ([][]big.Int, error) {
	matNode, ok := node.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_matrix_expected", node.String()))
	}
	if matNode.Rows == 0 || matNode.Cols == 0 {
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_empty_matrix"))
	}

	result := make([][]big.Int, matNode.Rows)
	for r := 0; r < matNode.Rows; r++ {
		result[r] = make([]big.Int, matNode.Cols)
		for c := 0; c < matNode.Cols; c++ {
			elem := matNode.Data[r][c]
			rat, isRat := elem.(*RationalNode)
			if !isRat || !rat.Val.IsInt() {
				return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_integer_matrix_expected"))
			}
			result[r][c].Set(rat.Val.Num())
		}
	}
	return result, nil
}

// buildMatrixNode converts a 2D big.Int slice into an exact MatrixNode.
func buildMatrixNode(mat [][]big.Int) (*MatrixNode, error) {
	rows := len(mat)
	if rows == 0 {
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_empty_matrix"))
	}
	cols := len(mat[0])
	data := make([][]Node, rows)
	for r := 0; r < rows; r++ {
		data[r] = make([]Node, cols)
		for c := 0; c < cols; c++ {
			val := new(big.Rat).SetInt(&mat[r][c])
			data[r][c] = NewRationalFromBigRat(val)
		}
	}
	return NewMatrix(rows, cols, data)
}

// copyIntMatrix creates a deep copy of a 2D big.Int matrix.
func copyIntMatrix(mat [][]big.Int) [][]big.Int {
	rows := len(mat)
	copied := make([][]big.Int, rows)
	for r := 0; r < rows; r++ {
		copied[r] = make([]big.Int, len(mat[r]))
		for c := 0; c < len(mat[r]); c++ {
			copied[r][c].Set(&mat[r][c])
		}
	}
	return copied
}

// identityIntMatrix returns an n x n identity matrix in big.Int.
func identityIntMatrix(n int) [][]big.Int {
	id := make([][]big.Int, n)
	for r := 0; r < n; r++ {
		id[r] = make([]big.Int, n)
		for c := 0; c < n; c++ {
			if r == c {
				id[r][c].SetInt64(1)
			} else {
				id[r][c].SetInt64(0)
			}
		}
	}
	return id
}

// euclideanDivMod computes q and rem such that a = q * b + rem with 0 <= rem < |b|.
// b must not be zero.
func euclideanDivMod(a, b *big.Int) (q, rem *big.Int) {
	q = new(big.Int)
	rem = new(big.Int)
	q.Div(a, b)
	rem.Mod(a, b)
	if rem.Sign() < 0 {
		if b.Sign() > 0 {
			rem.Add(rem, b)
			q.Sub(q, big.NewInt(1))
		} else {
			rem.Sub(rem, b)
			q.Add(q, big.NewInt(1))
		}
	}
	return q, rem
}

// ComputeHNF computes the Hermite Normal Form H and unimodular transform U (U * A = H).
// Algorithm: Kannan-Bachem (1979) with Chou-Collins (1982) immediate modulo reduction.
func ComputeHNF(mat [][]big.Int) ([][]big.Int, [][]big.Int, error) {
	m := len(mat)
	if m == 0 {
		return nil, nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_empty_matrix"))
	}
	n := len(mat[0])

	H := copyIntMatrix(mat)
	U := identityIntMatrix(m)

	pivotRow := 0

	for j := 0; j < n && pivotRow < m; j++ {
		// 1. Find a row k >= pivotRow with non-zero H[k][j] having minimum absolute value
		bestRow := -1
		minAbs := new(big.Int)

		for r := pivotRow; r < m; r++ {
			if H[r][j].Sign() != 0 {
				absVal := new(big.Int).Abs(&H[r][j])
				if bestRow == -1 || absVal.Cmp(minAbs) < 0 {
					bestRow = r
					minAbs.Set(absVal)
				}
			}
		}

		if bestRow == -1 {
			// Column j is entirely zero below pivotRow
			continue
		}

		// Swap bestRow to pivotRow
		if bestRow != pivotRow {
			H[pivotRow], H[bestRow] = H[bestRow], H[pivotRow]
			U[pivotRow], U[bestRow] = U[bestRow], U[pivotRow]
		}

		// 2. Eliminate all non-zero entries in column j below pivotRow using Euclidean reduction
		for r := pivotRow + 1; r < m; r++ {
			for H[r][j].Sign() != 0 {
				q := new(big.Int).Quo(&H[r][j], &H[pivotRow][j])
				if q.Sign() != 0 {
					// H[r] = H[r] - q * H[pivotRow]
					for c := 0; c < n; c++ {
						term := new(big.Int).Mul(q, &H[pivotRow][c])
						H[r][c].Sub(&H[r][c], term)
					}
					// U[r] = U[r] - q * U[pivotRow]
					for c := 0; c < m; c++ {
						term := new(big.Int).Mul(q, &U[pivotRow][c])
						U[r][c].Sub(&U[r][c], term)
					}
				}
				if H[r][j].Sign() != 0 {
					// Swap rows if remainder is non-zero
					H[pivotRow], H[r] = H[r], H[pivotRow]
					U[pivotRow], U[r] = U[r], U[pivotRow]
				}
			}
		}

		// 3. Ensure pivot H[pivotRow][j] > 0
		if H[pivotRow][j].Sign() < 0 {
			for c := 0; c < n; c++ {
				H[pivotRow][c].Neg(&H[pivotRow][c])
			}
			for c := 0; c < m; c++ {
				U[pivotRow][c].Neg(&U[pivotRow][c])
			}
		}

		// 4. Reduce elements above the pivot: 0 <= H[k][j] < H[pivotRow][j] for k < pivotRow
		pivotVal := new(big.Int).Set(&H[pivotRow][j])
		for k := 0; k < pivotRow; k++ {
			q, _ := euclideanDivMod(&H[k][j], pivotVal)
			if q.Sign() != 0 {
				for c := 0; c < n; c++ {
					term := new(big.Int).Mul(q, &H[pivotRow][c])
					H[k][c].Sub(&H[k][c], term)
				}
				for c := 0; c < m; c++ {
					term := new(big.Int).Mul(q, &U[pivotRow][c])
					U[k][c].Sub(&U[k][c], term)
				}
			}
		}

		pivotRow++
	}

	return H, U, nil
}

// ComputeSNF computes the Smith Normal Form D and unimodular transforms U, V such that U * A * V = D.
// D is diagonal with invariant factors d_0 | d_1 | ... | d_{r-1} > 0.
func ComputeSNF(mat [][]big.Int) ([][]big.Int, [][]big.Int, [][]big.Int, error) {
	m := len(mat)
	if m == 0 {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_empty_matrix"))
	}
	n := len(mat[0])

	D := copyIntMatrix(mat)
	U := identityIntMatrix(m)
	V := identityIntMatrix(n)

	fsm := NewSmithNormalFormFSM()
	minDim := m
	if n < minDim {
		minDim = n
	}

	for k := 0; k < minDim; k++ {
		if err := fsm.TransitionTo(SNFStateSelectPivot); err != nil {
			return nil, nil, nil, err
		}

		allZero := false
		for {
			// Step 1: Find non-zero element in D[k:, k:] with minimum absolute value
			pRow, pCol := -1, -1
			minAbs := new(big.Int)

			for r := k; r < m; r++ {
				for c := k; c < n; c++ {
					if D[r][c].Sign() != 0 {
						absVal := new(big.Int).Abs(&D[r][c])
						if pRow == -1 || absVal.Cmp(minAbs) < 0 {
							pRow = r
							pCol = c
							minAbs.Set(absVal)
						}
					}
				}
			}

			if pRow == -1 {
				// The entire active submatrix is 0
				allZero = true
				break
			}

			// Swap row pRow to k
			if pRow != k {
				D[k], D[pRow] = D[pRow], D[k]
				U[k], U[pRow] = U[pRow], U[k]
			}
			// Swap column pCol to k
			if pCol != k {
				for r := 0; r < m; r++ {
					D[r][k], D[r][pCol] = D[r][pCol], D[r][k]
				}
				for r := 0; r < n; r++ {
					V[r][k], V[r][pCol] = V[r][pCol], V[r][k]
				}
			}

			if err := fsm.TransitionTo(SNFStateEliminateRowCol); err != nil {
				return nil, nil, nil, err
			}

			// Step 2: Eliminate elements in column k (rows > k) and row k (cols > k)
			changed := false

			// Eliminate column entries D[r][k] for r > k
			for r := k + 1; r < m; r++ {
				if D[r][k].Sign() != 0 {
					q := new(big.Int).Quo(&D[r][k], &D[k][k])
					if q.Sign() != 0 {
						for c := 0; c < n; c++ {
							term := new(big.Int).Mul(q, &D[k][c])
							D[r][c].Sub(&D[r][c], term)
						}
						for c := 0; c < m; c++ {
							term := new(big.Int).Mul(q, &U[k][c])
							U[r][c].Sub(&U[r][c], term)
						}
					}
					if D[r][k].Sign() != 0 {
						// Remainder was non-zero, swap to pivot and restart
						D[k], D[r] = D[r], D[k]
						U[k], U[r] = U[r], U[k]
						changed = true
						break
					}
				}
			}
			if changed {
				continue
			}

			// Eliminate row entries D[k][c] for c > k
			for c := k + 1; c < n; c++ {
				if D[k][c].Sign() != 0 {
					q := new(big.Int).Quo(&D[k][c], &D[k][k])
					if q.Sign() != 0 {
						for r := 0; r < m; r++ {
							term := new(big.Int).Mul(q, &D[r][k])
							D[r][c].Sub(&D[r][c], term)
						}
						for r := 0; r < n; r++ {
							term := new(big.Int).Mul(q, &V[r][k])
							V[r][c].Sub(&V[r][c], term)
						}
					}
					if D[k][c].Sign() != 0 {
						// Remainder was non-zero, swap to pivot and restart
						for r := 0; r < m; r++ {
							D[r][k], D[r][c] = D[r][c], D[r][k]
						}
						for r := 0; r < n; r++ {
							V[r][k], V[r][c] = V[r][c], V[r][k]
						}
						changed = true
						break
					}
				}
			}
			if changed {
				continue
			}

			// Step 3: Check divisibility of D[k][k] across all entries in D[k+1:, k+1:]
			if err := fsm.TransitionTo(SNFStateCheckDivisibility); err != nil {
				return nil, nil, nil, err
			}

			divisibilityViolated := false
			for r := k + 1; r < m; r++ {
				for c := k + 1; c < n; c++ {
					if D[r][c].Sign() != 0 {
						rem := new(big.Int).Mod(&D[r][c], &D[k][k])
						if rem.Sign() != 0 {
							// Found entry not divisible by D[k][k]!
							if err := fsm.TransitionTo(SNFStateFixDivisibility); err != nil {
								return nil, nil, nil, err
							}
							// Add row r to row k: D[k] = D[k] + D[r]
							for colIdx := 0; colIdx < n; colIdx++ {
								D[k][colIdx].Add(&D[k][colIdx], &D[r][colIdx])
							}
							for colIdx := 0; colIdx < m; colIdx++ {
								U[k][colIdx].Add(&U[k][colIdx], &U[r][colIdx])
							}
							divisibilityViolated = true
							break
						}
					}
				}
				if divisibilityViolated {
					break
				}
			}

			if divisibilityViolated {
				// Re-eliminate row & col since D[k] now contains D[r][c]
				continue
			}

			// Both row & col are eliminated and all submatrix entries are divisible by D[k][k]
			break
		}

		if allZero {
			break
		}

		// Ensure pivot D[k][k] >= 0
		if D[k][k].Sign() < 0 {
			for c := 0; c < n; c++ {
				D[k][c].Neg(&D[k][c])
			}
			for c := 0; c < m; c++ {
				U[k][c].Neg(&U[k][c])
			}
		}

		if err := fsm.TransitionTo(SNFStateNextBlock); err != nil {
			return nil, nil, nil, err
		}
	}

	// Final verification of divisibility chain: d_0 | d_1 | ... | d_{r-1}
	if err := fsm.TransitionTo(SNFStateSuccess); err != nil {
		return nil, nil, nil, err
	}

	return D, U, V, nil
}

// ExtractInvariantFactors extracts the non-zero diagonal entries d_i from a Smith Normal Form matrix D.
func ExtractInvariantFactors(D [][]big.Int) []big.Int {
	var factors []big.Int
	m := len(D)
	if m == 0 {
		return factors
	}
	n := len(D[0])
	minDim := m
	if n < minDim {
		minDim = n
	}

	for i := 0; i < minDim; i++ {
		if D[i][i].Sign() > 0 {
			factors = append(factors, *new(big.Int).Set(&D[i][i]))
		}
	}
	return factors
}

// FormatAbelianGroupStructure generates the direct sum presentation for a finitely generated abelian group
// G = Z^m / Im(A), given the invariant factors and total presentation dimension m.
func FormatAbelianGroupStructure(factors []big.Int, m int) string {
	var terms []string

	// Cyclic groups Z/d_i Z for d_i > 1
	for _, d := range factors {
		if d.Cmp(big.NewInt(1)) > 0 {
			terms = append(terms, fmt.Sprintf("Z_%s", d.String()))
		}
	}

	// Free rank = m - rank(A) = m - len(factors)
	freeRank := m - len(factors)
	if freeRank > 0 {
		for i := 0; i < freeRank; i++ {
			terms = append(terms, "Z")
		}
	}

	if len(terms) == 0 {
		return "0"
	}
	return strings.Join(terms, " + ")
}

// -------------------------------------------------------------------------
// CLI Evaluation Handlers
// -------------------------------------------------------------------------

// EvalHNF evaluates hnf(matrix).
func EvalHNF(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_hnf_args"))
	}
	intMat, err := extractIntegerMatrix(args[0])
	if err != nil {
		return nil, err
	}
	H, _, err := ComputeHNF(intMat)
	if err != nil {
		return nil, err
	}
	return buildMatrixNode(H)
}

// EvalHNFTransform evaluates hnf_transform(matrix), returning [H, U].
func EvalHNFTransform(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_hnf_transform_args"))
	}
	intMat, err := extractIntegerMatrix(args[0])
	if err != nil {
		return nil, err
	}
	H, U, err := ComputeHNF(intMat)
	if err != nil {
		return nil, err
	}
	hNode, err := buildMatrixNode(H)
	if err != nil {
		return nil, err
	}
	uNode, err := buildMatrixNode(U)
	if err != nil {
		return nil, err
	}
	return &ListNode{Elements: []Node{hNode, uNode}}, nil
}

// EvalSNF evaluates snf(matrix).
func EvalSNF(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_snf_args"))
	}
	intMat, err := extractIntegerMatrix(args[0])
	if err != nil {
		return nil, err
	}
	D, _, _, err := ComputeSNF(intMat)
	if err != nil {
		return nil, err
	}
	return buildMatrixNode(D)
}

// EvalSNFTransform evaluates snf_transform(matrix), returning [D, U, V].
func EvalSNFTransform(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_snf_transform_args"))
	}
	intMat, err := extractIntegerMatrix(args[0])
	if err != nil {
		return nil, err
	}
	D, U, V, err := ComputeSNF(intMat)
	if err != nil {
		return nil, err
	}
	dNode, err := buildMatrixNode(D)
	if err != nil {
		return nil, err
	}
	uNode, err := buildMatrixNode(U)
	if err != nil {
		return nil, err
	}
	vNode, err := buildMatrixNode(V)
	if err != nil {
		return nil, err
	}
	return &ListNode{Elements: []Node{dNode, uNode, vNode}}, nil
}

// EvalInvariantFactors evaluates invariant_factors(matrix).
func EvalInvariantFactors(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_invariant_factors_args"))
	}
	intMat, err := extractIntegerMatrix(args[0])
	if err != nil {
		return nil, err
	}
	D, _, _, err := ComputeSNF(intMat)
	if err != nil {
		return nil, err
	}
	factors := ExtractInvariantFactors(D)
	var elemNodes []Node
	for _, f := range factors {
		elemNodes = append(elemNodes, NewRationalFromBigRat(new(big.Rat).SetInt(&f)))
	}
	return &ListNode{Elements: elemNodes}, nil
}

// EvalAbelianGroupStructure evaluates abelian_group_structure(matrix).
func EvalAbelianGroupStructure(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("matrix_integer.err_abelian_group_structure_args"))
	}
	intMat, err := extractIntegerMatrix(args[0])
	if err != nil {
		return nil, err
	}
	m := len(intMat)
	D, _, _, err := ComputeSNF(intMat)
	if err != nil {
		return nil, err
	}
	factors := ExtractInvariantFactors(D)
	structureStr := FormatAbelianGroupStructure(factors, m)
	return &ConstNode{Name: structureStr}, nil
}

func init() {
	RegisterHandler("hnf", func(args []Node, env *Env) (Node, error) {
		return EvalHNF(args, env)
	})
	RegisterHandler("hnf_transform", func(args []Node, env *Env) (Node, error) {
		return EvalHNFTransform(args, env)
	})
	RegisterHandler("snf", func(args []Node, env *Env) (Node, error) {
		return EvalSNF(args, env)
	})
	RegisterHandler("snf_transform", func(args []Node, env *Env) (Node, error) {
		return EvalSNFTransform(args, env)
	})
	RegisterHandler("invariant_factors", func(args []Node, env *Env) (Node, error) {
		return EvalInvariantFactors(args, env)
	})
	RegisterHandler("abelian_group_structure", func(args []Node, env *Env) (Node, error) {
		return EvalAbelianGroupStructure(args, env)
	})
}

