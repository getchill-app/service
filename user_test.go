package service

import (
	"context"
	"testing"

	"github.com/keys-pub/keys/http"
	"github.com/stretchr/testify/require"
)

func TestUserAddGithub(t *testing.T) {
	env := newTestServerEnv(t)
	service, closeFn := newTestService(t, env)
	defer closeFn()
	testAccountCreate(t, service, "alice@keys.pub", "testpassword")

	testImportKey(t, service, bob)

	resp, err := service.UserSign(context.TODO(), &UserSignRequest{
		KID:     bob.ID().String(),
		Service: "github",
		Name:    "bob",
	})
	require.NoError(t, err)

	service.users.Client().SetProxy("https://api.github.com/gists/1", func(ctx context.Context, req *http.Request) http.ProxyResponse {
		return http.ProxyResponse{Body: []byte(githubMock("bob", "1", resp.Message))}
	})

	// Bob
	addResp, err := service.UserAdd(context.TODO(), &UserAddRequest{
		KID:     bob.ID().String(),
		Service: "github",
		Name:    "Bob",
		URL:     "https://gist.github.com/Bob/1",
	})
	require.NoError(t, err)

	require.NotEmpty(t, addResp)
	require.NotEmpty(t, addResp.User)
	require.Equal(t, "bob", addResp.User.Name)
	require.Equal(t, "https://gist.github.com/bob/1", addResp.User.URL)
}
