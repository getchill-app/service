package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/dstore/events"
	"github.com/keys-pub/keys/encoding"
	"github.com/keys-pub/keys/tsutil"
	"github.com/pkg/errors"
)

// MessagePrepare returns a Message for an in progress display. The client
// should then use messageCreate to save the message. Prepare needs to be fast,
// so the client can show the a pending message right away.
// Preparing before create is optional.
func (s *service) MessagePrepare(ctx context.Context, req *MessagePrepareRequest) (*MessagePrepareResponse, error) {
	if req.Channel == "" {
		return nil, errors.Errorf("no channel specified")
	}

	account, err := s.account(true)
	if err != nil {
		return nil, err
	}
	sender, err := s.key(ctx, account.ID)
	if err != nil {
		return nil, err
	}

	text := processText(req.Text)

	id := encoding.MustEncode(keys.RandBytes(32), encoding.Base62)
	message := &Message{
		ID:        id,
		Sender:    sender,
		Text:      []string{text},
		Status:    MessagePending,
		CreatedAt: tsutil.Millis(s.clock.Now()),
	}

	return &MessagePrepareResponse{
		Message: message,
	}, nil
}

func processText(s string) string {
	return strings.TrimSpace(s)
}

// MessageCreate (RPC) creates a message for a recipient.
func (s *service) MessageCreate(ctx context.Context, req *MessageCreateRequest) (*MessageCreateResponse, error) {
	if req.Channel == "" {
		return nil, errors.Errorf("no channel specified")
	}

	text := processText(req.Text)
	if strings.HasPrefix(text, "/") {
		msg, err := s.command(ctx, text, req.Channel)
		if err != nil {
			return nil, err
		}
		return &MessageCreateResponse{Message: msg}, nil
	}

	account, err := s.account(true)
	if err != nil {
		return nil, err
	}

	channel, err := keys.ParseID(req.Channel)
	if err != nil {
		return nil, err
	}

	// TODO: Prev
	msg := messaging.NewMessage(channel, account.ID).WithText(text).WithTimestamp(s.clock.NowMillis())
	if req.ID != "" {
		msg.ID = req.ID
	}

	if err := s.messenger.Send(ctx, msg); err != nil {
		return nil, err
	}

	out, err := s.messageToRPC(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &MessageCreateResponse{
		Message: out,
	}, nil
}

// Messages (RPC) lists messages.
func (s *service) Messages(ctx context.Context, req *MessagesRequest) (*MessagesResponse, error) {
	channel, err := keys.ParseID(req.Channel)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid channel")
	}

	if req.Update {
		if err := s.messenger.SyncVault(ctx, channel); err != nil {
			return nil, err
		}
	}

	messages, err := s.messenger.Messages(channel)
	if err != nil {
		return nil, err
	}
	out, err := s.messagesToRPC(ctx, messages)
	if err != nil {
		return nil, err
	}

	return &MessagesResponse{
		Messages: out,
	}, nil
}

// MessagesOpts options for Messages.
type MessagesOpts struct {
	// Index to list to/from
	Index int64
	// Order ascending or descending
	Order events.Direction
	// Limit by
	Limit int
}

func (s *service) messagesToRPC(ctx context.Context, msgs []*messaging.Message) ([]*Message, error) {
	out := make([]*Message, 0, len(msgs))
	for _, msg := range msgs {
		m, err := s.messageToRPC(ctx, msg)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *service) messageToRPC(ctx context.Context, msg *messaging.Message) (*Message, error) {
	if msg == nil {
		return nil, nil
	}
	if msg.Sender == "" {
		return nil, errors.Errorf("no sender")
	}

	sender, err := s.key(ctx, msg.Sender)
	if err != nil {
		return nil, err
	}
	text, err := s.messageText(ctx, msg, sender)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:        msg.ID,
		Text:      text,
		Sender:    sender,
		CreatedAt: msg.Timestamp,
	}, nil
}

func userNameForKey(k *Key) string {
	if k.User != nil && k.User.ID != "" {
		return k.User.ID
	}
	return k.ID
}

func (s *service) messageText(ctx context.Context, msg *messaging.Message, sender *Key) ([]string, error) {
	texts := []string{}
	if msg.Text != "" {
		texts = append(texts, msg.Text)
	}

	if msg.Command != nil {
		if msg.Command.ChannelInfo != nil && msg.Command.ChannelInfo.Name != "" {
			texts = append(texts, fmt.Sprintf("%s set the channel name to %s", userNameForKey(sender), msg.Command.ChannelInfo.Name))
		}
		if msg.Command.ChannelInfo != nil && msg.Command.ChannelInfo.Description != "" {
			texts = append(texts, fmt.Sprintf("%s set the channel description to %s", userNameForKey(sender), msg.Command.ChannelInfo.Description))
		}

		// for _, invite := range msg.Command.ChannelInvites {
		// 	recipient, err := s.resolveKey(ctx, invite.Recipient)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	texts = append(texts, fmt.Sprintf("%s invited %s", sender.userName(), recipient.userName()))
		// }
	}

	return texts, nil
}
