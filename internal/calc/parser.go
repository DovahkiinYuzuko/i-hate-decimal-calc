package calc

import (
	"fmt"
)

// Precedence levels for Pratt parsing
const (
	_ int = iota
	PREC_LOWEST
	PREC_SUM         // + -
	PREC_PREFIX_MINUS // unary - (lower than POWER, so -3^2 becomes -(3^2))
	PREC_PRODUCT     // * /
	PREC_POWER       // ^ (right-associative)
	PREC_POSTFIX     // !
)

var precedences = map[TokenType]int{
	TokenPlus:     PREC_SUM,
	TokenMinus:    PREC_SUM,
	TokenAsterisk: PREC_PRODUCT,
	TokenSlash:    PREC_PRODUCT,
	TokenCaret:    PREC_POWER,
	TokenBang:     PREC_POSTFIX,
}

// Parser parses a token stream into an AST node.
type Parser struct {
	tokens  []Token
	pos     int
	curTok  Token
	peekTok Token
}

// NewParser creates a new Parser instance.
func NewParser(l *Lexer) *Parser {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF || tok.Type == TokenIllegal {
			break
		}
	}

	p := &Parser{tokens: tokens, pos: 0}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curTok = p.peekTok
	if p.pos < len(p.tokens) {
		p.peekTok = p.tokens[p.pos]
		p.pos++
	} else {
		p.peekTok = Token{Type: TokenEOF, Literal: ""}
	}
}

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekTok.Type]; ok {
		return prec
	}
	return PREC_LOWEST
}

// Parse parses the entire input string into an AST Node.
func Parse(input string) (Node, error) {
	l := NewLexer(input)
	p := NewParser(l)

	if p.curTok.Type == TokenEOF {
		return nil, fmt.Errorf("syntax error: empty expression")
	}

	node, err := p.ParseExpression(PREC_LOWEST)
	if err != nil {
		return nil, err
	}

	if p.peekTok.Type != TokenEOF {
		if p.isStartOfExpression(p.peekTok.Type) {
			return nil, fmt.Errorf("syntax error: implicit multiplication is not allowed. Please write explicit '*' operator near %q", p.peekTok.Literal)
		}
		return nil, fmt.Errorf("syntax error: unexpected token %q at position %d", p.peekTok.Literal, p.peekTok.Pos)
	}

	return node, nil
}

func isReservedFunc(name string) bool {
	switch name {
	case "sqrt", "sin", "cos", "tan", "log", "ln", "abs", "cbrt", "gcd", "lcm", "mod", "perm", "comb", "rand", "asin", "acos", "atan":
		return true
	default:
		return false
	}
}

func isReservedConst(name string) bool {
	switch name {
	case "pi", "e", "i", "deg":
		return true
	default:
		return false
	}
}

// ParseStatement parses an input string as either an assignment statement (*AssignStmt) or an expression (Node).
func ParseStatement(input string) (interface{}, error) {
	l := NewLexer(input)
	p := NewParser(l)

	if p.curTok.Type == TokenEOF {
		return nil, fmt.Errorf("syntax error: empty input")
	}

	// Check if this is an assignment: Ident = Expr
	if p.curTok.Type == TokenIdent && p.peekTok.Type == TokenAssign {
		varName := p.curTok.Literal
		if isReservedConst(varName) || isReservedFunc(varName) {
			return nil, fmt.Errorf(MsgErrReservedWord, varName)
		}
		p.nextToken() // move to '='
		p.nextToken() // move to start of expression

		if p.curTok.Type == TokenEOF {
			return nil, fmt.Errorf("syntax error: missing expression after '='")
		}

		expr, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}

		if p.peekTok.Type != TokenEOF {
			return nil, fmt.Errorf("syntax error: unexpected token %q after expression", p.peekTok.Literal)
		}

		return &AssignStmt{Name: varName, Value: expr}, nil
	}

	return Parse(input)
}

// ParseExpression parses an expression with Pratt parsing algorithm.
func (p *Parser) ParseExpression(precedence int) (Node, error) {
	if p.curTok.Type == TokenIllegal {
		return nil, fmt.Errorf("syntax error: illegal character %q at position %d", p.curTok.Literal, p.curTok.Pos)
	}

	// 1. Prefix parse
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}

	// Check if next token starts another expression directly (implicit multiplication like 2pi or (1+2)(3+4))
	if p.isStartOfExpression(p.peekTok.Type) {
		return nil, fmt.Errorf("syntax error: implicit multiplication is not allowed. Please write explicit '*' operator near %q", p.peekTok.Literal)
	}

	// 2. Infix parse
	for p.peekTok.Type != TokenEOF && precedence < p.peekPrecedence() {
		p.nextToken()
		left, err = p.parseInfix(left)
		if err != nil {
			return nil, err
		}

		if p.isStartOfExpression(p.peekTok.Type) {
			return nil, fmt.Errorf("syntax error: implicit multiplication is not allowed. Please write explicit '*' operator near %q", p.peekTok.Literal)
		}
	}

	return left, nil
}

