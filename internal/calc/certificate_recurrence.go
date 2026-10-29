package calc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// RecurrenceCertificate represents an algebraic verification certificate for linear recurrence solutions.
type RecurrenceCertificate struct {
	BaseCertificate
	RecurrenceNode       Node
	SolutionNode         Node
	SeqName              string
	IdxVar               string
	Order                int
	CharPolyCoeffs       []*big.Rat
	RationalRoots        []*big.Rat
	IsHomogeneous        bool
	HasOnlyRationalRoots bool
	InitialConditions    []Node
}

// NewRecurrenceCertificate creates a new verification certificate for a recurrence solution.
func NewRecurrenceCertificate(
	recNode Node,
	solNode Node,
	seqName string,
	idxVar string,
	order int,
	charPolyCoeffs []*big.Rat,
	ratRoots []*big.Rat,
	isHomogeneous bool,
	hasOnlyRatRoots bool,
	inits []Node,
	isVerified bool,
	details string,
) *RecurrenceCertificate {
	return &RecurrenceCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:  CertDomainRecurrence,
			Verified:   isVerified,
			DetailsMsg: details,
		},
		RecurrenceNode:       recNode,
		SolutionNode:         solNode,
		SeqName:              seqName,
		IdxVar:               idxVar,
		Order:                order,
		CharPolyCoeffs:       charPolyCoeffs,
		RationalRoots:        ratRoots,
		IsHomogeneous:        isHomogeneous,
		HasOnlyRationalRoots: hasOnlyRatRoots,
		InitialConditions:    inits,
	}
}

// CertDomainRecurrence identifies linear recurrence certificates.
const CertDomainRecurrence CertificateDomain = "recurrence"

// CheckEquation returns the equality representing the verification: LHS(sol) - RHS(sol) == 0.
func (c *RecurrenceCertificate) CheckEquation() string {
	if c.RecurrenceNode != nil && c.SolutionNode != nil {
		return fmt.Sprintf("%s == %s", c.RecurrenceNode.String(), c.SolutionNode.String())
	}
	return "Recurrence Residual == 0"
}

// RenderTraceExplain renders a pedagogical step-by-step trace for the recurrence solution.
func (c *RecurrenceCertificate) RenderTraceExplain(lang string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", i18n.T("rsolve.explain_recurrence"), c.RecurrenceNode.String()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", i18n.T("rsolve.explain_order"), c.Order))

	if len(c.CharPolyCoeffs) > 0 {
		var terms []string
		for i := len(c.CharPolyCoeffs) - 1; i >= 0; i-- {
			cf := c.CharPolyCoeffs[i]
			if cf.Sign() == 0 {
				continue
			}
			switch i {
			case 0:
				terms = append(terms, cf.RatString())
			case 1:
				terms = append(terms, fmt.Sprintf("%s * r", cf.RatString()))
			default:
				terms = append(terms, fmt.Sprintf("%s * r^%d", cf.RatString(), i))
			}
		}
		sb.WriteString(fmt.Sprintf("%s: %s == 0\n", i18n.T("rsolve.explain_char_poly"), strings.Join(terms, " + ")))
	}

	if len(c.RationalRoots) > 0 {
		var rootStrs []string
		for _, r := range c.RationalRoots {
			rootStrs = append(rootStrs, r.RatString())
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", i18n.T("rsolve.explain_roots"), strings.Join(rootStrs, ", ")))
	}

	sb.WriteString(fmt.Sprintf("%s: %s\n", i18n.T("rsolve.explain_solution"), c.SolutionNode.String()))
	if c.IsVerified() {
		sb.WriteString(fmt.Sprintf("%s: %s\n", i18n.T("rsolve.explain_verification"), i18n.T("rsolve.explain_verified_algebraic_and_numeric")))
	}
	return sb.String()
}
