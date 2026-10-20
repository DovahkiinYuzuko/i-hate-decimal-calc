package provenance

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// TraceRecorder manages scoped provenance recording for an evaluation pass.
// A nil *TraceRecorder acts as a no-op with true zero-overhead on all methods.
type TraceRecorder struct {
	root    *DerivationNode
	current *DerivationNode
	stack   []*DerivationNode
	nextID  int64
}

// NewTraceRecorder creates a fresh active TraceRecorder.
func NewTraceRecorder() *TraceRecorder {
	return &TraceRecorder{
		nextID: 1,
	}
}

// Root returns the top-level derivation node, if recorded.
func (r *TraceRecorder) Root() *DerivationNode {
	if r == nil {
		return nil
	}
	return r.root
}

// StartScope begins a new parent derivation node for a compound operation.
// Subsequent RecordStep calls will be attached as children until EndScope is called.
func (r *TraceRecorder) StartScope(ruleID string, rel RelationKind, path []PathElem, before ast.Node, vis Visibility) {
	if r == nil {
		return
	}
	node := &DerivationNode{
		ID:       r.nextID,
		RuleID:   ruleID,
		Relation: rel,
		Path:     path,
		Before:   before,
		Trust:    TrustCertified,
		Vis:      vis,
		Children: make([]*DerivationNode, 0, 4),
	}
	r.nextID++

	if r.current != nil {
		r.current.Children = append(r.current.Children, node)
		r.stack = append(r.stack, r.current)
	} else if r.root == nil {
		r.root = node
	}
	r.current = node
}

// EndScope finalizes the current scoped derivation node with its final expression.
func (r *TraceRecorder) EndScope(after ast.Node, cert AlgebraicCertificate) {
	if r == nil || r.current == nil {
		return
	}
	r.current.After = after
	r.current.Certificate = cert

	n := len(r.stack)
	if n > 0 {
		r.current = r.stack[n-1]
		r.stack = r.stack[:n-1]
	} else {
		r.current = nil
	}
}

// RecordStep records an atomic rewrite step under the current scope.
func (r *TraceRecorder) RecordStep(
	ruleID string,
	rel RelationKind,
	path []PathElem,
	before ast.Node,
	after ast.Node,
	params map[string]ast.Node,
	conditions []SideCondition,
	trust TrustLevel,
	vis Visibility,
) {
	if r == nil {
		return
	}
	step := &DerivationNode{
		ID:         r.nextID,
		RuleID:     ruleID,
		Relation:   rel,
		Path:       path,
		Before:     before,
		After:      after,
		Parameters: params,
		Conditions: conditions,
		Trust:      trust,
		Vis:        vis,
	}
	r.nextID++

	if r.current != nil {
		r.current.Children = append(r.current.Children, step)
	} else if r.root == nil {
		r.root = step
	} else {
		// If root exists as an atomic node without a scope, wrap it in a parent
		parent := &DerivationNode{
			ID:       r.nextID,
			RuleID:   "CompoundDerivation",
			Relation: RelationEq,
			Before:   r.root.Before,
			After:    after,
			Trust:    TrustCertified,
			Vis:      VisPresentable,
			Children: []*DerivationNode{r.root, step},
		}
		r.nextID++
		r.root = parent
		r.current = parent
	}
}

// RecordNormalize records an algebraic normalization step (term sorting, associativity, commutativity)
// as a collapsed structural node to prevent derivation tree explosion (FMCAD 2022).
func (r *TraceRecorder) RecordNormalize(path []PathElem, before, after ast.Node) {
	if r == nil {
		return
	}
	r.RecordStep(
		RuleAlgebraNormalize,
		RelationEq,
		path,
		before,
		after,
		nil,
		nil,
		TrustCertified,
		VisStructural,
	)
}

// AttachSubtree attaches an independently recorded child tree into the current scope.
// This is used for parallel evaluations where subtasks produce their own TraceRecorder.
func (r *TraceRecorder) AttachSubtree(child *DerivationNode) {
	if r == nil || child == nil {
		return
	}
	if r.current != nil {
		r.current.Children = append(r.current.Children, child)
	} else if r.root == nil {
		r.root = child
	} else {
		parent := &DerivationNode{
			ID:       r.nextID,
			RuleID:   "ParallelComposite",
			Relation: RelationEq,
			Before:   r.root.Before,
			After:    child.After,
			Trust:    TrustCertified,
			Vis:      VisPresentable,
			Children: []*DerivationNode{r.root, child},
		}
		r.nextID++
		r.root = parent
		r.current = parent
	}
}
