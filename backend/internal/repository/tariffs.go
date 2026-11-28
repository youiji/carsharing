package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/models"
	"carsharing/internal/util"
)

type TariffRepository struct {
	db *pgxpool.Pool
}

func NewTariffRepository(db *pgxpool.Pool) *TariffRepository {
	return &TariffRepository{db: db}
}

const tariffColumns = `id, name, price_per_min, waiting_per_min`

func scanTariff(row pgx.Row) (models.Tariff, error) {
	var t models.Tariff
	err := row.Scan(&t.ID, &t.Name, &t.PricePerMin, &t.WaitingPerMin)
	return t, err
}

func (r *TariffRepository) List(ctx context.Context) ([]models.Tariff, error) {
	rows, err := r.db.Query(ctx, `SELECT `+tariffColumns+` FROM tariffs ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list tariffs: %w", err)
	}
	defer rows.Close()

	var out []models.Tariff
	for rows.Next() {
		t, err := scanTariff(rows)
		if err != nil {
			return nil, fmt.Errorf("scan tariff: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TariffRepository) Create(ctx context.Context, t models.Tariff) (models.Tariff, error) {
	created, err := scanTariff(r.db.QueryRow(ctx, `
		INSERT INTO tariffs (name, price_per_min, waiting_per_min)
		VALUES ($1, $2, $3)
		RETURNING `+tariffColumns, t.Name, t.PricePerMin, t.WaitingPerMin))
	if err != nil {
		return models.Tariff{}, fmt.Errorf("create tariff: %w", util.MapPgError(err))
	}
	return created, nil
}

func (r *TariffRepository) Update(ctx context.Context, t models.Tariff) (models.Tariff, error) {
	updated, err := scanTariff(r.db.QueryRow(ctx, `
		UPDATE tariffs SET name = $2, price_per_min = $3, waiting_per_min = $4
		WHERE id = $1
		RETURNING `+tariffColumns, t.ID, t.Name, t.PricePerMin, t.WaitingPerMin))
	if err != nil {
		return models.Tariff{}, fmt.Errorf("update tariff: %w", notFound(util.MapPgError(err)))
	}
	return updated, nil
}

func (r *TariffRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM tariffs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tariff: %w", util.MapPgDeleteError(err))
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
