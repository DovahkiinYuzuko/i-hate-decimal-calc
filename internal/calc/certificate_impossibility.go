package calc

import (
	"fmt"
)

// ImpossibilityKind defines the type of mathematical impossibility proven.
type ImpossibilityKind string

const (
	ImpossibilityNonElementaryIntegral ImpossibilityKind = "non_elementary_integral"
	ImpossibilityNoSolution            ImpossibilityKind = "no_solution"
	ImpossibilityLinearlyDependent     ImpossibilityKind = "linearly_dependent"
)

// ImpossibilityCertificate represents a certified mathematical proof of impossibility / non-existence:
// e.g., Liouville's theorem proving exp(-x^2) has no elementary antiderivative, or 2 == 0 contradiction.
type ImpossibilityCertificate struct {
	BaseCertificate
	Kind        ImpossibilityKind
	Problem     Node
	Obstruction Node
}

// NewImpossibilityCertificate creates a new verified ImpossibilityCertificate.
func NewImpossibilityCertificate(kind ImpossibilityKind, problem, obstruction Node, isVerified bool, details string) *ImpossibilityCertificate {
	eqStr := ""
	if problem != nil {
		eqStr = fmt.Sprintf("Impossibility(%s: %s)", kind, problem.String())
	}

	return &ImpossibilityCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:   CertDomainImpossibility,
			Verified:    isVerified,
			DetailsMsg:  details,
			EquationStr: eqStr,
		},
		Kind:        kind,
		Problem:     problem,
		Obstruction: obstruction,
	}
}
