package ast

import (
	"fmt"
	"hash/maphash"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

// ExprID represents a unique, deterministic, monotonically increasing identifier for canonical AST nodes.
type ExprID uint64

// NodeKey is a flat, strictly comparable descriptor that encapsulates the structural identity of a node.
type NodeKey struct {
	Kind      NodeType
	Arity     uint8
	Flags     uint16
	Child0    ExprID
	Child1    ExprID
	PayloadID ExprID
	Hash      uint64
}

// CanonicalNode represents an immutable interned expression node in the canonical DAG.
type CanonicalNode struct {
	ID       ExprID
	Hash     uint64
	Key      NodeKey
	Original Node
}

// Type returns the NodeType of the canonical node.
func (c *CanonicalNode) Type() NodeType {
	return c.Key.Kind
}

// String returns a string representation of the canonical node.
func (c *CanonicalNode) String() string {
	if c.Original != nil {
		return c.Original.String()
	}
	return fmt.Sprintf("canonical#%d", c.ID)
}

// Equal evaluates equality against another Node. For two canonical nodes, it checks pointer identity.
func (c *CanonicalNode) Equal(other Node) bool {
	if other == nil {
		return false
	}
	if cn, ok := other.(*CanonicalNode); ok {
		return c == cn
	}
	if c.Original != nil {
		return c.Original.Equal(other)
	}
	return false
}

// shard holds a mutex-protected bucket of canonical nodes padded to prevent false sharing.
type shard struct {
	_  cpu.CacheLinePad
	mu sync.Mutex
	m  map[NodeKey]*CanonicalNode
}

// InternPool manages canonical DAG interning with 64 padded shards.
type InternPool struct {
	shards [64]shard
	idGen  atomic.Uint64
	seed   maphash.Seed
}

// NewInternPool initializes and returns a new concurrent InternPool.
func NewInternPool() *InternPool {
	p := &InternPool{
		seed: maphash.MakeSeed(),
	}
	for i := range p.shards {
		p.shards[i].m = make(map[NodeKey]*CanonicalNode, 64)
	}
	return p
}

// GetOrCreate returns an existing canonical node matching key, or creates and stores a new one.
func (p *InternPool) GetOrCreate(key NodeKey, createFn func(id ExprID) Node) *CanonicalNode {
	// If hash is not precomputed, compute it deterministically using maphash.Comparable
	if key.Hash == 0 {
		key.Hash = maphash.Comparable(p.seed, key)
	}

	shardIdx := key.Hash & 0x3F // 64 shards (0..63)
	s := &p.shards[shardIdx]

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.m[key]; ok {
		return existing
	}

	newID := ExprID(p.idGen.Add(1))
	var orig Node
	if createFn != nil {
		orig = createFn(newID)
	}

	node := &CanonicalNode{
		ID:       newID,
		Hash:     key.Hash,
		Key:      key,
		Original: orig,
	}

	s.m[key] = node
	return node
}

// Reset clears all interned entries in the pool for bulk session reclamation.
func (p *InternPool) Reset() {
	for i := range p.shards {
		s := &p.shards[i]
		s.mu.Lock()
		s.m = make(map[NodeKey]*CanonicalNode, 64)
		s.mu.Unlock()
	}
	p.idGen.Store(0)
}

// Identical evaluates whether two nodes point to the exact same canonical node (O(1)),
// or falls back to standard structural equality for non-canonical or cross-pool nodes.
func Identical(n1, n2 Node) bool {
	if n1 == nil && n2 == nil {
		return true
	}
	if n1 == nil || n2 == nil {
		return false
	}
	// Fast path: pointer equality
	if n1 == n2 {
		return true
	}

	// Canonical node fast path
	c1, ok1 := n1.(*CanonicalNode)
	c2, ok2 := n2.(*CanonicalNode)
	if ok1 && ok2 {
		return c1 == c2
	}

	// Unwrap CanonicalNode if only one is canonical
	if ok1 && c1.Original != nil {
		n1 = c1.Original
	}
	if ok2 && c2.Original != nil {
		n2 = c2.Original
	}

	return n1.Equal(n2)
}
