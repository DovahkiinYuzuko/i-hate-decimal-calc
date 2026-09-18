package calc

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func TestArenaBasicAllocation(t *testing.T) {
	arena := NewNodeArena()
	defer func() {
		_ = arena.Release()
	}()

	// 1. Rational
	rat, err := arena.AllocRational(3, 4)
	if err != nil {
		t.Fatalf("AllocRational failed: %v", err)
	}
	if rat.String() != "3/4" {
		t.Errorf("expected 3/4, got %s", rat.String())
	}

	// 2. VarNode
	sym, err := arena.AllocVar("x")
	if err != nil {
		t.Fatalf("AllocVar failed: %v", err)
	}
	if sym.String() != "x" {
		t.Errorf("expected x, got %s", sym.String())
	}

	// 3. Add
	add, err := arena.AllocAdd([]ast.Node{sym, rat})
	if err != nil {
		t.Fatalf("AllocAdd failed: %v", err)
	}
	if len(add.Terms) != 2 {
		t.Errorf("expected 2 terms, got %d", len(add.Terms))
	}

	// 4. Pow
	two, _ := arena.AllocRational(2, 1)
	pow, err := arena.AllocPow(add, two)
	if err != nil {
		t.Fatalf("AllocPow failed: %v", err)
	}
	if pow.Base != add || pow.Exp != two {
		t.Errorf("pow structure mismatch")
	}

	// 5. Sqrt
	five, _ := arena.AllocRational(5, 1)
	sqrt, err := arena.AllocSqrt(five)
	if err != nil {
		t.Fatalf("AllocSqrt failed: %v", err)
	}
	if sqrt.Radicand != five {
		t.Errorf("sqrt structure mismatch")
	}

	// 6. Mul
	mul, err := arena.AllocMul([]ast.Node{pow, sqrt})
	if err != nil {
		t.Fatalf("AllocMul failed: %v", err)
	}
	if len(mul.Factors) != 2 {
		t.Errorf("expected 2 factors, got %d", len(mul.Factors))
	}

	// 7. UnaryOp (-)
	neg, err := arena.AllocUnary("-", mul)
	if err != nil {
		t.Fatalf("AllocUnary failed: %v", err)
	}
	if neg.Expr != mul {
		t.Errorf("neg structure mismatch")
	}
}

func TestArenaFSMTransitions(t *testing.T) {
	arena := NewNodeArena()

	if arena.State() != ArenaStateActive {
		t.Fatalf("expected ArenaStateActive, got %s", arena.State())
	}

	// Allocate node while active
	sym, err := arena.AllocVar("y")
	if err != nil {
		t.Fatalf("failed to allocate var: %v", err)
	}

	// Detach
	detached := arena.Detach(sym)
	if arena.State() != ArenaStateDetached {
		t.Errorf("expected ArenaStateDetached, got %s", arena.State())
	}
	if detached.String() != "y" {
		t.Errorf("detached node corrupted: %s", detached.String())
	}

	// Release
	if err := arena.Release(); err != nil {
		t.Fatalf("release failed: %v", err)
	}
	if arena.State() != ArenaStateReleased {
		t.Errorf("expected ArenaStateReleased, got %s", arena.State())
	}

	// Allocation after release must fail
	_, err = arena.AllocVar("z")
	if err == nil {
		t.Fatalf("expected error when allocating from released arena, got nil")
	}

	// Double release must fail
	if err := arena.Release(); err == nil {
		t.Fatalf("expected error on double release, got nil")
	}
}

func TestArenaDetachDeepClone(t *testing.T) {
	arena := NewNodeArena()

	// Build: (x + 1)^2
	x, _ := arena.AllocVar("x")
	one, _ := arena.AllocRational(1, 1)
	add, _ := arena.AllocAdd([]ast.Node{x, one})
	two, _ := arena.AllocRational(2, 1)
	pow, _ := arena.AllocPow(add, two)

	// Detach result tree
	result := arena.Detach(pow)

	// Immediately release arena (wiping out slabs)
	if err := arena.Release(); err != nil {
		t.Fatalf("release failed: %v", err)
	}

	// Result must remain valid and fully functional
	powResult, ok := result.(*ast.PowNode)
	if !ok {
		t.Fatalf("expected *ast.PowNode, got %T", result)
	}

	addResult, ok := powResult.Base.(*ast.AddNode)
	if !ok {
		t.Fatalf("expected *ast.AddNode, got %T", powResult.Base)
	}

	if len(addResult.Terms) != 2 {
		t.Fatalf("expected 2 terms, got %d", len(addResult.Terms))
	}
	if addResult.Terms[0].String() != "x" {
		t.Errorf("expected term 'x', got %s", addResult.Terms[0].String())
	}
	if addResult.Terms[1].String() != "1" {
		t.Errorf("expected term '1', got %s", addResult.Terms[1].String())
	}
}

func TestArenaRecyclePool(t *testing.T) {
	arena := GetNodeArena()
	if arena.State() != ArenaStateActive {
		t.Fatalf("expected ArenaStateActive from pool, got %s", arena.State())
	}

	for i := 0; i < 500; i++ {
		_, err := arena.AllocRational(int64(i), 1)
		if err != nil {
			t.Fatalf("allocation %d failed: %v", i, err)
		}
	}

	// Recycle to pool
	if err := RecycleArena(arena); err != nil {
		t.Fatalf("RecycleArena failed: %v", err)
	}

	// Get again
	arena2 := GetNodeArena()
	if arena2.State() != ArenaStateActive {
		t.Fatalf("expected recycled arena to be active, got %s", arena2.State())
	}

	// Slabs should have been reset to 0 length
	if len(arena2.rationalSlab) != 0 {
		t.Errorf("expected 0 rational slab length, got %d", len(arena2.rationalSlab))
	}
	_ = RecycleArena(arena2)
}

func BenchmarkStandardAlloc(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		// Build 100 expressions using standard new
		var root ast.Node
		for j := 0; j < 100; j++ {
			r := new(big.Rat).SetInt64(int64(j))
			num := &ast.RationalNode{Val: r}
			sym := &ast.VarNode{Name: "x"}
			root = &ast.AddNode{Terms: []ast.Node{num, sym}}
		}
		_ = root
	}
}

func BenchmarkArenaAlloc(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		arena := GetNodeArena()
		var root ast.Node
		for j := 0; j < 100; j++ {
			num, _ := arena.AllocRational(int64(j), 1)
			sym, _ := arena.AllocVar("x")
			root, _ = arena.AllocBinaryAdd(num, sym)
		}
		_ = arena.Detach(root)
		_ = RecycleArena(arena)
	}
}

func BenchmarkPureArenaLifecycle(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		arena := GetNodeArena()
		var root ast.Node
		for j := 0; j < 100; j++ {
			num, _ := arena.AllocRational(int64(j), 1)
			sym, _ := arena.AllocVar("x")
			root, _ = arena.AllocBinaryAdd(num, sym)
		}
		_ = root
		_ = RecycleArena(arena)
	}
}
