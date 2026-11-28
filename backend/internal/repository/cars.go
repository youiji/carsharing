package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/models"
	"carsharing/internal/util"
)

type CarRepository struct {
	db *pgxpool.Pool
}

func NewCarRepository(db *pgxpool.Pool) *CarRepository {
	return &CarRepository{db: db}
}

var CarSortNames = []string{"id", "brand_model", "plate_number", "fuel_level", "price_per_min"}

var carSortColumns = map[string]string{
	"id":            "c.id",
	"brand_model":   "c.brand_model",
	"plate_number":  "c.plate_number",
	"fuel_level":    "c.fuel_level",
	"price_per_min": "t.price_per_min",
}

const carSelect = `
	SELECT c.id, c.tariff_id, c.brand_model, c.plate_number, c.fuel_level,
	       c.latitude, c.longitude, c.status,
	       t.id, t.name, t.price_per_min, t.waiting_per_min
	FROM cars c
	JOIN tariffs t ON t.id = c.tariff_id`

func scanCar(row pgx.Row) (models.Car, error) {
	var c models.Car
	err := row.Scan(&c.ID, &c.TariffID, &c.BrandModel, &c.PlateNumber, &c.FuelLevel,
		&c.Latitude, &c.Longitude, &c.Status,
		&c.Tariff.ID, &c.Tariff.Name, &c.Tariff.PricePerMin, &c.Tariff.WaitingPerMin)
	return c, err
}

func (r *CarRepository) List(ctx context.Context, f models.CarFilter) ([]models.Car, error) {
	var (
		where []string
		args  []any
	)
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if f.Status != nil {
		where = append(where, "c.status = "+arg(*f.Status))
	}
	if f.TariffID != nil {
		where = append(where, "c.tariff_id = "+arg(*f.TariffID))
	}
	if f.Search != "" {
		like := arg(util.LikeContains(f.Search))
		plate := arg(util.LikeContains(util.NormalizePlate(f.Search)))
		where = append(where, fmt.Sprintf(
			"(c.brand_model ILIKE %[1]s OR c.plate_number ILIKE %[1]s OR c.plate_number ILIKE %[2]s)", like, plate))
	}

	query := carSelect
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	col, ok := carSortColumns[f.Sort]
	if !ok {
		col = carSortColumns["id"]
	}
	dir := "ASC"
	if f.Desc {
		dir = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s, c.id %s", col, dir, dir)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list cars: %w", err)
	}
	defer rows.Close()

	var out []models.Car
	for rows.Next() {
		c, err := scanCar(rows)
		if err != nil {
			return nil, fmt.Errorf("scan car: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CarRepository) GetByID(ctx context.Context, id int64) (models.Car, error) {
	c, err := scanCar(r.db.QueryRow(ctx, carSelect+` WHERE c.id = $1`, id))
	if err != nil {
		return models.Car{}, fmt.Errorf("get car: %w", notFound(err))
	}
	return c, nil
}

func (r *CarRepository) InUserOpenRental(ctx context.Context, carID, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM rentals
		               WHERE car_id = $1 AND user_id = $2 AND status IN ('booked', 'active'))`,
		carID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check car rental: %w", err)
	}
	return exists, nil
}

func (r *CarRepository) Create(ctx context.Context, c models.Car) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO cars (tariff_id, brand_model, plate_number, fuel_level, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		c.TariffID, c.BrandModel, c.PlateNumber, c.FuelLevel, c.Latitude, c.Longitude).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create car: %w", util.MapPgError(err))
	}
	return id, nil
}

func (r *CarRepository) Update(ctx context.Context, c models.Car) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE cars SET tariff_id = $2, brand_model = $3, plate_number = $4,
		                fuel_level = $5, latitude = $6, longitude = $7
		WHERE id = $1`,
		c.ID, c.TariffID, c.BrandModel, c.PlateNumber, c.FuelLevel, c.Latitude, c.Longitude)
	if err != nil {
		return fmt.Errorf("update car: %w", util.MapPgError(err))
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CarRepository) ChangeStatus(ctx context.Context, id int64, to string, check func(from string) error) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var from string
		err := tx.QueryRow(ctx, `SELECT status FROM cars WHERE id = $1 FOR UPDATE`, id).Scan(&from)
		if err != nil {
			return fmt.Errorf("lock car: %w", notFound(err))
		}
		if err := check(from); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE cars SET status = $2 WHERE id = $1`, id, to); err != nil {
			return fmt.Errorf("set car status: %w", util.MapPgError(err))
		}
		return nil
	})
}

func (r *CarRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM cars WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete car: %w", util.MapPgDeleteError(err))
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
