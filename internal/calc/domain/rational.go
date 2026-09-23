package domain

import (
	"errors"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// RationalDomain represents the field of rational numbers Q.
// Elements are *big.Rat.
type RationalDomain struct{}

// NewRationalDomain creates a new instance of RationalDomain.
func NewRationalDomain() *RationalDomain {
	return &RationalDomain{}
}

func (d *RationalDomain) Name() string {
	return "Q"
}

func (d *RationalDomain) Zero() *big.Rat {
	return big.NewRat(0, 1)
}

func (d *RationalDomain) One() *big.Rat {
	return big.NewRat(1, 1)
}

func (d *RationalDomain) IsZero(a *big.Rat) bool {
	return a == nil || a.Sign() == 0
}

func (d *RationalDomain) IsOne(a *big.Rat) bool {
	if a == nil {
		return false
	}
	return a.Cmp(big.NewRat(1, 1)) == 0
}

func (d *RationalDomain) Equals(a, b *big.Rat) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Cmp(b) == 0
}

func (d *RationalDomain) Add(a, b *big.Rat) *big.Rat {
	res := new(big.Rat)
	if a == nil {
		a = d.Zero()
	}
	if b == nil {
		b = d.Zero()
	}
	return res.Add(a, b)
}

func (d *RationalDomain) Sub(a, b *big.Rat) *big.Rat {
	res := new(big.Rat)
	if a == nil {
		a = d.Zero()
	}
	if b == nil {
		b = d.Zero()
	}
	return res.Sub(a, b)
}

func (d *RationalDomain) Mul(a, b *big.Rat) *big.Rat {
	res := new(big.Rat)
	if a == nil || b == nil {
		return d.Zero()
	}
	return res.Mul(a, b)
}

func (d *RationalDomain) Neg(a *big.Rat) *big.Rat {
	if a == nil {
		return d.Zero()
	}
	res := new(big.Rat)
	return res.Neg(a)
}

func (d *RationalDomain) Inv(a *big.Rat) (*big.Rat, error) {
	if d.IsZero(a) {
		return nil, errors.New(i18n.T("domain.err_zero_division"))
	}
	res := new(big.Rat)
	return res.Inv(a), nil
}

func (d *RationalDomain) Div(a, b *big.Rat) (*big.Rat, error) {
	if d.IsZero(b) {
		return nil, errors.New(i18n.T("domain.err_zero_division"))
	}
	inv, err := d.Inv(b)
	if err != nil {
		return nil, err
	}
	return d.Mul(a, inv), nil
}

func (d *RationalDomain) String(a *big.Rat) string {
	if a == nil {
		return "0"
	}
	return a.RatString()
}

func (d *RationalDomain) Clone(a *big.Rat) *big.Rat {
	if a == nil {
		return d.Zero()
	}
	return new(big.Rat).Set(a)
}
