package calc

import (
	"math"
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

func TestPlot_EvalSampleFloatZeroAllocs(t *testing.T) {
	expr, err := Parse("x^2 + sin(x) - 1/3")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	v, ok := evalSampleFloat(expr, "x", 1.5)
	if !ok {
		t.Fatalf("expected ok=true for x=1.5")
	}
	want := 1.5*1.5 + math.Sin(1.5) - 1.0/3.0
	if math.Abs(v-want) > 1e-12 {
		t.Fatalf("unexpected value: got %v, want %v", v, want)
	}

	if _, ok := evalSampleFloat(expr, "x", math.NaN()); ok {
		t.Errorf("expected ok=false for NaN input")
	}
	if _, ok := evalSampleFloat(expr, "x", math.Inf(1)); ok {
		t.Errorf("expected ok=false for +Inf input")
	}

	allocs := testing.AllocsPerRun(100, func() {
		_, _ = evalSampleFloat(expr, "x", 1.5)
	})
	if allocs != 0 {
		t.Errorf("expected 0 heap allocs/op in evalSampleFloat, got %v", allocs)
	}
}

func TestPlot_LogBaseConsistency(t *testing.T) {
	// Deductive: log(x) is base-10, ln(x) is base-e, log(b, x) is base-b.
	cases := []struct {
		expr string
		x    float64
		want float64
	}{
		{"log(x)", 10, 1},
		{"log(x)", 100, 2},
		{"log(x)", 0.1, -1},
		{"ln(x)", math.E, 1},
		{"log(2, x)", 8, 3},
		{"log(1/2, x)", 4, -2},
		{"log(x, 100)", 10, 2},
	}

	for _, tc := range cases {
		node, err := EvalString(tc.expr)
		if err != nil {
			t.Fatalf("EvalString(%q) failed: %v", tc.expr, err)
		}
		got, ok := evalSampleFloat(node, "x", tc.x)
		if !ok {
			t.Fatalf("evalSampleFloat(%q, x=%v) returned ok=false", tc.expr, tc.x)
		}
		if math.Abs(got-tc.want) > 1e-12 {
			t.Errorf("evalSampleFloat(%q, x=%v) = %.15g, want %.15g", tc.expr, tc.x, got, tc.want)
		}
	}

	// Contradiction (residue pinning): ln(10) - log(10) == ln(10) - 1 (~1.302585092994046)
	logNode, _ := EvalString("log(x)")
	lnNode, _ := EvalString("ln(x)")
	log10Val, ok1 := evalSampleFloat(logNode, "x", 10)
	ln10Val, ok2 := evalSampleFloat(lnNode, "x", 10)
	if !ok1 || !ok2 {
		t.Fatalf("expected ok=true for log(10) and ln(10)")
	}
	wantResidue := math.Ln10 - 1.0
	if math.Abs((ln10Val-log10Val)-wantResidue) > 1e-12 {
		t.Errorf("ln(10) - log(10) residue = %.15g, want %.15g", ln10Val-log10Val, wantResidue)
	}

	// Metamorphic: change of base log(x) == ln(x)/ln(10), log(3, x) == ln(x)/ln(3)
	logViaLn, _ := EvalString("ln(x)/ln(10)")
	log3Node, _ := EvalString("log(3, x)")
	log3ViaLn, _ := EvalString("ln(x)/ln(3)")
	for _, x := range []float64{0.25, 1.5, 7.0, 42.0} {
		v1, okA := evalSampleFloat(logNode, "x", x)
		v2, okB := evalSampleFloat(logViaLn, "x", x)
		if !okA || !okB || math.Abs(v1-v2) > 1e-12 {
			t.Errorf("base-10 metamorphic mismatch at x=%v: log=%v(%v) vs ln/ln10=%v(%v)", x, v1, okA, v2, okB)
		}
		w1, okC := evalSampleFloat(log3Node, "x", x)
		w2, okD := evalSampleFloat(log3ViaLn, "x", x)
		if !okC || !okD || math.Abs(w1-w2) > 1e-12 {
			t.Errorf("base-3 metamorphic mismatch at x=%v: log(3,x)=%v(%v) vs ln/ln3=%v(%v)", x, w1, okC, w2, okD)
		}
	}

	// Contradiction / Contrapositive: domain violations must return ok=false
	invalidCases := []struct {
		expr string
		x    float64
	}{
		{"log(x)", 0},
		{"log(x)", -2},
		{"ln(x)", 0},
		{"log(x, 5)", 1},
		{"log(x, 5)", -2},
		{"log(2, x)", -4},
	}
	for _, tc := range invalidCases {
		node, err := Parse(tc.expr)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", tc.expr, err)
		}
		if v, ok := evalSampleFloat(node, "x", tc.x); ok {
			t.Errorf("expected domain rejection for %q at x=%v, got %v", tc.expr, tc.x, v)
		}
	}

	// Public Seam: BuildPlotScene auto y-range for log(x) on [1, 100] should be near [0, 2], NOT [0, 4.6]
	sceneArgs := []Node{logNode, NewList([]Node{mustRational(1, 1), mustRational(100, 1)})}
	scene, err := BuildPlotScene(sceneArgs)
	if err != nil {
		t.Fatalf("BuildPlotScene(log(x), [1, 100]) failed: %v", err)
	}
	if scene.YMax > 2.5 || scene.YMax < 1.8 {
		t.Errorf("BuildPlotScene(log(x), [1, 100]) YMax = %v, expected near 2.2 (log10(100)=2)", scene.YMax)
	}

	// Public Seam: 2-arg log plot via EvalString
	if _, err := EvalString("plot(log(2, x), [1, 8])"); err != nil {
		t.Fatalf("plot(log(2, x), [1, 8]) failed: %v", err)
	}

	// Public Seam: 3D plot with log(x^2 + y^2 + 1)
	if _, err := EvalString("plot3d(log(x^2 + y^2 + 1), [-3, 3], [-3, 3])"); err != nil {
		t.Fatalf("plot3d(log(x^2 + y^2 + 1)) failed: %v", err)
	}

	// Zero-alloc regression for 1-arg and 2-arg log
	log2Node, _ := EvalString("log(2, x)")
	if allocs := testing.AllocsPerRun(100, func() { _, _ = evalSampleFloat(logNode, "x", 10) }); allocs != 0 {
		t.Errorf("expected 0 allocs/op for log(x), got %v", allocs)
	}
	if allocs := testing.AllocsPerRun(100, func() { _, _ = evalSampleFloat(log2Node, "x", 8) }); allocs != 0 {
		t.Errorf("expected 0 allocs/op for log(2, x), got %v", allocs)
	}
}

