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

type MaintenanceRepository struct {
	db *pgxpool.Pool
}

func NewMaintenanceRepository(db *pgxpool.Pool) *MaintenanceRepository {
	return &MaintenanceRepository{db: db}
}

var MaintenanceSortNames = []string{"id", "logged_at", "cost"}

var maintenanceSortColumns = map[string]string{
	"id":        "m.id",
	"logged_at": "m.logged_at",
	"cost":      "m.cost",
}

const maintenanceSelect = `
	SELECT m.id, m.car_id, m.manager_id, m.description, m.cost, m.logged_at,
	       c.brand_model, c.plate_number, u.full_name
	FROM maintenance_logs m
	JOIN cars  c ON c.id = m.car_id
	JOIN users u ON u.id = m.manager_id`

func scanMaintenance(row pgx.Row) (models.MaintenanceLog, error) {
	var m models.MaintenanceLog
	err := row.Scan(&m.ID, &m.CarID, &m.ManagerID, &m.Description, &m.Cost, &m.LoggedAt,
		&m.CarBrandModel, &m.CarPlateNumber, &m.ManagerFullName)
	return m, err
}

func (r *MaintenanceRepository) List(ctx context.Context, f models.MaintenanceFilter) ([]models.MaintenanceLog, error) {
	var (
		where []string
		args  []any
	)
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if f.CarID != nil {
		where = append(where, "m.car_id = "+arg(*f.CarID))
	}
	if f.Search != "" {
		like := arg(util.LikeContains(f.Search))
		plate := arg(util.LikeContains(util.NormalizePlate(f.Search)))
		where = append(where, fmt.Sprintf(
			"(m.description ILIKE %[1]s OR c.brand_model ILIKE %[1]s OR c.plate_number ILIKE %[2]s)", like, plate))
	}

	query := maintenanceSelect
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	col, ok := maintenanceSortColumns[f.Sort]
	if !ok {
		col = maintenanceSortColumns["id"]
	}
	dir := "ASC"
	if f.Desc {
		dir = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s, m.id %s", col, dir, dir)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list maintenance: %w", err)
	}
	defer rows.Close()

	var out []models.MaintenanceLog
	for rows.Next() {
		m, err := scanMaintenance(rows)
		if err != nil {
			return nil, fmt.Errorf("scan maintenance: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MaintenanceRepository) GetByID(ctx context.Context, id int64) (models.MaintenanceLog, error) {
	m, err := scanMaintenance(r.db.QueryRow(ctx, maintenanceSelect+` WHERE m.id = $1`, id))
	if err != nil {
		return models.MaintenanceLog{}, fmt.Errorf("get maintenance: %w", notFound(err))
	}
	return m, nil
}

func (r *MaintenanceRepository) Create(ctx context.Context, m models.MaintenanceLog) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO maintenance_logs (car_id, manager_id, description, cost)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		m.CarID, m.ManagerID, m.Description, m.Cost).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create maintenance: %w", util.MapPgError(err))
	}
	return id, nil
}

// nil-параметр → COALESCE оставляет старое значение.
func (r *MaintenanceRepository) Update(ctx context.Context, id int64, description *string, cost *util.Money) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE maintenance_logs
		SET description = COALESCE($2, description),
		    cost        = COALESCE($3, cost)
		WHERE id = $1`, id, description, cost)
	if err != nil {
		return fmt.Errorf("update maintenance: %w", util.MapPgError(err))
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MaintenanceRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM maintenance_logs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete maintenance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
