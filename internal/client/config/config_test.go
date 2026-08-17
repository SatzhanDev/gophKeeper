package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearEnv гарантирует, что GOPHKEEPER_SERVER очищена перед тестом,
// независимо от того, что реально экспортировано в шелле разработчика.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GOPHKEEPER_SERVER", "")
	t.Setenv("TLS_CA_CERT_FILE", "")
}

func TestLoad_Default(t *testing.T) {
	clearEnv(t)

	cfg, err := Load(nil)
	require.NoError(t, err)

	assert.Equal(t, defaultServerAddr, cfg.ServerAddr)
	assert.False(t, cfg.ShowVersion)
}

func TestLoad_FromFlag(t *testing.T) {
	clearEnv(t)

	cfg, err := Load([]string{"-server", "example.com:1234"})
	require.NoError(t, err)

	assert.Equal(t, "example.com:1234", cfg.ServerAddr)
}

func TestLoad_FromEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("GOPHKEEPER_SERVER", "env.example.com:1234")

	cfg, err := Load(nil)
	require.NoError(t, err)

	assert.Equal(t, "env.example.com:1234", cfg.ServerAddr)
}

func TestLoad_FlagWinsOverEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("GOPHKEEPER_SERVER", "env.example.com:1234")

	cfg, err := Load([]string{"-server", "flag.example.com:1234"})
	require.NoError(t, err)

	assert.Equal(t, "flag.example.com:1234", cfg.ServerAddr)
}

func TestLoad_VersionFlag(t *testing.T) {
	clearEnv(t)

	cfg, err := Load([]string{"-version"})
	require.NoError(t, err)

	assert.True(t, cfg.ShowVersion)
}

func TestLoad_InvalidFlag(t *testing.T) {
	clearEnv(t)

	_, err := Load([]string{"-unknown-flag"})
	assert.Error(t, err)
}
