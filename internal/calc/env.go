package calc

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"sync"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

// Env represents a symbol table mapping variable names to AST nodes and assumptions.
type Env struct {
	mu             sync.RWMutex
	vars           map[string]Node
	evaluatingVars map[string]bool
	assumptions    *AssumptionStore
	LastCert       *VerificationCertificate
	WarnIterations *big.Int
	MaxIterations  *big.Int
	WarnWriter     io.Writer
	Tracer         *provenance.TraceRecorder
	LastTrace      *provenance.TraceRecorder
}

// NewEnv creates a new, empty Env instance.
func NewEnv() *Env {
	return &Env{
		vars:           make(map[string]Node),
		evaluatingVars: make(map[string]bool),
		assumptions:    NewAssumptionStore(),
		WarnWriter:     os.Stderr,
	}
}

// GetTracer safely returns the active provenance tracer or nil.
func (e *Env) GetTracer() *provenance.TraceRecorder {
	if e == nil {
		return nil
	}
	return e.Tracer
}

// GetLastTrace returns the trace recorder from the most recent evaluation pass.
func (e *Env) GetLastTrace() *provenance.TraceRecorder {
	if e == nil {
		return nil
	}
	return e.LastTrace
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

// SetIterationLimits sets loop warning/max limits and warning writer on this environment.
func (e *Env) SetIterationLimits(warnLimit, maxLimit *big.Int, warnWriter io.Writer) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if warnLimit != nil {
		e.WarnIterations = new(big.Int).Set(warnLimit)
	}
	if maxLimit != nil {
		e.MaxIterations = new(big.Int).Set(maxLimit)
	}
	if warnWriter != nil {
		e.WarnWriter = warnWriter
	}
}

// GetIterationLimits returns loop warning/max limits and warning writer, falling back to defaults if not set.
func (e *Env) GetIterationLimits() (*big.Int, *big.Int, io.Writer) {
	if e == nil {
		return big.NewInt(100000), big.NewInt(-1), os.Stderr
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	warn := e.WarnIterations
	max := e.MaxIterations
	writer := e.WarnWriter
	if writer == nil {
		writer = os.Stderr
	}
	return warn, max, writer
}

// IsPositive checks if variable name has positive assumption.
func (e *Env) IsPositive(name string) bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.assumptions == nil {
		return false
	}
	return e.assumptions.IsPositive(name)
}

// IsNegative checks if variable name has negative assumption.
func (e *Env) IsNegative(name string) bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.assumptions == nil {
		return false
	}
	return e.assumptions.IsNegative(name)
}

// IsNonNegative checks if variable name has non-negative assumption.
func (e *Env) IsNonNegative(name string) bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.assumptions == nil {
		return false
	}
	return e.assumptions.IsNonNegative(name)
}

// IsNonPositive checks if variable name has non-positive assumption.
func (e *Env) IsNonPositive(name string) bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.assumptions == nil {
		return false
	}
	return e.assumptions.IsNonPositive(name)
}

// IsInteger checks if variable name has integer assumption.
func (e *Env) IsInteger(name string) bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.assumptions == nil {
		return false
	}
	return e.assumptions.IsInteger(name)
}

// IsEven checks if variable name has even assumption.
func (e *Env) IsEven(name string) bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.assumptions == nil {
		return false
	}
	return e.assumptions.IsEven(name)
}

// IsOdd checks if variable name has odd assumption.
func (e *Env) IsOdd(name string) bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.assumptions == nil {
		return false
	}
	return e.assumptions.IsOdd(name)
}

// IsReal checks if variable name has real assumption.
func (e *Env) IsReal(name string) bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.assumptions == nil {
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

// IsEvaluatingVar returns true if the variable is currently being evaluated (cycle detection).
func (e *Env) IsEvaluatingVar(name string) bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.evaluatingVars != nil && e.evaluatingVars[name]
}

// PushEvaluatingVar marks a variable as currently being evaluated.
func (e *Env) PushEvaluatingVar(name string) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.evaluatingVars == nil {
		e.evaluatingVars = make(map[string]bool)
	}
	e.evaluatingVars[name] = true
}

// PopEvaluatingVar unmarks a variable after evaluation finishes.
func (e *Env) PopEvaluatingVar(name string) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.evaluatingVars != nil {
		delete(e.evaluatingVars, name)
	}
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
	if e.WarnIterations != nil {
		copied.WarnIterations = new(big.Int).Set(e.WarnIterations)
	}
	if e.MaxIterations != nil {
		copied.MaxIterations = new(big.Int).Set(e.MaxIterations)
	}
	copied.WarnWriter = e.WarnWriter
	return copied
}
