package service

import (
	"context"
	"regexp"
	"strings"

	capi "github.com/getchill-app/http/api"
	"github.com/getchill-app/http/client"
	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

func (s *service) Channels(ctx context.Context, req *ChannelsRequest) (*ChannelsResponse, error) {
	channels, err := s.messenger.Channels()
	if err != nil {
		return nil, err
	}
	out := make([]*Channel, 0, len(channels))
	for _, channel := range channels {
		if channel.Visibility == messaging.VisibilityHidden {
			continue
		}
		out = append(out, channelToRPC(channel))
	}
	return &ChannelsResponse{
		Channels: out,
	}, nil
}

func (s *service) ChannelCreate(ctx context.Context, req *ChannelCreateRequest) (*ChannelCreateResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.Errorf("no channel name specified")
	}
	match, _ := regexp.MatchString("^[a-z0-9-]*$", name)
	if !match {
		return nil, errors.Errorf("invalid channel name")
	}
	if len(name) > 16 {
		return nil, errors.Errorf("channel name too long (must be < 16)")
	}
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}

	// Create channel key
	channelKey := keys.GenerateEdX25519Key()
	logger.Debugf("Creating channel %s", channelKey.ID())

	var channel *api.Key
	if !req.Private {
		team, err := s.team(true)
		if err != nil {
			return nil, err
		}
		vault, err := s.client.TeamCreateVault(ctx, team.ID, account.AsEdX25519(), channelKey)
		if err != nil {
			return nil, err
		}
		key := api.NewKey(channelKey)
		key.SetExtString("token", vault.Token)
		if err := s.messenger.AddKey(key); err != nil {
			return nil, err
		}
		channel = key
	} else {
		reg, err := s.messenger.AddChannel(ctx, channelKey, account.AsEdX25519())
		if err != nil {
			return nil, err
		}
		channel = reg
	}

	info := &messaging.ChannelInfo{Name: name}
	msg := messaging.NewMessageForChannelInfo(channelKey.ID(), account.ID, info)
	logger.Debugf("Sending channel info message...")
	if err := s.messenger.Send(ctx, msg); err != nil {
		return nil, err
	}

	logger.Debugf("Relay...")
	s.relay.Authorize([]string{channel.ExtString("token")})
	s.relay.Send(&RelayOutput{
		Channel: channelKey.ID().String(),
	})

	out, err := s.messenger.Channel(channelKey.ID())
	if err != nil {
		return nil, err
	}
	logger.Debugf("Channel: %+v", out)

	return &ChannelCreateResponse{
		Channel: channelToRPC(out),
	}, nil
}

func channelToRPC(channel *messaging.Channel) *Channel {
	if channel == nil {
		return nil
	}
	return &Channel{
		ID:        channel.ID.String(),
		Name:      channel.Name,
		Snippet:   channel.Snippet,
		Index:     channel.Index,
		ReadIndex: channel.ReadIndex,
	}
}

func (s *service) ChannelInvite(ctx context.Context, req *ChannelInviteRequest) (*ChannelInviteResponse, error) {
	return nil, errors.Errorf("not implemented")
}

func (s *service) ChannelRead(ctx context.Context, req *ChannelReadRequest) (*ChannelReadResponse, error) {
	return nil, errors.Errorf("not implemented")
}

func (s *service) ChannelLeave(ctx context.Context, req *ChannelLeaveRequest) (*ChannelLeaveResponse, error) {
	channel, err := keys.ParseID(req.Channel)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid channel")
	}

	if err := s.messenger.LeaveChannel(ctx, channel); err != nil {
		return nil, err
	}

	return &ChannelLeaveResponse{}, nil
}

func (s *service) importTeamChannels(ctx context.Context) error {
	team, err := s.team(true)
	if err != nil {
		return err
	}
	resp, err := s.client.TeamVaults(ctx, team.AsEdX25519(), &client.TeamVaultsOpts{EncryptedKeys: true})
	if err != nil {
		return err
	}

	for _, vault := range resp.Vaults {
		channel, err := capi.DecryptKey(vault.EncryptedKey, team.AsEdX25519())
		if err != nil {
			return err
		}
		key, err := s.messenger.Key(channel.ID())
		if err != nil {
			return err
		}
		if key == nil {
			key = api.NewKey(channel)
			key.SetExtString("token", vault.Token)
			logger.Debugf("Import team key %s", key.ID)
			if err := s.messenger.AddKey(key); err != nil {
				return err
			}
		}
	}
	return nil
}
