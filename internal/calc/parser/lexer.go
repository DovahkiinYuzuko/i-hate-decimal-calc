package parser

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// TokenType represents lexical token types defined in token.go.

// Lexer breaks a mathematical expression into tokens.
type Lexer struct {
	input   string
	pos     int  // current byte offset
	readPos int  // next byte offset
	ch      rune // current character
}

// NewLexer creates and initializes a Lexer after normalizing Zenkaku characters to Hankaku.
func NewLexer(input string) *Lexer {
	normalized := NormalizeInput(input)
	l := &Lexer{input: normalized}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
		l.pos = l.readPos
		l.readPos++
	} else {
		r, size := utf8.DecodeRuneInString(l.input[l.readPos:])
		l.ch = r
		l.pos = l.readPos
		l.readPos += size
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
	return r
}

func (l *Lexer) skipWhitespace() {
	for unicode.IsSpace(l.ch) {
		l.readChar()
	}
}

// NextToken scans and returns the next token.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	startPos := l.pos
	var tok Token

	switch l.ch {
	case 0:
		tok = Token{Type: TokenEOF, Literal: "", Pos: startPos}
	case '+':
		tok = Token{Type: TokenPlus, Literal: "+", Pos: startPos}
		l.readChar()
	case '-':
		tok = Token{Type: TokenMinus, Literal: "-", Pos: startPos}
		l.readChar()
	case '*':
		tok = Token{Type: TokenAsterisk, Literal: "*", Pos: startPos}
		l.readChar()
	case '/':
		tok = Token{Type: TokenSlash, Literal: "/", Pos: startPos}
		l.readChar()
	case '^':
		tok = Token{Type: TokenCaret, Literal: "^", Pos: startPos}
		l.readChar()
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenNEQ, Literal: string(ch) + string(l.ch), Pos: startPos}
			l.readChar()
		} else {
			tok = Token{Type: TokenBang, Literal: "!", Pos: startPos}
			l.readChar()
		}
	case '(':
		tok = Token{Type: TokenLParen, Literal: "(", Pos: startPos}
		l.readChar()
	case ')':
		tok = Token{Type: TokenRParen, Literal: ")", Pos: startPos}
		l.readChar()
	case ',':
		tok = Token{Type: TokenComma, Literal: ",", Pos: startPos}
		l.readChar()
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenGTE, Literal: string(ch) + string(l.ch), Pos: startPos}
			l.readChar()
		} else {
			tok = Token{Type: TokenGT, Literal: ">", Pos: startPos}
			l.readChar()
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenLTE, Literal: string(ch) + string(l.ch), Pos: startPos}
			l.readChar()
		} else {
			tok = Token{Type: TokenLT, Literal: "<", Pos: startPos}
			l.readChar()
		}
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenEQ, Literal: string(ch) + string(l.ch), Pos: startPos}
			l.readChar()
		} else {
			tok = Token{Type: TokenAssign, Literal: "=", Pos: startPos}
			l.readChar()
		}
	case '[':
		tok = Token{Type: TokenLBracket, Literal: "[", Pos: startPos}
		l.readChar()
	case ']':
		tok = Token{Type: TokenRBracket, Literal: "]", Pos: startPos}
		l.readChar()
	case '√':
		tok = Token{Type: TokenIdent, Literal: "sqrt", Pos: startPos}
		l.readChar()
	case 'π':
		tok = Token{Type: TokenIdent, Literal: "pi", Pos: startPos}
		l.readChar()
	case '"':
		l.readChar() // consume opening '"'
		var sb strings.Builder
		for l.ch != 0 && l.ch != '"' {
			if l.ch == '\\' {
				l.readChar()
				switch l.ch {
				case '"', '\\':
					sb.WriteRune(l.ch)
				case 'n':
					sb.WriteRune('\n')
				case 't':
					sb.WriteRune('\t')
				default:
					sb.WriteRune('\\')
					sb.WriteRune(l.ch)
				}
			} else {
				sb.WriteRune(l.ch)
			}
			l.readChar()
		}
		if l.ch == '"' {
			l.readChar() // consume closing '"'
			tok = Token{Type: TokenString, Literal: sb.String(), Pos: startPos}
		} else {
			tok = Token{Type: TokenIllegal, Literal: "unterminated string literal", Pos: startPos}
		}
	default:
		if isDigit(l.ch) {
			return l.readNumber(startPos)
		} else if isLetter(l.ch) {
			lit := l.readIdentifier()
			if lit == "π" {
				lit = "pi"
			}
			return Token{Type: TokenIdent, Literal: lit, Pos: startPos}
		} else {
			tok = Token{Type: TokenIllegal, Literal: string(l.ch), Pos: startPos}
			l.readChar()
		}
	}

	return tok
}

