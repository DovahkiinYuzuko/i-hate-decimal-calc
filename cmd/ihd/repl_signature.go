package main

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/nyaosorg/go-readline-ny/moji"
)

// SignatureContext holds parsed signature metadata for the currently focused function call.
type SignatureContext struct {
	FuncName      string
	ArgIndex      int
	ActiveArgName string
	Args          []string
	MinArgs       int
	MaxArgs       int
	Description   string
}

type openCall struct {
	funcName string
	argIndex int
	kind     rune // '(' or '['
}

// ParseSignatureContext scans the input line up to the cursor position using a lexical FSM
// to identify the active function call and the current argument index.
func ParseSignatureContext(line string, cursor int) (*SignatureContext, bool) {
	if line == "" || cursor <= 0 {
		return nil, false
	}

	runes := []rune(line)
	if cursor > len(runes) {
		cursor = len(runes)
	}

	var stack []openCall
	inString := false
	escape := false

	var lastIdent string
	i := 0

	for i < cursor {
		r := runes[i]

		// Handle string literals
		if inString {
			if escape {
				escape = false
			} else if r == '\\' {
				escape = true
			} else if r == '"' {
				inString = false
			}
			i++
			continue
		}

		if r == '"' {
			inString = true
			escape = false
			lastIdent = ""
			i++
			continue
		}

		// Handle brackets
		if r == '(' || r == '[' {
			callName := ""
			if r == '(' && lastIdent != "" {
				lower := strings.ToLower(lastIdent)
				if calc.IsReservedFunc(lower) {
					callName = lower
				}
			}
			stack = append(stack, openCall{
				funcName: callName,
				argIndex: 0,
				kind:     r,
			})
			lastIdent = ""
			i++
			continue
		}

		if r == ')' || r == ']' {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			lastIdent = ""
			i++
			continue
		}

		// Handle argument delimiter
		if r == ',' {
			if len(stack) > 0 {
				stack[len(stack)-1].argIndex++
			}
			lastIdent = ""
			i++
			continue
		}

		// Handle identifier tokens
		if unicode.IsLetter(r) || r == '_' {
			start := i
			for i < cursor && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			lastIdent = string(runes[start:i])
			continue
		}

		// Whitespace
		if unicode.IsSpace(r) {
			i++
			continue
		}

		// Other characters (operators, digits, etc.)
		lastIdent = ""
		i++
	}

	// Find the innermost active function call in the stack
	for j := len(stack) - 1; j >= 0; j-- {
		call := stack[j]
		if call.kind == '(' && call.funcName != "" {
			sig, ok := calc.LookupSignature(call.funcName)
			if !ok {
				continue
			}

			activeArg := ""
			if call.argIndex < len(sig.Args) {
				activeArg = sig.Args[call.argIndex]
			} else if sig.MaxArgs == -1 || call.argIndex <= sig.MaxArgs {
				activeArg = fmt.Sprintf("arg%d", call.argIndex+1)
			}

			// Retrieve localized description
			desc := i18n.T(sig.DescKey)
			if desc == sig.DescKey || strings.HasPrefix(desc, "fn.") {
				desc = fmt.Sprintf("Function %s", sig.Name)
			}

			return &SignatureContext{
				FuncName:      sig.Name,
				ArgIndex:      call.argIndex,
				ActiveArgName: activeArg,
				Args:          sig.Args,
				MinArgs:       sig.MinArgs,
				MaxArgs:       sig.MaxArgs,
				Description:   desc,
			}, true
		}
	}

	return nil, false
}

