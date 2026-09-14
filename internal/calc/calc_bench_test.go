package calc

import (
	"testing"
)

// BenchmarkTaylorSeries measures performance of symbolic Taylor series expansion.
func BenchmarkTaylorSeries(b *testing.B) {
	expr, err := ParseStatement("taylor(sin(x), x, 0, 8)")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	node := expr.(Node)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Eval(node)
		if err != nil {
			b.Fatalf("eval error: %v", err)
		}
	}
}

// BenchmarkBareissLinearSolver measures solving a 3x3 system of linear equations using Bareiss algorithm.
func BenchmarkBareissLinearSolver(b *testing.B) {
	expr, err := ParseStatement("solve_linear([[2, 1, -1], [-3, -1, 2], [-2, 1, 2]], [8, -11, -3])")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	node := expr.(Node)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Eval(node)
		if err != nil {
			b.Fatalf("eval error: %v", err)
		}
	}
}

// BenchmarkContinuedFractions measures continuous fraction expansion and reconstruction.
func BenchmarkContinuedFractions(b *testing.B) {
	expr1, err := ParseStatement("cfrac(355/113)")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	node1 := expr1.(Node)

	expr2, err := ParseStatement("from_cfrac([3, 7, 16])")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	node2 := expr2.(Node)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Eval(node1)
		if err != nil {
			b.Fatalf("eval error: %v", err)
		}
		_, err = Eval(node2)
		if err != nil {
			b.Fatalf("eval error: %v", err)
		}
	}
}

// BenchmarkRadicalDenesting measures Borodin radical denesting.
func BenchmarkRadicalDenesting(b *testing.B) {
	expr, err := ParseStatement("sqrt(5 + 2*sqrt(6))")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	node := expr.(Node)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Eval(node)
		if err != nil {
			b.Fatalf("eval error: %v", err)
		}
	}
}

// BenchmarkNumberTheory measures Garner's CRT, 64-bit deterministic prime testing, and modular inverse.
func BenchmarkNumberTheory(b *testing.B) {
	exprCRT, err := ParseStatement("crt([123456, 654321], [1000000007, 1000000009])")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	nodeCRT := exprCRT.(Node)

	exprPrime, err := ParseStatement("is_prime(2305843009213693951)")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	nodePrime := exprPrime.(Node)

	exprInv, err := ParseStatement("inv_mod(65537, 1000000007)")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	nodeInv := exprInv.(Node)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Eval(nodeCRT)
		_, _ = Eval(nodePrime)
		_, _ = Eval(nodeInv)
	}
}

// BenchmarkPolynomialExpansion measures distributive expansion of polynomials.
func BenchmarkPolynomialExpansion(b *testing.B) {
	expr, err := ParseStatement("expand((x + 1)^4)")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	node := expr.(Node)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Eval(node)
		if err != nil {
			b.Fatalf("eval error: %v", err)
		}
	}
}

// BenchmarkPlotter measures 2x4 Braille high-resolution function plotting.
func BenchmarkPlotter(b *testing.B) {
	expr, err := ParseStatement("plot(sin(x), [-pi, pi])")
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	node := expr.(Node)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Eval(node)
		if err != nil {
			b.Fatalf("eval error: %v", err)
		}
	}
}
