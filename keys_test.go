package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/keys-pub/keys/http"
	"github.com/stretchr/testify/require"
)

func TestKeys(t *testing.T) {
	SetLogger(NewLogger(DebugLevel))
	env := newTestServerEnv(t)
	ctx := context.TODO()

	// Alice
	service, closeFn := newTestService(t, env)
	defer closeFn()

	testAuthSetup(t, service)
	testImportKey(t, service, alice)
	testUserSetupGithub(t, env, service, alice, "alice")

	testImportKey(t, service, bob)
	testUserSetupGithub(t, env, service, bob, "bob")

	testImportKey(t, service, charlie)
	testUserSetupGithub(t, env, service, charlie, "charlie")

	// Default
	resp, err := service.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, "user", resp.SortField)
	require.Equal(t, SortAsc, resp.SortDirection)
	require.Equal(t, 3, len(resp.Keys))
	require.Equal(t, alice.ID().String(), resp.Keys[0].ID)
	require.NotNil(t, resp.Keys[0].User)
	require.Equal(t, "alice", resp.Keys[0].User.Name)
	require.Equal(t, string(keys.EdX25519), resp.Keys[0].Type)
	require.Equal(t, bob.ID().String(), resp.Keys[1].ID)
	require.NotNil(t, resp.Keys[1].User)
	require.Equal(t, "bob", resp.Keys[1].User.Name)
	require.Equal(t, charlie.ID().String(), resp.Keys[2].ID)
	require.NotNil(t, resp.Keys[2].User)
	require.Equal(t, "charlie", resp.Keys[2].User.Name)

	// KID (asc)
	resp, err = service.Keys(ctx, &KeysRequest{
		SortField: "kid",
	})
	require.NoError(t, err)
	require.Equal(t, "kid", resp.SortField)
	require.Equal(t, SortAsc, resp.SortDirection)
	require.Equal(t, 3, len(resp.Keys))
	require.Equal(t, alice.ID().String(), resp.Keys[0].ID)
	require.Equal(t, charlie.ID().String(), resp.Keys[1].ID)
	require.Equal(t, bob.ID().String(), resp.Keys[2].ID)

	// KID (desc)
	resp, err = service.Keys(ctx, &KeysRequest{
		SortField:     "kid",
		SortDirection: SortDesc,
	})
	require.NoError(t, err)
	require.Equal(t, "kid", resp.SortField)
	require.Equal(t, SortDesc, resp.SortDirection)
	require.Equal(t, 3, len(resp.Keys))
	require.Equal(t, bob.ID().String(), resp.Keys[0].ID)
	require.Equal(t, charlie.ID().String(), resp.Keys[1].ID)
	require.Equal(t, alice.ID().String(), resp.Keys[2].ID)

	// User (asc)
	resp, err = service.Keys(ctx, &KeysRequest{
		SortField: "user",
	})
	require.NoError(t, err)
	require.Equal(t, "user", resp.SortField)
	require.Equal(t, SortAsc, resp.SortDirection)
	require.Equal(t, 3, len(resp.Keys))
	require.Equal(t, alice.ID().String(), resp.Keys[0].ID)
	require.Equal(t, bob.ID().String(), resp.Keys[1].ID)
	require.Equal(t, charlie.ID().String(), resp.Keys[2].ID)

	// User (desc)
	resp, err = service.Keys(ctx, &KeysRequest{
		SortField:     "user",
		SortDirection: SortDesc,
	})
	require.NoError(t, err)
	require.Equal(t, "user", resp.SortField)
	require.Equal(t, SortDesc, resp.SortDirection)
	require.Equal(t, 3, len(resp.Keys))
	require.Equal(t, charlie.ID().String(), resp.Keys[0].ID)
	require.Equal(t, bob.ID().String(), resp.Keys[1].ID)
	require.Equal(t, alice.ID().String(), resp.Keys[2].ID)

	// Type
	resp, err = service.Keys(ctx, &KeysRequest{
		SortField: "type",
	})
	require.NoError(t, err)
	require.Equal(t, "type", resp.SortField)
	require.Equal(t, SortAsc, resp.SortDirection)
	require.Equal(t, 3, len(resp.Keys))
	require.Equal(t, alice.ID().String(), resp.Keys[0].ID)
	require.Equal(t, bob.ID().String(), resp.Keys[1].ID)
	require.Equal(t, charlie.ID().String(), resp.Keys[2].ID)
}

func TestKeysMissingSigchain(t *testing.T) {
	env := newTestServerEnv(t)
	service, closeFn := newTestService(t, env)
	defer closeFn()
	ctx := context.TODO()

	testAuthSetup(t, service)
	testImportKey(t, service, alice)
	testUserSetupGithub(t, env, service, alice, "alice")

	_, err := service.scs.Delete(alice.ID())
	require.NoError(t, err)

	resp, err := service.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Keys))
}

var alice = keys.NewEdX25519KeyFromSeed(keys.Bytes32(bytes.Repeat([]byte{0x01}, 32)))
var bob = keys.NewEdX25519KeyFromSeed(keys.Bytes32(bytes.Repeat([]byte{0x02}, 32)))
var charlie = keys.NewEdX25519KeyFromSeed(keys.Bytes32(bytes.Repeat([]byte{0x03}, 32)))

func testImportKey(t *testing.T, service *service, key *keys.EdX25519Key) {
	encoded, err := api.EncodeKey(api.NewKey(key), authPassword)
	require.NoError(t, err)
	_, err = service.KeyImport(context.TODO(), &KeyImportRequest{
		In:       []byte(encoded),
		Password: authPassword,
	})
	require.NoError(t, err)
}

type testUser struct {
	URL      string
	Response string
}

func testUserSetupGithub(t *testing.T, serverEnv *testServerEnv, service *service, key *keys.EdX25519Key, username string) *testUser {
	tu, err := userSetupGithub(serverEnv, service, key, username)
	require.NoError(t, err)
	return tu
}

func userSetupGithub(serverEnv *testServerEnv, service *service, key *keys.EdX25519Key, username string) (*testUser, error) {
	serviceName := "github"
	resp, err := service.UserSign(context.TODO(), &UserSignRequest{
		KID:     key.ID().String(),
		Service: serviceName,
		Name:    username,
	})
	if err != nil {
		return nil, err
	}

	id := hex.EncodeToString(sha256.New().Sum([]byte(serviceName + "/" + username))[:8])
	url := fmt.Sprintf("https://gist.github.com/%s/%s", username, id)
	api := "https://api.github.com/gists/" + id
	body := []byte(githubMock(username, id, resp.Message))

	// Set proxy response (for local)
	service.users.Client().SetProxy(api, func(ctx context.Context, req *http.Request) http.ProxyResponse {
		return http.ProxyResponse{Body: body}
	})
	serverEnv.client.SetProxy(api, func(ctx context.Context, req *http.Request) http.ProxyResponse {
		return http.ProxyResponse{Body: body}
	})

	_, err = service.UserAdd(context.TODO(), &UserAddRequest{
		KID:     key.ID().String(),
		Service: serviceName,
		Name:    username,
		URL:     url,
	})
	return &testUser{URL: api, Response: string(body)}, err
}

func githubMock(name string, id string, msg string) string {
	msg = strings.ReplaceAll(msg, "\n", "")
	return `{
		"id": "` + id + `",
		"files": {
			"gistfile1.txt": {
				"content": "` + msg + `"
			}		  
		},
		"owner": {
			"login": "` + name + `"
		}
	  }`
}
