package calc

import "sync"

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
