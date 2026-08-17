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

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	secretv1 "github.com/SatzhanDev/gophKeeper/api/proto/secret/v1"
	"github.com/SatzhanDev/gophKeeper/internal/pkg/tlsutil"
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

// tlsServerName — имя, под которым выпущен самоподписанный сертификат
// сервера (см. `make certs`, CN=localhost) — используется внутренним
// подключением REST-шлюза к gRPC-серверу для проверки идентичности.
const tlsServerName = "localhost"

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

	// Миграции накатываются программно при каждом старте — не нужно
	// помнить о ручном `make migrate-up` перед запуском. golang-migrate
	// сам знает, какие миграции уже применены, и просто ничего не делает,
	// если новых нет.
	if err := applyMigrations(cfg.DatabaseDSN); err != nil {
		logger.Error("failed to apply migrations", "err", err)
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

	// TLS обязателен для gRPC-сервера — без него JWT-токены, мастер-пароль
	// при регистрации/логине и зашифрованные данные передавались бы по
	// сети в открытом виде поверх TCP. Сертификат самоподписанный
	// (`make certs`), поэтому и внутреннее подключение шлюза (ниже), и
	// клиент должны явно доверять именно этому файлу сертификата — обычный
	// системный пул CA его не примет.
	serverCreds, err := tlsutil.ServerCredentials(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		logger.Error("failed to load TLS server credentials", "err", err)
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(serverCreds),
		grpc.UnaryInterceptor(grpcserver.AuthInterceptor(jwtManager)),
	)
	authv1.RegisterAuthServiceServer(grpcServer, grpcserver.NewAuthServer(authService, logger.With("component", "grpcserver.AuthServer")))

	secrets := postgres.NewSecretRepo(pool)
	secretService := service.NewSecretService(secrets)
	secretv1.RegisterSecretServiceServer(grpcServer, grpcserver.NewSecretServer(secretService, logger.With("component", "grpcserver.SecretServer")))

	// Внутреннее подключение шлюза к gRPC-серверу использует тот же самый
	// сертификат, что и сам сервер (доверяем ему как единственному
	// известному "CA") — оба соединения (внешнее клиент→сервер и
	// внутреннее шлюз→сервер) защищены одинаково, они не могут разойтись,
	// потому что оба берут credentials из одной и той же пары файлов.
	gatewayCreds, err := tlsutil.ClientCredentials(cfg.TLSCertFile, tlsServerName)
	if err != nil {
		logger.Error("failed to load TLS gateway credentials", "err", err)
		os.Exit(1)
	}

	gw, err := gateway.New(ctx, "localhost:"+cfg.GRPCPort, grpc.WithTransportCredentials(gatewayCreds))
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
