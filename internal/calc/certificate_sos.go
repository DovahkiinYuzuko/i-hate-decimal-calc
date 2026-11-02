package calc

import (
	"fmt"
	"math/big"
	"strings"
)

// SOSCertificateTerm represents a certified square component: c_i * (q_i(x))^2.
type SOSCertificateTerm struct {
	Weight *big.Rat // c_i > 0
	Poly   Node     // q_i(x)
}

// SOSCertificate represents a verified algebraic proof of polynomial non-negativity
// via rational sum-of-squares decomposition P(x) = sum_{i=1}^k c_i * (q_i(x))^2.
type SOSCertificate struct {
	BaseCertificate
	Target         Node                 // Target polynomial P(x)
	Terms          []SOSCertificateTerm // Non-negative weighted squares
	Status         SOSStatus            // Certified / NoCertificateFound / Refuted
	Counterexample map[string]*big.Rat // Witness point when refuted (P(a) < 0)
	Variables      []string             // Free variables
}

// NewSOSCertificate constructs a verified SOSCertificate.
func NewSOSCertificate(
	target Node,
	terms []SOSCertificateTerm,
	status SOSStatus,
	counterexample map[string]*big.Rat,
	vars []string,
	residual Node,
	isVerified bool,
	details string,
) *SOSCertificate {
	var sumStrs []string
	for _, t := range terms {
		weightStr := t.Weight.String()
		if t.Weight.Cmp(big.NewRat(1, 1)) == 0 {
			sumStrs = append(sumStrs, fmt.Sprintf("(%s)^2", t.Poly.String()))
		} else {
			sumStrs = append(sumStrs, fmt.Sprintf("(%s) * (%s)^2", weightStr, t.Poly.String()))
		}
	}
	var eqStr string
	if len(sumStrs) > 0 {
		eqStr = fmt.Sprintf("%s == %s", target.String(), strings.Join(sumStrs, " + "))
	} else {
		eqStr = fmt.Sprintf("%s (status: %s)", target.String(), status.String())
	}

	verStatus := StatusVerified
	if status == SOSRefuted {
		verStatus = StatusRefuted
	} else if status == SOSNoCertificateFound {
		verStatus = StatusUnknown
	}

	return &SOSCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainSOS,
			Verified:     isVerified,
			Status:       verStatus,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Target:         target,
		Terms:          terms,
		Status:         status,
		Counterexample: counterexample,
		Variables:      vars,
	}
}
