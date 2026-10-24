package main

import (
	"errors"
	"strings"
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1, s2 string
		want   int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"solve", "solv", 1},
		{"integrate", "integrat", 1},
		{"groebner", "grobner", 1},
	}
	for _, tc := range tests {
		got := LevenshteinDistance(tc.s1, tc.s2)
		if got != tc.want {
			t.Errorf("LevenshteinDistance(%q, %q) = %d; want %d", tc.s1, tc.s2, got, tc.want)
		}
	}
}

func TestFindClosestFunction(t *testing.T) {
	tests := []struct {
		input    string
		maxDist  int
		wantName string
	}{
		{"integrat", 2, "integrate"},
		{"solv", 1, "solve"},
		{"grobner", 2, "groebner"},
		{"completely_bogus_xyz_name", 2, ""},
	}
	for _, tc := range tests {
		got := FindClosestFunction(tc.input, tc.maxDist)
		if got != tc.wantName {
			t.Errorf("FindClosestFunction(%q, %d) = %q; want %q", tc.input, tc.maxDist, got, tc.wantName)
		}
	}
}

func TestFormatErrorWithDiagnostics(t *testing.T) {
	t.Run("did you mean for unknown function", func(t *testing.T) {
		err := errors.New("unknown function: integrat")
		line := "integrat(x^2, x)"
		got := FormatErrorWithDiagnostics(err, line)
		if !strings.Contains(got, "integrate") {
			t.Errorf("expected suggestion 'integrate' in formatted error, got: %s", got)
		}
	})

	t.Run("suppress false positive for single letter custom var", func(t *testing.T) {
		err := errors.New("unknown function: f")
		line := "f(x)"
		got := FormatErrorWithDiagnostics(err, line)
		if strings.Contains(got, "Did you mean") || strings.Contains(got, "もしかして") {
			t.Errorf("expected no suggestion for single-letter function, got: %s", got)
		}
	})

	t.Run("caret display for syntax error position", func(t *testing.T) {
		err := errors.New("syntax error: unexpected token at position 5: +")
		line := "1 + * 2"
		got := FormatErrorWithDiagnostics(err, line)
		if !strings.Contains(got, "^") {
			t.Errorf("expected caret '^' in formatted error, got: %s", got)
		}
	})

	t.Run("caret display for japanese syntax error", func(t *testing.T) {
		err := errors.New("構文エラー: 予期しないトークン '+' が位置 5 にあります")
		line := "1 + * 2"
		got := FormatErrorWithDiagnostics(err, line)
		if !strings.Contains(got, "^") {
			t.Errorf("expected caret '^' for japanese syntax error, got: %s", got)
		}
	})
}
