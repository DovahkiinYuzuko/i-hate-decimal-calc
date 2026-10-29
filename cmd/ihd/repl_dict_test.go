package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func TestREPL_InteractiveDictionaryCommands(t *testing.T) {
	origLocale := i18n.CurrentLocale()
	_ = i18n.Init("en")
	defer func() { _ = i18n.Init(origLocale) }()
	tests := []struct {
		name     string
		input    string
		wantCont []string
	}{
		{
			name:     "help overview",
			input:    "help\nexit\n",
			wantCont: []string{"help <", "search <", "whatis <"},
		},
		{
			name:     "help specific function",
			input:    "help solve\nexit\n",
			wantCont: []string{"solve(eq, var)"},
		},
		{
			name:     "help unknown function",
			input:    "help unknown_func_xyz\nexit\n",
			wantCont: []string{"unknown_func_xyz"},
		},
		{
			name:     "search by function name keyword",
			input:    "search diophantine\nexit\n",
			wantCont: []string{"solve_diophantine"},
		},
		{
			name:     "search by description keyword",
			input:    "search elliptic\nexit\n",
			wantCont: []string{"ec_"},
		},
		{
			name:     "whatis function name",
			input:    "whatis groebner\nexit\n",
			wantCont: []string{"groebner", "primary_decomposition"},
		},
		{
			name:     "whatis concept domain",
			input:    "whatis 楕円曲線\nexit\n",
			wantCont: []string{"Elliptic Curves", "ec_torsion"},
		},
		{
			name:     "whatis equation syntax",
			input:    "whatis(x^2 - 1 == 0)\nexit\n",
			wantCont: []string{"solve", "verify"},
		},
		{
			name:     "whatis polynomial expression",
			input:    "whatis(x^3 + 2*x + 1)\nexit\n",
			wantCont: []string{"factor", "expand", "diff"},
		},
		{
			name:     "whatis matrix literal nested list",
			input:    "whatis([[1, 2], [3, 4]])\nexit\n",
			wantCont: []string{"det", "inv", "eigenvals"},
		},
		{
			name:     "help with function syntax and quotes",
			input:    "help(\"solve\")\nexit\n",
			wantCont: []string{"solve(eq, var)"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := strings.NewReader(tc.input)
			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			ro := runOptions{}
			code := runREPL(in, out, errOut, ro)
			if code != 0 {
				t.Fatalf("runREPL returned non-zero code %d, err: %s", code, errOut.String())
			}
			outStr := out.String()
			for _, want := range tc.wantCont {
				if !strings.Contains(outStr, want) {
					t.Errorf("expected output to contain %q, but got:\n%s", want, outStr)
				}
			}
		})
	}
}
