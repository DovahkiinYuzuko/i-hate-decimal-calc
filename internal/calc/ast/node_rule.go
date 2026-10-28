package ast

import "fmt"

// WildcardNode represents a pattern matching wildcard variable (e.g., _x, _a, _1).
type WildcardNode struct {
	Name string
}

// NewWildcardNode creates a new WildcardNode with the given name.
func NewWildcardNode(name string) *WildcardNode {
	return &WildcardNode{Name: name}
}

// Type returns NodeWildcard.
func (w *WildcardNode) Type() NodeType {
	return NodeWildcard
}

// String returns the wildcard variable name (e.g. "_x").
func (w *WildcardNode) String() string {
	return w.Name
}

// Equal returns true if other is a WildcardNode with the same Name.
func (w *WildcardNode) Equal(other Node) bool {
	if o, ok := other.(*WildcardNode); ok {
		return w.Name == o.Name
	}
	return false
}

// RuleNode represents an algebraic rewrite rule: LHS -> RHS (e.g., rule(_x^2, 1)).
type RuleNode struct {
	LHS Node
	RHS Node
}

// NewRuleNode creates a new RuleNode.
func NewRuleNode(lhs, rhs Node) *RuleNode {
	return &RuleNode{LHS: lhs, RHS: rhs}
}

// Type returns NodeRule.
func (r *RuleNode) Type() NodeType {
	return NodeRule
}

// String returns a human-readable representation of the rule.
func (r *RuleNode) String() string {
	var lhsStr, rhsStr string
	if r.LHS != nil {
		lhsStr = r.LHS.String()
	} else {
		lhsStr = "<nil>"
	}
	if r.RHS != nil {
		rhsStr = r.RHS.String()
	} else {
		rhsStr = "<nil>"
	}
	return fmt.Sprintf("rule(%s, %s)", lhsStr, rhsStr)
}

// Equal returns true if both LHS and RHS are equal.
func (r *RuleNode) Equal(other Node) bool {
	if o, ok := other.(*RuleNode); ok {
		if (r.LHS == nil) != (o.LHS == nil) {
			return false
		}
		if r.LHS != nil && !r.LHS.Equal(o.LHS) {
			return false
		}
		if (r.RHS == nil) != (o.RHS == nil) {
			return false
		}
		if r.RHS != nil && !r.RHS.Equal(o.RHS) {
			return false
		}
		return true
	}
	return false
}
