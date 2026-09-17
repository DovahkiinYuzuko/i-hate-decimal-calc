package calc

import (
	"errors"
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Helper matrix arithmetic functions for decompositions
// -------------------------------------------------------------------------

func matMul(a, b *ast.MatrixNode) (*ast.MatrixNode, error) {
	node, err := mulMatrixOrScalar(a, b)
	if err != nil {
		return nil, err
	}
	mat, ok := node.(*ast.MatrixNode)
	if !ok {
		return nil, fmt.Errorf("expected MatrixNode from multiplication, got %T", node)
	}
	return mat, nil
}

func dotProductNodes(u, v []ast.Node) (ast.Node, error) {
	if len(u) != len(v) {
		return nil, fmt.Errorf("vector length mismatch: %d vs %d", len(u), len(v))
	}
	var terms []ast.Node
	for i := 0; i < len(u); i++ {
		prod, err := simplifyMul([]ast.Node{u[i], v[i]})
		if err != nil {
			return nil, err
		}
		terms = append(terms, prod)
	}
	if len(terms) == 0 {
		return mustRational(0, 1), nil
	}
	return simplifyAdd(terms)
}

func isNodePositive(n ast.Node) bool {
	if rn, ok := n.(*ast.RationalNode); ok {
		return rn.Val.Sign() > 0
	}
	return false
}

func isNodeNonPositive(n ast.Node) bool {
	if rn, ok := n.(*ast.RationalNode); ok {
		return rn.Val.Sign() <= 0
	}
	// If it's a symbolic zero
	if isZero(n) {
		return true
	}
	return false
}

// -------------------------------------------------------------------------
// LU Decomposition (with row pivoting)
// P * A = L * U
// -------------------------------------------------------------------------

// evalLU computes the PLU decomposition of matrix A (m x n).
// Returns [P, L, U] where:
//   P is an m x m permutation matrix
//   L is an m x min(m, n) unit lower triangular/trapezoidal matrix
//   U is a min(m, n) x n upper triangular/trapezoidal matrix
// such that P * A = L * U.
func evalLU(mat *ast.MatrixNode) (*ast.ListNode, error) {
	if mat == nil || mat.Rows == 0 || mat.Cols == 0 {
		return nil, fmt.Errorf("cannot compute LU decomposition of empty matrix")
	}

	m := mat.Rows
	n := mat.Cols
	kMin := m
	if n < kMin {
		kMin = n
	}

	// P starts as m x m identity matrix
	pData := make([][]ast.Node, m)
	for r := 0; r < m; r++ {
		pData[r] = make([]ast.Node, m)
		for c := 0; c < m; c++ {
			if r == c {
				pData[r][c] = mustRational(1, 1)
			} else {
				pData[r][c] = mustRational(0, 1)
			}
		}
	}

	// L starts as m x kMin matrix with 1s on diagonal
	lData := make([][]ast.Node, m)
	for r := 0; r < m; r++ {
		lData[r] = make([]ast.Node, kMin)
		for c := 0; c < kMin; c++ {
			if r == c {
				lData[r][c] = mustRational(1, 1)
			} else {
				lData[r][c] = mustRational(0, 1)
			}
		}
	}

	// U starts as a copy of A (m x n), which we'll reduce and then truncate to kMin x n
	uData := make([][]ast.Node, m)
	for r := 0; r < m; r++ {
		uData[r] = make([]ast.Node, n)
		for c := 0; c < n; c++ {
			evaled, err := Eval(mat.Data[r][c])
			if err != nil {
				evaled = mat.Data[r][c]
			}
			uData[r][c] = evaled
		}
	}

	// Elimination with partial row pivoting
	for k := 0; k < kMin; k++ {
		// Find first non-zero pivot in column k at or below row k
		pivotRow := -1
		for i := k; i < m; i++ {
			if !isZero(uData[i][k]) {
				pivotRow = i
				break
			}
		}

		if pivotRow == -1 {
			// Column k has no non-zero pivot below diagonal, skip reduction for this column
			continue
		}

		// Swap rows in U, P, and previously calculated columns in L
		if pivotRow != k {
			uData[k], uData[pivotRow] = uData[pivotRow], uData[k]
			pData[k], pData[pivotRow] = pData[pivotRow], pData[k]
			// In L, swap entries before column k
			for col := 0; col < k; col++ {
				lData[k][col], lData[pivotRow][col] = lData[pivotRow][col], lData[k][col]
			}
		}

		curPivot := uData[k][k]
		invPivot, err := simplifyPow(curPivot, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}

		// Eliminate below pivot
		for i := k + 1; i < m; i++ {
			if isZero(uData[i][k]) {
				continue
			}

			// factor = uData[i][k] / curPivot
			factor, err := simplifyMul([]ast.Node{uData[i][k], invPivot})
			if err != nil {
				return nil, err
			}
			lData[i][k] = factor
			uData[i][k] = mustRational(0, 1)

			negFactor, err := simplifyUnaryOp("-", factor)
			if err != nil {
				return nil, err
			}

			for j := k + 1; j < n; j++ {
				term, err := simplifyMul([]ast.Node{negFactor, uData[k][j]})
				if err != nil {
					return nil, err
				}
				newVal, err := simplifyAdd([]ast.Node{uData[i][j], term})
				if err != nil {
					return nil, err
				}
				uData[i][j] = newVal
			}
		}
	}

	// Truncate U to kMin x n
	uFinalData := uData[:kMin]

	pMat, _ := ast.NewMatrix(m, m, pData)
	lMat, _ := ast.NewMatrix(m, kMin, lData)
	uMat, _ := ast.NewMatrix(kMin, n, uFinalData)

	return &ast.ListNode{Elements: []ast.Node{pMat, lMat, uMat}}, nil
}

// -------------------------------------------------------------------------
// Two-Stage QR Decomposition
// A = Q * R
// -------------------------------------------------------------------------

// evalTwoStageQR computes the QR decomposition of matrix A (m x n, m >= n)
// using Two-Stage Gram-Schmidt:
//   Stage 1: Pure rational orthogonalization over Q without normalization.
//   Stage 2: Terminal normalization with exact radicals.
// Returns [Q, R] where Q has orthonormal columns and R is upper triangular.
func evalTwoStageQR(mat *ast.MatrixNode) (*ast.ListNode, error) {
	if mat == nil || mat.Rows == 0 || mat.Cols == 0 {
		return nil, fmt.Errorf("cannot compute QR decomposition of empty matrix")
	}

	m := mat.Rows
	n := mat.Cols

	if m < n {
		return nil, fmt.Errorf("%s error: QR decomposition requires rows >= cols, got %dx%d", "qr", m, n)
	}

	// Extract column vectors of A
	aCols := make([][]ast.Node, n)
	for c := 0; c < n; c++ {
		aCols[c] = make([]ast.Node, m)
		for r := 0; r < m; r++ {
			evaled, err := Eval(mat.Data[r][c])
			if err != nil {
				evaled = mat.Data[r][c]
			}
			aCols[c][r] = evaled
		}
	}

	// Stage 1: Orthogonal basis u_0, ..., u_{n-1} over Q and unnormalized R_unit
	uCols := make([][]ast.Node, n)
	rUnit := make([][]ast.Node, n)
	for i := 0; i < n; i++ {
		rUnit[i] = make([]ast.Node, n)
		for j := 0; j < n; j++ {
			if i == j {
				rUnit[i][j] = mustRational(1, 1)
			} else {
				rUnit[i][j] = mustRational(0, 1)
			}
		}
	}

	normSqList := make([]ast.Node, n)

	for k := 0; k < n; k++ {
		// Start with u_k = a_k
		uk := make([]ast.Node, m)
		copy(uk, aCols[k])

		// Subtract projections onto prior u_j (j < k)
		for j := 0; j < k; j++ {
			den := normSqList[j]
			if isZero(den) {
				return nil, errors.New(i18n.T("errors.matrix_qr_rank_deficient", "qr", j+1))
			}

			// num = a_k . u_j
			num, err := dotProductNodes(aCols[k], uCols[j])
			if err != nil {
				return nil, err
			}

			// c = num / den
			invDen, err := simplifyPow(den, mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			c, err := simplifyMul([]ast.Node{num, invDen})
			if err != nil {
				return nil, err
			}

			rUnit[j][k] = c

			negC, err := simplifyUnaryOp("-", c)
			if err != nil {
				return nil, err
			}

			// uk = uk - c * u_j
			for r := 0; r < m; r++ {
				subTerm, err := simplifyMul([]ast.Node{negC, uCols[j][r]})
				if err != nil {
					return nil, err
				}
				newUkElem, err := simplifyAdd([]ast.Node{uk[r], subTerm})
				if err != nil {
					return nil, err
				}
				uk[r] = newUkElem
			}
		}

		uCols[k] = uk

		// Norm squared: u_k . u_k
		normSq, err := dotProductNodes(uk, uk)
		if err != nil {
			return nil, err
		}
		if isZero(normSq) {
			return nil, errors.New(i18n.T("errors.matrix_qr_rank_deficient", "qr", k+1))
		}
		normSqList[k] = normSq
	}

	// Stage 2: Terminal normalization with exact radicals
	dNorms := make([]ast.Node, n)
	for k := 0; k < n; k++ {
		normNode, err := simplifySqrt(normSqList[k])
		if err != nil {
			return nil, err
		}
		dNorms[k] = normNode
	}

	// Build Q (m x n): Q[r][k] = uCols[k][r] / dNorms[k]
	qData := make([][]ast.Node, m)
	for r := 0; r < m; r++ {
		qData[r] = make([]ast.Node, n)
		for k := 0; k < n; k++ {
			invNorm, err := simplifyPow(dNorms[k], mustRational(-1, 1))
			if err != nil {
				return nil, err
			}
			qElem, err := simplifyMul([]ast.Node{uCols[k][r], invNorm})
			if err != nil {
				return nil, err
			}
			qData[r][k] = qElem
		}
	}

	// Build R (n x n):
	// R[k][k] = dNorms[k]
	// R[j][k] = rUnit[j][k] * dNorms[j] for j < k
	// R[j][k] = 0 for j > k
	rData := make([][]ast.Node, n)
	for j := 0; j < n; j++ {
		rData[j] = make([]ast.Node, n)
		for k := 0; k < n; k++ {
			if j == k {
				rData[j][k] = dNorms[k]
			} else if j < k {
				rElem, err := simplifyMul([]ast.Node{rUnit[j][k], dNorms[j]})
				if err != nil {
					return nil, err
				}
				rData[j][k] = rElem
			} else {
				rData[j][k] = mustRational(0, 1)
			}
		}
	}

	qMat, _ := ast.NewMatrix(m, n, qData)
	rMat, _ := ast.NewMatrix(n, n, rData)

	return &ast.ListNode{Elements: []ast.Node{qMat, rMat}}, nil
}

// -------------------------------------------------------------------------
// LDL^T Decomposition & Cholesky Decomposition
// A = L * D * L^T and A = L * L^T
// -------------------------------------------------------------------------

// evalLDLT computes the square-root-free LDL^T decomposition of a symmetric positive-definite matrix A.
// Returns [L_unit, D_mat] where L_unit is unit lower triangular and D_mat is diagonal with positive entries.
func evalLDLT(mat *ast.MatrixNode) (*ast.ListNode, error) {
	if mat == nil || mat.Rows == 0 || mat.Cols == 0 {
		return nil, fmt.Errorf("cannot compute LDL^T decomposition of empty matrix")
	}

	n := mat.Rows
	if mat.Cols != n {
		return nil, errors.New(i18n.T("errors.matrix_must_be_square", "ldlt", mat.Rows, mat.Cols))
	}

	// Check symmetry: A[i][j] == A[j][i]
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			diff, err := simplifyAdd([]ast.Node{mat.Data[i][j], mustNegate(mat.Data[j][i])})
			if err != nil || !isZero(diff) {
				return nil, errors.New(i18n.T("errors.matrix_not_symmetric", "ldlt"))
			}
		}
	}

	lData := make([][]ast.Node, n)
	for i := 0; i < n; i++ {
		lData[i] = make([]ast.Node, n)
		for j := 0; j < n; j++ {
			if i == j {
				lData[i][j] = mustRational(1, 1)
			} else {
				lData[i][j] = mustRational(0, 1)
			}
		}
	}

	dList := make([]ast.Node, n)

	for j := 0; j < n; j++ {
		// D_j = A[j][j] - sum_{k=0}^{j-1} L[j][k]^2 * D_k
		var sumTerms []ast.Node
		for k := 0; k < j; k++ {
			l2, err := simplifyPow(lData[j][k], mustRational(2, 1))
			if err != nil {
				return nil, err
			}
			term, err := simplifyMul([]ast.Node{l2, dList[k]})
			if err != nil {
				return nil, err
			}
			sumTerms = append(sumTerms, term)
		}
		var sumTerm ast.Node = mustRational(0, 1)
		if len(sumTerms) > 0 {
			var err error
			sumTerm, err = simplifyAdd(sumTerms)
			if err != nil {
				return nil, err
			}
		}

		dj, err := simplifyAdd([]ast.Node{mat.Data[j][j], mustNegate(sumTerm)})
		if err != nil {
			return nil, err
		}

		// Positive-definiteness check: D_j must be strictly positive
		if isNodeNonPositive(dj) {
			return nil, errors.New(i18n.T("errors.matrix_not_positive_definite", "ldlt", j+1))
		}

		dList[j] = dj

		// For i = j+1 to n-1:
		// L[i][j] = (A[i][j] - sum_{k=0}^{j-1} L[i][k] * L[j][k] * D_k) / D_j
		invDj, err := simplifyPow(dj, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}

		for i := j + 1; i < n; i++ {
			var sumLTerms []ast.Node
			for k := 0; k < j; k++ {
				prod, err := simplifyMul([]ast.Node{lData[i][k], lData[j][k], dList[k]})
				if err != nil {
					return nil, err
				}
				sumLTerms = append(sumLTerms, prod)
			}
			var sumL ast.Node = mustRational(0, 1)
			if len(sumLTerms) > 0 {
				var err error
				sumL, err = simplifyAdd(sumLTerms)
				if err != nil {
					return nil, err
				}
			}

			num, err := simplifyAdd([]ast.Node{mat.Data[i][j], mustNegate(sumL)})
			if err != nil {
				return nil, err
			}

			lij, err := simplifyMul([]ast.Node{num, invDj})
			if err != nil {
				return nil, err
			}
			lData[i][j] = lij
		}
	}

	// Create D matrix
	dData := make([][]ast.Node, n)
	for i := 0; i < n; i++ {
		dData[i] = make([]ast.Node, n)
		for j := 0; j < n; j++ {
			if i == j {
				dData[i][j] = dList[i]
			} else {
				dData[i][j] = mustRational(0, 1)
			}
		}
	}

	lMat, _ := ast.NewMatrix(n, n, lData)
	dMat, _ := ast.NewMatrix(n, n, dData)

	return &ast.ListNode{Elements: []ast.Node{lMat, dMat}}, nil
}