func TestPlot_NegativeBaseOddRoots(t *testing.T) {
	// Deductive: odd denominator rational exponents p/q yield (-1)^p * |x|^(p/q) for x < 0
	cases := []struct {
		expr string
		x    float64
		want float64
	}{
		{"x^(1/3)", -8, -2},
		{"x^(2/3)", -8, 4},
		{"x^(-1/3)", -8, -0.5},
		{"x^(1/5)", -32, -2},
		{"x^(2/5)", -32, 4},
		{"x^(-1/5)", -32, -0.5},
		{"(-x)^(1/5)", 32, -2},
		{"(-x)^(3/7)", 128, -8},
		{"x^(3/15)", -32, -2}, // reduces to 1/5
	}

	for _, tc := range cases {
		// Verify both Eval'd AST (*RationalNode exp) and unsimplified Parse AST
		evalNode, err := EvalString(tc.expr)
		if err != nil {
			t.Fatalf("EvalString(%q) failed: %v", tc.expr, err)
		}
		gotEval, okEval := evalSampleFloat(evalNode, "x", tc.x)
		if !okEval {
			t.Fatalf("evalSampleFloat(Eval(%q), x=%v) returned ok=false", tc.expr, tc.x)
		}
		if math.Abs(gotEval-tc.want) > 1e-12 {
			t.Errorf("evalSampleFloat(Eval(%q), x=%v) = %.15g, want %.15g", tc.expr, tc.x, gotEval, tc.want)
		}

		parsedNode, err := Parse(tc.expr)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", tc.expr, err)
		}
		gotParsed, okParsed := evalSampleFloat(parsedNode, "x", tc.x)
		if !okParsed {
			t.Fatalf("evalSampleFloat(Parse(%q), x=%v) returned ok=false", tc.expr, tc.x)
		}
		if math.Abs(gotParsed-tc.want) > 1e-12 {
			t.Errorf("evalSampleFloat(Parse(%q), x=%v) = %.15g, want %.15g", tc.expr, tc.x, gotParsed, tc.want)
		}
	}

	// Metamorphic: odd/even symmetry f(-x) = -f(x) for p odd, g(-x) = g(x) for p even
	oddNode, _ := EvalString("x^(1/5)")
	evenNode, _ := EvalString("x^(2/5)")
	for _, x := range []float64{0.5, 3.0, 32.0} {
		posOdd, ok1 := evalSampleFloat(oddNode, "x", x)
		negOdd, ok2 := evalSampleFloat(oddNode, "x", -x)
		if !ok1 || !ok2 || math.Abs(negOdd+posOdd) > 1e-12 {
			t.Errorf("odd symmetry failed at x=%v: f(x)=%v, f(-x)=%v", x, posOdd, negOdd)
		}
		posEven, ok3 := evalSampleFloat(evenNode, "x", x)
		negEven, ok4 := evalSampleFloat(evenNode, "x", -x)
		if !ok3 || !ok4 || math.Abs(negEven-posEven) > 1e-12 {
			t.Errorf("even symmetry failed at x=%v: g(x)=%v, g(-x)=%v", x, posEven, negEven)
		}
	}

	// Contradiction: even denominator or non-rational exponent on negative base, and 0^(negative) must fail
	invalidCases := []struct {
		expr string
		x    float64
	}{
		{"x^(1/2)", -4},
		{"x^(3/4)", -16},
		{"x^pi", -2},
		{"x^(-1/5)", 0},
	}
	for _, tc := range invalidCases {
		node, err := EvalString(tc.expr)
		if err != nil {
			t.Fatalf("EvalString(%q) failed: %v", tc.expr, err)
		}
		if v, ok := evalSampleFloat(node, "x", tc.x); ok {
			t.Errorf("expected domain rejection for %q at x=%v, got %v", tc.expr, tc.x, v)
		}
	}

	// Public Seam: BuildPlotScene for x^(1/5) on [-32, 32] must produce negative-side points
	scene, err := BuildPlotScene([]Node{oddNode, NewList([]Node{mustRational(-32, 1), mustRational(32, 1)})})
	if err != nil {
		t.Fatalf("BuildPlotScene(x^(1/5), [-32, 32]) failed: %v", err)
	}
	hasNegSide := false
	for _, br := range scene.Branches {
		for _, pt := range br.Points {
			if pt.X < -10 && pt.Y < -1 {
				hasNegSide = true
				break
			}
		}
	}
	if !hasNegSide {
		t.Errorf("expected negative-side curve points for x^(1/5) on [-32, 32]")
	}

	// Public Seam: 3D plot with general odd root
	if _, err := EvalString("plot3d((x + y)^(1/5), [-4, 4], [-4, 4])"); err != nil {
		t.Fatalf("plot3d((x + y)^(1/5)) failed: %v", err)
	}

	// Zero-alloc regression for Eval'd rational exponent on negative base
	if allocs := testing.AllocsPerRun(100, func() { _, _ = evalSampleFloat(oddNode, "x", -32) }); allocs != 0 {
		t.Errorf("expected 0 allocs/op for x^(1/5) at x=-32, got %v", allocs)
	}
	if allocs := testing.AllocsPerRun(100, func() { _, _ = evalSampleFloat(evenNode, "x", -32) }); allocs != 0 {
		t.Errorf("expected 0 allocs/op for x^(2/5) at x=-32, got %v", allocs)
	}
}
