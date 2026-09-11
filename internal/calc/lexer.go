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
	TokenAssign
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
	case '=':
		tok = Token{Type: TokenAssign, Literal: "=", Pos: startPos}
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
					RatVal:  &RationalNode{Val: rat},
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
					RatVal:  &RationalNode{Val: rat},
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
					RatVal:  &RationalNode{Val: rat},
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
				RatVal:  &RationalNode{Val: rat},
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
		return nil, fmt.Errorf("invalid integer part: %s", intStr)
	}

	rLen := int64(len(repeatStr))
	if rLen == 0 {
		return nil, fmt.Errorf("repeating part cannot be empty")
	}

	repBig, ok := new(big.Int).SetString(repeatStr, 10)
	if !ok {
		return nil, fmt.Errorf("invalid repeating part: %s", repeatStr)
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
			return nil, fmt.Errorf("invalid non-repeating part: %s", nonRepeatStr)
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
