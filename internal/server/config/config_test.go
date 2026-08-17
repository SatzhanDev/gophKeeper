package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearEnv гарантирует, что все переменные окружения, которые понимает
// config.Load, очищены перед тестом — независимо от того, что реально
// экспортировано в шелле разработчика. Без этого тесты становятся
// "хрупкими": проходят или падают в зависимости от чужого состояния
// терминала, а не от кода.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_TTL", "")
	t.Setenv("GRPC_PORT", "")
	t.Setenv("HTTP_PORT", "")
	t.Setenv("TLS_CERT_FILE", "")
	t.Setenv("TLS_KEY_FILE", "")
}

func TestLoad_FromFlags(t *testing.T) {
	clearEnv(t)

	cfg, err := Load([]string{"-d", "dsn-from-flag", "-j", "secret-from-flag", "-t", "1h", "-gp", "9999"})
	require.NoError(t, err)

	assert.Equal(t, "dsn-from-flag", cfg.DatabaseDSN)
	assert.Equal(t, "secret-from-flag", cfg.JWTSecret)
	assert.Equal(t, time.Hour, cfg.JWTTTL)
	assert.Equal(t, "9999", cfg.GRPCPort)
}

func TestLoad_FromEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_DSN", "dsn-from-env")
	t.Setenv("JWT_SECRET", "secret-from-env")

	cfg, err := Load(nil)
	require.NoError(t, err)

	assert.Equal(t, "dsn-from-env", cfg.DatabaseDSN)
	assert.Equal(t, "secret-from-env", cfg.JWTSecret)
	assert.Equal(t, defaultJWTTTL, cfg.JWTTTL)
	assert.Equal(t, defaultGRPCPort, cfg.GRPCPort)
}

func TestLoad_FlagWinsOverEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_DSN", "dsn-from-env")
	t.Setenv("JWT_SECRET", "secret-from-env")

	cfg, err := Load([]string{"-d", "dsn-from-flag"})
	require.NoError(t, err)

	assert.Equal(t, "dsn-from-flag", cfg.DatabaseDSN, "флаг должен иметь приоритет над env")
	assert.Equal(t, "secret-from-env", cfg.JWTSecret, "для JWTSecret флаг не передан — должен подхватиться env")
}

func TestLoad_MissingDatabaseDSN(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "secret")

	_, err := Load(nil)
	assert.Error(t, err)
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_DSN", "dsn")

	_, err := Load(nil)
	assert.Error(t, err)
}

func TestLoad_InvalidJWTTTL(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_DSN", "dsn")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_TTL", "not-a-duration")

	_, err := Load(nil)
	assert.Error(t, err)
}

func TestLoad_InvalidFlag(t *testing.T) {
	clearEnv(t)

	_, err := Load([]string{"-unknown-flag"})
	assert.Error(t, err)
}
