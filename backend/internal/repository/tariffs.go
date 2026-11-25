package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/models"
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
