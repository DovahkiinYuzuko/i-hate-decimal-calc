package ast

import (
	"fmt"
	"hash/maphash"
	"sort"
	"sync"
	"sync/atomic"
	"unique"

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

// CanonicalVar returns an interned canonical variable node using Go 1.23 unique.Make for symbol names.
func (p *InternPool) CanonicalVar(name string) *CanonicalNode {
	h := unique.Make(name)
	payloadHash := maphash.Comparable(p.seed, h)

	key := NodeKey{
		Kind:      NodeVar,
		Arity:     0,
		PayloadID: ExprID(payloadHash),
	}

	return p.GetOrCreate(key, func(id ExprID) Node {
		return &VarNode{Name: name}
	})
}

// CanonicalRational returns an interned rational number node.
func (p *InternPool) CanonicalRational(num, denom int64) (*CanonicalNode, error) {
	ratNode, err := NewRational(num, denom)
	if err != nil {
		return nil, err
	}

	h := unique.Make(ratNode.Val.String())
	payloadHash := maphash.Comparable(p.seed, h)

	key := NodeKey{
		Kind:      NodeRational,
		Arity:     0,
		PayloadID: ExprID(payloadHash),
	}

	return p.GetOrCreate(key, func(id ExprID) Node {
		return ratNode
	}), nil
}

// ToCanonical ensures that a node is wrapped as a CanonicalNode in this pool.
func (p *InternPool) ToCanonical(n Node) *CanonicalNode {
	if cn, ok := n.(*CanonicalNode); ok {
		return cn
	}
	switch v := n.(type) {
	case *VarNode:
		return p.CanonicalVar(v.Name)
	case *RationalNode:
		if v.Val.Denom().IsInt64() && v.Val.Num().IsInt64() {
			cn, err := p.CanonicalRational(v.Val.Num().Int64(), v.Val.Denom().Int64())
			if err == nil {
				return cn
			}
		}
		h := unique.Make(v.Val.String())
		payloadHash := maphash.Comparable(p.seed, h)
		key := NodeKey{
			Kind:      NodeRational,
			Arity:     0,
			PayloadID: ExprID(payloadHash),
		}
		return p.GetOrCreate(key, func(id ExprID) Node { return v })
	case *AddNode:
		return p.CanonicalAdd(v.Terms)
	case *MulNode:
		return p.CanonicalMul(v.Factors)
	default:
		// Generic fallback: intern using string representation hash
		h := unique.Make(n.String())
		payloadHash := maphash.Comparable(p.seed, h)
		key := NodeKey{
			Kind:      n.Type(),
			Arity:     0,
			PayloadID: ExprID(payloadHash),
		}
		return p.GetOrCreate(key, func(id ExprID) Node { return n })
	}
}

// CanonicalAdd returns a flattened, deterministically sorted, interned canonical addition node.
func (p *InternPool) CanonicalAdd(terms []Node) *CanonicalNode {
	// 1. Flattening (結合則の適用)
	var flattened []*CanonicalNode
	for _, t := range terms {
		// CanonicalNode を剥がして実体が AddNode か確認
		inner := t
		if cn, ok := t.(*CanonicalNode); ok && cn.Original != nil {
			inner = cn.Original
		}
		if add, ok := inner.(*AddNode); ok {
			for _, sub := range add.Terms {
				flattened = append(flattened, p.ToCanonical(sub))
			}
		} else {
			flattened = append(flattened, p.ToCanonical(t))
		}
	}

	if len(flattened) == 0 {
		zero, _ := p.CanonicalRational(0, 1)
		return zero
	}
	if len(flattened) == 1 {
		return flattened[0]
	}

	// 2. Deterministic Canonical Sorting (ExprID順)
	sort.Slice(flattened, func(i, j int) bool {
		return flattened[i].ID < flattened[j].ID
	})

	// 3. Construct Key
	var child0, child1 ExprID
	if len(flattened) >= 1 {
		child0 = flattened[0].ID
	}
	if len(flattened) >= 2 {
		child1 = flattened[1].ID
	}

	var extraHash uint64
	if len(flattened) > 2 {
		// N-ary (3子以上) は子ID列から合成ハッシュを算出
		var ids []ExprID
		for _, child := range flattened {
			ids = append(ids, child.ID)
		}
		for i, id := range ids {
			extraHash ^= uint64(id) * uint64(31+i*7)
		}
	}

	key := NodeKey{
		Kind:      NodeAdd,
		Arity:     uint8(len(flattened)),
		Child0:    child0,
		Child1:    child1,
		PayloadID: ExprID(extraHash),
	}

	return p.GetOrCreate(key, func(id ExprID) Node {
		var origTerms []Node
		for _, c := range flattened {
			if c.Original != nil {
				origTerms = append(origTerms, c.Original)
			} else {
				origTerms = append(origTerms, c)
			}
		}
		return &AddNode{Terms: origTerms}
	})
}

// CanonicalMul returns a flattened, deterministically sorted, interned canonical multiplication node.
func (p *InternPool) CanonicalMul(factors []Node) *CanonicalNode {
	// 1. Flattening (結合則の適用)
	var flattened []*CanonicalNode
	for _, f := range factors {
		inner := f
		if cn, ok := f.(*CanonicalNode); ok && cn.Original != nil {
			inner = cn.Original
		}
		if mul, ok := inner.(*MulNode); ok {
			for _, sub := range mul.Factors {
				flattened = append(flattened, p.ToCanonical(sub))
			}
		} else {
			flattened = append(flattened, p.ToCanonical(f))
		}
	}

	if len(flattened) == 0 {
		one, _ := p.CanonicalRational(1, 1)
		return one
	}
	if len(flattened) == 1 {
		return flattened[0]
	}

	// 2. Deterministic Canonical Sorting (ExprID順)
	sort.Slice(flattened, func(i, j int) bool {
		return flattened[i].ID < flattened[j].ID
	})

	// 3. Construct Key
	var child0, child1 ExprID
	if len(flattened) >= 1 {
		child0 = flattened[0].ID
	}
	if len(flattened) >= 2 {
		child1 = flattened[1].ID
	}

	var extraHash uint64
	if len(flattened) > 2 {
		for i, child := range flattened {
			extraHash ^= uint64(child.ID) * uint64(37+i*11)
		}
	}

	key := NodeKey{
		Kind:      NodeMul,
		Arity:     uint8(len(flattened)),
		Child0:    child0,
		Child1:    child1,
		PayloadID: ExprID(extraHash),
	}

	return p.GetOrCreate(key, func(id ExprID) Node {
		var origFactors []Node
		for _, c := range flattened {
			if c.Original != nil {
				origFactors = append(origFactors, c.Original)
			} else {
				origFactors = append(origFactors, c)
			}
		}
		return &MulNode{Factors: origFactors}
	})
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
