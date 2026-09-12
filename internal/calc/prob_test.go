package calc

import (
	"testing"
)

func TestBinomPMF(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"binom(10, 3, 1/2)", "15/128"},
		{"binom(3, 2, 1/2)", "3/8"},
		{"binom(6, 1, 1/6)", "3125/7776"},
		{"binom(5, 0, 1/3)", "32/243"},
		{"binom(5, 5, 1/3)", "1/243"},
		{"binom(5, 6, 1/3)", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := EvalString(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			if Format(res) != tt.expected {
				t.Errorf("for %s, expected %s, got %s", tt.expr, tt.expected, Format(res))
			}
		})
	}
}

func TestBinomErrors(t *testing.T) {
	errTests := []string{
		"binom(10, 3, 2)",    // p > 1
		"binom(10, 3, -1/2)", // p < 0
		"binom(10.5, 3, 1/2)", // non-integer n
		"binom(-1, 3, 1/2)",  // negative n
		"binom(10, 3)",       // wrong arg count
	}

	for _, expr := range errTests {
		t.Run(expr, func(t *testing.T) {
			_, err := EvalString(expr)
			if err == nil {
				t.Errorf("expected error for %s, got nil", expr)
			}
		})
	}
}

func TestHyperPMF(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"hyper(10, 3, 2, 0)", "7/15"},
		{"hyper(10, 3, 2, 1)", "7/15"},
		{"hyper(10, 3, 2, 2)", "1/15"},
		{"hyper(10, 3, 2, 3)", "0"}, // k > n
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := EvalString(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			if Format(res) != tt.expected {
				t.Errorf("for %s, expected %s, got %s", tt.expr, tt.expected, Format(res))
			}
		})
	}
}

func TestHyperErrors(t *testing.T) {
	errTests := []string{
		"hyper(10, 12, 2, 1)", // K > N
		"hyper(10, 3, 15, 1)", // n > N
		"hyper(0, 0, 0, 0)",   // N <= 0
		"hyper(10, 3, 2)",     // wrong arg count
	}

	for _, expr := range errTests {
		t.Run(expr, func(t *testing.T) {
			_, err := EvalString(expr)
			if err == nil {
				t.Errorf("expected error for %s, got nil", expr)
			}
		})
	}
}

func TestGeomPMF(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"geom(1/3, 4)", "8/81"},
		{"geom(1/2, 1)", "1/2"},
		{"geom(1, 1)", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := EvalString(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			if Format(res) != tt.expected {
				t.Errorf("for %s, expected %s, got %s", tt.expr, tt.expected, Format(res))
			}
		})
	}
}

func TestGeomErrors(t *testing.T) {
	errTests := []string{
		"geom(0, 1)",   // p = 0
		"geom(1/2, 0)", // k = 0
		"geom(1/2, -1)",
		"geom(2, 1)",   // p > 1
	}

	for _, expr := range errTests {
		t.Run(expr, func(t *testing.T) {
			_, err := EvalString(expr)
			if err == nil {
				t.Errorf("expected error for %s, got nil", expr)
			}
		})
	}
}

func TestBayes(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"bayes(1/100, 9/10, 1/10)", "9/100"},
		{"bayes(1/2, 1/3, 1/2)", "1/3"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := EvalString(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			if Format(res) != tt.expected {
				t.Errorf("for %s, expected %s, got %s", tt.expr, tt.expected, Format(res))
			}
		})
	}
}

func TestBayesErrors(t *testing.T) {
	errTests := []string{
		"bayes(1/2, 1/2, 0)",   // marginal = 0
		"bayes(9/10, 9/10, 1/10)", // posterior > 1
	}

	for _, expr := range errTests {
		t.Run(expr, func(t *testing.T) {
			_, err := EvalString(expr)
			if err == nil {
				t.Errorf("expected error for %s, got nil", expr)
			}
		})
	}
}

func TestExpectation(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"expect(binom, 10, 1/2)", "5"},
		{"expect(geom, 1/4)", "4"},
		{"expect(hyper, 10, 3, 4)", "6/5"},
		{"expect([[1, 1/4], [2, 1/2], [3, 1/4]])", "2"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := EvalString(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			if Format(res) != tt.expected {
				t.Errorf("for %s, expected %s, got %s", tt.expr, tt.expected, Format(res))
			}
		})
	}
}

func TestVariance(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"variance(binom, 10, 1/2)", "5/2"},
		{"variance(geom, 1/3)", "6"},
		{"variance(hyper, 10, 3, 2)", "28/75"},
		{"variance([[1, 1/4], [2, 1/2], [3, 1/4]])", "1/2"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := EvalString(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			if Format(res) != tt.expected {
				t.Errorf("for %s, expected %s, got %s", tt.expr, tt.expected, Format(res))
			}
		})
	}
}

func TestStdDev(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"stddev(binom, 10, 1/2)", "√10/2"},
		{"stddev(geom, 1/3)", "√6"},
		{"stddev(binom, 16, 1/2)", "2"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := EvalString(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			if Format(res) != tt.expected {
				t.Errorf("for %s, expected %s, got %s", tt.expr, tt.expected, Format(res))
			}
		})
	}
}
