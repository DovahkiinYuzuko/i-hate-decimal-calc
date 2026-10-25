package calc

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/parser"
)

func TestVM_BasicArithmetic(t *testing.T) {
	st := NewSymbolTable()
	compiler := NewCompiler(st)

	// Expr: (2 + 3) * 4 - 5
	// In AST:
	//   AddNode [ MulNode [ AddNode [ 2, 3 ], 4 ], -5 ]
	two := &RationalNode{Val: big.NewRat(2, 1)}
	three := &RationalNode{Val: big.NewRat(3, 1)}
	four := &RationalNode{Val: big.NewRat(4, 1)}
	five := &RationalNode{Val: big.NewRat(5, 1)}

	expr := &AddNode{
		Terms: []Node{
			&MulNode{
				Factors: []Node{
					&AddNode{Terms: []Node{two, three}},
					four,
				},
			},
			&UnaryOpNode{Op: "-", Expr: five},
		},
	}

	bc, err := compiler.Compile(expr)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	env := NewEnv()
	vm := NewVM(bc, env, DefaultVMBudget)
	res, err := vm.Run()
	if err != nil {
		t.Fatalf("VM Run error: %v", err)
	}

	rat, ok := res.(*RationalNode)
	if !ok {
		t.Fatalf("expected RationalNode, got %T (%v)", res, res)
	}
	// (2 + 3) * 4 - 5 = 20 - 5 = 15
	if rat.Val.Cmp(big.NewRat(15, 1)) != 0 {
		t.Errorf("expected 15, got %v", rat.Val)
	}
}

func TestVM_LocalSlots(t *testing.T) {
	st := NewSymbolTable()
	st.Define("x")
	st.Define("y")

	compiler := NewCompiler(st)

	// x^2 + 2*x*y + y^2
	// x is slot 0, y is slot 1
	node, err := parser.Parse("x^2 + 2*x*y + y^2")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	bc, err := compiler.Compile(node)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	env := NewEnv()
	vm := NewVM(bc, env, DefaultVMBudget)
	// Push x = 3, y = 4 as slot arguments
	vm.PushArgs([]Node{
		&RationalNode{Val: big.NewRat(3, 1)},
		&RationalNode{Val: big.NewRat(4, 1)},
	})

	res, err := vm.Run()
	if err != nil {
		t.Fatalf("VM Run error: %v", err)
	}

	rat, ok := res.(*RationalNode)
	if !ok {
		t.Fatalf("expected RationalNode, got %T", res)
	}
	// (3 + 4)^2 = 49
	if rat.Val.Cmp(big.NewRat(49, 1)) != 0 {
		t.Errorf("expected 49, got %v", rat.Val)
	}
}

func TestUserFunction_NamedAndRecursion(t *testing.T) {
	env := NewEnv()

	// f(x, y) = x^2 + y^2
	stmt, err := parser.ParseStatement("f(x, y) = x^2 + y^2")
	if err != nil {
		t.Fatalf("ParseStatement error: %v", err)
	}
	fnDef, ok := stmt.(*ast.UserFuncDefStmt)
	if !ok {
		t.Fatalf("expected *ast.UserFuncDefStmt, got %T", stmt)
	}

	st := NewSymbolTable()
	for _, p := range fnDef.Params {
		st.Define(p)
	}
	c := NewCompiler(st)
	bc, err := c.Compile(fnDef.Body)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	if err := env.SetUserFunc(fnDef.Name, fnDef.Params, fnDef.Body); err != nil {
		t.Fatalf("SetUserFunc error: %v", err)
	}
	if fn, ok := env.GetUserFunc("f"); ok {
		fn.Bytecode = bc
	}

	// Evaluate f(3, 4) -> 25
	callNode, err := parser.Parse("f(3, 4)")
	if err != nil {
		t.Fatalf("Parse call error: %v", err)
	}
	res, err := EvalWithEnv(callNode, env)
	if err != nil {
		t.Fatalf("Eval call error: %v", err)
	}
	rat, ok := res.(*RationalNode)
	if !ok || rat.Val.Cmp(big.NewRat(25, 1)) != 0 {
		t.Errorf("expected 25, got %v", res)
	}

	// Recursive function: myfib(n) = piecewise([[n, n <= 1]], myfib(n - 1) + myfib(n - 2))
	fibStmt, err := parser.ParseStatement("myfib(n) = piecewise([[n, n <= 1]], myfib(n - 1) + myfib(n - 2))")
	if err != nil {
		t.Fatalf("Parse fib error: %v", err)
	}
	fibDef := fibStmt.(*ast.UserFuncDefStmt)
	if err := env.SetUserFunc(fibDef.Name, fibDef.Params, fibDef.Body); err != nil {
		t.Fatalf("SetUserFunc fib error: %v", err)
	}

	// Evaluate myfib(10) -> 55
	fib10, err := parser.Parse("myfib(10)")
	if err != nil {
		t.Fatalf("Parse myfib(10) error: %v", err)
	}
	fibRes, err := EvalWithEnv(fib10, env)
	if err != nil {
		t.Fatalf("Eval myfib(10) error: %v", err)
	}
	fibRat, ok := fibRes.(*RationalNode)
	if !ok || fibRat.Val.Cmp(big.NewRat(55, 1)) != 0 {
		t.Errorf("expected myfib(10) == 55, got %v", fibRes)
	}

	// Verify Memoization: second call should be instant and identical
	memoRes, err := EvalWithEnv(fib10, env)
	if err != nil {
		t.Fatalf("Eval memo fib(10) error: %v", err)
	}
	if !memoRes.Equal(fibRes) {
		t.Errorf("memoized result mismatch: %v vs %v", memoRes, fibRes)
	}
}

func TestVM_BudgetExceeded(t *testing.T) {
	st := NewSymbolTable()
	compiler := NewCompiler(st)

	expr, err := parser.Parse("1 + 2 + 3 + 4 + 5")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	bc, err := compiler.Compile(expr)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	env := NewEnv()
	// Extremely small budget: 2 instructions
	vm := NewVM(bc, env, 2)
	_, err = vm.Run()
	if err == nil {
		t.Fatal("expected budget exceeded error, got nil")
	}
}
