// Package tlsutil содержит вспомогательные функции для настройки TLS
// поверх самоподписанного сертификата: сервер использует его для приёма
// соединений, клиент (и внутреннее подключение REST-шлюза к gRPC-серверу)
// — чтобы явно доверять именно этому конкретному сертификату (обычный пул
// системных корневых CA его не примет, потому что он не выпущен реальным
// удостоверяющим центром).
package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

// ServerCredentials строит gRPC transport credentials сервера из пары
// файлов сертификат/приватный ключ (см. `make certs`).
func ServerCredentials(certFile, keyFile string) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS key pair: %w", err)
	}
	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}), nil
}

// ClientCredentials строит gRPC transport credentials, доверяющие ровно
// одному сертификату из файла certFile — тому самому самоподписанному
// сертификату, который сервер использует в ServerCredentials. serverName
// должен совпадать с CN/SAN в сертификате (у нас это "localhost").
func ClientCredentials(certFile, serverName string) (credentials.TransportCredentials, error) {
	pem, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert file: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("failed to parse CA cert file")
	}

	return credentials.NewTLS(&tls.Config{
		RootCAs:    pool,
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	}), nil
}
