package calc

import (
	"fmt"
	"strings"
)

// InvertibilityCertificate represents a certified matrix inversion:
// M * Inverse == I, with NonDegeneracy condition det(M) != 0.
type InvertibilityCertificate struct {
	BaseCertificate
	Matrix      Node
	Inverse     Node
	Determinant Node
}

// NewInvertibilityCertificate creates a new verified InvertibilityCertificate.
func NewInvertibilityCertificate(matrix, inverse, det, residual Node, isVerified bool, details string) *InvertibilityCertificate {
	eqStr := ""
	if matrix != nil && inverse != nil {
		eqStr = fmt.Sprintf("%s * %s == I", matrix.String(), inverse.String())
	}
	var conds []Node
	if det != nil {
		conds = append(conds, det)
	}

	return &InvertibilityCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainMatrix,
			Verified:     isVerified,
			ResidualNode: residual,
			Conditions:   conds,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Matrix:      matrix,
		Inverse:     inverse,
		Determinant: det,
	}
}

// DecompositionKind defines the type of matrix decomposition.
type DecompositionKind string

const (
	DecompLU       DecompositionKind = "lu"
	DecompQR       DecompositionKind = "qr"
	DecompCholesky DecompositionKind = "cholesky"
	DecompLDLT     DecompositionKind = "ldlt"
	DecompPinv     DecompositionKind = "pinv"
)

// DecompositionCertificate represents a certified matrix factor decomposition:
// e.g. A == P * L * U or A == Q * R.
type DecompositionCertificate struct {
	BaseCertificate
	Kind    DecompositionKind
	Matrix  Node
	Factors []Node
}

// NewDecompositionCertificate creates a new verified DecompositionCertificate.
func NewDecompositionCertificate(kind DecompositionKind, matrix Node, factors []Node, residual Node, isVerified bool, details string) *DecompositionCertificate {
	var factorStrs []string
	for _, f := range factors {
		if f != nil {
			factorStrs = append(factorStrs, f.String())
		}
	}
	eqStr := ""
	if matrix != nil && len(factorStrs) > 0 {
		eqStr = fmt.Sprintf("%s == %s", matrix.String(), strings.Join(factorStrs, " * "))
	}

	return &DecompositionCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainMatrix,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Kind:    kind,
		Matrix:  matrix,
		Factors: factors,
	}
}
