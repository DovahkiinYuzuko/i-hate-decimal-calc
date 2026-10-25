package calc

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSearchTrace_EventsAndJSONDump(t *testing.T) {
	collector := NewMemorySearchTraceCollector()

	collector.RecordSearchEvent(SearchEvent{
		Kind:  EventGroebnerPairSelected,
		Phase: "Buchberger",
		Payload: map[string]any{
			"pair": "p1, p2",
			"lcm":  "x^2*y",
		},
	})

	collector.RecordSearchEvent(SearchEvent{
		Kind:  EventCadProjected,
		Phase: "CAD",
		Payload: map[string]any{
			"dimension": 2,
			"polys":     1,
		},
	})

	events := collector.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	var buf bytes.Buffer
	if err := collector.DumpJSON(&buf); err != nil {
		t.Fatalf("DumpJSON failed: %v", err)
	}

	var parsed []SearchEvent
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("expected 2 parsed events, got %d", len(parsed))
	}
	if parsed[0].Kind != EventGroebnerPairSelected {
		t.Errorf("expected EventGroebnerPairSelected, got %v", parsed[0].Kind)
	}
}
