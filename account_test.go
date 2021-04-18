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
	serviceEnv, closeFn := newTestServiceEnv(t, env)
	defer closeFn()
	service := serviceEnv.service
	ctx := context.TODO()
	var err error

	status, err := service.AccountStatus(ctx, &AccountStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, AccountSetupNeeded, status.Status)

	_, err = service.AccountRegister(ctx, &AccountRegisterRequest{
		Email: "alice@keys.pub",
	})
	require.NoError(t, err)
	code := serviceEnv.getChillAppEnv.emailer.SentVerificationEmail("alice@keys.pub")
	require.NotEmpty(t, code)

	_, err = service.AccountCreate(ctx, &AccountCreateRequest{
		Email:    "alice@keys.pub",
		Code:     code,
		Password: "testpassword",
	})
	require.NoError(t, err)

	status, err = service.AccountStatus(ctx, &AccountStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, AccountOrgNeeded, status.Status)

	// Query keys
	ks, err := service.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(ks.Keys))
	account := ks.Keys[0]

	out, err := service.account(true)
	require.NoError(t, err)
	require.Equal(t, out.ID, keys.ID(account.ID))

	// Lock & Unlock
	_, err = service.AuthLock(ctx, &AuthLockRequest{})
	require.NoError(t, err)

	status, err = service.AccountStatus(ctx, &AccountStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, AccountLocked, status.Status)

	unlock, err := service.AuthUnlock(ctx, &AuthUnlockRequest{Secret: "testpassword", Type: PasswordAuth})
	require.NoError(t, err)
	require.NotEmpty(t, unlock.AuthToken)
}
