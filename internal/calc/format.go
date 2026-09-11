package calc

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// FormatOptions controls the string representation of expressions.
type FormatOptions struct {
	AsciiOnly bool
}

// Format formats a node using default options (Unicode symbols).
func Format(n Node) string {
	return FormatWithOptions(n, FormatOptions{AsciiOnly: false})
}

// FormatWithOptions formats a node using the provided options.
func FormatWithOptions(n Node, opts FormatOptions) string {
	if n == nil {
		return ""
	}
	return formatNode(n, opts)
}

func formatNode(n Node, opts FormatOptions) string {
	switch v := n.(type) {
	case *RationalNode:
		if v.Val.IsInt() {
			return v.Val.Num().String()
		}
		return v.Val.String()

	case *ConstNode:
		if v.Name == "pi" {
			if opts.AsciiOnly {
				return "pi"
			}
			return "π"
		}
		return v.Name

	case *VarNode:
		return v.Name

	case *SqrtNode:
		radStr := formatNode(v.Radicand, opts)
		if opts.AsciiOnly {
			return fmt.Sprintf("sqrt(%s)", radStr)
		}
		// If radicand is pure integer, omit parentheses: √2
		if rat, ok := v.Radicand.(*RationalNode); ok && rat.Val.IsInt() {
			return fmt.Sprintf("√%s", radStr)
		}
		return fmt.Sprintf("√(%s)", radStr)

	case *FuncNode:
		if v.Name == "cbrt" && len(v.Args) == 1 {
			radStr := formatNode(v.Args[0], opts)
			if opts.AsciiOnly {
				return fmt.Sprintf("cbrt(%s)", radStr)
			}
			if rat, ok := v.Args[0].(*RationalNode); ok && rat.Val.IsInt() {
				return fmt.Sprintf("³√%s", radStr)
			}
			return fmt.Sprintf("³√(%s)", radStr)
		}
		if v.Name == "abs" && len(v.Args) == 1 {
			argStr := formatNode(v.Args[0], opts)
			return fmt.Sprintf("|%s|", argStr)
		}
		argStrs := make([]string, len(v.Args))
		for i, a := range v.Args {
			argStrs[i] = formatNode(a, opts)
		}
		return fmt.Sprintf("%s(%s)", v.Name, strings.Join(argStrs, ", "))

	case *ComplexNode:
		realIsZero := isZero(v.Real)
		imagIsZero := isZero(v.Imag)

		if realIsZero && imagIsZero {
			return "0"
		}
		if imagIsZero {
			return formatNode(v.Real, opts)
		}
		if realIsZero {
			return formatImagPart(v.Imag, opts)
		}

		// Both non-zero: Real +/- Imag*i
		realStr := formatNode(v.Real, opts)
		if isNegative(v.Imag) {
			posImag, _ := simplifyUnaryOp("-", v.Imag)
			imagStr := formatImagPart(posImag, opts)
			return fmt.Sprintf("%s - %s", realStr, imagStr)
		}
		imagStr := formatImagPart(v.Imag, opts)
		return fmt.Sprintf("%s + %s", realStr, imagStr)

	case *PowNode:
		baseStr := formatNode(v.Base, opts)
		expStr := formatNode(v.Exp, opts)
		switch v.Base.(type) {
		case *AddNode, *MulNode, *UnaryOpNode, *ComplexNode:
			baseStr = fmt.Sprintf("(%s)", baseStr)
		}
		return fmt.Sprintf("%s^%s", baseStr, expStr)

	case *UnaryOpNode:
		if v.Op == "!" {
			exprStr := formatNode(v.Expr, opts)
			if _, isOp := v.Expr.(*AddNode); isOp {
				exprStr = fmt.Sprintf("(%s)", exprStr)
			}
			return fmt.Sprintf("%s!", exprStr)
		}
		exprStr := formatNode(v.Expr, opts)
		if _, isOp := v.Expr.(*AddNode); isOp {
			exprStr = fmt.Sprintf("-(%s)", exprStr)
		} else {
			exprStr = fmt.Sprintf("-%s", exprStr)
		}
		return exprStr

	case *MulNode:
		return formatMul(v, opts)

	case *AddNode:
		return formatAdd(v, opts)

	default:
		return n.String()
	}
}

