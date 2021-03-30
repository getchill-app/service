package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannel(t *testing.T) {
	// SetLogger(NewLogger(DebugLevel))
	// saltpack.SetLogger(NewLogger(DebugLevel))
	// client.SetLogger(NewLogger(DebugLevel))

	env := newTestServerEnv(t)

	aliceService, aliceCloseFn := newTestService(t, env)
	defer aliceCloseFn()
	testAuthSetup(t, aliceService)
	ctx := context.TODO()
	testImportKey(t, aliceService, alice)
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

	// export, err := aliceService.KeyExport(ctx, &KeyExportRequest{
	// 	KID:        channel.ID,
	// 	NoPassword: true,
	// })
	// require.NoError(t, err)

	// // Bob service
	// bobService, bobCloseFn := newTestService(t, env)
	// defer bobCloseFn()
	// testAuthSetup(t, bobService)
	// testImportKey(t, bobService, bob)
	// testUserSetupGithub(t, env, bobService, bob, "bob")

	// // Channels (bob)
	// _, err = bobService.KeyImport(ctx, &KeyImportRequest{
	// 	In: export.Export,
	// })
	// require.NoError(t, err)
	// channels, err = bobService.Channels(ctx, &ChannelsRequest{})
	// require.NoError(t, err)
	// require.Equal(t, 1, len(channels.Channels))
	// require.Equal(t, channel.ID, channels.Channels[0].ID)
	// require.Equal(t, "Test", channels.Channels[0].Name)
}
