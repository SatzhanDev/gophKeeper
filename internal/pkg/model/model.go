package model

import "time"

type SecretType int16

const (
	SecretTypeLoginPassword SecretType = iota + 1
	SecretTypeText
	SecretTypeBinary
	SecretTypeCard
)

// User — учётная запись владельца приватных данных.
type User struct {
	ID           int64
	Login        string
	PasswordHash string
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
