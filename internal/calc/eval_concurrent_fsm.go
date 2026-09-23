package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"sync"
)

// TaskState represents the execution lifecycle state of a concurrent CAS task.
type TaskState int

const (
	StateIdle TaskState = iota
	StateRunning
	StateCompleted
	StateFailed
	StateCancelled
)

func (s TaskState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateRunning:
		return "Running"
	case StateCompleted:
		return "Completed"
	case StateFailed:
		return "Failed"
	case StateCancelled:
		return "Cancelled"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// TaskFSM manages the state transitions of a concurrent task with thread safety.
type TaskFSM struct {
	state TaskState
	mu    sync.RWMutex
}

// NewTaskFSM creates a new TaskFSM in StateIdle.
func NewTaskFSM() *TaskFSM {
	return &TaskFSM{
		state: StateIdle,
	}
}

// CurrentState returns the current state in a thread-safe manner.
func (f *TaskFSM) CurrentState() TaskState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state
}

// IsDone returns true if the task has reached a terminal state (Completed, Failed, or Cancelled).
func (f *TaskFSM) IsDone() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state == StateCompleted || f.state == StateFailed || f.state == StateCancelled
}

// Transition attempts to move from the current state to the target state.
// Returns an error if the transition is invalid according to FSM governance.
func (f *TaskFSM) Transition(to TaskState) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch f.state {
	case StateIdle:
		if to == StateRunning || to == StateCancelled {
			f.state = to
			return nil
		}
	case StateRunning:
		if to == StateCompleted || to == StateFailed || to == StateCancelled {
			f.state = to
			return nil
		}
	case StateCompleted, StateFailed, StateCancelled:
		return fmt.Errorf("%s", i18n.T("concurrent.err_invalid_transition_from_terminal_state", f.state, to))
	}

	return fmt.Errorf("%s", i18n.T("concurrent.err_invalid_transition_from", f.state, to))
}