func formatImagPart(imag Node, opts FormatOptions) string {
	one := big.NewRat(1, 1)
	negOne := big.NewRat(-1, 1)

	if rat, ok := imag.(*RationalNode); ok {
		if rat.Val.Cmp(one) == 0 {
			return "i"
		}
		if rat.Val.Cmp(negOne) == 0 {
			return "-i"
		}
		return fmt.Sprintf("%s*i", rat.String())
	}
	return fmt.Sprintf("%s*i", formatNode(imag, opts))
}

func isNegative(n Node) bool {
	if rat, ok := n.(*RationalNode); ok {
		return rat.Val.Sign() < 0
	}
	if uop, ok := n.(*UnaryOpNode); ok && uop.Op == "-" {
		return true
	}
	if mul, ok := n.(*MulNode); ok && len(mul.Factors) > 0 {
		if rat, ok := mul.Factors[0].(*RationalNode); ok {
			return rat.Val.Sign() < 0
		}
	}
	if c, ok := n.(*ComplexNode); ok {
		if isZero(c.Real) && isNegative(c.Imag) {
			return true
		}
	}
	return false
}

func formatMul(m *MulNode, opts FormatOptions) string {
	if len(m.Factors) == 2 {
		if r, ok := m.Factors[0].(*RationalNode); ok {
			// Check for 1/d * Base -> Base/d
			if !r.Val.IsInt() && r.Val.Num().Cmp(big.NewInt(1)) == 0 {
				baseStr := formatNode(m.Factors[1], opts)
				return fmt.Sprintf("%s/%s", baseStr, r.Val.Denom().String())
			}
			// Check for -1/d * Base -> -Base/d
			if !r.Val.IsInt() && r.Val.Num().Cmp(big.NewInt(-1)) == 0 {
				baseStr := formatNode(m.Factors[1], opts)
				return fmt.Sprintf("-%s/%s", baseStr, r.Val.Denom().String())
			}
			// Check for 1 * Base -> Base
			one := big.NewRat(1, 1)
			if r.Val.Cmp(one) == 0 {
				return formatNode(m.Factors[1], opts)
			}
			// Check for -1 * Base -> -Base
			negOne := big.NewRat(-1, 1)
			if r.Val.Cmp(negOne) == 0 {
				return fmt.Sprintf("-%s", formatNode(m.Factors[1], opts))
			}
		}
	}

	strs := make([]string, len(m.Factors))
	for i, f := range m.Factors {
		fStr := formatNode(f, opts)
		if _, isAdd := f.(*AddNode); isAdd {
			fStr = fmt.Sprintf("(%s)", fStr)
		}
		strs[i] = fStr
	}
	return strings.Join(strs, "*")
}

func formatAdd(a *AddNode, opts FormatOptions) string {
	sortedTerms := sortTerms(a.Terms)

	var sb strings.Builder
	for i, t := range sortedTerms {
		if i == 0 {
			sb.WriteString(formatNode(t, opts))
		} else {
			if isNegative(t) {
				// term is negative -> " - positiveTerm"
				posTerm, _ := simplifyUnaryOp("-", t)
				sb.WriteString(" - ")
				sb.WriteString(formatNode(posTerm, opts))
			} else {
				sb.WriteString(" + ")
				sb.WriteString(formatNode(t, opts))
			}
		}
	}
	return sb.String()
}

// termCategory maps a node to an order index:
// 1: Rational
// 2: Sqrt
// 3: Const (pi, e)
// 4: Func (sin, cos, log)
// 5: Complex
// 6: Other
func termCategory(n Node) int {
	switch v := n.(type) {
	case *RationalNode:
		return 1
	case *VarNode:
		return 2
	case *SqrtNode:
		return 3
	case *ConstNode:
		return 4
	case *FuncNode:
		return 5
	case *ComplexNode:
		return 6
	case *MulNode:
		if len(v.Factors) >= 2 {
			return termCategory(v.Factors[len(v.Factors)-1])
		}
		return 6
	default:
		return 6
	}
}

func sortTerms(terms []Node) []Node {
	copied := make([]Node, len(terms))
	copy(copied, terms)

	sort.SliceStable(copied, func(i, j int) bool {
		catI := termCategory(copied[i])
		catJ := termCategory(copied[j])
		if catI != catJ {
			return catI < catJ
		}
		// Tie-break: sort by String representation
		return copied[i].String() < copied[j].String()
	})

	return copied
}

// FormatLaTeX formats an AST node as LaTeX math text wrapped in $$ ... $$.
func FormatLaTeX(n Node) string {
	if n == nil {
		return "$$  $$"
	}
	return fmt.Sprintf("$$ %s $$", formatLaTeXNode(n))
}