// RenderSignatureInline prints a single-line dimmed signature hint directly on the current row
// within the remaining availWidth column budget, strictly avoiding newlines or vertical cursor moves.
func RenderSignatureInline(w io.Writer, ctx *SignatureContext, theme string, availWidth int) error {
	if ctx == nil || availWidth <= 6 {
		return nil
	}

	// ANSI color palette (strictly SGR styling only, no cursor movement)
	prefixColor := "\x1B[90m"     // Dim Gray
	nameColor := "\x1B[1;36m"     // Bold Cyan
	activeColor := "\x1B[1;4;33m" // Bold Underline Yellow
	normalColor := "\x1B[37m"     // White
	dimColor := "\x1B[90m"        // Dim Gray
	descColor := "\x1B[32m"       // Green
	resetColor := "\x1B[0m"

	switch theme {
	case "light":
		prefixColor = "\x1B[90m"
		nameColor = "\x1B[1;34m"     // Bold Blue
		activeColor = "\x1B[1;4;35m" // Bold Underline Magenta
		normalColor = "\x1B[30m"     // Black
		dimColor = "\x1B[90m"
		descColor = "\x1B[32m"
	case "none":
		prefixColor = ""
		nameColor = ""
		activeColor = ""
		normalColor = ""
		dimColor = ""
		descColor = ""
		resetColor = ""
	}

	// Build plain text for width calculation
	var plainArgs []string
	for i, arg := range ctx.Args {
		if i == ctx.ArgIndex {
			plainArgs = append(plainArgs, "*"+arg+"*")
		} else {
			plainArgs = append(plainArgs, arg)
		}
	}
	if ctx.ArgIndex >= len(ctx.Args) && ctx.ActiveArgName != "" {
		plainArgs = append(plainArgs, "*"+ctx.ActiveArgName+"*")
	}

	// Format: "  [diff(expr, *var*) - Description]"
	rawHint := fmt.Sprintf("  [%s(%s) - %s]", ctx.FuncName, strings.Join(plainArgs, ", "), ctx.Description)
	rawShortHint := fmt.Sprintf("  [%s(%s)]", ctx.FuncName, strings.Join(plainArgs, ", "))

	// Check width using moji cell width
	targetRaw := rawHint
	if getDisplayWidth(targetRaw) > availWidth-1 {
		targetRaw = rawShortHint
	}
	if getDisplayWidth(targetRaw) > availWidth-1 {
		// Truncate cleanly to fit within availWidth - 2
		targetRaw = truncateDisplayWidth(targetRaw, availWidth-2) + "…]"
		if getDisplayWidth(targetRaw) > availWidth-1 {
			return nil
		}
	}

	// Now build styled ANSI string matching the target display text
	var styled strings.Builder
	writeStyled := func(c, s string) {
		styled.WriteString(c)
		styled.WriteString(s)
		styled.WriteString(resetColor)
	}
	styled.WriteString("  ")
	writeStyled(prefixColor, "[")
	writeStyled(nameColor, ctx.FuncName)
	writeStyled(dimColor, "(")

	for i, arg := range ctx.Args {
		if i > 0 {
			writeStyled(dimColor, ", ")
		}
		if i == ctx.ArgIndex {
			writeStyled(activeColor, arg)
		} else {
			writeStyled(normalColor, arg)
		}
	}
	if ctx.ArgIndex >= len(ctx.Args) && ctx.ActiveArgName != "" {
		if len(ctx.Args) > 0 {
			writeStyled(dimColor, ", ")
		}
		writeStyled(activeColor, ctx.ActiveArgName)
	}
	writeStyled(dimColor, ")")

	// If full hint fit, add description
	if targetRaw == rawHint {
		writeStyled(dimColor, " - ")
		writeStyled(descColor, ctx.Description)
	}
	writeStyled(prefixColor, "]")

	_, err := io.WriteString(w, styled.String())
	return err
}

// RenderSignatureGuide is maintained for backward compatibility. It proxies to RenderSignatureInline.
func RenderSignatureGuide(w io.Writer, ctx *SignatureContext, theme string) error {
	return RenderSignatureInline(w, ctx, theme, 120)
}

// ClearSignatureGuide is maintained for backward compatibility. Since go-readline-ny's
// B.repaint() clears with B.eraseline() automatically before OnAfterRender, this is a clean no-op.
func ClearSignatureGuide(w io.Writer) error {
	return nil
}

func getDisplayWidth(s string) int {
	mojis := moji.StringToMoji(s)
	w := 0
	for _, m := range mojis {
		w += int(m.Width())
	}
	return w
}

func truncateDisplayWidth(s string, maxW int) string {
	mojis := moji.StringToMoji(s)
	var sb strings.Builder
	w := 0
	for _, m := range mojis {
		mw := int(m.Width())
		if w+mw > maxW {
			break
		}
		m.WriteTo(&sb)
		w += mw
	}
	return sb.String()
}
