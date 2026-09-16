package parser

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
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
	TokenLBracket
	TokenRBracket
	TokenGT
	TokenLT
	TokenGTE
	TokenLTE
	TokenEQ
)

// Token represents a single lexical token.
type Token struct {
	Type    TokenType
	Literal string
	RatVal  *ast.RationalNode
	Pos     int
}
