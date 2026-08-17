package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	secretv1 "github.com/SatzhanDev/gophKeeper/api/proto/secret/v1"
	"github.com/SatzhanDev/gophKeeper/internal/client/crypto"
)

// fakeSecretClient — реализация secretv1.SecretServiceClient для тестов,
// без реального gRPC-соединения и сервера.
type fakeSecretClient struct {
	createResp *secretv1.CreateSecretResponse
	createErr  error
	getResp    *secretv1.GetSecretResponse
	getErr     error
	listResp   *secretv1.ListSecretsResponse
	listErr    error
	updateResp *secretv1.UpdateSecretResponse
	updateErr  error
	deleteResp *secretv1.DeleteSecretResponse
	deleteErr  error
}

func (f *fakeSecretClient) CreateSecret(context.Context, *secretv1.CreateSecretRequest, ...grpc.CallOption) (*secretv1.CreateSecretResponse, error) {
	return f.createResp, f.createErr
}
func (f *fakeSecretClient) GetSecret(context.Context, *secretv1.GetSecretRequest, ...grpc.CallOption) (*secretv1.GetSecretResponse, error) {
	return f.getResp, f.getErr
}
func (f *fakeSecretClient) ListSecrets(context.Context, *secretv1.ListSecretsRequest, ...grpc.CallOption) (*secretv1.ListSecretsResponse, error) {
	return f.listResp, f.listErr
}
func (f *fakeSecretClient) UpdateSecret(context.Context, *secretv1.UpdateSecretRequest, ...grpc.CallOption) (*secretv1.UpdateSecretResponse, error) {
	return f.updateResp, f.updateErr
}
func (f *fakeSecretClient) DeleteSecret(context.Context, *secretv1.DeleteSecretRequest, ...grpc.CallOption) (*secretv1.DeleteSecretResponse, error) {
	return f.deleteResp, f.deleteErr
}

// Небольшие конструкторы поверх Opaque API (сеттеры вместо struct literal) —
// переиспользуются в нескольких тестах ниже.

func newSecret(id int64, metadata string, data []byte, version int32) *secretv1.Secret {
	s := &secretv1.Secret{}
	s.SetId(id)
	s.SetMetadata(metadata)
	s.SetData(data)
	s.SetVersion(version)
	return s
}

func newGetSecretResponse(secret *secretv1.Secret) *secretv1.GetSecretResponse {
	resp := &secretv1.GetSecretResponse{}
	resp.SetSecret(secret)
	return resp
}

func newListSecretsResponse(secrets ...*secretv1.Secret) *secretv1.ListSecretsResponse {
	resp := &secretv1.ListSecretsResponse{}
	resp.SetSecrets(secrets)
	return resp
}

func newCreateSecretResponse(id int64) *secretv1.CreateSecretResponse {
	resp := &secretv1.CreateSecretResponse{}
	resp.SetId(id)
	return resp
}

func newUpdateSecretResponse(version int32) *secretv1.UpdateSecretResponse {
	resp := &secretv1.UpdateSecretResponse{}
	resp.SetVersion(version)
	return resp
}

