package calc

import (
	"fmt"
	"sort"
	"strings"
)

// ConditionKind identifies the mathematical domain restriction.
type ConditionKind int

const (
	CondNonZero ConditionKind = iota // expr != 0
	CondNonNegative                  // expr >= 0
	CondPositive                     // expr > 0
)

// DomainCondition represents a single domain side condition.
type DomainCondition struct {
	Kind ConditionKind
	Expr Node
}

// String returns human-readable representation of a single condition.
func (c DomainCondition) String() string {
	exprStr := ""
	if c.Expr != nil {
		exprStr = c.Expr.String()
	}
	switch c.Kind {
	case CondNonZero:
		return fmt.Sprintf("%s != 0", exprStr)
	case CondNonNegative:
		return fmt.Sprintf("%s >= 0", exprStr)
	case CondPositive:
		return fmt.Sprintf("%s > 0", exprStr)
	default:
		return fmt.Sprintf("%s (cond %d)", exprStr, c.Kind)
	}
}

// DomainConditionSet stores a deduplicated set of domain conditions.
type DomainConditionSet struct {
	conditions []DomainCondition
}

// NewDomainConditionSet creates an empty DomainConditionSet.
func NewDomainConditionSet() *DomainConditionSet {
	return &DomainConditionSet{
		conditions: make([]DomainCondition, 0),
	}
}

// Add adds a condition to the set if not already present.
func (s *DomainConditionSet) Add(kind ConditionKind, expr Node) {
	if s == nil || expr == nil {
		return
	}
	// Check for syntactic equality to deduplicate
	for _, c := range s.conditions {
		if c.Kind == kind && c.Expr != nil && c.Expr.Equal(expr) {
			return
		}
	}
	s.conditions = append(s.conditions, DomainCondition{
		Kind: kind,
		Expr: expr,
	})
}

// Merge merges another DomainConditionSet into this set.
func (s *DomainConditionSet) Merge(other *DomainConditionSet) {
	if s == nil || other == nil {
		return
	}
	for _, c := range other.conditions {
		s.Add(c.Kind, c.Expr)
	}
}

// Conditions returns a copy of the conditions in a deterministic order.
func (s *DomainConditionSet) Conditions() []DomainCondition {
	if s == nil || len(s.conditions) == 0 {
		return nil
	}
	res := make([]DomainCondition, len(s.conditions))
	copy(res, s.conditions)
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].Kind != res[j].Kind {
			return res[i].Kind < res[j].Kind
		}
		return res[i].String() < res[j].String()
	})
	return res
}

// Len returns the count of conditions.
func (s *DomainConditionSet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.conditions)
}

// IsEmpty returns true if there are no conditions.
func (s *DomainConditionSet) IsEmpty() bool {
	return s.Len() == 0
}

// Clone returns a shallow copy of the condition set.
func (s *DomainConditionSet) Clone() *DomainConditionSet {
	if s == nil {
		return NewDomainConditionSet()
	}
	cloned := NewDomainConditionSet()
	for _, c := range s.conditions {
		cloned.conditions = append(cloned.conditions, c)
	}
	return cloned
}

// String returns formatted conditions separated by comma.
func (s *DomainConditionSet) String() string {
	if s == nil || len(s.conditions) == 0 {
		return ""
	}
	conds := s.Conditions()
	parts := make([]string, len(conds))
	for i, c := range conds {
		parts[i] = c.String()
	}
	return strings.Join(parts, ", ")
}
