package ast

import (
	"sync"
	"testing"
)

func TestInternPool_BasicDeduplication(t *testing.T) {
	pool := NewInternPool()

	key1 := NodeKey{
		Kind:      NodeAdd,
		Arity:     2,
		Child0:    10,
		Child1:    20,
		PayloadID: 0,
	}

	createCalls := 0
	createFn := func(id ExprID) Node {
		createCalls++
		return nil
	}

	node1 := pool.GetOrCreate(key1, createFn)
	if node1 == nil {
		t.Fatal("expected non-nil node1")
	}
	if node1.ID == 0 {
		t.Fatal("expected non-zero ExprID")
	}
	if createCalls != 1 {
		t.Fatalf("expected createCalls=1, got %d", createCalls)
	}

	// 2回目は同一のキーなので、createFn は呼ばれず同じポインタが返るべき
	node2 := pool.GetOrCreate(key1, createFn)
	if node2 != node1 {
		t.Fatalf("expected pointer equality node1 (%p) == node2 (%p)", node1, node2)
	}
	if createCalls != 1 {
		t.Fatalf("expected createCalls to remain 1, got %d", createCalls)
	}
}

func TestInternPool_DifferentKeys(t *testing.T) {
	pool := NewInternPool()

	key1 := NodeKey{Kind: NodeAdd, Arity: 2, Child0: 1, Child1: 2}
	key2 := NodeKey{Kind: NodeMul, Arity: 2, Child0: 1, Child1: 2}

	node1 := pool.GetOrCreate(key1, func(id ExprID) Node { return nil })
	node2 := pool.GetOrCreate(key2, func(id ExprID) Node { return nil })

	if node1 == node2 {
		t.Fatal("expected different nodes for different keys")
	}
	if node1.ID == node2.ID {
		t.Fatalf("expected different ExprIDs, got both %d", node1.ID)
	}
}

func TestInternPool_Concurrent(t *testing.T) {
	pool := NewInternPool()
	const numGoroutines = 20
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				// 各ループでいくつかの固定キーと可変キーを生成
				keyFixed := NodeKey{Kind: NodeAdd, Arity: 2, Child0: 100, Child1: 200}
				nFixed := pool.GetOrCreate(keyFixed, func(id ExprID) Node { return nil })
				if nFixed == nil {
					t.Errorf("got nil for fixed key")
				}

				keyVar := NodeKey{Kind: NodeMul, Arity: 2, Child0: ExprID(i), Child1: ExprID(gid)}
				nVar := pool.GetOrCreate(keyVar, func(id ExprID) Node { return nil })
				if nVar == nil {
					t.Errorf("got nil for var key")
				}
			}
		}(g)
	}

	wg.Wait()

	// 全 Goroutine で同一の fixed キーが同じポインタに集約されているか確認
	keyFixed := NodeKey{Kind: NodeAdd, Arity: 2, Child0: 100, Child1: 200}
	finalNode := pool.GetOrCreate(keyFixed, func(id ExprID) Node { return nil })
	if finalNode == nil {
		t.Fatal("expected non-nil finalNode")
	}
}

func TestIdentical(t *testing.T) {
	pool := NewInternPool()

	key := NodeKey{Kind: NodeRational, Arity: 0, PayloadID: 42}
	node1 := pool.GetOrCreate(key, func(id ExprID) Node { return nil })
	node2 := pool.GetOrCreate(key, func(id ExprID) Node { return nil })

	if !Identical(node1, node2) {
		t.Fatal("expected Identical to return true for same canonical node pointers")
	}

	keyDifferent := NodeKey{Kind: NodeRational, Arity: 0, PayloadID: 43}
	node3 := pool.GetOrCreate(keyDifferent, func(id ExprID) Node { return nil })
	if Identical(node1, node3) {
		t.Fatal("expected Identical to return false for different nodes")
	}
}
