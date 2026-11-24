package calc

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

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
	if !isFiniteFloat(depth) || maxD-minD < 1e-9 || buckets <= 1 {
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
				if !isFiniteFloat(pt.Depth) {
					continue
				}
				if pt.Depth < minD {
					minD = pt.Depth
				}
				if pt.Depth > maxD {
					maxD = pt.Depth
				}
			}
		}
	}
	if !isFiniteFloat(minD) || !isFiniteFloat(maxD) || maxD-minD < 1e-9 {
		return -1.0, 1.0
	}
	return minD, maxD
}

type orderedStroke struct {
	stroke   Stroke2D
	avgDepth float64
	priority int
}

func strokeKindPriority(kind string) int {
	switch kind {
	case "box":
		return 0
	case "axis":
		return 1
	default:
		return 2
	}
}

func orderedVectorStrokes(strokes []Stroke2D) []orderedStroke {
	out := make([]orderedStroke, 0, len(strokes))
	for _, st := range strokes {
		if len(st.Points) < 2 {
			continue
		}
		sumD := 0.0
		cnt := 0
		for _, pt := range st.Points {
			if isFiniteFloat(pt.Depth) {
				sumD += pt.Depth
				cnt++
			}
		}
		avgD := 0.0
		if cnt > 0 {
			avgD = sumD / float64(cnt)
		}
		out = append(out, orderedStroke{
			stroke:   st,
			avgDepth: avgD,
			priority: strokeKindPriority(st.Kind),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		diff := out[i].avgDepth - out[j].avgDepth
		if math.Abs(diff) > 1e-12 {
			return out[i].avgDepth > out[j].avgDepth
		}
		return out[i].priority < out[j].priority
	})
	return out
}

func formatBoundSVG(n Node, fallback float64) string {
	if n == nil {
		return fmtSVGNum(fallback)
	}
	return Format(n)
}

func formatBoundTypst(n Node, fallback float64) string {
	if n == nil {
		return fmtTypstNum(fallback)
	}
	return formatTypstNode(n)
}

func formatTypstMathIdent(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "x"
	}
	runes := []rune(trimmed)
	if len(runes) == 1 && unicode.IsLetter(runes[0]) {
		return trimmed
	}
	escaped := strings.ReplaceAll(trimmed, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return fmt.Sprintf(`italic("%s")`, escaped)
}

type svgStrokeStyle struct {
	color     string
	width     string
	roundCaps bool
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
	aspect := effectiveAspect(scene.Camera)
	scale := math.Min(plotW, plotH*aspect) * 0.48
	cx := width * 0.5
	cy := height * 0.5

	toSVGX := func(x float64) float64 { return cx + x*scale }
	toSVGY := func(y float64) float64 { return cy - y*scale/aspect }

	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 500" width="800" height="500">`)
	sb.WriteByte('\n')

	exprText := ""
	if scene.Expr != nil {
		exprText = Format(scene.Expr)
	}
	xMinText := formatBoundSVG(scene.XMinNode, scene.XMin)
	xMaxText := formatBoundSVG(scene.XMaxNode, scene.XMax)
	yMinText := formatBoundSVG(scene.YMinNode, scene.YMin)
	yMaxText := formatBoundSVG(scene.YMaxNode, scene.YMax)
	zMinText := formatBoundSVG(scene.ZMinNode, scene.ZMin)
	zMaxText := formatBoundSVG(scene.ZMaxNode, scene.ZMax)

	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(i18n.T("plot3d.svg_title", exprText))))
	if scene.Kind == "curve" {
		sb.WriteString(fmt.Sprintf("  <desc>%s</desc>\n",
			html.EscapeString(i18n.T("plot3d.svg_desc_curve",
				scene.VarX, xMinText, xMaxText,
				zMinText, zMaxText,
			))))
	} else {
		sb.WriteString(fmt.Sprintf("  <desc>%s</desc>\n",
			html.EscapeString(i18n.T("plot3d.svg_desc_surface",
				scene.VarX, xMinText, xMaxText,
				scene.VarY, yMinText, yMaxText,
				zMinText, zMaxText,
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

	// Depth-ordered drawing integrating box, axis, surface, and curve back-to-front.
	// Consecutive strokes sharing an identical style are grouped into a single <g>.
	minD, maxD := computeMainDepthBounds(scene.Strokes)
	ordered := orderedVectorStrokes(scene.Strokes)
	groupOpen := false
	var currentStyle svgStrokeStyle

	openGroup := func(style svgStrokeStyle) {
		if style.roundCaps {
			sb.WriteString(fmt.Sprintf(`    <g fill="none" stroke="%s" stroke-width="%s" stroke-linecap="round" stroke-linejoin="round">`,
				style.color, style.width))
		} else {
			sb.WriteString(fmt.Sprintf(`    <g fill="none" stroke="%s" stroke-width="%s">`,
				style.color, style.width))
		}
		sb.WriteByte('\n')
		currentStyle = style
		groupOpen = true
	}

	for _, item := range ordered {
		st := item.stroke
		var pts []string
		for _, pt := range st.Points {
			if !isFiniteFloat(pt.X) || !isFiniteFloat(pt.Y) {
				continue
			}
			pts = append(pts, fmt.Sprintf("%s,%s", fmtSVGNum(toSVGX(pt.X)), fmtSVGNum(toSVGY(pt.Y))))
		}
		if len(pts) < 2 {
			continue
		}

		var style svgStrokeStyle
		switch st.Kind {
		case "box":
			style = svgStrokeStyle{color: "#cbd5e1", width: "1", roundCaps: false}
		case "axis":
			style = svgStrokeStyle{color: "#64748b", width: "1.3", roundCaps: false}
		default:
			bi := depthBucketIndex(item.avgDepth, minD, maxD, len(svgDepthPalette))
			style = svgStrokeStyle{color: svgDepthPalette[bi], width: "1.6", roundCaps: true}
		}

		if !groupOpen || style != currentStyle {
			if groupOpen {
				sb.WriteString("    </g>\n")
			}
			openGroup(style)
		}
		sb.WriteString(fmt.Sprintf(`      <polyline points="%s"/>`, strings.Join(pts, " ")))
		sb.WriteByte('\n')
	}
	if groupOpen {
		sb.WriteString("    </g>\n")
	}

	sb.WriteString("  </g>\n")

	// Exact domain caption
	var domainCaption string
	if scene.Kind == "curve" {
		domainCaption = i18n.T("plot3d.svg_caption_curve",
			scene.VarX, xMinText, xMaxText,
			zMinText, zMaxText)
	} else {
		domainCaption = i18n.T("plot3d.svg_caption_surface",
			scene.VarX, xMinText, xMaxText,
			scene.VarY, yMinText, yMaxText,
			zMinText, zMaxText)
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
	const baseHalf = 4.2
	aspect := effectiveAspect(scene.Camera)
	scale := math.Min(baseHalf, baseHalf*aspect)
	toTypstX := func(x float64) float64 { return x * scale }
	toTypstY := func(y float64) float64 { return y * scale / aspect }

	var sb strings.Builder
	sb.WriteString(`#import "@preview/cetz:0.5.2"`)
	sb.WriteByte('\n')
	sb.WriteByte('\n')
	sb.WriteString("#cetz.canvas({\n")
	sb.WriteString("  import cetz.draw: *\n")

	minD, maxD := computeMainDepthBounds(scene.Strokes)
	ordered := orderedVectorStrokes(scene.Strokes)
	minTypstY := math.Inf(1)
	hasPlottedY := false

	for _, item := range ordered {
		st := item.stroke
		var pts []string
		for _, pt := range st.Points {
			if !isFiniteFloat(pt.X) || !isFiniteFloat(pt.Y) {
				continue
			}
			tx := toTypstX(pt.X)
			ty := toTypstY(pt.Y)
			if !isFiniteFloat(tx) || !isFiniteFloat(ty) {
				continue
			}
			if ty < minTypstY {
				minTypstY = ty
				hasPlottedY = true
			}
			pts = append(pts, fmt.Sprintf("(%s, %s)", fmtTypstNum(tx), fmtTypstNum(ty)))
		}
		if len(pts) < 2 {
			continue
		}

		var strokeStyle string
		switch st.Kind {
		case "box":
			strokeStyle = "(paint: luma(200), thickness: 0.6pt)"
		case "axis":
			strokeStyle = "(paint: luma(110), thickness: 0.9pt)"
		default:
			bi := depthBucketIndex(item.avgDepth, minD, maxD, len(typstDepthPalette))
			strokeStyle = fmt.Sprintf("(paint: %s, thickness: 1pt)", typstDepthPalette[bi])
		}
		sb.WriteString(fmt.Sprintf("  line(%s, stroke: %s)\n", strings.Join(pts, ", "), strokeStyle))
	}

	captionY := -4.6
	if hasPlottedY && isFiniteFloat(minTypstY) {
		captionY = minTypstY - 0.35
	}
	varXTypst := formatTypstMathIdent(scene.VarX)
	varYTypst := formatTypstMathIdent(scene.VarY)
	xMinTypst := formatBoundTypst(scene.XMinNode, scene.XMin)
	xMaxTypst := formatBoundTypst(scene.XMaxNode, scene.XMax)
	yMinTypst := formatBoundTypst(scene.YMinNode, scene.YMin)
	yMaxTypst := formatBoundTypst(scene.YMaxNode, scene.YMax)
	zMinTypst := formatBoundTypst(scene.ZMinNode, scene.ZMin)
	zMaxTypst := formatBoundTypst(scene.ZMaxNode, scene.ZMax)

	if scene.Kind == "curve" {
		sb.WriteString(fmt.Sprintf("  content((0, %s), [$%s in [%s, %s], space z in [%s, %s]$], anchor: \"north\")\n",
			fmtTypstNum(captionY),
			varXTypst, xMinTypst, xMaxTypst,
			zMinTypst, zMaxTypst))
	} else {
		sb.WriteString(fmt.Sprintf("  content((0, %s), [$%s in [%s, %s], space %s in [%s, %s], space z in [%s, %s]$], anchor: \"north\")\n",
			fmtTypstNum(captionY),
			varXTypst, xMinTypst, xMaxTypst,
			varYTypst, yMinTypst, yMaxTypst,
			zMinTypst, zMaxTypst))
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
