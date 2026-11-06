package calc

import (
	"fmt"
)

// KirchhoffCertificate represents a certified application of Kirchhoff's Matrix Tree Theorem:
// det(LaplacianMinor) == SpanningTrees.
type KirchhoffCertificate struct {
	BaseCertificate
	Adjacency      Node
	LaplacianMinor Node
	SpanningTrees  Node
}

// NewKirchhoffCertificate creates a new verified KirchhoffCertificate.
func NewKirchhoffCertificate(adj, minor, trees, residual Node, isVerified bool, details string) *KirchhoffCertificate {
	eqStr := ""
	if minor != nil && trees != nil {
		eqStr = fmt.Sprintf("det(%s) == %s", minor.String(), trees.String())
	}
	return &KirchhoffCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainGraph,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Adjacency:      adj,
		LaplacianMinor: minor,
		SpanningTrees:  trees,
	}
}

// MinimaxCertificate represents a certified Nash equilibrium for two-player zero-sum games:
// x^T A >= v * 1^T, A y <= v * 1, sum(x) = 1, sum(y) = 1, x >= 0, y >= 0.
type MinimaxCertificate struct {
	BaseCertificate
	PayoffMatrix Node
	StrategyP1   Node
	StrategyP2   Node
	GameValue    Node
}

// NewMinimaxCertificate creates a new verified MinimaxCertificate.
func NewMinimaxCertificate(payoff, p1, p2, val, residual Node, isVerified bool, details string) *MinimaxCertificate {
	eqStr := ""
	if payoff != nil && val != nil {
		eqStr = fmt.Sprintf("minimax_value(%s) == %s", payoff.String(), val.String())
	}
	return &MinimaxCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainGame,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		PayoffMatrix: payoff,
		StrategyP1:   p1,
		StrategyP2:   p2,
		GameValue:    val,
	}
}

// MarkovStationaryCertificate represents a certified stationary distribution for a Markov chain:
// pi * P == pi, sum(pi) == 1, pi >= 0.
type MarkovStationaryCertificate struct {
	BaseCertificate
	TransitionMatrix Node
	StationaryDist   Node
}

// NewMarkovStationaryCertificate creates a new verified MarkovStationaryCertificate.
func NewMarkovStationaryCertificate(trans, dist, residual Node, isVerified bool, details string) *MarkovStationaryCertificate {
	eqStr := ""
	if trans != nil && dist != nil {
		eqStr = fmt.Sprintf("%s * %s == %s", dist.String(), trans.String(), dist.String())
	}
	return &MarkovStationaryCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainProbability,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		TransitionMatrix: trans,
		StationaryDist:   dist,
	}
}
