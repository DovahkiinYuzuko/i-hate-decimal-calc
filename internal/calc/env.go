package calc

import (
	"fmt"
	"sync"
)

// Env represents a symbol table mapping variable names to AST nodes and assumptions.
type Env struct {
	mu          sync.RWMutex
	vars        map[string]Node
	assumptions *AssumptionStore
	LastCert    *VerificationCertificate
}

// NewEnv creates a new, empty Env instance.
func NewEnv() *Env {
	return &Env{
		vars:        make(map[string]Node),
		assumptions: NewAssumptionStore(),
	}
}

// Assumptions returns the assumption store of this environment.
func (e *Env) Assumptions() *AssumptionStore {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.assumptions == nil {
		e.assumptions = NewAssumptionStore()
	}
	return e.assumptions
}

// IsPositive checks if variable name has positive assumption.
func (e *Env) IsPositive(name string) bool {
	if e == nil || e.assumptions == nil {
		return false
	}
	return e.assumptions.IsPositive(name)
}

// IsNegative checks if variable name has negative assumption.
func (e *Env) IsNegative(name string) bool {
	if e == nil || e.assumptions == nil {
		return false
	}
	return e.assumptions.IsNegative(name)
}

// IsNonNegative checks if variable name has non-negative assumption.
func (e *Env) IsNonNegative(name string) bool {
	if e == nil || e.assumptions == nil {
		return false
	}
	return e.assumptions.IsNonNegative(name)
}

// IsNonPositive checks if variable name has non-positive assumption.
func (e *Env) IsNonPositive(name string) bool {
	if e == nil || e.assumptions == nil {
		return false
	}
	return e.assumptions.IsNonPositive(name)
}

// IsInteger checks if variable name has integer assumption.
func (e *Env) IsInteger(name string) bool {
	if e == nil || e.assumptions == nil {
		return false
	}
	return e.assumptions.IsInteger(name)
}

// IsEven checks if variable name has even assumption.
func (e *Env) IsEven(name string) bool {
	if e == nil || e.assumptions == nil {
		return false
	}
	return e.assumptions.IsEven(name)
}

// IsOdd checks if variable name has odd assumption.
func (e *Env) IsOdd(name string) bool {
	if e == nil || e.assumptions == nil {
		return false
	}
	return e.assumptions.IsOdd(name)
}

// IsReal checks if variable name has real assumption.
func (e *Env) IsReal(name string) bool {
	if e == nil || e.assumptions == nil {
		return false
	}
	return e.assumptions.IsReal(name)
}

// Get returns the AST node bound to the given variable name.
func (e *Env) Get(name string) (Node, bool) {
	if e == nil {
		return nil, false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	val, ok := e.vars[name]
	return val, ok
}

// Set binds a variable name to an AST node.
func (e *Env) Set(name string, val Node) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.vars[name] = val
}

// Delete removes a variable binding from the environment.
func (e *Env) Delete(name string) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.vars, name)
}

// WithScopedVar executes fn within a dynamically scoped context where name is temporarily
// bound to initialVal. The callback setVar can be called during iteration to update the variable in-place.
// Upon return (or panic/error), the previous binding of name is guaranteed to be restored.
func (e *Env) WithScopedVar(name string, initialVal Node, fn func(setVar func(Node)) error) error {
	if e == nil {
		return fmt.Errorf("nil environment")
	}

	e.mu.Lock()
	if e.vars == nil {
		e.vars = make(map[string]Node)
	}
	prevVal, existed := e.vars[name]
	e.vars[name] = initialVal
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		if existed {
			e.vars[name] = prevVal
		} else {
			delete(e.vars, name)
		}
		prevVal = nil
		e.mu.Unlock()
	}()

	setVar := func(next Node) {
		e.mu.Lock()
		e.vars[name] = next
		e.mu.Unlock()
	}

	return fn(setVar)
}


// All returns a shallow copy of all variable bindings.
func (e *Env) All() map[string]Node {
	if e == nil {
		return make(map[string]Node)
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make(map[string]Node, len(e.vars))
	for k, v := range e.vars {
		res[k] = v
	}
	return res
}

// Clone creates an independent copy of the environment.
func (e *Env) Clone() *Env {
	if e == nil {
		return NewEnv()
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	copied := NewEnv()
	for k, v := range e.vars {
		copied.vars[k] = v
	}
	if e.assumptions != nil {
		copied.assumptions = e.assumptions.Clone()
	}
	return copied
}
