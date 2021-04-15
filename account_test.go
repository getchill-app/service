package service

import (
	"context"
	"testing"

	"github.com/keys-pub/keys"
	"github.com/stretchr/testify/require"
)

func TestAccountCreate(t *testing.T) {
	defer SetLogger(NewLogger(DebugLevel))()
	// client.SetLogger(NewLogger(DebugLevel))

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

	out, err := service.currentAccount()
	require.NoError(t, err)
	require.Equal(t, out.ID, keys.ID(account.ID))

	_, err = service.AuthLock(ctx, &AuthLockRequest{})
	require.NoError(t, err)

	unlock, err := service.AuthUnlock(ctx, &AuthUnlockRequest{Secret: "testpassword", Type: PasswordAuth})
	require.NoError(t, err)
	require.NotEmpty(t, unlock.AuthToken)
}
