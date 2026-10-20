package provenance_test

import (
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

func TestNilRecorderZeroOverhead(t *testing.T) {
	var rec *provenance.TraceRecorder
	// Ensure calling methods on nil pointer does not panic
	rec.StartScope("TestScope", provenance.RelationEq, nil, nil, provenance.VisPresentable)
	rec.RecordStep("TestRule", provenance.RelationEq, nil, nil, nil, nil, nil, provenance.TrustCertified, provenance.VisPresentable)
	rec.RecordNormalize(nil, nil, nil)
	rec.EndScope(nil, nil)
	rec.AttachSubtree(nil)

	if rec.Root() != nil {
		t.Fatalf("expected nil root on nil recorder")
	}
}

func TestHierarchicalScope(t *testing.T) {
	rec := provenance.NewTraceRecorder()
	node1, _ := ast.NewRational(1, 2)
	node2, _ := ast.NewRational(2, 4)
	node3, _ := ast.NewRational(1, 2)

	rec.StartScope("IntegrationByParts", provenance.RelationEq, nil, node1, provenance.VisPresentable)
	rec.RecordStep(
		"SubIntegration",
		provenance.RelationEq,
		[]provenance.PathElem{{Kind: provenance.PathArg, Index: 0}},
		node1,
		node2,
		map[string]ast.Node{"u": node1},
		[]provenance.SideCondition{{Kind: "NonZero", Target: node1, Description: "x != 0"}},
		provenance.TrustCertified,
		provenance.VisInternal,
	)
	rec.EndScope(node3, &provenance.AntiderivativeCertificate{
		Integrand:      node1,
		Antiderivative: node3,
		Variable:       "x",
		Domain:         "R",
	})

	root := rec.Root()
	if root == nil {
		t.Fatalf("expected non-nil root")
	}
	if root.RuleID != "IntegrationByParts" {
		t.Errorf("expected root rule IntegrationByParts, got %s", root.RuleID)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}
	child := root.Children[0]
	if child.RuleID != "SubIntegration" {
		t.Errorf("expected child rule SubIntegration, got %s", child.RuleID)
	}
	if len(child.Conditions) != 1 || child.Conditions[0].Kind != "NonZero" {
		t.Errorf("expected NonZero condition on child, got %+v", child.Conditions)
	}
	if root.Certificate == nil || root.Certificate.CertificateKind() != "Antiderivative" {
		t.Errorf("expected AntiderivativeCertificate on root, got %+v", root.Certificate)
	}
}

func TestRecordNormalize(t *testing.T) {
	rec := provenance.NewTraceRecorder()
	nodeBefore, _ := ast.NewRational(2, 4)
	nodeAfter, _ := ast.NewRational(1, 2)

	rec.RecordNormalize(nil, nodeBefore, nodeAfter)

	root := rec.Root()
	if root == nil {
		t.Fatalf("expected non-nil root")
	}
	if root.RuleID != provenance.RuleAlgebraNormalize {
		t.Errorf("expected %s, got %s", provenance.RuleAlgebraNormalize, root.RuleID)
	}
	if root.Vis != provenance.VisStructural {
		t.Errorf("expected VisStructural, got %s", root.Vis)
	}
}

func TestAttachSubtree(t *testing.T) {
	recMain := provenance.NewTraceRecorder()
	recSub := provenance.NewTraceRecorder()

	nodeA, _ := ast.NewRational(1, 1)
	nodeB, _ := ast.NewRational(2, 1)

	recSub.RecordStep("SubTaskRule", provenance.RelationEq, nil, nodeA, nodeB, nil, nil, provenance.TrustCertified, provenance.VisPresentable)
	subRoot := recSub.Root()

	recMain.StartScope("MainTask", provenance.RelationEq, nil, nodeA, provenance.VisPresentable)
	recMain.AttachSubtree(subRoot)
	recMain.EndScope(nodeB, nil)

	mainRoot := recMain.Root()
	if len(mainRoot.Children) != 1 {
		t.Fatalf("expected 1 child in main, got %d", len(mainRoot.Children))
	}
	if mainRoot.Children[0].RuleID != "SubTaskRule" {
		t.Errorf("expected attached subtree with rule SubTaskRule, got %s", mainRoot.Children[0].RuleID)
	}
}
