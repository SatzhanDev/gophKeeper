// Command gophkeeper-server — gRPC-сервер GophKeeper: регистрация и
// аутентификация пользователей, хранение и выдача приватных данных.
// Конфигурируется через флаги командной строки или переменные окружения
// (см. internal/server/config).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	secretv1 "github.com/SatzhanDev/gophKeeper/api/proto/secret/v1"
	"github.com/SatzhanDev/gophKeeper/internal/server/auth"
	"github.com/SatzhanDev/gophKeeper/internal/server/config"
	"github.com/SatzhanDev/gophKeeper/internal/server/gateway"
	"github.com/SatzhanDev/gophKeeper/internal/server/grpcserver"
	"github.com/SatzhanDev/gophKeeper/internal/server/service"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage/postgres"
)

// shutdownTimeout — сколько ждать корректного завершения in-flight
// HTTP-запросов при остановке сервера, прежде чем закрыть соединение
// принудительно.
const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// signal.NotifyContext возвращает context, который сам отменяется при
	// получении SIGINT/SIGTERM — это и есть точка входа в graceful shutdown:
	// вся дальнейшая остановка триггерится через отмену этого контекста,
	// а не через немедленное завершение процесса.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("failed to create db pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping db", "err", err)
		os.Exit(1)
	}

	users := postgres.NewUserRepo(pool)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)
	authService := service.NewAuthService(users, jwtManager)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcserver.AuthInterceptor(jwtManager)),
	)
	authv1.RegisterAuthServiceServer(grpcServer, grpcserver.NewAuthServer(authService))

	secrets := postgres.NewSecretRepo(pool)
	secretService := service.NewSecretService(secrets)
	secretv1.RegisterSecretServiceServer(grpcServer, grpcserver.NewSecretServer(secretService))

	// TODO(TLS): сейчас и gRPC-сервер (выше, grpc.NewServer без
	// credentials), и это внутреннее подключение шлюза к нему — оба без
	// TLS. Когда сервер переведут на TLS, credentials здесь нужно поменять
	// синхронно на TLS-клиентские, иначе шлюз продолжит стучаться к
	// серверу в открытом виде, даже если снаружи он уже будет HTTPS.
	grpcDialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	gw, err := gateway.New(ctx, "localhost:"+cfg.GRPCPort, grpcDialOpts...)
	if err != nil {
		logger.Error("failed to create gateway", "err", err)
		os.Exit(1)
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/", gw)
	httpMux.HandleFunc("/swagger/auth.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/proto/auth/v1/auth.swagger.json")
	})
	httpMux.HandleFunc("/swagger/secret.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/proto/secret/v1/secret.swagger.json")
	})

	httpServer := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: httpMux,
	}

	// errgroup.WithContext даёт группу горутин, объединённых общим
	// контекстом gCtx: он отменяется, как только либо отменяется исходный
	// ctx (пришёл сигнал остановки), либо любая из горутин группы вернула
	// ошибку (тогда останавливаем всё остальное тоже, а не подвисаем).
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		logger.Info("gRPC server listening", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			return err
		}
		return nil
	})

	g.Go(func() error {
		logger.Info("REST/Swagger gateway listening", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// Третья горутина — сама логика остановки. Она ничего не делает, пока
	// gCtx жив, а как только он отменяется — останавливает оба сервера
	// вежливо: gRPC перестаёт принимать новые вызовы и ждёт завершения
	// уже начатых (GracefulStop), HTTP делает то же самое через Shutdown
	// с ограничением по времени (shutdownTimeout), чтобы не ждать вечно
	// зависший запрос.
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("shutdown signal received, stopping servers")

		grpcServer.GracefulStop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error("server stopped with error", "err", err)
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}
