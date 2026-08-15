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

// SecretRepository описывает доступ к хранилищу приватных данных. Все
// методы принимают userID и сами отвечают за то, чтобы вернуть/изменить
// только секреты, принадлежащие этому пользователю — это защита от доступа
// к чужим данным на уровне хранилища, а не только сервисного слоя.
type SecretRepository interface {
	// Create сохраняет новый секрет и возвращает его ID.
	Create(ctx context.Context, s *model.Secret) (int64, error)

	// GetByID возвращает секрет пользователя userID по ID.
	// Возвращает ErrSecretNotFound, если секрета нет либо он принадлежит
	// другому пользователю.
	GetByID(ctx context.Context, userID, id int64) (*model.Secret, error)

	// ListByUser возвращает все неудалённые секреты пользователя userID.
	ListByUser(ctx context.Context, userID int64) ([]*model.Secret, error)

	// Update обновляет секрет пользователя userID, используя s.Version для
	// optimistic concurrency. Возвращает ErrVersionConflict, если версия
	// устарела либо секрет принадлежит другому пользователю.
	Update(ctx context.Context, userID int64, s *model.Secret) error

	// Delete помечает секрет пользователя userID удалённым (soft delete).
	// Возвращает ErrSecretNotFound, если секрета нет либо он принадлежит
	// другому пользователю.
	Delete(ctx context.Context, userID, id int64) error
}
