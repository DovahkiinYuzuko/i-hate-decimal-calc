package lp

import (
	"fmt"
	"math/big"
)

// FSMState represents state in the simplex solver state machine.
type FSMState int

const (
	StateInit FSMState = iota
	StatePhase1
	StatePhase1Feasible
	StatePhase1Infeasible
	StatePhase2
	StateOptimal
	StateUnbounded
)

func (s FSMState) String() string {
	switch s {
	case StateInit:
		return "INIT"
	case StatePhase1:
		return "PHASE1"
	case StatePhase1Feasible:
		return "PHASE1_FEASIBLE"
	case StatePhase1Infeasible:
		return "PHASE1_INFEASIBLE"
	case StatePhase2:
		return "PHASE2"
	case StateOptimal:
		return "OPTIMAL"
	case StateUnbounded:
		return "UNBOUNDED"
	default:
		return "UNKNOWN"
	}
}

// SimplexFSM manages the state machine and exact rational execution of the Two-Phase Revised Simplex algorithm.
type SimplexFSM struct {
	State        FSMState
	Problem      *Problem
	Tableau      *Tableau
	NumOrigVars  int
	NumOrigCons  int
	PivotHistory []PivotStep
	Iteration    int
	MaxIter      int

	// Column mapping metadata
	IsArtificial   []bool
	SlackColForCon []int // constraint index -> slack/surplus column index in tableau
	ArtColForCon   []int // constraint index -> artificial column index in tableau
	RedundantCon   map[int]bool
	OrigRowScale   []*big.Rat // scale factor (+1 or -1) applied to normalize RHS >= 0
}

// Solve is the public entry point for the simplex solver.
func Solve(origProb *Problem) (*Result, error) {
	if err := origProb.Validate(); err != nil {
		return nil, fmt.Errorf("invalid LP problem: %w", err)
	}

	fsm := NewSimplexFSM(origProb)
	return fsm.Run()
}

// NewSimplexFSM initializes the simplex state machine.
func NewSimplexFSM(prob *Problem) *SimplexFSM {
	return &SimplexFSM{
		State:          StateInit,
		Problem:        prob.Clone(),
		NumOrigVars:    len(prob.VarNames),
		NumOrigCons:    len(prob.Constraints),
		MaxIter:        10000,
		OrigRowScale:   make([]*big.Rat, len(prob.Constraints)),
		SlackColForCon: make([]int, len(prob.Constraints)),
		ArtColForCon:   make([]int, len(prob.Constraints)),
		RedundantCon:   make(map[int]bool),
	}
}

// Run executes state transitions until reaching a terminal state.
func (fsm *SimplexFSM) Run() (*Result, error) {
	for {
		switch fsm.State {
		case StateInit:
			if err := fsm.transitionInit(); err != nil {
				return nil, err
			}
		case StatePhase1:
			if err := fsm.stepPhase1(); err != nil {
				return nil, err
			}
		case StatePhase1Feasible:
			if err := fsm.transitionPhase1Feasible(); err != nil {
				return nil, err
			}
		case StatePhase1Infeasible:
			return fsm.buildInfeasibleResult()
		case StatePhase2:
			if err := fsm.stepPhase2(); err != nil {
				return nil, err
			}
		case StateOptimal:
			return fsm.buildOptimalResult()
		case StateUnbounded:
			return fsm.buildUnboundedResult()
		default:
			return nil, fmt.Errorf("invalid FSM state: %v", fsm.State)
		}
	}
}

