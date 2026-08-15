package grpcserver

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/SatzhanDev/gophKeeper/internal/server/auth"
)

// ctxKeyUserID — ключ контекста для ID аутентифицированного пользователя.
type ctxKeyUserID struct{}

// publicMethods — методы, не требующие аутентификации.
var publicMethods = map[string]bool{
	"/auth.v1.AuthService/Register": true,
	"/auth.v1.AuthService/Login":    true,
}

// AuthInterceptor возвращает unary-интерцептор, который проверяет
// JWT-токен из метаданных запроса для всех методов, кроме publicMethods.
func AuthInterceptor(jwtManager *auth.JWTManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		token := strings.TrimPrefix(values[0], "Bearer ")
		userID, err := jwtManager.Parse(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, ctxKeyUserID{}, userID)
		return handler(ctx, req)
	}
}

// UserIDFromContext достаёт ID аутентифицированного пользователя,
// положенный туда AuthInterceptor.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ctxKeyUserID{}).(int64)
	return id, ok
}
