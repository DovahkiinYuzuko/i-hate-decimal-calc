package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// computeRREF calculates the exact Reduced Row Echelon Form (RREF) of a matrix
// using Bareiss Fraction-Free Gaussian Elimination followed by back-substitution and normalization.
// It returns the resulting RREF MatrixNode and a slice of 0-based pivot column indices.
func computeRREF(m *MatrixNode) (*MatrixNode, []int, error) {
	if m == nil {
		return nil, nil, fmt.Errorf("cannot compute RREF of nil matrix")
	}

	rows := m.Rows
	cols := m.Cols

	// Deep copy data
	A := make([][]Node, rows)
	for r := 0; r < rows; r++ {
		A[r] = make([]Node, cols)
		for c := 0; c < cols; c++ {
			evaled, err := Eval(m.Data[r][c])
			if err != nil {
				evaled = m.Data[r][c]
			}
			A[r][c] = evaled
		}
	}

	var pivotCols []int
	lead := 0
	var prevPivot Node = mustRational(1, 1)

	for r := 0; r < rows && lead < cols; lead++ {
		// Find pivot row
		pivotRow := -1
		for i := r; i < rows; i++ {
			if !isZero(A[i][lead]) {
				pivotRow = i
				break
			}
		}

		if pivotRow == -1 {
			// No pivot in this column
			continue
		}

		// Swap rows if necessary
		if pivotRow != r {
			A[r], A[pivotRow] = A[pivotRow], A[r]
		}

		curPivot := A[r][lead]
		pivotCols = append(pivotCols, lead)

		// Bareiss elimination below row r
		for i := r + 1; i < rows; i++ {
			if isZero(A[i][lead]) {
				continue
			}

			ail := A[i][lead]
			for j := lead + 1; j < cols; j++ {
				// A[i][j] = (curPivot * A[i][j] - A[i][lead] * A[r][j]) / prevPivot
				p1, err := simplifyMul([]Node{curPivot, A[i][j]})
				if err != nil {
					return nil, nil, err
				}
				p2, err := simplifyMul([]Node{ail, A[r][j]})
				if err != nil {
					return nil, nil, err
				}
				negP2, err := simplifyUnaryOp("-", p2)
				if err != nil {
					return nil, nil, err
				}
				num, err := simplifyAdd([]Node{p1, negP2})
				if err != nil {
					return nil, nil, err
				}

				invPrev, err := simplifyPow(prevPivot, mustRational(-1, 1))
				if err != nil {
					return nil, nil, err
				}
				resElem, err := simplifyMul([]Node{num, invPrev})
				if err != nil {
					return nil, nil, err
				}
				evaledElem, err := Eval(resElem)
				if err != nil {
					evaledElem = resElem
				}
				A[i][j] = evaledElem
			}
			A[i][lead] = mustRational(0, 1)
		}

		prevPivot = curPivot
		r++
	}

	// Back-elimination and pivot normalization to achieve RREF
	numPivots := len(pivotCols)
	for k := numPivots - 1; k >= 0; k-- {
		pCol := pivotCols[k]
		pRow := k
		pivotVal := A[pRow][pCol]

		// Normalize pivot row: divide by pivotVal so A[pRow][pCol] = 1
		invPivot, err := simplifyPow(pivotVal, mustRational(-1, 1))
		if err != nil {
			return nil, nil, err
		}
		for j := pCol; j < cols; j++ {
			if j == pCol {
				A[pRow][j] = mustRational(1, 1)
				continue
			}
			normElem, err := simplifyMul([]Node{A[pRow][j], invPivot})
			if err != nil {
				return nil, nil, err
			}
			evaledNorm, err := Eval(normElem)
			if err != nil {
				evaledNorm = normElem
			}
			A[pRow][j] = evaledNorm
		}

		// Eliminate entries above row pRow in column pCol
		for i := 0; i < pRow; i++ {
			factor := A[i][pCol]
			if isZero(factor) {
				continue
			}

			negFactor, err := simplifyUnaryOp("-", factor)
			if err != nil {
				return nil, nil, err
			}

			for j := pCol; j < cols; j++ {
				if j == pCol {
					A[i][j] = mustRational(0, 1)
					continue
				}
				term, err := simplifyMul([]Node{negFactor, A[pRow][j]})
				if err != nil {
					return nil, nil, err
				}
				sum, err := simplifyAdd([]Node{A[i][j], term})
				if err != nil {
					return nil, nil, err
				}
				evaledSum, err := Eval(sum)
				if err != nil {
					evaledSum = sum
				}
				A[i][j] = evaledSum
			}
		}
	}

	resMat, err := NewMatrix(rows, cols, A)
	if err != nil {
		return nil, nil, err
	}
	return resMat, pivotCols, nil
}

