package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// SecretRepo — реализация storage.SecretRepository поверх PostgreSQL.
type SecretRepo struct {
	pool *pgxpool.Pool
}

// NewSecretRepo создаёт SecretRepo на основе готового пула соединений.
func NewSecretRepo(pool *pgxpool.Pool) *SecretRepo {
	return &SecretRepo{pool: pool}
}

// Create реализует storage.SecretRepository.
func (r *SecretRepo) Create(ctx context.Context, s *model.Secret) (int64, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO secrets (user_id, type, data, metadata)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, version, created_at, updated_at`,
		s.UserID, s.Type, s.Data, s.Metadata,
	).Scan(&s.ID, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	return s.ID, err
}

// GetByID реализует storage.SecretRepository.
func (r *SecretRepo) GetByID(ctx context.Context, userID, id int64) (*model.Secret, error) {
	var s model.Secret
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, type, data, metadata, version, created_at, updated_at
		 FROM secrets
		 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id, userID,
	).Scan(&s.ID, &s.UserID, &s.Type, &s.Data, &s.Metadata, &s.Version, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrSecretNotFound
		}
		return nil, err
	}
	return &s, nil
}

// ListByUser реализует storage.SecretRepository.
func (r *SecretRepo) ListByUser(ctx context.Context, userID int64) ([]*model.Secret, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, type, data, metadata, version, created_at, updated_at
		 FROM secrets
		 WHERE user_id = $1 AND deleted_at IS NULL
		 ORDER BY id`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Secret
	for rows.Next() {
		var s model.Secret
		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &s.Data, &s.Metadata, &s.Version, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &s)
	}
	return result, rows.Err()
}

// Update реализует storage.SecretRepository. Использует version как условие
// optimistic concurrency: обновление проходит, только если версия в базе
// совпадает с той, что передал клиент.
func (r *SecretRepo) Update(ctx context.Context, userID int64, s *model.Secret) error {
	var newVersion int
	err := r.pool.QueryRow(ctx,
		`UPDATE secrets
		 SET data = $1, metadata = $2, version = version + 1, updated_at = now()
		 WHERE id = $3 AND user_id = $4 AND version = $5 AND deleted_at IS NULL
		 RETURNING version`,
		s.Data, s.Metadata, s.ID, userID, s.Version,
	).Scan(&newVersion)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storage.ErrVersionConflict
		}
		return err
	}
	s.Version = newVersion
	return nil
}

// Delete реализует storage.SecretRepository. Это soft delete — строка
// остаётся в таблице с проставленным deleted_at, чтобы удаление можно было
// синхронизировать на другие клиенты пользователя
func (r *SecretRepo) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE secrets SET deleted_at = $1 WHERE id = $2 AND user_id = $3 AND deleted_at IS NULL`,
		time.Now(), id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return storage.ErrSecretNotFound
	}
	return nil
}
