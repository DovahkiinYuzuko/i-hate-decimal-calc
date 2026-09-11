package calc

import "sync"

// Env represents a symbol table mapping variable names to AST nodes.
type Env struct {
	mu   sync.RWMutex
	vars map[string]Node
}

// NewEnv creates a new, empty Env instance.
func NewEnv() *Env {
	return &Env{
		vars: make(map[string]Node),
	}
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
	return copied
}
