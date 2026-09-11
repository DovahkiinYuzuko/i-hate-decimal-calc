package calc

import (
	"fmt"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType represents the type of a lexical token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIllegal
	TokenNumber
	TokenIdent
	TokenPlus
	TokenMinus
	TokenAsterisk
	TokenSlash
	TokenCaret
	TokenBang
	TokenLParen
	TokenRParen
	TokenComma
)

// Token represents a single lexical token.
type Token struct {
	Type    TokenType
	Literal string
	RatVal  *RationalNode
	Pos     int
}

// Lexer breaks a mathematical expression into tokens.
type Lexer struct {
	input   string
	pos     int  // current byte offset
	readPos int  // next byte offset
	ch      rune // current character
}

// NewLexer creates and initializes a Lexer.
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
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
		tok = Token{Type: TokenBang, Literal: "!", Pos: startPos}
		l.readChar()
	case '(':
		tok = Token{Type: TokenLParen, Literal: "(", Pos: startPos}
		l.readChar()
	case ')':
		tok = Token{Type: TokenRParen, Literal: ")", Pos: startPos}
		l.readChar()
	case ',':
		tok = Token{Type: TokenComma, Literal: ",", Pos: startPos}
		l.readChar()
	case '√':
		tok = Token{Type: TokenIdent, Literal: "sqrt", Pos: startPos}
		l.readChar()
	case 'π':
		tok = Token{Type: TokenIdent, Literal: "pi", Pos: startPos}
		l.readChar()
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

func (l *Lexer) readNumber(startPos int) Token {
	hasDot := false
	var intPart strings.Builder
	var fracPart strings.Builder

	for isDigit(l.ch) || (l.ch == '.' && !hasDot) {
		if l.ch == '.' {
			hasDot = true
			l.readChar()
			continue
		}
		if hasDot {
			fracPart.WriteRune(l.ch)
		} else {
			intPart.WriteRune(l.ch)
		}
		l.readChar()
	}

	intStr := intPart.String()
	fracStr := fracPart.String()

	if !hasDot {
		// Pure integer
		numBig := new(big.Int)
		numBig.SetString(intStr, 10)
		rat := new(big.Rat).SetInt(numBig)
		return Token{
			Type:    TokenNumber,
			Literal: intStr,
			RatVal:  &RationalNode{Val: rat},
			Pos:     startPos,
		}
	}

	// Finite decimal: intStr.fracStr -> Exact rational without float64
	// e.g. 6.441 -> (6 * 1000 + 441) / 1000 = 6441 / 1000
	numBig := new(big.Int)
	numBig.SetString(intStr+fracStr, 10)

	denomBig := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(len(fracStr))), nil)
	rat := new(big.Rat).SetFrac(numBig, denomBig)

	lit := intStr + "." + fracStr
	return Token{
		Type:    TokenNumber,
		Literal: lit,
		RatVal:  &RationalNode{Val: rat},
		Pos:     startPos,
	}
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

// Tokenize reads all tokens from input until TokenEOF. Returns error if TokenIllegal is encountered.
func Tokenize(input string) ([]Token, error) {
	l := NewLexer(input)
	var tokens []Token
	for {
		tok := l.NextToken()
		if tok.Type == TokenIllegal {
			return nil, fmt.Errorf("illegal character '%s' at position %d", tok.Literal, tok.Pos)
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}
