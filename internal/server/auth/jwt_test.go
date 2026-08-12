package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_GenerateParse_RoundTrip(t *testing.T) {
	m := NewJWTManager("test-secret", time.Hour)

	token, err := m.Generate(42)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	userID, err := m.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, int64(42), userID)
}

func TestJWTManager_Parse_WrongSecret(t *testing.T) {
	m1 := NewJWTManager("secret-one", time.Hour)
	m2 := NewJWTManager("secret-two", time.Hour)

	token, err := m1.Generate(1)
	require.NoError(t, err)

	_, err = m2.Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTManager_Parse_Expired(t *testing.T) {
	// Отрицательный TTL — срок действия токена истекает в момент выпуска.
	m := NewJWTManager("test-secret", -time.Hour)

	token, err := m.Generate(1)
	require.NoError(t, err)

	_, err = m.Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTManager_Parse_MalformedString(t *testing.T) {
	m := NewJWTManager("test-secret", time.Hour)

	_, err := m.Parse("not-a-valid-jwt-token")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTManager_Parse_RejectsNoneAlgorithm(t *testing.T) {
	m := NewJWTManager("test-secret", time.Hour)

	// Собираем токен с alg=none вручную — Parse обязан его отклонить
	// благодаря проверке t.Method.(*jwt.SigningMethodHMAC).
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": 999,
	})
	unsigned, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = m.Parse(unsigned)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTManager_Parse_RejectsDifferentHMACSecretEvenWithSameAlg(t *testing.T) {
	m := NewJWTManager("real-secret", time.Hour)

	// Токен подписан HS256, но чужим секретом — валидная структура,
	// невалидная подпись.
	forged := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, err := forged.SignedString([]byte("attacker-secret"))
	require.NoError(t, err)

	_, err = m.Parse(signed)
	assert.ErrorIs(t, err, ErrInvalidToken)
}
