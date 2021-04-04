package service

import (
	"context"
	"strings"

	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	"github.com/pkg/errors"
)

func (s *service) Channels(ctx context.Context, req *ChannelsRequest) (*ChannelsResponse, error) {
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
	user, err := s.currentUser()
	if err != nil {
		return nil, err
	}
	logger.Debugf("Current user: %s", user.ID)

	// Create channel key
	channelKey := keys.GenerateEdX25519Key()

	logger.Debugf("Adding channel %s", channelKey.ID())
	reg, err := s.messenger.AddChannel(ctx, channelKey, user.AsEdX25519())
	if err != nil {
		return nil, err
	}

	info := &messaging.ChannelInfo{Name: name}
	msg := messaging.NewMessageForChannelInfo(channelKey.ID(), user.ID, info)
	logger.Debugf("Sending channel info message...")
	if err := s.messenger.Send(ctx, msg); err != nil {
		return nil, err
	}

	logger.Debugf("Relay...")
	s.relay.Authorize([]string{reg.Token})
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
