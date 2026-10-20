package lp

import (
	"math/big"
	"testing"
)

func rat(n, d int64) *big.Rat {
	return big.NewRat(n, d)
}

func ratInt(n int64) *big.Rat {
	return big.NewRat(n, 1)
}

// TestSimplex_Easy2D verifies a basic 2-variable maximization problem:
// Maximize 2x + 3y
// s.t. x + y <= 4
//
//	x, y >= 0
//
// Optimal: x = 0, y = 4, z* = 12, dual: y_1 = 3
func TestSimplex_Easy2D(t *testing.T) {
	prob := &Problem{
		Sense: OptMaximize,
		Objective: []*big.Rat{
			ratInt(2),
			ratInt(3),
		},
		VarNames: []string{"x", "y"},
		Constraints: []Constraint{
			{
				Coeffs: []*big.Rat{ratInt(1), ratInt(1)},
				Sense:  RelLe,
				RHS:    ratInt(4),
				Name:   "c1",
			},
		},
	}

	res, err := Solve(prob)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if res.Status != StatusOptimal {
		t.Fatalf("expected StatusOptimal, got %v", res.Status)
	}

	if res.OptimalValue.Cmp(ratInt(12)) != 0 {
		t.Errorf("expected optimal value 12, got %s", res.OptimalValue.String())
	}

	if res.PrimalSolution["x"].Cmp(ratInt(0)) != 0 || res.PrimalSolution["y"].Cmp(ratInt(4)) != 0 {
		t.Errorf("expected x=0, y=4, got x=%s, y=%s", res.PrimalSolution["x"], res.PrimalSolution["y"])
	}

	// Verify internal sanitizer passes
	if err := SanitizeAndVerifyCertificate(prob, res.Certificate); err != nil {
		t.Errorf("certificate verification failed: %v", err)
	}
}

// TestSimplex_BealeCycling verifies that Bland's rule terminates on Beale's degenerate cycling example:
// Minimize -3/4 x1 + 150 x2 - 1/50 x3 + 6 x4
// s.t. 1/4 x1 - 60 x2 - 1/25 x3 + 9 x4 <= 0
//
//	1/2 x1 - 90 x2 - 1/50 x3 + 3 x4 <= 0
//	x3 <= 1
//	x >= 0
func TestSimplex_BealeCycling(t *testing.T) {
	prob := &Problem{
		Sense: OptMinimize,
		Objective: []*big.Rat{
			rat(-3, 4),
			ratInt(150),
			rat(-1, 50),
			ratInt(6),
		},
		VarNames: []string{"x1", "x2", "x3", "x4"},
		Constraints: []Constraint{
			{
				Coeffs: []*big.Rat{rat(1, 4), ratInt(-60), rat(-1, 25), ratInt(9)},
				Sense:  RelLe,
				RHS:    ratInt(0),
				Name:   "c1",
			},
			{
				Coeffs: []*big.Rat{rat(1, 2), ratInt(-90), rat(-1, 50), ratInt(3)},
				Sense:  RelLe,
				RHS:    ratInt(0),
				Name:   "c2",
			},
			{
				Coeffs: []*big.Rat{ratInt(0), ratInt(0), ratInt(1), ratInt(0)},
				Sense:  RelLe,
				RHS:    ratInt(1),
				Name:   "c3",
			},
		},
	}

	res, err := Solve(prob)
	if err != nil {
		t.Fatalf("Solve failed on Beale cycling example: %v", err)
	}

	if res.Status != StatusOptimal {
		t.Fatalf("expected StatusOptimal, got %v", res.Status)
	}

	// Beale's minimum is known to be -1/25 (or -0.04) at x1 = 1/25 / (1/50) = ...
	// Verify that it reached an optimal solution without looping forever
	if res.OptimalValue == nil {
		t.Fatalf("expected non-nil optimal value")
	}

	if err := SanitizeAndVerifyCertificate(prob, res.Certificate); err != nil {
		t.Errorf("certificate verification failed: %v", err)
	}
}

