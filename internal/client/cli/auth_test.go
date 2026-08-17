package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	"github.com/SatzhanDev/gophKeeper/internal/client/crypto"
)

// fakeAuthClient — реализация authv1.AuthServiceClient для тестов,
// без реального gRPC-соединения и сервера.
type fakeAuthClient struct {
	registerResp *authv1.RegisterResponse
	registerErr  error
	loginResp    *authv1.LoginResponse
	loginErr     error
}

func (f *fakeAuthClient) Register(context.Context, *authv1.RegisterRequest, ...grpc.CallOption) (*authv1.RegisterResponse, error) {
	return f.registerResp, f.registerErr
}

func (f *fakeAuthClient) Login(context.Context, *authv1.LoginRequest, ...grpc.CallOption) (*authv1.LoginResponse, error) {
	return f.loginResp, f.loginErr
}

func newRegisterResponse(token string) *authv1.RegisterResponse {
	resp := &authv1.RegisterResponse{}
	resp.SetToken(token)
	return resp
}

func newLoginResponse(token string, salt []byte, params crypto.KDFParams, wrapped []byte) *authv1.LoginResponse {
	resp := &authv1.LoginResponse{}
	resp.SetToken(token)
	resp.SetKdfSalt(salt)
	resp.SetKdfTime(params.Time)
	resp.SetKdfMemoryKb(params.MemoryKB)
	resp.SetKdfThreads(uint32(params.Threads))
	resp.SetWrappedDek(wrapped)
	return resp
}

// withFakePassword подменяет readPassword на время теста, чтобы не
// трогать реальный терминал.
func withFakePassword(t *testing.T, password string) {
	t.Helper()
	old := readPassword
	readPassword = func(string) (string, error) { return password, nil }
	t.Cleanup(func() { readPassword = old })
}

func TestCmdRegister_Success(t *testing.T) {
	withFakePassword(t, "master-password")

	st := &state{authClient: &fakeAuthClient{registerResp: newRegisterResponse("tok")}}

	err := cmdRegister(st, []string{"ivan"})
	require.NoError(t, err)
	assert.Equal(t, "tok", st.token)
	assert.True(t, st.loggedIn)
	assert.Len(t, st.dek, 32)
}

func TestCmdRegister_WrongArgs(t *testing.T) {
	st := &state{}
	err := cmdRegister(st, []string{})
	assert.Error(t, err)
	assert.False(t, st.loggedIn)
}

func TestCmdRegister_ServerError(t *testing.T) {
	withFakePassword(t, "master-password")

	st := &state{authClient: &fakeAuthClient{registerErr: errors.New("login taken")}}

	err := cmdRegister(st, []string{"ivan"})
	assert.Error(t, err)
	assert.False(t, st.loggedIn)
}

func TestCmdLogin_Success(t *testing.T) {
	withFakePassword(t, "master-password")

	// Собираем на лету wrapped DEK тем же способом, что и сервер бы
	// его хранил, чтобы cmdLogin реально смог его развернуть.
	salt := []byte("0123456789abcdef")
	params := crypto.DefaultKDFParams()
	kek := crypto.DeriveKey("master-password", salt, params)
	dek, err := crypto.GenerateDEK()
	require.NoError(t, err)
	wrapped, err := crypto.WrapDEK(dek, kek)
	require.NoError(t, err)

	st := &state{authClient: &fakeAuthClient{loginResp: newLoginResponse("tok", salt, params, wrapped)}}

	err = cmdLogin(st, []string{"ivan"})
	require.NoError(t, err)
	assert.Equal(t, "tok", st.token)
	assert.True(t, st.loggedIn)
	assert.Equal(t, dek, st.dek)
}

func TestCmdLogin_WrongPasswordCannotUnwrapDEK(t *testing.T) {
	withFakePassword(t, "wrong-password")

	salt := []byte("0123456789abcdef")
	params := crypto.DefaultKDFParams()
	kek := crypto.DeriveKey("real-password", salt, params) // завёрнуто ДРУГИМ паролем
	dek, err := crypto.GenerateDEK()
	require.NoError(t, err)
	wrapped, err := crypto.WrapDEK(dek, kek)
	require.NoError(t, err)

	st := &state{authClient: &fakeAuthClient{loginResp: newLoginResponse("tok", salt, params, wrapped)}}

	err = cmdLogin(st, []string{"ivan"})
	assert.Error(t, err)
	assert.False(t, st.loggedIn)
}

func TestCmdLogin_WrongArgs(t *testing.T) {
	st := &state{}
	err := cmdLogin(st, []string{})
	assert.Error(t, err)
}

func TestCmdLogin_ServerError(t *testing.T) {
	withFakePassword(t, "password")
	st := &state{authClient: &fakeAuthClient{loginErr: errors.New("invalid credentials")}}

	err := cmdLogin(st, []string{"ivan"})
	assert.Error(t, err)
	assert.False(t, st.loggedIn)
}
