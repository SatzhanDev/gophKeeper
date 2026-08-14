// Command gophkeeper-server — gRPC-сервер GophKeeper: регистрация и
// аутентификация пользователей, хранение и выдача приватных данных.
// Конфигурируется через флаги командной строки или переменные окружения
// (см. internal/server/config).
package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	secretv1 "github.com/SatzhanDev/gophKeeper/api/proto/secret/v1"
	"github.com/SatzhanDev/gophKeeper/internal/server/auth"
	"github.com/SatzhanDev/gophKeeper/internal/server/config"
	"github.com/SatzhanDev/gophKeeper/internal/server/gateway"
	"github.com/SatzhanDev/gophKeeper/internal/server/grpcserver"
	"github.com/SatzhanDev/gophKeeper/internal/server/service"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage/postgres"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("failed to create db pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	users := postgres.NewUserRepo(pool)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)
	authService := service.NewAuthService(users, jwtManager)
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcserver.AuthInterceptor(jwtManager)),
	)
	authv1.RegisterAuthServiceServer(grpcServer, grpcserver.NewAuthServer(authService))

	secrets := postgres.NewSecretRepo(pool)
	secretService := service.NewSecretService(secrets)
	secretv1.RegisterSecretServiceServer(grpcServer, grpcserver.NewSecretServer(secretService))

	go func() {
		log.Printf("gophkeeper server: gRPC listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve grpc: %v", err)
		}
	}()

	gw, err := gateway.New(ctx, "localhost:"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to create gateway: %v", err)
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/", gw)
	httpMux.HandleFunc("/swagger/auth.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/proto/auth/v1/auth.swagger.json")
	})
	httpMux.HandleFunc("/swagger/secret.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/proto/secret/v1/secret.swagger.json")
	})

	log.Printf("gophkeeper server: REST/Swagger gateway on :%s", cfg.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, httpMux))
}