func formatLaTeXNode(n Node) string {
	switch v := n.(type) {
	case *RationalNode:
		if v.Val.IsInt() {
			return v.Val.Num().String()
		}
		num := new(big.Int).Abs(v.Val.Num())
		denom := v.Val.Denom()
		if v.Val.Sign() < 0 {
			return fmt.Sprintf("-\\frac{%s}{%s}", num.String(), denom.String())
		}
		return fmt.Sprintf("\\frac{%s}{%s}", num.String(), denom.String())

	case *ConstNode:
		if v.Name == "pi" {
			return "\\pi"
		}
		return v.Name

	case *VarNode:
		return v.Name

	case *SqrtNode:
		return fmt.Sprintf("\\sqrt{%s}", formatLaTeXNode(v.Radicand))

	case *FuncNode:
		if v.Name == "cbrt" && len(v.Args) == 1 {
			return fmt.Sprintf("\\sqrt[3]{%s}", formatLaTeXNode(v.Args[0]))
		}
		if v.Name == "abs" && len(v.Args) == 1 {
			return fmt.Sprintf("\\left|%s\\right|", formatLaTeXNode(v.Args[0]))
		}
		fnName := v.Name
		switch fnName {
		case "sin", "cos", "tan", "log", "ln":
			fnName = "\\" + fnName
		case "asin":
			fnName = "\\arcsin"
		case "acos":
			fnName = "\\arccos"
		case "atan":
			fnName = "\\arctan"
		}
		if len(v.Args) == 1 {
			return fmt.Sprintf("%s(%s)", fnName, formatLaTeXNode(v.Args[0]))
		}
		if len(v.Args) == 2 && v.Name == "log" {
			// log_b(x)
			return fmt.Sprintf("\\log_{%s}(%s)", formatLaTeXNode(v.Args[0]), formatLaTeXNode(v.Args[1]))
		}
		argStrs := make([]string, len(v.Args))
		for i, a := range v.Args {
			argStrs[i] = formatLaTeXNode(a)
		}
		return fmt.Sprintf("%s(%s)", fnName, strings.Join(argStrs, ", "))

	case *ComplexNode:
		realIsZero := isZero(v.Real)
		imagIsZero := isZero(v.Imag)

		if realIsZero && imagIsZero {
			return "0"
		}
		if imagIsZero {
			return formatLaTeXNode(v.Real)
		}
		if realIsZero {
			return formatLaTeXImagPart(v.Imag)
		}

		realStr := formatLaTeXNode(v.Real)
		if isNegative(v.Imag) {
			posImag, _ := simplifyUnaryOp("-", v.Imag)
			return fmt.Sprintf("%s - %s", realStr, formatLaTeXImagPart(posImag))
		}
		return fmt.Sprintf("%s + %s", realStr, formatLaTeXImagPart(v.Imag))

	case *PowNode:
		baseStr := formatLaTeXNode(v.Base)
		expStr := formatLaTeXNode(v.Exp)
		switch v.Base.(type) {
		case *AddNode, *MulNode, *UnaryOpNode, *ComplexNode:
			baseStr = fmt.Sprintf("(%s)", baseStr)
		}
		return fmt.Sprintf("%s^{%s}", baseStr, expStr)

	case *UnaryOpNode:
		if v.Op == "!" {
			exprStr := formatLaTeXNode(v.Expr)
			if _, isOp := v.Expr.(*AddNode); isOp {
				exprStr = fmt.Sprintf("(%s)", exprStr)
			}
			return fmt.Sprintf("%s!", exprStr)
		}
		exprStr := formatLaTeXNode(v.Expr)
		if _, isOp := v.Expr.(*AddNode); isOp {
			exprStr = fmt.Sprintf("-(%s)", exprStr)
		} else {
			exprStr = fmt.Sprintf("-%s", exprStr)
		}
		return exprStr

	case *MulNode:
		return formatLaTeXMul(v)

	case *AddNode:
		return formatLaTeXAdd(v)

	default:
		return n.String()
	}
}

func formatLaTeXImagPart(imag Node) string {
	one := big.NewRat(1, 1)
	negOne := big.NewRat(-1, 1)

	if rat, ok := imag.(*RationalNode); ok {
		if rat.Val.Cmp(one) == 0 {
			return "i"
		}
		if rat.Val.Cmp(negOne) == 0 {
			return "-i"
		}
		return fmt.Sprintf("%si", formatLaTeXNode(rat))
	}
	return fmt.Sprintf("%si", formatLaTeXNode(imag))
}

