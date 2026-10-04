package calc

import (
	"testing"
)

func TestPiecewiseNode_Basics(t *testing.T) {
	input := "piecewise([[-x, x < 0], [x, x >= 0]])"
	node, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", input, err)
	}
	got, err := Eval(node)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", input, err)
	}
	formatted := Format(got)
	expected := "piecewise([[-x, x < 0], [x, x >= 0]])"
	if formatted != expected {
		t.Errorf("Format(Eval(%q)) = %q, want %q", input, formatted, expected)
	}
}

func TestPiecewise_Normalize(t *testing.T) {
	// Case with false condition pruned: 0 == 1
	input := "piecewise([[x, 0 == 1], [2*x, x > 0]])"
	node, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", input, err)
	}
	got, err := Eval(node)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", input, err)
	}
	formatted := Format(got)
	expected := "piecewise([[2*x, x > 0]])"
	if formatted != expected {
		t.Errorf("Format(Eval(%q)) = %q, want %q", input, formatted, expected)
	}

	// Case with true condition collapsing: 1 == 1
	inputTrue := "piecewise([[x, 1 == 1], [2*x, x > 0]])"
	nodeTrue, err := Parse(inputTrue)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputTrue, err)
	}
	gotTrue, err := Eval(nodeTrue)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputTrue, err)
	}
	formattedTrue := Format(gotTrue)
	if formattedTrue != "x" {
		t.Errorf("Format(Eval(%q)) = %q, want %q", inputTrue, formattedTrue, "x")
	}

	// All identical branches collapse when fallback covers domain
	inputIdentical := "piecewise([[x, x < 0], [x, x >= 0]], x)"
	nodeId, err := Parse(inputIdentical)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputIdentical, err)
	}
	gotId, err := Eval(nodeId)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputIdentical, err)
	}
	formattedId := Format(gotId)
	if formattedId != "x" {
		t.Errorf("Format(Eval(%q)) = %q, want %q", inputIdentical, formattedId, "x")
	}
}

func TestPiecewise_AlgebraFold(t *testing.T) {
	// Unary negation folding
	inputNeg := "-piecewise([[-x, x < 0], [x, x >= 0]])"
	nodeNeg, err := Parse(inputNeg)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputNeg, err)
	}
	gotNeg, err := Eval(nodeNeg)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputNeg, err)
	}
	formattedNeg := Format(gotNeg)
	expectedNeg := "piecewise([[x, x < 0], [-x, x >= 0]])"
	if formattedNeg != expectedNeg {
		t.Errorf("Format(Eval(%q)) = %q, want %q", inputNeg, formattedNeg, expectedNeg)
	}

	// Scalar multiplication folding
	inputMul := "2 * piecewise([[-x, x < 0], [x, x >= 0]])"
	nodeMul, err := Parse(inputMul)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputMul, err)
	}
	gotMul, err := Eval(nodeMul)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputMul, err)
	}
	formattedMul := Format(gotMul)
	expectedMul := "piecewise([[-2*x, x < 0], [2*x, x >= 0]])"
	if formattedMul != expectedMul {
		t.Errorf("Format(Eval(%q)) = %q, want %q", inputMul, formattedMul, expectedMul)
	}
}

