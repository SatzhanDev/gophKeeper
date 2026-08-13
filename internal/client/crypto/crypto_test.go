package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, keySize)
	plaintext := []byte("login=ivan;password=secret123")

	ciphertext, err := Encrypt(key, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext, "шифротекст не должен совпадать с открытым текстом")

	got, err := Decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestEncrypt_DifferentNonceEachTime(t *testing.T) {
	key := make([]byte, keySize)
	plaintext := []byte("same plaintext")

	ct1, err := Encrypt(key, plaintext)
	require.NoError(t, err)
	ct2, err := Encrypt(key, plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "одинаковый plaintext должен давать разный шифротекст из-за случайного nonce")
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, keySize)
	key2 := make([]byte, keySize)
	key2[0] = 0xFF

	ciphertext, err := Encrypt(key1, []byte("secret"))
	require.NoError(t, err)

	_, err = Decrypt(key2, ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_TamperedData(t *testing.T) {
	key := make([]byte, keySize)
	ciphertext, err := Encrypt(key, []byte("secret"))
	require.NoError(t, err)

	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = Decrypt(key, ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_TooShort(t *testing.T) {
	_, err := Decrypt(make([]byte, keySize), []byte{0x01, 0x02})
	assert.ErrorIs(t, err, ErrCiphertextTooShort)
}

func TestDecrypt_UnsupportedVersion(t *testing.T) {
	key := make([]byte, keySize)
	ciphertext, err := Encrypt(key, []byte("secret"))
	require.NoError(t, err)

	ciphertext[0] = 0x02

	_, err = Decrypt(key, ciphertext)
	assert.ErrorIs(t, err, ErrUnsupportedVersion)
}

func TestDeriveKey_Deterministic(t *testing.T) {
	salt := []byte("0123456789abcdef")
	params := DefaultKDFParams()

	key1 := DeriveKey("password123", salt, params)
	key2 := DeriveKey("password123", salt, params)
	assert.Equal(t, key1, key2)
}

func TestDeriveKey_DifferentSaltDifferentKey(t *testing.T) {
	params := DefaultKDFParams()

	key1 := DeriveKey("password123", []byte("saltAAAAAAAAAAAA"), params)
	key2 := DeriveKey("password123", []byte("saltBBBBBBBBBBBB"), params)
	assert.NotEqual(t, key1, key2)
}

func TestDeriveKey_DifferentPasswordDifferentKey(t *testing.T) {
	salt := []byte("0123456789abcdef")
	params := DefaultKDFParams()

	key1 := DeriveKey("passwordAAA", salt, params)
	key2 := DeriveKey("passwordBBB", salt, params)
	assert.NotEqual(t, key1, key2)
}

func TestDeriveKey_Length(t *testing.T) {
	key := DeriveKey("password123", []byte("salt"), DefaultKDFParams())
	assert.Len(t, key, keySize)
}

func TestWrapUnwrapDEK_RoundTrip(t *testing.T) {
	kek := make([]byte, keySize)
	dek, err := GenerateDEK()
	require.NoError(t, err)

	wrapped, err := WrapDEK(dek, kek)
	require.NoError(t, err)

	unwrapped, err := UnwrapDEK(wrapped, kek)
	require.NoError(t, err)
	assert.Equal(t, dek, unwrapped)
}

func TestUnwrapDEK_WrongKEK(t *testing.T) {
	kek1 := make([]byte, keySize)
	kek2 := make([]byte, keySize)
	kek2[0] = 0xFF

	dek, err := GenerateDEK()
	require.NoError(t, err)

	wrapped, err := WrapDEK(dek, kek1)
	require.NoError(t, err)

	_, err = UnwrapDEK(wrapped, kek2)
	assert.Error(t, err)
}

func TestGenerateDEK_Uniqueness(t *testing.T) {
	dek1, err := GenerateDEK()
	require.NoError(t, err)
	dek2, err := GenerateDEK()
	require.NoError(t, err)

	assert.NotEqual(t, dek1, dek2, "два вызова GenerateDEK не должны давать одинаковый результат")
}

func TestGenerateDEK_Length(t *testing.T) {
	dek, err := GenerateDEK()
	require.NoError(t, err)
	assert.Len(t, dek, keySize)
}

func TestDefaultKDFParams(t *testing.T) {
	params := DefaultKDFParams()
	assert.Equal(t, uint32(1), params.Time)
	assert.Equal(t, uint32(64*1024), params.MemoryKB)
	assert.Equal(t, uint8(4), params.Threads)
}
