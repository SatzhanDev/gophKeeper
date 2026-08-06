package service

import (
	"context"
	"errors"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// ErrInvalidSecretType возвращается при попытке создать секрет
// с неизвестным типом.
var ErrInvalidSecretType = errors.New("invalid secret type")

// SecretService реализует CRUD-операции над приватными данными пользователя.
type SecretService struct {
	secrets storage.SecretRepository
}

// NewSecretService создаёт SecretService.
func NewSecretService(secrets storage.SecretRepository) *SecretService {
	return &SecretService{secrets: secrets}
}

func validSecretType(t model.SecretType) bool {
	return t >= model.SecretTypeLoginPassword && t <= model.SecretTypeCard
}

// Create создаёт новый секрет для пользователя userID.
func (s *SecretService) Create(ctx context.Context, userID int64, t model.SecretType, data []byte, metadata string) (*model.Secret, error) {
	if !validSecretType(t) {
		return nil, ErrInvalidSecretType
	}
	secret := &model.Secret{UserID: userID, Type: t, Data: data, Metadata: metadata}
	if _, err := s.secrets.Create(ctx, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// Get возвращает секрет пользователя userID по ID.
func (s *SecretService) Get(ctx context.Context, userID, id int64) (*model.Secret, error) {
	return s.secrets.GetByID(ctx, userID, id)
}

// List возвращает все секреты пользователя userID.
func (s *SecretService) List(ctx context.Context, userID int64) ([]*model.Secret, error) {
	return s.secrets.ListByUser(ctx, userID)
}

// Update обновляет секрет пользователя userID, проверяя версию
// на конфликт параллельного изменения.
func (s *SecretService) Update(ctx context.Context, userID, id int64, data []byte, metadata string, version int) (int, error) {
	secret := &model.Secret{ID: id, Data: data, Metadata: metadata, Version: version}
	if err := s.secrets.Update(ctx, userID, secret); err != nil {
		return 0, err
	}
	return secret.Version, nil
}

// Delete удаляет (soft delete) секрет пользователя userID.
func (s *SecretService) Delete(ctx context.Context, userID, id int64) error {
	return s.secrets.Delete(ctx, userID, id)
}