// evalCholesky computes the Cholesky decomposition A = L * L^T for a symmetric positive-definite matrix A.
// It leverages evalLDLT internally to maintain exact rational arithmetic and constructs L = L_unit * sqrt(D).
func evalCholesky(mat *ast.MatrixNode) (*ast.MatrixNode, error) {
	ldltRes, err := evalLDLT(mat)
	if err != nil {
		return nil, err
	}

	lUnitMat := ldltRes.Elements[0].(*ast.MatrixNode)
	dMat := ldltRes.Elements[1].(*ast.MatrixNode)
	n := mat.Rows

	// L[i][j] = L_unit[i][j] * sqrt(D_j)
	lData := make([][]ast.Node, n)
	for i := 0; i < n; i++ {
		lData[i] = make([]ast.Node, n)
		for j := 0; j < n; j++ {
			if j > i {
				lData[i][j] = mustRational(0, 1)
				continue
			}
			dj := dMat.Data[j][j]
			sqrtDj, err := simplifySqrt(dj)
			if err != nil {
				return nil, err
			}

			elem, err := simplifyMul([]ast.Node{lUnitMat.Data[i][j], sqrtDj})
			if err != nil {
				return nil, err
			}
			lData[i][j] = elem
		}
	}

	return ast.NewMatrix(n, n, lData)
}

