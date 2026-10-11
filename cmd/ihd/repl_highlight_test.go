package main

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

func TestLexerHighlightFSM_RainbowBrackets(t *testing.T) {
	input := "((((x))))"
	spans := TokenizeHighlightFSM(input, nil)

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

func TestLexerHighlightFSM_ParenthesesAndSquareBracketsOnly(t *testing.T) {
	// () and [] are supported delimiters in CAS.
	// {} is NOT a valid CAS mathematical delimiter and MUST NOT be treated as a rainbow bracket.
	input := "([ x ]) { y }"
	spans := TokenizeHighlightFSM(input, nil)

	bracketDepths := make(map[int]int)
	for _, s := range spans {
		if s.Type == TokenRainbow {
			bracketDepths[s.Start] = s.Depth
		}
	}

	// 0: '(', 1: '[', 5: ']', 6: ')'
	expected := map[int]int{
		0: 0, 1: 1,
		5: 1, 6: 0,
	}

	for pos, expDepth := range expected {
		if got, ok := bracketDepths[pos]; !ok || got != expDepth {
			t.Errorf("pos %d: expected depth %d, got %d", pos, expDepth, got)
		}
	}

	// '{' at 8 and '}' at 12 must NOT be in bracketDepths
	if _, ok := bracketDepths[8]; ok {
		t.Errorf("pos 8 '{' must not be recognized as TokenRainbow")
	}
	if _, ok := bracketDepths[12]; ok {
		t.Errorf("pos 12 '}' must not be recognized as TokenRainbow")
	}
}

func TestLexerHighlightFSM_StringExclusionAndEscapes(t *testing.T) {
	input := `"hello (world) [1]" + "escaped \"(paren)\""`
	spans := TokenizeHighlightFSM(input, nil)

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
	spans := TokenizeHighlightFSM(input, nil)

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

func TestLexerHighlightFSM_TokensAndAllReservedFunctions(t *testing.T) {
	// Verify that functions beyond the previous hardcoded 33 (e.g. clifford, groebner)
	// are accurately identified as TokenFunc via calc.IsReservedFunc
	input := "groebner(clifford(16^^FF + 3.14 * x)) == vars"
	spans := TokenizeHighlightFSM(input, nil)

	type check struct {
		start, end int
		tokenType  TokenType
	}

	expectedChecks := []check{
		{0, 8, TokenFunc},     // groebner
		{9, 17, TokenFunc},    // clifford
		{18, 24, TokenNumber}, // 16^^FF
		{25, 26, TokenOp},     // +
		{27, 31, TokenNumber}, // 3.14
		{32, 33, TokenOp},     // *
		{38, 40, TokenOp},     // ==
		{41, 45, TokenCmd},    // vars
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

func TestLexerHighlightFSM_CustomVariableHybridHighlight(t *testing.T) {
	env := calc.NewEnv()
	env.Set("my_var", &calc.RationalNode{Val: big.NewRat(42, 1)})

	// In "sin(my_var + x)":
	// - sin: TokenFunc
	// - my_var: TokenCustomVar (because it exists in env)
	// - x: TokenNone (unbound algebraic symbol remains uncolored default text)
	input := "sin(my_var + x)"
	spans := TokenizeHighlightFSM(input, env)

	var hasCustomVar, hasUnboundVar bool
	for _, s := range spans {
		if s.Start == 4 && s.End == 10 { // my_var
			if s.Type == TokenCustomVar {
				hasCustomVar = true
			} else {
				t.Errorf("expected my_var to be TokenCustomVar, got %v", s.Type)
			}
		}
		if s.Start == 13 && s.End == 14 { // x
			hasUnboundVar = true
			t.Errorf("unbound symbol x must remain TokenNone, but found token span: %v", s)
		}
	}

	if !hasCustomVar {
		t.Errorf("expected TokenCustomVar for defined variable 'my_var', but not found")
	}
	if hasUnboundVar {
		t.Errorf("unbound symbol x should not produce a colored token span")
	}
}

func TestFSMHighlightMatcher_FindAllStringIndex(t *testing.T) {
	env := calc.NewEnv()
	env.Set("target_var", &calc.RationalNode{Val: big.NewRat(100, 1)})

	matcherDepth0 := &FSMHighlightMatcher{TargetType: TokenRainbow, TargetDepth: 0, Env: env}
	matcherMismatch := &FSMHighlightMatcher{TargetType: TokenMismatch, TargetDepth: -1, Env: env}
	matcherFunc := &FSMHighlightMatcher{TargetType: TokenFunc, Env: env}
	matcherNum := &FSMHighlightMatcher{TargetType: TokenNumber, Env: env}
	matcherCustom := &FSMHighlightMatcher{TargetType: TokenCustomVar, Env: env}

	input := "groebner(cos(114514 + target_var + x)) + ]"
	// Depth 0 brackets: '(' at 8, ')' at 37
	idx0 := matcherDepth0.FindAllStringIndex(input, -1)
	expected0 := [][]int{{8, 9}, {37, 38}}
	if !reflect.DeepEqual(idx0, expected0) {
		t.Errorf("matcher depth 0: expected %v, got %v", expected0, idx0)
	}

	// Functions: 'groebner' at 0..8, 'cos' at 9..12
	idxFunc := matcherFunc.FindAllStringIndex(input, -1)
	expectedFunc := [][]int{{0, 8}, {9, 12}}
	if !reflect.DeepEqual(idxFunc, expectedFunc) {
		t.Errorf("matcher func: expected %v, got %v", expectedFunc, idxFunc)
	}

	// Numbers: '114514' at 13..19
	idxNum := matcherNum.FindAllStringIndex(input, -1)
	expectedNum := [][]int{{13, 19}}
	if !reflect.DeepEqual(idxNum, expectedNum) {
		t.Errorf("matcher num: expected %v, got %v", expectedNum, idxNum)
	}

	// Custom variable: 'target_var' at 22..32
	idxCustom := matcherCustom.FindAllStringIndex(input, -1)
	expectedCustom := [][]int{{22, 32}}
	if !reflect.DeepEqual(idxCustom, expectedCustom) {
		t.Errorf("matcher custom var: expected %v, got %v", expectedCustom, idxCustom)
	}

	// Mismatched bracket: ']' at 41..42
	idxErr := matcherMismatch.FindAllStringIndex(input, -1)
	expectedErr := [][]int{{41, 42}}
	if !reflect.DeepEqual(idxErr, expectedErr) {
		t.Errorf("matcher mismatch: expected %v, got %v", expectedErr, idxErr)
	}
}

func TestBuildREPLHighlights_ThemeSequencesAndColorSeparation(t *testing.T) {
	env := calc.NewEnv()
	darkHighlights := BuildREPLHighlights("dark", env)
	if len(darkHighlights) == 0 {
		t.Fatalf("expected non-empty dark highlights")
	}

	// Ensure Bracket Depth 0 color and Number color are strictly different
	var darkDepth0Seq, darkNumSeq, darkCustomVarSeq string
	for _, h := range darkHighlights {
		if m, ok := h.Pattern.(*FSMHighlightMatcher); ok {
			if m.TargetType == TokenRainbow && m.TargetDepth == 0 {
				darkDepth0Seq = h.Sequence
			}
			if m.TargetType == TokenNumber {
				darkNumSeq = h.Sequence
			}
			if m.TargetType == TokenCustomVar {
				darkCustomVarSeq = h.Sequence
			}
		}
	}
	if darkDepth0Seq == "" || darkNumSeq == "" || darkCustomVarSeq == "" {
		t.Fatalf("expected depth 0, number, and custom var highlights configured, got depth0=%q, num=%q, var=%q", darkDepth0Seq, darkNumSeq, darkCustomVarSeq)
	}
	if darkDepth0Seq == darkNumSeq {
		t.Errorf("bracket depth 0 color (%q) MUST NOT match number color (%q)", darkDepth0Seq, darkNumSeq)
	}

	// Operators must not use dim gray \x1B[90m which is unreadable on transparent dark terminals
	for _, h := range darkHighlights {
		if m, ok := h.Pattern.(*FSMHighlightMatcher); ok && m.TargetType == TokenOp {
			if strings.Contains(h.Sequence, "90m") {
				t.Errorf("dark theme operator sequence should avoid unreadable dim gray 90m: got %q", h.Sequence)
			}
		}
	}

	lightHighlights := BuildREPLHighlights("light", env)
	if len(lightHighlights) == 0 {
		t.Fatalf("expected non-empty light highlights")
	}

	noneHighlights := BuildREPLHighlights("none", env)
	if len(noneHighlights) != 0 {
		t.Errorf("expected empty highlights for theme 'none', got %d", len(noneHighlights))
	}
}