// TestSimplex_KleeMintyCube verifies performance on Klee-Minty 3D cube.
// Maximize 100 x1 + 10 x2 + x3
// s.t. x1 <= 1
//
//	20 x1 + x2 <= 100
//	200 x1 + 20 x2 + x3 <= 10000
//	x1, x2, x3 >= 0
//
// Optimal: x = (0, 0, 10000), z* = 10000
func TestSimplex_KleeMintyCube(t *testing.T) {
	prob := &Problem{
		Sense: OptMaximize,
		Objective: []*big.Rat{
			ratInt(100),
			ratInt(10),
			ratInt(1),
		},
		VarNames: []string{"x1", "x2", "x3"},
		Constraints: []Constraint{
			{
				Coeffs: []*big.Rat{ratInt(1), ratInt(0), ratInt(0)},
				Sense:  RelLe,
				RHS:    ratInt(1),
				Name:   "c1",
			},
			{
				Coeffs: []*big.Rat{ratInt(20), ratInt(1), ratInt(0)},
				Sense:  RelLe,
				RHS:    ratInt(100),
				Name:   "c2",
			},
			{
				Coeffs: []*big.Rat{ratInt(200), ratInt(20), ratInt(1)},
				Sense:  RelLe,
				RHS:    ratInt(10000),
				Name:   "c3",
			},
		},
	}

	res, err := Solve(prob)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if res.Status != StatusOptimal {
		t.Fatalf("expected StatusOptimal, got %v", res.Status)
	}

	if res.OptimalValue.Cmp(ratInt(10000)) != 0 {
		t.Errorf("expected optimal value 10000, got %s", res.OptimalValue.String())
	}

	if res.PrimalSolution["x3"].Cmp(ratInt(10000)) != 0 {
		t.Errorf("expected x3=10000, got %s", res.PrimalSolution["x3"])
	}

	if err := SanitizeAndVerifyCertificate(prob, res.Certificate); err != nil {
		t.Errorf("certificate verification failed: %v", err)
	}
}

// TestSimplex_Infeasible verifies detection of contradictory constraints and Farkas witness:
// Maximize x1 + x2
// s.t. x1 + x2 <= 2
//
//	2 x1 + 2 x2 >= 6  (i.e. -2 x1 - 2 x2 <= -6)
//	x1, x2 >= 0
func TestSimplex_Infeasible(t *testing.T) {
	prob := &Problem{
		Sense: OptMaximize,
		Objective: []*big.Rat{
			ratInt(1),
			ratInt(1),
		},
		VarNames: []string{"x1", "x2"},
		Constraints: []Constraint{
			{
				Coeffs: []*big.Rat{ratInt(1), ratInt(1)},
				Sense:  RelLe,
				RHS:    ratInt(2),
				Name:   "c1",
			},
			{
				Coeffs: []*big.Rat{ratInt(2), ratInt(2)},
				Sense:  RelGe,
				RHS:    ratInt(6),
				Name:   "c2",
			},
		},
	}

	res, err := Solve(prob)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if res.Status != StatusInfeasible {
		t.Fatalf("expected StatusInfeasible, got %v", res.Status)
	}

	if res.Certificate == nil || res.Certificate.FarkasRay == nil {
		t.Fatalf("expected non-nil Farkas certificate for infeasible LP")
	}

	if err := SanitizeAndVerifyCertificate(prob, res.Certificate); err != nil {
		t.Errorf("Farkas certificate failed verification: %v", err)
	}
}

