package calc

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func fmtSVGNum(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	if v > 1e6 {
		v = 1e6
	} else if v < -1e6 {
		v = -1e6
	}
	if math.Abs(v) < 1e-9 {
		v = 0
	}
	s := strconv.FormatFloat(v, 'f', 3, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if s == "-0" || s == "" {
		return "0"
	}
	return s
}

// RenderPlotSVG renders a PlotScene into a self-contained, XML-valid standalone SVG string.
func RenderPlotSVG(scene *PlotScene) string {
	if scene == nil {
		return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 500"></svg>`
	}

	const (
		width  = 800.0
		height = 500.0
		padL   = 70.0
		padR   = 40.0
		padT   = 40.0
		padB   = 55.0
	)
	plotW := width - padL - padR
	plotH := height - padT - padB

	xSpan := scene.XMax - scene.XMin
	if math.IsNaN(xSpan) || math.IsInf(xSpan, 0) || xSpan <= 0 {
		xSpan = 1
	}
	ySpan := scene.YMax - scene.YMin
	if math.IsNaN(ySpan) || math.IsInf(ySpan, 0) || ySpan <= 0 {
		ySpan = 1
	}

	toSVGX := func(x float64) float64 {
		return padL + (x-scene.XMin)/xSpan*plotW
	}
	toSVGY := func(y float64) float64 {
		return padT + (scene.YMax-y)/ySpan*plotH
	}

	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 500" width="800" height="500">`)
	sb.WriteByte('\n')

	exprText := ""
	if scene.Expr != nil {
		exprText = Format(scene.Expr)
	}
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(i18n.T("plot.svg_title", exprText))))

	var descParts []string
	for _, feat := range scene.Features {
		descParts = append(descParts, translatePlotFeatureKind(feat.Kind)+":"+feat.LabelStr)
	}
	sb.WriteString(fmt.Sprintf("  <desc>%s</desc>\n",
		html.EscapeString(i18n.T("plot.svg_desc",
			Format(scene.XMinNode),
			Format(scene.XMaxNode),
			Format(scene.YMinNode),
			Format(scene.YMaxNode),
			strings.Join(descParts, "; "),
		)),
	))

	sb.WriteString("  <defs>\n")
	sb.WriteString(fmt.Sprintf(`    <clipPath id="plot-area"><rect x="%s" y="%s" width="%s" height="%s"/></clipPath>`,
		fmtSVGNum(padL), fmtSVGNum(padT), fmtSVGNum(plotW), fmtSVGNum(plotH)))
	sb.WriteByte('\n')
	sb.WriteString("  </defs>\n")

	// Background and frame
	sb.WriteString(`  <rect x="0" y="0" width="800" height="500" fill="#ffffff"/>`)
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf(`  <rect x="%s" y="%s" width="%s" height="%s" fill="#f8fafc" stroke="#cbd5e1" stroke-width="1"/>`,
		fmtSVGNum(padL), fmtSVGNum(padT), fmtSVGNum(plotW), fmtSVGNum(plotH)))
	sb.WriteByte('\n')

	// Coordinate axes
	if scene.YMin <= 0 && scene.YMax >= 0 {
		ay := toSVGY(0)
		sb.WriteString(fmt.Sprintf(`  <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#64748b" stroke-width="1.2"/>`,
			fmtSVGNum(padL), fmtSVGNum(ay), fmtSVGNum(padL+plotW), fmtSVGNum(ay)))
		sb.WriteByte('\n')
	}
	if scene.XMin <= 0 && scene.XMax >= 0 {
		ax := toSVGX(0)
		sb.WriteString(fmt.Sprintf(`  <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#64748b" stroke-width="1.2"/>`,
			fmtSVGNum(ax), fmtSVGNum(padT), fmtSVGNum(ax), fmtSVGNum(padT+plotH)))
		sb.WriteByte('\n')
	}

	// Axis domain tick labels
	sb.WriteString(fmt.Sprintf(`  <text x="%s" y="%s" font-family="sans-serif" font-size="12" fill="#334155" text-anchor="middle">%s</text>`,
		fmtSVGNum(padL), fmtSVGNum(padT+plotH+20), html.EscapeString(Format(scene.XMinNode))))
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf(`  <text x="%s" y="%s" font-family="sans-serif" font-size="12" fill="#334155" text-anchor="middle">%s</text>`,
		fmtSVGNum(padL+plotW), fmtSVGNum(padT+plotH+20), html.EscapeString(Format(scene.XMaxNode))))
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf(`  <text x="%s" y="%s" font-family="sans-serif" font-size="12" fill="#334155" text-anchor="end">%s</text>`,
		fmtSVGNum(padL-8), fmtSVGNum(padT+12), html.EscapeString(Format(scene.YMaxNode))))
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf(`  <text x="%s" y="%s" font-family="sans-serif" font-size="12" fill="#334155" text-anchor="end">%s</text>`,
		fmtSVGNum(padL-8), fmtSVGNum(padT+plotH), html.EscapeString(Format(scene.YMinNode))))
	sb.WriteByte('\n')

	// Clipped plot group
	sb.WriteString(`  <g clip-path="url(#plot-area)">`)
	sb.WriteByte('\n')

	// Asymptotes (dashed)
	for _, asym := range scene.Asymptotes {
		switch asym.Kind {
		case "vertical":
			sx := toSVGX(asym.XVal)
			sb.WriteString(fmt.Sprintf(`    <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#94a3b8" stroke-width="1.2" stroke-dasharray="6,4"/>`,
				fmtSVGNum(sx), fmtSVGNum(padT), fmtSVGNum(sx), fmtSVGNum(padT+plotH)))
			sb.WriteByte('\n')
		case "horizontal":
			sy := toSVGY(asym.YVal)
			sb.WriteString(fmt.Sprintf(`    <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#94a3b8" stroke-width="1.2" stroke-dasharray="6,4"/>`,
				fmtSVGNum(padL), fmtSVGNum(sy), fmtSVGNum(padL+plotW), fmtSVGNum(sy)))
			sb.WriteByte('\n')
		case "oblique":
			yStart := asym.Slope*scene.XMin + asym.Intercept
			yEnd := asym.Slope*scene.XMax + asym.Intercept
			cx0, cy0, cx1, cy1, ok := clipSegmentLiangBarsky(scene.XMin, yStart, scene.XMax, yEnd, scene.XMin, scene.XMax, scene.YMin, scene.YMax)
			if ok {
				sb.WriteString(fmt.Sprintf(`    <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#94a3b8" stroke-width="1.2" stroke-dasharray="6,4"/>`,
					fmtSVGNum(toSVGX(cx0)), fmtSVGNum(toSVGY(cy0)), fmtSVGNum(toSVGX(cx1)), fmtSVGNum(toSVGY(cy1))))
				sb.WriteByte('\n')
			}
		}
	}

	// Continuous curve subpaths
	var pathBuf strings.Builder
	for _, br := range scene.Branches {
		if len(br.Points) < 2 {
			continue
		}
		for i, pt := range br.Points {
			cmd := "L"
			if i == 0 {
				cmd = "M"
			}
			if pathBuf.Len() > 0 {
				pathBuf.WriteByte(' ')
			}
			pathBuf.WriteString(fmt.Sprintf("%s %s %s", cmd, fmtSVGNum(toSVGX(pt.X)), fmtSVGNum(toSVGY(pt.Y))))
		}
	}
	if pathBuf.Len() > 0 {
		sb.WriteString(fmt.Sprintf(`    <path d="%s" fill="none" stroke="#0055cc" stroke-width="2" stroke-linejoin="round" stroke-linecap="round"/>`, pathBuf.String()))
		sb.WriteByte('\n')
	}

	// Exact CAS feature markers & labels inside viewport
	for _, feat := range scene.Features {
		if feat.Kind == "asymptote" || feat.Kind == "horiz_asymptote" || feat.Kind == "oblique_asymptote" {
			continue
		}
		if math.IsNaN(feat.XVal) || math.IsInf(feat.XVal, 0) || math.IsNaN(feat.YVal) || math.IsInf(feat.YVal, 0) {
			continue
		}
		if feat.XVal < scene.XMin || feat.XVal > scene.XMax || feat.YVal < scene.YMin || feat.YVal > scene.YMax {
			continue
		}
		cx := toSVGX(feat.XVal)
		cy := toSVGY(feat.YVal)

		switch feat.Kind {
		case "hole":
			sb.WriteString(fmt.Sprintf(`    <circle cx="%s" cy="%s" r="4.5" fill="#ffffff" stroke="#0055cc" stroke-width="2"/>`,
				fmtSVGNum(cx), fmtSVGNum(cy)))
		case "root":
			sb.WriteString(fmt.Sprintf(`    <circle cx="%s" cy="%s" r="4" fill="#d97706" stroke="#ffffff" stroke-width="1"/>`,
				fmtSVGNum(cx), fmtSVGNum(cy)))
		default:
			sb.WriteString(fmt.Sprintf(`    <circle cx="%s" cy="%s" r="4" fill="#059669" stroke="#ffffff" stroke-width="1"/>`,
				fmtSVGNum(cx), fmtSVGNum(cy)))
		}
		sb.WriteByte('\n')

		dx := 8.0
		dy := -8.0
		anchor := "start"
		if cx > padL+plotW*0.75 {
			dx = -8.0
			anchor = "end"
		}
		if cy < padT+24 {
			dy = 16.0
		}
		sb.WriteString(fmt.Sprintf(`    <text x="%s" y="%s" font-family="sans-serif" font-size="11" fill="#0f172a" text-anchor="%s">%s</text>`,
			fmtSVGNum(cx+dx), fmtSVGNum(cy+dy), anchor, html.EscapeString(feat.LabelStr)))
		sb.WriteByte('\n')
	}

	sb.WriteString("  </g>\n")
	sb.WriteString("</svg>\n")
	return sb.String()
}

