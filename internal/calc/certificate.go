package calc

import (
	"fmt"
)

// CertificateDomain defines the mathematical domain of the verified operation.
type CertificateDomain string

const (
	CertDomainIdentity      CertificateDomain = "identity"
	CertDomainFactor        CertificateDomain = "factor"
	CertDomainCalculus      CertificateDomain = "calculus"
	CertDomainODE           CertificateDomain = "ode"
	CertDomainMatrix        CertificateDomain = "matrix"
	CertDomainRationalApart CertificateDomain = "rational"
	CertDomainWZ            CertificateDomain = "wz"
	CertDomainImpossibility CertificateDomain = "impossibility"
	CertDomainEquivalence   CertificateDomain = "equivalence"
	CertDomainGeneral       CertificateDomain = "general"
)

// Certificate represents a certified mathematical proof of correctness.
// It serves as an immutable intermediate representation (IR) between the CAS solver,
// internal verification engine, and external formal provers (Lean 4, Coq, etc.).
type Certificate interface {
	Domain() CertificateDomain
	IsVerified() bool
	Residual() Node
	NonDegeneracyConditions() []Node
	Details() string
	EquationString() string
	String() string
}

// BaseCertificate holds standard fields common to all algebraic certificates.
type BaseCertificate struct {
	DomainVal       CertificateDomain
	Verified        bool
	ResidualNode    Node
	Conditions      []Node
	DetailsMsg      string
	EquationStr     string
}

// Domain returns the mathematical domain of the certificate.
func (b *BaseCertificate) Domain() CertificateDomain {
	return b.DomainVal
}

// IsVerified returns whether the verification check succeeded.
func (b *BaseCertificate) IsVerified() bool {
	return b.Verified
}

// Residual returns the zero-residual node (expected to evaluate to 0 for successful verification).
func (b *BaseCertificate) Residual() Node {
	return b.ResidualNode
}

// NonDegeneracyConditions returns required non-zero / non-empty hypotheses (e.g. det(M) != 0, denom != 0).
func (b *BaseCertificate) NonDegeneracyConditions() []Node {
	return b.Conditions
}

// Details returns supplementary diagnostic or explanation messages.
func (b *BaseCertificate) Details() string {
	return b.DetailsMsg
}

// EquationString returns the mathematical equation representing the certificate.
func (b *BaseCertificate) EquationString() string {
	return b.EquationStr
}

// String returns a human-readable CLI representation of the certificate.
func (b *BaseCertificate) String() string {
	if b == nil {
		return ""
	}
	if b.Verified {
		return fmt.Sprintf("[VERIFIED: %s]", b.EquationStr)
	}
	return fmt.Sprintf("[FAILED VERIFICATION: %s] %s", b.EquationStr, b.DetailsMsg)
}
