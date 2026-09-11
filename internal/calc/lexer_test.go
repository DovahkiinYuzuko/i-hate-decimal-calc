package calc

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
	// 6.441 must be converted to 6441/1000 without using float64
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

	// 0.005 -> 5/1000 -> 1/200 (auto-reduced by big.Rat)
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

	// Syntax errors in repeating decimals
	errorCases := []string{
		"0.()",
		"0.(3",
	}
	for _, ec := range errorCases {
		tokens, err := Tokenize(ec)
		// Either Tokenize returns an error, or the tokens contain TokenIllegal
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

