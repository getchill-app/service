package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountCreate(t *testing.T) {
	// defer SetLogger(NewLogger(DebugLevel))()

	env := newTestServerEnv(t)
	// env.logLevel = DebugLevel
	serviceEnv, closeFn := newTestServiceEnv(t, env)
	defer closeFn()
	service := serviceEnv.service
	ctx := context.TODO()
	var err error
	testAuthSetup(t, service)

	status, err := service.AccountStatus(ctx, &AccountStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, AccountStatusCreate, status.Status)

	_, err = service.AccountRegister(ctx, &AccountRegisterRequest{
		Email: "alice@keys.pub",
	})
	require.NoError(t, err)
	code := serviceEnv.getChillAppEnv.emailer.SentVerificationEmail("alice@keys.pub")
	require.NotEmpty(t, code)

	_, err = service.AccountCreate(ctx, &AccountCreateRequest{
		Email: "alice@keys.pub",
		Code:  code,
	})
	require.NoError(t, err)

	status, err = service.AccountStatus(ctx, &AccountStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, AccountStatusInviteCode, status.Status)

	// inviteCode := ""
	// _, err = service.AccountInviteCode(ctx, &AccountInviteCodeRequest{Code: inviteCode})
	// require.NoError(t, err)
	_, err = service.TeamCreate(ctx, &TeamCreateRequest{})
	require.NoError(t, err)

	status, err = service.AccountStatus(ctx, &AccountStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, AccountStatusUsername, status.Status)

	_, err = service.AccountSetUsername(ctx, &AccountSetUsernameRequest{
		Username: "alice",
	})
	require.NoError(t, err)

	status, err = service.AccountStatus(ctx, &AccountStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, AccountStatusComplete, status.Status)
}
