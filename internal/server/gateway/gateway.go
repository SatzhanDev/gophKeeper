// Package gateway реализует REST/JSON-шлюз поверх gRPC-сервисов GophKeeper:
// транслирует HTTP-запросы в gRPC-вызовы к уже работающему серверу.
package gateway

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	secretv1 "github.com/SatzhanDev/gophKeeper/api/proto/secret/v1"
)

// New создаёт HTTP-хендлер, транслирующий REST/JSON-запросы в gRPC-вызовы
// к серверу по адресу grpcAddr.
//
// dialOpts — опции для внутреннего gRPC-подключения шлюза к серверу
// (в частности, transport credentials). Они намеренно не выбираются внутри
// этого пакета, а передаются вызывающим кодом (main.go) — там же, где
// настраивается сам grpcServer. Так оба соединения (внешнее и внутреннее)
// гарантированно используют одни и те же credentials и не могут разойтись,
// если один из них позже переведут на TLS, а другой — забудут.
func New(ctx context.Context, grpcAddr string, dialOpts ...grpc.DialOption) (http.Handler, error) {
	mux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(headerMatcher))

	if err := authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts); err != nil {
		return nil, err
	}
	if err := secretv1.RegisterSecretServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts); err != nil {
		return nil, err
	}
	return mux, nil
}

// headerMatcher разрешает проброс заголовка Authorization из HTTP-запроса
// в gRPC metadata — без этого AuthInterceptor не увидит токен и REST-версия
// защищённых методов всегда будет отвечать Unauthenticated.
func headerMatcher(key string) (string, bool) {
	if key == "Authorization" {
		return "authorization", true
	}
	return runtime.DefaultHeaderMatcher(key)
}
