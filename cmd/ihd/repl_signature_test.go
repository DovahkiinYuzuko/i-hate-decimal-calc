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

func TestRenderAndClearSignatureGuide(t *testing.T) {
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
	err := RenderSignatureGuide(&buf, ctx, "dark")
	if err != nil {
		t.Fatalf("RenderSignatureGuide failed: %v", err)
	}

	rendered := buf.String()
	if !strings.Contains(rendered, "diff") {
		t.Errorf("rendered guide should contain function name 'diff', got: %q", rendered)
	}
	if !strings.Contains(rendered, "var") {
		t.Errorf("rendered guide should contain active arg 'var', got: %q", rendered)
	}
	if !strings.Contains(rendered, "Calculate derivative") {
		t.Errorf("rendered guide should contain description, got: %q", rendered)
	}

	// Test clear
	buf.Reset()
	err = ClearSignatureGuide(&buf)
	if err != nil {
		t.Fatalf("ClearSignatureGuide failed: %v", err)
	}
	if len(buf.Bytes()) == 0 {
		t.Errorf("ClearSignatureGuide should output escape codes")
	}
}
