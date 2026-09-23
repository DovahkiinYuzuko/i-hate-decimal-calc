package calc

import (
	"strings"
	"testing"
)

func TestCadFSM_Transitions(t *testing.T) {
	fsm := NewCadLifecycleFSM()
	if fsm.CurrentState() != CadStateInit {
		t.Fatalf("expected state Init, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateNormalized); err != nil {
		t.Fatalf("failed to transition to Normalized: %v", err)
	}
	if fsm.CurrentState() != CadStateNormalized {
		t.Fatalf("expected state Normalized, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateProjected); err != nil {
		t.Fatalf("failed to transition to Projected: %v", err)
	}
	if fsm.CurrentState() != CadStateProjected {
		t.Fatalf("expected state Projected, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateValidated); err != nil {
		t.Fatalf("failed to transition to Validated: %v", err)
	}
	if fsm.CurrentState() != CadStateValidated {
		t.Fatalf("expected state Validated, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateSampled); err != nil {
		t.Fatalf("failed to transition to Sampled: %v", err)
	}
	if fsm.CurrentState() != CadStateSampled {
		t.Fatalf("expected state Sampled, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateLifted); err != nil {
		t.Fatalf("failed to transition to Lifted: %v", err)
	}
	if fsm.CurrentState() != CadStateLifted {
		t.Fatalf("expected state Lifted, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateDecided); err != nil {
		t.Fatalf("failed to transition to Decided: %v", err)
	}
	if fsm.CurrentState() != CadStateDecided {
		t.Fatalf("expected state Decided, got %v", fsm.CurrentState())
	}
}

func TestCad_1D_Linear(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("x - 3 > 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rel, ok := expr.(*RelOpNode)
	if !ok {
		t.Fatalf("expected RelOpNode, got %T", expr)
	}

	sol, err := SolveInequality(rel, "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	str := sol.String()
	if !strings.Contains(str, "3") || !strings.Contains(str, "inf") {
		t.Fatalf("expected interval [3, inf], got %s", str)
	}
}

func TestCad_1D_Quadratic_Strict(t *testing.T) {
	env := NewEnv()
	// x^2 - 4 < 0 -> interval (-2, 2)
	expr, err := Parse("x^2 - 4 < 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rel, ok := expr.(*RelOpNode)
	if !ok {
		t.Fatalf("expected RelOpNode, got %T", expr)
	}

	sol, err := SolveInequality(rel, "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	str := sol.String()
	if !strings.Contains(str, "-2") || !strings.Contains(str, "2") {
		t.Fatalf("expected interval with -2 and 2, got %s", str)
	}
}

func TestCad_1D_Quadratic_NonStrict_Radical(t *testing.T) {
	env := NewEnv()
	// x^2 - 2 <= 0 -> interval [-sqrt(2), sqrt(2)]
	expr, err := Parse("x^2 - 2 <= 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rel, ok := expr.(*RelOpNode)
	if !ok {
		t.Fatalf("expected RelOpNode, got %T", expr)
	}

	sol, err := SolveInequality(rel, "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	str := sol.String()
	if !strings.Contains(str, "2") {
		t.Fatalf("expected interval with sqrt(2), got %s", str)
	}
}

func TestCad_1D_RepeatedRoot_Singleton(t *testing.T) {
	env := NewEnv()
	// (x - 1)^2 <= 0 -> singleton [1, 1]
	expr, err := Parse("(x - 1)^2 <= 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rel, ok := expr.(*RelOpNode)
	if !ok {
		t.Fatalf("expected RelOpNode, got %T", expr)
	}

	sol, err := SolveInequality(rel, "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	str := sol.String()
	if !strings.Contains(str, "1") {
		t.Fatalf("expected solution containing 1, got %s", str)
	}
}

func TestCad_1D_Tautology_And_UNSAT(t *testing.T) {
	env := NewEnv()

	// x^2 + 1 > 0 -> true (Tautology)
	exprTrue, err := Parse("x^2 + 1 > 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	solTrue, err := SolveInequality(exprTrue.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if solTrue.String() != "true" {
		t.Fatalf("expected true, got %s", solTrue.String())
	}

	// x^2 + 1 < 0 -> false (UNSAT)
	exprFalse, err := Parse("x^2 + 1 < 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	solFalse, err := SolveInequality(exprFalse.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if solFalse.String() != "false" {
		t.Fatalf("expected false, got %s", solFalse.String())
	}
}

func TestCad_Solve_BuiltinIntegration(t *testing.T) {
	env := NewEnv()

	// solve(x^2 - 9 < 0, x) -> [[-3, 3]]
	expr, err := Parse("solve(x^2 - 9 < 0, x)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	str := evaled.String()
	if !strings.Contains(str, "-3") || !strings.Contains(str, "3") {
		t.Fatalf("expected interval [-3, 3], got %s", str)
	}
}

func TestCad_CADDecompose_Builtin(t *testing.T) {
	env := NewEnv()

	// cad([x^2 + y^2 - 1], [x, y])
	expr, err := Parse("cad([x^2 + y^2 - 1], [x, y])")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	list, ok := evaled.(*ListNode)
	if !ok || len(list.Elements) == 0 {
		t.Fatalf("expected non-empty list of samples, got %v", evaled)
	}
}

func TestDecompose1DCAD_Structure(t *testing.T) {
	env := NewEnv()

	// Polynomial with 2 real roots: x^2 - 4 = (x - 2)(x + 2)
	pNode, _ := Parse("x^2 - 4")
	p, ok := extractPoly(pNode, "x")
	if !ok {
		t.Fatalf("failed to extract poly")
	}

	cells, err := Decompose1DCAD([]*univariatePoly{p}, "x", env)
	if err != nil {
		t.Fatalf("Decompose1DCAD failed: %v", err)
	}

	// 2 roots => 2*2 + 1 = 5 cells
	if len(cells) != 5 {
		t.Fatalf("expected 5 cells (2m + 1), got %d", len(cells))
	}

	// Check alternating Sector (Dim 1, IsSection false) and Section (Dim 0, IsSection true)
	expectedTypes := []struct {
		dim       int
		isSection bool
	}{
		{dim: 1, isSection: false}, // (-inf, -2)
		{dim: 0, isSection: true},  // {-2}
		{dim: 1, isSection: false}, // (-2, 2)
		{dim: 0, isSection: true},  // {2}
		{dim: 1, isSection: false}, // (2, +inf)
	}

	for i, exp := range expectedTypes {
		if cells[i].Dimension != exp.dim {
			t.Errorf("cell %d: expected dim %d, got %d", i, exp.dim, cells[i].Dimension)
		}
		if cells[i].IsSection != exp.isSection {
			t.Errorf("cell %d: expected isSection %v, got %v", i, exp.isSection, cells[i].IsSection)
		}
		if len(cells[i].SamplePoint) != 1 {
			t.Errorf("cell %d: expected 1 sample point, got %d", i, len(cells[i].SamplePoint))
		}
	}
}

func TestDecompose1DCAD_CloseRoots(t *testing.T) {
	env := NewEnv()

	// Inequality with extremely close roots: (x - 1)*(x - 10001/10000) < 0
	// Distance between roots is 1/10000 = 0.0001, which broke old floatToRatSample (f * 1000).
	expr, err := Parse("(x - 1) * (x - 10001/10000) < 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	sol, err := SolveInequality(expr.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("SolveInequality failed: %v", err)
	}

	str := sol.String()
	if !strings.Contains(str, "1") || !strings.Contains(str, "10001/10000") {
		t.Fatalf("expected interval between 1 and 10001/10000, got: %s", str)
	}
}

func TestDecompose1DCAD_DegenerateAndEqualities(t *testing.T) {
	env := NewEnv()

	// 1. Double root isolated point: (x - 5)^2 <= 0 => [5, 5]
	exprDouble, _ := Parse("(x - 5)^2 <= 0")
	solDouble, err := SolveInequality(exprDouble.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if !strings.Contains(solDouble.String(), "5") {
		t.Fatalf("expected singleton with 5, got %s", solDouble.String())
	}

	// 2. Strict inequality with double root: (x - 5)^2 < 0 => false
	exprStrict, _ := Parse("(x - 5)^2 < 0")
	solStrict, err := SolveInequality(exprStrict.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if solStrict.String() != "false" {
		t.Fatalf("expected false, got %s", solStrict.String())
	}

	// 3. No real roots tautology: x^2 + 4 >= 0 => true
	exprTautology, _ := Parse("x^2 + 4 >= 0")
	solTautology, err := SolveInequality(exprTautology.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if solTautology.String() != "true" {
		t.Fatalf("expected true, got %s", solTautology.String())
	}
}

func TestCAD_RecursiveLifting_2D_Circle(t *testing.T) {
	env := NewEnv()
	circleNode, err := Parse("x^2 + y^2 - 1")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	cells, err := CADDecomposeCells([]Node{circleNode}, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADDecomposeCells failed: %v", err)
	}

	// For circle x^2 + y^2 - 1:
	// Base x has 2 roots (-1, 1) -> 5 cells.
	// Over x < -1: y^2 + positive -> no real roots -> 1 sector
	// Over x = -1: y^2 = 0 -> 1 root (0) -> 3 cells (sector, section, sector)
	// Over -1 < x < 1 (e.g. x=0): y^2 - 1 = 0 -> 2 roots (-1, 1) -> 5 cells
	// Over x = 1: y^2 = 0 -> 1 root (0) -> 3 cells
	// Over x > 1: y^2 + positive -> no real roots -> 1 sector
	// Total cells: 1 + 3 + 5 + 3 + 1 = 13 cells!
	if len(cells) != 13 {
		t.Fatalf("expected 13 cells for circle decomposition, got %d", len(cells))
	}

	foundInside := false
	foundBoundary := false
	foundOutside := false

	key := circleNode.String()
	for _, c := range cells {
		if len(c.SamplePoint) != 2 {
			t.Fatalf("expected 2D sample point, got %v", c.SamplePoint)
		}
		sign := c.SignVector[key]
		if sign < 0 {
			foundInside = true
			if c.Dimension != 2 {
				t.Errorf("interior cell should have dimension 2, got %d", c.Dimension)
			}
		} else if sign == 0 {
			foundBoundary = true
			if c.Dimension > 1 {
				t.Errorf("boundary cell should have dimension <= 1, got %d", c.Dimension)
			}
		} else if sign > 0 {
			foundOutside = true
		}
	}

	if !foundInside {
		t.Errorf("expected to find at least one cell strictly inside the circle (sign < 0)")
	}
	if !foundBoundary {
		t.Errorf("expected to find at least one cell on the circle boundary (sign == 0)")
	}
	if !foundOutside {
		t.Errorf("expected to find at least one cell strictly outside the circle (sign > 0)")
	}
}

func TestCAD_RecursiveLifting_CylindricalAncestry(t *testing.T) {
	env := NewEnv()
	circleNode, _ := Parse("x^2 + y^2 - 1")

	cells, err := CADDecomposeCells([]Node{circleNode}, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADDecomposeCells failed: %v", err)
	}

	for i, c := range cells {
		if c.Parent == nil {
			t.Fatalf("cell %d has nil Parent; cylindrical ancestry broken", i)
		}
		// Parent should be in R^1
		if len(c.Parent.SamplePoint) != 1 {
			t.Errorf("cell %d: expected parent sample in R^1, got %d coordinates", i, len(c.Parent.SamplePoint))
		}
		// First coordinate of child must match parent coordinate
		if !c.SamplePoint[0].Equal(c.Parent.SamplePoint[0]) {
			t.Errorf("cell %d: child x-coord %v does not match parent x-coord %v", i, c.SamplePoint[0], c.Parent.SamplePoint[0])
		}
		// Dimension relation: dim(Child) = dim(Parent) + (isSection ? 0 : 1)
		expectedDim := c.Parent.Dimension
		if !c.IsSection {
			expectedDim++
		}
		if c.Dimension != expectedDim {
			t.Errorf("cell %d: expected dim %d, got %d", i, expectedDim, c.Dimension)
		}
	}
}

func TestCAD_RecursiveLifting_2D_Intersection(t *testing.T) {
	env := NewEnv()
	// Line and parabola: y - x and y - x^2
	p1, _ := Parse("y - x")
	p2, _ := Parse("y - x^2")

	cells, err := CADDecomposeCells([]Node{p1, p2}, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADDecomposeCells failed: %v", err)
	}

	if len(cells) == 0 {
		t.Fatalf("expected non-empty decomposition for line and parabola")
	}

	// Verify that each cell has valid 2D coordinates and sign vectors for both polynomials
	for _, c := range cells {
		if len(c.SamplePoint) != 2 {
			t.Fatalf("expected 2D sample point, got %v", c.SamplePoint)
		}
		if _, ok := c.SignVector[p1.String()]; !ok {
			t.Fatalf("missing sign for %s", p1.String())
		}
		if _, ok := c.SignVector[p2.String()]; !ok {
			t.Fatalf("missing sign for %s", p2.String())
		}
	}
}

func TestCAD_SolutionReconstruction_2D_CircleInterior(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("x^2 + y^2 - 1 < 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	sol, err := CADSolveFormula(expr, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADSolveFormula failed: %v", err)
	}

	list, ok := sol.(*ListNode)
	if !ok || len(list.Elements) == 0 {
		t.Fatalf("expected non-empty ListNode for circle interior, got %v", sol)
	}

	// Must include origin [0, 0] as satisfying sample
	foundOrigin := false
	for _, elem := range list.Elements {
		pair, ok := elem.(*ListNode)
		if ok && len(pair.Elements) == 2 {
			xRat, xOk := pair.Elements[0].(*RationalNode)
			yRat, yOk := pair.Elements[1].(*RationalNode)
			if xOk && yOk && xRat.Val.Sign() == 0 && yRat.Val.Sign() == 0 {
				foundOrigin = true
				break
			}
		}
	}
	if !foundOrigin {
		t.Errorf("expected circle interior solution to contain origin [0, 0], got: %s", sol.String())
	}
}

func TestCAD_SolutionReconstruction_2D_CircleBoundary(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("x^2 + y^2 - 1 == 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	sol, cells, err := CADSolveFormulaCells(expr, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADSolveFormulaCells failed: %v", err)
	}

	if len(cells) == 0 {
		t.Fatalf("expected satisfying boundary cells, got empty")
	}

	// All satisfying cells must be sections (dim <= 1)
	for _, c := range cells {
		if !c.IsSection && c.Dimension == 2 {
			t.Errorf("boundary equation solution should not contain 2D sector cell: %v", c.SamplePoint)
		}
	}
	_ = sol
}

func TestCAD_SolutionReconstruction_BooleanFormula(t *testing.T) {
	env := NewEnv()
	// Circle interior AND right half-plane: x^2 + y^2 - 1 < 0 AND x > 0
	c1, _ := Parse("x^2 + y^2 - 1 < 0")
	c2, _ := Parse("x > 0")
	andNode := &FuncNode{Name: "and", Args: []Node{c1, c2}}

	sol, cells, err := CADSolveFormulaCells(andNode, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADSolveFormulaCells failed: %v", err)
	}

	if len(cells) == 0 {
		t.Fatalf("expected non-empty solution for and(circle, right half plane)")
	}

	// In all satisfying cells, x coordinate must be positive
	for _, c := range cells {
		xVal := c.SamplePoint[0]
		if rat, ok := xVal.(*RationalNode); ok {
			if rat.Val.Sign() <= 0 {
				t.Errorf("expected positive x coordinate in satisfying cell, got %v", rat.Val)
			}
		}
	}
	_ = sol
}

func TestCAD_SolutionReconstruction_UNSAT(t *testing.T) {
	env := NewEnv()
	// x^2 + y^2 + 1 <= 0 is completely unsatisfiable over R^2
	expr, _ := Parse("x^2 + y^2 + 1 <= 0")
	sol, err := CADSolveFormula(expr, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADSolveFormula failed: %v", err)
	}

	vNode, ok := sol.(*VarNode)
	if !ok || vNode.Name != "false" {
		t.Fatalf("expected false for UNSAT system, got %v", sol)
	}
}

func TestCAD_SolutionReconstruction_Tautology(t *testing.T) {
	env := NewEnv()
	// x^2 + y^2 + 1 > 0 is true everywhere on R^2
	expr, _ := Parse("x^2 + y^2 + 1 > 0")
	sol, err := CADSolveFormula(expr, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADSolveFormula failed: %v", err)
	}

	vNode, ok := sol.(*VarNode)
	if !ok || vNode.Name != "true" {
		t.Fatalf("expected true for TAUTOLOGY system, got %v", sol)
	}
}

func TestCAD_SolutionReconstruction_Disconnected(t *testing.T) {
	env := NewEnv()
	// (x^2 - 1) * (y^2 - 1) > 0 has 5 disconnected open components in R^2
	expr, _ := Parse("(x^2 - 1) * (y^2 - 1) > 0")
	sol, cells, err := CADSolveFormulaCells(expr, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADSolveFormulaCells failed: %v", err)
	}

	if len(cells) < 4 {
		t.Fatalf("expected at least 4 disconnected sector components, got %d cells", len(cells))
	}
	_ = sol
}


