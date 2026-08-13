// Package storage описывает интерфейсы доступа к хранилищу GophKeeper
// (пользователи и приватные данные) и связанные с ними ошибки.
// Конкретные реализации поверх PostgreSQL находятся в подпакете postgres —
// сам пакет storage не зависит ни от какой конкретной СУБД.
package storage

import (
	"context"
	"errors"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
)

// ErrUserNotFound возвращается, если пользователь с указанным логином
// не найден в хранилище.
var ErrUserNotFound = errors.New("user not found")

// ErrLoginTaken возвращается при попытке зарегистрировать пользователя
// с логином, который уже занят.
var ErrLoginTaken = errors.New("login already taken")

// UserRepository описывает доступ к хранилищу пользователей.
type UserRepository interface {
	// Create сохраняет нового пользователя и возвращает его ID.
	// Возвращает ErrLoginTaken, если логин уже занят.
	Create(ctx context.Context, u *model.User) (int64, error)

	// GetByLogin возвращает пользователя по логину.
	// Возвращает ErrUserNotFound, если такого пользователя нет.
	GetByLogin(ctx context.Context, login string) (*model.User, error)
}
