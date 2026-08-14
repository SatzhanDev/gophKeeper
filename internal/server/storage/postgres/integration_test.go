//go:build integration

// Эти тесты поднимают настоящий Postgres в Docker-контейнере через
// testcontainers-go и проверяют реальный SQL, а не мок. Запускаются
// отдельно от обычных юнит-тестов (нужен Docker):
//
//	make test-integration
//
// Обычный `go test ./...` эти тесты не видит и не запускает — файл
// компилируется только с build-тегом "integration".
package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage/postgres"
)

// setupTestDB поднимает временный Postgres-контейнер, накатывает на него
// все миграции проекта и возвращает готовый к использованию пул соединений.
// Контейнер и пул автоматически останавливаются/закрываются после теста
// через t.Cleanup.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	// Официальный образ postgres при первом запуске один раз перезапускает
	// сам себя после initdb — если подключиться в этот момент, соединение
	// обрывается ("connection reset by peer"). Ждём фразу о готовности в
	// логах ДВАЖДЫ: первый раз — до внутреннего рестарта, второй — после,
	// это и есть надёжный сигнал реальной готовности принимать соединения.
	container, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("gophkeeper_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	applyMigrations(t, pool)

	return pool
}

// applyMigrations выполняет все *.up.sql файлы из папки migrations в
// порядке их номеров — вручную через pool.Exec, без внешнего инструмента
// migrate, чтобы тестам не требовалась ничего, кроме Docker и Go.
func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	files, err := filepath.Glob("../../../../migrations/*.up.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files, "не найдено ни одного файла миграции")
	sort.Strings(files)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, f := range files {
		content, err := os.ReadFile(f)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, string(content))
		require.NoErrorf(t, err, "миграция %s", f)
	}
}

func testUser(login string) *model.User {
	return &model.User{
		Login:        login,
		PasswordHash: "bcrypt-hash",
		KDFSalt:      []byte("salt"),
		KDFParams:    model.KDFParams{Time: 1, MemoryKB: 65536, Threads: 4},
		WrappedDEK:   []byte("wrapped-dek"),
	}
}

func TestUserRepo_CreateAndGetByLogin(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewUserRepo(pool)
	ctx := context.Background()

	id, err := repo.Create(ctx, testUser("ivan"))
	require.NoError(t, err)
	require.NotZero(t, id)

	u, err := repo.GetByLogin(ctx, "ivan")
	require.NoError(t, err)
	require.Equal(t, "ivan", u.Login)
	require.Equal(t, uint32(1), u.KDFParams.Time)
	require.Equal(t, uint32(65536), u.KDFParams.MemoryKB)
	require.Equal(t, uint8(4), u.KDFParams.Threads)
	require.Equal(t, []byte("wrapped-dek"), u.WrappedDEK)
}

func TestUserRepo_GetByLogin_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewUserRepo(pool)

	_, err := repo.GetByLogin(context.Background(), "no-such-user")
	require.ErrorIs(t, err, storage.ErrUserNotFound)
}

func TestUserRepo_Create_DuplicateLogin(t *testing.T) {
	pool := setupTestDB(t)
	repo := postgres.NewUserRepo(pool)
	ctx := context.Background()

	_, err := repo.Create(ctx, testUser("dup"))
	require.NoError(t, err)

	_, err = repo.Create(ctx, testUser("dup"))
	require.ErrorIs(t, err, storage.ErrLoginTaken)
}

func TestSecretRepo_OwnershipIsolation(t *testing.T) {
	pool := setupTestDB(t)
	users := postgres.NewUserRepo(pool)
	secrets := postgres.NewSecretRepo(pool)
	ctx := context.Background()

	userA, err := users.Create(ctx, testUser("a"))
	require.NoError(t, err)
	userB, err := users.Create(ctx, testUser("b"))
	require.NoError(t, err)

	secret := &model.Secret{UserID: userA, Type: model.SecretTypeText, Data: []byte("data"), Metadata: "meta"}
	id, err := secrets.Create(ctx, secret)
	require.NoError(t, err)

	// userB не должен увидеть секрет userA — ключевая проверка изоляции
	// владельцев, ради которой userID зашит в каждый SQL-запрос.
	_, err = secrets.GetByID(ctx, userB, id)
	require.ErrorIs(t, err, storage.ErrSecretNotFound)

	// userA видит свой секрет.
	got, err := secrets.GetByID(ctx, userA, id)
	require.NoError(t, err)
	require.Equal(t, "meta", got.Metadata)
	require.Equal(t, []byte("data"), got.Data)
	require.Equal(t, 1, got.Version)
}