// transitionInit converts the problem into standard form and prepares initial tableau.
func (fsm *SimplexFSM) transitionInit() error {
	m := fsm.NumOrigCons
	n := fsm.NumOrigVars

	// Step 1: Normalize RHS >= 0 by multiplying -1 if RHS < 0
	for i := 0; i < m; i++ {
		c := &fsm.Problem.Constraints[i]
		if c.RHS.Sign() < 0 {
			fsm.OrigRowScale[i] = big.NewRat(-1, 1)
			c.RHS.Neg(c.RHS)
			for j := 0; j < n; j++ {
				c.Coeffs[j].Neg(c.Coeffs[j])
			}
			if c.Sense == RelLe {
				c.Sense = RelGe
			} else if c.Sense == RelGe {
				c.Sense = RelLe
			}
		} else {
			fsm.OrigRowScale[i] = big.NewRat(1, 1)
		}
	}

	// Step 2: Determine auxiliary variables needed (slack, surplus, artificial)
	numSlack := 0
	numArtificial := 0
	needsPhase1 := false

	for _, c := range fsm.Problem.Constraints {
		switch c.Sense {
		case RelLe:
			numSlack++
		case RelGe:
			numSlack++ // surplus variable (-1)
			numArtificial++
			needsPhase1 = true
		case RelEq:
			numArtificial++
			needsPhase1 = true
		}
	}

	totalCols := n + numSlack + numArtificial + 1 // +1 for RHS
	totalRows := m + 1                           // m constraints + 1 obj row
	if needsPhase1 {
		totalRows = m + 2 // +1 Phase II obj row, +1 Phase I obj row
	}

	varNames := make([]string, totalCols-1)
	copy(varNames, fsm.Problem.VarNames)

	t := NewTableau(totalRows, totalCols, varNames)
	fsm.Tableau = t
	fsm.IsArtificial = make([]bool, totalCols-1)
	fsm.SlackColForCon = make([]int, m)

	currSlackCol := n
	currArtCol := n + numSlack

	// Fill constraint rows
	for i, c := range fsm.Problem.Constraints {
		// Decision variable coefficients
		for j := 0; j < n; j++ {
			t.SetCell(i, j, c.Coeffs[j])
		}
		// Auxiliary variables
		switch c.Sense {
		case RelLe:
			// Add slack variable +1 * s_i
			t.SetCell(i, currSlackCol, big.NewRat(1, 1))
			t.VarNames[currSlackCol] = fmt.Sprintf("s%d", i+1)
			t.Basis[i] = currSlackCol
			t.IsBasic[currSlackCol] = true
			fsm.SlackColForCon[i] = currSlackCol
			fsm.ArtColForCon[i] = -1
			currSlackCol++
		case RelGe:
			// Add surplus variable -1 * e_i
			t.SetCell(i, currSlackCol, big.NewRat(-1, 1))
			t.VarNames[currSlackCol] = fmt.Sprintf("e%d", i+1)
			fsm.SlackColForCon[i] = currSlackCol
			currSlackCol++
			// Add artificial variable +1 * a_i
			t.SetCell(i, currArtCol, big.NewRat(1, 1))
			t.VarNames[currArtCol] = fmt.Sprintf("a%d", i+1)
			fsm.IsArtificial[currArtCol] = true
			t.Basis[i] = currArtCol
			t.IsBasic[currArtCol] = true
			fsm.ArtColForCon[i] = currArtCol
			currArtCol++
		case RelEq:
			// Add artificial variable +1 * a_i
			t.SetCell(i, currArtCol, big.NewRat(1, 1))
			t.VarNames[currArtCol] = fmt.Sprintf("a%d", i+1)
			fsm.IsArtificial[currArtCol] = true
			t.Basis[i] = currArtCol
			t.IsBasic[currArtCol] = true
			fsm.SlackColForCon[i] = -1 // equality has no slack
			fsm.ArtColForCon[i] = currArtCol
			currArtCol++
		}
		// RHS
		t.SetRHS(i, c.RHS)
	}

	// Setup Objective Row(s)
	// Phase II objective: Maximize c^T x (or Maximize -c^T x if minimization)
	phase2ObjRow := m
	for j := 0; j < n; j++ {
		coeff := new(big.Rat).Set(fsm.Problem.Objective[j])
		if fsm.Problem.Sense == OptMinimize {
			// Minimizing c^T x is maximizing -c^T x
			coeff.Neg(coeff)
		}
		// Tableau stores -c_j in objective row
		t.SetCell(phase2ObjRow, j, new(big.Rat).Neg(coeff))
	}
	if fsm.Problem.ObjConstant != nil {
		cst := new(big.Rat).Set(fsm.Problem.ObjConstant)
		if fsm.Problem.Sense == OptMinimize {
			cst.Neg(cst)
		}
		t.SetRHS(phase2ObjRow, cst)
	}

	if needsPhase1 {
		// Phase I: Minimize sum(a_i) <=> Maximize -sum(a_i)
		// W = -sum(a_i)
		// Initially, row m+1 has +1 for each artificial variable: W + sum(a_i) = 0
		phase1ObjRow := m + 1
		for j := 0; j < totalCols-1; j++ {
			if fsm.IsArtificial[j] {
				t.SetCell(phase1ObjRow, j, big.NewRat(1, 1))
			}
		}
		// Eliminate artificial variables from Phase I objective row to achieve canonical form:
		// row(m+1) -= sum(row_i for rows where artificial is basic)
		for i := 0; i < m; i++ {
			if fsm.IsArtificial[t.Basis[i]] {
				for j := 0; j < totalCols; j++ {
					t.Data[phase1ObjRow][j].Sub(t.Data[phase1ObjRow][j], t.Data[i][j])
				}
			}
		}
		fsm.State = StatePhase1
	} else {
		fsm.State = StatePhase2
	}

	return nil
}

