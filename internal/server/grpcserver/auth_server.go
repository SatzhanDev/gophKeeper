// Package grpcserver содержит реализацию gRPC-сервисов GophKeeper поверх
// бизнес-логики из internal/server/service.
package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/service"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// authServiceIface — минимальный набор методов AuthService, нужный этому
// gRPC-хендлеру. Определён на стороне потребителя (а не в пакете service),
// чтобы хендлер зависел только от того, что реально использует — это
// позволяет подменять реализацию фейком в юнит-тестах хендлера, не поднимая
// настоящий AuthService и базу данных. *service.AuthService реализует этот
// интерфейс неявно, никаких изменений в service не требуется.
type authServiceIface interface {
	Register(ctx context.Context, login, password string, kdfSalt []byte, params model.KDFParams, wrappedDEK []byte) (string, error)
	Login(ctx context.Context, login, password string) (token string, salt []byte, params model.KDFParams, wrappedDEK []byte, err error)
}

// AuthServer реализует authv1.AuthServiceServer.
type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	authService authServiceIface
	logger      *slog.Logger
}

// NewAuthServer создаёт AuthServer поверх готового AuthService. logger
// передаётся явным аргументом (а не берётся из глобального slog.Default())
// — это позволяет настраивать логирование конкретно для этого компонента
// (например, добавить постоянное поле "component") и подменять логгер
// в тестах, вместо того чтобы зависеть от глобального изменяемого состояния.
func NewAuthServer(authService authServiceIface, logger *slog.Logger) *AuthServer {
	return &AuthServer{authService: authService, logger: logger}
}

// Register реализует authv1.AuthServiceServer.
func (s *AuthServer) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	token, err := s.authService.Register(ctx, req.GetLogin(), req.GetPassword(), req.GetKdfSalt(), model.KDFParams{
		Time:     req.GetKdfTime(),
		MemoryKB: req.GetKdfMemoryKb(),
		Threads:  uint8(req.GetKdfThreads()),
	}, req.GetWrappedDek())
	if err != nil {
		if errors.Is(err, storage.ErrLoginTaken) {
			return nil, status.Error(codes.AlreadyExists, "login already taken")
		}
		s.logger.Error("register failed", "method", "Register", "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	resp := &authv1.RegisterResponse{}
	resp.SetToken(token)
	return resp, nil
}

// Login реализует authv1.AuthServiceServer.
func (s *AuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	token, salt, params, wrapDek, err := s.authService.Login(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid login or password")
		}
		s.logger.Error("login failed", "method", "Login", "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	resp := &authv1.LoginResponse{}
	resp.SetToken(token)
	resp.SetKdfSalt(salt)
	resp.SetKdfTime(params.Time)
	resp.SetKdfMemoryKb(params.MemoryKB)
	resp.SetKdfThreads(uint32(params.Threads))
	resp.SetWrappedDek(wrapDek)
	return resp, nil
}
