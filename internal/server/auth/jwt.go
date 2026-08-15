package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken возвращается, если токен некорректен, просрочен или
// подписан не тем секретом.
var ErrInvalidToken = errors.New("invalid token")

// claims — набор данных, зашитых в JWT.
type claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTManager отвечает за выпуск и проверку JWT-токенов.
type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTManager создаёт JWTManager с заданным секретом подписи и временем жизни токена.
func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), ttl: ttl}
}

// Generate выпускает подписанный токен для указанного пользователя.
func (m *JWTManager) Generate(userID int64) (string, error) {
	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(m.secret)
}

// Parse проверяет подпись и срок действия токена, возвращает ID пользователя.
func (m *JWTManager) Parse(tokenString string) (int64, error) {
	var c claims
	token, err := jwt.ParseWithClaims(tokenString, &c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return 0, ErrInvalidToken
	}
	return c.UserID, nil
}