// stepPhase1 performs one or more pivot steps for Phase I until optimal or infeasible.
func (fsm *SimplexFSM) stepPhase1() error {
	m := fsm.NumOrigCons
	phase1ObjRow := m + 1

	for {
		fsm.Iteration++
		if fsm.Iteration > fsm.MaxIter {
			return fmt.Errorf("simplex iteration limit exceeded in Phase I (%d iterations)", fsm.MaxIter)
		}

		// Find entering column using Bland's rule (smallest index with negative reduced cost)
		enteringCol := fsm.Tableau.FindEnteringVarBland(phase1ObjRow, nil)
		if enteringCol == -1 {
			// Phase I optimal reached
			break
		}

		leavingRow, bounded := fsm.Tableau.FindLeavingVarBland(enteringCol, m)
		if !bounded {
			// Phase I should never be unbounded (sum of artificials >= 0)
			return fmt.Errorf("unexpected unbounded state in Phase I auxiliary problem")
		}

		// Record step
		fsm.recordPivot(leavingRow, enteringCol, 1, fsm.Tableau.RHS(phase1ObjRow))

		if err := fsm.Tableau.Pivot(leavingRow, enteringCol); err != nil {
			return fmt.Errorf("pivot error in Phase I: %w", err)
		}
	}

	// Check Phase I objective value
	// In canonical tableau, RHS of phase1ObjRow stores current value of W - (-sum a_i)
	// Since W = -sum a_i <= 0, optimal value = 0 if and only if RHS == 0.
	objVal := fsm.Tableau.RHS(phase1ObjRow)
	if objVal.Sign() < 0 {
		// Sum of artificials > 0 => Infeasible
		fsm.State = StatePhase1Infeasible
	} else {
		// Sum of artificials == 0 => Feasible!
		fsm.State = StatePhase1Feasible
	}
	return nil
}

// transitionPhase1Feasible eliminates artificial variables and transitions to Phase II.
func (fsm *SimplexFSM) transitionPhase1Feasible() error {
	m := fsm.NumOrigCons

	// 1. Drive out any artificial variables still in basis at level 0
	redundantRows, err := fsm.Tableau.DriveOutArtificials(fsm.IsArtificial, m)
	if err != nil {
		return fmt.Errorf("error driving out artificial variables: %w", err)
	}

	for _, r := range redundantRows {
		fsm.RedundantCon[r] = true
	}

	// 2. Drop redundant rows from tableau
	if len(redundantRows) > 0 {
		fsm.Tableau.DropRows(redundantRows)
		m = fsm.Tableau.Rows - 2 // Phase I row still present
	}

	// 3. Drop Phase I objective row
	fsm.Tableau.Rows--
	fsm.Tableau.Data = fsm.Tableau.Data[:fsm.Tableau.Rows]
	if len(fsm.Tableau.Basis) > fsm.Tableau.Rows-1 {
		fsm.Tableau.Basis = fsm.Tableau.Basis[:fsm.Tableau.Rows-1]
	}

	// 4. Ensure Phase II objective row is canonical with respect to current basis
	// Phase II objective row is at index m (which is now Rows - 1)
	phase2ObjRow := fsm.Tableau.Rows - 1
	for i := 0; i < len(fsm.Tableau.Basis); i++ {
		bVar := fsm.Tableau.Basis[i]
		coeff := fsm.Tableau.Data[phase2ObjRow][bVar]
		if coeff.Sign() != 0 {
			factor := new(big.Rat).Set(coeff)
			for j := 0; j < fsm.Tableau.Cols; j++ {
				term := new(big.Rat).Mul(factor, fsm.Tableau.Data[i][j])
				fsm.Tableau.Data[phase2ObjRow][j].Sub(fsm.Tableau.Data[phase2ObjRow][j], term)
			}
		}
	}

	fsm.State = StatePhase2
	return nil
}

