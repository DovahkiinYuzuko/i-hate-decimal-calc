package calc

import (
	"fmt"
)

// WZCertificate represents a certified Wilf-Zeilberger pair:
// F(n+1, k) - F(n, k) == G(n, k+1) - G(n, k) where G(n, k) = R(n, k) * F(n, k).
type WZCertificate struct {
	BaseCertificate
	Summand             Node
	CertificateFunction Node
	NVar                string
	KVar                string
}

// NewWZCertificate creates a new verified WZCertificate.
func NewWZCertificate(summand, certFunc, residual Node, nVar, kVar string, isVerified bool, details string) *WZCertificate {
	eqStr := ""
	if summand != nil && certFunc != nil {
		eqStr = fmt.Sprintf("WZ(F=%s, R=%s, n=%s, k=%s)", summand.String(), certFunc.String(), nVar, kVar)
	}

	return &WZCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainWZ,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Summand:             summand,
		CertificateFunction: certFunc,
		NVar:                nVar,
		KVar:                kVar,
	}
}
