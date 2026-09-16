package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/parser"
)

func init() {
	// Inject calc-level dependencies into ast package hooks
	ast.ZeroDivisionErrorFactory = func(msg string) error {
		return NewZeroDivisionError(msg)
	}
	ast.DomainErrorFactory = func(msgKey, fallback string, args ...any) error {
		return NewDomainError(msgKey, fallback, args...)
	}
	ast.FuncValidator = ValidateFuncArgs

	// Inject calc-level dependencies into parser package hooks
	parser.SyntaxErrorFactory = func(msg string, args ...any) error {
		return NewSyntaxError(msg, args...)
	}
	parser.ReservedFuncChecker = IsReservedFunc
	parser.BareSymbolAllowedChecker = IsBareSymbolAllowed
	parser.FuncLookup = func(name string) (string, int, int, bool) {
		spec, ok := LookupFunction(name)
		if ok {
			return spec.Name, spec.MinArgs, spec.MaxArgs, true
		}
		return name, 0, 0, false
	}
}

// -------------------------------------------------------------------------
// Type aliases for AST types (backward compatibility layer)
// -------------------------------------------------------------------------

type NodeType = ast.NodeType

const (
	NodeRational = ast.NodeRational
	NodeSqrt     = ast.NodeSqrt
	NodeConst    = ast.NodeConst
	NodeFunc     = ast.NodeFunc
	NodeComplex  = ast.NodeComplex
	NodeAdd      = ast.NodeAdd
	NodeMul      = ast.NodeMul
	NodePow      = ast.NodePow
	NodeUnaryOp  = ast.NodeUnaryOp
	NodeVar      = ast.NodeVar
	NodeList     = ast.NodeList
	NodeMatrix   = ast.NodeMatrix
	NodePlot     = ast.NodePlot
	NodeRelOp    = ast.NodeRelOp
)

type (
	Node         = ast.Node
	RationalNode = ast.RationalNode
	SqrtNode     = ast.SqrtNode
	ConstNode    = ast.ConstNode
	FuncNode     = ast.FuncNode
	ComplexNode  = ast.ComplexNode
	AddNode      = ast.AddNode
	MulNode      = ast.MulNode
	PowNode      = ast.PowNode
	UnaryOpNode  = ast.UnaryOpNode
	VarNode      = ast.VarNode
	AssignStmt   = ast.AssignStmt
	ListNode     = ast.ListNode
	MatrixNode   = ast.MatrixNode
	PlotNode     = ast.PlotNode
	RelOpNode    = ast.RelOpNode
)

var (
	NewRational             = ast.NewRational
	NewRationalFromBigRat   = ast.NewRationalFromBigRat
	NewSqrt                 = ast.NewSqrt
	NewConst                = ast.NewConst
	NewFunc                 = ast.NewFunc
	NewComplex              = ast.NewComplex
	NewAdd                  = ast.NewAdd
	NewMul                  = ast.NewMul
	NewPow                  = ast.NewPow
	NewUnaryOp              = ast.NewUnaryOp
	NewVar                  = ast.NewVar
	NewList                 = ast.NewList
	NewMatrix               = ast.NewMatrix
	NewPlotNode             = ast.NewPlotNode
	NewRelOp                = ast.NewRelOp
	Walk                    = ast.Walk
	Inspect                 = ast.Inspect
	Transform               = ast.Transform
	Substitute              = ast.Substitute
	ContainsVar             = ast.ContainsVar
	ExtractFreeVariables    = ast.ExtractFreeVariables
)

// -------------------------------------------------------------------------
// Type aliases for Parser / Token / Lexer types
// -------------------------------------------------------------------------

type TokenType = parser.TokenType

const (
	TokenEOF      = parser.TokenEOF
	TokenIllegal  = parser.TokenIllegal
	TokenNumber   = parser.TokenNumber
	TokenIdent    = parser.TokenIdent
	TokenPlus     = parser.TokenPlus
	TokenMinus    = parser.TokenMinus
	TokenAsterisk = parser.TokenAsterisk
	TokenSlash    = parser.TokenSlash
	TokenCaret    = parser.TokenCaret
	TokenBang     = parser.TokenBang
	TokenLParen   = parser.TokenLParen
	TokenRParen   = parser.TokenRParen
	TokenComma    = parser.TokenComma
	TokenAssign   = parser.TokenAssign
	TokenLBracket = parser.TokenLBracket
	TokenRBracket = parser.TokenRBracket
	TokenGT       = parser.TokenGT
	TokenLT       = parser.TokenLT
	TokenGTE      = parser.TokenGTE
	TokenLTE      = parser.TokenLTE
	TokenEQ       = parser.TokenEQ
)

type (
	Token  = parser.Token
	Lexer  = parser.Lexer
	Parser = parser.Parser
)

var (
	NewLexer       = parser.NewLexer
	NewParser      = parser.NewParser
	Tokenize       = parser.Tokenize
	Parse          = parser.Parse
	ParseStatement = parser.ParseStatement
	NormalizeInput = parser.NormalizeInput
)
