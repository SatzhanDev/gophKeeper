package grpcclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestDial(t *testing.T) {
	// grpc.NewClient не устанавливает соединение немедленно (ленивое
	// подключение), поэтому Dial успешно отработает даже без реального
	// сервера на этом адресе — реальная попытка соединения произойдёт
	// только при первом RPC-вызове.
	conn, err := Dial("localhost:1")
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()
}

func TestAuthContext(t *testing.T) {
	ctx := AuthContext(context.Background(), "my-token")

	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)
	assert.Equal(t, []string{"Bearer my-token"}, md.Get("authorization"))
}
