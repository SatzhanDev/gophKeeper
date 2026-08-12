// Package crypto реализует клиентское end-to-end шифрование: вывод ключа
// из мастер-пароля (Argon2id), генерацию и оборачивание Data Encryption Key,
// и симметричное шифрование payload'а приватных данных (AES-256-GCM).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/argon2"
)

const (
	keySize    = 32 // AES-256
	nonceSize  = 12
	formatV1   = 0x01
	headerSize = 1 + nonceSize // версия + nonce
)

var (
	// ErrCiphertextTooShort возвращается, если данные короче, чем минимально
	// необходимо для заголовка (версия + nonce).
	ErrCiphertextTooShort = errors.New("ciphertext too short")
	// ErrUnsupportedVersion возвращается при попытке расшифровать данные
	// неизвестной версии формата.
	ErrUnsupportedVersion = errors.New("unsupported ciphertext version")
)

// KDFParams — параметры вывода ключа. Совпадает с model.KDFParams,
// но пакет crypto не должен зависеть от model, поэтому определён отдельно.
type KDFParams struct {
	Time     uint32
	MemoryKB uint32
	Threads  uint8
}

// DefaultKDFParams возвращает рекомендованные параметры Argon2id
// для новых пользователей.
func DefaultKDFParams() KDFParams {
	return KDFParams{Time: 1, MemoryKB: 64 * 1024, Threads: 4}
}

// DeriveKey выводит 256-битный ключ (KEK) из мастер-пароля и соли
// по заданным параметрам Argon2id.
func DeriveKey(password string, salt []byte, p KDFParams) []byte {
	return argon2.IDKey([]byte(password), salt, p.Time, p.MemoryKB, p.Threads, keySize)
}

// GenerateDEK создаёт новый случайный Data Encryption Key.
func GenerateDEK() ([]byte, error) {
	dek := make([]byte, keySize)
	if _, err := rand.Read(dek); err != nil {
		return nil, err
	}
	return dek, nil
}

// Encrypt шифрует plaintext ключом key и возвращает блоб формата
// [версия(1 байт)][nonce(12 байт)][шифротекст].
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	out := make([]byte, 0, headerSize+len(plaintext)+gcm.Overhead())
	out = append(out, formatV1)
	out = append(out, nonce...)
	return gcm.Seal(out, nonce, plaintext, nil), nil
}

// Decrypt расшифровывает данные, полученные из Encrypt, тем же ключом key.
func Decrypt(key, data []byte) ([]byte, error) {
	if len(data) < headerSize {
		return nil, ErrCiphertextTooShort
	}
	if data[0] != formatV1 {
		return nil, ErrUnsupportedVersion
	}
	nonce := data[1:headerSize]
	ciphertext := data[headerSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// WrapDEK шифрует DEK ключом kek (для хранения на сервере в непрозрачном виде).
func WrapDEK(dek, kek []byte) ([]byte, error) {
	return Encrypt(kek, dek)
}

// UnwrapDEK расшифровывает DEK, ранее полученный от WrapDEK.
func UnwrapDEK(wrapped, kek []byte) ([]byte, error) {
	return Decrypt(kek, wrapped)
}