// TestSimplex_Unbounded verifies detection of infinite ray:
// Maximize 2 x1 + x2
// s.t. x1 - x2 <= 2
//
//	x1 - 2 x2 <= 1
//	x1, x2 >= 0
func TestSimplex_Unbounded(t *testing.T) {
	prob := &Problem{
		Sense: OptMaximize,
		Objective: []*big.Rat{
			ratInt(2),
			ratInt(1),
		},
		VarNames: []string{"x1", "x2"},
		Constraints: []Constraint{
			{
				Coeffs: []*big.Rat{ratInt(1), ratInt(-1)},
				Sense:  RelLe,
				RHS:    ratInt(2),
				Name:   "c1",
			},
			{
				Coeffs: []*big.Rat{ratInt(1), ratInt(-2)},
				Sense:  RelLe,
				RHS:    ratInt(1),
				Name:   "c2",
			},
		},
	}

	res, err := Solve(prob)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if res.Status != StatusUnbounded {
		t.Fatalf("expected StatusUnbounded, got %v", res.Status)
	}

	if res.Certificate == nil || res.Certificate.RecessionRay == nil {
		t.Fatalf("expected non-nil recession ray for unbounded LP")
	}

	if err := SanitizeAndVerifyCertificate(prob, res.Certificate); err != nil {
		t.Errorf("Recession ray failed verification: %v", err)
	}
}

// TestSimplex_RedundantEquality verifies driving out artificial variables when constraints are dependent:
// Maximize x + y
// s.t. x + y = 4
//
//	2x + 2y = 8
//	x, y >= 0
func TestSimplex_RedundantEquality(t *testing.T) {
	prob := &Problem{
		Sense: OptMaximize,
		Objective: []*big.Rat{
			ratInt(1),
			ratInt(1),
		},
		VarNames: []string{"x", "y"},
		Constraints: []Constraint{
			{
				Coeffs: []*big.Rat{ratInt(1), ratInt(1)},
				Sense:  RelEq,
				RHS:    ratInt(4),
				Name:   "c1",
			},
			{
				Coeffs: []*big.Rat{ratInt(2), ratInt(2)},
				Sense:  RelEq,
				RHS:    ratInt(8),
				Name:   "c2",
			},
		},
	}

	res, err := Solve(prob)
	if err != nil {
		t.Fatalf("Solve failed on redundant equality: %v", err)
	}

	if res.Status != StatusOptimal {
		t.Fatalf("expected StatusOptimal, got %v", res.Status)
	}

	if res.OptimalValue.Cmp(ratInt(4)) != 0 {
		t.Errorf("expected optimal value 4, got %s", res.OptimalValue.String())
	}

	if err := SanitizeAndVerifyCertificate(prob, res.Certificate); err != nil {
		t.Errorf("certificate verification failed: %v", err)
	}
}

// TestSimplex_IllConditionedScale verifies exact rational precision on extreme scale disparities (10^-9 to 10^20):
// Maximize (1/1000000007) x1 + 10^20 x2
// s.t. 3/7 x1 + 10^20 x2 <= 10^20
//
//	x1 <= 10^9
//	x1, x2 >= 0
func TestSimplex_IllConditionedScale(t *testing.T) {
	c1_large := new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)
	c1_large_rat := new(big.Rat).SetInt(c1_large)

	x1_limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
	x1_limit_rat := new(big.Rat).SetInt(x1_limit)

	prob := &Problem{
		Sense: OptMaximize,
		Objective: []*big.Rat{
			rat(1, 1000000007),
			new(big.Rat).Set(c1_large_rat),
		},
		VarNames: []string{"x1", "x2"},
		Constraints: []Constraint{
			{
				Coeffs: []*big.Rat{rat(3, 7), new(big.Rat).Set(c1_large_rat)},
				Sense:  RelLe,
				RHS:    new(big.Rat).Set(c1_large_rat),
				Name:   "c1",
			},
			{
				Coeffs: []*big.Rat{ratInt(1), ratInt(0)},
				Sense:  RelLe,
				RHS:    new(big.Rat).Set(x1_limit_rat),
				Name:   "c2",
			},
		},
	}

	res, err := Solve(prob)
	if err != nil {
		t.Fatalf("Solve failed on ill-conditioned scale: %v", err)
	}

	if res.Status != StatusOptimal {
		t.Fatalf("expected StatusOptimal, got %v", res.Status)
	}

	if err := SanitizeAndVerifyCertificate(prob, res.Certificate); err != nil {
		t.Errorf("certificate verification failed on ill-conditioned scale: %v", err)
	}
}
