package domain

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// FiniteFieldDomain represents a prime Galois field F_p.
// Elements are *big.Int in the range [0, p-1].
type FiniteFieldDomain struct {
	p *big.Int
}

// NewFiniteFieldDomain creates a new F_p domain with modulus p.
// Returns an error if p <= 1 or if p is not prime.
func NewFiniteFieldDomain(p *big.Int) (*FiniteFieldDomain, error) {
	if p == nil || p.Cmp(big.NewInt(1)) <= 0 {
		return nil, errors.New(i18n.T("domain.err_modulus_positive"))
	}
	// math/big's ProbablyPrime(0) runs Baillie-PSW, 100% deterministic with zero known counterexamples.
	if !p.ProbablyPrime(0) {
		return nil, errors.New(i18n.T("domain.err_modulus_must_be_prime", p.String()))
	}
	return &FiniteFieldDomain{
		p: new(big.Int).Set(p),
	}, nil
}

// Modulus returns the characteristic / modulus p.
func (d *FiniteFieldDomain) Modulus() *big.Int {
	return new(big.Int).Set(d.p)
}

func (d *FiniteFieldDomain) Name() string {
	return fmt.Sprintf("F_%s", d.p.String())
}

func (d *FiniteFieldDomain) Zero() *big.Int {
	return big.NewInt(0)
}

func (d *FiniteFieldDomain) One() *big.Int {
	return big.NewInt(1)
}

func (d *FiniteFieldDomain) normalize(a *big.Int) *big.Int {
	if a == nil {
		return d.Zero()
	}
	rem := new(big.Int).Mod(a, d.p)
	if rem.Sign() < 0 {
		rem.Add(rem, d.p)
	}
	return rem
}

func (d *FiniteFieldDomain) IsZero(a *big.Int) bool {
	if a == nil {
		return true
	}
	return d.normalize(a).Sign() == 0
}

func (d *FiniteFieldDomain) IsOne(a *big.Int) bool {
	if a == nil {
		return false
	}
	return d.normalize(a).Cmp(d.One()) == 0
}

func (d *FiniteFieldDomain) Equals(a, b *big.Int) bool {
	normA := d.normalize(a)
	normB := d.normalize(b)
	return normA.Cmp(normB) == 0
}

func (d *FiniteFieldDomain) Add(a, b *big.Int) *big.Int {
	res := new(big.Int).Add(a, b)
	return d.normalize(res)
}

func (d *FiniteFieldDomain) Sub(a, b *big.Int) *big.Int {
	res := new(big.Int).Sub(a, b)
	return d.normalize(res)
}

func (d *FiniteFieldDomain) Mul(a, b *big.Int) *big.Int {
	res := new(big.Int).Mul(a, b)
	return d.normalize(res)
}

func (d *FiniteFieldDomain) Neg(a *big.Int) *big.Int {
	res := new(big.Int).Neg(a)
	return d.normalize(res)
}

func (d *FiniteFieldDomain) Inv(a *big.Int) (*big.Int, error) {
	normA := d.normalize(a)
	if normA.Sign() == 0 {
		return nil, errors.New(i18n.T("domain.err_zero_division"))
	}
	inv := new(big.Int).ModInverse(normA, d.p)
	if inv == nil {
		return nil, errors.New(i18n.T("domain.err_no_inverse", a.String()))
	}
	return inv, nil
}

func (d *FiniteFieldDomain) Div(a, b *big.Int) (*big.Int, error) {
	inv, err := d.Inv(b)
	if err != nil {
		return nil, err
	}
	return d.Mul(a, inv), nil
}

func (d *FiniteFieldDomain) String(a *big.Int) string {
	if a == nil {
		return "0"
	}
	return d.normalize(a).String()
}

func (d *FiniteFieldDomain) Clone(a *big.Int) *big.Int {
	if a == nil {
		return d.Zero()
	}
	return new(big.Int).Set(d.normalize(a))
}
