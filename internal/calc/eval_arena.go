package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"math/big"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

const defaultSlabChunkSize = 256

// NodeArena is a bump/slab allocator for AST nodes that enforces contiguous
// cache locality, zero-GC execution in hot loops, and strict lifecycle safety via FSM.
type NodeArena struct {
	fsm *ArenaLifecycleFSM

	// Typed slab slices
	rationalSlab   []ast.RationalNode
	ratValSlab     []big.Rat
	addSlab        []ast.AddNode
	mulSlab        []ast.MulNode
	powSlab        []ast.PowNode
	sqrtSlab       []ast.SqrtNode
	unarySlab      []ast.UnaryOpNode
	varSlab        []ast.VarNode
	binaryPairSlab [][2]ast.Node

	// Overflow slices if allocations exceed initial chunk size
	extraRationals []*ast.RationalNode
	extraAdds      []*ast.AddNode
	extraMuls      []*ast.MulNode
	extraPows      []*ast.PowNode
	extraSqrts     []*ast.SqrtNode
	extraUnaries   []*ast.UnaryOpNode
	extraVars      []*ast.VarNode
}

// Global pool of reusable NodeArenas to minimize OS heap allocations.
var nodeArenaPool = sync.Pool{
	New: func() any {
		return &NodeArena{
			fsm:            NewArenaLifecycleFSM(),
			rationalSlab:   make([]ast.RationalNode, 0, defaultSlabChunkSize),
			ratValSlab:     make([]big.Rat, 0, defaultSlabChunkSize),
			addSlab:        make([]ast.AddNode, 0, defaultSlabChunkSize),
			mulSlab:        make([]ast.MulNode, 0, defaultSlabChunkSize),
			powSlab:        make([]ast.PowNode, 0, defaultSlabChunkSize),
			sqrtSlab:       make([]ast.SqrtNode, 0, defaultSlabChunkSize),
			unarySlab:      make([]ast.UnaryOpNode, 0, defaultSlabChunkSize),
			varSlab:        make([]ast.VarNode, 0, defaultSlabChunkSize),
			binaryPairSlab: make([][2]ast.Node, 0, defaultSlabChunkSize),
		}
	},
}

// GetNodeArena retrieves an active NodeArena from the pool.
func GetNodeArena() *NodeArena {
	arena := nodeArenaPool.Get().(*NodeArena)
	_ = arena.fsm.Activate()
	return arena
}

// NewNodeArena allocates a standalone active NodeArena.
func NewNodeArena() *NodeArena {
	return &NodeArena{
		fsm:            NewArenaLifecycleFSM(),
		rationalSlab:   make([]ast.RationalNode, 0, defaultSlabChunkSize),
		ratValSlab:     make([]big.Rat, 0, defaultSlabChunkSize),
		addSlab:        make([]ast.AddNode, 0, defaultSlabChunkSize),
		mulSlab:        make([]ast.MulNode, 0, defaultSlabChunkSize),
		powSlab:        make([]ast.PowNode, 0, defaultSlabChunkSize),
		sqrtSlab:       make([]ast.SqrtNode, 0, defaultSlabChunkSize),
		unarySlab:      make([]ast.UnaryOpNode, 0, defaultSlabChunkSize),
		varSlab:        make([]ast.VarNode, 0, defaultSlabChunkSize),
		binaryPairSlab: make([][2]ast.Node, 0, defaultSlabChunkSize),
	}
}

// State returns the current lifecycle state of the arena.
func (a *NodeArena) State() ArenaState {
	return a.fsm.State()
}

// AllocRational bumps a RationalNode into the slab, avoiding big.Rat heap allocations.
func (a *NodeArena) AllocRational(num, denom int64) (*ast.RationalNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	if denom == 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.zero_division_denominator"))
	}

	var r *big.Rat
	if len(a.ratValSlab) < cap(a.ratValSlab) {
		idx := len(a.ratValSlab)
		a.ratValSlab = a.ratValSlab[:idx+1]
		r = &a.ratValSlab[idx]
		r.SetFrac64(num, denom)
	} else {
		r = new(big.Rat).SetFrac64(num, denom)
	}

	if len(a.rationalSlab) < cap(a.rationalSlab) {
		idx := len(a.rationalSlab)
		a.rationalSlab = a.rationalSlab[:idx+1]
		node := &a.rationalSlab[idx]
		node.Val = r
		return node, nil
	}

	node := &ast.RationalNode{Val: r}
	a.extraRationals = append(a.extraRationals, node)
	return node, nil
}

