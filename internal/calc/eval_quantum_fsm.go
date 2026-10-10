package calc

import (
	"fmt"
)

// QuantumFSMState represents the state of the quantum circuit evaluation pipeline.
type QuantumFSMState int

const (
	QuantumStateInit QuantumFSMState = iota
	QuantumStateValidating
	QuantumStateSimulating
	QuantumStateComparing
	QuantumStateComplete
	QuantumStateError
)

func (s QuantumFSMState) String() string {
	switch s {
	case QuantumStateInit:
		return "Init"
	case QuantumStateValidating:
		return "Validating"
	case QuantumStateSimulating:
		return "Simulating"
	case QuantumStateComparing:
		return "Comparing"
	case QuantumStateComplete:
		return "Complete"
	case QuantumStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// QuantumCircuitFSM governs the state transitions for circuit execution.
type QuantumCircuitFSM struct {
	State     QuantumFSMState
	NumQubits int
	LastError error
}

// NewQuantumCircuitFSM initializes a new pipeline FSM.
func NewQuantumCircuitFSM(numQubits int) *QuantumCircuitFSM {
	return &QuantumCircuitFSM{
		State:     QuantumStateInit,
		NumQubits: numQubits,
	}
}

// Transition performs a validated transition to the next state.
func (fsm *QuantumCircuitFSM) Transition(next QuantumFSMState) error {
	valid := false

	switch fsm.State {
	case QuantumStateInit:
		valid = (next == QuantumStateValidating || next == QuantumStateError)
	case QuantumStateValidating:
		valid = (next == QuantumStateSimulating || next == QuantumStateComparing || next == QuantumStateError)
	case QuantumStateSimulating:
		valid = (next == QuantumStateComparing || next == QuantumStateComplete || next == QuantumStateError)
	case QuantumStateComparing:
		valid = (next == QuantumStateComplete || next == QuantumStateError)
	case QuantumStateComplete:
		valid = (next == QuantumStateInit)
	case QuantumStateError:
		valid = (next == QuantumStateInit)
	}

	if !valid {
		err := fmt.Errorf("invalid quantum FSM transition from %s to %s", fsm.State, next)
		fsm.LastError = err
		fsm.State = QuantumStateError
		return err
	}

	fsm.State = next
	return nil
}
