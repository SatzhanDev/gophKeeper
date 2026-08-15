package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	oldVersion, oldDate := Version, BuildDate
	t.Cleanup(func() { Version, BuildDate = oldVersion, oldDate })

	Version = "v1.2.3"
	BuildDate = "2026-01-01T00:00:00Z"

	got := String("client")
	assert.Equal(t, "GophKeeper client v1.2.3 (built 2026-01-01T00:00:00Z)", got)
}

func TestString_DifferentComponent(t *testing.T) {
	got := String("server")
	assert.Contains(t, got, "GophKeeper server")
}
