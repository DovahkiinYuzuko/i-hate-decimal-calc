package ast

import (
	"testing"
)

// BenchmarkInternPool_Hit measures the retrieval speed and zero-allocation
// property when requesting already interned nodes.
func BenchmarkInternPool_Hit(b *testing.B) {
	pool := NewInternPool()
	key := NodeKey{
		Kind:      NodeAdd,
		Arity:     2,
		Child0:    100,
		Child1:    200,
		PayloadID: 0,
	}

	// Pre-populate
	_ = pool.GetOrCreate(key, func(id ExprID) Node {
		return &CanonicalNode{ID: id, Key: key}
	})

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		node := pool.GetOrCreate(key, nil)
		if node == nil {
			b.Fatal("unexpected nil node")
		}
	}
}

// BenchmarkInternPool_Parallel measures the scalability of the 64-shard
// cache-line padded intern pool under high concurrent contention.
func BenchmarkInternPool_Parallel(b *testing.B) {
	pool := NewInternPool()

	// Pre-populate multiple keys
	const numKeys = 256
	keys := make([]NodeKey, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = NodeKey{
			Kind:      NodeAdd,
			Arity:     2,
			Child0:    ExprID(i),
			Child1:    ExprID(i + 1),
			PayloadID: 0,
		}
		k := keys[i]
		_ = pool.GetOrCreate(k, func(id ExprID) Node {
			return &CanonicalNode{ID: id, Key: k}
		})
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			k := keys[idx&(numKeys-1)]
			node := pool.GetOrCreate(k, nil)
			if node == nil {
				b.Fatal("unexpected nil node")
			}
			idx++
		}
	})
}

// BenchmarkCanonicalVar measures interning of variable leaf nodes.
func BenchmarkCanonicalVar(b *testing.B) {
	pool := NewInternPool()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		node := pool.CanonicalVar("x")
		if node == nil {
			b.Fatal("unexpected nil node")
		}
	}
}

// BenchmarkCanonicalRational measures interning of rational leaf nodes.
func BenchmarkCanonicalRational(b *testing.B) {
	pool := NewInternPool()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		node, err := pool.CanonicalRational(355, 113)
		if err != nil || node == nil {
			b.Fatal("unexpected nil node or error")
		}
	}
}

// BenchmarkIdentical_vs_DeepEqual compares the performance of O(1) Identical
// against recursive Equal across a multi-level expression tree.
func BenchmarkIdentical_vs_DeepEqual(b *testing.B) {
	pool := NewInternPool()

	// Build a deep composite node: (((x + y) * (x + y)) + ((x + y) * (x + y)))
	vx := pool.CanonicalVar("x")
	vy := pool.CanonicalVar("y")
	add1 := pool.CanonicalAdd([]Node{vx, vy})
	mul1 := pool.CanonicalMul([]Node{add1, add1})
	root1 := pool.CanonicalAdd([]Node{mul1, mul1})

	// Recreate or reference the identical expression
	add2 := pool.CanonicalAdd([]Node{vx, vy})
	mul2 := pool.CanonicalMul([]Node{add2, add2})
	root2 := pool.CanonicalAdd([]Node{mul2, mul2})

	b.Run("Identical_O1", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !Identical(root1, root2) {
				b.Fatal("expected Identical to be true")
			}
		}
	})

	b.Run("Recursive_Equal", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !root1.Equal(root2) {
				b.Fatal("expected Equal to be true")
			}
		}
	})
}
