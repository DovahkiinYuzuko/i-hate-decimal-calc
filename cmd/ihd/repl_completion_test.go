package main

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

func TestExtractWordBeforeCursor(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		cursor    int
		wantWord  string
		wantStart int
	}{
		{
			name:      "simple identifier at start",
			line:      "diff",
			cursor:    4,
			wantWord:  "diff",
			wantStart: 0,
		},
		{
			name:      "single letter d",
			line:      "d",
			cursor:    1,
			wantWord:  "d",
			wantStart: 0,
		},
		{
			name:      "after operator",
			line:      "1 + di",
			cursor:    6,
			wantWord:  "di",
			wantStart: 4,
		},
		{
			name:      "after open parenthesis",
			line:      "solve(di",
			cursor:    8,
			wantWord:  "di",
			wantStart: 6,
		},
		{
			name:      "after comma and space",
			line:      "diff(x, di",
			cursor:    10,
			wantWord:  "di",
			wantStart: 8,
		},
		{
			name:      "cursor at space",
			line:      "diff(x, ",
			cursor:    8,
			wantWord:  "",
			wantStart: 8,
		},
		{
			name:      "empty line",
			line:      "",
			cursor:    0,
			wantWord:  "",
			wantStart: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			word, start := ExtractWordBeforeCursor(tt.line, tt.cursor)
			if word != tt.wantWord {
				t.Errorf("word = %q, want %q", word, tt.wantWord)
			}
			if start != tt.wantStart {
				t.Errorf("start = %d, want %d", start, tt.wantStart)
			}
		})
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{input: []string{"diff", "digit_count", "div"}, want: "di"},
		{input: []string{"deg", "degree"}, want: "deg"},
		{input: []string{"diff"}, want: "diff"},
		{input: []string{}, want: ""},
		{input: []string{"abc", "xyz"}, want: ""},
	}

	for _, tt := range tests {
		got := LongestCommonPrefix(tt.input)
		if got != tt.want {
			t.Errorf("LongestCommonPrefix(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFindCompletionCandidates(t *testing.T) {
	env := calc.NewEnv()
	env.Set("my_custom_var", &calc.RationalNode{Val: big.NewRat(42, 1)})

	t.Run("empty prefix returns nothing", func(t *testing.T) {
		res := FindCompletionCandidates("", env)
		if len(res) != 0 {
			t.Errorf("expected 0 candidates, got %d", len(res))
		}
	})

	t.Run("prefix d finds functions", func(t *testing.T) {
		res := FindCompletionCandidates("d", env)
		foundDiff := false
		foundDet := false
		for _, name := range res {
			if name == "diff" {
				foundDiff = true
			}
			if name == "det" {
				foundDet = true
			}
		}
		if !foundDiff || !foundDet {
			t.Errorf("expected 'diff' and 'det' in candidates, got %v", res)
		}
	})

	t.Run("prefix dif finds diff", func(t *testing.T) {
		res := FindCompletionCandidates("dif", env)
		if len(res) != 1 || res[0] != "diff" {
			t.Errorf("expected ['diff'], got %v", res)
		}
	})

	t.Run("constant pi", func(t *testing.T) {
		res := FindCompletionCandidates("p", env)
		foundPi := false
		for _, name := range res {
			if name == "pi" {
				foundPi = true
			}
		}
		if !foundPi {
			t.Errorf("expected 'pi' in candidates, got %v", res)
		}
	})

	t.Run("env custom variable", func(t *testing.T) {
		res := FindCompletionCandidates("my_c", env)
		if len(res) != 1 || res[0] != "my_custom_var" {
			t.Errorf("expected ['my_custom_var'], got %v", res)
		}
	})
}

func TestCompletionFSM_Step(t *testing.T) {
	env := calc.NewEnv()
	fsm := NewCompletionFSM(env)

	// First Tab on "dif": single match "diff", immediate completion
	replacement, ok := fsm.Step("dif", 0, 3)
	if !ok || replacement != "diff" {
		t.Errorf("step 1: expected 'diff', got %q", replacement)
	}

	// First Tab on "d": multiple matches (deg, delta, det, diff, etc.)
	fsm.Reset()
	replacement, ok = fsm.Step("d", 0, 1)
	if !ok {
		t.Fatalf("step 1 for 'd' failed")
	}
	// Common prefix of "d" matches is "d", so cycling starts with first candidate
	firstCand := replacement

	// Next Tab at the same position cycles to second candidate
	nextCand, ok := fsm.Step(firstCand, 0, len(firstCand))
	if !ok || nextCand == firstCand {
		t.Errorf("step 2 should cycle to distinct candidate, got %q", nextCand)
	}
}
