package ast

import (
	"fmt"
	"strings"
)

// -------------------------------------------------------------------------
// RelOpNode (Relational Operation: >, <, >=, <=)
// -------------------------------------------------------------------------

// RelOpNode represents a binary relational operation such as x > 0 or n <= 5.
type RelOpNode struct {
	LHS Node
	Op  string
	RHS Node
}

// NewRelOp creates a new RelOpNode.
func NewRelOp(lhs Node, op string, rhs Node) *RelOpNode {
	return &RelOpNode{LHS: lhs, Op: op, RHS: rhs}
}

func (n *RelOpNode) Type() NodeType { return NodeRelOp }
func (n *RelOpNode) String() string {
	return fmt.Sprintf("%s %s %s", n.LHS.String(), n.Op, n.RHS.String())
}
func (n *RelOpNode) Equal(other Node) bool {
	o, ok := other.(*RelOpNode)
	if !ok {
		return false
	}
	return n.Op == o.Op && n.LHS.Equal(o.LHS) && n.RHS.Equal(o.RHS)
}

// -------------------------------------------------------------------------
// QuantifierNode (First-Order Logic: forall, exists)
// -------------------------------------------------------------------------

// QuantifierKind defines the type of logical quantifier (forall or exists).
type QuantifierKind int

const (
	// QuantifierForall represents universal quantification (forall / ∀).
	QuantifierForall QuantifierKind = iota
	// QuantifierExists represents existential quantification (exists / ∃).
	QuantifierExists
)

func (k QuantifierKind) String() string {
	switch k {
	case QuantifierForall:
		return "forall"
	case QuantifierExists:
		return "exists"
	default:
		return "unknown_quantifier"
	}
}

// QuantifierNode represents a first-order logic quantified formula such as forall([x], phi) or exists([x, y], phi).
type QuantifierNode struct {
	Kind QuantifierKind
	Vars []string
	Body Node
}

// NewQuantifier constructs a new QuantifierNode.
func NewQuantifier(kind QuantifierKind, vars []string, body Node) *QuantifierNode {
	vCopy := make([]string, len(vars))
	copy(vCopy, vars)
	return &QuantifierNode{
		Kind: kind,
		Vars: vCopy,
		Body: body,
	}
}

func (n *QuantifierNode) Type() NodeType { return NodeQuantifier }

func (n *QuantifierNode) String() string {
	var sb strings.Builder
	sb.WriteString(n.Kind.String())
	sb.WriteString("([")
	sb.WriteString(strings.Join(n.Vars, ", "))
	sb.WriteString("], ")
	if n.Body != nil {
		sb.WriteString(n.Body.String())
	}
	sb.WriteString(")")
	return sb.String()
}

func (n *QuantifierNode) Equal(other Node) bool {
	o, ok := other.(*QuantifierNode)
	if !ok || n.Kind != o.Kind || len(n.Vars) != len(o.Vars) {
		return false
	}
	for i, v := range n.Vars {
		if v != o.Vars[i] {
			return false
		}
	}
	if n.Body == nil && o.Body == nil {
		return true
	}
	if n.Body == nil || o.Body == nil {
		return false
	}
	return n.Body.Equal(o.Body)
}