type numState int

const (
	stateInt numState = iota
	stateDot
	stateNonRepeat
	stateOpenParen
	stateRepeat
	stateCloseParen
)

func (l *Lexer) readNumber(startPos int) Token {
	// 1. Check binary, octal, hex prefixes: 0b, 0o, 0x
	if l.ch == '0' {
		peek := l.peekChar()
		switch peek {
		case 'b', 'B':
			l.readChar() // consume '0'
			l.readChar() // consume 'b'/'B'
			var b strings.Builder
			for l.ch == '0' || l.ch == '1' {
				b.WriteRune(l.ch)
				l.readChar()
			}
			if b.Len() == 0 {
				return Token{Type: TokenIllegal, Literal: "0" + string(peek), Pos: startPos}
			}
			n := new(big.Int)
			n.SetString(b.String(), 2)
			return Token{
				Type:    TokenNumber,
				Literal: "0" + string(peek) + b.String(),
				RatVal:  &ast.RationalNode{Val: new(big.Rat).SetInt(n)},
				Pos:     startPos,
			}
		case 'o', 'O':
			l.readChar() // consume '0'
			l.readChar() // consume 'o'/'O'
			var b strings.Builder
			for l.ch >= '0' && l.ch <= '7' {
				b.WriteRune(l.ch)
				l.readChar()
			}
			if b.Len() == 0 {
				return Token{Type: TokenIllegal, Literal: "0" + string(peek), Pos: startPos}
			}
			n := new(big.Int)
			n.SetString(b.String(), 8)
			return Token{
				Type:    TokenNumber,
				Literal: "0" + string(peek) + b.String(),
				RatVal:  &ast.RationalNode{Val: new(big.Rat).SetInt(n)},
				Pos:     startPos,
			}
		case 'x', 'X':
			l.readChar() // consume '0'
			l.readChar() // consume 'x'/'X'
			var b strings.Builder
			for isHexDigit(l.ch) {
				b.WriteRune(l.ch)
				l.readChar()
			}
			if b.Len() == 0 {
				return Token{Type: TokenIllegal, Literal: "0" + string(peek), Pos: startPos}
			}
			n := new(big.Int)
			n.SetString(b.String(), 16)
			return Token{
				Type:    TokenNumber,
				Literal: "0" + string(peek) + b.String(),
				RatVal:  &ast.RationalNode{Val: new(big.Rat).SetInt(n)},
				Pos:     startPos,
			}
		}
	}

	state := stateInt
	var intPart strings.Builder
	var nonRepeatPart strings.Builder
	var repeatPart strings.Builder

	for {
		ch := l.ch
		switch state {
		case stateInt:
			if isDigit(ch) {
				intPart.WriteRune(ch)
				l.readChar()
			} else if ch == '^' && l.peekChar() == '^' {
				// Mathematica base^^digits syntax
				baseStr := intPart.String()
				base, err := strconv.Atoi(baseStr)
				if err == nil && base >= 2 && base <= 64 {
					l.readChar() // consume first '^'
					l.readChar() // consume second '^'
					var digits strings.Builder
					for isValidDigitInBase(l.ch, base) {
						digits.WriteRune(l.ch)
						l.readChar()
					}
					if digits.Len() == 0 {
						return Token{Type: TokenIllegal, Literal: baseStr + "^^", Pos: startPos}
					}
					n, ok := parseBigIntInBase(digits.String(), base)
					if !ok {
						return Token{Type: TokenIllegal, Literal: baseStr + "^^" + digits.String(), Pos: startPos}
					}
					return Token{
						Type:    TokenNumber,
						Literal: baseStr + "^^" + digits.String(),
						RatVal:  &ast.RationalNode{Val: new(big.Rat).SetInt(n)},
						Pos:     startPos,
					}
				}
				// If not valid base, fall through to pure integer and let '^' be operator
				intStr := intPart.String()
				numBig := new(big.Int)
				numBig.SetString(intStr, 10)
				rat := new(big.Rat).SetInt(numBig)
				return Token{
					Type:    TokenNumber,
					Literal: intStr,
					RatVal:  &ast.RationalNode{Val: rat},
					Pos:     startPos,
				}
			} else if ch == '.' {
				state = stateDot
				l.readChar()
			} else {
				// Pure integer
				intStr := intPart.String()
				numBig := new(big.Int)
				numBig.SetString(intStr, 10)
				rat := new(big.Rat).SetInt(numBig)
				return Token{
					Type:    TokenNumber,
					Literal: intStr,
					RatVal:  &ast.RationalNode{Val: rat},
					Pos:     startPos,
				}
			}

		case stateDot:
			if isDigit(ch) {
				state = stateNonRepeat
				nonRepeatPart.WriteRune(ch)
				l.readChar()
			} else if ch == '(' {
				state = stateOpenParen
				l.readChar()
			} else {
				// "123." without fraction digits -> treated as integer with dot
				intStr := intPart.String()
				numBig := new(big.Int)
				numBig.SetString(intStr, 10)
				rat := new(big.Rat).SetInt(numBig)
				return Token{
					Type:    TokenNumber,
					Literal: intStr + ".",
					RatVal:  &ast.RationalNode{Val: rat},
					Pos:     startPos,
				}
			}

		case stateNonRepeat:
			if isDigit(ch) {
				nonRepeatPart.WriteRune(ch)
				l.readChar()
			} else if ch == '(' {
				state = stateOpenParen
				l.readChar()
			} else {
				// Finite decimal: intStr.fracStr -> Exact rational
				intStr := intPart.String()
				fracStr := nonRepeatPart.String()
				numBig := new(big.Int)
				numBig.SetString(intStr+fracStr, 10)
				denomBig := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(len(fracStr))), nil)
				rat := new(big.Rat).SetFrac(numBig, denomBig)
				return Token{
					Type:    TokenNumber,
					Literal: intStr + "." + fracStr,
					RatVal:  &ast.RationalNode{Val: rat},
					Pos:     startPos,
				}
			}

		case stateOpenParen:
			if isDigit(ch) {
				state = stateRepeat
				repeatPart.WriteRune(ch)
				l.readChar()
			} else {
				// Empty repeating part like "0.()" or invalid char -> TokenIllegal
				return Token{
					Type:    TokenIllegal,
					Literal: string(ch),
					Pos:     l.pos,
				}
			}

		case stateRepeat:
			if isDigit(ch) {
				repeatPart.WriteRune(ch)
				l.readChar()
			} else if ch == ')' {
				state = stateCloseParen
				l.readChar()
			} else {
				// Missing closing paren or non-digit in repeat part -> TokenIllegal
				return Token{
					Type:    TokenIllegal,
					Literal: string(ch),
					Pos:     l.pos,
				}
			}

		case stateCloseParen:
			intStr := intPart.String()
			nonRepeatStr := nonRepeatPart.String()
			repeatStr := repeatPart.String()

			rat, err := repeatingDecimalToRat(intStr, nonRepeatStr, repeatStr)
			if err != nil {
				return Token{
					Type:    TokenIllegal,
					Literal: err.Error(),
					Pos:     startPos,
				}
			}
			lit := intStr + "." + nonRepeatStr + "(" + repeatStr + ")"
			return Token{
				Type:    TokenNumber,
				Literal: lit,
				RatVal:  &ast.RationalNode{Val: rat},
				Pos:     startPos,
			}
		}
	}
}

