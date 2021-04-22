package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannel(t *testing.T) {
	// SetLogger(NewLogger(DebugLevel))
	// client.SetLogger(NewLogger(DebugLevel))
	// vault.SetLogger(NewLogger(DebugLevel))

	env := newTestServerEnv(t)
	ctx := context.TODO()

	aliceServiceEnv, aliceCloseFn := newTestServiceEnv(t, env)
	defer aliceCloseFn()
	aliceService := aliceServiceEnv.service

	testAccountSetup(t, aliceServiceEnv, "alice@keys.pub", alice)
	testUserSetupGithub(t, env, aliceService, alice, "alice")

	// Alice creates a channel
	channelCreate, err := aliceService.ChannelCreate(ctx, &ChannelCreateRequest{
		Name: "Test",
	})
	require.NoError(t, err)
	require.NotEmpty(t, channelCreate.Channel)
	// channel := channelCreate.Channel

	// Channels (alice)
	channels, err := aliceService.Channels(ctx, &ChannelsRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(channels.Channels))
	require.Equal(t, "Test", channels.Channels[0].Name)

	// Leave
	_, err = aliceService.ChannelLeave(ctx, &ChannelLeaveRequest{
		Channel: channelCreate.Channel.ID,
	})
	require.NoError(t, err)
}
