package poly

import (
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/domain"
)

// FromPolyNode converts an ast.PolyNode (rational polynomial) to a generic Polynomial[*big.Rat].
func FromPolyNode(node *ast.PolyNode) *Polynomial[*big.Rat] {
	if node == nil {
		return nil
	}
	dom := domain.NewRationalDomain()
	terms := make([]Term[*big.Rat], len(node.Terms))
	for i, t := range node.Terms {
		expCopy := make([]int, len(t.Exponents))
		copy(expCopy, t.Exponents)
		terms[i] = Term[*big.Rat]{
			Coeff:     new(big.Rat).Set(t.Coeff),
			Exponents: expCopy,
		}
	}
	return NewPolynomial(dom, node.Vars, node.Order, terms)
}

// ToPolyNode converts a generic Polynomial[*big.Rat] back to an ast.PolyNode.
func ToPolyNode(p *Polynomial[*big.Rat]) *ast.PolyNode {
	if p == nil {
		return nil
	}
	vCopy := make([]string, len(p.Vars))
	copy(vCopy, p.Vars)

	terms := make([]ast.Monomial, len(p.Terms))
	for i, t := range p.Terms {
		expCopy := make([]int, len(t.Exponents))
		copy(expCopy, t.Exponents)
		terms[i] = ast.Monomial{
			Coeff:     new(big.Rat).Set(t.Coeff),
			Exponents: expCopy,
		}
	}

	return &ast.PolyNode{
		Vars:  vCopy,
		Order: p.Order,
		Terms: terms,
	}
}
