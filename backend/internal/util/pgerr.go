package util

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
)

var uniqueConstraintErrors = map[string]*AppError{
	"uq_users_email":                 ErrEmailTaken,
	"uq_users_driver_license":        ErrLicenseTaken,
	"uq_cars_plate_number":           ErrPlateTaken,
	"uq_tariffs_name":                ErrNameTaken,
	"uq_rentals_one_active_per_car":  ErrCarUnavailable,
	"uq_rentals_one_active_per_user": ErrUserHasActiveRental,
}

var foreignKeyFields = map[string]string{
	"fk_cars_tariff":    "tariff_id",
	"fk_rentals_user":   "user_id",
	"fk_rentals_tariff": "tariff_id",
	"fk_rentals_car":    "car_id",
	"fk_ml_car":         "car_id",
	"fk_ml_manager":     "manager_id",
}

func MapPgError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case pgUniqueViolation:
		if appErr, ok := uniqueConstraintErrors[pgErr.ConstraintName]; ok {
			return appErr
		}
		return ErrConflict
	case pgForeignKeyViolation:
		if field, ok := foreignKeyFields[pgErr.ConstraintName]; ok {
			return ErrValidation.WithDetails(map[string]string{field: "объект не найден"})
		}
		return ErrValidation.WithMessage("Ссылка на несуществующий объект")
	case pgCheckViolation:
		return ErrValidation
	}
	return err
}

func MapPgDeleteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return ErrHasRelatedRecords
	}
	return MapPgError(err)
}
