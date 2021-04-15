package service

import (
	"context"
	"testing"

	"github.com/keys-pub/keys/http"
	"github.com/stretchr/testify/require"
)

func TestOrgSignCreate(t *testing.T) {
	// SetLogger(NewLogger(DebugLevel))
	// client.SetLogger(NewLogger(DebugLevel))

	serverEnv := newTestServerEnv(t)
	// serverEnv.logLevel = DebugLevel
	service, closeFn := newTestService(t, serverEnv)
	defer closeFn()
	ctx := context.TODO()
	var err error
	_, err = service.AccountCreate(ctx, &AccountCreateRequest{
		Email:    "alice@keys.pub",
		Password: "testpassword",
	})
	require.NoError(t, err)

	keyResp, err := service.OrgKey(ctx, &OrgKeyRequest{Domain: "test.domain"})
	require.NoError(t, err)
	require.NotEmpty(t, keyResp.KID)

	resp, err := service.OrgSign(ctx, &OrgSignRequest{
		Domain: "test.domain",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Sig)

	// 400 testing
	serverEnv.client.SetProxy("https://test.domain/.well-known/getchill.txt", func(ctx context.Context, req *http.Request) http.ProxyResponse {
		return http.ProxyResponse{Err: http.Err{Code: 400, Message: "testing"}}
	})
	_, err = service.OrgCreate(ctx, &OrgCreateRequest{
		Domain: "test.domain",
	})
	require.EqualError(t, err, "failed to verify domain: testing (400)")

	// Sig found
	serverEnv.client.SetProxy("https://test.domain/.well-known/getchill.txt", func(ctx context.Context, req *http.Request) http.ProxyResponse {
		return http.ProxyResponse{Body: []byte(resp.Sig)}
	})
	_, err = service.OrgCreate(ctx, &OrgCreateRequest{
		Domain: "test.domain",
	})
	require.NoError(t, err)

}