func translatePlotFeatureKind(kind string) string {
	switch kind {
	case "root":
		return i18n.T("plot.feature_root")
	case "min":
		return i18n.T("plot.feature_min")
	case "max":
		return i18n.T("plot.feature_max")
	case "hole":
		return i18n.T("plot.feature_hole")
	case "asymptote":
		return i18n.T("plot.feature_asymptote")
	case "horiz_asymptote":
		return i18n.T("plot.feature_horiz_asymptote")
	case "oblique_asymptote":
		return i18n.T("plot.feature_oblique_asymptote")
	default:
		return kind
	}
}

func fmtTypstNum(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	if math.Abs(v) < 1e-12 {
		return "0"
	}
	s := strconv.FormatFloat(v, 'g', 6, 64)
	if s == "-0" || s == "" {
		return "0"
	}
	return s
}

// RenderPlotTypst renders a PlotScene into compilable Typst (cetz-plot) vector graphics markup.
func RenderPlotTypst(scene *PlotScene) string {
	if scene == nil {
		return ""
	}
	xMin, xMax := scene.XMin, scene.XMax
	yMin, yMax := scene.YMin, scene.YMax
	if math.IsNaN(xMin) || math.IsInf(xMin, 0) || math.IsNaN(xMax) || math.IsInf(xMax, 0) || xMax <= xMin {
		xMin, xMax = -10, 10
	}
	if math.IsNaN(yMin) || math.IsInf(yMin, 0) || math.IsNaN(yMax) || math.IsInf(yMax, 0) || yMax <= yMin {
		yMin, yMax = -10, 10
	}
	varName := scene.VarName
	if strings.TrimSpace(varName) == "" {
		varName = "x"
	}

	var sb strings.Builder
	sb.WriteString(`#import "@preview/cetz:0.5.2"`)
	sb.WriteByte('\n')
	sb.WriteString(`#import "@preview/cetz-plot:0.1.4": plot`)
	sb.WriteByte('\n')
	sb.WriteByte('\n')
	sb.WriteString("#cetz.canvas({\n")
	sb.WriteString("  plot.plot(\n")
	sb.WriteString("    size: (10, 6),\n")
	sb.WriteString(fmt.Sprintf("    x-min: %s, x-max: %s,\n", fmtTypstNum(xMin), fmtTypstNum(xMax)))
	sb.WriteString(fmt.Sprintf("    y-min: %s, y-max: %s,\n", fmtTypstNum(yMin), fmtTypstNum(yMax)))
	sb.WriteString(fmt.Sprintf("    x-label: [$%s$],\n", varName))
	sb.WriteString("    y-label: [$y$],\n")
	sb.WriteString("    {\n")

	// Continuous branches as pre-sampled point arrays
	firstBranch := true
	exprLabel := "y"
	if scene.Expr != nil {
		exprLabel = formatTypstNode(scene.Expr)
	}
	for _, br := range scene.Branches {
		if len(br.Points) < 2 {
			continue
		}
		var pts []string
		for _, pt := range br.Points {
			pts = append(pts, fmt.Sprintf("(%s, %s)", fmtTypstNum(pt.X), fmtTypstNum(pt.Y)))
		}
		if firstBranch {
			sb.WriteString(fmt.Sprintf("      plot.add((%s), label: [$%s$], style: (stroke: (paint: blue, thickness: 1.5pt)))\n",
				strings.Join(pts, ", "), exprLabel))
			firstBranch = false
		} else {
			sb.WriteString(fmt.Sprintf("      plot.add((%s), style: (stroke: (paint: blue, thickness: 1.5pt)))\n",
				strings.Join(pts, ", ")))
		}
	}

	// Asymptotes
	for _, asym := range scene.Asymptotes {
		switch asym.Kind {
		case "vertical":
			if !math.IsNaN(asym.XVal) && !math.IsInf(asym.XVal, 0) && asym.XVal >= xMin && asym.XVal <= xMax {
				sb.WriteString(fmt.Sprintf("      plot.add-vline(%s, style: (stroke: (dash: \"dashed\", paint: gray)))\n", fmtTypstNum(asym.XVal)))
			}
		case "horizontal":
			if !math.IsNaN(asym.YVal) && !math.IsInf(asym.YVal, 0) && asym.YVal >= yMin && asym.YVal <= yMax {
				sb.WriteString(fmt.Sprintf("      plot.add-hline(%s, style: (stroke: (dash: \"dashed\", paint: gray)))\n", fmtTypstNum(asym.YVal)))
			}
		case "oblique":
			y0 := asym.Slope*xMin + asym.Intercept
			y1 := asym.Slope*xMax + asym.Intercept
			cx0, _, cx1, _, vis := clipSegmentLiangBarsky(xMin, y0, xMax, y1, xMin, xMax, yMin, yMax)
			if vis && cx1 > cx0 {
				sb.WriteString(fmt.Sprintf("      plot.add(domain: (%s, %s), x => %s * x + (%s), style: (stroke: (dash: \"dashed\", paint: gray)))\n",
					fmtTypstNum(cx0), fmtTypstNum(cx1), fmtTypstNum(asym.Slope), fmtTypstNum(asym.Intercept)))
			}
		}
	}

	// Exact symbolic annotations inside plot coordinate system
	var annots []string
	for _, feat := range scene.Features {
		if feat.Kind == "asymptote" || feat.Kind == "horiz_asymptote" || feat.Kind == "oblique_asymptote" {
			continue
		}
		if math.IsNaN(feat.XVal) || math.IsInf(feat.XVal, 0) || math.IsNaN(feat.YVal) || math.IsInf(feat.YVal, 0) {
			continue
		}
		if feat.XVal < xMin || feat.XVal > xMax || feat.YVal < yMin || feat.YVal > yMax {
			continue
		}
		xTex := fmtTypstNum(feat.XVal)
		yTex := fmtTypstNum(feat.YVal)
		if feat.XNode != nil {
			xTex = formatTypstNode(feat.XNode)
		}
		if feat.YNode != nil {
			yTex = formatTypstNode(feat.YNode)
		}
		anchor := "south-west"
		if feat.XVal > scene.XMin+(scene.XMax-scene.XMin)*0.75 {
			anchor = "south-east"
		}
		if feat.Kind == "hole" {
			annots = append(annots, fmt.Sprintf("        circle((%s, %s), radius: 0.08, fill: white, stroke: blue)", fmtTypstNum(feat.XVal), fmtTypstNum(feat.YVal)))
		} else {
			annots = append(annots, fmt.Sprintf("        circle((%s, %s), radius: 0.07, fill: red, stroke: none)", fmtTypstNum(feat.XVal), fmtTypstNum(feat.YVal)))
		}
		annots = append(annots, fmt.Sprintf("        content((%s, %s), [$(%s, %s)$], anchor: \"%s\", padding: 0.1)",
			fmtTypstNum(feat.XVal), fmtTypstNum(feat.YVal), xTex, yTex, anchor))
	}
	if len(annots) > 0 {
		sb.WriteString("      plot.annotate({\n")
		sb.WriteString("        import cetz.draw: *\n")
		for _, a := range annots {
			sb.WriteString(a)
			sb.WriteByte('\n')
		}
		sb.WriteString("      })\n")
	}

	sb.WriteString("    }\n")
	sb.WriteString("  )\n")
	sb.WriteString("})\n")
	return sb.String()
}

