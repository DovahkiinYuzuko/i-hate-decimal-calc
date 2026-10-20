package provenance

import (
	"math/big"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func TestLeanCalcVisitor_ZeroSorryContract(t *testing.T) {
	visitor := NewLeanCalcVisitor()

	// 1. Unsupported rule must immediately return error, never emit 'sorry'
	unsupportedNode := &DerivationNode{
		ID:     1,
		RuleID: "heuristic.magic_trick",
		Before: &ast.VarNode{Name: "x"},
		After:  &ast.VarNode{Name: "y"},
	}

	_, err := visitor.RenderCalc(unsupportedNode, "test_theorem")
	if err == nil {
		t.Fatalf("expected error for unsupported rule in LeanCalcVisitor, got nil")
	}

	// 2. Nil root must return error
	_, err = visitor.RenderCalc(nil, "test_theorem")
	if err == nil {
		t.Fatalf("expected error for nil DerivationNode, got nil")
	}

	// 3. Node with no steps must return error
	emptyNode := &DerivationNode{ID: 2}
	_, err = visitor.RenderCalc(emptyNode, "test_theorem")
	if err == nil {
		t.Fatalf("expected error for empty DerivationNode, got nil")
	}
}

func TestLeanCalcVisitor_AlgebraicCalcBlock(t *testing.T) {
	visitor := NewLeanCalcVisitor()

	// Step 1: (x + 1)^2 -> (x + 1)*(x + 1) via algebra.expand
	step1 := &DerivationNode{
		ID:     1,
		RuleID: RuleAlgebraExpand,
		Before: &ast.PowNode{
			Base: &ast.AddNode{Terms: []ast.Node{&ast.VarNode{Name: "x"}, &ast.RationalNode{Val: big.NewRat(1, 1)}}},
			Exp:  &ast.RationalNode{Val: big.NewRat(2, 1)},
		},
		After: &ast.MulNode{
			Factors: []ast.Node{
				&ast.AddNode{Terms: []ast.Node{&ast.VarNode{Name: "x"}, &ast.RationalNode{Val: big.NewRat(1, 1)}}},
				&ast.AddNode{Terms: []ast.Node{&ast.VarNode{Name: "x"}, &ast.RationalNode{Val: big.NewRat(1, 1)}}},
			},
		},
	}

	// Step 2: (x + 1)*(x + 1) -> x^2 + 2*x + 1 via algebra.normalize
	step2 := &DerivationNode{
		ID:     2,
		RuleID: RuleAlgebraNormalize,
		Before: step1.After,
		After: &ast.AddNode{
			Terms: []ast.Node{
				&ast.PowNode{Base: &ast.VarNode{Name: "x"}, Exp: &ast.RationalNode{Val: big.NewRat(2, 1)}},
				&ast.MulNode{Factors: []ast.Node{&ast.RationalNode{Val: big.NewRat(2, 1)}, &ast.VarNode{Name: "x"}}},
				&ast.RationalNode{Val: big.NewRat(1, 1)},
			},
		},
	}

	root := &DerivationNode{
		ID:       0,
		Children: []*DerivationNode{step1, step2},
	}

	code, err := visitor.RenderCalc(root, "square_expansion")
	if err != nil {
		t.Fatalf("unexpected error rendering Lean calc: %v", err)
	}

	// Assertions
	if strings.Contains(code, "sorry") {
		t.Errorf("Lean code must NEVER contain 'sorry', got:\n%s", code)
	}
	if !strings.Contains(code, "theorem square_expansion") {
		t.Errorf("missing theorem declaration in code:\n%s", code)
	}
	if !strings.Contains(code, "calc") {
		t.Errorf("missing calc block in code:\n%s", code)
	}
	if !strings.Contains(code, "by ring") {
		t.Errorf("missing 'by ring' tactic in code:\n%s", code)
	}
	if !strings.Contains(code, "#print axioms square_expansion") {
		t.Errorf("missing '#print axioms' verification in code:\n%s", code)
	}
}

func TestLeanCalcVisitor_NormNumStep(t *testing.T) {
	visitor := NewLeanCalcVisitor()

	step := &DerivationNode{
		ID:     1,
		RuleID: "arith.eval",
		Before: &ast.AddNode{
			Terms: []ast.Node{
				&ast.RationalNode{Val: big.NewRat(1, 2)},
				&ast.RationalNode{Val: big.NewRat(1, 3)},
			},
		},
		After: &ast.RationalNode{Val: big.NewRat(5, 6)},
	}

	code, err := visitor.RenderCalc(step, "arith_proof")
	if err != nil {
		t.Fatalf("unexpected error rendering arithmetic Lean calc: %v", err)
	}

	if strings.Contains(code, "sorry") {
		t.Errorf("Lean code must NEVER contain 'sorry', got:\n%s", code)
	}
	if !strings.Contains(code, "import Mathlib.Tactic.NormNum") {
		t.Errorf("missing Mathlib.Tactic.NormNum import in code:\n%s", code)
	}
	if !strings.Contains(code, "by norm_num") {
		t.Errorf("missing 'by norm_num' tactic in code:\n%s", code)
	}
}
