package calc

import (
	"fmt"
)

// -------------------------------------------------------------------------
// Exact Matrix Calculations (det, inv, transpose, multiplication)
// -------------------------------------------------------------------------

func mulMatrixOrScalar(a, b Node) (Node, error) {
	matA, isMatA := a.(*MatrixNode)
	matB, isMatB := b.(*MatrixNode)

	if isMatA && isMatB {
		if matA.Cols != matB.Rows {
			return nil, fmt.Errorf("matrix dimension mismatch: cannot multiply %dx%d matrix by %dx%d matrix",
				matA.Rows, matA.Cols, matB.Rows, matB.Cols)
		}
		resData := make([][]Node, matA.Rows)
		for r := 0; r < matA.Rows; r++ {
			resData[r] = make([]Node, matB.Cols)
			for c := 0; c < matB.Cols; c++ {
				var dotTerms []Node
				for k := 0; k < matA.Cols; k++ {
					prod, err := simplifyMul([]Node{matA.Data[r][k], matB.Data[k][c]})
					if err != nil {
						return nil, err
					}
					dotTerms = append(dotTerms, prod)
				}
				sum, err := simplifyAdd(dotTerms)
				if err != nil {
					return nil, err
				}
				resData[r][c] = sum
			}
		}
		return NewMatrix(matA.Rows, matB.Cols, resData)
	}

	if isMatA {
		// Matrix * Scalar
		resData := make([][]Node, matA.Rows)
		for r := 0; r < matA.Rows; r++ {
			resData[r] = make([]Node, matA.Cols)
			for c := 0; c < matA.Cols; c++ {
				prod, err := simplifyMul([]Node{matA.Data[r][c], b})
				if err != nil {
					return nil, err
				}
				resData[r][c] = prod
			}
		}
		return NewMatrix(matA.Rows, matA.Cols, resData)
	}

	if isMatB {
		// Scalar * Matrix
		resData := make([][]Node, matB.Rows)
		for r := 0; r < matB.Rows; r++ {
			resData[r] = make([]Node, matB.Cols)
			for c := 0; c < matB.Cols; c++ {
				prod, err := simplifyMul([]Node{a, matB.Data[r][c]})
				if err != nil {
					return nil, err
				}
				resData[r][c] = prod
			}
		}
		return NewMatrix(matB.Rows, matB.Cols, resData)
	}

	return simplifyMul([]Node{a, b})
}

// submatrix returns a copy of m without dropR row and dropC col.
func submatrix(m *MatrixNode, dropR, dropC int) *MatrixNode {
	var data [][]Node
	for r := 0; r < m.Rows; r++ {
		if r == dropR {
			continue
		}
		var row []Node
		for c := 0; c < m.Cols; c++ {
			if c == dropC {
				continue
			}
			row = append(row, m.Data[r][c])
		}
		data = append(data, row)
	}
	res, _ := NewMatrix(m.Rows-1, m.Cols-1, data)
	return res
}

// evalDet calculates the exact determinant using division-free Laplace expansion.
func evalDet(m *MatrixNode) (Node, error) {
	if m.Rows != m.Cols {
		return nil, fmt.Errorf("matrix dimension error: det requires square matrix, got %dx%d", m.Rows, m.Cols)
	}

	n := m.Rows
	if n == 1 {
		return m.Data[0][0], nil
	}

	if n == 2 {
		// ad - bc
		ad, err := simplifyMul([]Node{m.Data[0][0], m.Data[1][1]})
		if err != nil {
			return nil, err
		}
		bc, err := simplifyMul([]Node{m.Data[0][1], m.Data[1][0]})
		if err != nil {
			return nil, err
		}
		negBC, err := simplifyUnaryOp("-", bc)
		if err != nil {
			return nil, err
		}
		return simplifyAdd([]Node{ad, negBC})
	}

	// n >= 3: Laplace expansion along row 0
	var sumTerms []Node
	for c := 0; c < n; c++ {
		elem := m.Data[0][c]
		if isZero(elem) {
			continue
		}
		sub := submatrix(m, 0, c)
		subDet, err := evalDet(sub)
		if err != nil {
			return nil, err
		}
		term, err := simplifyMul([]Node{elem, subDet})
		if err != nil {
			return nil, err
		}
		if c%2 == 1 {
			negTerm, err := simplifyUnaryOp("-", term)
			if err != nil {
				return nil, err
			}
			sumTerms = append(sumTerms, negTerm)
		} else {
			sumTerms = append(sumTerms, term)
		}
	}

	if len(sumTerms) == 0 {
		return mustRational(0, 1), nil
	}
	return simplifyAdd(sumTerms)
}

// evalInv calculates the exact matrix inverse using the adjugate matrix method.
func evalInv(m *MatrixNode) (*MatrixNode, error) {
	if m.Rows != m.Cols {
		return nil, fmt.Errorf("matrix dimension error: inv requires square matrix, got %dx%d", m.Rows, m.Cols)
	}

	d, err := evalDet(m)
	if err != nil {
		return nil, err
	}
	if isZero(d) {
		return nil, fmt.Errorf("math error: singular matrix (det = 0), inverse does not exist")
	}

	invDet, err := simplifyPow(d, mustRational(-1, 1))
	if err != nil {
		return nil, err
	}

	n := m.Rows
	if n == 1 {
		invElem, err := simplifyPow(m.Data[0][0], mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		return NewMatrix(1, 1, [][]Node{{invElem}})
	}

	// Adjugate matrix: adj(A)[r][c] = (-1)^(r+c) * det(submatrix(m, c, r)) (note transposed c, r!)
	invData := make([][]Node, n)
	for r := 0; r < n; r++ {
		invData[r] = make([]Node, n)
		for c := 0; c < n; c++ {
			sub := submatrix(m, c, r)
			subDet, err := evalDet(sub)
			if err != nil {
				return nil, err
			}
			var cofactor Node
			if (r+c)%2 == 1 {
				cofactor, err = simplifyUnaryOp("-", subDet)
				if err != nil {
					return nil, err
				}
			} else {
				cofactor = subDet
			}

			// elem = cofactor * (1/det)
			elem, err := simplifyMul([]Node{cofactor, invDet})
			if err != nil {
				return nil, err
			}
			invData[r][c] = elem
		}
	}

	return NewMatrix(n, n, invData)
}

// evalTranspose returns the transpose of matrix m.
func evalTranspose(m *MatrixNode) *MatrixNode {
	tData := make([][]Node, m.Cols)
	for c := 0; c < m.Cols; c++ {
		tData[c] = make([]Node, m.Rows)
		for r := 0; r < m.Rows; r++ {
			tData[c][r] = m.Data[r][c]
		}
	}
	res, _ := NewMatrix(m.Cols, m.Rows, tData)
	return res
}
