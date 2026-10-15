package units

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// UnitState represents the lifecycle state during unit parsing, resolution, and normalization.
type UnitState string

const (
	UnitStateInit              UnitState = "INIT"
	UnitStateTokenizing        UnitState = "TOKENIZING"
	UnitStatePrefixResolved    UnitState = "PREFIX_RESOLVED"
	UnitStateBaseIdentified    UnitState = "BASE_IDENTIFIED"
	UnitStateVectorSynthesized UnitState = "VECTOR_SYNTHESIZED"
	UnitStateNormalized        UnitState = "NORMALIZED"
	UnitStateSuccess           UnitState = "SUCCESS"
	UnitStateFailure           UnitState = "FAILURE"
)

// QuantityFSM governs the deterministic state transitions for physical unit analysis.
type QuantityFSM struct {
	mu         sync.RWMutex
	current    UnitState
	history    []UnitState
	validTrans map[UnitState]map[UnitState]bool
}

// NewQuantityFSM creates an initialized FSM in UnitStateInit with bounded valid transitions.
func NewQuantityFSM() *QuantityFSM {
	fsm := &QuantityFSM{
		current:    UnitStateInit,
		history:    []UnitState{UnitStateInit},
		validTrans: make(map[UnitState]map[UnitState]bool),
	}

	allow := func(from UnitState, toList ...UnitState) {
		if fsm.validTrans[from] == nil {
			fsm.validTrans[from] = make(map[UnitState]bool)
		}
		for _, to := range toList {
			fsm.validTrans[from][to] = true
		}
	}

	allow(UnitStateInit, UnitStateTokenizing, UnitStateFailure)
	allow(UnitStateTokenizing, UnitStatePrefixResolved, UnitStateBaseIdentified, UnitStateFailure)
	allow(UnitStatePrefixResolved, UnitStateBaseIdentified, UnitStateFailure)
	allow(UnitStateBaseIdentified, UnitStateVectorSynthesized, UnitStateFailure)
	allow(UnitStateVectorSynthesized, UnitStateNormalized, UnitStateFailure)
	allow(UnitStateNormalized, UnitStateSuccess, UnitStateFailure)
	allow(UnitStateFailure, UnitStateInit) // allow reset

	return fsm
}

// CurrentState returns the current state of the FSM thread-safely.
func (f *QuantityFSM) CurrentState() UnitState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current
}

// History returns a copy of the transition history.
func (f *QuantityFSM) History() []UnitState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	copied := make([]UnitState, len(f.history))
	copy(copied, f.history)
	return copied
}

// Transition moves the FSM to the target state if valid, or records Failure and returns an error.
func (f *QuantityFSM) Transition(target UnitState) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	allowed := f.validTrans[f.current][target]
	if !allowed {
		from := f.current
		f.current = UnitStateFailure
		f.history = append(f.history, UnitStateFailure)
		return fmt.Errorf("invalid state transition from %s to %s", from, target)
	}

	f.current = target
	f.history = append(f.history, target)
	return nil
}

// Reset resets the FSM to UnitStateInit.
func (f *QuantityFSM) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = UnitStateInit
	f.history = []UnitState{UnitStateInit}
}

// ResolveUnitExprWithFSM executes the complete unit resolution lifecycle through deterministic FSM state transitions.
func (f *QuantityFSM) ResolveUnitExprWithFSM(node ast.Node) (*big.Rat, ast.DimensionVector, error) {
	f.Reset()

	if err := f.Transition(UnitStateTokenizing); err != nil {
		return nil, ast.NewDimensionVector(), err
	}

	scale, dims, err := EvalUnitExpr(node)
	if err != nil {
		_ = f.Transition(UnitStateFailure)
		return nil, ast.NewDimensionVector(), err
	}

	if err := f.Transition(UnitStatePrefixResolved); err != nil {
		return nil, ast.NewDimensionVector(), err
	}

	if err := f.Transition(UnitStateBaseIdentified); err != nil {
		return nil, ast.NewDimensionVector(), err
	}

	if err := f.Transition(UnitStateVectorSynthesized); err != nil {
		return nil, ast.NewDimensionVector(), err
	}

	// Invariant validation: scale != nil and all 7 dimension components are valid *big.Rat
	for i := 0; i < 7; i++ {
		if dims[i] == nil {
			_ = f.Transition(UnitStateFailure)
			return nil, ast.NewDimensionVector(), fmt.Errorf("FSM invariant violated: dimension component %d is nil", i)
		}
	}
	if scale == nil || scale.Denom().Sign() == 0 {
		_ = f.Transition(UnitStateFailure)
		return nil, ast.NewDimensionVector(), fmt.Errorf("FSM invariant violated: invalid rational scale factor")
	}

	if err := f.Transition(UnitStateNormalized); err != nil {
		return nil, ast.NewDimensionVector(), err
	}

	if err := f.Transition(UnitStateSuccess); err != nil {
		return nil, ast.NewDimensionVector(), err
	}

	return scale, dims, nil
}

// ResolveUnit parses and resolves a unit string into an exact scale and dimension vector using FSM transitions.
func (f *QuantityFSM) ResolveUnit(raw string) (*big.Rat, ast.DimensionVector, error) {
	scale, dims, err := ResolveUnitSymbol(raw)
	if err == nil {
		return scale, dims, nil
	}
	return f.ResolveUnitExprWithFSM(ast.NewVar(raw))
}

// ResolveUnit parses and resolves a unit string into an exact scale and dimension vector using a new FSM.
func ResolveUnit(raw string) (*big.Rat, ast.DimensionVector, error) {
	fsm := NewQuantityFSM()
	return fsm.ResolveUnit(raw)
}



