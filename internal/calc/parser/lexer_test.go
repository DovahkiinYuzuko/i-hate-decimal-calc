package parser

import (
	"testing"
)

func TestLexer_BasicTokens(t *testing.T) {
	input := "123 + 45 * 67 / 89 ^ 2! - (3, 4)"
	tokens, err := Tokenize(input)
	if err != nil {
		t.Fatalf("unexpected error tokenizing: %v", err)
	}

	expectedTypes := []TokenType{
		TokenNumber, TokenPlus, TokenNumber, TokenAsterisk, TokenNumber,
		TokenSlash, TokenNumber, TokenCaret, TokenNumber, TokenBang,
		TokenMinus, TokenLParen, TokenNumber, TokenComma, TokenNumber, TokenRParen,
		TokenEOF,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	for i, expected := range expectedTypes {
		if tokens[i].Type != expected {
			t.Errorf("token %d: expected type %v, got %v (literal: %q)", i, expected, tokens[i].Type, tokens[i].Literal)
		}
	}
}

func TestLexer_DecimalToFraction(t *testing.T) {
	input := "6.441"
	tokens, err := Tokenize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 2 || tokens[0].Type != TokenNumber {
		t.Fatalf("expected number token, got %+v", tokens)
	}
	expected := "6441/1000"
	if tokens[0].RatVal.String() != expected {
		t.Errorf("expected %s, got %s", expected, tokens[0].RatVal.String())
	}

	tokens2, err := Tokenize("0.005")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens2[0].RatVal.String() != "1/200" {
		t.Errorf("expected 1/200, got %s", tokens2[0].RatVal.String())
	}
}

func TestLexer_IdentifiersAndAliases(t *testing.T) {
	input := "pi π e sqrt √ sin cos tan log ln"
	tokens, err := Tokenize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLiterals := []string{"pi", "pi", "e", "sqrt", "sqrt", "sin", "cos", "tan", "log", "ln"}
	for i, lit := range expectedLiterals {
		if tokens[i].Type != TokenIdent {
			t.Errorf("token %d: expected TokenIdent, got %v", i, tokens[i].Type)
		}
		if tokens[i].Literal != lit {
			t.Errorf("token %d: expected literal %q, got %q", i, lit, tokens[i].Literal)
		}
	}
}

func TestLexer_IllegalCharacter_Error(t *testing.T) {
	input := "123 @ 456"
	_, err := Tokenize(input)
	if err == nil {
		t.Fatal("expected error on illegal character '@', got nil")
	}
}

func TestLexer_RepeatingDecimals(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"0.(3)", "1/3"},
		{"0.(9)", "1"},
		{"0.1(6)", "1/6"},
		{"1.2(34)", "611/495"},
		{"0.(142857)", "1/7"},
	}

	for _, tc := range testCases {
		tokens, err := Tokenize(tc.input)
		if err != nil {
			t.Fatalf("unexpected error tokenizing %q: %v", tc.input, err)
		}
		if len(tokens) != 2 || tokens[0].Type != TokenNumber {
			t.Fatalf("expected 1 number token for %q, got %+v", tc.input, tokens)
		}
		if actual := tokens[0].RatVal.String(); actual != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, actual)
		}
	}

	errorCases := []string{
		"0.()",
		"0.(3",
	}
	for _, ec := range errorCases {
		tokens, err := Tokenize(ec)
		hasIllegal := false
		if err != nil {
			hasIllegal = true
		} else {
			for _, tok := range tokens {
				if tok.Type == TokenIllegal {
					hasIllegal = true
					break
				}
			}
		}
		if !hasIllegal {
			t.Errorf("expected error or TokenIllegal for invalid repeating decimal %q, got tokens: %+v", ec, tokens)
		}
	}
}

