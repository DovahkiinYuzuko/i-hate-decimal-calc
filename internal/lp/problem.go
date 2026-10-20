package lp

import (
	"fmt"
	"math/big"
)

// OptSense represents optimization direction.
type OptSense int

const (
	OptMaximize OptSense = iota
	OptMinimize
)

// RelSense represents constraint relation.
type RelSense int

const (
	RelLe RelSense = iota // <=
	RelGe                 // >=
	RelEq                 // ==
)

func (r RelSense) String() string {
	switch r {
	case RelLe:
		return "<="
	case RelGe:
		return ">="
	case RelEq:
		return "=="
	default:
		return "?"
	}
}

// Status represents the termination outcome of the simplex solver.
type Status int

const (
	StatusOptimal Status = iota
	StatusInfeasible
	StatusUnbounded
)

func (s Status) String() string {
	switch s {
	case StatusOptimal:
		return "OPTIMAL"
	case StatusInfeasible:
		return "INFEASIBLE"
	case StatusUnbounded:
		return "UNBOUNDED"
	default:
		return "UNKNOWN"
	}
}

// Constraint represents a linear relation: sum(Coeffs[j] * x_j) (Sense) RHS.
type Constraint struct {
	Name   string
	Coeffs []*big.Rat
	Sense  RelSense
	RHS    *big.Rat
}

// Problem represents a linear programming problem.
type Problem struct {
	Sense       OptSense
	Objective   []*big.Rat
	ObjConstant *big.Rat // optional constant offset
	VarNames    []string
	Constraints []Constraint
}

// Certificate stores mathematically verifiable witnesses for Lean 4, verify, and explain.
type Certificate struct {
	// For StatusOptimal:
	PrimalPoint  []*big.Rat            // x* in original variable space
	DualVector   []*big.Rat            // y* in original constraint space
	DualValues   map[string]*big.Rat   // constraintName -> y_i*
	SlackValues  map[string]*big.Rat   // constraintName -> slack_i

	// For StatusInfeasible:
	FarkasRay    []*big.Rat            // y >= 0 such that y^T A >= 0 and y^T b < 0 (or direction-adjusted)

	// For StatusUnbounded:
	BasePoint    []*big.Rat            // x0 feasible
	RecessionRay []*big.Rat            // d >= 0 such that A d <= 0 and c^T d > 0

	// Provenance map: connects original constraint row to standard tableau rows
	RowMap       map[int]int
}

// Result represents the complete solution returned by Solve.
type Result struct {
	Status         Status
	OptimalValue   *big.Rat
	PrimalSolution map[string]*big.Rat
	Certificate    *Certificate
	PivotHistory   []PivotStep
}

// PivotStep records tableau state transitions for explain and provenance.
type PivotStep struct {
	Iteration    int
	EnteringVar  string
	LeavingVar   string
	PivotRow     int
	PivotCol     int
	ObjVal       *big.Rat
	Phase        int
}

func (p *Problem) Clone() *Problem {
	cp := &Problem{
		Sense:       p.Sense,
		Objective:   make([]*big.Rat, len(p.Objective)),
		VarNames:    append([]string(nil), p.VarNames...),
		Constraints: make([]Constraint, len(p.Constraints)),
	}
	if p.ObjConstant != nil {
		cp.ObjConstant = new(big.Rat).Set(p.ObjConstant)
	} else {
		cp.ObjConstant = new(big.Rat)
	}
	for i, c := range p.Objective {
		if c != nil {
			cp.Objective[i] = new(big.Rat).Set(c)
		} else {
			cp.Objective[i] = new(big.Rat)
		}
	}
	for i, c := range p.Constraints {
		coeffs := make([]*big.Rat, len(c.Coeffs))
		for j, a := range c.Coeffs {
			if a != nil {
				coeffs[j] = new(big.Rat).Set(a)
			} else {
				coeffs[j] = new(big.Rat)
			}
		}
		var rhs *big.Rat
		if c.RHS != nil {
			rhs = new(big.Rat).Set(c.RHS)
		} else {
			rhs = new(big.Rat)
		}
		cp.Constraints[i] = Constraint{
			Name:   c.Name,
			Coeffs: coeffs,
			Sense:  c.Sense,
			RHS:    rhs,
		}
	}
	return cp
}

func (p *Problem) Validate() error {
	numVars := len(p.VarNames)
	if numVars == 0 {
		return fmt.Errorf("LP problem has 0 variables")
	}
	if len(p.Objective) != numVars {
		return fmt.Errorf("objective coefficient count (%d) mismatch with variable count (%d)", len(p.Objective), numVars)
	}
	for i, c := range p.Constraints {
		if len(c.Coeffs) != numVars {
			return fmt.Errorf("constraint %d (%s) coefficient count (%d) mismatch with variable count (%d)", i, c.Name, len(c.Coeffs), numVars)
		}
		if c.RHS == nil {
			return fmt.Errorf("constraint %d (%s) RHS is nil", i, c.Name)
		}
	}
	return nil
}
