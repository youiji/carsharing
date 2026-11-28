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

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

const userColumns = `id, email, password_hash, full_name, driver_license, role, created_at`

func scanUser(row pgx.Row) (models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.DriverLicense, &u.Role, &u.CreatedAt)
	return u, err
}

func (r *UserRepository) Create(ctx context.Context, u models.User) (models.User, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, driver_license, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+userColumns,
		u.Email, u.PasswordHash, u.FullName, u.DriverLicense, u.Role)
	created, err := scanUser(row)
	if err != nil {
		return models.User{}, fmt.Errorf("create user: %w", util.MapPgError(err))
	}
	return created, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (models.User, error) {
	u, err := scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if err != nil {
		return models.User{}, fmt.Errorf("get user by id: %w", notFound(err))
	}
	return u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	u, err := scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
	if err != nil {
		return models.User{}, fmt.Errorf("get user by email: %w", notFound(err))
	}
	return u, nil
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id int64, hash string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, hash, id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

var UserSortNames = []string{"id", "email", "full_name", "created_at"}

var userSortColumns = map[string]string{
	"id": "id", "email": "email", "full_name": "full_name", "created_at": "created_at",
}

func (r *UserRepository) List(ctx context.Context, f models.UserFilter) ([]models.User, error) {
	var (
		where []string
		args  []any
	)
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if f.Role != nil {
		where = append(where, "role = "+arg(*f.Role))
	}
	if f.Search != "" {
		like := arg(util.LikeContains(f.Search))
		where = append(where, fmt.Sprintf("(email ILIKE %[1]s OR full_name ILIKE %[1]s)", like))
	}

	query := `SELECT ` + userColumns + ` FROM users`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	col, ok := userSortColumns[f.Sort]
	if !ok {
		col = "id"
	}
	dir := "ASC"
	if f.Desc {
		dir = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s, id %s", col, dir, dir)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var out []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", util.MapPgDeleteError(err))
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) InTx(ctx context.Context, fn func(tx *UserTx) error) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		return fn(&UserTx{tx: tx})
	})
}

type UserTx struct {
	tx pgx.Tx
}

// FOR NO KEY UPDATE конфликтует с FOR SHARE, который берёт бронирование
// на строку пользователя: смена роли и бронирование идут строго по очереди.
func (t *UserTx) Lock(ctx context.Context, id int64) (models.User, error) {
	u, err := scanUser(t.tx.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1 FOR NO KEY UPDATE`, id))
	if err != nil {
		return models.User{}, fmt.Errorf("lock user: %w", notFound(err))
	}
	return u, nil
}

func (t *UserTx) HasOpenRental(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := t.tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM rentals WHERE user_id = $1 AND status IN ('booked', 'active'))`,
		id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check open rental: %w", err)
	}
	return exists, nil
}

func (t *UserTx) Save(ctx context.Context, u models.User) (models.User, error) {
	saved, err := scanUser(t.tx.QueryRow(ctx, `
		UPDATE users SET role = $2, full_name = $3, driver_license = $4, email = $5
		WHERE id = $1
		RETURNING `+userColumns,
		u.ID, u.Role, u.FullName, u.DriverLicense, u.Email))
	if err != nil {
		return models.User{}, fmt.Errorf("save user: %w", util.MapPgError(err))
	}
	return saved, nil
}
