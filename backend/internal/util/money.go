package util

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

type Money int64

func (m Money) String() string {
	sign := ""
	v := int64(m)
	if v < 0 {
		sign, v = "-", -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *Money) ScanNumeric(n pgtype.Numeric) error {
	if !n.Valid {
		return errors.New("money: NULL нельзя прочитать в Money, используй *Money")
	}

	if n.NaN || n.InfinityModifier != pgtype.Finite {
		return errors.New("money: недопустимое значение NUMERIC")
	}
	v := new(big.Int).Set(n.Int)
	shift := int64(n.Exp) + 2
	ten := big.NewInt(10)
	if shift >= 0 {
		v.Mul(v, new(big.Int).Exp(ten, big.NewInt(shift), nil))
	} else {
		div := new(big.Int).Exp(ten, big.NewInt(-shift), nil)
		var rem big.Int
		v.QuoRem(v, div, &rem)
		if rem.Sign() != 0 {
			return errors.New("money: в значении больше двуч знаков полсе запятой")
		}
	}
	if !v.IsInt64() {
		return errors.New("money: занчение слишком велико")
	}
	*m = Money(v.Int64())
	return nil
}

func (m Money) NumericValue() (pgtype.Numeric, error) {
	return pgtype.Numeric{Int: big.NewInt(int64(m)), Exp: -2, Valid: true}, nil
}
