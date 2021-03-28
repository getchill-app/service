package service

import (
	"context"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys-ext/http/api"
	"github.com/pkg/errors"
)

func (s *service) Channels(ctx context.Context, req *ChannelsRequest) (*ChannelsResponse, error) {
	return nil, errors.Errorf("not implemented")
}

func (s *service) ChannelCreate(ctx context.Context, req *ChannelCreateRequest) (*ChannelCreateResponse, error) {
	return nil, errors.Errorf("not implemented")
}

func (s *service) ChannelInvite(ctx context.Context, req *ChannelInviteRequest) (*ChannelInviteResponse, error) {
	return nil, errors.Errorf("not implemented")
}

func (s *service) ChannelRead(ctx context.Context, req *ChannelReadRequest) (*ChannelReadResponse, error) {
	return nil, errors.Errorf("not implemented")
}

func (s *service) ChannelLeave(ctx context.Context, req *ChannelLeaveRequest) (*ChannelLeaveResponse, error) {
	return nil, errors.Errorf("not implemented")
}

type channelStatus struct {
	ID              keys.ID `json:"id,omitempty" msgpack:"id,omitempty"`
	Name            string  `json:"name,omitempty" msgpack:"name,omitempty"`
	Description     string  `json:"desc,omitempty" msgpack:"desc,omitempty"`
	Snippet         string  `json:"snippet,omitempty" msgpack:"snippet,omitempty"`
	Index           int64   `json:"index,omitempty" msgpack:"index,omitempty"`
	Timestamp       int64   `json:"ts,omitempty" msgpack:"ts,omitempty"`
	RemoteTimestamp int64   `json:"rts,omitempty" msgpack:"rts,omitempty"`
	ReadIndex       int64   `json:"readIndex,omitempty" msgpack:"readIndex,omitempty"`
}

func (s channelStatus) Info() *api.ChannelInfo {
	return &api.ChannelInfo{
		Name:        s.Name,
		Description: s.Description,
	}
}

func (s channelStatus) Channel() *Channel {
	return &Channel{
		ID:        s.ID.String(),
		Name:      s.Name,
		Snippet:   s.Snippet,
		UpdatedAt: s.RemoteTimestamp,
		Index:     s.Index,
		ReadIndex: s.ReadIndex,
	}
}

func (s *service) channelStatus(ctx context.Context, cid keys.ID) (*channelStatus, error) {
	return nil, errors.Errorf("not implemented")
}

func (s *service) updateChannelStatus(ctx context.Context, status *channelStatus) error {
	return errors.Errorf("not implemented")
}
