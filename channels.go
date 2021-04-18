package service

import (
	"context"
	"strings"

	"github.com/getchill-app/http/client"
	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

func (s *service) Channels(ctx context.Context, req *ChannelsRequest) (*ChannelsResponse, error) {
	// TODO: Move to channels init
	// if err := s.importOrgChannels(ctx); err != nil {
	// 	return nil, err
	// }

	status, err := s.messenger.ChannelStatuses()
	if err != nil {
		return nil, err
	}
	out := make([]*Channel, 0, len(status))
	for _, st := range status {
		out = append(out, channelToRPC(st))
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
		org, err := s.org(true)
		if err != nil {
			return nil, err
		}
		vault, err := s.client.OrgCreateVault(ctx, org.AsEdX25519(), channelKey)
		if err != nil {
			return nil, err
		}
		key := api.NewKey(channelKey)
		key.Token = vault.Token
		if _, err := s.messenger.AddKey(key); err != nil {
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
	s.relay.Authorize([]string{channel.Token})
	s.relay.Send(&RelayOutput{
		Channel: channelKey.ID().String(),
	})

	status, err := s.messenger.ChannelStatus(channelKey.ID())
	if err != nil {
		return nil, err
	}
	logger.Debugf("Channel status: %+v", status)

	return &ChannelCreateResponse{
		Channel: channelToRPC(status),
	}, nil
}

func channelToRPC(status *messaging.ChannelStatus) *Channel {
	if status == nil {
		return nil
	}
	return &Channel{
		ID:        status.Channel.String(),
		Name:      status.Name,
		Snippet:   status.Snippet,
		Index:     status.Index,
		ReadIndex: status.ReadIndex,
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

func (s *service) importOrgChannels(ctx context.Context) error {
	org, err := s.org(true)
	if err != nil {
		return err
	}
	resp, err := s.client.OrgVaults(ctx, org.AsEdX25519(), &client.OrgVaultsOpts{EncryptedKeys: true})
	if err != nil {
		return err
	}

	for _, vault := range resp.Vaults {
		channel, err := client.DecryptKey(vault.EncryptedKey, org.AsEdX25519())
		if err != nil {
			return err
		}
		key := api.NewKey(channel)
		key.Token = vault.Token
		logger.Debugf("Import org key %s", key.ID)
		if _, err := s.messenger.AddKey(key); err != nil {
			return err
		}
	}
	return nil
}
