package service

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/getchill-app/http/api"
	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	kapi "github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

func (s *service) Channels(ctx context.Context, req *ChannelsRequest) (*ChannelsResponse, error) {
	if req.Update {
		if err := s.updateChannels(ctx); err != nil {
			return nil, err
		}
	}
	channels, err := s.messenger.Channels()
	if err != nil {
		return nil, err
	}
	out := make([]*Channel, 0, len(channels))
	for _, channel := range channels {
		out = append(out, channelToRPC(channel))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
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
	info := &api.ChannelInfo{Name: name, Description: req.Description}

	if !req.Private {
		team, err := s.team(true)
		if err != nil {
			return nil, err
		}
		if _, err := s.client.ChannelCreateWithTeam(ctx, channelKey, info, team.ID, account.AsEdX25519()); err != nil {
			return nil, err
		}
	} else {
		if _, err := s.client.ChannelCreateWithUsers(ctx, channelKey, info, []keys.ID{account.ID}, account.AsEdX25519()); err != nil {
			return nil, err
		}
	}

	// Relay will get the update, or manually update channels list if not listening on relay.

	return &ChannelCreateResponse{
		ID: channelKey.ID().String(),
	}, nil
}

func (s *service) ChannelUsers(ctx context.Context, req *ChannelUsersRequest) (*ChannelUsersResponse, error) {
	cid, err := keys.ParseID(req.Channel)
	if err != nil {
		return nil, err
	}
	channelKey, err := s.keyring.Key(cid)
	if err != nil {
		return nil, err
	}
	users, err := s.client.ChannelUsers(ctx, channelKey.AsEdX25519())
	if err != nil {
		return nil, err
	}
	out := []*ChannelUser{}
	for _, u := range users {
		name, err := s.userName(ctx, u)
		if err != nil {
			return nil, err
		}
		out = append(out, &ChannelUser{
			ID:   u.String(),
			Name: name,
		})
	}
	return &ChannelUsersResponse{Users: out}, nil
}

func (s *service) ChannelUsersAdd(ctx context.Context, req *ChannelUsersAddRequest) (*ChannelUsersAddResponse, error) {
	cid, err := keys.ParseID(req.Channel)
	if err != nil {
		return nil, err
	}
	channelKey, err := s.keyring.Key(cid)
	if err != nil {
		return nil, err
	}
	users := []keys.ID{}
	for _, u := range req.Users {
		user, err := keys.ParseID(u)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := s.client.ChannelUsersAdd(ctx, channelKey.AsEdX25519(), users); err != nil {
		return nil, err
	}
	return &ChannelUsersAddResponse{}, nil
}

func (s *service) ChannelUsersRemove(ctx context.Context, req *ChannelUsersRemoveRequest) (*ChannelUsersRemoveResponse, error) {
	cid, err := keys.ParseID(req.Channel)
	if err != nil {
		return nil, err
	}
	channelKey, err := s.keyring.Key(cid)
	if err != nil {
		return nil, err
	}
	users := []keys.ID{}
	for _, u := range req.Users {
		user, err := keys.ParseID(u)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := s.client.ChannelUsersRemove(ctx, channelKey.AsEdX25519(), users); err != nil {
		return nil, err
	}
	return &ChannelUsersRemoveResponse{}, nil
}

func channelToRPC(channel *messaging.Channel) *Channel {
	if channel == nil {
		return nil
	}

	c := &Channel{
		ID:      channel.ID.String(),
		Name:    channel.Name,
		Snippet: channel.Snippet,
		Index:   channel.MessageIndex,
	}
	if channel.Team != "" {
		c.Type = TeamChannelType
	} else {
		c.Type = UsersChannelType
	}
	return c
}

func (s *service) ChannelRead(ctx context.Context, req *ChannelReadRequest) (*ChannelReadResponse, error) {
	return nil, errors.Errorf("not implemented")
}

func (s *service) ChannelLeave(ctx context.Context, req *ChannelLeaveRequest) (*ChannelLeaveResponse, error) {
	// channel, err := keys.ParseID(req.Channel)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "invalid channel")
	// }
	return nil, errors.Errorf("not implemented")
}

func (s *service) updateChannels(ctx context.Context) error {
	logger.Debugf("List channels...")
	account, err := s.account(true)
	if err != nil {
		return err
	}
	team, err := s.team(true)
	if err != nil {
		return err
	}
	channels, err := s.client.Channels(ctx, team.ID, account.AsEdX25519())
	if err != nil {
		return err
	}
	logger.Debugf("Found %d channel(s)", len(channels))
	for _, channel := range channels {
		var channelKey *keys.EdX25519Key
		if channel.TeamKey != nil {
			k, err := api.DecryptKey(channel.TeamKey, team.AsEdX25519())
			if err != nil {
				return err
			}
			channelKey = k
		} else if channel.UserKey != nil {
			k, err := api.DecryptKey(channel.UserKey, account.AsEdX25519())
			if err != nil {
				return err
			}
			channelKey = k
		}
		key, err := s.keyring.Get(channelKey.ID())
		if err != nil {
			return err
		}
		if key == nil {
			key = kapi.NewKey(channelKey).Created(s.clock.NowMillis()).WithLabels("channel")
			key.SetExtString("token", channel.Token)
			logger.Debugf("Saving channel %s", key.ID)
			if err := s.keyring.Set(key); err != nil {
				return err
			}
		}

		// Update token
		if key.ExtString("token") != channel.Token {
			logger.Debugf("Updating channel token for %s", channel.ID)
			key.SetExtString("token", channel.Token)
			if err := s.keyring.Set(key); err != nil {
				return err
			}
		}

		// Check channel
		existing, err := s.messenger.Channel(channelKey.ID())
		if err != nil {
			return err
		}

		ch, err := messaging.NewChannelFromAPI(channel, channelKey)
		if err != nil {
			return err
		}

		// If we don't have it, add it
		if existing == nil {
			if err := s.messenger.AddChannel(ch); err != nil {
				return err
			}
		} else {
			if err := s.messenger.UpdateChannel(ch); err != nil {
				return err
			}
		}

		// Check if we need to pull messages
		if ch.MessageIndex != channel.Index {
			if err := s.PullMessages(ctx, channelKey.ID()); err != nil {
				return err
			}
		}
	}
	return nil
}
