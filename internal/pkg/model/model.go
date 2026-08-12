// Package model содержит доменные сущности GophKeeper: пользователей и
// приватные данные (Secret), не зависящие от конкретного способа хранения
// или транспорта.
package model

import "time"

// SecretType задаёт тип приватных данных, хранимых в Secret. Нулевое
// значение (SecretType(0)) намеренно не используется ни одним из типов —
// оно означает "тип не указан" и должно трактоваться как ошибка.
type SecretType int16

// Поддерживаемые типы приватных данных согласно ТЗ: пары логин/пароль,
// произвольный текст, произвольные бинарные данные и данные банковских карт.
const (
	SecretTypeLoginPassword SecretType = iota + 1
	SecretTypeText
	SecretTypeBinary
	SecretTypeCard
)

// KDFParams — параметры Argon2id, с которыми был выведен ключ шифрования
// конкретного пользователя. Хранятся у каждого пользователя отдельно,
// чтобы менять параметры для новых пользователей, не ломая существующих.
type KDFParams struct {
	Time     uint32
	MemoryKB uint32
	Threads  uint8
}

// User — учётная запись владельца приватных данных.
type User struct {
	ID           int64
	Login        string
	PasswordHash string
	KDFSalt      []byte
	KDFParams    KDFParams
	WrappedDEK   []byte
	CreatedAt    time.Time
}

// Secret — единица приватных данных произвольного типа с метаинформацией.
// Data хранит зашифрованный на клиенте payload — сервер не имеет доступа
// к его содержимому в открытом виде.
type Secret struct {
	ID        int64
	UserID    int64
	Type      SecretType
	Data      []byte
	Metadata  string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
