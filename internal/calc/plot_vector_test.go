package calc_test

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

func TestPlotVector_Slice3_PlotSceneAndSVG(t *testing.T) {
	cases := []struct {
		name           string
		expr           string
		minBranches    int
		wantSubstrings []string
	}{
		{
			name:        "1/x rational pole and asymptotes",
			expr:        "plot(1/x, [-3, 3], [-5, 5])",
			minBranches: 2,
			wantSubstrings: []string{
				"<svg",
				"<clipPath",
				"<title>",
				"<desc>",
				"stroke-dasharray",
			},
		},
		{
			name:        "x^2 - 2 parabola exact roots and extremum",
			expr:        "plot(x^2 - 2, [-2, 2])",
			minBranches: 1,
			wantSubstrings: []string{
				"<svg",
				"<path",
				"√2",
			},
		},
		{
			name:        "removable singularity (x^2-1)/(x-1)",
			expr:        "plot((x^2 - 1)/(x - 1), [-2, 3])",
			minBranches: 1,
			wantSubstrings: []string{
				"<svg",
				"<circle",
			},
		},
		{
			name:        "oblique asymptote x + 1/x",
			expr:        "plot(x + 1/x, [-4, 4], [-6, 6])",
			minBranches: 2,
			wantSubstrings: []string{
				"<svg",
				"stroke-dasharray",
			},
		},
		{
			name:        "periodic pole 1/sin(x)",
			expr:        "plot(1/sin(x), [-4, 4], [-5, 5])",
			minBranches: 3,
			wantSubstrings: []string{
				"<svg",
				"<path",
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
				t.Fatalf("expected *PlotNode, got %T", res)
			}
			if pn.SVG == "" {
				t.Fatalf("expected non-empty SVG on PlotNode")
			}
			// Must not leak NaN, Inf, or exponential notation
			for _, bad := range []string{"NaN", "Inf", "-Inf", "+Inf"} {
				if strings.Contains(pn.SVG, bad) {
					t.Errorf("SVG contains invalid token %q:\n%s", bad, pn.SVG)
				}
			}
			// Must be valid XML
			var v any
			if err := xml.Unmarshal([]byte(pn.SVG), &v); err != nil {
				t.Fatalf("SVG is not valid XML: %v\n%s", err, pn.SVG)
			}
			for _, sub := range tc.wantSubstrings {
				if !strings.Contains(pn.SVG, sub) {
					t.Errorf("SVG missing %q:\n%s", sub, pn.SVG)
				}
			}
			branchCount := strings.Count(pn.SVG, "M ")
			if branchCount < tc.minBranches {
				t.Errorf("expected at least %d SVG branches (M subpaths), got %d:\n%s", tc.minBranches, branchCount, pn.SVG)
			}
			// Check Typst cetz-plot projection
			typstOut := calc.FormatTypst(pn)
			for _, sub := range []string{
				`#import "@preview/cetz:0.5.2"`,
				`#import "@preview/cetz-plot:0.1.4": plot`,
				"plot.plot(",
				"x-min:",
				"y-min:",
				"plot.add(",
			} {
				if !strings.Contains(typstOut, sub) {
					t.Errorf("Typst plot output missing %q:\n%s", sub, typstOut)
				}
			}
			if strings.HasPrefix(typstOut, "$#import") {
				t.Errorf("FormatTypst must not wrap cetz block in $...$")
			}
		})
	}
}