func repeatingDecimalToRat(intStr, nonRepeatStr, repeatStr string) (*big.Rat, error) {
	if intStr == "" {
		intStr = "0"
	}
	intBig, ok := new(big.Int).SetString(intStr, 10)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("parser.err_invalid_integer_part", intStr))
	}

	rLen := int64(len(repeatStr))
	if rLen == 0 {
		return nil, fmt.Errorf("%s", i18n.T("parser.err_repeating_part_cannot_be_empty"))
	}

	repBig, ok := new(big.Int).SetString(repeatStr, 10)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("parser.err_invalid_repeating_part", repeatStr))
	}

	tenPowR := new(big.Int).Exp(big.NewInt(10), big.NewInt(rLen), nil)
	nines := new(big.Int).Sub(tenPowR, big.NewInt(1))

	nLen := int64(len(nonRepeatStr))
	var fracRat *big.Rat

	if nLen == 0 {
		// Pure repeating decimal: R / (10^r - 1)
		fracRat = new(big.Rat).SetFrac(repBig, nines)
	} else {
		// Mixed repeating decimal: (N * (10^r - 1) + R) / (10^n * (10^r - 1))
		nonRepBig, ok := new(big.Int).SetString(nonRepeatStr, 10)
		if !ok {
			return nil, fmt.Errorf("%s", i18n.T("parser.err_invalid_non_repeating_part", nonRepeatStr))
		}
		num := new(big.Int).Mul(nonRepBig, nines)
		num.Add(num, repBig)

		tenPowN := new(big.Int).Exp(big.NewInt(10), big.NewInt(nLen), nil)
		denom := new(big.Int).Mul(tenPowN, nines)

		fracRat = new(big.Rat).SetFrac(num, denom)
	}

	result := new(big.Rat).SetInt(intBig)
	if intBig.Sign() < 0 {
		result.Sub(result, fracRat)
	} else {
		result.Add(result, fracRat)
	}
	return result, nil
}