func TestPiecewise_Calculus(t *testing.T) {
	// Differentiation
	inputDiff := "diff(piecewise([[-x^2, x < 0], [x^2, x >= 0]]), x)"
	nodeDiff, err := Parse(inputDiff)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputDiff, err)
	}
	gotDiff, err := Eval(nodeDiff)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputDiff, err)
	}
	formattedDiff := Format(gotDiff)
	expectedDiff := "piecewise([[-2*x, x < 0], [2*x, x >= 0]])"
	if formattedDiff != expectedDiff {
		t.Errorf("Format(Eval(%q)) = %q, want %q", inputDiff, formattedDiff, expectedDiff)
	}

	// Indefinite Integration
	inputInt := "integrate(piecewise([[-x, x < 0], [x, x >= 0]]), x)"
	nodeInt, err := Parse(inputInt)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputInt, err)
	}
	gotInt, err := Eval(nodeInt)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputInt, err)
	}
	formattedInt := Format(gotInt)
	if formattedInt == "" {
		t.Errorf("integrate returned empty format")
	}

	// Definite Integration with partition: [-1, 2] split at 0 -> 1/2 + 2 = 5/2
	inputDefInt := "integrate(piecewise([[-x, x < 0], [x, x >= 0]]), x, -1, 2)"
	nodeDefInt, err := Parse(inputDefInt)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputDefInt, err)
	}
	gotDefInt, err := Eval(nodeDefInt)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputDefInt, err)
	}
	formattedDefInt := Format(gotDefInt)
	expectedDefInt := "5/2"
	if formattedDefInt != expectedDefInt {
		t.Errorf("Format(Eval(%q)) = %q, want %q", inputDefInt, formattedDefInt, expectedDefInt)
	}

	// Definite Integration within single partition: [1, 3] -> (3^2/2 - 1^2/2) = 4
	inputDefInt2 := "integrate(piecewise([[-x, x < 0], [x, x >= 0]]), x, 1, 3)"
	nodeDefInt2, err := Parse(inputDefInt2)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputDefInt2, err)
	}
	gotDefInt2, err := Eval(nodeDefInt2)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputDefInt2, err)
	}
	formattedDefInt2 := Format(gotDefInt2)
	expectedDefInt2 := "4"
	if formattedDefInt2 != expectedDefInt2 {
		t.Errorf("Format(Eval(%q)) = %q, want %q", inputDefInt2, formattedDefInt2, expectedDefInt2)
	}
}

func TestPiecewise_Solve(t *testing.T) {
	// solve(piecewise([[-x - 2, x < 0], [x - 2, x >= 0]]) == 0, x) -> [-2, 2]
	inputSolve := "solve(piecewise([[-x - 2, x < 0], [x - 2, x >= 0]]) == 0, x)"
	nodeSolve, err := Parse(inputSolve)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputSolve, err)
	}
	gotSolve, err := Eval(nodeSolve)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputSolve, err)
	}
	formattedSolve := Format(gotSolve)
	expectedSolve := "[-2, 2]"
	if formattedSolve != expectedSolve {
		t.Errorf("Format(Eval(%q)) = %q, want %q", inputSolve, formattedSolve, expectedSolve)
	}

	// Also without == 0: solve(piecewise([[-x - 2, x < 0], [x - 2, x >= 0]]), x) -> [-2, 2]
	inputSolve2 := "solve(piecewise([[-x - 2, x < 0], [x - 2, x >= 0]]), x)"
	nodeSolve2, err := Parse(inputSolve2)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputSolve2, err)
	}
	gotSolve2, err := Eval(nodeSolve2)
	if err != nil {
		t.Fatalf("Eval(%q) failed: %v", inputSolve2, err)
	}
	formattedSolve2 := Format(gotSolve2)
	if formattedSolve2 != expectedSolve {
		t.Errorf("Format(Eval(%q)) = %q, want %q", inputSolve2, formattedSolve2, expectedSolve)
	}

	// solve with extraneous roots eliminated:
	// - x - 2 = 0 -> x = 2 (rejected since not x < 0)
	// - -x - 2 = 0 -> x = -2 (rejected since not x >= 0)
	inputSolveNone := "solve(piecewise([[x - 2, x < 0], [-x - 2, x >= 0]]) == 0, x)"
	nodeSolveNone, err := Parse(inputSolveNone)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", inputSolveNone, err)
	}
	// Expecting error because equation has no solution
	_, err = Eval(nodeSolveNone)
	if err == nil {
		t.Errorf("expected no solution error for %q, but got nil", inputSolveNone)
	}
}
