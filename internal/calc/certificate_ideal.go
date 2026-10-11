package calc

import (
	"fmt"
	"strings"
)

// IdealSyzygyRelation represents f_i = sum_j q_{i,j} * g_j.
type IdealSyzygyRelation struct {
	TargetNode Node   // f_i
	CombCoeffs []Node // q_{i,j}
	Gens       []Node // g_j
}

// IdealDecompositionCertificate encapsulates the verified algebraic proof of
// primary decomposition I = cap_{i=1}^m Q_i and its associated primes P_i = sqrt(Q_i).
type IdealDecompositionCertificate struct {
	BaseCertificate
	IdealGens           []Node
	Components          [][]Node
	Primes              [][]Node
	Dimension           int
	ForwardSyzygies     []IdealSyzygyRelation // Proves I subseteq cap Q_i
	BackwardSyzygies    []IdealSyzygyRelation // Proves cap Q_i subseteq I
	IrredundancyWitness []Node                // Witness elements in (cap_{j != i} Q_j) \ Q_i
}

// NewIdealDecompositionCertificate constructs an IdealDecompositionCertificate.
func NewIdealDecompositionCertificate(
	idealGens []Node,
	comps [][]Node,
	primes [][]Node,
	dim int,
	fwd, bwd []IdealSyzygyRelation,
	witnesses []Node,
	residual Node,
	isVerified bool,
	details string,
) *IdealDecompositionCertificate {
	var compStrs []string
	for _, c := range comps {
		var gStrs []string
		for _, g := range c {
			gStrs = append(gStrs, g.String())
		}
		compStrs = append(compStrs, fmt.Sprintf("<%s>", strings.Join(gStrs, ", ")))
	}
	var idealStrs []string
	for _, g := range idealGens {
		idealStrs = append(idealStrs, g.String())
	}
	eqStr := fmt.Sprintf("<%s> == %s", strings.Join(idealStrs, ", "), strings.Join(compStrs, " ∩ "))

	return &IdealDecompositionCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainIdeal,
			Verified:     isVerified,
			Status:       StatusVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		IdealGens:           idealGens,
		Components:          comps,
		Primes:              primes,
		Dimension:           dim,
		ForwardSyzygies:     fwd,
		BackwardSyzygies:    bwd,
		IrredundancyWitness: witnesses,
	}
}
