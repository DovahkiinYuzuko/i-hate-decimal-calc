package calc

import (
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/parser"
)

func TestHandlers_Clifford(t *testing.T) {
	env := NewEnv()

	evalExpr := func(input string) (string, error) {
		parsed, err := parser.Parse(input)
		if err != nil {
			return "", err
		}
		res, err := EvalWithEnv(parsed, env)
		if err != nil {
			return "", err
		}
		return res.String(), nil
	}

	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "Cl(3,0) anticommutation e0*e1 + e1*e0",
			expr:     "clifford(e0 * e1 + e1 * e0, 3, 0)",
			expected: "0",
		},
		{
			name:     "Cl(3,0) positive square e0*e0",
			expr:     "clifford(e0 * e0, 3, 0)",
			expected: "1",
		},
		{
			name:     "Cl(1,3) spacetime negative square e1*e1",
			expr:     "clifford(e1 * e1, 1, 3)",
			expected: "-1",
		},
		{
			name:     "Cl(3,0,1) PGA null square e3*e3",
			expr:     "clifford(e3 * e3, 3, 0, 1)",
			expected: "0",
		},
		{
			name:     "Clifford wedge product",
			expr:     "clifford_wedge(e0, e1, 3, 0)",
			expected: "e0^e1",
		},
		{
			name:     "Clifford left contraction",
			expr:     "clifford_contract(e0, e0, \"left\", 3, 0)",
			expected: "1",
		},
		{
			name:     "Cl(3,0,1) PGA dual of e3",
			expr:     "clifford_dual(e3, 3, 0, 1)",
			expected: "-e0^e1^e2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := evalExpr(tc.expr)
			if err != nil {
				t.Fatalf("unexpected error evaluating %s: %v", tc.expr, err)
			}
			if got != tc.expected {
				t.Errorf("eval(%s) = %q, expected %q", tc.expr, got, tc.expected)
			}
		})
	}
}

func TestHandlers_Quantum(t *testing.T) {
	env := NewEnv()

	evalExpr := func(input string) (ast.Node, error) {
		parsed, err := parser.Parse(input)
		if err != nil {
			return nil, err
		}
		return EvalWithEnv(parsed, env)
	}

	t.Run("Quantum H H identity", func(t *testing.T) {
		// H followed by H is equivalent to identity (empty circuit [])
		res, err := evalExpr("quantum_equiv([ [\"H\", 0], [\"H\", 0] ], [])")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		list, ok := res.(*ast.ListNode)
		if !ok || len(list.Elements) != 3 {
			t.Fatalf("expected 3-element list, got %v", res)
		}
		if list.Elements[0].String() != "1" {
			t.Errorf("expected is_equiv = 1, got %s", list.Elements[0])
		}
		if list.Elements[1].String() != "0" {
			t.Errorf("expected phase = 0, got %s", list.Elements[1])
		}
	})

	t.Run("Quantum global phase difference", func(t *testing.T) {
		// X then Z is -i Y = omega^6 Y
		res, err := evalExpr("quantum_equiv([ [\"X\", 0], [\"Z\", 0] ], [ [\"Y\", 0] ], false)")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		list, ok := res.(*ast.ListNode)
		if !ok || len(list.Elements) != 3 {
			t.Fatalf("expected 3-element list, got %v", res)
		}
		if list.Elements[0].String() != "1" {
			t.Errorf("expected is_equiv = 1 up to global phase, got %s", list.Elements[0])
		}
		if list.Elements[1].String() != "2" {
			t.Errorf("expected phase = 2 (omega^2 = i), got %s", list.Elements[1])
		}

		// Strict mode must fail
		resStrict, err := evalExpr("quantum_equiv([ [\"X\", 0], [\"Z\", 0] ], [ [\"Y\", 0] ], true)")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		listStrict := resStrict.(*ast.ListNode)
		if listStrict.Elements[0].String() != "0" {
			t.Errorf("expected strict mismatch = 0, got %s", listStrict.Elements[0])
		}
	})

	t.Run("Quantum eval matrix", func(t *testing.T) {
		res, err := evalExpr("quantum_eval([ [\"X\", 0] ])")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mat, ok := res.(*ast.MatrixNode)
		if !ok {
			t.Fatalf("expected MatrixNode, got %T", res)
		}
		if mat.Rows != 2 || mat.Cols != 2 {
			t.Fatalf("expected 2x2 matrix, got %dx%d", mat.Rows, mat.Cols)
		}
		// Pauli X: [[0, 1], [1, 0]]
		if mat.Data[0][0].String() != "0" || mat.Data[0][1].String() != "1" ||
			mat.Data[1][0].String() != "1" || mat.Data[1][1].String() != "0" {
			t.Errorf("unexpected Pauli X matrix:\n%s", mat.String())
		}
	})
}