func TestLexer_NormalizeZenkaku(t *testing.T) {
	testCases := []struct {
		input    string
		expected []TokenType
	}{
		{"１２３", []TokenType{TokenNumber, TokenEOF}},
		{"（１＋２）＊３＾２", []TokenType{TokenLParen, TokenNumber, TokenPlus, TokenNumber, TokenRParen, TokenAsterisk, TokenNumber, TokenCaret, TokenNumber, TokenEOF}},
		{"４　ー　２　÷　１", []TokenType{TokenNumber, TokenMinus, TokenNumber, TokenSlash, TokenNumber, TokenEOF}},
		{"ｘ　＝　１／２", []TokenType{TokenIdent, TokenAssign, TokenNumber, TokenSlash, TokenNumber, TokenEOF}},
	}

	for _, tc := range testCases {
		tokens, err := Tokenize(tc.input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.input, err)
		}
		if len(tokens) != len(tc.expected) {
			t.Fatalf("input %q: expected %d tokens, got %d", tc.input, len(tc.expected), len(tokens))
		}
		for i, exp := range tc.expected {
			if tokens[i].Type != exp {
				t.Errorf("input %q token %d: expected type %v, got %v (%q)", tc.input, i, exp, tokens[i].Type, tokens[i].Literal)
			}
		}
	}
}

func TestLexer_NotEqualAndFactorial(t *testing.T) {
	// 1!=2 -> [TokenNumber(1), TokenNEQ("!="), TokenNumber(2), TokenEOF]
	tokens, err := Tokenize("1!=2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []TokenType{TokenNumber, TokenNEQ, TokenNumber, TokenEOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Type != exp {
			t.Errorf("token %d: expected %v, got %v (%q)", i, exp, tokens[i].Type, tokens[i].Literal)
		}
	}

	// 5! != 120 -> [TokenNumber(5), TokenBang("!"), TokenNEQ("!="), TokenNumber(120), TokenEOF]
	tokensFact, err := Tokenize("5! != 120")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedFact := []TokenType{TokenNumber, TokenBang, TokenNEQ, TokenNumber, TokenEOF}
	if len(tokensFact) != len(expectedFact) {
		t.Fatalf("expected %d tokens, got %d", len(expectedFact), len(tokensFact))
	}
	for i, exp := range expectedFact {
		if tokensFact[i].Type != exp {
			t.Errorf("token %d: expected %v, got %v (%q)", i, exp, tokensFact[i].Type, tokensFact[i].Literal)
		}
	}
}

func TestLexer_MultiRadixLiterals(t *testing.T) {
	testCases := []struct {
		input       string
		expectedVal string
	}{
		// Binary (0b/0B)
		{"0b1011", "11"},
		{"0B1101", "13"},
		{"0b0", "0"},

		// Octal (0o/0O)
		{"0o755", "493"},
		{"0O10", "8"},

		// Hexadecimal (0x/0X)
		{"0x1f", "31"},
		{"0xFF", "255"},
		{"0x0", "0"},

		// Mathematica base^^digits
		{"2^^101", "5"},
		{"8^^77", "63"},
		{"16^^ff", "255"},
		{"16^^FF", "255"},
		{"36^^z", "35"},
		{"36^^10", "36"},
		{"62^^Z", "61"},
		{"62^^10", "62"},
		{"64^^@_", "4031"},
		{"64^^10", "64"},
	}

	for _, tc := range testCases {
		tokens, err := Tokenize(tc.input)
		if err != nil {
			t.Fatalf("unexpected error tokenizing %q: %v", tc.input, err)
		}
		if len(tokens) != 2 || tokens[0].Type != TokenNumber {
			t.Fatalf("input %q: expected [TokenNumber, TokenEOF], got %+v", tc.input, tokens)
		}
		got := tokens[0].RatVal.String()
		if got != tc.expectedVal {
			t.Errorf("input %q: expected value %s, got %s", tc.input, tc.expectedVal, got)
		}
	}

	// Mixed arithmetic expression
	tokensExpr, err := Tokenize("0x10 + 0b11")
	if err != nil {
		t.Fatalf("unexpected error tokenizing '0x10 + 0b11': %v", err)
	}
	if len(tokensExpr) != 4 || tokensExpr[0].RatVal.String() != "16" || tokensExpr[2].RatVal.String() != "3" {
		t.Fatalf("unexpected tokens for '0x10 + 0b11': %+v", tokensExpr)
	}
}