// stepPhase2 performs pivot steps for Phase II until reaching optimal or unbounded.
func (fsm *SimplexFSM) stepPhase2() error {
	objRow := fsm.Tableau.Rows - 1
	numConsRows := fsm.Tableau.Rows - 1

	for {
		fsm.Iteration++
		if fsm.Iteration > fsm.MaxIter {
			return fmt.Errorf("simplex iteration limit exceeded in Phase II (%d iterations)", fsm.MaxIter)
		}

		enteringCol := fsm.Tableau.FindEnteringVarBland(objRow, fsm.IsArtificial)
		if enteringCol == -1 {
			// Optimal reached!
			fsm.State = StateOptimal
			return nil
		}

		leavingRow, bounded := fsm.Tableau.FindLeavingVarBland(enteringCol, numConsRows)
		if !bounded {
			// Unbounded along enteringCol!
			fsm.State = StateUnbounded
			return nil
		}

		// Record step
		fsm.recordPivot(leavingRow, enteringCol, 2, fsm.Tableau.RHS(objRow))

		if err := fsm.Tableau.Pivot(leavingRow, enteringCol); err != nil {
			return fmt.Errorf("pivot error in Phase II: %w", err)
		}
	}
}

func (fsm *SimplexFSM) recordPivot(row, col, phase int, objVal *big.Rat) {
	entering := fsm.Tableau.VarNames[col]
	leaving := fsm.Tableau.VarNames[fsm.Tableau.Basis[row]]
	val := new(big.Rat)
	if objVal != nil {
		val.Set(objVal)
	}
	fsm.PivotHistory = append(fsm.PivotHistory, PivotStep{
		Iteration:   fsm.Iteration,
		EnteringVar: entering,
		LeavingVar:  leaving,
		PivotRow:    row,
		PivotCol:    col,
		ObjVal:      val,
		Phase:       phase,
	})
}

// buildOptimalResult extracts primal solution, dual solution, and certificates.
func (fsm *SimplexFSM) buildOptimalResult() (*Result, error) {
	n := fsm.NumOrigVars
	m := fsm.NumOrigCons
	t := fsm.Tableau
	objRow := t.Rows - 1

	// Primal solution x*
	primalMap := make(map[string]*big.Rat)
	primalVec := make([]*big.Rat, n)
	for j := 0; j < n; j++ {
		primalVec[j] = new(big.Rat)
		primalMap[fsm.Problem.VarNames[j]] = primalVec[j]
	}

	for i := 0; i < t.Rows-1; i++ {
		bVar := t.Basis[i]
		if bVar < n {
			val := new(big.Rat).Set(t.RHS(i))
			primalVec[bVar].Set(val)
			primalMap[fsm.Problem.VarNames[bVar]].Set(val)
		}
	}

	// Optimal objective value
	// In maximization: Z = RHS(objRow)
	// In minimization: original Z = -RHS(objRow)
	optVal := new(big.Rat).Set(t.RHS(objRow))
	if fsm.Problem.Sense == OptMinimize {
		optVal.Neg(optVal)
	}

	// Extract dual variables y*
	dualVec := make([]*big.Rat, m)
	dualMap := make(map[string]*big.Rat)
	slackMap := make(map[string]*big.Rat)

	for i := 0; i < m; i++ {
		dualVec[i] = new(big.Rat)
		conName := fsm.Problem.Constraints[i].Name
		if conName == "" {
			conName = fmt.Sprintf("c%d", i+1)
		}

		if fsm.RedundantCon[i] {
			// Redundant constraint has zero shadow price
			dualVec[i].SetInt64(0)
		} else {
			targetCol := -1
			isEq := (fsm.Problem.Constraints[i].Sense == RelEq)
			if isEq {
				targetCol = fsm.ArtColForCon[i]
			} else {
				targetCol = fsm.SlackColForCon[i]
			}

			if targetCol != -1 && targetCol < t.Cols-1 {
				redCost := t.Data[objRow][targetCol]
				val := new(big.Rat).Set(redCost)

				// For normalized RelGe, the surplus column had initial coefficient -1 in row i
				if fsm.Problem.Constraints[i].Sense == RelGe {
					val.Neg(val)
				}

				// Scale by OrigRowScale[i] (+1 or -1) to match original constraint before RHS normalization
				val.Mul(val, fsm.OrigRowScale[i])

				if fsm.Problem.Sense == OptMinimize {
					val.Neg(val)
				}
				dualVec[i].Set(val)
			}
		}
		dualMap[conName] = dualVec[i]
	}

	cert := &Certificate{
		PrimalPoint: primalVec,
		DualVector:  dualVec,
		DualValues:  dualMap,
		SlackValues: slackMap,
	}

	return &Result{
		Status:         StatusOptimal,
		OptimalValue:   optVal,
		PrimalSolution: primalMap,
		Certificate:    cert,
		PivotHistory:   fsm.PivotHistory,
	}, nil
}

