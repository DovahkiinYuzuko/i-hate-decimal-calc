package calc

import (
	"strings"
	"testing"
)

func TestTraceRecordAndCompress(t *testing.T) {
	sink := NewMemoryTraceSink()

	// Record events
	n1 := mustRational(1, 1)
	n2 := mustRational(2, 1)
	sink.RecordRewrite(RuleRationalize, n1, n2, "Multiply conjugate to denominator")
	sink.RecordRewrite(RuleDenestRadical, n2, n1, "Denest nested radical")

	events := sink.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Rule != RuleRationalize {
		t.Errorf("expected RuleRationalize, got %s", events[0].Rule)
	}

	// Compress
	steps := CompressTrace(events)
	if len(steps) != 2 {
		t.Fatalf("expected 2 pedagogical steps, got %d", len(steps))
	}
	if steps[0].StepNumber != 1 || steps[0].Rule != RuleRationalize {
		t.Errorf("step 0 mismatch: %+v", steps[0])
	}
	if steps[1].StepNumber != 2 || steps[1].Rule != RuleDenestRadical {
		t.Errorf("step 1 mismatch: %+v", steps[1])
	}

	// Format
	formatted := FormatTrace("1 / (sqrt(2) + 1)", steps, n1)
	if !strings.Contains(formatted, "Step 1") {
		t.Errorf("expected 'Step 1' in formatted trace, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "Result") {
		t.Errorf("expected 'Result' in formatted trace, got:\n%s", formatted)
	}
}

func TestWithTraceSink(t *testing.T) {
	sink := NewMemoryTraceSink()
	WithTraceSink(sink, func() {
		RecordTraceRewrite(RuleRationalize, mustRational(1, 2), mustRational(2, 4), "Test rewrite")
	})

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Note != "Test rewrite" {
		t.Errorf("note mismatch: got %q", events[0].Note)
	}
}

func TestEvalWithTrace(t *testing.T) {
	// 1. Rationalize trace
	ast1, err := Parse("1 / (sqrt(2) + 1)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res1, steps1, err := EvalWithTrace(ast1)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if len(steps1) == 0 {
		t.Errorf("expected rationalize steps, got none")
	}
	t.Logf("Rationalize trace:\n%s", FormatTrace("1 / (sqrt(2) + 1)", steps1, res1))

	// 2. Denest trace
	ast2, err := Parse("sqrt(5 + 2*sqrt(6))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res2, steps2, err := EvalWithTrace(ast2)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if len(steps2) == 0 {
		t.Errorf("expected denest steps, got none")
	}
	t.Logf("Denest trace:\n%s", FormatTrace("sqrt(5 + 2*sqrt(6))", steps2, res2))

	// 3. Solve quadratic trace
	ast3, err := Parse("solve(x^2 - 5*x + 6, x)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res3, steps3, err := EvalWithTrace(ast3)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if len(steps3) == 0 {
		t.Errorf("expected solve quadratic steps, got none")
	}
	t.Logf("Solve trace:\n%s", FormatTrace("solve(x^2 - 5*x + 6, x)", steps3, res3))

	// 4. Diff power trace
	ast4, err := Parse("diff(x^3, x)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res4, steps4, err := EvalWithTrace(ast4)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if len(steps4) == 0 {
		t.Errorf("expected diff steps, got none")
	}
	t.Logf("Diff trace:\n%s", FormatTrace("diff(x^3, x)", steps4, res4))
}
