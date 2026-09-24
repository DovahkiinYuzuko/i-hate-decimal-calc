package calc

import (
	"testing"
)

func TestCertificate_Identity(t *testing.T) {
	lhs := NewVar("x")
	rhs := NewVar("x")
	res, _ := NewRational(0, 1)

	cert := NewIdentityCertificate(IdentityKindFactor, lhs, rhs, res, true, "Factorization verified")
	if cert.Domain() != CertDomainFactor {
		t.Errorf("expected CertDomainFactor, got %v", cert.Domain())
	}
	if !cert.IsVerified() {
		t.Errorf("expected IsVerified true")
	}
	if cert.Residual() != res {
		t.Errorf("expected residual to match")
	}
	if cert.String() != "[VERIFIED: x == x]" {
		t.Errorf("unexpected String(): %s", cert.String())
	}
}

func TestCertificate_Deriv(t *testing.T) {
	f := NewVar("x")
	F := NewVar("x")
	res, _ := NewRational(0, 1)

	cert := NewDerivCertificate(f, F, res, "x", true, "Deriv verified")
	if cert.Domain() != CertDomainCalculus {
		t.Errorf("expected CertDomainCalculus, got %v", cert.Domain())
	}
	if !cert.IsVerified() {
		t.Errorf("expected IsVerified true")
	}
	if cert.String() != "[VERIFIED: d/dx [x] == x]" {
		t.Errorf("unexpected String(): %s", cert.String())
	}
}

func TestCertificate_Invertibility(t *testing.T) {
	m := NewVar("A")
	inv := NewVar("A_inv")
	det, _ := NewRational(1, 1)
	res, _ := NewRational(0, 1)

	cert := NewInvertibilityCertificate(m, inv, det, res, true, "Invertibility verified")
	if cert.Domain() != CertDomainMatrix {
		t.Errorf("expected CertDomainMatrix, got %v", cert.Domain())
	}
	if len(cert.NonDegeneracyConditions()) != 1 {
		t.Errorf("expected 1 condition (det), got %d", len(cert.NonDegeneracyConditions()))
	}
	if cert.String() != "[VERIFIED: A * A_inv == I]" {
		t.Errorf("unexpected String(): %s", cert.String())
	}
}

func TestCertificate_Impossibility(t *testing.T) {
	prob := NewVar("exp(-x^2)")
	cert := NewImpossibilityCertificate(ImpossibilityNonElementaryIntegral, prob, nil, true, "Liouville theorem non-elementary")
	if cert.Domain() != CertDomainImpossibility {
		t.Errorf("expected CertDomainImpossibility, got %v", cert.Domain())
	}
	if !cert.IsVerified() {
		t.Errorf("expected IsVerified true")
	}
	if cert.String() != "[VERIFIED: Impossibility(non_elementary_integral: exp(-x^2))]" {
		t.Errorf("unexpected String(): %s", cert.String())
	}
}