// EvalPlotSVG evaluates plot_svg(expr, x_range, [y_range], [output_file]) and returns standalone SVG or file confirmation.
func EvalPlotSVG(args []Node) (Node, error) {
	if len(args) < 2 || len(args) > 4 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot_svg_args_count", len(args)))
	}

	var outPath string
	plotArgs := args
	if strNode, ok := args[len(args)-1].(*StringNode); ok && len(args) >= 3 {
		if strings.TrimSpace(strNode.Value) == "" {
			return nil, fmt.Errorf("%s", i18n.T("errors.plot_svg_empty_path"))
		}
		outPath = strNode.Value
		plotArgs = args[:len(args)-1]
	} else if len(args) == 4 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot_svg_invalid_path", args[3].String()))
	}

	scene, err := BuildPlotScene(plotArgs)
	if err != nil {
		return nil, err
	}
	svgStr := RenderPlotSVG(scene)

	if outPath != "" {
		dir := filepath.Dir(outPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("%s", i18n.T("errors.plot_svg_write_failed", outPath, err))
			}
		}
		// Write UTF-8 without BOM
		if err := os.WriteFile(outPath, []byte(svgStr), 0644); err != nil {
			return nil, fmt.Errorf("%s", i18n.T("errors.plot_svg_write_failed", outPath, err))
		}
		return NewStringNode(i18n.T("plot.svg_saved", outPath)), nil
	}
	return NewStringNode(svgStr), nil
}
