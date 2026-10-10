package main

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/nyaosorg/go-readline-ny"
)

// TokenType represents syntactic token categories for REPL highlighting.
type TokenType int

const (
	TokenNone TokenType = iota
	TokenNumber
	TokenString
	TokenFunc
	TokenCmd
	TokenOp
	TokenRainbow
	TokenMismatch
)

// TokenSpan records the byte slice range, category, and bracket depth of a token.
type TokenSpan struct {
	Type  TokenType
	Start int
	End   int
	Depth int
}

// RainbowMatcher implements the Pattern interface required by readline.Highlight:
// interface{ FindAllStringIndex(string, int) [][]int }
type RainbowMatcher struct {
	TargetDepth int // 0..3 for rainbow depths; -1 for bracket mismatch
}

// FindAllStringIndex scans the input string using LexerHighlightFSM and returns
// matched byte index ranges for the targeted bracket depth or mismatch.
func (m *RainbowMatcher) FindAllStringIndex(str string, n int) [][]int {
	spans := TokenizeHighlightFSM(str)
	var matches [][]int
	for _, s := range spans {
		if m.TargetDepth == -1 {
			if s.Type == TokenMismatch {
				matches = append(matches, []int{s.Start, s.End})
			}
		} else {
			if s.Type == TokenRainbow && (s.Depth%4) == m.TargetDepth {
				matches = append(matches, []int{s.Start, s.End})
			}
		}
		if n > 0 && len(matches) >= n {
			break
		}
	}
	return matches
}

type bracketFrame struct {
	r     rune
	start int
	depth int
}

// TokenizeHighlightFSM executes a lexical finite-state machine over the input line,
// tracking strings, numbers, identifiers, operators, and a bracket stack for rainbow delimiters.
func TokenizeHighlightFSM(input string) []TokenSpan {
	var spans []TokenSpan
	runes := []rune(input)
	n := len(runes)

	// Pre-build byte offset map for accurate slicing
	byteOffsets := make([]int, n+1)
	bytePos := 0
	for i, r := range runes {
		byteOffsets[i] = bytePos
		bytePos += len(string(r))
	}
	byteOffsets[n] = bytePos

	var bracketStack []bracketFrame

	i := 0
	for i < n {
		r := runes[i]

		// 1. String literal FSM: " ... "
		if r == '"' {
			startIdx := i
			i++
			for i < n {
				if runes[i] == '\\' {
					i += 2 // skip escaped character
					continue
				}
				if runes[i] == '"' {
					i++ // closing quote
					break
				}
				i++
			}
			spans = append(spans, TokenSpan{
				Type:  TokenString,
				Start: byteOffsets[startIdx],
				End:   byteOffsets[i],
			})
			continue
		}

		// 2. Delimiters & Brackets
		if isOpeningBracket(r) {
			depth := len(bracketStack)
			bracketStack = append(bracketStack, bracketFrame{
				r:     r,
				start: byteOffsets[i],
				depth: depth,
			})
			spans = append(spans, TokenSpan{
				Type:  TokenRainbow,
				Start: byteOffsets[i],
				End:   byteOffsets[i+1],
				Depth: depth,
			})
			i++
			continue
		}

		if isClosingBracket(r) {
			if len(bracketStack) == 0 {
				// Unmatched closing bracket
				spans = append(spans, TokenSpan{
					Type:  TokenMismatch,
					Start: byteOffsets[i],
					End:   byteOffsets[i+1],
				})
			} else {
				top := bracketStack[len(bracketStack)-1]
				bracketStack = bracketStack[:len(bracketStack)-1]
				if matchesBracket(top.r, r) {
					spans = append(spans, TokenSpan{
						Type:  TokenRainbow,
						Start: byteOffsets[i],
						End:   byteOffsets[i+1],
						Depth: top.depth,
					})
				} else {
					// Mismatched bracket type
					spans = append(spans, TokenSpan{
						Type:  TokenMismatch,
						Start: byteOffsets[i],
						End:   byteOffsets[i+1],
					})
				}
			}
			i++
			continue
		}

		// 3. Numbers & Radix literals (e.g. 16^^FF, 0x1A, 3.14, 42)
		if unicode.IsDigit(r) || (r == '.' && i+1 < n && unicode.IsDigit(runes[i+1])) {
			startIdx := i
			for i < n && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '.' || runes[i] == '^') {
				i++
			}
			spans = append(spans, TokenSpan{
				Type:  TokenNumber,
				Start: byteOffsets[startIdx],
				End:   byteOffsets[i],
			})
			continue
		}

		// 4. Identifiers (Functions, Commands, Variables)
		if unicode.IsLetter(r) || r == '_' {
			startIdx := i
			for i < n && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			ident := string(runes[startIdx:i])
			tokType := TokenNone
			lower := strings.ToLower(ident)
			if isCommand(lower) {
				tokType = TokenCmd
			} else if calc.IsReservedFunc(lower) {
				tokType = TokenFunc
			}

			if tokType != TokenNone {
				spans = append(spans, TokenSpan{
					Type:  tokType,
					Start: byteOffsets[startIdx],
					End:   byteOffsets[i],
				})
			}
			continue
		}

		// 5. Operators & Relational symbols
		if isOperatorRune(r) {
			startIdx := i
			for i < n && isOperatorRune(runes[i]) {
				i++
			}
			spans = append(spans, TokenSpan{
				Type:  TokenOp,
				Start: byteOffsets[startIdx],
				End:   byteOffsets[i],
			})
			continue
		}

		// Whitespace or unknown characters
		i++
	}

	return spans
}

