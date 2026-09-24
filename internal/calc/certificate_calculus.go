package calc

import (
	"fmt"
)

// DerivCertificate represents a certified differentiation / anti-derivative relation:
// d/dx [Antiderivative] == Integrand.
type DerivCertificate struct {
	BaseCertificate
	Integrand      Node
	Antiderivative Node
	Variable       string
}

// NewDerivCertificate creates a new verified DerivCertificate.
func NewDerivCertificate(integrand, antiderivative, residual Node, variable string, isVerified bool, details string) *DerivCertificate {
	eqStr := ""
	if integrand != nil && antiderivative != nil {
		eqStr = fmt.Sprintf("d/d%s [%s] == %s", variable, antiderivative.String(), integrand.String())
	}
	return &DerivCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainCalculus,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Integrand:      integrand,
		Antiderivative: antiderivative,
		Variable:       variable,
	}
}

// ODECertificate represents a certified ordinary differential equation solution:
// L[Solution] == 0.
type ODECertificate struct {
	BaseCertificate
	ODE            Node
	Solution       Node
	IndependentVar string
	DependentVar   string
}

// NewODECertificate creates a new verified ODECertificate.
func NewODECertificate(ode, solution, residual Node, indepVar, depVar string, isVerified bool, details string) *ODECertificate {
	eqStr := ""
	if ode != nil && solution != nil {
		eqStr = fmt.Sprintf("%s with %s(%s)=%s == 0", ode.String(), depVar, indepVar, solution.String())
	}
	return &ODECertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainODE,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		ODE:            ode,
		Solution:       solution,
		IndependentVar: indepVar,
		DependentVar:   depVar,
	}
}
