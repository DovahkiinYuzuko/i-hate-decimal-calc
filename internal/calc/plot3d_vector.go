package calc

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

var svgDepthPalette = []string{
	"#0284c7", // nearest
	"#0ea5e9",
	"#2563eb",
	"#4f46e5",
	"#7c3aed",
	"#9333ea", // farthest
}

func depthBucketIndex(depth, minD, maxD float64, buckets int) int {
	if maxD-minD < 1e-9 || buckets <= 1 {
		return 0
	}
	t := (depth - minD) / (maxD - minD)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	idx := int(math.Round(t * float64(buckets-1)))
	if idx < 0 {
		return 0
	}
	if idx >= buckets {
		return buckets - 1
	}
	return idx
}

func computeMainDepthBounds(strokes []Stroke2D) (float64, float64) {
	minD, maxD := math.Inf(1), math.Inf(-1)
	for _, st := range strokes {
		if st.Kind == "surface" || st.Kind == "curve" {
			for _, pt := range st.Points {
				if pt.Depth < minD {
					minD = pt.Depth
				}
				if pt.Depth > maxD {
					maxD = pt.Depth
				}
			}
		}
	}
	if math.IsInf(minD, 1) || maxD-minD < 1e-9 {
		return -1.0, 1.0
	}
	return minD, maxD
}

// RenderPlot3DSVG renders a Plot3DScene into a standalone, XML-valid SVG string.
func RenderPlot3DSVG(scene *Plot3DScene) string {
	if scene == nil {
		return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 500"></svg>`
	}

	const (
		width  = 800.0
		height = 500.0
		pad    = 45.0
	)
	plotW := width - 2*pad
	plotH := height - 2*pad
	scale := math.Min(plotW, plotH) * 0.48
	cx := width * 0.5
	cy := height * 0.5

	toSVGX := func(x float64) float64 { return cx + x*scale }
	toSVGY := func(y float64) float64 { return cy - y*scale }

	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 500" width="800" height="500">`)
	sb.WriteByte('\n')

	exprText := ""
	if scene.Expr != nil {
		exprText = Format(scene.Expr)
	}
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(i18n.T("plot3d.svg_title", exprText))))
	if scene.Kind == "curve" {
		sb.WriteString(fmt.Sprintf("  <desc>%s</desc>\n",
			html.EscapeString(i18n.T("plot3d.svg_desc_curve",
				scene.VarX, Format(scene.XMinNode), Format(scene.XMaxNode),
				Format(scene.ZMinNode), Format(scene.ZMaxNode),
			))))
	} else {
		sb.WriteString(fmt.Sprintf("  <desc>%s</desc>\n",
			html.EscapeString(i18n.T("plot3d.svg_desc_surface",
				scene.VarX, Format(scene.XMinNode), Format(scene.XMaxNode),
				scene.VarY, Format(scene.YMinNode), Format(scene.YMaxNode),
				Format(scene.ZMinNode), Format(scene.ZMaxNode),
			))))
	}

	sb.WriteString("  <defs>\n")
	sb.WriteString(fmt.Sprintf(`    <clipPath id="plot3d-area"><rect x="%s" y="%s" width="%s" height="%s"/></clipPath>`,
		fmtSVGNum(pad), fmtSVGNum(pad), fmtSVGNum(plotW), fmtSVGNum(plotH)))
	sb.WriteString("\n  </defs>\n")

	sb.WriteString(`  <rect x="0" y="0" width="800" height="500" fill="#ffffff"/>`)
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf(`  <rect x="%s" y="%s" width="%s" height="%s" fill="#f8fafc" stroke="#cbd5e1" stroke-width="1"/>`,
		fmtSVGNum(pad), fmtSVGNum(pad), fmtSVGNum(plotW), fmtSVGNum(plotH)))
	sb.WriteByte('\n')

	sb.WriteString(`  <g clip-path="url(#plot3d-area)">`)
	sb.WriteByte('\n')

	// Box and axes first
	for _, st := range scene.Strokes {
		if (st.Kind != "box" && st.Kind != "axis") || len(st.Points) < 2 {
			continue
		}
		strokeCol := "#cbd5e1"
		strokeW := "1"
		if st.Kind == "axis" {
			strokeCol = "#64748b"
			strokeW = "1.3"
		}
		var pts []string
		for _, pt := range st.Points {
			pts = append(pts, fmt.Sprintf("%s,%s", fmtSVGNum(toSVGX(pt.X)), fmtSVGNum(toSVGY(pt.Y))))
		}
		sb.WriteString(fmt.Sprintf(`    <polyline fill="none" stroke="%s" stroke-width="%s" points="%s"/>`,
			strokeCol, strokeW, strings.Join(pts, " ")))
		sb.WriteByte('\n')
	}

	// Main strokes grouped by depth bucket (back-to-front bucket order so closer lines draw last)
	minD, maxD := computeMainDepthBounds(scene.Strokes)
	buckets := make([][]Stroke2D, len(svgDepthPalette))
	for _, st := range scene.Strokes {
		if (st.Kind != "surface" && st.Kind != "curve") || len(st.Points) < 2 {
			continue
		}
		avgD := 0.0
		for _, pt := range st.Points {
			avgD += pt.Depth
		}
		avgD /= float64(len(st.Points))
		bi := depthBucketIndex(avgD, minD, maxD, len(svgDepthPalette))
		buckets[bi] = append(buckets[bi], st)
	}

	for bi := len(svgDepthPalette) - 1; bi >= 0; bi-- {
		if len(buckets[bi]) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf(`    <g fill="none" stroke="%s" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">`, svgDepthPalette[bi]))
		sb.WriteByte('\n')
		for _, st := range buckets[bi] {
			var pts []string
			for _, pt := range st.Points {
				pts = append(pts, fmt.Sprintf("%s,%s", fmtSVGNum(toSVGX(pt.X)), fmtSVGNum(toSVGY(pt.Y))))
			}
			sb.WriteString(fmt.Sprintf(`      <polyline points="%s"/>`, strings.Join(pts, " ")))
			sb.WriteByte('\n')
		}
		sb.WriteString("    </g>\n")
	}

	sb.WriteString("  </g>\n")

	// Exact domain caption
	var domainCaption string
	if scene.Kind == "curve" {
		domainCaption = i18n.T("plot3d.svg_caption_curve",
			scene.VarX, Format(scene.XMinNode), Format(scene.XMaxNode),
			Format(scene.ZMinNode), Format(scene.ZMaxNode))
	} else {
		domainCaption = i18n.T("plot3d.svg_caption_surface",
			scene.VarX, Format(scene.XMinNode), Format(scene.XMaxNode),
			scene.VarY, Format(scene.YMinNode), Format(scene.YMaxNode),
			Format(scene.ZMinNode), Format(scene.ZMaxNode))
	}
	sb.WriteString(fmt.Sprintf(`  <text x="400" y="480" font-family="sans-serif" font-size="12" fill="#334155" text-anchor="middle">%s</text>`,
		html.EscapeString(domainCaption)))
	sb.WriteByte('\n')

	sb.WriteString("</svg>\n")
	return sb.String()
}

