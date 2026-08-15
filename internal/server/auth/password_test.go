package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_ComparePassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("Sup3rSecret!")
	require.NoError(t, err)
	assert.NotEqual(t, "Sup3rSecret!", hash, "хэш не должен совпадать с исходным паролем")

	err = ComparePassword(hash, "Sup3rSecret!")
	assert.NoError(t, err)
}

func TestComparePassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("Sup3rSecret!")
	require.NoError(t, err)

	err = ComparePassword(hash, "WrongPassword")
	assert.Error(t, err)
}

func TestHashPassword_DifferentSaltEachTime(t *testing.T) {
	hash1, err := HashPassword("samepassword")
	require.NoError(t, err)
	hash2, err := HashPassword("samepassword")
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "bcrypt должен генерировать разную соль на каждый вызов")

	// Но оба хэша всё равно должны проходить проверку тем же паролем.
	assert.NoError(t, ComparePassword(hash1, "samepassword"))
	assert.NoError(t, ComparePassword(hash2, "samepassword"))
}