// buildInfeasibleResult extracts Farkas infeasibility certificate y such that y^T A >= 0, y^T b < 0.
func (fsm *SimplexFSM) buildInfeasibleResult() (*Result, error) {
	m := fsm.NumOrigCons
	phase1ObjRow := m + 1
	t := fsm.Tableau

	farkasVec := make([]*big.Rat, m)
	for i := 0; i < m; i++ {
		farkasVec[i] = new(big.Rat)
		targetCol := fsm.SlackColForCon[i]
		if targetCol == -1 {
			targetCol = fsm.ArtColForCon[i]
		}
		if targetCol != -1 && targetCol < t.Cols-1 {
			// Extract exact row multiplier w_i of normalized row i in Phase I objective row, then scale by OrigRowScale[i].
			val := new(big.Rat).Set(t.Data[phase1ObjRow][targetCol])
			switch fsm.Problem.Constraints[i].Sense {
			case RelGe:
				val.Neg(val)
			case RelEq:
				val.Sub(val, big.NewRat(1, 1))
			}
			val.Mul(val, fsm.OrigRowScale[i])
			farkasVec[i].Set(val)
		}
	}

	cert := &Certificate{
		FarkasRay: farkasVec,
	}

	return &Result{
		Status:       StatusInfeasible,
		Certificate:  cert,
		PivotHistory: fsm.PivotHistory,
	}, nil
}

// buildUnboundedResult extracts feasible point and recession ray d.
func (fsm *SimplexFSM) buildUnboundedResult() (*Result, error) {
	n := fsm.NumOrigVars
	t := fsm.Tableau
	objRow := t.Rows - 1

	// Entering column that triggers unboundedness
	enteringCol := t.FindEnteringVarBland(objRow, fsm.IsArtificial)
	if enteringCol == -1 {
		return nil, fmt.Errorf("inconsistent unbounded state without entering column")
	}

	// Base point x0 (current BFS)
	x0 := make([]*big.Rat, n)
	for j := 0; j < n; j++ {
		x0[j] = new(big.Rat)
	}
	for i, bVar := range t.Basis {
		if bVar < n {
			x0[bVar].Set(t.RHS(i))
		}
	}

	// Ray direction d
	d := make([]*big.Rat, n)
	for j := 0; j < n; j++ {
		d[j] = new(big.Rat)
	}
	if enteringCol < n {
		d[enteringCol].SetInt64(1)
	}
	for i, bVar := range t.Basis {
		if bVar < n {
			coeff := new(big.Rat).Neg(t.Data[i][enteringCol])
			d[bVar].Set(coeff)
		}
	}

	cert := &Certificate{
		BasePoint:    x0,
		RecessionRay: d,
	}

	return &Result{
		Status:       StatusUnbounded,
		Certificate:  cert,
		PivotHistory: fsm.PivotHistory,
	}, nil
}
