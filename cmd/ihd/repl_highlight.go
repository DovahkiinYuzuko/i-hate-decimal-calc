package main

import (
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
	TokenCustomVar
)

// TokenSpan records the byte slice range, category, and bracket depth of a token.
type TokenSpan struct {
	Type  TokenType
	Start int
	End   int
	Depth int
}

// FSMHighlightMatcher implements the Pattern interface required by readline.Highlight:
// interface{ FindAllStringIndex(string, int) [][]int }
// It drives highlight ranges entirely through the lexical FSM without regular expressions.
type FSMHighlightMatcher struct {
	TargetType  TokenType
	TargetDepth int // 0..3 for rainbow depths; -1 for bracket mismatch; 0 for other tokens
	Env         *calc.Env
}

// FindAllStringIndex scans the input string using TokenizeHighlightFSM and returns
// matched byte index ranges for the targeted token category or bracket depth.
func (m *FSMHighlightMatcher) FindAllStringIndex(str string, n int) [][]int {
	spans := TokenizeHighlightFSM(str, m.Env)
	var matches [][]int
	for _, s := range spans {
		if s.Type != m.TargetType {
			continue
		}
		if m.TargetType == TokenRainbow {
			if (s.Depth % 4) != m.TargetDepth {
				continue
			}
		}
		matches = append(matches, []int{s.Start, s.End})
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
// tracking strings, numbers, identifiers, operators, custom variables, and a bracket stack.
func TokenizeHighlightFSM(input string, env *calc.Env) []TokenSpan {
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

		// 2. Delimiters & Brackets (Only () and [] are supported CAS mathematical delimiters)
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

		// 4. Identifiers (Functions, Commands, Defined Custom Variables, or Unbound Symbols)
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
			} else if env != nil {
				if _, ok := env.Get(ident); ok {
					// Defined custom variable in active environment (semantic highlight)
					tokType = TokenCustomVar
				}
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

		// Whitespace or unknown characters (e.g. { })
		i++
	}

	return spans
}

func isOpeningBracket(r rune) bool {
	return r == '(' || r == '['
}

func isClosingBracket(r rune) bool {
	return r == ')' || r == ']'
}

func matchesBracket(open, close rune) bool {
	return (open == '(' && close == ')') ||
		(open == '[' && close == ']')
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
// customized for the specified color theme ("dark", "light", or "none") and environment.
func BuildREPLHighlights(theme string, env *calc.Env) []readline.Highlight {
	themeLower := strings.ToLower(strings.TrimSpace(theme))
	if themeLower == "none" || themeLower == "off" {
		return nil
	}

	isLight := themeLower == "light"

	// High-contrast, transparent-terminal-safe ANSI Palette Definitions
	var (
		seqRainbow0   string
		seqRainbow1   string
		seqRainbow2   string
		seqRainbow3   string
		seqMismatch   string = "\x1B[97;41m" // White on red background (error alert)
		seqFunc       string
		seqCmd        string
		seqNumber     string
		seqString     string
		seqOp         string
		seqCustomVar  string
	)

	if isLight {
		// Light Theme (Optimized for white / bright backgrounds)
		seqRainbow0 = "\x1B[34;1m" // Bold Dark Blue
		seqRainbow1 = "\x1B[35;1m" // Bold Dark Magenta
		seqRainbow2 = "\x1B[32;1m" // Bold Dark Green
		seqRainbow3 = "\x1B[36;1m" // Bold Dark Cyan
		seqFunc = "\x1B[34m"       // Dark Blue
		seqCmd = "\x1B[35m"        // Dark Magenta
		seqNumber = "\x1B[31;1m"   // Bold Brick Red (Completely separated from rainbow blue/cyan)
		seqString = "\x1B[32m"     // Dark Green
		seqOp = "\x1B[30m"         // Solid Black
		seqCustomVar = "\x1B[36m"  // Teal / Cyan (Semantic defined variable)
	} else {
		// Dark Theme (Optimized for dark backgrounds and transparent photo terminals)
		// Brackets rotate through Cyan, Magenta, Green, Blue
		seqRainbow0 = "\x1B[96m"   // Bright Cyan
		seqRainbow1 = "\x1B[95m"   // Bright Magenta
		seqRainbow2 = "\x1B[92m"   // Bright Green
		seqRainbow3 = "\x1B[94m"   // Bright Blue
		seqFunc = "\x1B[94;1m"     // Bold Bright Blue
		seqCmd = "\x1B[36;1m"      // Bold Cyan
		seqNumber = "\x1B[93m"     // Bright Yellow (Completely separated from rainbow cyan/magenta)
		seqString = "\x1B[32m"     // Green
		seqOp = "\x1B[37m"         // Bright White (Clear & readable on dark / transparent backgrounds)
		seqCustomVar = "\x1B[96;2m" // Soft Cyan / Mint (Semantic defined variable; unbound symbols remain white)
	}

	// Order of highlights: operators, numbers, strings, commands, funcs, custom variables, rainbow, mismatch
	return []readline.Highlight{
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenOp, Env: env},
			Sequence: seqOp,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenNumber, Env: env},
			Sequence: seqNumber,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenString, Env: env},
			Sequence: seqString,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenCmd, Env: env},
			Sequence: seqCmd,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenFunc, Env: env},
			Sequence: seqFunc,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenCustomVar, Env: env},
			Sequence: seqCustomVar,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenRainbow, TargetDepth: 0, Env: env},
			Sequence: seqRainbow0,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenRainbow, TargetDepth: 1, Env: env},
			Sequence: seqRainbow1,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenRainbow, TargetDepth: 2, Env: env},
			Sequence: seqRainbow2,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenRainbow, TargetDepth: 3, Env: env},
			Sequence: seqRainbow3,
		},
		{
			Pattern:  &FSMHighlightMatcher{TargetType: TokenMismatch, TargetDepth: -1, Env: env},
			Sequence: seqMismatch,
		},
	}
}
