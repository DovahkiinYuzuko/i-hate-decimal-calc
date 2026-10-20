package lp

import (
	"fmt"
	"math/big"
)

// Tableau represents the simplex dictionary in standard canonical form.
// Layout:
// Rows 0 .. m-1: constraint rows
// Row m: objective row (reduced costs and current objective value)
//
// Columns 0 .. n-1: variable columns
// Column n: RHS (constants b)
type Tableau struct {
	Rows       int        // total rows (m constraints + 1 objective row)
	Cols       int        // total cols (n variables + 1 RHS col)
	Data       [][]*big.Rat
	Basis      []int      // Basis[i] is the variable index basic in row i (0 <= i < m)
	VarNames   []string   // names for each variable column (0 .. n-1)
	IsBasic    []bool     // IsBasic[j] true if var j is in basis

	// Scratch variables to avoid GC thrashing in inner loops
	scratch1   *big.Rat
	scratch2   *big.Rat
	scratch3   *big.Rat
}

// NewTableau allocates a fresh tableau of size rows x cols with independent big.Rat cells.
func NewTableau(rows, cols int, varNames []string) *Tableau {
	data := make([][]*big.Rat, rows)
	for i := 0; i < rows; i++ {
		data[i] = make([]*big.Rat, cols)
		for j := 0; j < cols; j++ {
			data[i][j] = new(big.Rat)
		}
	}
	names := make([]string, cols-1)
	if varNames != nil && len(varNames) >= cols-1 {
		copy(names, varNames[:cols-1])
	} else {
		for j := 0; j < cols-1; j++ {
			if j < len(varNames) {
				names[j] = varNames[j]
			} else {
				names[j] = fmt.Sprintf("v%d", j)
			}
		}
	}
	return &Tableau{
		Rows:     rows,
		Cols:     cols,
		Data:     data,
		Basis:    make([]int, rows-1),
		VarNames: names,
		IsBasic:  make([]bool, cols-1),
		scratch1: new(big.Rat),
		scratch2: new(big.Rat),
		scratch3: new(big.Rat),
	}
}

// Clone creates an exact deep-copy of the tableau with independent cells.
func (t *Tableau) Clone() *Tableau {
	cp := NewTableau(t.Rows, t.Cols, t.VarNames)
	for i := 0; i < t.Rows; i++ {
		for j := 0; j < t.Cols; j++ {
			cp.Data[i][j].Set(t.Data[i][j])
		}
	}
	copy(cp.Basis, t.Basis)
	copy(cp.IsBasic, t.IsBasic)
	return cp
}

// GetCell returns a read-only pointer to cell (r, c).
func (t *Tableau) GetCell(r, c int) *big.Rat {
	return t.Data[r][c]
}

// SetCell sets cell (r, c) to val.
func (t *Tableau) SetCell(r, c int, val *big.Rat) {
	if val == nil {
		t.Data[r][c].SetInt64(0)
	} else {
		t.Data[r][c].Set(val)
	}
}

// RHS returns cell (r, t.Cols - 1).
func (t *Tableau) RHS(r int) *big.Rat {
	return t.Data[r][t.Cols-1]
}

// SetRHS sets RHS of row r.
func (t *Tableau) SetRHS(r int, val *big.Rat) {
	t.SetCell(r, t.Cols-1, val)
}

// Pivot performs a Gauss-Jordan elimination step on pivot element (pivotRow, pivotCol).
func (t *Tableau) Pivot(pivotRow, pivotCol int) error {
	pivotVal := t.Data[pivotRow][pivotCol]
	if pivotVal.Sign() == 0 {
		return fmt.Errorf("cannot pivot on zero element at (%d, %d)", pivotRow, pivotCol)
	}

	// 1. Scale pivot row so that Data[pivotRow][pivotCol] == 1
	invPivot := t.scratch1.Inv(pivotVal)
	for j := 0; j < t.Cols; j++ {
		t.Data[pivotRow][j].Mul(t.Data[pivotRow][j], invPivot)
	}

	// 2. Eliminate all other rows
	for i := 0; i < t.Rows; i++ {
		if i == pivotRow {
			continue
		}
		factor := t.Data[i][pivotCol]
		if factor.Sign() == 0 {
			continue
		}
		// factorVal is copied to scratch2 to avoid aliasing when modifying row i
		t.scratch2.Set(factor)
		for j := 0; j < t.Cols; j++ {
			// Data[i][j] -= factor * Data[pivotRow][j]
			t.scratch3.Mul(t.scratch2, t.Data[pivotRow][j])
			t.Data[i][j].Sub(t.Data[i][j], t.scratch3)
		}
	}

	// 3. Update Basis
	leavingVar := t.Basis[pivotRow]
	t.IsBasic[leavingVar] = false
	t.Basis[pivotRow] = pivotCol
	t.IsBasic[pivotCol] = true

	return nil
}

