package parser

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// Hooks for error factories and function registry lookup to allow decoupling from higher-level calc/registry packages.
var (
	// SyntaxErrorFactory creates structured syntax errors. If nil, standard fmt.Errorf is used.
	SyntaxErrorFactory func(msg string, args ...any) error

	// ReservedFuncChecker checks if an identifier is a registered function name.
	ReservedFuncChecker func(name string) bool

	// BareSymbolAllowedChecker checks if a function name can be used as a bare symbol (e.g. 'e', 'deg').
	BareSymbolAllowedChecker func(name string) bool

	// FuncLookup queries function metadata (canonical name, minArgs, maxArgs).
	FuncLookup func(name string) (canonicalName string, minArgs int, maxArgs int, ok bool)
)

func newSyntaxErr(key string, args ...any) error {
	msg := i18n.T(key, args...)
	if SyntaxErrorFactory != nil {
		return SyntaxErrorFactory(msg, args...)
	}
	return fmt.Errorf(msg, args...)
}

func isReservedFunc(name string) bool {
	if ReservedFuncChecker != nil {
		return ReservedFuncChecker(name)
	}
	return false
}

func isBareSymbolAllowed(name string) bool {
	if BareSymbolAllowedChecker != nil {
		return BareSymbolAllowedChecker(name)
	}
	return false
}

func isReservedConst(name string) bool {
	switch name {
	case "pi", "e", "i", "deg":
		return true
	default:
		return false
	}
}

// Precedence levels for Pratt parsing
const (
	_ int = iota
	PREC_LOWEST
	PREC_RELATIONAL   // > < >= <=
	PREC_SUM          // + -
	PREC_PREFIX_MINUS // unary - (lower than POWER, so -3^2 becomes -(3^2))
	PREC_PRODUCT      // * /
	PREC_POWER        // ^ (right-associative)
	PREC_POSTFIX      // !
	PREC_INDEX        // [ (array indexing / slicing)
)

var precedences = map[TokenType]int{
	TokenEQ:       PREC_RELATIONAL,
	TokenNEQ:      PREC_RELATIONAL,
	TokenAssign:   PREC_RELATIONAL,
	TokenGT:       PREC_RELATIONAL,
	TokenLT:       PREC_RELATIONAL,
	TokenGTE:      PREC_RELATIONAL,
	TokenLTE:      PREC_RELATIONAL,
	TokenPlus:     PREC_SUM,
	TokenMinus:    PREC_SUM,
	TokenAsterisk: PREC_PRODUCT,
	TokenSlash:    PREC_PRODUCT,
	TokenCaret:    PREC_POWER,
	TokenBang:     PREC_POSTFIX,
	TokenLBracket: PREC_INDEX,
	TokenLParen:   PREC_INDEX,
}

// Parser parses a token stream into an AST node.
type Parser struct {
	tokens   []Token
	pos      int
	curTok   Token
	peekTok  Token
	argDepth int
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
func Parse(input string) (ast.Node, error) {
	l := NewLexer(input)
	p := NewParser(l)

	if p.curTok.Type == TokenEOF {
		return nil, newSyntaxErr("parser.err_empty_expr")
	}

	node, err := p.ParseExpression(PREC_LOWEST)
	if err != nil {
		return nil, err
	}

	if p.peekTok.Type != TokenEOF {
		if p.isStartOfExpression(p.peekTok.Type) {
			return nil, newSyntaxErr("parser.err_implicit_multiplication", p.peekTok.Literal)
		}
		return nil, newSyntaxErr("parser.err_unexpected_token_pos", p.peekTok.Literal, p.peekTok.Pos)
	}

	return node, nil
}

// ParseStatement parses an input string as either an assignment statement (*ast.AssignStmt) or an expression (ast.Node).
func ParseStatement(input string) (interface{}, error) {
	l := NewLexer(input)
	p := NewParser(l)

	if p.curTok.Type == TokenEOF {
		return nil, fmt.Errorf("%s", i18n.T("parser.err_syntax_error_empty_input"))
	}

	// Check if this is an assignment: Ident = Expr
	if p.curTok.Type == TokenIdent && p.peekTok.Type == TokenAssign {
		varName := p.curTok.Literal
		if isReservedConst(varName) || isReservedFunc(varName) {
			return nil, fmt.Errorf("%s", i18n.T("errors.reserved_word", varName))
		}
		p.nextToken() // move to '='
		p.nextToken() // move to start of expression

		if p.curTok.Type == TokenEOF {
			return nil, newSyntaxErr("parser.err_missing_expr_after_assign")
		}

		expr, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}

		if p.peekTok.Type != TokenEOF {
			return nil, newSyntaxErr("parser.err_unexpected_token_after_expr", p.peekTok.Literal)
		}

		return &ast.AssignStmt{Name: varName, Value: expr}, nil
	}

	return Parse(input)
}

