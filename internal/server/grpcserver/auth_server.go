// Package grpcserver содержит реализацию gRPC-сервисов GophKeeper поверх
// бизнес-логики из internal/server/service.
package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	"github.com/SatzhanDev/gophKeeper/internal/server/service"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// AuthServer реализует authv1.AuthServiceServer.
type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	authService *service.AuthService
}

// NewAuthServer создаёт AuthServer поверх готового AuthService.
func NewAuthServer(authService *service.AuthService) *AuthServer {
	return &AuthServer{authService: authService}
}

// Register реализует authv1.AuthServiceServer.
func (s *AuthServer) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	token, err := s.authService.Register(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		if errors.Is(err, storage.ErrLoginTaken) {
			return nil, status.Error(codes.AlreadyExists, "login already taken")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &authv1.RegisterResponse{Token: token}, nil
}

// Login реализует authv1.AuthServiceServer.
func (s *AuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	token, err := s.authService.Login(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid login or password")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &authv1.LoginResponse{Token: token}, nil
}