func formatLaTeXMul(m *MulNode) string {
	if len(m.Factors) == 2 {
		if r, ok := m.Factors[0].(*RationalNode); ok {
			// Check for 1/d * Base -> \frac{Base}{d}
			if !r.Val.IsInt() && r.Val.Num().Cmp(big.NewInt(1)) == 0 {
				baseStr := formatLaTeXNode(m.Factors[1])
				return fmt.Sprintf("\\frac{%s}{%s}", baseStr, r.Val.Denom().String())
			}
			// Check for -1/d * Base -> -\frac{Base}{d}
			if !r.Val.IsInt() && r.Val.Num().Cmp(big.NewInt(-1)) == 0 {
				baseStr := formatLaTeXNode(m.Factors[1])
				return fmt.Sprintf("-\\frac{%s}{%s}", baseStr, r.Val.Denom().String())
			}
			// Check for 1 * Base -> Base
			one := big.NewRat(1, 1)
			if r.Val.Cmp(one) == 0 {
				return formatLaTeXNode(m.Factors[1])
			}
			// Check for -1 * Base -> -Base
			negOne := big.NewRat(-1, 1)
			if r.Val.Cmp(negOne) == 0 {
				return fmt.Sprintf("-%s", formatLaTeXNode(m.Factors[1]))
			}
			// Integer coeff * Sqrt/Const/Func/Symbol -> 3\sqrt{2}
			if r.Val.IsInt() {
				coeffStr := r.Val.Num().String()
				baseStr := formatLaTeXNode(m.Factors[1])
				return fmt.Sprintf("%s%s", coeffStr, baseStr)
			}
		}
	}

	strs := make([]string, len(m.Factors))
	for i, f := range m.Factors {
		fStr := formatLaTeXNode(f)
		if _, isAdd := f.(*AddNode); isAdd {
			fStr = fmt.Sprintf("(%s)", fStr)
		}
		strs[i] = fStr
	}
	return strings.Join(strs, " \\cdot ")
}

func formatLaTeXAdd(a *AddNode) string {
	sortedTerms := sortTerms(a.Terms)

	var sb strings.Builder
	for i, t := range sortedTerms {
		if i == 0 {
			sb.WriteString(formatLaTeXNode(t))
		} else {
			if isNegative(t) {
				posTerm, _ := simplifyUnaryOp("-", t)
				sb.WriteString(" - ")
				sb.WriteString(formatLaTeXNode(posTerm))
			} else {
				sb.WriteString(" + ")
				sb.WriteString(formatLaTeXNode(t))
			}
		}
	}
	return sb.String()
}

// Box represents a 2D text box with height, width, and a baseline row index (0-indexed).
type Box struct {
	Lines    []string
	Width    int
	Height   int
	Baseline int
}

func newTextBox(text string) Box {
	lines := strings.Split(text, "\n")
	maxWidth := 0
	for _, l := range lines {
		w := len([]rune(l))
		if w > maxWidth {
			maxWidth = w
		}
	}
	return Box{
		Lines:    lines,
		Width:    maxWidth,
		Height:   len(lines),
		Baseline: len(lines) / 2,
	}
}

func hConcat(boxes ...Box) Box {
	if len(boxes) == 0 {
		return Box{}
	}
	maxAbove := 0
	maxBelow := 0
	for _, b := range boxes {
		above := b.Baseline
		below := b.Height - 1 - b.Baseline
		if above > maxAbove {
			maxAbove = above
		}
		if below > maxBelow {
			maxBelow = below
		}
	}
	totalHeight := maxAbove + 1 + maxBelow
	newBaseline := maxAbove

	resLines := make([]string, totalHeight)
	for row := 0; row < totalHeight; row++ {
		var rowSb strings.Builder
		for _, b := range boxes {
			bRow := row - (newBaseline - b.Baseline)
			if bRow >= 0 && bRow < b.Height {
				line := b.Lines[bRow]
				rowSb.WriteString(line)
				pad := b.Width - len([]rune(line))
				if pad > 0 {
					rowSb.WriteString(strings.Repeat(" ", pad))
				}
			} else {
				rowSb.WriteString(strings.Repeat(" ", b.Width))
			}
		}
		resLines[row] = rowSb.String()
	}

	totalWidth := 0
	for _, b := range boxes {
		totalWidth += b.Width
	}

	return Box{
		Lines:    resLines,
		Width:    totalWidth,
		Height:   totalHeight,
		Baseline: newBaseline,
	}
}