func isOpeningBracket(r rune) bool {
	return r == '(' || r == '[' || r == '{'
}

func isClosingBracket(r rune) bool {
	return r == ')' || r == ']' || r == '}'
}

func matchesBracket(open, close rune) bool {
	return (open == '(' && close == ')') ||
		(open == '[' && close == ']') ||
		(open == '{' && close == '}')
}

func isOperatorRune(r rune) bool {
	switch r {
	case '+', '-', '*', '/', '^', '=', '!', '<', '>', '%', '&', '|', '~', ':':
		return true
	default:
		return false
	}
}

func isCommand(cmd string) bool {
	switch cmd {
	case "vars", "exit", "quit", "help", "theme", "highlight", "color":
		return true
	default:
		return false
	}
}

// BuildREPLHighlights constructs a slice of readline.Highlight configurations
// customized for the specified color theme ("dark", "light", or "none").
func BuildREPLHighlights(theme string) []readline.Highlight {
	themeLower := strings.ToLower(strings.TrimSpace(theme))
	if themeLower == "none" || themeLower == "off" {
		return nil
	}

	isLight := themeLower == "light"

	// ANSI Palette Definitions
	var (
		seqRainbow0 string
		seqRainbow1 string
		seqRainbow2 string
		seqRainbow3 string
		seqMismatch string = "\x1B[97;41m" // White on red background
		seqFunc     string
		seqCmd      string
		seqNumber   string
		seqString   string
		seqOp       string
	)

	if isLight {
		// Light Theme (Optimized for white / bright backgrounds)
		seqRainbow0 = "\x1B[34;1m" // Bold Blue
		seqRainbow1 = "\x1B[35;1m" // Bold Magenta
		seqRainbow2 = "\x1B[36;1m" // Bold Cyan
		seqRainbow3 = "\x1B[32;1m" // Bold Green
		seqFunc = "\x1B[34m"       // Dark Blue
		seqCmd = "\x1B[36m"        // Dark Cyan
		seqNumber = "\x1B[31m"     // Dark Red / Brick
		seqString = "\x1B[32m"     // Dark Green
		seqOp = "\x1B[30;1m"       // Dark Charcoal / Dim
	} else {
		// Dark Theme (Optimized for black / dark purple backgrounds)
		seqRainbow0 = "\x1B[93m" // Bright Yellow
		seqRainbow1 = "\x1B[95m" // Bright Magenta
		seqRainbow2 = "\x1B[96m" // Bright Cyan
		seqRainbow3 = "\x1B[92m" // Bright Green
		seqFunc = "\x1B[94m"     // Bright Blue
		seqCmd = "\x1B[36m"      // Cyan
		seqNumber = "\x1B[33m"   // Amber / Yellow
		seqString = "\x1B[32m"   // Green
		seqOp = "\x1B[90m"       // Gray / Dim
	}

	highlights := []readline.Highlight{
		// 1. String Literals
		{
			Pattern:  regexp.MustCompile(`"([^"\\]|\\.)*"`),
			Sequence: seqString,
		},
		// 2. CLI Commands
		{
			Pattern:  regexp.MustCompile(`\b(?i)(vars|exit|quit|help|theme|highlight|color)\b`),
			Sequence: seqCmd,
		},
		// 3. Numbers (Standard & Radix e.g. 16^^FF, 0x1A, 3.14)
		{
			Pattern:  regexp.MustCompile(`\b\d+(\^\^[0-9a-zA-Z]+|\.\d+|[xXoObB][0-9a-fA-F]+)?\b`),
			Sequence: seqNumber,
		},
		// 4. Operators
		{
			Pattern:  regexp.MustCompile(`[+\-*/^=<>!%&|~:]+`),
			Sequence: seqOp,
		},
		// 5. Rainbow Delimiters (Depth 0..3)
		{
			Pattern:  &RainbowMatcher{TargetDepth: 0},
			Sequence: seqRainbow0,
		},
		{
			Pattern:  &RainbowMatcher{TargetDepth: 1},
			Sequence: seqRainbow1,
		},
		{
			Pattern:  &RainbowMatcher{TargetDepth: 2},
			Sequence: seqRainbow2,
		},
		{
			Pattern:  &RainbowMatcher{TargetDepth: 3},
			Sequence: seqRainbow3,
		},
		// 6. Bracket Mismatches (Errors)
		{
			Pattern:  &RainbowMatcher{TargetDepth: -1},
			Sequence: seqMismatch,
		},
	}

	// 7. Built-in Function Names (Case-insensitive matching)
	funcPattern := buildReservedFuncRegex()
	if funcPattern != nil {
		// Insert function regex right after commands
		highlights = append([]readline.Highlight{
			highlights[0],
			highlights[1],
			{Pattern: funcPattern, Sequence: seqFunc},
		}, highlights[2:]...)
	}

	return highlights
}

func buildReservedFuncRegex() *regexp.Regexp {
	// Sample key built-ins for efficient regex pattern
	funcs := []string{
		"sin", "cos", "tan", "asin", "acos", "atan", "sinh", "cosh", "tanh",
		"solve", "diff", "integrate", "limit", "dsolve", "taylor", "series",
		"matrix", "det", "inv", "eigenvalues", "eigenvectors",
		"gcd", "lcm", "factor", "prime", "groebner", "cad", "qe",
		"to_base", "from_base", "table", "for", "product", "sum", "plot",
	}
	pattern := `\b(?i)(` + strings.Join(funcs, "|") + `)\b`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	return re
}
