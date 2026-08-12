package grpcserver

import (
	"context"
	"errors"
	"log"

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
}

// NewSecretServer создаёт SecretServer поверх готового SecretService.
func NewSecretServer(secretService secretServiceIface) *SecretServer {
	return &SecretServer{secretService: secretService}
}

func toProtoSecret(s *model.Secret) *secretv1.Secret {
	return &secretv1.Secret{
		Id:        s.ID,
		Type:      secretv1.SecretType(s.Type),
		Data:      s.Data,
		Metadata:  s.Metadata,
		Version:   int32(s.Version),
		CreatedAt: s.CreatedAt.Unix(),
		UpdatedAt: s.UpdatedAt.Unix(),
	}
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
		log.Printf("CreateSecret: %v", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &secretv1.CreateSecretResponse{Id: secret.ID}, nil
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
		log.Printf("GetSecret: %v", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &secretv1.GetSecretResponse{Secret: toProtoSecret(secret)}, nil
}

// ListSecrets реализует secretv1.SecretServiceServer.
func (s *SecretServer) ListSecrets(ctx context.Context, _ *secretv1.ListSecretsRequest) (*secretv1.ListSecretsResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user")
	}

	secrets, err := s.secretService.List(ctx, userID)
	if err != nil {
		log.Printf("ListSecrets: %v", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	resp := &secretv1.ListSecretsResponse{}
	for _, sec := range secrets {
		resp.Secrets = append(resp.Secrets, toProtoSecret(sec))
	}
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
		log.Printf("UpdateSecret: %v", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &secretv1.UpdateSecretResponse{Version: int32(newVersion)}, nil
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
		log.Printf("DeleteSecret: %v", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &secretv1.DeleteSecretResponse{}, nil
}