// FindEnteringVarBland selects entering variable using Bland's minimal index rule.
// Variables with isExcluded[j] == true are skipped (e.g. artificial variables in Phase II).
func (t *Tableau) FindEnteringVarBland(objRow int, isExcluded []bool) int {
	for j := 0; j < t.Cols-1; j++ {
		if isExcluded != nil && j < len(isExcluded) && isExcluded[j] {
			continue
		}
		if !t.IsBasic[j] && t.Data[objRow][j].Sign() < 0 {
			return j
		}
	}
	return -1 // Optimal
}

// FindLeavingVarBland selects leaving row using the minimum ratio test with Bland's minimal index tie-breaking.
// Bland's rule: If multiple rows have the same minimum ratio RHS_i / A_i,j, choose the row whose current basic variable
// index Basis[i] is minimal.
func (t *Tableau) FindLeavingVarBland(pivotCol int, numConstraintRows int) (int, bool) {
	bestRow := -1
	bestRatio := new(big.Rat)
	bestBasisVar := -1

	ratio := t.scratch1

	for i := 0; i < numConstraintRows; i++ {
		coeff := t.Data[i][pivotCol]
		// Must have strictly positive coefficient in pivot column to bound the step
		if coeff.Sign() <= 0 {
			continue
		}

		// ratio = RHS / coeff
		rhs := t.Data[i][t.Cols-1]
		ratio.Quo(rhs, coeff)

		// If ratio is negative, this is degenerate or numerical inconsistency, but ratio >= 0 for feasible BFS
		if ratio.Sign() < 0 {
			continue
		}

		if bestRow == -1 {
			bestRow = i
			bestRatio.Set(ratio)
			bestBasisVar = t.Basis[i]
		} else {
			cmp := ratio.Cmp(bestRatio)
			if cmp < 0 {
				bestRow = i
				bestRatio.Set(ratio)
				bestBasisVar = t.Basis[i]
			} else if cmp == 0 {
				// Tie-breaking by Bland's rule: choose smallest basic variable index
				if t.Basis[i] < bestBasisVar {
					bestRow = i
					bestBasisVar = t.Basis[i]
				}
			}
		}
	}

	if bestRow == -1 {
		return -1, false // Unbounded
	}
	return bestRow, true
}

// DriveOutArtificials checks if any artificial variable remains basic at value 0.
// If found, it attempts to pivot out using any non-zero coefficient in a non-artificial column.
// If all coefficients in that row for non-artificial columns are zero, the row is redundant.
func (t *Tableau) DriveOutArtificials(isArtificial []bool, numConstraintRows int) ([]int, error) {
	var redundantRows []int
	for i := 0; i < numConstraintRows; i++ {
		bVar := t.Basis[i]
		if isArtificial[bVar] {
			// Check if we can pivot out
			pivoted := false
			for j := 0; j < t.Cols-1; j++ {
				if !isArtificial[j] && t.Data[i][j].Sign() != 0 {
					// Pivot on (i, j) to replace artificial variable
					if err := t.Pivot(i, j); err != nil {
						return nil, err
					}
					pivoted = true
					break
				}
			}
			if !pivoted {
				// The row is completely redundant (0 * x_non_art = 0)
				redundantRows = append(redundantRows, i)
			}
		}
	}
	return redundantRows, nil
}

// DropRows removes the specified constraint rows from the tableau.
func (t *Tableau) DropRows(rowsToDrop []int) {
	if len(rowsToDrop) == 0 {
		return
	}
	dropSet := make(map[int]bool)
	for _, r := range rowsToDrop {
		dropSet[r] = true
	}

	var newTableauData [][]*big.Rat
	var newBasis []int

	for i := 0; i < t.Rows; i++ {
		if dropSet[i] {
			// Drop this row
			continue
		}
		newTableauData = append(newTableauData, t.Data[i])
		if i < t.Rows-1 {
			newBasis = append(newBasis, t.Basis[i])
		}
	}

	t.Data = newTableauData
	t.Basis = newBasis
	t.Rows = len(newTableauData)
}
