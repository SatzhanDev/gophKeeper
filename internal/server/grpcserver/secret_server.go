package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	secretv1 "github.com/SatzhanDev/gophKeeper/api/proto/secret/v1"
	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/service"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// secretServiceIface — минимальный набор методов SecretService, нужный
// этому gRPC-хендлеру (см. пояснение в auth_server.go про authServiceIface).
type secretServiceIface interface {
	Create(ctx context.Context, userID int64, t model.SecretType, data []byte, metadata string) (*model.Secret, error)
	Get(ctx context.Context, userID, id int64) (*model.Secret, error)
	List(ctx context.Context, userID int64) ([]*model.Secret, error)
	Update(ctx context.Context, userID, id int64, data []byte, metadata string, version int) (int, error)
	Delete(ctx context.Context, userID, id int64) error
}

// SecretServer реализует secretv1.SecretServiceServer.
type SecretServer struct {
	secretv1.UnimplementedSecretServiceServer
	secretService secretServiceIface
	logger        *slog.Logger
}

// NewSecretServer создаёт SecretServer поверх готового SecretService.
// logger передаётся явным аргументом — см. пояснение в NewAuthServer.
func NewSecretServer(secretService secretServiceIface, logger *slog.Logger) *SecretServer {
	return &SecretServer{secretService: secretService, logger: logger}
}

func toProtoSecret(s *model.Secret) *secretv1.Secret {
	ps := &secretv1.Secret{}
	ps.SetId(s.ID)
	ps.SetType(secretv1.SecretType(s.Type))
	ps.SetData(s.Data)
	ps.SetMetadata(s.Metadata)
	ps.SetVersion(int32(s.Version))
	ps.SetCreatedAt(s.CreatedAt.Unix())
	ps.SetUpdatedAt(s.UpdatedAt.Unix())
	return ps
}

// CreateSecret реализует secretv1.SecretServiceServer.
func (s *SecretServer) CreateSecret(ctx context.Context, req *secretv1.CreateSecretRequest) (*secretv1.CreateSecretResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user")
	}

	secret, err := s.secretService.Create(ctx, userID, model.SecretType(req.GetType()), req.GetData(), req.GetMetadata())
	if err != nil {
		if errors.Is(err, service.ErrInvalidSecretType) {
			return nil, status.Error(codes.InvalidArgument, "invalid secret type")
		}
		s.logger.Error("create secret failed", "method", "CreateSecret", "user_id", userID, "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	resp := &secretv1.CreateSecretResponse{}
	resp.SetId(secret.ID)
	return resp, nil
}

// GetSecret реализует secretv1.SecretServiceServer.
func (s *SecretServer) GetSecret(ctx context.Context, req *secretv1.GetSecretRequest) (*secretv1.GetSecretResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user")
	}

	secret, err := s.secretService.Get(ctx, userID, req.GetId())
	if err != nil {
		if errors.Is(err, storage.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "secret not found")
		}
		s.logger.Error("get secret failed", "method", "GetSecret", "user_id", userID, "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	resp := &secretv1.GetSecretResponse{}
	resp.SetSecret(toProtoSecret(secret))
	return resp, nil
}

// ListSecrets реализует secretv1.SecretServiceServer.
func (s *SecretServer) ListSecrets(ctx context.Context, _ *secretv1.ListSecretsRequest) (*secretv1.ListSecretsResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user")
	}

	secrets, err := s.secretService.List(ctx, userID)
	if err != nil {
		s.logger.Error("list secrets failed", "method", "ListSecrets", "user_id", userID, "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Собираем обычный Go-слайс и кладём его в сообщение целиком через
	// SetSecrets — единообразно с остальным кодом, который везде использует
	// сеттеры, а не прямое обращение к полям (см. пояснение в auth.go).
	protoSecrets := make([]*secretv1.Secret, 0, len(secrets))
	for _, sec := range secrets {
		protoSecrets = append(protoSecrets, toProtoSecret(sec))
	}

	resp := &secretv1.ListSecretsResponse{}
	resp.SetSecrets(protoSecrets)
	return resp, nil
}

// UpdateSecret реализует secretv1.SecretServiceServer.
func (s *SecretServer) UpdateSecret(ctx context.Context, req *secretv1.UpdateSecretRequest) (*secretv1.UpdateSecretResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user")
	}

	newVersion, err := s.secretService.Update(ctx, userID, req.GetId(), req.GetData(), req.GetMetadata(), int(req.GetVersion()))
	if err != nil {
		if errors.Is(err, storage.ErrVersionConflict) {
			return nil, status.Error(codes.Aborted, "version conflict")
		}
		s.logger.Error("update secret failed", "method", "UpdateSecret", "user_id", userID, "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	resp := &secretv1.UpdateSecretResponse{}
	resp.SetVersion(int32(newVersion))
	return resp, nil
}

// DeleteSecret реализует secretv1.SecretServiceServer.
func (s *SecretServer) DeleteSecret(ctx context.Context, req *secretv1.DeleteSecretRequest) (*secretv1.DeleteSecretResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user")
	}

	if err := s.secretService.Delete(ctx, userID, req.GetId()); err != nil {
		if errors.Is(err, storage.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "secret not found")
		}
		s.logger.Error("delete secret failed", "method", "DeleteSecret", "user_id", userID, "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &secretv1.DeleteSecretResponse{}, nil
}