// AllocRationalFromBigRat bumps a RationalNode wrapping an existing big.Rat.
func (a *NodeArena) AllocRationalFromBigRat(r *big.Rat) (*ast.RationalNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	var copied *big.Rat
	if len(a.ratValSlab) < cap(a.ratValSlab) {
		idx := len(a.ratValSlab)
		a.ratValSlab = a.ratValSlab[:idx+1]
		copied = &a.ratValSlab[idx]
		copied.Set(r)
	} else {
		copied = new(big.Rat).Set(r)
	}

	if len(a.rationalSlab) < cap(a.rationalSlab) {
		idx := len(a.rationalSlab)
		a.rationalSlab = a.rationalSlab[:idx+1]
		node := &a.rationalSlab[idx]
		node.Val = copied
		return node, nil
	}

	node := &ast.RationalNode{Val: copied}
	a.extraRationals = append(a.extraRationals, node)
	return node, nil
}

// AllocBinaryAdd bumps an AddNode with 2 terms using pre-allocated slab slices to avoid slice allocations.
func (a *NodeArena) AllocBinaryAdd(left, right ast.Node) (*ast.AddNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	var terms []ast.Node
	if len(a.binaryPairSlab) < cap(a.binaryPairSlab) {
		idx := len(a.binaryPairSlab)
		a.binaryPairSlab = a.binaryPairSlab[:idx+1]
		a.binaryPairSlab[idx] = [2]ast.Node{left, right}
		terms = a.binaryPairSlab[idx][:]
	} else {
		terms = []ast.Node{left, right}
	}
	return a.AllocAdd(terms)
}

// AllocBinaryMul bumps a MulNode with 2 factors using pre-allocated slab slices to avoid slice allocations.
func (a *NodeArena) AllocBinaryMul(left, right ast.Node) (*ast.MulNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	var factors []ast.Node
	if len(a.binaryPairSlab) < cap(a.binaryPairSlab) {
		idx := len(a.binaryPairSlab)
		a.binaryPairSlab = a.binaryPairSlab[:idx+1]
		a.binaryPairSlab[idx] = [2]ast.Node{left, right}
		factors = a.binaryPairSlab[idx][:]
	} else {
		factors = []ast.Node{left, right}
	}
	return a.AllocMul(factors)
}

// AllocAdd bumps an AddNode into the slab.
func (a *NodeArena) AllocAdd(terms []ast.Node) (*ast.AddNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	if len(a.addSlab) < cap(a.addSlab) {
		idx := len(a.addSlab)
		a.addSlab = a.addSlab[:idx+1]
		node := &a.addSlab[idx]
		node.Terms = terms
		return node, nil
	}

	node := &ast.AddNode{Terms: terms}
	a.extraAdds = append(a.extraAdds, node)
	return node, nil
}

// AllocMul bumps a MulNode into the slab.
func (a *NodeArena) AllocMul(factors []ast.Node) (*ast.MulNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	if len(a.mulSlab) < cap(a.mulSlab) {
		idx := len(a.mulSlab)
		a.mulSlab = a.mulSlab[:idx+1]
		node := &a.mulSlab[idx]
		node.Factors = factors
		return node, nil
	}

	node := &ast.MulNode{Factors: factors}
	a.extraMuls = append(a.extraMuls, node)
	return node, nil
}

// AllocPow bumps a PowNode into the slab.
func (a *NodeArena) AllocPow(base, exp ast.Node) (*ast.PowNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	if len(a.powSlab) < cap(a.powSlab) {
		idx := len(a.powSlab)
		a.powSlab = a.powSlab[:idx+1]
		node := &a.powSlab[idx]
		node.Base = base
		node.Exp = exp
		return node, nil
	}

	node := &ast.PowNode{Base: base, Exp: exp}
	a.extraPows = append(a.extraPows, node)
	return node, nil
}

// AllocSqrt bumps a SqrtNode into the slab.
func (a *NodeArena) AllocSqrt(rad ast.Node) (*ast.SqrtNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	if len(a.sqrtSlab) < cap(a.sqrtSlab) {
		idx := len(a.sqrtSlab)
		a.sqrtSlab = a.sqrtSlab[:idx+1]
		node := &a.sqrtSlab[idx]
		node.Radicand = rad
		return node, nil
	}

	node := &ast.SqrtNode{Radicand: rad}
	a.extraSqrts = append(a.extraSqrts, node)
	return node, nil
}

