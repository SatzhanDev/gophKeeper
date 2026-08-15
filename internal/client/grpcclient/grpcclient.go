// Package grpcclient отвечает за установку gRPC-соединения с сервером
// GophKeeper и подготовку контекста с токеном авторизации для запросов.
package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Dial устанавливает соединение с gRPC-сервером по адресу addr.
// TLS пока не используется (insecure) — это временное упрощение для
// локальной разработки, для реального использования нужно добавить TLS.
func Dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

// AuthContext возвращает context с приложенным JWT-токеном в metadata —
// именно оттуда его читает AuthInterceptor на сервере
func AuthContext(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}