func (p *Parser) parsePrefix() (Node, error) {
	switch p.curTok.Type {
	case TokenNumber:
		return p.curTok.RatVal, nil

	case TokenIdent:
		name := p.curTok.Literal
		if p.peekTok.Type == TokenLParen {
			p.nextToken() // move to '('
			return p.parseFuncCall(name)
		}
		if isReservedFunc(name) {
			return nil, fmt.Errorf("syntax error at position %d: function %q missing arguments", p.curTok.Pos, name)
		}
		// Constants: pi, e, or imaginary unit i
		if name == "i" {
			zero, _ := NewRational(0, 1)
			one, _ := NewRational(1, 1)
			return NewComplex(zero, one), nil
		}
		if isReservedConst(name) {
			return NewConst(name)
		}
		// Otherwise, it is a variable symbol (e.g., "x", "ans")
		return NewVar(name), nil

	case TokenMinus:
		// Unary minus: use PREC_PREFIX_MINUS so that -3^2 parses as -(3^2)
		p.nextToken()
		expr, err := p.ParseExpression(PREC_PREFIX_MINUS)
		if err != nil {
			return nil, err
		}
		return NewUnaryOp("-", expr)

	case TokenPlus:
		p.nextToken()
		return p.ParseExpression(PREC_PREFIX_MINUS)

	case TokenLParen:
		p.nextToken() // consume '('
		expr, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		if p.peekTok.Type != TokenRParen {
			return nil, fmt.Errorf("syntax error: expected matching ')', got %q", p.peekTok.Literal)
		}
		p.nextToken() // move curTok to ')'
		return expr, nil

	default:
		return nil, fmt.Errorf("syntax error: unexpected token %q at position %d", p.curTok.Literal, p.curTok.Pos)
	}
}

func (p *Parser) parseInfix(left Node) (Node, error) {
	switch p.curTok.Type {
	case TokenPlus:
		p.nextToken()
		right, err := p.ParseExpression(PREC_SUM)
		if err != nil {
			return nil, err
		}
		return NewAdd([]Node{left, right}), nil

	case TokenMinus:
		p.nextToken()
		right, err := p.ParseExpression(PREC_SUM)
		if err != nil {
			return nil, err
		}
		negRight, err := NewUnaryOp("-", right)
		if err != nil {
			return nil, err
		}
		return NewAdd([]Node{left, negRight}), nil

	case TokenAsterisk:
		p.nextToken()
		right, err := p.ParseExpression(PREC_PRODUCT)
		if err != nil {
			return nil, err
		}
		return NewMul([]Node{left, right}), nil

	case TokenSlash:
		p.nextToken()
		right, err := p.ParseExpression(PREC_PRODUCT)
		if err != nil {
			return nil, err
		}
		// Division: left / right -> Mul(left, Pow(right, -1))
		negOne, _ := NewRational(-1, 1)
		reciprocal, err := NewPow(right, negOne)
		if err != nil {
			return nil, err
		}
		return NewMul([]Node{left, reciprocal}), nil

	case TokenCaret:
		// Right associative: pass PREC_POWER - 1
		p.nextToken()
		right, err := p.ParseExpression(PREC_POWER - 1)
		if err != nil {
			return nil, err
		}
		return NewPow(left, right)

	case TokenBang:
		// Postfix factorial: left!
		return NewUnaryOp("!", left)

	default:
		return nil, fmt.Errorf("syntax error: unexpected infix token %q", p.curTok.Literal)
	}
}

func (p *Parser) parseFuncCall(name string) (Node, error) {
	// curTok is '('
	if p.peekTok.Type == TokenRParen {
		return nil, fmt.Errorf("syntax error: function %q requires arguments", name)
	}

	p.nextToken() // move to first arg

	var args []Node
	for {
		arg, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		if p.peekTok.Type == TokenComma {
			p.nextToken() // curTok is ','
			p.nextToken() // move to next arg
			continue
		}
		if p.peekTok.Type == TokenRParen {
			p.nextToken() // curTok is ')'
			break
		}
		return nil, fmt.Errorf("syntax error: expected ',' or ')' in function call %q, got %q", name, p.peekTok.Literal)
	}

	if name == "sqrt" {
		if len(args) != 1 {
			return nil, fmt.Errorf("sqrt requires exactly 1 argument, got %d", len(args))
		}
		return NewSqrt(args[0]), nil
	}

	return NewFunc(name, args)
}

func (p *Parser) isStartOfExpression(t TokenType) bool {
	switch t {
	case TokenNumber, TokenIdent, TokenLParen:
		return true
	default:
		return false
	}
}
