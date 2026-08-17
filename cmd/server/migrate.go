package main

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // регистрирует драйвер БД для схемы "postgres://"
	_ "github.com/golang-migrate/migrate/v4/source/file"       // регистрирует источник миграций для схемы "file://"
)

// migrationsPath — путь к каталогу с SQL-миграциями, относительно рабочей
// директории процесса (той же, что и для swagger-файлов в main.go).
const migrationsPath = "file://migrations"

// applyMigrations программно накатывает все непримененные миграции на базу
// dsn при старте сервера — вместо того чтобы полагаться на то, что кто-то
// не забудет вручную выполнить `make migrate-up` перед запуском. golang-migrate
// сам хранит в БД (таблица schema_migrations), какие миграции уже применены,
// поэтому повторный вызов при следующем перезапуске сервера безопасен и
// ничего не делает, если новых миграций нет.
//
// Используется отдельное соединение (через database/postgres драйвер), а не
// уже открытый pgxpool.Pool сервера — golang-migrate управляет своим
// подключением самостоятельно и не умеет работать поверх чужого пула.
func applyMigrations(dsn string) error {
	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
