package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseSignatureContext(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		cursor        int
		wantOK        bool
		wantFunc      string
		wantArgIndex  int
		wantActiveArg string
	}{
		{
			name:          "simple diff first arg",
			line:          "diff(",
			cursor:        5,
			wantOK:        true,
			wantFunc:      "diff",
			wantArgIndex:  0,
			wantActiveArg: "expr",
		},
		{
			name:          "simple diff second arg",
			line:          "diff(x^2, ",
			cursor:        10,
			wantOK:        true,
			wantFunc:      "diff",
			wantArgIndex:  1,
			wantActiveArg: "var",
		},
		{
			name:          "case insensitive function name",
			line:          "DIFF(x^2, ",
			cursor:        10,
			wantOK:        true,
			wantFunc:      "diff",
			wantArgIndex:  1,
			wantActiveArg: "var",
		},
		{
			name:          "nested closed call inside outer call",
			line:          "plot(sin(x), [x, ",
			cursor:        17,
			wantOK:        true,
			wantFunc:      "plot",
			wantArgIndex:  1,
			wantActiveArg: "range",
		},
		{
			name:          "cursor inside nested inner call",
			line:          "plot(sin(x",
			cursor:        10,
			wantOK:        true,
			wantFunc:      "sin",
			wantArgIndex:  0,
			wantActiveArg: "x",
		},
		{
			name:          "string literal with commas ignored",
			line:          `clifford("e0, e1", `,
			cursor:        19,
			wantOK:        true,
			wantFunc:      "clifford",
			wantArgIndex:  1,
			wantActiveArg: "p",
		},
		{
			name:     "fully closed function",
			line:     "sin(x)",
			cursor:   6,
			wantOK:   false,
			wantFunc: "",
		},
		{
			name:     "unknown identifier",
			line:     "unknown_func(1, ",
			cursor:   16,
			wantOK:   false,
			wantFunc: "",
		},
		{
			name:     "empty line",
			line:     "",
			cursor:   0,
			wantOK:   false,
			wantFunc: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, ok := ParseSignatureContext(tt.line, tt.cursor)
			if ok != tt.wantOK {
				t.Fatalf("ParseSignatureContext() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if !strings.EqualFold(ctx.FuncName, tt.wantFunc) {
				t.Errorf("FuncName = %q, want %q", ctx.FuncName, tt.wantFunc)
			}
			if ctx.ArgIndex != tt.wantArgIndex {
				t.Errorf("ArgIndex = %d, want %d", ctx.ArgIndex, tt.wantArgIndex)
			}
			if tt.wantActiveArg != "" && ctx.ActiveArgName != tt.wantActiveArg {
				t.Errorf("ActiveArgName = %q, want %q", ctx.ActiveArgName, tt.wantActiveArg)
			}
		})
	}
}

func TestRenderSignatureInline(t *testing.T) {
	ctx := &SignatureContext{
		FuncName:      "diff",
		ArgIndex:      1,
		ActiveArgName: "var",
		Args:          []string{"expr", "var"},
		MinArgs:       2,
		MaxArgs:       2,
		Description:   "Calculate derivative",
	}

	var buf bytes.Buffer
	err := RenderSignatureInline(&buf, ctx, "dark", 80)
	if err != nil {
		t.Fatalf("RenderSignatureInline failed: %v", err)
	}

	rendered := buf.String()
	// Must NOT contain any vertical cursor movements or newlines that break the terminal
	if strings.Contains(rendered, "\n") || strings.Contains(rendered, "\r") {
		t.Errorf("rendered guide must not contain newlines or CR, got: %q", rendered)
	}
	if strings.Contains(rendered, "\x1B[1B") || strings.Contains(rendered, "\x1B7") || strings.Contains(rendered, "\x1B8") {
		t.Errorf("rendered guide must not contain vertical cursor movement codes, got: %q", rendered)
	}

	if !strings.Contains(rendered, "diff") {
		t.Errorf("rendered guide should contain function name 'diff', got: %q", rendered)
	}
	if !strings.Contains(rendered, "var") {
		t.Errorf("rendered guide should contain active arg 'var', got: %q", rendered)
	}
	if !strings.Contains(rendered, "Calculate derivative") {
		t.Errorf("rendered guide should contain description, got: %q", rendered)
	}

	// Test light theme
	buf.Reset()
	err = RenderSignatureInline(&buf, ctx, "light", 80)
	if err != nil {
		t.Fatalf("RenderSignatureInline with light theme failed: %v", err)
	}
	renderedLight := buf.String()
	if !strings.Contains(renderedLight, "\x1B[1;34m") {
		t.Errorf("light theme should use blue for function name, got: %q", renderedLight)
	}
	if !strings.Contains(renderedLight, "diff") || !strings.Contains(renderedLight, "var") {
		t.Errorf("light theme should contain func name and active arg, got: %q", renderedLight)
	}

	// Test none theme (plain text, no ANSI escape sequences)
	buf.Reset()
	err = RenderSignatureInline(&buf, ctx, "none", 80)
	if err != nil {
		t.Fatalf("RenderSignatureInline with none theme failed: %v", err)
	}
	renderedNone := buf.String()
	if strings.Contains(renderedNone, "\x1B[") {
		t.Errorf("none theme should not contain any ANSI escape codes, got: %q", renderedNone)
	}
	expectedNone := "  [diff(expr, var) - Calculate derivative]"
	if renderedNone != expectedNone {
		t.Errorf("none theme output mismatch: want %q, got %q", expectedNone, renderedNone)
	}

	// Test insufficient width suppression
	buf.Reset()
	err = RenderSignatureInline(&buf, ctx, "dark", 5)
	if err != nil {
		t.Fatalf("RenderSignatureInline with small width failed: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("RenderSignatureInline should output nothing when availWidth <= 6, got: %q", buf.String())
	}
}
