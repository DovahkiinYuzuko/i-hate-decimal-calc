package calc

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"sync"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

// UserFunction represents a user-defined function in the environment.
type UserFunction struct {
	Name      string
	Params    []string
	Body      Node
	Bytecode  *Bytecode
}

// Env represents a symbol table mapping variable names to AST nodes and assumptions.
type Env struct {
	mu             sync.RWMutex
	vars           map[string]Node
	funcs          map[string]*UserFunction
	evaluatingVars map[string]bool
	callDepth      int
	memoCache      map[string]Node
	importStack    map[string]bool
	assumptions    *AssumptionStore
	LastCert       *VerificationCertificate
	WarnIterations *big.Int
	MaxIterations  *big.Int
	WarnWriter     io.Writer
	Tracer         *provenance.TraceRecorder
	LastTrace      *provenance.TraceRecorder
	SearchTracer   SearchTraceCollector
	Budget         *ExecutionBudget
}

// MaxCallDepth defines the maximum permitted call stack recursion depth for user functions.
const MaxCallDepth = 500

// NewEnv creates a new, empty Env instance.
func NewEnv() *Env {
	return &Env{
		vars:           make(map[string]Node),
		funcs:          make(map[string]*UserFunction),
		evaluatingVars: make(map[string]bool),
		memoCache:      make(map[string]Node),
		importStack:    make(map[string]bool),
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

// GetBudget returns the active execution budget or nil.
func (e *Env) GetBudget() *ExecutionBudget {
	if e == nil {
		return nil
	}
	return e.Budget
}

// SetBudget sets the active execution budget.
func (e *Env) SetBudget(b *ExecutionBudget) {
	if e != nil {
		e.Budget = b
	}
}

// GetSearchTracer returns the active algorithm search trace collector or nil.
func (e *Env) GetSearchTracer() SearchTraceCollector {
	if e == nil {
		return nil
	}
	return e.SearchTracer
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

// Clear removes all variables and resets all assumptions.
func (e *Env) Clear() {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.vars = make(map[string]Node)
	e.evaluatingVars = make(map[string]bool)
	if e.assumptions != nil {
		e.assumptions.ClearAll()
	} else {
		e.assumptions = NewAssumptionStore()
	}
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
	for k, v := range e.funcs {
		copied.funcs[k] = v
	}
	return copied
}

func isEnvReservedConst(name string) bool {
	switch name {
	case "pi", "e", "i", "deg":
		return true
	default:
		return false
	}
}

// SetUserFunc registers a user-defined function in the environment.
func (e *Env) SetUserFunc(name string, params []string, body Node) error {
	if e == nil {
		return fmt.Errorf("environment is nil")
	}
	if isEnvReservedConst(name) || IsReservedFunc(name) {
		return fmt.Errorf("%s", i18n.T("errors.reserved_word", name))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.funcs == nil {
		e.funcs = make(map[string]*UserFunction)
	}
	e.funcs[name] = &UserFunction{
		Name:   name,
		Params: params,
		Body:   body,
	}
	// Invalidate memoCache on function definition
	e.memoCache = make(map[string]Node)
	return nil
}

// GetUserFunc retrieves a registered user-defined function.
func (e *Env) GetUserFunc(name string) (*UserFunction, bool) {
	if e == nil {
		return nil, false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.funcs == nil {
		return nil, false
	}
	fn, ok := e.funcs[name]
	return fn, ok
}

// AllUserFuncs returns a copy of all user-defined functions.
func (e *Env) AllUserFuncs() map[string]*UserFunction {
	if e == nil {
		return make(map[string]*UserFunction)
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make(map[string]*UserFunction, len(e.funcs))
	for k, v := range e.funcs {
		res[k] = v
	}
	return res
}

// CallDepth returns the current call depth.
func (e *Env) CallDepth() int {
	if e == nil {
		return 0
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.callDepth
}

// EnterCall increments call depth and validates against MaxCallDepth.
func (e *Env) EnterCall(fnName string) error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.callDepth >= MaxCallDepth {
		return fmt.Errorf("%s", i18n.T("errors.recursion_limit_exceeded", MaxCallDepth, fnName))
	}
	e.callDepth++
	return nil
}

// LeaveCall decrements call depth.
func (e *Env) LeaveCall() {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.callDepth > 0 {
		e.callDepth--
	}
}

// CheckMemo checks if a function call with given argument signature has been cached.
func (e *Env) CheckMemo(key string) (Node, bool) {
	if e == nil {
		return nil, false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.memoCache == nil {
		return nil, false
	}
	val, ok := e.memoCache[key]
	return val, ok
}

// SaveMemo stores a function evaluation result in the memo cache.
func (e *Env) SaveMemo(key string, val Node) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.memoCache == nil {
		e.memoCache = make(map[string]Node)
	}
	// Limit memoCache size to prevent memory leaks in long-running sessions
	if len(e.memoCache) > 10000 {
		e.memoCache = make(map[string]Node)
	}
	e.memoCache[key] = val
}

// EnterImport tracks imported files to detect circular dependencies.
func (e *Env) EnterImport(path string) error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.importStack == nil {
		e.importStack = make(map[string]bool)
	}
	if e.importStack[path] {
		return fmt.Errorf("%s", i18n.T("errors.circular_import_detected", path))
	}
	e.importStack[path] = true
	return nil
}

// LeaveImport releases an imported file from the active stack.
func (e *Env) LeaveImport(path string) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.importStack, path)
}
