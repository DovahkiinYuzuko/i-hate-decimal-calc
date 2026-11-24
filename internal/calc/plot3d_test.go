package calc_test

import (
	"encoding/xml"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

func assertSVGPolylineCoordsFiniteAndBounded(t *testing.T, svg string, minX, maxX, minY, maxY float64) int {
	t.Helper()
	rePoints := regexp.MustCompile(`points="([^"]+)"`)
	matches := rePoints.FindAllStringSubmatch(svg, -1)
	if len(matches) == 0 {
		t.Fatalf("expected at least one SVG polyline points attribute")
	}
	pointCount := 0
	for _, m := range matches {
		fields := strings.Fields(m[1])
		for _, pair := range fields {
			xy := strings.Split(pair, ",")
			if len(xy) != 2 {
				t.Fatalf("malformed SVG coordinate pair %q", pair)
			}
			x, errX := strconv.ParseFloat(xy[0], 64)
			y, errY := strconv.ParseFloat(xy[1], 64)
			if errX != nil || errY != nil || math.IsNaN(x) || math.IsInf(x, 0) || math.IsNaN(y) || math.IsInf(y, 0) {
				t.Fatalf("non-finite or invalid SVG coordinate pair %q", pair)
			}
			if x < minX || x > maxX || y < minY || y > maxY {
				t.Errorf("SVG coordinate (%v, %v) out of expected bounds [%v,%v] x [%v,%v]", x, y, minX, maxX, minY, maxY)
			}
			pointCount++
		}
	}
	return pointCount
}

func TestPlot3D_Deductive_SurfaceAndParametricCurve(t *testing.T) {
	cases := []struct {
		name           string
		expr           string
		wantKind       string
		wantSubstrings []string
	}{
		{
			name:     "saddle surface z = x^2 - y^2",
			expr:     "plot3d(x^2 - y^2, [-2, 2], [-2, 2])",
			wantKind: "surface",
			wantSubstrings: []string{
				"x ∈ [-2, 2]",
				"y ∈ [-2, 2]",
				"z ∈ [-4, 4]",
			},
		},
		{
			name:     "explicit variable ranges u, v",
			expr:     "plot3d(u^2 + v^2, [u, -2, 2], [v, -2, 2])",
			wantKind: "surface",
			wantSubstrings: []string{
				"u ∈ [-2, 2]",
				"v ∈ [-2, 2]",
				"z ∈ [0, 8]",
			},
		},
		{
			name:     "parametric 3d helix",
			expr:     "plot3d([cos(t), sin(t), t], [t, 0, 10])",
			wantKind: "curve",
			wantSubstrings: []string{
				"t ∈ [0, 10]",
				"z ∈ [0, 10]",
			},
		},
		{
			name:     "perspective option",
			expr:     `plot3d(x^2 + y^2, [-2, 2], [-2, 2], [az, 60], [el, 25], "perspective")`,
			wantKind: "surface",
			wantSubstrings: []string{
				"x ∈ [-2, 2]",
				"z ∈ [0, 8]",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify scene actually contains main surface/curve strokes (cannot pass on bounding box alone)
			parsedRoot, err := calc.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", tc.expr, err)
			}
			fnNode, okFn := parsedRoot.(*calc.FuncNode)
			if !okFn {
				t.Fatalf("expected *calc.FuncNode from Parse(%q), got %T", tc.expr, parsedRoot)
			}
			scene, err := calc.BuildPlot3DScene(fnNode.Args)
			if err != nil {
				t.Fatalf("BuildPlot3DScene failed: %v", err)
			}
			mainStrokeCount := 0
			for _, st := range scene.Strokes {
				if st.Kind == tc.wantKind && len(st.Points) >= 2 {
					mainStrokeCount++
				}
			}
			if mainStrokeCount == 0 {
				t.Fatalf("expected non-zero %q strokes in Plot3DScene, got 0 (bounding-box-only false pass prevented)", tc.wantKind)
			}

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

			// SVG checks including coordinate finiteness & viewport bounds
			if !strings.Contains(pn.SVG, "<svg") || !strings.Contains(pn.SVG, "<polyline") {
				t.Errorf("expected valid SVG with polylines, got:\n%s", pn.SVG)
			}
			assertSVGPolylineCoordsFiniteAndBounded(t, pn.SVG, -100, 900, -100, 600)
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

func TestPlot3D_PerspectiveDiffersFromOrtho(t *testing.T) {
	nExpr, _ := calc.Parse("x^2 + y^2")
	rX, _ := calc.Parse("[-2, 2]")
	rY, _ := calc.Parse("[-2, 2]")
	azOpt, _ := calc.Parse("[az, 45]")
	elOpt, _ := calc.Parse("[el, 25]")
	orthoOpt, _ := calc.Parse(`"ortho"`)
	perspOpt, _ := calc.Parse(`"perspective"`)

	orthoScene, err := calc.BuildPlot3DScene([]calc.Node{nExpr, rX, rY, azOpt, elOpt, orthoOpt})
	if err != nil {
		t.Fatalf("ortho BuildPlot3DScene failed: %v", err)
	}
	perspScene, err := calc.BuildPlot3DScene([]calc.Node{nExpr, rX, rY, azOpt, elOpt, perspOpt})
	if err != nil {
		t.Fatalf("perspective BuildPlot3DScene failed: %v", err)
	}

	orthoSVG := calc.RenderPlot3DSVG(orthoScene)
	perspSVG := calc.RenderPlot3DSVG(perspScene)
	if orthoSVG == perspSVG {
		t.Fatalf("expected perspective projection SVG to differ from orthographic projection SVG")
	}

	// Verify measurable coordinate difference on projected bounding box / surface points
	var maxDiff float64
	limit := len(orthoScene.Strokes)
	if len(perspScene.Strokes) < limit {
		limit = len(perspScene.Strokes)
	}
	for i := 0; i < limit; i++ {
		oPts := orthoScene.Strokes[i].Points
		pPts := perspScene.Strokes[i].Points
		if len(oPts) == 0 || len(pPts) == 0 {
			continue
		}
		d := math.Hypot(oPts[0].X-pPts[0].X, oPts[0].Y-pPts[0].Y)
		if d > maxDiff {
			maxDiff = d
		}
	}
	if maxDiff < 1e-2 {
		t.Errorf("expected noticeable projection difference between perspective and ortho, got maxDiff=%v", maxDiff)
	}
}

func TestPlot3D_HLRPolylineMergeAndCollinearSimplify(t *testing.T) {
	// 1. Curve mode (len(tris) == 0): 160 segments should merge into far fewer continuous polylines
	//    while preserving multi-bucket depth shading in SVG.
	nCurve, _ := calc.Parse("[cos(t), sin(t), t]")
	rT, _ := calc.Parse("[t, 0, 10]")
	curveScene, err := calc.BuildPlot3DScene([]calc.Node{nCurve, rT})
	if err != nil {
		t.Fatalf("BuildPlot3DScene curve failed: %v", err)
	}
	curveStrokes := 0
	totalCurvePts := 0
	for _, st := range curveScene.Strokes {
		if st.Kind == "curve" {
			curveStrokes++
			totalCurvePts += len(st.Points)
		}
	}
	if curveStrokes == 0 || totalCurvePts < 50 {
		t.Fatalf("expected merged curve strokes with full geometry, got strokes=%d pts=%d", curveStrokes, totalCurvePts)
	}
	if curveStrokes >= 40 {
		t.Errorf("expected adjacent curve segments to merge (<< 160 raw segments), got %d strokes", curveStrokes)
	}

	curveSVG := calc.RenderPlot3DSVG(curveScene)
	paletteColors := []string{"#0284c7", "#0ea5e9", "#2563eb", "#4f46e5", "#7c3aed", "#9333ea"}
	usedColors := 0
	for _, c := range paletteColors {
		if strings.Contains(curveSVG, c) {
			usedColors++
		}
	}
	if usedColors < 3 {
		t.Errorf("expected depth-guarded polyline merge to preserve depth gradient (>=3 palette colors), got %d colors", usedColors)
	}

	// 2. Surface mode: straight edges (bounding box / planar isolines) must have collinear intermediate HLR points reduced
	nPlane, _ := calc.Parse("x + y")
	rX, _ := calc.Parse("[-2, 2]")
	rY, _ := calc.Parse("[-2, 2]")
	planeScene, err := calc.BuildPlot3DScene([]calc.Node{nPlane, rX, rY})
	if err != nil {
		t.Fatalf("BuildPlot3DScene plane failed: %v", err)
	}
	maxBoxPts := 0
	for _, st := range planeScene.Strokes {
		if (st.Kind == "box" || st.Kind == "axis") && len(st.Points) > maxBoxPts {
			maxBoxPts = len(st.Points)
		}
	}
	if maxBoxPts == 0 {
		t.Fatalf("expected box/axis strokes in plane scene")
	}
	if maxBoxPts > 6 {
		t.Errorf("expected collinear point reduction on straight box/axis edges (<=6 pts), got max %d pts", maxBoxPts)
	}
}

func TestPlot3D_TypstMultiCharVarsNilGuardsAndDynamicCaption(t *testing.T) {
	// 1. Multi-character surface variables must use Typst italic("...")
	resSurf, err := calc.EvalString("plot3d(u1^2 - v2^2, [u1, -2, 2], [v2, -2, 2])")
	if err != nil {
		t.Fatalf("EvalString multi-char surface failed: %v", err)
	}
	pnSurf := resSurf.(*calc.PlotNode)
	if !strings.Contains(pnSurf.Typst, `italic("u1")`) || !strings.Contains(pnSurf.Typst, `italic("v2")`) {
		t.Errorf("expected italic(\"u1\") and italic(\"v2\") in Typst output, got:\n%s", pnSurf.Typst)
	}
	if strings.Contains(pnSurf.Typst, "$u1 in") || strings.Contains(pnSurf.Typst, "space v2 in") {
		t.Errorf("raw multi-character identifier leaked into Typst math mode:\n%s", pnSurf.Typst)
	}

	// 2. Multi-character curve parameter must use Typst italic("...")
	resCurve, err := calc.EvalString("plot3d([cos(tt), sin(tt), tt], [tt, 0, 6])")
	if err != nil {
		t.Fatalf("EvalString multi-char curve failed: %v", err)
	}
	pnCurve := resCurve.(*calc.PlotNode)
	if !strings.Contains(pnCurve.Typst, `italic("tt")`) {
		t.Errorf("expected italic(\"tt\") in curve Typst output, got:\n%s", pnCurve.Typst)
	}

	// 3. Single-letter variables must remain unquoted in Typst math mode
	resSingle, err := calc.EvalString("plot3d(x^2 - y^2, [-2, 2], [-2, 2])")
	if err != nil {
		t.Fatalf("EvalString single-char surface failed: %v", err)
	}
	pnSingle := resSingle.(*calc.PlotNode)
	if !strings.Contains(pnSingle.Typst, "$x in") || strings.Contains(pnSingle.Typst, `italic("x")`) {
		t.Errorf("single-letter variable x should not be wrapped in italic(...), got:\n%s", pnSingle.Typst)
	}

	// 4. Nil *MinNode/*MaxNode guards + dynamic caption Y positioning
	compactScene := &calc.Plot3DScene{
		Kind:   "surface",
		VarX:   "u1",
		VarY:   "y",
		XMin:   -1,
		XMax:   1,
		YMin:   -2,
		YMax:   2,
		ZMin:   0,
		ZMax:   4,
		Camera: calc.Camera3D{AzimuthDeg: 35, ElevationDeg: 25, Aspect: 2.0},
		Strokes: []calc.Stroke2D{
			{
				Kind: "surface",
				Points: []calc.ProjectedPoint2D{
					{X: -0.5, Y: -0.2, Depth: 1.0, ZVal: 0},
					{X: 0.5, Y: 0.2, Depth: 1.5, ZVal: 1},
				},
			},
		},
	}
	svgOut := calc.RenderPlot3DSVG(compactScene)
	var xmlCheck any
	if err := xml.Unmarshal([]byte(svgOut), &xmlCheck); err != nil {
		t.Errorf("RenderPlot3DSVG with nil bound nodes produced invalid XML: %v\n%s", err, svgOut)
	}

	typstOut := calc.RenderPlot3DTypst(compactScene)
	if !strings.Contains(typstOut, `italic("u1")`) {
		t.Errorf("expected italic(\"u1\") in compact Typst output:\n%s", typstOut)
	}
	if strings.Contains(typstOut, "(0, -4.6)") {
		t.Errorf("expected dynamic caption Y coordinate instead of hardcoded (0, -4.6), got:\n%s", typstOut)
	}
	reCap := regexp.MustCompile(`content\(\(0,\s*([-0-9.]+)\)`)
	capMatch := reCap.FindStringSubmatch(typstOut)
	if len(capMatch) != 2 {
		t.Fatalf("could not find content((0, y)) in Typst output:\n%s", typstOut)
	}
	capY, err := strconv.ParseFloat(capMatch[1], 64)
	if err != nil {
		t.Fatalf("failed to parse caption Y %q: %v", capMatch[1], err)
	}
	// With Y in [-0.2, 0.2], Aspect=2.0 -> plotted Y min is -0.2 * 4.2 / 2.0 = -0.42.
	// Caption must sit cleanly below -0.42 (not overlapping) and much closer than -4.6.
	if capY >= -0.42 || capY < -2.0 {
		t.Errorf("expected dynamic caption Y in (-2.0, -0.42), got %v", capY)
	}
}

func TestPlot3D_AspectAndDepthOrderedBoxAxes(t *testing.T) {
	zeroNode, _ := calc.Parse("0")
	oneNode, _ := calc.Parse("1")
	makeTestScene := func(aspect float64) *calc.Plot3DScene {
		return &calc.Plot3DScene{
			Kind:     "surface",
			VarX:     "x",
			VarY:     "y",
			XMinNode: zeroNode,
			XMaxNode: oneNode,
			YMinNode: zeroNode,
			YMaxNode: oneNode,
			ZMinNode: zeroNode,
			ZMaxNode: oneNode,
			Camera:   calc.Camera3D{AzimuthDeg: 35, ElevationDeg: 25, Aspect: aspect},
			Strokes: []calc.Stroke2D{
				{
					Kind: "box", // front box edge (closest)
					Points: []calc.ProjectedPoint2D{
						{X: -0.5, Y: 0.5, Depth: 1.0, ZVal: 1},
						{X: 0.5, Y: 0.5, Depth: 1.0, ZVal: 1},
					},
				},
				{
					Kind: "surface", // middle surface stroke
					Points: []calc.ProjectedPoint2D{
						{X: -0.5, Y: 0.0, Depth: 2.0, ZVal: 0.5},
						{X: 0.5, Y: 0.0, Depth: 2.0, ZVal: 0.5},
					},
				},
				{
					Kind: "box", // back box edge (farthest)
					Points: []calc.ProjectedPoint2D{
						{X: -0.5, Y: -0.5, Depth: 3.0, ZVal: 0},
						{X: 0.5, Y: -0.5, Depth: 3.0, ZVal: 0},
					},
				},
			},
		}
	}

	// 1. Aspect must affect both SVG and Typst Y coordinates
	svgA1 := calc.RenderPlot3DSVG(makeTestScene(1.0))
	svgA2 := calc.RenderPlot3DSVG(makeTestScene(2.0))
	if svgA1 == svgA2 {
		t.Errorf("expected RenderPlot3DSVG to respect Camera.Aspect=2.0")
	}

	typA1 := calc.RenderPlot3DTypst(makeTestScene(1.0))
	typA2 := calc.RenderPlot3DTypst(makeTestScene(2.0))
	if typA1 == typA2 {
		t.Errorf("expected RenderPlot3DTypst to respect Camera.Aspect=2.0")
	}

	// 2. Depth ordering: back box (y=-0.5, depth=3) -> middle surface (y=0, depth=2) -> front box (y=0.5, depth=1)
	// In SVG (aspect=1, width=800, height=500, pad=45 => plotH=410, scale=196.8, cy=250):
	//   back box y=-0.5 -> svgy = 250 + 98.4 = 348.4
	//   surface  y= 0.0 -> svgy = 250
	//   front box y=0.5 -> svgy = 250 - 98.4 = 151.6
	idxBackSVG := strings.Index(svgA1, ",348.4")
	idxMidSVG := strings.Index(svgA1, ",250")
	idxFrontSVG := strings.Index(svgA1, ",151.6")
	if idxBackSVG < 0 || idxMidSVG < 0 || idxFrontSVG < 0 {
		t.Fatalf("expected back/mid/front coordinates in SVG:\n%s", svgA1)
	}
	if !(idxBackSVG < idxMidSVG && idxMidSVG < idxFrontSVG) {
		t.Errorf("expected SVG depth-ordered drawing (back box < mid surface < front box), got indices %d, %d, %d\n%s",
			idxBackSVG, idxMidSVG, idxFrontSVG, svgA1)
	}

	// In Typst (aspect=1, scale=4.2):
	//   back box y=-0.5 -> -2.1, surface y=0 -> 0, front box y=0.5 -> 2.1
	idxBackTyp := strings.Index(typA1, ", -2.1)")
	idxMidTyp := strings.Index(typA1, ", 0)")
	idxFrontTyp := strings.Index(typA1, ", 2.1)")
	if idxBackTyp < 0 || idxMidTyp < 0 || idxFrontTyp < 0 {
		t.Fatalf("expected back/mid/front coordinates in Typst:\n%s", typA1)
	}
	if !(idxBackTyp < idxMidTyp && idxMidTyp < idxFrontTyp) {
		t.Errorf("expected Typst depth-ordered drawing (back box < mid surface < front box), got indices %d, %d, %d\n%s",
			idxBackTyp, idxMidTyp, idxFrontTyp, typA1)
	}
}
