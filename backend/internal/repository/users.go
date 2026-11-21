package repository

import (
	"context"
	"fmt"

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
