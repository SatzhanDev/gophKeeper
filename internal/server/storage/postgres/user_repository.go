package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// uniqueViolationCode — код ошибки Postgres при нарушении UNIQUE-constraint.
const uniqueViolationCode = "23505"

// UserRepo — реализация storage.UserRepository поверх PostgreSQL.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo создаёт UserRepo на основе готового пула соединений.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Create реализует storage.UserRepository.
func (r *UserRepo) Create(ctx context.Context, u *model.User) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (login, password_hash, kdf_salt, kdf_time, kdf_memory_kb, kdf_threads, wrapped_dek)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		u.Login, u.PasswordHash, u.KDFSalt, u.KDFParams.Time, u.KDFParams.MemoryKB, u.KDFParams.Threads, u.WrappedDEK,
	).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return 0, storage.ErrLoginTaken
		}
		return 0, err
	}
	return id, nil
}

// GetByLogin реализует storage.UserRepository.
func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	var u model.User
	var kdfTime, kdfMemoryKB int32
	var kdfThreads int16

	err := r.pool.QueryRow(ctx,
		`SELECT id, login, password_hash, kdf_salt, kdf_time, kdf_memory_kb, kdf_threads, wrapped_dek, created_at
		 FROM users WHERE login = $1`,
		login,
	).Scan(
		&u.ID, &u.Login, &u.PasswordHash, &u.KDFSalt,
		&kdfTime, &kdfMemoryKB, &kdfThreads,
		&u.WrappedDEK, &u.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrUserNotFound
		}
		return nil, err
	}

	u.KDFParams = model.KDFParams{
		Time:     uint32(kdfTime),
		MemoryKB: uint32(kdfMemoryKB),
		Threads:  uint8(kdfThreads),
	}
	return &u, nil
}
