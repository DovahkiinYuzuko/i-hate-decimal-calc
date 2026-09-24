package calc

import (
	"fmt"
	"strings"
)

// GeometricCertificate represents a certified mathematical proof of a geometric theorem
// derived using Wu's method and characteristic sets.
type GeometricCertificate struct {
	BaseCertificate
	Hypotheses         []Node
	Conclusion         Node
	TriangularChain    []Node
	GeometricNDGs      []Node
	ConstructionNDGs   []Node
	SaturationInitials []Node
	RemainderNode      Node
}

// NewGeometricCertificate creates a new GeometricCertificate.
func NewGeometricCertificate(
	hypotheses []Node,
	conclusion Node,
	chain []Node,
	geoNDGs []Node,
	constNDGs []Node,
	initials []Node,
	remainder Node,
	isVerified bool,
	details string,
) *GeometricCertificate {
	eqStr := ""
	if conclusion != nil {
		eqStr = fmt.Sprintf("prem(%s, CS) == %s", conclusion.String(), remainder.String())
	}

	var allConds []Node
	allConds = append(allConds, geoNDGs...)
	allConds = append(allConds, constNDGs...)
	allConds = append(allConds, initials...)

	return &GeometricCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainGeometry,
			Verified:     isVerified,
			ResidualNode: remainder,
			Conditions:   allConds,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Hypotheses:         hypotheses,
		Conclusion:         conclusion,
		TriangularChain:    chain,
		GeometricNDGs:      geoNDGs,
		ConstructionNDGs:   constNDGs,
		SaturationInitials: initials,
		RemainderNode:      remainder,
	}
}

// String returns a human-readable CLI representation of the geometric proof.
func (c *GeometricCertificate) String() string {
	if c == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[GEOMETRIC PROOF: Wu's Method]\n")

	sb.WriteString("Hypotheses:\n")
	for i, h := range c.Hypotheses {
		sb.WriteString(fmt.Sprintf("  H%d: %s == 0\n", i+1, h.String()))
	}

	if len(c.TriangularChain) > 0 {
		sb.WriteString("Ascending Chain (Triangular Set):\n")
		for i, a := range c.TriangularChain {
			sb.WriteString(fmt.Sprintf("  A%d: %s == 0\n", i+1, a.String()))
		}
	}

	if c.Conclusion != nil {
		sb.WriteString(fmt.Sprintf("Conclusion: %s == 0\n", c.Conclusion.String()))
	}

	if len(c.GeometricNDGs) > 0 || len(c.ConstructionNDGs) > 0 || len(c.SaturationInitials) > 0 {
		sb.WriteString("Non-Degeneracy Conditions:\n")
		for i, ndg := range c.GeometricNDGs {
			sb.WriteString(fmt.Sprintf("  [Geometric] G%d: %s != 0\n", i+1, ndg.String()))
		}
		for i, ndg := range c.ConstructionNDGs {
			sb.WriteString(fmt.Sprintf("  [Construction] C%d: %s != 0\n", i+1, ndg.String()))
		}
		for i, initNode := range c.SaturationInitials {
			sb.WriteString(fmt.Sprintf("  [Initial] I%d: %s != 0\n", i+1, initNode.String()))
		}
	}

	sb.WriteString(fmt.Sprintf("Pseudo-Remainder: %s\n", c.RemainderNode.String()))
	if c.Verified {
		sb.WriteString("Status: Generically Certified (Holds under non-degeneracy conditions)")
	} else {
		sb.WriteString("Status: Inconclusive (Remainder non-zero on current component)")
	}

	return sb.String()
}