// evalRREF computes the Reduced Row Echelon Form of a matrix.
func evalRREF(m *MatrixNode) (*MatrixNode, error) {
	rrefMat, _, err := computeRREF(m)
	return rrefMat, err
}

// evalRank returns the exact algebraic rank of a matrix (number of pivots).
func evalRank(m *MatrixNode) (Node, error) {
	_, pivotCols, err := computeRREF(m)
	if err != nil {
		return nil, err
	}
	return mustRational(int64(len(pivotCols)), 1), nil
}

// evalSolveLinear solves the linear system A * x = b for exact solutions.
func evalSolveLinear(matNode, bNode Node) (Node, error) {
	evalA, err := Eval(matNode)
	if err != nil {
		return nil, err
	}
	A, ok := evalA.(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("solve_linear error: first argument must be a matrix, got %T", evalA)
	}

	evalB, err := Eval(bNode)
	if err != nil {
		return nil, err
	}

	var bElems []Node
	switch vb := evalB.(type) {
	case *ListNode:
		bElems = vb.Elements
	case *MatrixNode:
		if vb.Cols == 1 {
			bElems = make([]Node, vb.Rows)
			for r := 0; r < vb.Rows; r++ {
				bElems[r] = vb.Data[r][0]
			}
		} else if vb.Rows == 1 {
			bElems = vb.Data[0]
		} else {
			return nil, fmt.Errorf("solve_linear error: constant vector b must be 1D or column matrix, got %dx%d", vb.Rows, vb.Cols)
		}
	default:
		return nil, fmt.Errorf("solve_linear error: second argument must be a vector or list, got %T", evalB)
	}

	if len(bElems) != A.Rows {
		return nil, fmt.Errorf("%s (matrix has %d rows, vector has %d elements)",
			i18n.T("errors.linear_dim_mismatch"), A.Rows, len(bElems))
	}

	// Build augmented matrix [A | b] with dimensions A.Rows x (A.Cols + 1)
	augCols := A.Cols + 1
	augData := make([][]Node, A.Rows)
	for r := 0; r < A.Rows; r++ {
		augData[r] = make([]Node, augCols)
		for c := 0; c < A.Cols; c++ {
			augData[r][c] = A.Data[r][c]
		}
		augData[r][A.Cols] = bElems[r]
	}

	augMat, err := NewMatrix(A.Rows, augCols, augData)
	if err != nil {
		return nil, err
	}

	augRREF, pivotCols, err := computeRREF(augMat)
	if err != nil {
		return nil, err
	}

	// Check consistency: if the last column (index A.Cols) is a pivot column,
	// then there is a row [0, 0, ..., 0 | 1], which means 0 = 1 (inconsistent).
	for _, pc := range pivotCols {
		if pc == A.Cols {
			return nil, fmt.Errorf("%s", i18n.T("errors.linear_inconsistent"))
		}
	}

	// Check if unique solution exists: rank == A.Cols
	if len(pivotCols) == A.Cols {
		solution := make([]Node, A.Cols)
		for r, pc := range pivotCols {
			solution[pc] = augRREF.Data[r][A.Cols]
		}
		return &ListNode{Elements: solution}, nil
	}

	// Underdetermined system (infinitely many solutions):
	// Express each pivot variable in terms of free variables.
	isPivot := make([]bool, A.Cols)
	pivotRowForCol := make([]int, A.Cols)
	for r, pc := range pivotCols {
		isPivot[pc] = true
		pivotRowForCol[pc] = r
	}

	solution := make([]Node, A.Cols)
	for c := 0; c < A.Cols; c++ {
		if !isPivot[c] {
			// Free variable: assign parameter x_c+1 (e.g. x1, x2...)
			solution[c] = &VarNode{Name: fmt.Sprintf("x%d", c+1)}
		} else {
			r := pivotRowForCol[c]
			constTerm := augRREF.Data[r][A.Cols]
			var terms []Node
			if !isZero(constTerm) {
				terms = append(terms, constTerm)
			}
			for j := c + 1; j < A.Cols; j++ {
				coeff := augRREF.Data[r][j]
				if isZero(coeff) {
					continue
				}
				negCoeff, _ := simplifyUnaryOp("-", coeff)
				freeVar := &VarNode{Name: fmt.Sprintf("x%d", j+1)}
				prod, _ := simplifyMul([]Node{negCoeff, freeVar})
				terms = append(terms, prod)
			}
			if len(terms) == 0 {
				solution[c] = mustRational(0, 1)
			} else if len(terms) == 1 {
				solution[c] = terms[0]
			} else {
				solution[c], _ = Eval(NewAdd(terms))
			}
		}
	}

	return &ListNode{Elements: solution}, nil
}