func makeFracBox(numBox, denomBox Box) Box {
	width := numBox.Width
	if denomBox.Width > width {
		width = denomBox.Width
	}
	width += 2 // 1 char padding each side

	padBoxLines := func(b Box, targetWidth int) []string {
		out := make([]string, b.Height)
		for i, l := range b.Lines {
			rCount := len([]rune(l))
			leftPad := (targetWidth - rCount) / 2
			rightPad := targetWidth - rCount - leftPad
			out[i] = strings.Repeat(" ", leftPad) + l + strings.Repeat(" ", rightPad)
		}
		return out
	}

	numLines := padBoxLines(numBox, width)
	denomLines := padBoxLines(denomBox, width)
	barLine := strings.Repeat("-", width)

	lines := make([]string, 0, len(numLines)+1+len(denomLines))
	lines = append(lines, numLines...)
	lines = append(lines, barLine)
	lines = append(lines, denomLines...)

	return Box{
		Lines:    lines,
		Width:    width,
		Height:   len(lines),
		Baseline: len(numLines), // the bar line is baseline
	}
}

// FormatPretty2D formats an AST node into a multi-line 2D pretty-printed string.
func FormatPretty2D(n Node) string {
	if n == nil {
		return ""
	}
	box := nodeToBox(n)
	return strings.Join(box.Lines, "\n")
}

func nodeToBox(n Node) Box {
	switch v := n.(type) {
	case *RationalNode:
		if v.Val.IsInt() {
			return newTextBox(v.Val.Num().String())
		}
		num := new(big.Int).Abs(v.Val.Num())
		denom := v.Val.Denom()
		fracBox := makeFracBox(newTextBox(num.String()), newTextBox(denom.String()))
		if v.Val.Sign() < 0 {
			return hConcat(newTextBox("-"), fracBox)
		}
		return fracBox

	case *ConstNode:
		if v.Name == "pi" {
			return newTextBox("π")
		}
		return newTextBox(v.Name)

	case *VarNode:
		return newTextBox(v.Name)

	case *SqrtNode:
		radBox := nodeToBox(v.Radicand)
		if radBox.Height == 1 {
			return newTextBox("√" + radBox.Lines[0])
		}
		return hConcat(newTextBox("√("), radBox, newTextBox(")"))

	case *FuncNode:
		if len(v.Args) == 1 {
			argBox := nodeToBox(v.Args[0])
			return hConcat(newTextBox(v.Name+"("), argBox, newTextBox(")"))
		}
		boxes := []Box{newTextBox(v.Name + "(")}
		for i, a := range v.Args {
			if i > 0 {
				boxes = append(boxes, newTextBox(", "))
			}
			boxes = append(boxes, nodeToBox(a))
		}
		boxes = append(boxes, newTextBox(")"))
		return hConcat(boxes...)

	case *ComplexNode:
		realIsZero := isZero(v.Real)
		imagIsZero := isZero(v.Imag)

		if realIsZero && imagIsZero {
			return newTextBox("0")
		}
		if imagIsZero {
			return nodeToBox(v.Real)
		}
		if realIsZero {
			return imagToBox(v.Imag)
		}

		realBox := nodeToBox(v.Real)
		if isNegative(v.Imag) {
			posImag, _ := simplifyUnaryOp("-", v.Imag)
			return hConcat(realBox, newTextBox(" - "), imagToBox(posImag))
		}
		return hConcat(realBox, newTextBox(" + "), imagToBox(v.Imag))

	case *PowNode:
		// Negative integer power: X^-1 -> 1 / X
		if rExp, ok := v.Exp.(*RationalNode); ok && rExp.Val.IsInt() && rExp.Val.Sign() < 0 {
			expVal := rExp.Val.Num().Int64()
			if expVal == -1 {
				return makeFracBox(newTextBox("1"), nodeToBox(v.Base))
			}
			posExpNode := mustRational(-expVal, 1)
			denomBox := nodeToBox(&PowNode{Base: v.Base, Exp: posExpNode})
			return makeFracBox(newTextBox("1"), denomBox)
		}

		baseBox := nodeToBox(v.Base)
		expBox := nodeToBox(v.Exp)
		switch v.Base.(type) {
		case *AddNode, *MulNode, *UnaryOpNode, *ComplexNode:
			baseBox = hConcat(newTextBox("("), baseBox, newTextBox(")"))
		}
		return hConcat(baseBox, newTextBox("^"), expBox)

	case *UnaryOpNode:
		if v.Op == "!" {
			exprBox := nodeToBox(v.Expr)
			if _, isOp := v.Expr.(*AddNode); isOp {
				exprBox = hConcat(newTextBox("("), exprBox, newTextBox(")"))
			}
			return hConcat(exprBox, newTextBox("!"))
		}
		exprBox := nodeToBox(v.Expr)
		if _, isOp := v.Expr.(*AddNode); isOp {
			return hConcat(newTextBox("-("), exprBox, newTextBox(")"))
		}
		return hConcat(newTextBox("-"), exprBox)

	case *MulNode:
		return mulToBox(v)

	case *AddNode:
		return addToBox(v)

	default:
		return newTextBox(n.String())
	}
}