func TestPlotSVG_BuiltinAndFileExport(t *testing.T) {
	// 1. Direct string return
	res, err := calc.EvalString("plot_svg(sin(x), [0, 2*pi])")
	if err != nil {
		t.Fatalf("plot_svg failed: %v", err)
	}
	sn, ok := res.(*calc.StringNode)
	if !ok {
		t.Fatalf("expected *StringNode, got %T", res)
	}
	if !strings.Contains(sn.Value, "<svg") {
		t.Errorf("expected SVG content, got: %s", sn.Value)
	}

	// 2. File export (UTF-8 without BOM)
	outPath := filepath.Join(t.TempDir(), "out.svg")
	expr := `plot_svg(x^2 - 1, [-2, 2], "` + filepath.ToSlash(outPath) + `")`
	_, err = calc.EvalString(expr)
	if err != nil {
		t.Fatalf("plot_svg file export failed: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed reading exported SVG: %v", err)
	}
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		t.Errorf("exported SVG must not contain UTF-8 BOM")
	}
	if !strings.Contains(string(data), "<svg") {
		t.Errorf("exported file does not contain <svg: %s", string(data))
	}

	// 3. Nested directory creation and empty path rejection
	nestedPath := filepath.Join(t.TempDir(), "sub", "dir", "nested.svg")
	if _, err := calc.EvalString(`plot_svg(x, [-1, 1], "` + filepath.ToSlash(nestedPath) + `")`); err != nil {
		t.Fatalf("expected nested dir plot_svg to succeed, got: %v", err)
	}
	if _, err := os.Stat(nestedPath); err != nil {
		t.Fatalf("expected nested SVG file to exist: %v", err)
	}
	if _, err := calc.EvalString(`plot_svg(x, [-1, 1], "   ")`); err == nil {
		t.Errorf("expected error for empty output path in plot_svg")
	}
}

func TestPlotVector_AuditRegressions(t *testing.T) {
	// 1. Sum-form oblique asymptote: x + 1/x must detect y = x oblique asymptote
	exprSum, err := calc.Parse("x + 1/x")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if quot, ok := calc.ExtractPolyQuotient(exprSum, "x"); !ok || quot == nil {
		t.Errorf("expected ExtractPolyQuotient(x + 1/x, x) to succeed, got ok=%v", ok)
	}
	resSum, err := calc.EvalString("plot(x + 1/x, [-4, 4], [-6, 6])")
	if err != nil {
		t.Fatalf("plot(x + 1/x) failed: %v", err)
	}
	typSum := calc.FormatTypst(resSum)
	if !strings.Contains(typSum, "x =>") {
		t.Errorf("expected oblique asymptote plot.add(domain: ..., x => ...) in Typst output, got:\n%s", typSum)
	}

	// 2. Removable singularity (x^2 - 1)/(x - 1) must NOT emit an oblique asymptote
	exprHole, err := calc.Parse("(x^2 - 1)/(x - 1)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if quot, ok := calc.ExtractPolyQuotient(exprHole, "x"); ok {
		t.Errorf("expected ExtractPolyQuotient((x^2-1)/(x-1), x) to return false (rem==0), got %v", quot)
	}
	resHole, err := calc.EvalString("plot((x^2 - 1)/(x - 1), [-2, 3])")
	if err != nil {
		t.Fatalf("plot hole failed: %v", err)
	}
	if strings.Contains(calc.FormatTypst(resHole), "x =>") {
		t.Errorf("removable singularity must not emit oblique asymptote")
	}

	// 3. Small-scale amplitude sin(x)/10000 must preserve non-zero y-min/y-max precision in Typst
	resSmall, err := calc.EvalString("plot(sin(x)/10000, [-3, 3])")
	if err != nil {
		t.Fatalf("small scale plot failed: %v", err)
	}
	typSmall := calc.FormatTypst(resSmall)
	if strings.Contains(typSmall, "y-min: 0, y-max: 0") {
		t.Errorf("small-scale plot collapsed y bounds to 0:\n%s", typSmall)
	}

	// 4. Shifted viewport periodic poles tan(x) + 10 on [5, 15] must split into multiple branches
	resShift, err := calc.EvalString("plot(tan(x) + 10, [-4, 4], [5, 15])")
	if err != nil {
		t.Fatalf("shifted tan plot failed: %v", err)
	}
	pnShift := resShift.(*calc.PlotNode)
	if cnt := strings.Count(pnShift.SVG, "M "); cnt < 3 {
		t.Errorf("expected shifted tan(x)+10 to split into >= 3 branches, got %d", cnt)
	}

	// 5. Partial domain sqrt(x) on [-10, 10] must not exhaust budget on negative side
	resSqrt, err := calc.EvalString("plot(sqrt(x), [-10, 10])")
	if err != nil {
		t.Fatalf("sqrt plot failed: %v", err)
	}
	pnSqrt := resSqrt.(*calc.PlotNode)
	if !strings.Contains(pnSqrt.SVG, "<path") {
		t.Errorf("expected valid curve path on positive side of sqrt(x)")
	}
}
