package calc

import (
	"errors"
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/parser"
)

// TestVM_Invariants_Budget tests deterministic termination upon budget exhaustion.
func TestVM_Invariants_Budget(t *testing.T) {
	stmt, err := parser.Parse("x^2 + 2*x + 1")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	st := NewSymbolTable()
	st.Define("x")
	compiler := NewCompiler(st)
	bc, err := compiler.Compile(stmt)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	env := NewEnv()
	// Deliberately starved budget
	vm := NewVM(bc, env, 2)
	vm.PushArgs([]Node{&RationalNode{Val: big.NewRat(5, 1)}})

	_, err = vm.Run()
	if err == nil {
		t.Fatal("expected error on budget exhaustion, got nil")
	}
	if !errors.Is(err, ErrVMBudgetExceeded) {
		t.Fatalf("expected ErrVMBudgetExceeded, got: %v", err)
	}
}

// TestVM_Invariants_Allocs verifies that the VM loop maintains low/bounded allocations.
func TestVM_Invariants_Allocs(t *testing.T) {
	stmt, err := parser.Parse("x + y")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	st := NewSymbolTable()
	st.Define("x")
	st.Define("y")
	compiler := NewCompiler(st)
	bc, err := compiler.Compile(stmt)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	env := NewEnv()
	vm := NewVM(bc, env, DefaultVMBudget)
	arg1 := &RationalNode{Val: big.NewRat(10, 1)}
	arg2 := &RationalNode{Val: big.NewRat(20, 1)}

	allocs := testing.AllocsPerRun(100, func() {
		vm.Reset()
		vm.Push(arg1)
		vm.Push(arg2)
		res, runErr := vm.Run()
		if runErr != nil || res == nil {
			t.Fatalf("VM run failed: %v", runErr)
		}
	})

	// Exact rational arithmetic evaluates AddNode in evalWithEnv which allocates big.Rat terms,
	// but dispatch itself should have bounded allocations (<= 25 allocs per operation).
	if allocs > 25 {
		t.Errorf("expected bounded allocs <= 25, got %v", allocs)
	}
}

// TestVM_Differential_Metamorphic verifies that AST evaluation and VM evaluation produce identical results.
func TestVM_Differential_Metamorphic(t *testing.T) {
	testCases := []struct {
		expr string
		vars map[string]int64
	}{
		{"x^2 + 2*x + 1", map[string]int64{"x": 3}},
		{"x^3 - 3*x^2 + 3*x - 1", map[string]int64{"x": 4}},
		{"(x + y) * (x - y)", map[string]int64{"x": 7, "y": 2}},
		{"x^2 + y^2 + 2*x*y", map[string]int64{"x": 5, "y": 6}},
		{"-x + 10", map[string]int64{"x": 12}},
		{"x * 0 + y * 1", map[string]int64{"x": 99, "y": 42}},
	}

	for _, tc := range testCases {
		t.Run(tc.expr, func(t *testing.T) {
			parsed, err := parser.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse error on %s: %v", tc.expr, err)
			}

			// AST Evaluation
			astEnv := NewEnv()
			st := NewSymbolTable()
			var paramNames []string
			var paramVals []Node
			for k, v := range tc.vars {
				ratVal := &RationalNode{Val: big.NewRat(v, 1)}
				astEnv.Set(k, ratVal)
				st.Define(k)
				paramNames = append(paramNames, k)
				paramVals = append(paramVals, ratVal)
			}

			astRes, err := EvalWithEnv(parsed, astEnv)
			if err != nil {
				t.Fatalf("AST Eval error on %s: %v", tc.expr, err)
			}

			// VM Evaluation
			compiler := NewCompiler(st)
			bc, err := compiler.Compile(parsed)
			if err != nil {
				t.Fatalf("VM Compile error on %s: %v", tc.expr, err)
			}

			vmEnv := NewEnv()
			vm := NewVM(bc, vmEnv, DefaultVMBudget)
			vm.PushArgs(paramVals)
			vmRes, err := vm.Run()
			if err != nil {
				t.Fatalf("VM Run error on %s: %v", tc.expr, err)
			}

			if !astRes.Equal(vmRes) {
				t.Errorf("Differential mismatch on %s: AST=%s, VM=%s", tc.expr, astRes, vmRes)
			}
		})
	}
}

// BenchmarkVM_RelativeSpeedup compares AST tree-walk evaluation against VM bytecode execution.
func BenchmarkVM_RelativeSpeedup(b *testing.B) {
	exprStr := "x^2 + 2*x + 1"
	parsed, err := parser.Parse(exprStr)
	if err != nil {
		b.Fatalf("Parse error: %v", err)
	}

	st := NewSymbolTable()
	st.Define("x")
	compiler := NewCompiler(st)
	bc, err := compiler.Compile(parsed)
	if err != nil {
		b.Fatalf("Compile error: %v", err)
	}

	arg := &RationalNode{Val: big.NewRat(5, 1)}

	b.Run("AST", func(b *testing.B) {
		env := NewEnv()
		env.Set("x", arg)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res, err := EvalWithEnv(parsed, env)
			if err != nil || res == nil {
				b.Fatalf("AST eval failed: %v", err)
			}
		}
	})

	b.Run("VM", func(b *testing.B) {
		env := NewEnv()
		vm := NewVM(bc, env, DefaultVMBudget)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vm.Reset()
			vm.Push(arg)
			res, err := vm.Run()
			if err != nil || res == nil {
				b.Fatalf("VM eval failed: %v", err)
			}
		}
	})
}
