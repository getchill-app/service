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
	aliceService, aliceCloseFn := testServiceSetup(t, env, "alice@keys.pub", alice)
	defer aliceCloseFn()
	testTeamCreate(t, aliceService, team)
	testUserSetupGithub(t, env, aliceService, alice, "alice")

	respKeys, err := aliceService.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, 2, len(respKeys.Keys))
	require.Equal(t, alice.ID().String(), respKeys.Keys[0].ID)

	// Bob
	testAccountInvite(t, aliceService, "bob@keys.pub")
	bobService, bobCloseFn := testServiceSetup(t, env, "bob@keys.pub", bob)
	defer bobCloseFn()
	testUserSetupGithub(t, env, bobService, bob, "bob")

	// Alice (pull bob)
	resp, err := aliceService.Pull(ctx, &PullRequest{Key: bob.ID().String()})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.KIDs))
	require.Equal(t, bob.ID().String(), resp.KIDs[0])
	respKeys, err = aliceService.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, 3, len(respKeys.Keys))
	require.Equal(t, alice.ID().String(), respKeys.Keys[0].ID)
	require.Equal(t, bob.ID().String(), respKeys.Keys[1].ID)

	// Charlie
	testAccountInvite(t, aliceService, "charlie@keys.pub")
	charlieService, charlieCloseFn := testServiceSetup(t, env, "charlie@keys.pub", charlie)
	defer charlieCloseFn()
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
