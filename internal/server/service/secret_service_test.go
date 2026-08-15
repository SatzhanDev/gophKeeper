package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// fakeSecretRepo — реализация storage.SecretRepository в памяти для тестов.
// Намеренно повторяет ключевое поведение реального Postgres-репозитория
// (фильтрацию по userID и проверку version при обновлении) — иначе тесты
// SecretService ничего не скажут о реальной защите от чужого доступа.
type fakeSecretRepo struct {
	byID   map[int64]*model.Secret
	nextID int64
}

func newFakeSecretRepo() *fakeSecretRepo {
	return &fakeSecretRepo{byID: make(map[int64]*model.Secret)}
}

func (f *fakeSecretRepo) Create(_ context.Context, s *model.Secret) (int64, error) {
	f.nextID++
	stored := *s
	stored.ID = f.nextID
	stored.Version = 1
	f.byID[stored.ID] = &stored
	s.ID = stored.ID
	s.Version = stored.Version
	return stored.ID, nil
}

func (f *fakeSecretRepo) GetByID(_ context.Context, userID, id int64) (*model.Secret, error) {
	s, ok := f.byID[id]
	if !ok || s.UserID != userID || s.DeletedAt != nil {
		return nil, storage.ErrSecretNotFound
	}
	return s, nil
}

func (f *fakeSecretRepo) ListByUser(_ context.Context, userID int64) ([]*model.Secret, error) {
	var result []*model.Secret
	for _, s := range f.byID {
		if s.UserID == userID && s.DeletedAt == nil {
			result = append(result, s)
		}
	}
	return result, nil
}

func (f *fakeSecretRepo) Update(_ context.Context, userID int64, s *model.Secret) error {
	existing, ok := f.byID[s.ID]
	if !ok || existing.UserID != userID || existing.DeletedAt != nil || existing.Version != s.Version {
		return storage.ErrVersionConflict
	}
	existing.Data = s.Data
	existing.Metadata = s.Metadata
	existing.Version++
	s.Version = existing.Version
	return nil
}

func (f *fakeSecretRepo) Delete(_ context.Context, userID, id int64) error {
	existing, ok := f.byID[id]
	if !ok || existing.UserID != userID || existing.DeletedAt != nil {
		return storage.ErrSecretNotFound
	}
	now := time.Now()
	existing.DeletedAt = &now
	return nil
}

func TestSecretService_Create_InvalidType(t *testing.T) {
	svc := NewSecretService(newFakeSecretRepo())

	_, err := svc.Create(context.Background(), 1, model.SecretType(99), []byte("data"), "meta")
	assert.ErrorIs(t, err, ErrInvalidSecretType)
}

func TestSecretService_Create_Success(t *testing.T) {
	svc := NewSecretService(newFakeSecretRepo())

	secret, err := svc.Create(context.Background(), 1, model.SecretTypeText, []byte("data"), "meta")
	require.NoError(t, err)
	assert.NotZero(t, secret.ID)
	assert.Equal(t, 1, secret.Version)
}

func TestSecretService_Create_AllFourTypesAreValid(t *testing.T) {
	svc := NewSecretService(newFakeSecretRepo())
	types := []model.SecretType{
		model.SecretTypeLoginPassword,
		model.SecretTypeText,
		model.SecretTypeBinary,
		model.SecretTypeCard,
	}

	for _, tp := range types {
		_, err := svc.Create(context.Background(), 1, tp, []byte("data"), "meta")
		assert.NoError(t, err, "тип %v должен быть валиден", tp)
	}
}

func TestSecretService_Get_WrongOwner(t *testing.T) {
	repo := newFakeSecretRepo()
	svc := NewSecretService(repo)
	ctx := context.Background()

	secret, err := svc.Create(ctx, 1, model.SecretTypeText, []byte("data"), "meta")
	require.NoError(t, err)

	_, err = svc.Get(ctx, 2, secret.ID) // другой userID
	assert.ErrorIs(t, err, storage.ErrSecretNotFound)
}

func TestSecretService_List_OnlyOwnSecrets(t *testing.T) {
	repo := newFakeSecretRepo()
	svc := NewSecretService(repo)
	ctx := context.Background()

	_, err := svc.Create(ctx, 1, model.SecretTypeText, []byte("a"), "meta-a")
	require.NoError(t, err)
	_, err = svc.Create(ctx, 2, model.SecretTypeText, []byte("b"), "meta-b")
	require.NoError(t, err)

	secrets, err := svc.List(ctx, 1)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "meta-a", secrets[0].Metadata)
}

func TestSecretService_Update_VersionConflict(t *testing.T) {
	repo := newFakeSecretRepo()
	svc := NewSecretService(repo)
	ctx := context.Background()

	secret, err := svc.Create(ctx, 1, model.SecretTypeText, []byte("data"), "meta")
	require.NoError(t, err)

	_, err = svc.Update(ctx, 1, secret.ID, []byte("new-data"), "meta", secret.Version+1)
	assert.ErrorIs(t, err, storage.ErrVersionConflict)
}

func TestSecretService_Update_Success(t *testing.T) {
	repo := newFakeSecretRepo()
	svc := NewSecretService(repo)
	ctx := context.Background()

	secret, err := svc.Create(ctx, 1, model.SecretTypeText, []byte("data"), "meta")
	require.NoError(t, err)

	newVersion, err := svc.Update(ctx, 1, secret.ID, []byte("new-data"), "new-meta", secret.Version)
	require.NoError(t, err)
	assert.Equal(t, secret.Version+1, newVersion)
}

func TestSecretService_Update_WrongOwner(t *testing.T) {
	repo := newFakeSecretRepo()
	svc := NewSecretService(repo)
	ctx := context.Background()

	secret, err := svc.Create(ctx, 1, model.SecretTypeText, []byte("data"), "meta")
	require.NoError(t, err)

	_, err = svc.Update(ctx, 2, secret.ID, []byte("new-data"), "meta", secret.Version)
	assert.ErrorIs(t, err, storage.ErrVersionConflict)
}

func TestSecretService_Delete_ThenGet_NotFound(t *testing.T) {
	repo := newFakeSecretRepo()
	svc := NewSecretService(repo)
	ctx := context.Background()

	secret, err := svc.Create(ctx, 1, model.SecretTypeText, []byte("data"), "meta")
	require.NoError(t, err)

	err = svc.Delete(ctx, 1, secret.ID)
	require.NoError(t, err)

	_, err = svc.Get(ctx, 1, secret.ID)
	assert.ErrorIs(t, err, storage.ErrSecretNotFound)
}

func TestSecretService_Delete_WrongOwner(t *testing.T) {
	repo := newFakeSecretRepo()
	svc := NewSecretService(repo)
	ctx := context.Background()

	secret, err := svc.Create(ctx, 1, model.SecretTypeText, []byte("data"), "meta")
	require.NoError(t, err)

	err = svc.Delete(ctx, 2, secret.ID)
	assert.ErrorIs(t, err, storage.ErrSecretNotFound)
}