var typstDepthPalette = []string{
	"rgb(\"#0284c7\")",
	"rgb(\"#0ea5e9\")",
	"rgb(\"#2563eb\")",
	"rgb(\"#4f46e5\")",
	"rgb(\"#7c3aed\")",
	"rgb(\"#9333ea\")",
}

// RenderPlot3DTypst renders pre-projected, HLR-clipped 2D polylines into a 2D CeTZ canvas.
func RenderPlot3DTypst(scene *Plot3DScene) string {
	if scene == nil {
		return ""
	}
	const scale = 4.2

	var sb strings.Builder
	sb.WriteString(`#import "@preview/cetz:0.5.2"`)
	sb.WriteByte('\n')
	sb.WriteByte('\n')
	sb.WriteString("#cetz.canvas({\n")
	sb.WriteString("  import cetz.draw: *\n")

	// Box and axes
	for _, st := range scene.Strokes {
		if (st.Kind != "box" && st.Kind != "axis") || len(st.Points) < 2 {
			continue
		}
		strokeStyle := "(paint: luma(200), thickness: 0.6pt)"
		if st.Kind == "axis" {
			strokeStyle = "(paint: luma(110), thickness: 0.9pt)"
		}
		var pts []string
		for _, pt := range st.Points {
			pts = append(pts, fmt.Sprintf("(%s, %s)", fmtTypstNum(pt.X*scale), fmtTypstNum(pt.Y*scale)))
		}
		sb.WriteString(fmt.Sprintf("  line(%s, stroke: %s)\n", strings.Join(pts, ", "), strokeStyle))
	}

	minD, maxD := computeMainDepthBounds(scene.Strokes)
	buckets := make([][]Stroke2D, len(typstDepthPalette))
	for _, st := range scene.Strokes {
		if (st.Kind != "surface" && st.Kind != "curve") || len(st.Points) < 2 {
			continue
		}
		avgD := 0.0
		for _, pt := range st.Points {
			avgD += pt.Depth
		}
		avgD /= float64(len(st.Points))
		bi := depthBucketIndex(avgD, minD, maxD, len(typstDepthPalette))
		buckets[bi] = append(buckets[bi], st)
	}

	for bi := len(typstDepthPalette) - 1; bi >= 0; bi-- {
		col := typstDepthPalette[bi]
		for _, st := range buckets[bi] {
			var pts []string
			for _, pt := range st.Points {
				pts = append(pts, fmt.Sprintf("(%s, %s)", fmtTypstNum(pt.X*scale), fmtTypstNum(pt.Y*scale)))
			}
			sb.WriteString(fmt.Sprintf("  line(%s, stroke: (paint: %s, thickness: 1pt))\n", strings.Join(pts, ", "), col))
		}
	}

	if scene.Kind == "curve" {
		sb.WriteString(fmt.Sprintf("  content((0, -4.6), [$%s in [%s, %s], space z in [%s, %s]$])\n",
			scene.VarX, formatTypstNode(scene.XMinNode), formatTypstNode(scene.XMaxNode),
			formatTypstNode(scene.ZMinNode), formatTypstNode(scene.ZMaxNode)))
	} else {
		sb.WriteString(fmt.Sprintf("  content((0, -4.6), [$%s in [%s, %s], space %s in [%s, %s], space z in [%s, %s]$])\n",
			scene.VarX, formatTypstNode(scene.XMinNode), formatTypstNode(scene.XMaxNode),
			scene.VarY, formatTypstNode(scene.YMinNode), formatTypstNode(scene.YMaxNode),
			formatTypstNode(scene.ZMinNode), formatTypstNode(scene.ZMaxNode)))
	}
	sb.WriteString("})\n")
	return sb.String()
}

