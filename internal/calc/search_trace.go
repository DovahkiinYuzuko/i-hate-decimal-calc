package calc

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// SearchEventKind defines the fine-grained category of search/algorithm events.
type SearchEventKind string

const (
	EventGroebnerPairSelected SearchEventKind = "groebner.pair_selected"
	EventGroebnerSpolyReduced SearchEventKind = "groebner.spoly_reduced"
	EventCadProjected         SearchEventKind = "cad.projected"
	EventCadSampledLifted     SearchEventKind = "cad.sampled_lifted"
	EventRischTowerStep       SearchEventKind = "risch.tower_step"
)

// SearchEvent represents an individual exploration or optimization step in an algorithm.
type SearchEvent struct {
	Timestamp int64           `json:"timestamp_ns"`
	Kind      SearchEventKind `json:"kind"`
	Phase     string          `json:"phase"`
	Payload   map[string]any  `json:"payload,omitempty"`
}

// SearchTraceCollector defines the interface for recording and dumping internal search events.
type SearchTraceCollector interface {
	RecordSearchEvent(event SearchEvent)
	Events() []SearchEvent
	DumpJSON(w io.Writer) error
}

// MemorySearchTraceCollector is a thread-safe in-memory implementation of SearchTraceCollector.
type MemorySearchTraceCollector struct {
	mu     sync.RWMutex
	events []SearchEvent
}

// NewMemorySearchTraceCollector creates a new empty MemorySearchTraceCollector.
func NewMemorySearchTraceCollector() *MemorySearchTraceCollector {
	return &MemorySearchTraceCollector{
		events: make([]SearchEvent, 0),
	}
}

// RecordSearchEvent appends a new search event.
func (c *MemorySearchTraceCollector) RecordSearchEvent(event SearchEvent) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixNano()
	}
	c.events = append(c.events, event)
}

// Events returns a copy of recorded events.
func (c *MemorySearchTraceCollector) Events() []SearchEvent {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]SearchEvent, len(c.events))
	copy(out, c.events)
	return out
}

// DumpJSON serializes recorded search events to an io.Writer as pretty JSON.
func (c *MemorySearchTraceCollector) DumpJSON(w io.Writer) error {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(c.events)
}

// Global active search trace collector for CLI --trace execution.
var (
	activeSearchCollectorMu sync.RWMutex
	activeSearchCollector   SearchTraceCollector
)

// SetActiveSearchCollector registers the global collector for internal algorithm hooks.
func SetActiveSearchCollector(c SearchTraceCollector) {
	activeSearchCollectorMu.Lock()
	defer activeSearchCollectorMu.Unlock()
	activeSearchCollector = c
}

// GetActiveSearchCollector returns the currently registered global collector.
func GetActiveSearchCollector() SearchTraceCollector {
	activeSearchCollectorMu.RLock()
	defer activeSearchCollectorMu.RUnlock()
	return activeSearchCollector
}

// EmitSearchEvent records an event to the active collector if enabled (zero overhead when disabled).
func EmitSearchEvent(kind SearchEventKind, phase string, payload map[string]any) {
	c := GetActiveSearchCollector()
	if c != nil {
		c.RecordSearchEvent(SearchEvent{
			Kind:    kind,
			Phase:   phase,
			Payload: payload,
		})
	}
}
