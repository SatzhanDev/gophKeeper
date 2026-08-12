package grpcserver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/SatzhanDev/gophKeeper/internal/server/auth"
)

func noopHandler(_ context.Context, _ interface{}) (interface{}, error) {
	return "ok", nil
}

func TestAuthInterceptor_PublicMethodBypassesAuth(t *testing.T) {
	interceptor := AuthInterceptor(auth.NewJWTManager("secret", time.Hour))

	resp, err := interceptor(context.Background(), nil,
		&grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/Login"}, noopHandler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	interceptor := AuthInterceptor(auth.NewJWTManager("secret", time.Hour))

	_, err := interceptor(context.Background(), nil,
		&grpc.UnaryServerInfo{FullMethod: "/secret.v1.SecretService/ListSecrets"}, noopHandler)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_MissingToken(t *testing.T) {
	interceptor := AuthInterceptor(auth.NewJWTManager("secret", time.Hour))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	_, err := interceptor(ctx, nil,
		&grpc.UnaryServerInfo{FullMethod: "/secret.v1.SecretService/ListSecrets"}, noopHandler)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	interceptor := AuthInterceptor(auth.NewJWTManager("secret", time.Hour))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer not-a-real-token"))

	_, err := interceptor(ctx, nil,
		&grpc.UnaryServerInfo{FullMethod: "/secret.v1.SecretService/ListSecrets"}, noopHandler)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_TokenFromDifferentSecret(t *testing.T) {
	otherManager := auth.NewJWTManager("other-secret", time.Hour)
	token, err := otherManager.Generate(1)
	require.NoError(t, err)

	interceptor := AuthInterceptor(auth.NewJWTManager("secret", time.Hour))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	_, err = interceptor(ctx, nil,
		&grpc.UnaryServerInfo{FullMethod: "/secret.v1.SecretService/ListSecrets"}, noopHandler)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_ValidToken_SetsUserIDInContext(t *testing.T) {
	jwtManager := auth.NewJWTManager("secret", time.Hour)
	token, err := jwtManager.Generate(42)
	require.NoError(t, err)

	interceptor := AuthInterceptor(jwtManager)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	var gotUserID int64
	var gotOK bool
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		gotUserID, gotOK = UserIDFromContext(ctx)
		return nil, nil
	}

	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/secret.v1.SecretService/ListSecrets"}, handler)
	require.NoError(t, err)
	require.True(t, gotOK)
	assert.Equal(t, int64(42), gotUserID)
}

func TestUserIDFromContext_NotSet(t *testing.T) {
	_, ok := UserIDFromContext(context.Background())
	assert.False(t, ok)
}
