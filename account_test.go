package service

import (
	"context"
	"testing"

	client "github.com/getchill-app/http-client"
	"github.com/keys-pub/keys"
	"github.com/stretchr/testify/require"
)

func TestAccountCreate(t *testing.T) {
	SetLogger(NewLogger(DebugLevel))
	client.SetLogger(NewLogger(DebugLevel))

	env := newTestServerEnv(t)
	env.logLevel = DebugLevel
	service, closeFn := newTestService(t, env)
	defer closeFn()
	ctx := context.TODO()
	var err error

	_, err = service.AccountCreate(ctx, &AccountCreateRequest{
		Email:    "alice@keys.pub",
		Password: "testpassword",
	})
	require.NoError(t, err)

	ks, err := service.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(ks.Keys))
	account := ks.Keys[0]

	user, err := service.currentUser()
	require.NoError(t, err)
	require.Equal(t, user.ID, keys.ID(account.ID))
}
