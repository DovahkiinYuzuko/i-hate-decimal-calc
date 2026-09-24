package calc

import (
	"fmt"
)

// IdentityKind distinguishes the source of the identity certificate.
type IdentityKind string

const (
	IdentityKindFactor      IdentityKind = "factor"
	IdentityKindApart       IdentityKind = "apart"
	IdentityKindEquivalence IdentityKind = "equivalence"
	IdentityKindGeneral     IdentityKind = "general"
)

// IdentityCertificate represents a verified algebraic identity of the form LHS == RHS.
// Example: factor(x^2 - 1) == (x - 1)*(x + 1), apart((x+1)/(x^2-1)) == 1/(x-1)
type IdentityCertificate struct {
	BaseCertificate
	Kind IdentityKind
	LHS  Node
	RHS  Node
}

// NewIdentityCertificate creates a new verified IdentityCertificate.
func NewIdentityCertificate(kind IdentityKind, lhs, rhs, residual Node, isVerified bool, details string) *IdentityCertificate {
	eqStr := ""
	if lhs != nil && rhs != nil {
		eqStr = fmt.Sprintf("%s == %s", lhs.String(), rhs.String())
	}
	domain := CertDomainIdentity
	switch kind {
	case IdentityKindFactor:
		domain = CertDomainFactor
	case IdentityKindApart:
		domain = CertDomainRationalApart
	case IdentityKindEquivalence:
		domain = CertDomainEquivalence
	}

	return &IdentityCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    domain,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Kind: kind,
		LHS:  lhs,
		RHS:  rhs,
	}
}