func TestSecretRepo_ListByUser_OnlyOwnAndNotDeleted(t *testing.T) {
	pool := setupTestDB(t)
	users := postgres.NewUserRepo(pool)
	secrets := postgres.NewSecretRepo(pool)
	ctx := context.Background()

	userA, err := users.Create(ctx, testUser("list-a"))
	require.NoError(t, err)
	userB, err := users.Create(ctx, testUser("list-b"))
	require.NoError(t, err)

	s1 := &model.Secret{UserID: userA, Type: model.SecretTypeText, Data: []byte("1"), Metadata: "one"}
	_, err = secrets.Create(ctx, s1)
	require.NoError(t, err)

	s2 := &model.Secret{UserID: userA, Type: model.SecretTypeText, Data: []byte("2"), Metadata: "two"}
	_, err = secrets.Create(ctx, s2)
	require.NoError(t, err)

	sOther := &model.Secret{UserID: userB, Type: model.SecretTypeText, Data: []byte("3"), Metadata: "other"}
	_, err = secrets.Create(ctx, sOther)
	require.NoError(t, err)

	require.NoError(t, secrets.Delete(ctx, userA, s2.ID))

	list, err := secrets.ListByUser(ctx, userA)
	require.NoError(t, err)
	require.Len(t, list, 1, "должен остаться только s1 — s2 удалён, sOther принадлежит другому пользователю")
	require.Equal(t, "one", list[0].Metadata)
}

func TestSecretRepo_Update_OptimisticConcurrency(t *testing.T) {
	pool := setupTestDB(t)
	users := postgres.NewUserRepo(pool)
	secrets := postgres.NewSecretRepo(pool)
	ctx := context.Background()

	userID, err := users.Create(ctx, testUser("update-user"))
	require.NoError(t, err)

	secret := &model.Secret{UserID: userID, Type: model.SecretTypeText, Data: []byte("v1"), Metadata: "meta"}
	_, err = secrets.Create(ctx, secret)
	require.NoError(t, err)
	require.Equal(t, 1, secret.Version)

	// Первое обновление с правильной версией — должно пройти.
	update1 := &model.Secret{ID: secret.ID, Data: []byte("v2"), Metadata: "meta", Version: secret.Version}
	require.NoError(t, secrets.Update(ctx, userID, update1))
	require.Equal(t, 2, update1.Version)

	// Повторное обновление с уже устаревшей версией (той же, что была
	// у secret до первого апдейта) должно провалиться конфликтом.
	update2 := &model.Secret{ID: secret.ID, Data: []byte("v3"), Metadata: "meta", Version: secret.Version}
	err = secrets.Update(ctx, userID, update2)
	require.ErrorIs(t, err, storage.ErrVersionConflict)
}

func TestSecretRepo_Update_WrongOwner(t *testing.T) {
	pool := setupTestDB(t)
	users := postgres.NewUserRepo(pool)
	secrets := postgres.NewSecretRepo(pool)
	ctx := context.Background()

	userA, err := users.Create(ctx, testUser("owner-a"))
	require.NoError(t, err)
	userB, err := users.Create(ctx, testUser("owner-b"))
	require.NoError(t, err)

	secret := &model.Secret{UserID: userA, Type: model.SecretTypeText, Data: []byte("v1"), Metadata: "meta"}
	_, err = secrets.Create(ctx, secret)
	require.NoError(t, err)

	update := &model.Secret{ID: secret.ID, Data: []byte("hacked"), Metadata: "meta", Version: secret.Version}
	err = secrets.Update(ctx, userB, update)
	require.ErrorIs(t, err, storage.ErrVersionConflict, "чужой userID должен трактоваться так же, как конфликт версии — обновление не проходит")
}

func TestSecretRepo_Delete_IsSoft(t *testing.T) {
	pool := setupTestDB(t)
	users := postgres.NewUserRepo(pool)
	secrets := postgres.NewSecretRepo(pool)
	ctx := context.Background()

	userID, err := users.Create(ctx, testUser("delete-user"))
	require.NoError(t, err)

	secret := &model.Secret{UserID: userID, Type: model.SecretTypeText, Data: []byte("v1"), Metadata: "meta"}
	_, err = secrets.Create(ctx, secret)
	require.NoError(t, err)

	require.NoError(t, secrets.Delete(ctx, userID, secret.ID))

	_, err = secrets.GetByID(ctx, userID, secret.ID)
	require.ErrorIs(t, err, storage.ErrSecretNotFound, "после soft delete GetByID не должен находить запись")

	var count int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM secrets WHERE id = $1", secret.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "строка должна физически остаться в таблице — это soft delete, не DELETE")
}

func TestSecretRepo_Delete_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	users := postgres.NewUserRepo(pool)
	secrets := postgres.NewSecretRepo(pool)
	ctx := context.Background()

	userID, err := users.Create(ctx, testUser("delete-notfound-user"))
	require.NoError(t, err)

	err = secrets.Delete(ctx, userID, 999999)
	require.ErrorIs(t, err, storage.ErrSecretNotFound)
}
