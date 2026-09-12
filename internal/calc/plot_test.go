package calc

import (
	"strings"
	"testing"
)

func TestPlot_Parabola(t *testing.T) {
	expr := "plot(x^2 - 2, [-2, 2])"
	res, err := EvalString(expr)
	if err != nil {
		t.Fatalf("unexpected error for %s: %v", expr, err)
	}

	plotNode, ok := res.(*PlotNode)
	if !ok {
		t.Fatalf("expected *PlotNode, got %T", res)
	}

	content := plotNode.Content

	// Verify CAS features are detected in the metadata section
	if !strings.Contains(content, "[CAS Features Detected]") {
		t.Errorf("expected [CAS Features Detected] in output, got:\n%s", content)
	}
	if !strings.Contains(content, "x = -√2") || !strings.Contains(content, "x = √2") {
		t.Errorf("expected roots x = ±√2 in output, got:\n%s", content)
	}
	if !strings.Contains(content, "(0, -2)") {
		t.Errorf("expected local minimum (0, -2) in output, got:\n%s", content)
	}

	// Verify domain information
	if !strings.Contains(content, "x ∈ [-2, 2]") {
		t.Errorf("expected domain x ∈ [-2, 2] in output, got:\n%s", content)
	}
}

func TestPlot_RationalAsymptote(t *testing.T) {
	expr := "plot(1/x, [-3, 3])"
	res, err := EvalString(expr)
	if err != nil {
		t.Fatalf("unexpected error for %s: %v", expr, err)
	}

	plotNode, ok := res.(*PlotNode)
	if !ok {
		t.Fatalf("expected *PlotNode, got %T", res)
	}

	content := plotNode.Content

	// Verify vertical asymptote detection
	if !strings.Contains(content, "x = 0") || (!strings.Contains(content, "Asymptote") && !strings.Contains(content, "漸近線")) {
		t.Errorf("expected vertical asymptote x = 0 in output, got:\n%s", content)
	}

	// Verify asymptote dashed line '┆' is drawn
	if !strings.Contains(content, "┆") {
		t.Errorf("expected dashed asymptote character '┆' in output, got:\n%s", content)
	}
}

func TestPlot_ExplicitYDomain(t *testing.T) {
	expr := "plot(x^2, [-2, 2], [0, 4])"
	res, err := EvalString(expr)
	if err != nil {
		t.Fatalf("unexpected error for %s: %v", expr, err)
	}

	plotNode, ok := res.(*PlotNode)
	if !ok {
		t.Fatalf("expected *PlotNode, got %T", res)
	}

	content := plotNode.Content
	if !strings.Contains(content, "y ∈ [0, 4]") {
		t.Errorf("expected y ∈ [0, 4] in output, got:\n%s", content)
	}
}

func TestPlot_Errors(t *testing.T) {
	errTests := []string{
		"plot(x, [2, -2])", // lower bound >= upper bound
		"plot(x, [1])",     // wrong domain length
		"plot(x)",          // missing domain
		"plot(x, 1)",       // non-list domain
	}

	for _, expr := range errTests {
		t.Run(expr, func(t *testing.T) {
			_, err := EvalString(expr)
			if err == nil {
				t.Errorf("expected error for %s, got nil", expr)
			}
		})
	}
}
