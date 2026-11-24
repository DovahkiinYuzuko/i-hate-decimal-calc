package calc_test

import (
	"encoding/xml"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

func TestPlot3D_Deductive_SurfaceAndParametricCurve(t *testing.T) {
	cases := []struct {
		name           string
		expr           string
		wantSubstrings []string
	}{
		{
			name: "saddle surface z = x^2 - y^2",
			expr: "plot3d(x^2 - y^2, [-2, 2], [-2, 2])",
			wantSubstrings: []string{
				"x ∈ [-2, 2]",
				"y ∈ [-2, 2]",
			},
		},
		{
			name: "explicit variable ranges u, v",
			expr: "plot3d(u^2 + v^2, [u, -2, 2], [v, -2, 2])",
			wantSubstrings: []string{
				"u ∈ [-2, 2]",
				"v ∈ [-2, 2]",
			},
		},
		{
			name: "parametric 3d helix",
			expr: "plot3d([cos(t), sin(t), t], [t, 0, 10])",
			wantSubstrings: []string{
				"t ∈ [0, 10]",
			},
		},
		{
			name: "perspective option",
			expr: `plot3d(x^2 + y^2, [-2, 2], [-2, 2], [az, 60], [el, 25], "perspective")`,
			wantSubstrings: []string{
				"x ∈ [-2, 2]",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := calc.EvalString(tc.expr)
			if err != nil {
				t.Fatalf("EvalString(%q) failed: %v", tc.expr, err)
			}
			pn, ok := res.(*calc.PlotNode)
			if !ok {
				t.Fatalf("expected *calc.PlotNode, got %T", res)
			}

			// Monochrome Content must contain Braille runes and NO ANSI escape sequences
			if !containsBrailleRune(pn.Content) {
				t.Errorf("expected Braille characters in monochrome Content:\n%s", pn.Content)
			}
			if strings.Contains(pn.Content, "\x1b[") {
				t.Errorf("monochrome Content must not contain ANSI escapes:\n%s", pn.Content)
			}

			// ColorContent must contain 256-color foreground ANSI escapes (38;5;) and no background escapes (48;)
			if !strings.Contains(pn.ColorContent, "\x1b[38;5;") {
				t.Errorf("expected ANSI 256-color foreground in ColorContent:\n%s", pn.ColorContent)
			}
			if strings.Contains(pn.ColorContent, "\x1b[48;") {
				t.Errorf("ColorContent must be fg-only and not touch background:\n%s", pn.ColorContent)
			}

			for _, sub := range tc.wantSubstrings {
				if !strings.Contains(pn.Content, sub) {
					t.Errorf("expected %q in Content:\n%s", sub, pn.Content)
				}
			}

			// SVG checks
			if !strings.Contains(pn.SVG, "<svg") || !strings.Contains(pn.SVG, "<polyline") {
				t.Errorf("expected valid SVG with polylines, got:\n%s", pn.SVG)
			}
			for _, bad := range []string{"NaN", "Inf", "-Inf", "+Inf"} {
				if strings.Contains(pn.SVG, bad) || strings.Contains(pn.Typst, bad) {
					t.Errorf("vector output contains invalid token %q", bad)
				}
			}
			var v any
			if err := xml.Unmarshal([]byte(pn.SVG), &v); err != nil {
				t.Errorf("SVG is not valid XML: %v\n%s", err, pn.SVG)
			}

			// Typst checks (projected 2D cetz canvas, no CeTZ 3D dependency)
			if !strings.Contains(pn.Typst, `#import "@preview/cetz:0.5.2"`) || !strings.Contains(pn.Typst, "cetz.canvas") {
				t.Errorf("expected cetz 2D canvas in Typst output, got:\n%s", pn.Typst)
			}
			if !strings.Contains(pn.Typst, "line(") {
				t.Errorf("expected line() polylines in Typst output, got:\n%s", pn.Typst)
			}
		})
	}
}

func TestPlot3D_Contradiction_HiddenLineRemovalAndSingularity(t *testing.T) {
	// 1. Occlusion check: a mountain z = 4 - x^2 - y^2 at az=45, el=15 must hide back grid lines.
	exprNode, err := calc.Parse("4 - x^2 - y^2")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rX, _ := calc.Parse("[-2, 2]")
	rY, _ := calc.Parse("[-2, 2]")
	azOpt, _ := calc.Parse("[az, 45]")
	elOpt, _ := calc.Parse("[el, 15]")

	scene, err := calc.BuildPlot3DScene([]calc.Node{exprNode, rX, rY, azOpt, elOpt})
	if err != nil {
		t.Fatalf("BuildPlot3DScene failed: %v", err)
	}

	var surfaceStrokes int
	var totalVisibleLen float64
	for _, s := range scene.Strokes {
		if s.Kind == "surface" && len(s.Points) >= 2 {
			surfaceStrokes++
			for i := 0; i < len(s.Points)-1; i++ {
				totalVisibleLen += math.Hypot(s.Points[i+1].X-s.Points[i].X, s.Points[i+1].Y-s.Points[i].Y)
			}
		}
	}
	if surfaceStrokes == 0 || totalVisibleLen <= 0 {
		t.Fatalf("expected non-empty surface strokes, got count=%d len=%v", surfaceStrokes, totalVisibleLen)
	}

	// Verify occlusion actually dropped/clipped hidden segments compared to raw unoccluded isoline count
	// (on a mountain at low elevation=15 deg, significant back-side isoline length is hidden).
	if scene.UnoccludedSurfaceLen <= 0 || totalVisibleLen >= scene.UnoccludedSurfaceLen*0.98 {
		t.Errorf("expected hidden-line removal to reduce visible surface stroke length: visible=%v, unoccluded=%v",
			totalVisibleLen, scene.UnoccludedSurfaceLen)
	}

	// 2. Singularity / pole clipping: 1/(x^2 + y^2) must not leak NaN/Inf or spike outside viewport
	resPole, err := calc.EvalString("plot3d(1/(x^2 + y^2), [-2, 2], [-2, 2])")
	if err != nil {
		t.Fatalf("EvalString for pole failed: %v", err)
	}
	pnPole := resPole.(*calc.PlotNode)
	for _, bad := range []string{"NaN", "Inf", "e+"} {
		if strings.Contains(pnPole.SVG, bad) || strings.Contains(pnPole.Typst, bad) {
			t.Errorf("pole output contains %q", bad)
		}
	}
	var v any
	if err := xml.Unmarshal([]byte(pnPole.SVG), &v); err != nil {
		t.Errorf("pole SVG is invalid XML: %v", err)
	}
}

func TestPlot3D_Contrapositive_InvalidInputs(t *testing.T) {
	badInputs := []string{
		"plot3d(x)",
		"plot3d(x^2 + y^2, [3, -3], [-2, 2])",
		"plot3d(x^2 + y^2, [-2, 2])",
		"plot3d([t, t^2], [t, 0, 1])",
		`plot3d_svg(x^2 + y^2, [-2, 2], [-2, 2], "")`,
	}
	for _, expr := range badInputs {
		if _, err := calc.EvalString(expr); err == nil {
			t.Errorf("expected error for invalid input %q, got nil", expr)
		}
	}
}

func TestPlot3D_Metamorphic_DeterminismAndSVGExport(t *testing.T) {
	expr := "plot3d(x^2 - y^2, [-2, 2], [-2, 2])"
	r1, err1 := calc.EvalString(expr)
	r2, err2 := calc.EvalString(expr)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}
	p1 := r1.(*calc.PlotNode)
	p2 := r2.(*calc.PlotNode)
	if p1.Content != p2.Content || p1.ColorContent != p2.ColorContent || p1.SVG != p2.SVG || p1.Typst != p2.Typst {
		t.Errorf("plot3d output is not deterministic across repeated runs")
	}

	// Symmetry under 180-degree azimuth rotation on symmetric paraboloid x^2 + y^2
	nExpr, _ := calc.Parse("x^2 + y^2")
	rX, _ := calc.Parse("[-2, 2]")
	rY, _ := calc.Parse("[-2, 2]")
	az0, _ := calc.Parse("[az, 45]")
	az180, _ := calc.Parse("[az, 225]")
	s1, err := calc.BuildPlot3DScene([]calc.Node{nExpr, rX, rY, az0})
	if err != nil {
		t.Fatalf("BuildPlot3DScene az=45 failed: %v", err)
	}
	s2, err := calc.BuildPlot3DScene([]calc.Node{nExpr, rX, rY, az180})
	if err != nil {
		t.Fatalf("BuildPlot3DScene az=225 failed: %v", err)
	}
	if len(s1.Strokes) == 0 || math.Abs(float64(len(s1.Strokes)-len(s2.Strokes))) > 4 {
		t.Errorf("expected symmetric stroke counts under 180deg rotation: %d vs %d", len(s1.Strokes), len(s2.Strokes))
	}

	// File export via plot3d_svg
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "saddle3d.svg")
	svgExpr := `plot3d_svg(x^2 - y^2, [-2, 2], [-2, 2], "` + filepath.ToSlash(outPath) + `")`
	res, err := calc.EvalString(svgExpr)
	if err != nil {
		t.Fatalf("plot3d_svg failed: %v", err)
	}
	if _, ok := res.(*calc.StringNode); !ok {
		t.Fatalf("expected *calc.StringNode, got %T", res)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read exported SVG: %v", err)
	}
	var v any
	if err := xml.Unmarshal(data, &v); err != nil {
		t.Errorf("exported SVG file is not valid XML: %v", err)
	}
}

