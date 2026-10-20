package main

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
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
				// Fallback description if translation key is missing
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

// RenderSignatureGuide prints the formatted signature guide below the current line using ANSI codes.
func RenderSignatureGuide(w io.Writer, ctx *SignatureContext, theme string) error {
	if ctx == nil {
		return nil
	}

	// ANSI color palette
	nameColor := "\x1B[1;36m" // Bold Cyan
	activeColor := "\x1B[1;4;33m" // Bold Underline Yellow
	normalColor := "\x1B[37m" // White
	dimColor := "\x1B[90m" // Gray
	descColor := "\x1B[32m" // Green
	resetColor := "\x1B[0m"

	if theme == "light" {
		nameColor = "\x1B[1;34m" // Bold Blue
		activeColor = "\x1B[1;4;35m" // Bold Underline Magenta
		normalColor = "\x1B[30m" // Black
		dimColor = "\x1B[90m"
		descColor = "\x1B[32m"
	} else if theme == "none" {
		nameColor = ""
		activeColor = ""
		normalColor = ""
		dimColor = ""
		descColor = ""
		resetColor = ""
	}

	// Build arguments representation
	var argsBuilder strings.Builder
	for i, arg := range ctx.Args {
		if i > 0 {
			argsBuilder.WriteString(dimColor + ", " + resetColor)
		}
		if i == ctx.ArgIndex {
			argsBuilder.WriteString(activeColor + arg + resetColor)
		} else {
			argsBuilder.WriteString(normalColor + arg + resetColor)
		}
	}
	if ctx.ArgIndex >= len(ctx.Args) && ctx.ActiveArgName != "" {
		if len(ctx.Args) > 0 {
			argsBuilder.WriteString(dimColor + ", " + resetColor)
		}
		argsBuilder.WriteString(activeColor + ctx.ActiveArgName + resetColor)
	}

	guideText := fmt.Sprintf("%s[%s]%s %s%s(%s)%s - %s%s%s",
		dimColor, ctx.FuncName, resetColor,
		nameColor, ctx.FuncName, argsBuilder.String(), resetColor,
		descColor, ctx.Description, resetColor,
	)

	// Save cursor -> move down 1 line -> clear line -> print guide -> restore cursor
	_, err := fmt.Fprintf(w, "\x1B7\x1B[1B\r\x1B[2K%s\x1B8", guideText)
	return err
}

// ClearSignatureGuide clears the status line below the current cursor without disturbing input.
func ClearSignatureGuide(w io.Writer) error {
	_, err := fmt.Fprint(w, "\x1B7\x1B[1B\r\x1B[2K\x1B8")
	return err
}
