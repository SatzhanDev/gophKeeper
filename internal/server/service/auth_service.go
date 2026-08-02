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

// Register регистрирует нового пользователя и возвращает токен доступа.
func (s *AuthService) Register(ctx context.Context, login, password string) (string, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", err
	}

	id, err := s.users.Create(ctx, &model.User{Login: login, PasswordHash: hash})
	if err != nil {
		return "", err // storage.ErrLoginTaken пробрасывается как есть
	}

	return s.jwt.Generate(id)
}

// Login проверяет учётные данные и возвращает токен доступа.
func (s *AuthService) Login(ctx context.Context, login, password string) (string, error) {
	u, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if err := auth.ComparePassword(u.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}

	return s.jwt.Generate(u.ID)
}