func containsBrailleRune(s string) bool {
	for _, r := range s {
		if r >= 0x2800 && r <= 0x28FF {
			return true
		}
	}
	return false
}

func TestPlot3D_EnvEvaluationAndVarProtection(t *testing.T) {
	env := calc.NewEnv()
	for _, stmt := range []string{"x = 100", "t = 50", "a = 2", "az0 = 60"} {
		n, err := calc.Parse(stmt)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", stmt, err)
		}
		if _, err := calc.EvalWithEnv(n, env); err != nil {
			t.Fatalf("EvalWithEnv(%q) failed: %v", stmt, err)
		}
	}

	// Surface: x must remain free plot variable, while a and az0 resolve from env
	surfNode, err := calc.Parse("plot3d(a*x^2 - y^2, [-a, a], [-a, a], [az, az0])")
	if err != nil {
		t.Fatalf("Parse surface failed: %v", err)
	}
	res, err := calc.EvalWithEnv(surfNode, env)
	if err != nil {
		t.Fatalf("EvalWithEnv surface failed: %v", err)
	}
	pn, ok := res.(*calc.PlotNode)
	if !ok {
		t.Fatalf("expected *calc.PlotNode, got %T", res)
	}
	if !strings.Contains(pn.Content, "x ∈ [-2, 2]") || !strings.Contains(pn.Content, "z ∈ [-4, 8]") {
		t.Errorf("expected env-evaluated range [-2, 2] and z ∈ [-4, 8], got:\n%s", pn.Content)
	}
	if !strings.Contains(pn.Content, "az = 60°") {
		t.Errorf("expected env-evaluated az = 60°, got:\n%s", pn.Content)
	}

	// Curve: t must remain free plot variable even when t = 50 in env
	curveNode, err := calc.Parse("plot3d([a*cos(t), a*sin(t), t], [t, 0, a])")
	if err != nil {
		t.Fatalf("Parse curve failed: %v", err)
	}
	resC, err := calc.EvalWithEnv(curveNode, env)
	if err != nil {
		t.Fatalf("EvalWithEnv curve failed: %v", err)
	}
	pnC, ok := resC.(*calc.PlotNode)
	if !ok {
		t.Fatalf("expected *calc.PlotNode, got %T", resC)
	}
	if !strings.Contains(pnC.Content, "t ∈ [0, 2]") {
		t.Errorf("expected env-evaluated t ∈ [0, 2], got:\n%s", pnC.Content)
	}
}

