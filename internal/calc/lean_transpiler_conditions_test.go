package calc

import (
	"strings"
	"testing"
)

func TestGenerateLeanSource_WithConditions(t *testing.T) {
	lhs, err := Parse("(x^2 - 1) / (x - 1)")
	if err != nil {
		t.Fatalf("parse lhs: %v", err)
	}
	rhs, err := Parse("x + 1")
	if err != nil {
		t.Fatalf("parse rhs: %v", err)
	}

	condSet := NewDomainConditionSet()
	xMinusOne, _ := Parse("x - 1")
	condSet.Add(CondNonZero, xMinusOne)

	cert := &VerificationCertificate{
		Domain:     DomainGeneral,
		Equation:   "((x^2 - 1) / (x - 1)) == (x + 1)",
		Residual:   mustRational(0, 1),
		IsVerified: true,
		Details:    "verified under domain conditions",
		State:      VerifyStateCertified,
		CertIR:     NewIdentityCertificate(IdentityKindEquivalence, lhs, rhs, mustRational(0, 1), true, "verified"),
	}

	leanCode, err := GenerateLeanSourceWithConditions("test_rational_cond", cert, lhs, rhs, condSet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(leanCode, "h0 :") || !strings.Contains(leanCode, "≠ 0") {
		t.Errorf("expected hypothesis binder with '≠ 0', got:\n%s", leanCode)
	}
	if !strings.Contains(leanCode, "field_simp") {
		t.Errorf("expected tactic to use field_simp, got:\n%s", leanCode)
	}
}

func TestGenerateLeanSourceWithConditions_Empty(t *testing.T) {
	lhs, _ := Parse("x + x")
	rhs, _ := Parse("2 * x")

	cert := &VerificationCertificate{
		Domain:     DomainGeneral,
		Equation:   "(x + x) == (2 * x)",
		Residual:   mustRational(0, 1),
		IsVerified: true,
		State:      VerifyStateCertified,
		CertIR:     NewIdentityCertificate(IdentityKindEquivalence, lhs, rhs, mustRational(0, 1), true, "verified"),
	}

	leanCode, err := GenerateLeanSourceWithConditions("test_poly", cert, lhs, rhs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(leanCode, "h0 :") {
		t.Errorf("expected no hypothesis binder for empty conditions, got:\n%s", leanCode)
	}
	if !strings.Contains(leanCode, "by ring") {
		t.Errorf("expected 'by ring', got:\n%s", leanCode)
	}
}