// AllocUnary bumps a UnaryOpNode into the slab.
func (a *NodeArena) AllocUnary(op string, expr ast.Node) (*ast.UnaryOpNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	if len(a.unarySlab) < cap(a.unarySlab) {
		idx := len(a.unarySlab)
		a.unarySlab = a.unarySlab[:idx+1]
		node := &a.unarySlab[idx]
		node.Op = op
		node.Expr = expr
		return node, nil
	}

	node := &ast.UnaryOpNode{Op: op, Expr: expr}
	a.extraUnaries = append(a.extraUnaries, node)
	return node, nil
}

// AllocVar bumps a VarNode into the slab.
func (a *NodeArena) AllocVar(name string) (*ast.VarNode, error) {
	if err := a.fsm.CheckCanAllocate(); err != nil {
		return nil, err
	}
	if len(a.varSlab) < cap(a.varSlab) {
		idx := len(a.varSlab)
		a.varSlab = a.varSlab[:idx+1]
		node := &a.varSlab[idx]
		node.Name = name
		return node, nil
	}

	node := &ast.VarNode{Name: name}
	a.extraVars = append(a.extraVars, node)
	return node, nil
}

// Detach clones the expression node tree into independent heap-allocated nodes,
// decoupling the result from the arena's memory before Release() is called.
func (a *NodeArena) Detach(n ast.Node) ast.Node {
	if n == nil {
		return nil
	}
	_ = a.fsm.MarkDetached()
	return deepCloneNode(n)
}

// deepCloneNode performs an explicit deep copy of an AST node to the standard heap.
func deepCloneNode(n ast.Node) ast.Node {
	if n == nil {
		return nil
	}

	switch v := n.(type) {
	case *ast.RationalNode:
		return &ast.RationalNode{Val: new(big.Rat).Set(v.Val)}
	case *ast.VarNode:
		return &ast.VarNode{Name: v.Name}
	case *ast.SqrtNode:
		return &ast.SqrtNode{Radicand: deepCloneNode(v.Radicand)}
	case *ast.UnaryOpNode:
		return &ast.UnaryOpNode{Op: v.Op, Expr: deepCloneNode(v.Expr)}
	case *ast.PowNode:
		return &ast.PowNode{
			Base: deepCloneNode(v.Base),
			Exp:  deepCloneNode(v.Exp),
		}
	case *ast.AddNode:
		newTerms := make([]ast.Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = deepCloneNode(t)
		}
		return &ast.AddNode{Terms: newTerms}
	case *ast.MulNode:
		newFactors := make([]ast.Node, len(v.Factors))
		for i, f := range v.Factors {
			newFactors[i] = deepCloneNode(f)
		}
		return &ast.MulNode{Factors: newFactors}
	case *ast.FuncNode:
		newArgs := make([]ast.Node, len(v.Args))
		for i, arg := range v.Args {
			newArgs[i] = deepCloneNode(arg)
		}
		return &ast.FuncNode{Name: v.Name, Args: newArgs}
	case *ast.RelOpNode:
		return &ast.RelOpNode{
			Op:  v.Op,
			LHS: deepCloneNode(v.LHS),
			RHS: deepCloneNode(v.RHS),
		}
	default:
		// For other types (e.g. PolyNode, AlgebraicNumberNode, ConstNode), return directly
		return n
	}
}

// Release clears the arena memory buffers and transitions state to Released.
func (a *NodeArena) Release() error {
	if err := a.fsm.Release(); err != nil {
		return err
	}

	// Zero out slice elements to avoid holding GC references, but preserve capacity
	clear(a.rationalSlab)
	a.rationalSlab = a.rationalSlab[:0]

	clear(a.ratValSlab)
	a.ratValSlab = a.ratValSlab[:0]

	clear(a.addSlab)
	a.addSlab = a.addSlab[:0]

	clear(a.mulSlab)
	a.mulSlab = a.mulSlab[:0]

	clear(a.powSlab)
	a.powSlab = a.powSlab[:0]

	clear(a.sqrtSlab)
	a.sqrtSlab = a.sqrtSlab[:0]

	clear(a.unarySlab)
	a.unarySlab = a.unarySlab[:0]

	clear(a.varSlab)
	a.varSlab = a.varSlab[:0]

	clear(a.binaryPairSlab)
	a.binaryPairSlab = a.binaryPairSlab[:0]

	a.extraRationals = nil
	a.extraAdds = nil
	a.extraMuls = nil
	a.extraPows = nil
	a.extraSqrts = nil
	a.extraUnaries = nil
	a.extraVars = nil

	return nil
}

// RecycleArena releases the arena and returns it to the global pool for future operations.
func RecycleArena(a *NodeArena) error {
	if a == nil {
		return nil
	}
	if a.State() != ArenaStateReleased {
		if err := a.Release(); err != nil {
			return err
		}
	}
	nodeArenaPool.Put(a)
	return nil
}