// ParseExpression parses an expression with Pratt parsing algorithm.
func (p *Parser) ParseExpression(precedence int) (ast.Node, error) {
	if p.curTok.Type == TokenIllegal {
		return nil, newSyntaxErr("parser.err_illegal_char_pos", p.curTok.Literal, p.curTok.Pos)
	}

	// 1. Prefix parse
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}

	// Check if next token starts another expression directly (implicit multiplication like 2pi or (1+2)(3+4))
	if err := p.checkImplicitMul(left, precedence); err != nil {
		return nil, err
	}

	// 2. Infix parse
	for p.peekTok.Type != TokenEOF && precedence < p.peekPrecedence() {
		p.nextToken()
		left, err = p.parseInfix(left)
		if err != nil {
			return nil, err
		}

		if err := p.checkImplicitMul(left, precedence); err != nil {
			return nil, err
		}
	}

	return left, nil
}

func (p *Parser) parsePrefix() (ast.Node, error) {
	switch p.curTok.Type {
	case TokenNumber:
		return p.curTok.RatVal, nil

	case TokenString:
		return ast.NewStringNode(p.curTok.Literal), nil

	case TokenIdent:
		name := p.curTok.Literal
		if p.peekTok.Type == TokenLParen {
			p.nextToken() // move to '('
			return p.parseFuncCall(name)
		}
		if isReservedFunc(name) && !isBareSymbolAllowed(name) {
			if p.argDepth == 0 {
				return nil, newSyntaxErr("parser.err_func_missing_args", p.curTok.Pos, name)
			}
		}
		// Constants: pi, e, or imaginary unit i
		if name == "i" {
			zero, _ := ast.NewRational(0, 1)
			one, _ := ast.NewRational(1, 1)
			return ast.NewComplex(zero, one), nil
		}
		if isReservedConst(name) {
			return ast.NewConst(name)
		}
		// Otherwise, it is a variable symbol (e.g., "x", "ans")
		return ast.NewVar(name), nil

	case TokenMinus:
		// Unary minus: use PREC_PREFIX_MINUS so that -3^2 parses as -(3^2)
		p.nextToken()
		expr, err := p.ParseExpression(PREC_PREFIX_MINUS)
		if err != nil {
			return nil, err
		}
		return ast.NewUnaryOp("-", expr)

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
			return nil, newSyntaxErr("parser.err_expected_rparen", p.peekTok.Literal)
		}
		p.nextToken() // move curTok to ')'
		return expr, nil

	case TokenLBracket:
		return p.parseBracketExpression()

	default:
		return nil, newSyntaxErr("parser.err_unexpected_token_pos", p.curTok.Literal, p.curTok.Pos)
	}
}

