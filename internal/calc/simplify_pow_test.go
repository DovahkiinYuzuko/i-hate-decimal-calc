package calc

import (
	"errors"
	"testing"
)

func TestPowZeroCases(t *testing.T) {
	tests := []struct {
		expr        string
		expected    string
		expectError bool
		expectedErr ErrCode
	}{
		{expr: "0^0", expected: "1", expectError: false},
		{expr: "(0+0)^0", expected: "1", expectError: false},
		{expr: "(0+0*i)^0", expected: "1", expectError: false},
		{expr: "(0/3)^0", expected: "1", expectError: false},
		{expr: "0^2", expected: "0", expectError: false},
		{expr: "0^sqrt(2)", expected: "0", expectError: false},
		{expr: "0^-1", expectError: true, expectedErr: ErrZeroDivision},
		{expr: "0^-10", expectError: true, expectedErr: ErrZeroDivision},
		{expr: "0^sqrt(-2)", expectError: true, expectedErr: ErrDomain},
		{expr: "0^i", expectError: true, expectedErr: ErrDomain},
		{expr: "0^(1+i)", expectError: true, expectedErr: ErrDomain},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			node, err := Parse(tc.expr)
			if err != nil {
				t.Fatalf("parse error on %s: %v", tc.expr, err)
			}
			res, err := EvalWithEnv(node, NewEnv())
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error for %s, got result: %s", tc.expr, res.String())
				}
				var calcErr *CalcError
				if errors.As(err, &calcErr) {
					if calcErr.Code != tc.expectedErr {
						t.Errorf("expected ErrCode %s for %s, got: %s (%v)", tc.expectedErr, tc.expr, calcErr.Code, err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for %s: %v", tc.expr, err)
				}
				if res == nil || res.String() != tc.expected {
					t.Errorf("expected %s -> %s, got: %v", tc.expr, tc.expected, res)
				}
			}
		})
	}
}
