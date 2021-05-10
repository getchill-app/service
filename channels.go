package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/getchill-app/http/api"
	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	kapi "github.com/keys-pub/keys/api"
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
	info := &api.ChannelInfo{Name: name}

	token := ""
	if !req.Private {
		team, err := s.team(true)
		if err != nil {
			return nil, err
		}
		t, err := s.client.TeamCreateChannel(ctx, team.ID, channelKey, info, account.AsEdX25519())
		if err != nil {
			return nil, err
		}
		token = t
	} else {
		return nil, errors.Errorf("not implemented")
	}

	key := kapi.NewKey(channelKey).WithLabels("channel")
	key.SetExtString("token", token)
	if err := s.keyring.Set(key); err != nil {
		return nil, err
	}
	if err := s.messenger.AddChannel(channelKey.ID()); err != nil {
		return nil, err
	}
	if err := s.messenger.UpdateChannelInfo(channelKey.ID(), info); err != nil {
		return nil, err
	}

	out, err := s.messenger.Channel(channelKey.ID())
	if err != nil {
		return nil, err
	}

	logger.Debugf("Relay...")
	s.relay.RegisterTokens([]string{token})
	s.relay.Send(&RelayOutput{
		Channel: channelKey.ID().String(),
	})

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

	if err := s.messenger.HideChannel(ctx, channel); err != nil {
		return nil, err
	}

	return &ChannelLeaveResponse{}, nil
}

func (s *service) importTeamChannels(ctx context.Context) error {
	logger.Debugf("Import team channels...")
	team, err := s.team(true)
	if err != nil {
		return err
	}
	channels, err := s.client.TeamChannelKeys(ctx, team.AsEdX25519())
	if err != nil {
		return err
	}
	logger.Debugf("Found %d channel(s)", len(channels))
	for _, channel := range channels {
		logger.Debugf("Importing channel %s", channel.ID)
		channelKey, err := api.DecryptKey(channel.EncryptedKey, team.AsEdX25519())
		if err != nil {
			return err
		}
		key, err := s.keyring.Get(channelKey.ID())
		if err != nil {
			return err
		}
		if key == nil {
			key = kapi.NewKey(channelKey).Created(s.clock.NowMillis()).WithLabels("channel")
			key.SetExtString("token", channel.Token)
			logger.Debugf("Saving channel key %s", key.ID)
			if err := s.keyring.Set(key); err != nil {
				return err
			}
			if err := s.messenger.AddChannel(channelKey.ID()); err != nil {
				return err
			}
			info := channel.Info(channelKey)
			if info != nil {
				if err := s.messenger.UpdateChannelInfo(channelKey.ID(), info); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
