// Package auth содержит примитивы аутентификации: хэширование паролей
// и работу с JWT-токенами.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword возвращает bcrypt-хэш пароля с солью, встроенной в сам хэш.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ComparePassword сверяет пароль в открытом виде с ранее сохранённым хэшем.
// Возвращает nil, если пароль верный.
func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
