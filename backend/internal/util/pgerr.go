package util

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func MapPgError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "uq_users_email":
			return ErrEmailTaken
		case "uq_users_driver_license":
			return ErrLicenseTaken
		}
	}
	return err
}