func (p *Parser) parseInfix(left ast.Node) (ast.Node, error) {
	switch p.curTok.Type {
	case TokenEQ, TokenNEQ, TokenAssign, TokenGT, TokenLT, TokenGTE, TokenLTE:
		op := p.curTok.Literal
		p.nextToken()
		right, err := p.ParseExpression(PREC_RELATIONAL)
		if err != nil {
			return nil, err
		}
		return ast.NewRelOp(left, op, right), nil

	case TokenPlus:
		p.nextToken()
		right, err := p.ParseExpression(PREC_SUM)
		if err != nil {
			return nil, err
		}
		return ast.NewAdd([]ast.Node{left, right}), nil

	case TokenMinus:
		p.nextToken()
		right, err := p.ParseExpression(PREC_SUM)
		if err != nil {
			return nil, err
		}
		negRight, err := ast.NewUnaryOp("-", right)
		if err != nil {
			return nil, err
		}
		return ast.NewAdd([]ast.Node{left, negRight}), nil

	case TokenAsterisk:
		p.nextToken()
		right, err := p.ParseExpression(PREC_PRODUCT)
		if err != nil {
			return nil, err
		}
		return ast.NewMul([]ast.Node{left, right}), nil

	case TokenSlash:
		p.nextToken()
		right, err := p.ParseExpression(PREC_PRODUCT)
		if err != nil {
			return nil, err
		}
		// Division: left / right -> Mul(left, Pow(right, -1))
		negOne, _ := ast.NewRational(-1, 1)
		reciprocal, err := ast.NewPow(right, negOne)
		if err != nil {
			return nil, err
		}
		return ast.NewMul([]ast.Node{left, reciprocal}), nil

	case TokenCaret:
		// Right associative: pass PREC_POWER - 1
		p.nextToken()
		right, err := p.ParseExpression(PREC_POWER - 1)
		if err != nil {
			return nil, err
		}
		return ast.NewPow(left, right)

	case TokenBang:
		// Postfix factorial: left!
		return ast.NewUnaryOp("!", left)

	case TokenLBracket:
		// Postfix index/slice: left[...]
		return p.parseBracketPostfix(left)

	case TokenLParen:
		// Postfix application: left(...)
		if isCallable(left) {
			return p.parseCallPostfix(left)
		}
		return nil, newSyntaxErr("parser.err_implicit_multiplication", p.curTok.Literal)

	default:
		return nil, newSyntaxErr("parser.err_unexpected_infix_token", p.curTok.Literal)
	}
}

func (p *Parser) parseBracketPostfix(left ast.Node) (ast.Node, error) {
	// curTok is TokenLBracket '['
	if p.peekTok.Type == TokenColon {
		// Slicing with omitted start (e.g., [:end], [::step], or [:])
		p.nextToken() // move to ':'
		if p.peekTok.Type == TokenRBracket {
			p.nextToken() // move to ']'
			return ast.NewSliceNode(left, nil, nil, nil), nil
		}
		if p.peekTok.Type == TokenColon {
			p.nextToken() // move to second ':'
			if p.peekTok.Type == TokenRBracket {
				p.nextToken() // move to ']'
				return ast.NewSliceNode(left, nil, nil, nil), nil
			}
			p.nextToken() // move to step expr
			step, err := p.ParseExpression(PREC_LOWEST)
			if err != nil {
				return nil, err
			}
			if p.peekTok.Type != TokenRBracket {
				return nil, newSyntaxErr("parser.err_expected_rbracket", p.peekTok.Literal)
			}
			p.nextToken() // move to ']'
			return ast.NewSliceNode(left, nil, nil, step), nil
		}

		// Has stop expression
		p.nextToken() // move to stop expr
		stop, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		if p.peekTok.Type == TokenRBracket {
			p.nextToken() // move to ']'
			return ast.NewSliceNode(left, nil, stop, nil), nil
		}
		if p.peekTok.Type == TokenColon {
			p.nextToken() // move to second ':'
			if p.peekTok.Type == TokenRBracket {
				p.nextToken() // move to ']'
				return ast.NewSliceNode(left, nil, stop, nil), nil
			}
			p.nextToken() // move to step expr
			step, err := p.ParseExpression(PREC_LOWEST)
			if err != nil {
				return nil, err
			}
			if p.peekTok.Type != TokenRBracket {
				return nil, newSyntaxErr("parser.err_expected_rbracket", p.peekTok.Literal)
			}
			p.nextToken() // move to ']'
			return ast.NewSliceNode(left, nil, stop, step), nil
		}
		return nil, newSyntaxErr("parser.err_expected_rbracket", p.peekTok.Literal)
	}

	if p.peekTok.Type == TokenRBracket {
		return nil, newSyntaxErr("parser.err_empty_index_bracket")
	}

	// First expression (either single index or slice start)
	p.nextToken() // move to start of first expression
	firstExpr, err := p.ParseExpression(PREC_LOWEST)
	if err != nil {
		return nil, err
	}

	// Single index lookup: left[firstExpr]
	if p.peekTok.Type == TokenRBracket {
		p.nextToken() // move to ']'
		return ast.NewIndexNode(left, firstExpr), nil
	}

	// If not ']', it must be ':' for slicing: left[firstExpr : ...]
	if p.peekTok.Type != TokenColon {
		return nil, newSyntaxErr("parser.err_expected_rbracket_or_colon", p.peekTok.Literal)
	}

	p.nextToken() // move to ':'
	start := firstExpr

	if p.peekTok.Type == TokenRBracket {
		p.nextToken() // move to ']'
		return ast.NewSliceNode(left, start, nil, nil), nil
	}
	if p.peekTok.Type == TokenColon {
		// left[start::step]
		p.nextToken() // move to second ':'
		if p.peekTok.Type == TokenRBracket {
			p.nextToken() // move to ']'
			return ast.NewSliceNode(left, start, nil, nil), nil
		}
		p.nextToken() // move to step expr
		step, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		if p.peekTok.Type != TokenRBracket {
			return nil, newSyntaxErr("parser.err_expected_rbracket", p.peekTok.Literal)
		}
		p.nextToken() // move to ']'
		return ast.NewSliceNode(left, start, nil, step), nil
	}

	// Has stop expression: left[start:stop...]
	p.nextToken() // move to stop expr
	stop, err := p.ParseExpression(PREC_LOWEST)
	if err != nil {
		return nil, err
	}
	if p.peekTok.Type == TokenRBracket {
		p.nextToken() // move to ']'
		return ast.NewSliceNode(left, start, stop, nil), nil
	}
	if p.peekTok.Type == TokenColon {
		// left[start:stop:step]
		p.nextToken() // move to second ':'
		if p.peekTok.Type == TokenRBracket {
			p.nextToken() // move to ']'
			return ast.NewSliceNode(left, start, stop, nil), nil
		}
		p.nextToken() // move to step expr
		step, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		if p.peekTok.Type != TokenRBracket {
			return nil, newSyntaxErr("parser.err_expected_rbracket", p.peekTok.Literal)
		}
		p.nextToken() // move to ']'
		return ast.NewSliceNode(left, start, stop, step), nil
	}

	return nil, newSyntaxErr("parser.err_expected_rbracket", p.peekTok.Literal)
}

