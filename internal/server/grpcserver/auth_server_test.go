package grpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	"github.com/SatzhanDev/gophKeeper/internal/pkg/model"
	"github.com/SatzhanDev/gophKeeper/internal/server/service"
	"github.com/SatzhanDev/gophKeeper/internal/server/storage"
)

// fakeAuthService — реализация authServiceIface для тестов хендлера,
// без реального AuthService и базы данных.
type fakeAuthService struct {
	registerToken string
	registerErr   error

	loginToken   string
	loginSalt    []byte
	loginParams  model.KDFParams
	loginWrapped []byte
	loginErr     error
}

func (f *fakeAuthService) Register(context.Context, string, string, []byte, model.KDFParams, []byte) (string, error) {
	return f.registerToken, f.registerErr
}

func (f *fakeAuthService) Login(context.Context, string, string) (string, []byte, model.KDFParams, []byte, error) {
	return f.loginToken, f.loginSalt, f.loginParams, f.loginWrapped, f.loginErr
}

func TestAuthServer_Register_Success(t *testing.T) {
	srv := NewAuthServer(&fakeAuthService{registerToken: "tok"})

	resp, err := srv.Register(context.Background(), &authv1.RegisterRequest{Login: "ivan", Password: "pass"})
	require.NoError(t, err)
	assert.Equal(t, "tok", resp.GetToken())
}

func TestAuthServer_Register_LoginTaken(t *testing.T) {
	srv := NewAuthServer(&fakeAuthService{registerErr: storage.ErrLoginTaken})

	_, err := srv.Register(context.Background(), &authv1.RegisterRequest{Login: "ivan", Password: "pass"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestAuthServer_Register_InternalError(t *testing.T) {
	srv := NewAuthServer(&fakeAuthService{registerErr: errors.New("db is down")})

	_, err := srv.Register(context.Background(), &authv1.RegisterRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestAuthServer_Login_Success(t *testing.T) {
	srv := NewAuthServer(&fakeAuthService{
		loginToken:   "tok",
		loginSalt:    []byte("salt"),
		loginParams:  model.KDFParams{Time: 1, MemoryKB: 65536, Threads: 4},
		loginWrapped: []byte("wrapped"),
	})

	resp, err := srv.Login(context.Background(), &authv1.LoginRequest{Login: "ivan", Password: "pass"})
	require.NoError(t, err)
	assert.Equal(t, "tok", resp.GetToken())
	assert.Equal(t, []byte("salt"), resp.GetKdfSalt())
	assert.Equal(t, uint32(4), resp.GetKdfThreads())
	assert.Equal(t, []byte("wrapped"), resp.GetWrappedDek())
}

func TestAuthServer_Login_InvalidCredentials(t *testing.T) {
	srv := NewAuthServer(&fakeAuthService{loginErr: service.ErrInvalidCredentials})

	_, err := srv.Login(context.Background(), &authv1.LoginRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthServer_Login_InternalError(t *testing.T) {
	srv := NewAuthServer(&fakeAuthService{loginErr: errors.New("db is down")})

	_, err := srv.Login(context.Background(), &authv1.LoginRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
