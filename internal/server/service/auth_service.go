package service

import (
	"context"
	"errors"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/auth"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// ErrInvalidCredentials возвращается при неверном логине или пароле.
var ErrInvalidCredentials = errors.New("invalid login or password")

// AuthService реализует регистрацию и аутентификацию пользователей.
type AuthService struct {
	users storage.UserRepository
	jwt   *auth.JWTManager
}

// NewAuthService создаёт AuthService.
func NewAuthService(users storage.UserRepository, jwt *auth.JWTManager) *AuthService {
	return &AuthService{users: users, jwt: jwt}
}

// Register регистрирует пользователя с уже готовым клиентским крипто-материалом
// (соль, параметры KDF, обёрнутый DEK) и возвращает токен доступа.
func (s *AuthService) Register(ctx context.Context, login, password string, kdfSalt []byte, params model.KDFParams, wrappedDEK []byte) (string, error) {
	if len(kdfSalt) == 0 || len(wrappedDEK) == 0 {
		return "", errors.New("missing crypto material")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", err
	}

	id, err := s.users.Create(ctx, &model.User{
		Login:        login,
		PasswordHash: hash,
		KDFSalt:      kdfSalt,
		KDFParams:    params,
		WrappedDEK:   wrappedDEK,
	})
	if err != nil {
		return "", err
	}

	return s.jwt.Generate(id)
}

// Login проверяет учётные данные и возвращает токен доступа вместе
// с крипто-материалом, нужным клиенту для восстановления DEK.
func (s *AuthService) Login(ctx context.Context, login, password string) (token string, salt []byte, params model.KDFParams, wrappedDEK []byte, err error) {
	u, getErr := s.users.GetByLogin(ctx, login)
	if getErr != nil {
		if errors.Is(getErr, storage.ErrUserNotFound) {
			return "", nil, model.KDFParams{}, nil, ErrInvalidCredentials
		}
		return "", nil, model.KDFParams{}, nil, getErr
	}

	if cmpErr := auth.ComparePassword(u.PasswordHash, password); cmpErr != nil {
		return "", nil, model.KDFParams{}, nil, ErrInvalidCredentials
	}

	token, err = s.jwt.Generate(u.ID)
	return token, u.KDFSalt, u.KDFParams, u.WrappedDEK, err
}
