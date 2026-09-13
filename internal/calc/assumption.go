package calc

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Property Lattice & Constants (Bitmask)
// -------------------------------------------------------------------------

// Property represents a mathematical domain property or constraint as a bitmask.
type Property uint32

const (
	PropNone Property = 0

	// Type domain properties
	PropComplex  Property = 1 << 0
	PropReal     Property = 1 << 1
	PropRational Property = 1 << 2
	PropInteger  Property = 1 << 3
	PropEven     Property = 1 << 4
	PropOdd      Property = 1 << 5

	// Sign & order properties
	PropZero        Property = 1 << 6
	PropNonZero     Property = 1 << 7
	PropPositive    Property = 1 << 8
	PropNegative    Property = 1 << 9
	PropNonNegative Property = 1 << 10
	PropNonPositive Property = 1 << 11
)

// Ternary represents a three-valued logic state (Kleene Strong Logic: Unknown, True, False).
type Ternary int

const (
	TernaryUnknown Ternary = iota
	TernaryTrue
	TernaryFalse
)

// AssumptionState represents the FSM state of a variable's assumptions.
type AssumptionState int

const (
	StateUnconstrained AssumptionState = iota
	StateConstrained
)

// -------------------------------------------------------------------------
// VariableAssumptions (Per-Variable State Machine & Deduction)
// -------------------------------------------------------------------------

// VariableAssumptions holds the assumptions and state machine for a single variable.
type VariableAssumptions struct {
	Name  string
	State AssumptionState
	Props Property
}

// NewVariableAssumptions creates an unconstrained variable assumption state.
func NewVariableAssumptions(name string) *VariableAssumptions {
	return &VariableAssumptions{
		Name:  name,
		State: StateUnconstrained,
		Props: PropNone,
	}
}

// Deduce propagates mathematical implications through the property lattice.
func (va *VariableAssumptions) Deduce() {
	p := va.Props

	// Positive implies NonNegative, NonZero, Real, Complex
	if p&PropPositive != 0 {
		p |= PropNonNegative | PropNonZero | PropReal | PropComplex
	}
	// Negative implies NonPositive, NonZero, Real, Complex
	if p&PropNegative != 0 {
		p |= PropNonPositive | PropNonZero | PropReal | PropComplex
	}
	// Zero implies NonNegative, NonPositive, Real, Complex
	if p&PropZero != 0 {
		p |= PropNonNegative | PropNonPositive | PropReal | PropComplex
	}
	// NonNegative or NonPositive or NonZero implies Real, Complex
	if p&(PropNonNegative|PropNonPositive|PropNonZero) != 0 {
		p |= PropReal | PropComplex
	}
	// Even implies Integer, Rational, Real, Complex
	if p&PropEven != 0 {
		p |= PropInteger | PropRational | PropReal | PropComplex
	}
	// Odd implies Integer, Rational, Real, Complex
	if p&PropOdd != 0 {
		p |= PropInteger | PropRational | PropReal | PropComplex
	}
	// Integer implies Rational, Real, Complex
	if p&PropInteger != 0 {
		p |= PropRational | PropReal | PropComplex
	}
	// Rational implies Real, Complex
	if p&PropRational != 0 {
		p |= PropReal | PropComplex
	}
	// Real implies Complex
	if p&PropReal != 0 {
		p |= PropComplex
	}

	va.Props = p
}

// IsConsistent checks whether the combined properties contain any mathematical contradiction.
func (va *VariableAssumptions) IsConsistent() bool {
	p := va.Props

	// Positive and Negative are mutually exclusive
	if p&PropPositive != 0 && p&PropNegative != 0 {
		return false
	}
	// Positive and Zero are mutually exclusive
	if p&PropPositive != 0 && p&PropZero != 0 {
		return false
	}
	// Negative and Zero are mutually exclusive
	if p&PropNegative != 0 && p&PropZero != 0 {
		return false
	}
	// Zero and NonZero are mutually exclusive
	if p&PropZero != 0 && p&PropNonZero != 0 {
		return false
	}
	// Even and Odd are mutually exclusive
	if p&PropEven != 0 && p&PropOdd != 0 {
		return false
	}

	return true
}

// -------------------------------------------------------------------------
// AssumptionStore (Thread-safe registry of variable constraints)
// -------------------------------------------------------------------------

// AssumptionStore manages domain constraints for all variables in an environment.
type AssumptionStore struct {
	mu   sync.RWMutex
	vars map[string]*VariableAssumptions
}

// NewAssumptionStore creates a new, empty AssumptionStore.
func NewAssumptionStore() *AssumptionStore {
	return &AssumptionStore{
		vars: make(map[string]*VariableAssumptions),
	}
}

// Clone creates an independent deep copy of the assumption store.
func (s *AssumptionStore) Clone() *AssumptionStore {
	if s == nil {
		return NewAssumptionStore()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	copied := NewAssumptionStore()
	for k, v := range s.vars {
		copied.vars[k] = &VariableAssumptions{
			Name:  v.Name,
			State: v.State,
			Props: v.Props,
		}
	}
	return copied
}

// Assume registers a property constraint for a variable.
// Uses FSM transition: StateUnconstrained -> StateConstrained (or stays StateConstrained).
func (s *AssumptionStore) Assume(varName string, prop Property) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	va, ok := s.vars[varName]
	if !ok {
		va = NewVariableAssumptions(varName)
		s.vars[varName] = va
	}

	// Backup existing properties
	oldProps := va.Props
	oldState := va.State

	// Apply new constraint and deduce
	va.Props |= prop
	va.Deduce()

	if !va.IsConsistent() {
		// Rollback on inconsistency
		va.Props = oldProps
		va.State = oldState
		return fmt.Errorf("%s", i18n.T("errors.assume_inconsistent", varName))
	}

	va.State = StateConstrained
	return nil
}