func imagToBox(imag Node) Box {
	one := big.NewRat(1, 1)
	negOne := big.NewRat(-1, 1)

	if rat, ok := imag.(*RationalNode); ok {
		if rat.Val.Cmp(one) == 0 {
			return newTextBox("i")
		}
		if rat.Val.Cmp(negOne) == 0 {
			return newTextBox("-i")
		}
		return hConcat(nodeToBox(rat), newTextBox("*i"))
	}
	return hConcat(nodeToBox(imag), newTextBox("*i"))
}

func mulToBox(m *MulNode) Box {
	if len(m.Factors) == 2 {
		if r, ok := m.Factors[0].(*RationalNode); ok {
			// Check for 1/d * Base -> Base / d
			if !r.Val.IsInt() && r.Val.Num().Cmp(big.NewInt(1)) == 0 {
				baseBox := nodeToBox(m.Factors[1])
				denomBox := newTextBox(r.Val.Denom().String())
				return makeFracBox(baseBox, denomBox)
			}
			// Check for -1/d * Base -> - Base / d
			if !r.Val.IsInt() && r.Val.Num().Cmp(big.NewInt(-1)) == 0 {
				baseBox := nodeToBox(m.Factors[1])
				denomBox := newTextBox(r.Val.Denom().String())
				return hConcat(newTextBox("-"), makeFracBox(baseBox, denomBox))
			}
			// Check for 1 * Base -> Base
			one := big.NewRat(1, 1)
			if r.Val.Cmp(one) == 0 {
				return nodeToBox(m.Factors[1])
			}
			// Check for -1 * Base -> -Base
			negOne := big.NewRat(-1, 1)
			if r.Val.Cmp(negOne) == 0 {
				return hConcat(newTextBox("-"), nodeToBox(m.Factors[1]))
			}
		}

		// Check for A * B^-1 -> A / B
		if pow, ok := m.Factors[1].(*PowNode); ok {
			if rExp, isRat := pow.Exp.(*RationalNode); isRat && rExp.Val.IsInt() && rExp.Val.Cmp(big.NewRat(-1, 1)) == 0 {
				numBox := nodeToBox(m.Factors[0])
				denomBox := nodeToBox(pow.Base)
				return makeFracBox(numBox, denomBox)
			}
		}
	}

	boxes := make([]Box, 0, len(m.Factors)*2)
	for i, f := range m.Factors {
		fBox := nodeToBox(f)
		if _, isAdd := f.(*AddNode); isAdd {
			fBox = hConcat(newTextBox("("), fBox, newTextBox(")"))
		}
		if i > 0 {
			boxes = append(boxes, newTextBox("*"))
		}
		boxes = append(boxes, fBox)
	}
	return hConcat(boxes...)
}

func addToBox(a *AddNode) Box {
	sortedTerms := sortTerms(a.Terms)

	boxes := make([]Box, 0, len(sortedTerms)*2)
	for i, t := range sortedTerms {
		if i == 0 {
			boxes = append(boxes, nodeToBox(t))
		} else {
			if isNegative(t) {
				posTerm, _ := simplifyUnaryOp("-", t)
				boxes = append(boxes, newTextBox(" - "))
				boxes = append(boxes, nodeToBox(posTerm))
			} else {
				boxes = append(boxes, newTextBox(" + "))
				boxes = append(boxes, nodeToBox(t))
			}
		}
	}
	return hConcat(boxes...)
}