func (p *Parser) parseFuncCall(name string) (ast.Node, error) {
	if FuncLookup != nil {
		if cName, _, _, ok := FuncLookup(name); ok {
			name = cName
		}
	}

	// curTok is '('
	if p.peekTok.Type == TokenRParen {
		if FuncLookup != nil {
			if _, minArgs, _, ok := FuncLookup(name); ok && minArgs == 0 {
				p.nextToken() // move to ')'
				return ast.NewFunc(name, nil)
			}
		}
		return nil, newSyntaxErr("parser.err_func_requires_args", name)
	}

	p.nextToken() // move to first arg

	p.argDepth++
	var args []ast.Node
	for {
		arg, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			p.argDepth--
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
		p.argDepth--
		return nil, newSyntaxErr("parser.err_expected_comma_or_rparen", name, p.peekTok.Literal)
	}
	p.argDepth--

	if name == "sqrt" {
		if len(args) != 1 {
			return nil, fmt.Errorf("%s", i18n.T("parser.err_sqrt_requires_exactly_1_argument", len(args)))
		}
		return ast.NewSqrt(args[0]), nil
	}

	if name == "forall" || name == "exists" {
		if len(args) != 2 {
			return nil, fmt.Errorf("%s", i18n.T("parser.err_variables_formula_got", name, len(args)))
		}
		var vars []string
		if list, ok := args[0].(*ast.ListNode); ok {
			for _, elem := range list.Elements {
				if v, ok := elem.(*ast.VarNode); ok {
					vars = append(vars, v.Name)
				} else {
					return nil, fmt.Errorf("%s", i18n.T("parser.err_first_argument_must_be_a", name, elem))
				}
			}
		} else if v, ok := args[0].(*ast.VarNode); ok {
			vars = append(vars, v.Name)
		} else {
			return nil, fmt.Errorf("%s", i18n.T("parser.err_first_argument_must_be_a_1", name, args[0]))
		}
		kind := ast.QuantifierForall
		if name == "exists" {
			kind = ast.QuantifierExists
		}
		return ast.NewQuantifier(kind, vars, args[1]), nil
	}

	if name == "lambda" {
		if len(args) != 2 {
			return nil, fmt.Errorf("%s", i18n.T("errors.lambda_param_count_mismatch", 2, len(args)))
		}
		var params []string
		if list, ok := args[0].(*ast.ListNode); ok {
			for _, elem := range list.Elements {
				if v, ok := elem.(*ast.VarNode); ok {
					params = append(params, v.Name)
				} else {
					return nil, fmt.Errorf("%s", i18n.T("errors.lambda_param_must_be_ident", elem.String()))
				}
			}
		} else if v, ok := args[0].(*ast.VarNode); ok {
			params = append(params, v.Name)
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.lambda_param_must_be_ident", args[0].String()))
		}
		return ast.NewLambdaNode(params, args[1]), nil
	}

	return ast.NewFunc(name, args)
}