func TestParseSecretType(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want secretv1.SecretType
		ok   bool
	}{
		{"login", "login", secretv1.SecretType_SECRET_TYPE_LOGIN_PASSWORD, true},
		{"text", "text", secretv1.SecretType_SECRET_TYPE_TEXT, true},
		{"binary", "binary", secretv1.SecretType_SECRET_TYPE_BINARY, true},
		{"card", "card", secretv1.SecretType_SECRET_TYPE_CARD, true},
		{"unknown", "unknown", 0, false},
		{"empty", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSecretType(tt.in)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCmdList_Success(t *testing.T) {
	st := &state{
		token:        "tok",
		secretClient: &fakeSecretClient{listResp: newListSecretsResponse(newSecret(1, "site.com", nil, 0))},
	}
	assert.NoError(t, cmdList(st))
}

func TestCmdList_Error(t *testing.T) {
	st := &state{token: "tok", secretClient: &fakeSecretClient{listErr: errors.New("down")}}
	assert.Error(t, cmdList(st))
}

func TestCmdGet_Success(t *testing.T) {
	dek := make([]byte, 32)
	ciphertext, err := crypto.Encrypt(dek, []byte("plaintext-secret"))
	require.NoError(t, err)

	st := &state{
		token:        "tok",
		dek:          dek,
		secretClient: &fakeSecretClient{getResp: newGetSecretResponse(newSecret(1, "", ciphertext, 0))},
	}

	assert.NoError(t, cmdGet(st, []string{"1"}))
}

func TestCmdGet_WrongArgs(t *testing.T) {
	st := &state{}
	assert.Error(t, cmdGet(st, []string{}))
}

func TestCmdGet_InvalidID(t *testing.T) {
	st := &state{}
	assert.Error(t, cmdGet(st, []string{"not-a-number"}))
}

func TestCmdGet_ServerError(t *testing.T) {
	st := &state{token: "tok", secretClient: &fakeSecretClient{getErr: errors.New("not found")}}
	assert.Error(t, cmdGet(st, []string{"1"}))
}

func TestCmdGet_DecryptFailsWithWrongKey(t *testing.T) {
	realDEK := make([]byte, 32)
	ciphertext, err := crypto.Encrypt(realDEK, []byte("plaintext-secret"))
	require.NoError(t, err)

	wrongDEK := make([]byte, 32)
	wrongDEK[0] = 0xFF

	st := &state{
		token:        "tok",
		dek:          wrongDEK,
		secretClient: &fakeSecretClient{getResp: newGetSecretResponse(newSecret(1, "", ciphertext, 0))},
	}

	assert.Error(t, cmdGet(st, []string{"1"}))
}

func TestCmdAdd_Success(t *testing.T) {
	st := &state{
		token:        "tok",
		dek:          make([]byte, 32),
		secretClient: &fakeSecretClient{createResp: newCreateSecretResponse(5)},
	}

	assert.NoError(t, cmdAdd(st, []string{"login", "github.com", "login=ivan;password=secret"}))
}

func TestCmdAdd_TooFewArgs(t *testing.T) {
	st := &state{}
	assert.Error(t, cmdAdd(st, []string{"login", "meta"}))
}

func TestCmdAdd_UnknownType(t *testing.T) {
	st := &state{dek: make([]byte, 32)}
	assert.Error(t, cmdAdd(st, []string{"unknown-type", "meta", "data"}))
}

func TestCmdAdd_ServerError(t *testing.T) {
	st := &state{
		token:        "tok",
		dek:          make([]byte, 32),
		secretClient: &fakeSecretClient{createErr: errors.New("down")},
	}
	assert.Error(t, cmdAdd(st, []string{"text", "meta", "data"}))
}

func TestCmdUpdate_Success(t *testing.T) {
	st := &state{
		token: "tok",
		dek:   make([]byte, 32),
		secretClient: &fakeSecretClient{
			getResp:    newGetSecretResponse(newSecret(1, "", nil, 3)),
			updateResp: newUpdateSecretResponse(4),
		},
	}

	assert.NoError(t, cmdUpdate(st, []string{"1", "meta", "new-data"}))
}

func TestCmdUpdate_TooFewArgs(t *testing.T) {
	st := &state{}
	assert.Error(t, cmdUpdate(st, []string{"1", "meta"}))
}

func TestCmdUpdate_InvalidID(t *testing.T) {
	st := &state{}
	assert.Error(t, cmdUpdate(st, []string{"not-a-number", "meta", "data"}))
}

func TestCmdUpdate_GetError(t *testing.T) {
	st := &state{token: "tok", secretClient: &fakeSecretClient{getErr: errors.New("not found")}}
	assert.Error(t, cmdUpdate(st, []string{"1", "meta", "data"}))
}

func TestCmdUpdate_VersionConflict(t *testing.T) {
	st := &state{
		token: "tok",
		dek:   make([]byte, 32),
		secretClient: &fakeSecretClient{
			getResp:   newGetSecretResponse(newSecret(1, "", nil, 3)),
			updateErr: status.Error(codes.Aborted, "version conflict"),
		},
	}

	err := cmdUpdate(st, []string{"1", "meta", "new-data"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "изменена на другом устройстве")
}

func TestCmdDelete_Success(t *testing.T) {
	st := &state{token: "tok", secretClient: &fakeSecretClient{deleteResp: &secretv1.DeleteSecretResponse{}}}
	assert.NoError(t, cmdDelete(st, []string{"1"}))
}

func TestCmdDelete_InvalidID(t *testing.T) {
	st := &state{}
	assert.Error(t, cmdDelete(st, []string{"not-a-number"}))
}

func TestCmdDelete_WrongArgs(t *testing.T) {
	st := &state{}
	assert.Error(t, cmdDelete(st, []string{}))
}

func TestCmdDelete_ServerError(t *testing.T) {
	st := &state{token: "tok", secretClient: &fakeSecretClient{deleteErr: errors.New("down")}}
	assert.Error(t, cmdDelete(st, []string{"1"}))
}
