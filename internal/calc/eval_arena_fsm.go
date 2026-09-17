package calc

import (
	"fmt"
	"sync"
)

// ArenaState represents the lifecycle states of a NodeArena.
type ArenaState int

const (
	// ArenaStateInactive indicates the arena is uninitialized or sitting idle in a pool.
	ArenaStateInactive ArenaState = iota
	// ArenaStateActive indicates the arena is actively allocating nodes.
	ArenaStateActive
	// ArenaStateDetached indicates the final result node has been cloned out (detached).
	ArenaStateDetached
	// ArenaStateReleased indicates the arena memory has been cleared and returned to the pool.
	ArenaStateReleased
)

// String returns a human-readable name of the arena state.
func (s ArenaState) String() string {
	switch s {
	case ArenaStateInactive:
		return "Inactive"
	case ArenaStateActive:
		return "Active"
	case ArenaStateDetached:
		return "Detached"
	case ArenaStateReleased:
		return "Released"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// ArenaLifecycleFSM governs valid state transitions for memory arenas to prevent
// memory corruption, use-after-free, or multiple releases.
type ArenaLifecycleFSM struct {
	mu    sync.RWMutex
	state ArenaState
}

// NewArenaLifecycleFSM creates an FSM initialized to ArenaStateActive.
func NewArenaLifecycleFSM() *ArenaLifecycleFSM {
	return &ArenaLifecycleFSM{
		state: ArenaStateActive,
	}
}

// State returns the current arena state.
func (fsm *ArenaLifecycleFSM) State() ArenaState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.state
}

// Activate transitions the state to ArenaStateActive (e.g. when pulled from a pool).
func (fsm *ArenaLifecycleFSM) Activate() error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	if fsm.state == ArenaStateActive {
		return nil
	}
	fsm.state = ArenaStateActive
	return nil
}

// CheckCanAllocate verifies that the arena is in an active state capable of allocating memory.
func (fsm *ArenaLifecycleFSM) CheckCanAllocate() error {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	switch fsm.state {
	case ArenaStateActive, ArenaStateDetached:
		return nil
	case ArenaStateReleased:
		return fmt.Errorf("arena allocation violation: cannot allocate from released arena (state: %s)", fsm.state)
	case ArenaStateInactive:
		return fmt.Errorf("arena allocation violation: cannot allocate from inactive arena (state: %s)", fsm.state)
	default:
		return fmt.Errorf("arena allocation violation: invalid arena state %s", fsm.state)
	}
}

// MarkDetached records that an expression result has been detached from this arena.
func (fsm *ArenaLifecycleFSM) MarkDetached() error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	if fsm.state != ArenaStateActive && fsm.state != ArenaStateDetached {
		return fmt.Errorf("cannot mark detached from state: %s", fsm.state)
	}
	fsm.state = ArenaStateDetached
	return nil
}

// Release transitions the arena to ArenaStateReleased.
func (fsm *ArenaLifecycleFSM) Release() error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	if fsm.state == ArenaStateReleased {
		return fmt.Errorf("arena double release violation: arena is already released")
	}
	fsm.state = ArenaStateReleased
	return nil
}
