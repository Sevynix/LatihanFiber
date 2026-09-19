package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tugas2/app/model"
)

type UserRepository interface {
	Create(ctx context.Context, u model.User) (model.User, error)
	FindByID(ctx context.Context, id int) (model.User, error)
	FindByUsername(ctx context.Context, username string) (model.User, error)
}

type userPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userPostgresRepository{pool: pool}
}

const userColumns = "id, username, email, password_hash, role, is_active, created_at"

func scanUser(row pgx.Row) (model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.Role, &u.IsActive, &u.CreatedAt)
	return u, err
}

func (r *userPostgresRepository) Create(
	ctx context.Context, u model.User,
) (model.User, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, role, is_active)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, created_at`,
		u.Username, u.Email, u.PasswordHash, u.Role, u.IsActive,
	).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.User{}, ErrDuplicate
		}
		return model.User{}, fmt.Errorf("menyimpan user: %w", err)
	}
	return u, nil
}

func (r *userPostgresRepository) FindByID(
	ctx context.Context, id int,
) (model.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx,
		"SELECT "+userColumns+" FROM users WHERE id = $1", id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}
	return u, nil
}

func (r *userPostgresRepository) FindByUsername(
	ctx context.Context, username string,
) (model.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx,
		"SELECT "+userColumns+" FROM users WHERE LOWER(username) = LOWER($1)",
		username))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}
	return u, nil
}