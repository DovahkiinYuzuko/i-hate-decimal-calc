package calc

import (
	"io"
	"math/big"
	"testing"
)

func BenchmarkFormatString_Medium(b *testing.B) {
	val := new(big.Int).Lsh(big.NewInt(1), 20000) // ~6,000 digits

	b.ResetTimer()
	for b.Loop() {
		_ = val.String()
	}
}

func BenchmarkWriteIntDecimal_Medium(b *testing.B) {
	val := new(big.Int).Lsh(big.NewInt(1), 20000) // ~6,000 digits

	b.ResetTimer()
	for b.Loop() {
		_ = WriteIntDecimal(io.Discard, val)
	}
}

func BenchmarkWriteIntDecimal_Large(b *testing.B) {
	val := new(big.Int).Lsh(big.NewInt(1), 100000) // ~30,000 digits

	b.ResetTimer()
	for b.Loop() {
		_ = WriteIntDecimal(io.Discard, val)
	}
}

func BenchmarkComputeExactDecimalDigits(b *testing.B) {
	val := new(big.Int).Lsh(big.NewInt(1), 1048576) // ~315,653 digits

	b.ResetTimer()
	for b.Loop() {
		_ = ComputeExactDecimalDigits(val)
	}
}
