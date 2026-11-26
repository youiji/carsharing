package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/models"
	"carsharing/internal/util"
)

type RentalRepository struct {
	db *pgxpool.Pool
}

type RentalTx struct {
	tx pgx.Tx
}

type CarForBooking struct {
	ID            int64
	Status        string
	TariffID      int64
	PricePerMin   util.Money
	WaitingPerMin util.Money
}

func NewRentalRepository(db *pgxpool.Pool) *RentalRepository {
	return &RentalRepository{db: db}
}

const rentalSelect = `
	SELECT r.id, r.user_id, r.tariff_id, r.car_id, r.booked_at, r.started_at, r.ended_at,
	       r.price_per_min, r.waiting_per_min, r.total_cost, r.status,
	       u.full_name, c.brand_model, c.plate_number
	FROM rentals r
	JOIN users u ON u.id = r.user_id
	JOIN cars  c ON c.id = r.car_id`

func scanRental(row pgx.Row) (models.Rental, error) {
	var r models.Rental
	err := row.Scan(&r.ID, &r.UserID, &r.TariffID, &r.CarID, &r.BookedAt, &r.StartedAt, &r.EndedAt,
		&r.PricePerMin, &r.WaitingPerMin, &r.TotalCost, &r.Status,
		&r.UserFullName, &r.CarBrandModel, &r.CarPlateNumber)
	return r, err
}

func (r *RentalRepository) GetByID(ctx context.Context, id int64) (models.Rental, error) {
	rental, err := scanRental(r.db.QueryRow(ctx, rentalSelect+` WHERE r.id = $1`, id))
	if err != nil {
		return models.Rental{}, fmt.Errorf("get rental: %w", notFound(err))
	}
	return rental, nil
}

func (r *RentalRepository) InTx(ctx context.Context, fn func(tx *RentalTx) error) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		return fn(&RentalTx{tx: tx})
	})
}

func (t *RentalTx) LockCar(ctx context.Context, carID int64) (CarForBooking, error) {
	var c CarForBooking
	err := t.tx.QueryRow(ctx, `
		SELECT c.id, c.status, c.tariff_id, tr.price_per_min, tr.waiting_per_min
		FROM cars c
		JOIN tariffs tr ON tr.id = c.tariff_id
		WHERE c.id = $1
		FOR UPDATE OF c`, carID).
		Scan(&c.ID, &c.Status, &c.TariffID, &c.PricePerMin, &c.WaitingPerMin)
	if err != nil {
		return CarForBooking{}, fmt.Errorf("lock car: %w", notFound(err))
	}
	return c, nil
}

func (t *RentalTx) LockUser(ctx context.Context, userID int64) (role string, license *string, err error) {
	err = t.tx.QueryRow(ctx, `SELECT role, driver_license FROM users WHERE id = $1 FOR SHARE`, userID).
		Scan(&role, &license)
	if err != nil {
		return "", nil, fmt.Errorf("lock user: %w", notFound(err))
	}
	return role, license, nil
}

func (t *RentalTx) UserHasOpenRental(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := t.tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM rentals WHERE user_id = $1 AND status IN ('booked', 'active'))`,
		userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check open rental: %w", err)
	}
	return exists, nil
}

func (t *RentalTx) Insert(ctx context.Context, r models.Rental) (int64, error) {
	var id int64
	err := t.tx.QueryRow(ctx, `
		INSERT INTO rentals (user_id, tariff_id, car_id, booked_at, price_per_min, waiting_per_min, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'booked')
		RETURNING id`,
		r.UserID, r.TariffID, r.CarID, r.BookedAt, r.PricePerMin, r.WaitingPerMin).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert rental: %w", util.MapPgError(err))
	}
	return id, nil
}

func (t *RentalTx) SetCarStatus(ctx context.Context, carID int64, from, to string) error {
	return t.execOne(ctx, "set car status", `
		UPDATE cars SET status = $3 WHERE id = $1 AND status = $2`, carID, from, to)
}

func (t *RentalTx) execOne(ctx context.Context, op, sql string, args ...any) error {
	tag, err := t.tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, util.MapPgError(err))
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s: изменено строк %d вместо 1 (статусы аренды и автомобиля рассогласованы)",
			op, tag.RowsAffected())
	}
	return nil
}

func (t *RentalTx) LockRental(ctx context.Context, id int64) (models.Rental, error) {
	var r models.Rental
	err := t.tx.QueryRow(ctx, `
		SELECT id, user_id, tariff_id, car_id, booked_at, started_at, ended_at,
		       price_per_min, waiting_per_min, total_cost, status
		FROM rentals WHERE id = $1
		FOR UPDATE`, id).
		Scan(&r.ID, &r.UserID, &r.TariffID, &r.CarID, &r.BookedAt, &r.StartedAt, &r.EndedAt,
			&r.PricePerMin, &r.WaitingPerMin, &r.TotalCost, &r.Status)
	if err != nil {
		return models.Rental{}, fmt.Errorf("lock rental: %w", notFound(err))
	}
	return r, nil
}

func (t *RentalTx) Start(ctx context.Context, id int64, startedAt time.Time) error {
	return t.execOne(ctx, "start rental", `
		UPDATE rentals SET status = 'active', started_at = $2
		WHERE id = $1 AND status = 'booked'`, id, startedAt)
}

func (t *RentalTx) Complete(ctx context.Context, id int64, endedAt time.Time, cost util.Money) error {
	return t.execOne(ctx, "complete rental", `
		UPDATE rentals SET status = 'completed', ended_at = $2, total_cost = $3
		WHERE id = $1 AND status = 'active'`, id, endedAt, cost)
}

func (t *RentalTx) Cancel(ctx context.Context, id int64, cancelledAt time.Time, fee util.Money) error {
	return t.execOne(ctx, "cancel rental", `
		UPDATE rentals SET status = 'cancelled', ended_at = $2, total_cost = $3
		WHERE id = $1 AND status = 'booked'`, id, cancelledAt, fee)
}

func (t *RentalTx) ReleaseCar(ctx context.Context, carID int64, loc models.CarLocation) error {
	return t.execOne(ctx, "release car", `
		UPDATE cars SET status    = 'available',
		                latitude  = COALESCE($2, latitude),
		                longitude = COALESCE($3, longitude)
		WHERE id = $1 AND status = 'in_use'`,
		carID, loc.Latitude, loc.Longitude)
}