func isPlot3DSVGOutputPath(s string) bool {
	return strings.HasSuffix(strings.ToLower(s), ".svg") || strings.ContainsAny(s, "/\\")
}

// EvalPlot3DSVG evaluates plot3d_svg(...) without an explicit environment.
func EvalPlot3DSVG(args []Node) (Node, error) {
	return EvalPlot3DSVGWithEnv(args, nil)
}

// EvalPlot3DSVGWithEnv evaluates plot3d_svg(...) using env and returns standalone SVG or writes it to the specified path.
func EvalPlot3DSVGWithEnv(args []Node, env *Env) (Node, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot3d_svg_args_count", len(args)))
	}

	var outPath string
	plotArgs := args
	if strNode, ok := args[len(args)-1].(*StringNode); ok {
		trimmed := strings.TrimSpace(strNode.Value)
		if !isPlot3DStringOption(trimmed) {
			if trimmed == "" {
				return nil, fmt.Errorf("%s", i18n.T("errors.plot_svg_empty_path"))
			}
			if isPlot3DSVGOutputPath(trimmed) {
				outPath = strNode.Value
				plotArgs = args[:len(args)-1]
			}
		}
	}

	scene, err := BuildPlot3DSceneWithEnv(plotArgs, env)
	if err != nil {
		return nil, err
	}
	svgStr := RenderPlot3DSVG(scene)

	if outPath != "" {
		dir := filepath.Dir(outPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("%s", i18n.T("errors.plot_svg_write_failed", outPath, err))
			}
		}
		if err := os.WriteFile(outPath, []byte(svgStr), 0644); err != nil {
			return nil, fmt.Errorf("%s", i18n.T("errors.plot_svg_write_failed", outPath, err))
		}
		return NewStringNode(i18n.T("plot.svg_saved", outPath)), nil
	}
	return NewStringNode(svgStr), nil
}
