package provenance_test

import (
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

func BenchmarkNilRecorderOverhead(b *testing.B) {
	var rec *provenance.TraceRecorder // nil pointer
	node, _ := ast.NewRational(1, 2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec.StartScope("Scope", provenance.RelationEq, nil, node, provenance.VisPresentable)
		rec.RecordStep("Rule", provenance.RelationEq, nil, node, node, nil, nil, provenance.TrustCertified, provenance.VisInternal)
		rec.EndScope(node, nil)
	}
}

func BenchmarkActiveRecorder(b *testing.B) {
	node, _ := ast.NewRational(1, 2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := provenance.NewTraceRecorder()
		rec.StartScope("Scope", provenance.RelationEq, nil, node, provenance.VisPresentable)
		rec.RecordStep("Rule", provenance.RelationEq, nil, node, node, nil, nil, provenance.TrustCertified, provenance.VisInternal)
		rec.EndScope(node, nil)
	}
}