func TestPlot3D_PerspectiveNearClipAndDynamicHLR(t *testing.T) {
	// Near camera (dist=1.85): front corner of unit cube reaches pz = sqrt(3) ≈ 1.732, so w = 1.85 - pz < 0.2.
	// 3D near-plane clipping + dynamic AABB HLR must keep clipped surface strokes and hide back lines.
	exprNode, err := calc.Parse("4 - x^2 - y^2")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rX, _ := calc.Parse("[-2, 2]")
	rY, _ := calc.Parse("[-2, 2]")
	azOpt, _ := calc.Parse("[az, 45]")
	elOpt, _ := calc.Parse("[el, 15]")
	distOpt, _ := calc.Parse("[dist, 1.85]")
	perspOpt, _ := calc.Parse(`"perspective"`)

	scene, err := calc.BuildPlot3DScene([]calc.Node{exprNode, rX, rY, azOpt, elOpt, distOpt, perspOpt})
	if err != nil {
		t.Fatalf("BuildPlot3DScene failed: %v", err)
	}

	var visibleLen float64
	var maxAbsCoord float64
	for _, s := range scene.Strokes {
		if s.Kind != "surface" || len(s.Points) < 2 {
			continue
		}
		for i, pt := range s.Points {
			if math.IsNaN(pt.X) || math.IsInf(pt.X, 0) || math.IsNaN(pt.Y) || math.IsInf(pt.Y, 0) {
				t.Fatalf("non-finite projected point: %+v", pt)
			}
			if ax := math.Abs(pt.X); ax > maxAbsCoord {
				maxAbsCoord = ax
			}
			if ay := math.Abs(pt.Y); ay > maxAbsCoord {
				maxAbsCoord = ay
			}
			if i+1 < len(s.Points) {
				visibleLen += math.Hypot(s.Points[i+1].X-pt.X, s.Points[i+1].Y-pt.Y)
			}
		}
	}
	if visibleLen <= 0 {
		t.Fatalf("expected visible surface strokes after near-plane clipping, got 0")
	}
	if scene.UnoccludedSurfaceLen <= 0 || visibleLen >= scene.UnoccludedSurfaceLen*0.95 {
		t.Errorf("expected dynamic HLR occlusion under near perspective: visible=%v, unoccluded=%v",
			visibleLen, scene.UnoccludedSurfaceLen)
	}
	if maxAbsCoord <= 1.2 {
		t.Logf("note: maxAbsCoord=%v (clipped near-plane test still validates HLR)", maxAbsCoord)
	}

	mono, _ := calc.RenderBraille3D(scene)
	if !containsBrailleRune(mono) {
		t.Errorf("expected Braille runes for clipped perspective render")
	}
}