func (p *Parser) isStartOfExpression(t TokenType) bool {
	switch t {
	case TokenNumber, TokenIdent, TokenLParen, TokenLBracket, TokenString:
		return true
	default:
		return false
	}
}

func (p *Parser) parseBracketExpression() (ast.Node, error) {
	// curTok is TokenLBracket '['
	if p.peekTok.Type == TokenRBracket {
		p.nextToken() // move to ']'
		return ast.NewList([]ast.Node{}), nil
	}

	if p.peekTok.Type == TokenLBracket {
		// Matrix literal: [[row1], [row2], ...]
		var matrixData [][]ast.Node
		for {
			if p.peekTok.Type != TokenLBracket {
				return nil, newSyntaxErr("parser.err_expected_lbracket_matrix_row", p.peekTok.Literal)
			}
			p.nextToken() // move to '[' of row

			row, err := p.parseBracketList()
			if err != nil {
				return nil, err
			}
			matrixData = append(matrixData, row)

			if p.peekTok.Type == TokenComma {
				p.nextToken() // consume ','
				continue
			}
			break
		}

		if p.peekTok.Type != TokenRBracket {
			return nil, newSyntaxErr("parser.err_expected_rbracket_close_matrix", p.peekTok.Literal)
		}
		p.nextToken() // move to ']' of matrix

		rows := len(matrixData)
		if rows == 0 {
			return nil, newSyntaxErr("parser.err_empty_matrix")
		}
		cols := len(matrixData[0])
		return ast.NewMatrix(rows, cols, matrixData)
	}

	// 1D list: [elem1, elem2, ...]
	elems, err := p.parseBracketList()
	if err != nil {
		return nil, err
	}
	return ast.NewList(elems), nil
}

func (p *Parser) parseBracketList() ([]ast.Node, error) {
	var elems []ast.Node

	if p.peekTok.Type == TokenRBracket {
		p.nextToken() // move to ']'
		return elems, nil
	}

	p.nextToken() // move to first expression
	first, err := p.ParseExpression(PREC_LOWEST)
	if err != nil {
		return nil, err
	}
	elems = append(elems, first)

	for p.peekTok.Type == TokenComma {
		p.nextToken() // move to ','
		p.nextToken() // move to next expression
		next, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		elems = append(elems, next)
	}

	if p.peekTok.Type != TokenRBracket {
		return nil, newSyntaxErr("parser.err_expected_rbracket", p.peekTok.Literal)
	}
	p.nextToken() // move to ']'
	return elems, nil
}

func isCallable(n ast.Node) bool {
	switch n.(type) {
	case *ast.LambdaNode, *ast.ApplyNode:
		return true
	default:
		return false
	}
}

func (p *Parser) checkImplicitMul(left ast.Node, precedence int) error {
	if p.isStartOfExpression(p.peekTok.Type) {
		if p.peekTok.Type == TokenLBracket && precedence < PREC_INDEX {
			return nil
		}
		if p.peekTok.Type == TokenLParen && isCallable(left) && precedence < PREC_INDEX {
			return nil
		}
		return newSyntaxErr("parser.err_implicit_multiplication", p.peekTok.Literal)
	}
	return nil
}

func (p *Parser) parseCallPostfix(left ast.Node) (ast.Node, error) {
	// curTok is TokenLParen '('
	if p.peekTok.Type == TokenRParen {
		p.nextToken() // move to ')'
		return ast.NewApplyNode(left, []ast.Node{}), nil
	}

	p.nextToken() // move to first arg
	p.argDepth++
	var args []ast.Node
	for {
		arg, err := p.ParseExpression(PREC_LOWEST)
		if err != nil {
			p.argDepth--
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
		p.argDepth--
		return nil, newSyntaxErr("parser.err_expected_comma_or_rparen", "call", p.peekTok.Literal)
	}
	p.argDepth--
	return ast.NewApplyNode(left, args), nil
}