func (l *Lexer) readIdentifier() string {
	startPos := l.pos
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[startPos:l.pos]
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_' || ch == 'π' || ch == '√'
}

func isHexDigit(ch rune) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func digitValInBase(r rune, base int) (int, bool) {
	var val int
	if '0' <= r && r <= '9' {
		val = int(r - '0')
	} else if base <= 36 {
		if 'a' <= r && r <= 'z' {
			val = int(r - 'a' + 10)
		} else if 'A' <= r && r <= 'Z' {
			val = int(r - 'A' + 10)
		} else {
			return 0, false
		}
	} else {
		// base > 36
		if 'a' <= r && r <= 'z' {
			val = int(r - 'a' + 10)
		} else if 'A' <= r && r <= 'Z' {
			val = int(r - 'A' + 36)
		} else if r == '@' {
			val = 62
		} else if r == '_' {
			val = 63
		} else {
			return 0, false
		}
	}
	if val < base {
		return val, true
	}
	return 0, false
}

func isValidDigitInBase(r rune, base int) bool {
	_, ok := digitValInBase(r, base)
	return ok
}

func parseBigIntInBase(s string, base int) (*big.Int, bool) {
	if base <= 62 {
		n := new(big.Int)
		return n.SetString(s, base)
	}
	// For base 63 and 64:
	bigBase := big.NewInt(int64(base))
	res := new(big.Int)
	for _, r := range s {
		val, ok := digitValInBase(r, base)
		if !ok {
			return nil, false
		}
		res.Mul(res, bigBase)
		res.Add(res, big.NewInt(int64(val)))
	}
	return res, true
}

// Tokenize reads all tokens from input until TokenEOF. Returns error if TokenIllegal is encountered.
func Tokenize(input string) ([]Token, error) {
	l := NewLexer(input)
	var tokens []Token
	for {
		tok := l.NextToken()
		if tok.Type == TokenIllegal {
			return nil, fmt.Errorf("%s", i18n.T("parser.err_illegal_character_at_position", tok.Literal, tok.Pos))
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}