func TestPlot3D_LiangBarskyAndHugeBoundsSafety(t *testing.T) {
	// 1. Huge coefficient bounds must not overflow int64 to MinInt64
	res, err := calc.EvalString("plot3d(10^20*x^2 + y^2, [-2, 2], [-2, 2])")
	if err != nil {
		t.Fatalf("EvalString huge bounds failed: %v", err)
	}
	pn := res.(*calc.PlotNode)
	if strings.Contains(pn.Content, "-9223372036854775808") || strings.Contains(pn.SVG, "-9223372036854775808") {
		t.Fatalf("makeExactBoundNode overflowed int64:\n%s", pn.Content)
	}

	// 2. RenderBraille3D must clip huge out-of-viewport / non-finite coordinates immediately without hanging
	zeroNode, _ := calc.Parse("0")
	oneNode, _ := calc.Parse("1")
	extremeScene := &calc.Plot3DScene{
		Kind:     "curve",
		VarX:     "t",
		XMinNode: zeroNode,
		XMaxNode: oneNode,
		ZMinNode: zeroNode,
		ZMaxNode: oneNode,
		Camera:   calc.Camera3D{AzimuthDeg: 35, ElevationDeg: 25, Aspect: 1},
		Strokes: []calc.Stroke2D{
			{
				Kind: "curve",
				Points: []calc.ProjectedPoint2D{
					{X: -1e9, Y: -1e9, Depth: 0.1, ZVal: 0},
					{X: 1e9, Y: 1e9, Depth: 0.9, ZVal: 1},
				},
			},
			{
				Kind: "curve",
				Points: []calc.ProjectedPoint2D{
					{X: math.NaN(), Y: 0, Depth: 0.5, ZVal: 0.5},
					{X: math.Inf(1), Y: 0, Depth: 0.5, ZVal: 0.5},
				},
			},
		},
	}

	done := make(chan string, 1)
	go func() {
		mono, _ := calc.RenderBraille3D(extremeScene)
		done <- mono
	}()

	select {
	case mono := <-done:
		if !containsBrailleRune(mono) {
			t.Errorf("expected diagonal clipped line to render Braille dots, got:\n%s", mono)
		}
	case <-time.After(400 * time.Millisecond):
		t.Fatalf("RenderBraille3D hung on extreme coordinates (missing Liang-Barsky clipping)")
	}
}

func TestPlot3D_ValidationHardening(t *testing.T) {
	t.Cleanup(func() {
		_ = os.Remove("perspectiv")
	})

	badExprs := []string{
		"plot3d(x^2 + y^2, [x, -2, 2], [x, -2, 2])",
		"plot3d(x^2, [x, -2, 2], [x, -2, 2])",
		"plot3d(x^2 + y^2, [-2, 2], [-2, 2], [az, 45, 999])",
		`plot3d_svg(x^2 + y^2, [-2, 2], [-2, 2], "perspectiv")`,
	}
	for _, expr := range badExprs {
		if _, err := calc.EvalString(expr); err == nil {
			t.Errorf("expected validation error for %q, got nil", expr)
		}
	}
	if _, err := os.Stat("perspectiv"); err == nil {
		t.Errorf("typo option 'perspectiv' unexpectedly created a file")
	}

	// Space curve X/Y jump cut: [1/t, 1/t, t] over [-1, 1] has a pole jump in X and Y across t = 0
	// while z = t changes smoothly. No curve stroke segment may jump across the entire box.
	nExpr, _ := calc.Parse("[1/t, 1/t, t]")
	rT, _ := calc.Parse("[t, -1, 1]")
	scene, err := calc.BuildPlot3DScene([]calc.Node{nExpr, rT})
	if err != nil {
		t.Fatalf("BuildPlot3DScene for [1/t, 1/t, t] failed: %v", err)
	}
	for _, st := range scene.Strokes {
		if st.Kind != "curve" {
			continue
		}
		for i := 0; i < len(st.Points)-1; i++ {
			segLen := math.Hypot(st.Points[i+1].X-st.Points[i].X, st.Points[i+1].Y-st.Points[i].Y)
			if segLen > 0.8 {
				t.Errorf("expected jump cut on X/Y pole for [1/t, 1/t, t], found long segment len=%v", segLen)
			}
		}
	}
}
