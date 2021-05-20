package service

import (
	"context"
	"strings"

	"github.com/pkg/errors"
)

func (s *service) command(ctx context.Context, cmd string, channel string) (*Message, error) {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return nil, errors.Errorf("no command")
	}

	cmd0, args := fields[0], fields[1:]
	logger.Debugf("Channel command: %s %v", cmd0, args)

	switch cmd0 {
	case "/leave":
		_, err := s.ChannelLeave(ctx, &ChannelLeaveRequest{Channel: channel})
		if err != nil {
			return nil, err
		}
		return nil, nil
	case "/create":
		_, err := s.ChannelCreate(ctx, &ChannelCreateRequest{Name: args[0]})
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	return nil, errors.Errorf("unrecognized command")
}
