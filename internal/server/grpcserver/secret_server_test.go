package grpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	secretv1 "github.com/SatzhanDev/gophKeeper/api/proto/secret/v1"
	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/service"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// fakeSecretService — реализация secretServiceIface для тестов хендлера,
// без реального SecretService и базы данных.
type fakeSecretService struct {
	createSecret *model.Secret
	createErr    error

	getSecret *model.Secret
	getErr    error

	listSecrets []*model.Secret
	listErr     error

	updateVersion int
	updateErr     error

	deleteErr error
}

func (f *fakeSecretService) Create(context.Context, int64, model.SecretType, []byte, string) (*model.Secret, error) {
	return f.createSecret, f.createErr
}
func (f *fakeSecretService) Get(context.Context, int64, int64) (*model.Secret, error) {
	return f.getSecret, f.getErr
}
func (f *fakeSecretService) List(context.Context, int64) ([]*model.Secret, error) {
	return f.listSecrets, f.listErr
}
func (f *fakeSecretService) Update(context.Context, int64, int64, []byte, string, int) (int, error) {
	return f.updateVersion, f.updateErr
}
func (f *fakeSecretService) Delete(context.Context, int64, int64) error {
	return f.deleteErr
}

// authedCtx возвращает контекст с userID, как если бы его туда положил
// AuthInterceptor после успешной проверки токена.
func authedCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), ctxKeyUserID{}, userID)
}

func TestSecretServer_CreateSecret_Unauthenticated(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{})

	_, err := srv.CreateSecret(context.Background(), &secretv1.CreateSecretRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestSecretServer_CreateSecret_Success(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{createSecret: &model.Secret{ID: 5}})

	resp, err := srv.CreateSecret(authedCtx(1), &secretv1.CreateSecretRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(5), resp.GetId())
}

func TestSecretServer_CreateSecret_InvalidType(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{createErr: service.ErrInvalidSecretType})

	_, err := srv.CreateSecret(authedCtx(1), &secretv1.CreateSecretRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSecretServer_CreateSecret_InternalError(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{createErr: errors.New("db is down")})

	_, err := srv.CreateSecret(authedCtx(1), &secretv1.CreateSecretRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestSecretServer_GetSecret_Unauthenticated(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{})

	_, err := srv.GetSecret(context.Background(), &secretv1.GetSecretRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestSecretServer_GetSecret_NotFound(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{getErr: storage.ErrSecretNotFound})

	_, err := srv.GetSecret(authedCtx(1), &secretv1.GetSecretRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestSecretServer_GetSecret_Success(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{getSecret: &model.Secret{ID: 1, Metadata: "site.com"}})

	resp, err := srv.GetSecret(authedCtx(1), &secretv1.GetSecretRequest{Id: 1})
	require.NoError(t, err)
	assert.Equal(t, "site.com", resp.GetSecret().GetMetadata())
}

func TestSecretServer_ListSecrets_Success(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{listSecrets: []*model.Secret{{ID: 1}, {ID: 2}}})

	resp, err := srv.ListSecrets(authedCtx(1), &secretv1.ListSecretsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.GetSecrets(), 2)
}

func TestSecretServer_ListSecrets_InternalError(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{listErr: errors.New("db is down")})

	_, err := srv.ListSecrets(authedCtx(1), &secretv1.ListSecretsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestSecretServer_UpdateSecret_VersionConflict(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{updateErr: storage.ErrVersionConflict})

	_, err := srv.UpdateSecret(authedCtx(1), &secretv1.UpdateSecretRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Aborted, status.Code(err))
}

func TestSecretServer_UpdateSecret_Success(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{updateVersion: 3})

	resp, err := srv.UpdateSecret(authedCtx(1), &secretv1.UpdateSecretRequest{Id: 1})
	require.NoError(t, err)
	assert.Equal(t, int32(3), resp.GetVersion())
}

func TestSecretServer_DeleteSecret_Success(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{})

	_, err := srv.DeleteSecret(authedCtx(1), &secretv1.DeleteSecretRequest{Id: 1})
	assert.NoError(t, err)
}

func TestSecretServer_DeleteSecret_NotFound(t *testing.T) {
	srv := NewSecretServer(&fakeSecretService{deleteErr: storage.ErrSecretNotFound})

	_, err := srv.DeleteSecret(authedCtx(1), &secretv1.DeleteSecretRequest{Id: 1})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}
