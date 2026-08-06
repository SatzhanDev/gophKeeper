package storage

import (
	"context"
	"errors"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
)

// ErrSecretNotFound возвращается, если секрет с указанным ID не найден
// либо не принадлежит запрашивающему пользователю.
var ErrSecretNotFound = errors.New("secret not found")

// ErrVersionConflict возвращается при попытке обновить секрет с устаревшей
// версией (конфликт параллельного изменения).
var ErrVersionConflict = errors.New("version conflict")

// SecretRepository описывает доступ к хранилищу приватных данных.
type SecretRepository interface {
	Create(ctx context.Context, s *model.Secret) (int64, error)
	GetByID(ctx context.Context, userID, id int64) (*model.Secret, error)
	ListByUser(ctx context.Context, userID int64) ([]*model.Secret, error)
	Update(ctx context.Context, userID int64, s *model.Secret) error
	Delete(ctx context.Context, userID, id int64) error
}
