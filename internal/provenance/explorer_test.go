package provenance

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func TestStepExplorer_Navigation(t *testing.T) {
	step1 := &DerivationNode{
		RuleID:   RuleAlgebraExpand,
		Relation: RelationEq,
		Before:   ast.NewVar("x"),
		After:    ast.NewVar("y"),
	}
	step2 := &DerivationNode{
		RuleID:   RuleAlgebraNormalize,
		Relation: RelationEq,
		Before:   ast.NewVar("y"),
		After:    ast.NewVar("z"),
	}
	root := &DerivationNode{
		RuleID:   "root",
		Relation: RelationEq,
		Children: []*DerivationNode{step1, step2},
	}

	// Simulated user input:
	// "" (Enter -> Next: step 2)
	// "b" (Back -> step 1)
	// "q" (Quit)
	input := strings.NewReader("\nb\nq\n")
	var output bytes.Buffer

	explorer := NewStepExplorer(root, input, &output)
	if len(explorer.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(explorer.Steps))
	}

	if err := explorer.Run(); err != nil {
		t.Fatalf("explorer.Run() returned error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "Derivation Step 1") && !strings.Contains(outStr, "導出ステップ 1") {
		t.Errorf("expected step 1 in output, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Derivation Step 2") && !strings.Contains(outStr, "導出ステップ 2") {
		t.Errorf("expected step 2 in output, got:\n%s", outStr)
	}
}