// -------------------------------------------------------------------------
// Moore-Penrose Pseudoinverse (pinv)
// Full-Rank Factorization via RREF: A = B * C -> A^+ = C^T * (C * C^T)^-1 * (B^T * B)^-1 * B^T
// -------------------------------------------------------------------------

// evalPinv computes the exact Moore-Penrose pseudoinverse A^+ of matrix A (m x n).
func evalPinv(mat *ast.MatrixNode) (*ast.MatrixNode, error) {
	if mat == nil || mat.Rows == 0 || mat.Cols == 0 {
		return nil, fmt.Errorf("cannot compute pseudoinverse of empty matrix")
	}

	m := mat.Rows
	n := mat.Cols

	// Case 1: All zero matrix -> returns n x m zero matrix
	isAllZero := true
	for r := 0; r < m; r++ {
		for c := 0; c < n; c++ {
			if !isZero(mat.Data[r][c]) {
				isAllZero = false
				break
			}
		}
		if !isAllZero {
			break
		}
	}
	if isAllZero {
		zeroData := make([][]ast.Node, n)
		for r := 0; r < n; r++ {
			zeroData[r] = make([]ast.Node, m)
			for c := 0; c < m; c++ {
				zeroData[r][c] = mustRational(0, 1)
			}
		}
		return ast.NewMatrix(n, m, zeroData)
	}

	// Case 2: 1x1 matrix [a]
	if m == 1 && n == 1 {
		a := mat.Data[0][0]
		if isZero(a) {
			return ast.NewMatrix(1, 1, [][]ast.Node{{mustRational(0, 1)}})
		}
		invA, err := simplifyPow(a, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		return ast.NewMatrix(1, 1, [][]ast.Node{{invA}})
	}

	// Case 3: Square non-singular matrix -> standard inverse
	if m == n {
		det, err := evalDet(mat)
		if err == nil && !isZero(det) {
			invMat, err := evalInv(mat)
			if err == nil {
				return invMat, nil
			}
		}
	}

	// Case 4: General Full-Rank Factorization via RREF
	rrefMat, pivotCols, err := computeRREF(mat)
	if err != nil {
		return nil, err
	}

	r := len(pivotCols)
	if r == 0 {
		zeroData := make([][]ast.Node, n)
		for row := 0; row < n; row++ {
			zeroData[row] = make([]ast.Node, m)
			for col := 0; col < m; col++ {
				zeroData[row][col] = mustRational(0, 1)
			}
		}
		return ast.NewMatrix(n, m, zeroData)
	}

	// Shortcut: Full column rank (r == n) -> A^+ = (A^T * A)^-1 * A^T
	if r == n {
		at := evalTranspose(mat)
		ata, err := matMul(at, mat)
		if err == nil {
			invAta, err := evalInv(ata)
			if err == nil {
				return matMul(invAta, at)
			}
		}
	}

	// Shortcut: Full row rank (r == m) -> A^+ = A^T * (A * A^T)^-1
	if r == m {
		at := evalTranspose(mat)
		aat, err := matMul(mat, at)
		if err == nil {
			invAat, err := evalInv(aat)
			if err == nil {
				return matMul(at, invAat)
			}
		}
	}

	// General rank r: A = B * C
	// B (m x r): take pivot columns from the ORIGINAL matrix A
	bData := make([][]ast.Node, m)
	for row := 0; row < m; row++ {
		bData[row] = make([]ast.Node, r)
		for col := 0; col < r; col++ {
			bData[row][col] = mat.Data[row][pivotCols[col]]
		}
	}
	bMat, _ := ast.NewMatrix(m, r, bData)

	// C (r x n): take first r non-zero rows from RREF(A)
	cData := make([][]ast.Node, r)
	for row := 0; row < r; row++ {
		cData[row] = make([]ast.Node, n)
		for col := 0; col < n; col++ {
			cData[row][col] = rrefMat.Data[row][col]
		}
	}
	cMat, _ := ast.NewMatrix(r, n, cData)

	// Transposes
	btMat := evalTranspose(bMat) // r x m
	ctMat := evalTranspose(cMat) // n x r

	// Gram matrices
	gb, err := matMul(btMat, bMat) // r x r
	if err != nil {
		return nil, err
	}
	gc, err := matMul(cMat, ctMat) // r x r
	if err != nil {
		return nil, err
	}

	// Invert r x r Gram matrices
	invGb, err := evalInv(gb)
	if err != nil {
		return nil, err
	}
	invGc, err := evalInv(gc)
	if err != nil {
		return nil, err
	}

	// Associative order optimization:
	// A^+ = C^T * (invGc * invGb) * B^T
	mid, err := matMul(invGc, invGb) // r x r
	if err != nil {
		return nil, err
	}
	left, err := matMul(ctMat, mid) // n x r
	if err != nil {
		return nil, err
	}
	pinv, err := matMul(left, btMat) // n x m
	if err != nil {
		return nil, err
	}

	return pinv, nil
}

