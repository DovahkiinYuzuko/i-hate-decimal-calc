package main

import (
	"reflect"
	"testing"
)

func TestLexerHighlightFSM_RainbowBrackets(t *testing.T) {
	input := "((((x))))"
	spans := TokenizeHighlightFSM(input)

	// Expected depths:
	// pos 0: '(', depth 0
	// pos 1: '(', depth 1
	// pos 2: '(', depth 2
	// pos 3: '(', depth 3
	// pos 5: ')', depth 3
	// pos 6: ')', depth 2
	// pos 7: ')', depth 1
	// pos 8: ')', depth 0
	bracketDepths := make(map[int]int)
	for _, s := range spans {
		if s.Type == TokenRainbow {
			bracketDepths[s.Start] = s.Depth
		}
	}

	expected := map[int]int{
		0: 0, 1: 1, 2: 2, 3: 3,
		5: 3, 6: 2, 7: 1, 8: 0,
	}

	for pos, expDepth := range expected {
		if got, ok := bracketDepths[pos]; !ok || got != expDepth {
			t.Errorf("pos %d: expected depth %d, got %d (found=%v)", pos, expDepth, got, ok)
		}
	}
}

func TestLexerHighlightFSM_MixedBrackets(t *testing.T) {
	input := "([ { x } ])"
	spans := TokenizeHighlightFSM(input)

	bracketDepths := make(map[int]int)
	for _, s := range spans {
		if s.Type == TokenRainbow {
			bracketDepths[s.Start] = s.Depth
		}
	}

	// 0: '(', 1: '[', 3: '{'
	// 7: '}', 9: ']', 10: ')'
	expected := map[int]int{
		0: 0, 1: 1, 3: 2,
		7: 2, 9: 1, 10: 0,
	}

	for pos, expDepth := range expected {
		if got, ok := bracketDepths[pos]; !ok || got != expDepth {
			t.Errorf("pos %d: expected depth %d, got %d", pos, expDepth, got)
		}
	}
}

func TestLexerHighlightFSM_StringExclusionAndEscapes(t *testing.T) {
	input := `"hello (world) [1]" + "escaped \"(paren)\""`
	spans := TokenizeHighlightFSM(input)

	// None of the brackets inside quotes should be TokenRainbow
	for _, s := range spans {
		if s.Type == TokenRainbow {
			t.Errorf("unexpected rainbow bracket inside string: %v", s)
		}
	}

	// Should have two TokenString spans
	var strCount int
	for _, s := range spans {
		if s.Type == TokenString {
			strCount++
		}
	}
	if strCount != 2 {
		t.Errorf("expected 2 string tokens, got %d", strCount)
	}
}

func TestLexerHighlightFSM_BracketMismatch(t *testing.T) {
	input := "(1 + 2] + )"
	spans := TokenizeHighlightFSM(input)

	var mismatches []int
	for _, s := range spans {
		if s.Type == TokenMismatch {
			mismatches = append(mismatches, s.Start)
		}
	}

	// ']' at pos 6 is mismatch for '('
	// ')' at pos 10 is unmatched
	expected := []int{6, 10}
	if !reflect.DeepEqual(mismatches, expected) {
		t.Errorf("expected mismatch positions %v, got %v", expected, mismatches)
	}
}

func TestLexerHighlightFSM_Tokens(t *testing.T) {
	input := "sin(16^^FF + 3.14 * x) == vars"
	spans := TokenizeHighlightFSM(input)

	type check struct {
		start, end int
		tokenType  TokenType
	}

	expectedChecks := []check{
		{0, 3, TokenFunc},     // sin
		{4, 10, TokenNumber},  // 16^^FF
		{11, 12, TokenOp},     // +
		{13, 17, TokenNumber}, // 3.14
		{18, 19, TokenOp},     // *
		{23, 25, TokenOp},     // ==
		{26, 30, TokenCmd},    // vars
	}

	for _, exp := range expectedChecks {
		found := false
		for _, s := range spans {
			if s.Start == exp.start && s.End == exp.end && s.Type == exp.tokenType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected token %v at [%d:%d], not found in spans: %v", exp.tokenType, exp.start, exp.end, spans)
		}
	}
}

func TestRainbowMatcher_FindAllStringIndex(t *testing.T) {
	matcherDepth0 := &RainbowMatcher{TargetDepth: 0}
	matcherMismatch := &RainbowMatcher{TargetDepth: -1}

	input := "sin(cos(x)) + ]"
	// Depth 0 brackets: '(' at 3, ')' at 10
	idx0 := matcherDepth0.FindAllStringIndex(input, -1)
	expected0 := [][]int{{3, 4}, {10, 11}}
	if !reflect.DeepEqual(idx0, expected0) {
		t.Errorf("matcher depth 0: expected %v, got %v", expected0, idx0)
	}

	// Mismatched bracket: ']' at 14
	idxErr := matcherMismatch.FindAllStringIndex(input, -1)
	expectedErr := [][]int{{14, 15}}
	if !reflect.DeepEqual(idxErr, expectedErr) {
		t.Errorf("matcher mismatch: expected %v, got %v", expectedErr, idxErr)
	}
}

func TestBuildREPLHighlights_ThemeSequences(t *testing.T) {
	darkHighlights := BuildREPLHighlights("dark")
	if len(darkHighlights) == 0 {
		t.Fatalf("expected non-empty dark highlights")
	}

	lightHighlights := BuildREPLHighlights("light")
	if len(lightHighlights) == 0 {
		t.Fatalf("expected non-empty light highlights")
	}

	noneHighlights := BuildREPLHighlights("none")
	if len(noneHighlights) != 0 {
		t.Errorf("expected empty highlights for theme 'none', got %d", len(noneHighlights))
	}
}
