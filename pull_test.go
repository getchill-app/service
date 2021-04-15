package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPull(t *testing.T) {
	ctx := context.TODO()
	env := newTestServerEnv(t)

	// Alice
	aliceService, aliceCloseFn := newTestService(t, env)
	defer aliceCloseFn()
	testAccountCreate(t, aliceService, "alice@keys.pub", "testpassword")
	testImportKey(t, aliceService, alice)
	testUserSetupGithub(t, env, aliceService, alice, "alice")

	respKeys, err := aliceService.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(respKeys.Keys))
	require.Equal(t, alice.ID().String(), respKeys.Keys[0].ID)

	// Bob
	bobService, bobCloseFn := newTestService(t, env)
	defer bobCloseFn()
	testAccountCreate(t, bobService, "bob@keys.pub", "testpassword")
	testImportKey(t, bobService, bob)
	testUserSetupGithub(t, env, bobService, bob, "bob")

	// Alice (pull bob)
	resp, err := aliceService.Pull(ctx, &PullRequest{Key: bob.ID().String()})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.KIDs))
	require.Equal(t, bob.ID().String(), resp.KIDs[0])
	respKeys, err = aliceService.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, 2, len(respKeys.Keys))
	require.Equal(t, alice.ID().String(), respKeys.Keys[0].ID)
	require.Equal(t, bob.ID().String(), respKeys.Keys[1].ID)

	// Charlie
	charlieService, charlieCloseFn := newTestService(t, env)
	defer charlieCloseFn()
	testAccountCreate(t, charlieService, "charlie@keys.pub", "testpassword")
	testImportKey(t, charlieService, charlie)
	testUserSetupGithub(t, env, charlieService, charlie, "charlie")

	// Charlie (pull alice@github)
	resp, err = charlieService.Pull(ctx, &PullRequest{Key: "alice@github"})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.KIDs))
	require.Equal(t, alice.ID().String(), resp.KIDs[0])
	respKeys, err = charlieService.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, 2, len(respKeys.Keys))
	require.Equal(t, alice.ID().String(), respKeys.Keys[0].ID)
	require.Equal(t, charlie.ID().String(), respKeys.Keys[1].ID)

	// Alice (pull alice@github)
	resp, err = aliceService.Pull(ctx, &PullRequest{Key: "alice@github"})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.KIDs))
	require.Equal(t, alice.ID().String(), resp.KIDs[0])
}
