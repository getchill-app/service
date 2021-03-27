package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthUnlock(t *testing.T) {
	env := newTestEnv(t)
	service, closeFn := newTestService(t, env)
	defer closeFn()
	ctx := context.TODO()

	// Setup/Unlock
	var err error
	password := "password123"
	_, err = service.AuthSetup(ctx, &AuthSetupRequest{
		Secret: password,
		Type:   PasswordAuth,
	})
	require.NoError(t, err)
	_, err = service.AuthUnlock(ctx, &AuthUnlockRequest{
		Secret: password,
		Type:   PasswordAuth,
		Client: "test",
	})
	require.NoError(t, err)

	// Unlock again
	_, err = service.AuthUnlock(ctx, &AuthUnlockRequest{
		Secret: password,
		Type:   PasswordAuth,
		Client: "test",
	})
	require.NoError(t, err)
}

func TestAuthUnlockMultipleClients(t *testing.T) {
	env := newTestEnv(t)
	service, closeFn := newTestService(t, env)
	defer closeFn()
	ctx := context.TODO()

	var err error
	password := "password123"
	_, err = service.AuthSetup(ctx, &AuthSetupRequest{
		Secret: password,
		Type:   PasswordAuth,
	})
	require.NoError(t, err)

	// Unlock app
	app, err := service.AuthUnlock(ctx, &AuthUnlockRequest{
		Secret: password,
		Type:   PasswordAuth,
		Client: "app",
	})
	require.NoError(t, err)

	// Unlock CLI
	cli, err := service.AuthUnlock(ctx, &AuthUnlockRequest{
		Secret: password,
		Type:   PasswordAuth,
		Client: "cli",
	})
	require.NoError(t, err)

	// Check tokens
	err = service.authIr.checkToken(app.AuthToken)
	require.NoError(t, err)
	err = service.authIr.checkToken(cli.AuthToken)
	require.NoError(t, err)

	// Lock
	_, err = service.AuthLock(ctx, &AuthLockRequest{})
	require.NoError(t, err)

	err = service.authIr.checkToken(app.AuthToken)
	require.EqualError(t, err, "rpc error: code = Unauthenticated desc = invalid token")
	err = service.authIr.checkToken(cli.AuthToken)
	require.EqualError(t, err, "rpc error: code = Unauthenticated desc = invalid token")

	require.False(t, service.db.IsOpen())
}

func TestGenerateToken(t *testing.T) {
	token := generateToken()
	require.NotEmpty(t, token)
}
