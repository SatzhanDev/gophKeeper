package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/auth"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// fakeUserRepo — реализация storage.UserRepository в памяти для тестов,
// без обращения к реальной базе данных.
type fakeUserRepo struct {
	byLogin map[string]*model.User
	nextID  int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byLogin: make(map[string]*model.User)}
}

func (f *fakeUserRepo) Create(_ context.Context, u *model.User) (int64, error) {
	if _, exists := f.byLogin[u.Login]; exists {
		return 0, storage.ErrLoginTaken
	}
	f.nextID++
	stored := *u
	stored.ID = f.nextID
	f.byLogin[u.Login] = &stored
	return stored.ID, nil
}

func (f *fakeUserRepo) GetByLogin(_ context.Context, login string) (*model.User, error) {
	u, ok := f.byLogin[login]
	if !ok {
		return nil, storage.ErrUserNotFound
	}
	return u, nil
}

func newTestAuthService() *AuthService {
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	return NewAuthService(newFakeUserRepo(), jwtManager)
}

func TestAuthService_Register_Success(t *testing.T) {
	svc := newTestAuthService()
	ctx := context.Background()

	token, err := svc.Register(ctx, "ivan", "password123", []byte("salt"),
		model.KDFParams{Time: 1, MemoryKB: 64 * 1024, Threads: 4}, []byte("wrapped-dek"))
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthService_Register_DuplicateLogin(t *testing.T) {
	svc := newTestAuthService()
	ctx := context.Background()
	params := model.KDFParams{Time: 1, MemoryKB: 64 * 1024, Threads: 4}

	_, err := svc.Register(ctx, "ivan", "password123", []byte("salt"), params, []byte("wrapped-dek"))
	require.NoError(t, err)

	_, err = svc.Register(ctx, "ivan", "another-password", []byte("salt"), params, []byte("wrapped-dek"))
	assert.ErrorIs(t, err, storage.ErrLoginTaken)
}

func TestAuthService_Register_MissingCryptoMaterial(t *testing.T) {
	svc := newTestAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "ivan", "password123", nil, model.KDFParams{}, nil)
	assert.Error(t, err)
}

func TestAuthService_Login_Success(t *testing.T) {
	svc := newTestAuthService()
	ctx := context.Background()
	params := model.KDFParams{Time: 1, MemoryKB: 64 * 1024, Threads: 4}

	_, err := svc.Register(ctx, "ivan", "password123", []byte("salt"), params, []byte("wrapped-dek"))
	require.NoError(t, err)

	token, salt, gotParams, wrappedDEK, err := svc.Login(ctx, "ivan", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, []byte("salt"), salt)
	assert.Equal(t, params, gotParams)
	assert.Equal(t, []byte("wrapped-dek"), wrappedDEK)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc := newTestAuthService()
	ctx := context.Background()
	params := model.KDFParams{Time: 1, MemoryKB: 64 * 1024, Threads: 4}

	_, err := svc.Register(ctx, "ivan", "password123", []byte("salt"), params, []byte("wrapped-dek"))
	require.NoError(t, err)

	_, _, _, _, err = svc.Login(ctx, "ivan", "wrong-password")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_UnknownUser(t *testing.T) {
	svc := newTestAuthService()
	ctx := context.Background()

	_, _, _, _, err := svc.Login(ctx, "no-such-user", "password123")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_UnknownUserAndWrongPassword_SameError(t *testing.T) {
	// Важное свойство безопасности: неизвестный логин и неверный пароль
	// должны давать одну и ту же ошибку, чтобы нельзя было по разнице
	// в ответе узнавать, какие логины вообще зарегистрированы.
	svc := newTestAuthService()
	ctx := context.Background()
	params := model.KDFParams{Time: 1, MemoryKB: 64 * 1024, Threads: 4}

	_, err := svc.Register(ctx, "ivan", "password123", []byte("salt"), params, []byte("wrapped-dek"))
	require.NoError(t, err)

	_, _, _, _, errUnknown := svc.Login(ctx, "no-such-user", "password123")
	_, _, _, _, errWrongPass := svc.Login(ctx, "ivan", "wrong-password")

	assert.Equal(t, errUnknown, errWrongPass)
}
