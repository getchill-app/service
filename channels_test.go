package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMessages(t *testing.T) {
	// SetLogger(NewLogger(DebugLevel))
	// client.SetLogger(NewLogger(DebugLevel))

	env := newTestServerEnv(t)
	ctx := context.TODO()

	// Alice
	aliceServiceEnv, aliceCloseFn := newTestServiceEnv(t, env)
	defer aliceCloseFn()
	aliceService := aliceServiceEnv.service

	testAuthSetup(t, aliceService)

	testAccountSetup(t, aliceServiceEnv, "alice@keys.pub", alice)
	testTeamCreate(t, aliceService, team)
	testUserSetupGithub(t, env, aliceService, alice, "alice")

	// Alice creates a channel
	channelCreate, err := aliceService.ChannelCreate(ctx, &ChannelCreateRequest{
		Name: "testing",
	})
	require.NoError(t, err)
	require.NotEmpty(t, channelCreate.Channel)
	channel := channelCreate.Channel

	// Channels (alice)
	channelsAlice, err := aliceService.Channels(ctx, &ChannelsRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(channelsAlice.Channels))
	require.Equal(t, "testing", channelsAlice.Channels[0].Name)

	msgAlice, err := aliceService.MessageSend(ctx, &MessageSendRequest{Channel: channel.ID, Text: "hi bob"})
	require.NoError(t, err)
	require.NotEmpty(t, msgAlice.Message.ID)

	msgsAlice, err := aliceService.Messages(ctx, &MessagesRequest{Channel: channel.ID, Update: true})
	require.NoError(t, err)
	require.Equal(t, 1, len(msgsAlice.Messages))
	require.Equal(t, []string{"hi bob"}, msgsAlice.Messages[0].Text)

	// // Leave
	// _, err = aliceService.ChannelLeave(ctx, &ChannelLeaveRequest{
	// 	Channel: channelCreate.Channel.ID,
	// })
	// require.NoError(t, err)

	// Bob
	bobServiceEnv, bobCloseFn := newTestServiceEnv(t, env)
	defer bobCloseFn()
	bobService := bobServiceEnv.service

	testAuthSetup(t, bobService)

	inviteCode := testAccountInvite(t, aliceService, "bob@keys.pub")
	testAccountSetup(t, bobServiceEnv, "bob@keys.pub", bob)
	testAccountInviteAccept(t, bobService, inviteCode)

	// Channels (bob)
	channelsBob, err := bobService.Channels(ctx, &ChannelsRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(channelsBob.Channels))
	require.Equal(t, "testing", channelsBob.Channels[0].Name)

	msgsBob, err := bobService.Messages(ctx, &MessagesRequest{Channel: channel.ID, Update: true})
	require.NoError(t, err)
	require.Equal(t, 1, len(msgsBob.Messages))
	require.Equal(t, []string{"hi bob"}, msgsBob.Messages[0].Text)
}
