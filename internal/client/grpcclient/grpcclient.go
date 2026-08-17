// Package grpcclient отвечает за установку gRPC-соединения с сервером
// GophKeeper и подготовку контекста с токеном авторизации для запросов.
package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/SatzhanDev/gophKeeper/internal/pkg/tlsutil"
)

// tlsServerName — имя сервера, под которым выпущен его самоподписанный
// TLS-сертификат (см. `make certs`, CN=localhost).
const tlsServerName = "localhost"

// Dial устанавливает TLS-соединение с gRPC-сервером по адресу addr,
// доверяя ровно тому сертификату, что лежит в caCertFile (сервер
// использует самоподписанный сертификат, поэтому системный пул
// доверенных CA его не примет).
func Dial(addr, caCertFile string) (*grpc.ClientConn, error) {
	creds, err := tlsutil.ClientCredentials(caCertFile, tlsServerName)
	if err != nil {
		return nil, err
	}
	return grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
}

// AuthContext возвращает context с приложенным JWT-токеном в metadata —
// именно оттуда его читает AuthInterceptor на сервере.
func AuthContext(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}