// Unassume removes all assumptions for a variable, resetting its state to StateUnconstrained.
func (s *AssumptionStore) Unassume(varName string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.vars, varName)
}

// ClearAll removes all assumptions across all variables.
func (s *AssumptionStore) ClearAll() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.vars = make(map[string]*VariableAssumptions)
}

// Query checks whether a variable satisfies a given property using 3-valued logic.
func (s *AssumptionStore) Query(varName string, prop Property) Ternary {
	if s == nil {
		return TernaryUnknown
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	va, ok := s.vars[varName]
	if !ok || va.State != StateConstrained {
		return TernaryUnknown
	}

	if va.Props&prop == prop {
		return TernaryTrue
	}

	// Check if the negation is proven
	switch prop {
	case PropPositive:
		if va.Props&PropNonPositive != 0 || va.Props&PropNegative != 0 || va.Props&PropZero != 0 {
			return TernaryFalse
		}
	case PropNegative:
		if va.Props&PropNonNegative != 0 || va.Props&PropPositive != 0 || va.Props&PropZero != 0 {
			return TernaryFalse
		}
	case PropZero:
		if va.Props&PropNonZero != 0 || va.Props&PropPositive != 0 || va.Props&PropNegative != 0 {
			return TernaryFalse
		}
	case PropNonZero:
		if va.Props&PropZero != 0 {
			return TernaryFalse
		}
	case PropEven:
		if va.Props&PropOdd != 0 {
			return TernaryFalse
		}
	case PropOdd:
		if va.Props&PropEven != 0 {
			return TernaryFalse
		}
	}

	return TernaryUnknown
}

// Convenience query methods
func (s *AssumptionStore) IsPositive(varName string) bool {
	return s.Query(varName, PropPositive) == TernaryTrue
}

func (s *AssumptionStore) IsNegative(varName string) bool {
	return s.Query(varName, PropNegative) == TernaryTrue
}

func (s *AssumptionStore) IsNonNegative(varName string) bool {
	return s.Query(varName, PropNonNegative) == TernaryTrue
}

func (s *AssumptionStore) IsNonPositive(varName string) bool {
	return s.Query(varName, PropNonPositive) == TernaryTrue
}

func (s *AssumptionStore) IsInteger(varName string) bool {
	return s.Query(varName, PropInteger) == TernaryTrue
}

func (s *AssumptionStore) IsEven(varName string) bool {
	return s.Query(varName, PropEven) == TernaryTrue
}

func (s *AssumptionStore) IsOdd(varName string) bool {
	return s.Query(varName, PropOdd) == TernaryTrue
}

func (s *AssumptionStore) IsReal(varName string) bool {
	return s.Query(varName, PropReal) == TernaryTrue
}

// ParseProperty parses a string or identifier into a Property bitmask.
func ParseProperty(name string) (Property, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "positive", "pos", ">0":
		return PropPositive, nil
	case "negative", "neg", "<0":
		return PropNegative, nil
	case "nonnegative", "non_negative", ">=0":
		return PropNonNegative, nil
	case "nonpositive", "non_positive", "<=0":
		return PropNonPositive, nil
	case "zero", "==0", "=0":
		return PropZero, nil
	case "nonzero", "non_zero", "!=0":
		return PropNonZero, nil
	case "integer", "int":
		return PropInteger, nil
	case "even":
		return PropEven, nil
	case "odd":
		return PropOdd, nil
	case "rational", "rat":
		return PropRational, nil
	case "real":
		return PropReal, nil
	case "complex":
		return PropComplex, nil
	default:
		return PropNone, fmt.Errorf("%s", i18n.T("errors.assume_unsupported", name))
	}
}

// FormatVariableAssumptions formats the properties of a variable into human-readable strings.
func (va *VariableAssumptions) Strings() []string {
	var res []string
	p := va.Props

	if p&PropPositive != 0 {
		res = append(res, "positive")
	} else if p&PropNegative != 0 {
		res = append(res, "negative")
	} else if p&PropZero != 0 {
		res = append(res, "zero")
	} else {
		if p&PropNonNegative != 0 {
			res = append(res, "nonnegative")
		}
		if p&PropNonPositive != 0 {
			res = append(res, "nonpositive")
		}
		if p&PropNonZero != 0 {
			res = append(res, "nonzero")
		}
	}

	if p&PropEven != 0 {
		res = append(res, "even")
	} else if p&PropOdd != 0 {
		res = append(res, "odd")
	} else if p&PropInteger != 0 {
		res = append(res, "integer")
	} else if p&PropRational != 0 {
		res = append(res, "rational")
	} else if p&PropReal != 0 {
		res = append(res, "real")
	} else if p&PropComplex != 0 {
		res = append(res, "complex")
	}

	return res
}

// ListAssumptions returns formatted strings of all active assumptions.
func (s *AssumptionStore) ListAssumptions() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var varNames []string
	for k := range s.vars {
		varNames = append(varNames, k)
	}
	sort.Strings(varNames)

	var res []string
	for _, name := range varNames {
		va := s.vars[name]
		if va.State == StateConstrained {
			props := va.Strings()
			if len(props) > 0 {
				res = append(res, fmt.Sprintf("%s: %s", name, strings.Join(props, ", ")))
			}
		}
	}
	return res
}
