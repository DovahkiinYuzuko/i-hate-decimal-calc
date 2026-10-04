package calc

import (
	"fmt"
)

// StreamState represents the lifecycle state of a streaming radix conversion.
type StreamState int

const (
	// StateInit is the initial state when streaming starts.
	StateInit StreamState = iota
	// StateDirect is for numbers below the leaf threshold, directly converted.
	StateDirect
	// StatePowerTableGen generates/retrieves 10^K split divisors.
	StatePowerTableGen
	// StateSplitting performs X = Q * 10^K + R.
	StateSplitting
	// StateStreamingUpper recursively streams quotient Q (depth-first).
	StateStreamingUpper
	// StateFreeUpper releases reference to Q for GC reclamation.
	StateFreeUpper
	// StateStreamingLower recursively streams remainder R with zero padding.
	StateStreamingLower
	// StateFlushing flushes any buffered output to the underlying io.Writer.
	StateFlushing
	// StateDone indicates successful completion of streaming.
	StateDone
	// StateError indicates an I/O or internal state error occurred.
	StateError
)

// String returns a human-readable representation of StreamState.
func (s StreamState) String() string {
	switch s {
	case StateInit:
		return "StateInit"
	case StateDirect:
		return "StateDirect"
	case StatePowerTableGen:
		return "StatePowerTableGen"
	case StateSplitting:
		return "StateSplitting"
	case StateStreamingUpper:
		return "StateStreamingUpper"
	case StateFreeUpper:
		return "StateFreeUpper"
	case StateStreamingLower:
		return "StateStreamingLower"
	case StateFlushing:
		return "StateFlushing"
	case StateDone:
		return "StateDone"
	case StateError:
		return "StateError"
	default:
		return fmt.Sprintf("StreamState(%d)", s)
	}
}

// StreamFSM manages state transitions and error safety for streaming radix conversion.
type StreamFSM struct {
	CurrentState StreamState
	LastError    error
}

// NewStreamFSM initializes a new StreamFSM in StateInit.
func NewStreamFSM() *StreamFSM {
	return &StreamFSM{
		CurrentState: StateInit,
		LastError:    nil,
	}
}

// validTransitions defines permissible transitions between states.
var validTransitions = map[StreamState][]StreamState{
	StateInit: {
		StateDirect,
		StatePowerTableGen,
		StateError,
	},
	StateDirect: {
		StateFlushing,
		StateDone,
		StateError,
	},
	StatePowerTableGen: {
		StateSplitting,
		StateError,
	},
	StateSplitting: {
		StateStreamingUpper,
		StateError,
	},
	StateStreamingUpper: {
		StateFreeUpper,
		StateError,
	},
	StateFreeUpper: {
		StateStreamingLower,
		StateError,
	},
	StateStreamingLower: {
		StateFlushing,
		StateDone,
		StateError,
	},
	StateFlushing: {
		StateDone,
		StateError,
	},
	StateDone: {
		StateInit, // Reusable
	},
	StateError: {
		StateInit, // Reset after error
	},
}

// Transition advances the FSM to the target next state if permitted.
func (f *StreamFSM) Transition(next StreamState) error {
	if f.CurrentState == StateError && next != StateInit {
		return fmt.Errorf("cannot transition from StateError to %s: %w", next, f.LastError)
	}

	allowed := validTransitions[f.CurrentState]
	for _, s := range allowed {
		if s == next {
			f.CurrentState = next
			return nil
		}
	}

	err := fmt.Errorf("illegal state transition from %s to %s", f.CurrentState, next)
	f.Fail(err)
	return err
}

// Fail forces transition to StateError and stores the cause.
func (f *StreamFSM) Fail(err error) {
	f.CurrentState = StateError
	f.LastError = err
}

// Reset restores the FSM to StateInit for reuse.
func (f *StreamFSM) Reset() {
	f.CurrentState = StateInit
	f.LastError = nil
}
