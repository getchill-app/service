package service

import (
	"context"
	"sync"
	"time"

	wsapi "github.com/getchill-app/ws/api"
	wsclient "github.com/getchill-app/ws/client"
	"github.com/keys-pub/keys/dstore"
	"github.com/pkg/errors"
)

type relayClient struct {
	ui RPC_RelayServer
	ws *wsclient.Client
}

type relay struct {
	sync.Mutex
	client *relayClient
}

func newRelay() *relay {
	return &relay{}
}

func (r *relay) Register(client *relayClient) {
	r.Lock()
	defer r.Unlock()
	r.client = client
}

func (r *relay) Unregister(client *relayClient) {
	r.Lock()
	defer r.Unlock()
	if client == r.client {
		r.client = nil
	}
}

func (r *relay) Send(out *RelayOutput) {
	r.Lock()
	defer r.Unlock()
	if r.client != nil {
		if err := r.client.ui.Send(out); err != nil {
			logger.Errorf("Failed to relay event: %v", err)
		}
	}
}

func (r *relay) RegisterTokens(tokens []string) {
	r.Lock()
	defer r.Unlock()
	if r.client != nil {
		if err := r.client.ws.Register(tokens); err != nil {
			logger.Errorf("Failed to relay auth: %v", err)
		}
	}
}

// Relay (RPC) ...
func (s *service) Relay(req *RelayRequest, srv RPC_RelayServer) error {
	ctx := srv.Context()

	account, err := s.account(true)
	if err != nil {
		return err
	}

	config, err := s.client.Config(ctx, account.AsEdX25519())
	if err != nil {
		return err
	}
	if config == nil {
		return errors.Errorf("no config")
	}

	relay, err := wsclient.New(config.RelayURL, config.RelayAuth)
	if err != nil {
		return err
	}

	if err := relay.Connect(); err != nil {
		return err
	}
	defer relay.Close()

	client := &relayClient{ui: srv, ws: relay}
	s.relay.Register(client)
	defer s.relay.Unregister(client)

	tokens, err := s.relayTokens(ctx)
	if err != nil {
		return err
	}

	logger.Debugf("Relay tokens (%d)", len(tokens))
	if err := relay.Register(tokens); err != nil {
		return err
	}

	chEvents := make(chan []*wsapi.Event)

	wctx, cancel := context.WithCancel(ctx)

	go func() {
		for {
			logger.Infof("Read relay events...")
			events, err := relay.ReadEvents()
			if err != nil {
				logger.Errorf("Error reading events: %v", err)
				cancel()
				return
			}
			chEvents <- events
		}
	}()

	if err := s.updateChannels(ctx); err != nil {
		return err
	}

	// Send relay event after connect/register/update
	if err := srv.Send(&RelayOutput{Type: "connected"}); err != nil {
		return err
	}

	ticker := time.NewTicker(50 * time.Second)

	for {
		select {
		case <-wctx.Done():
			return errors.Wrapf(wctx.Err(), "relay failed")
		case <-ticker.C:
			logger.Debugf("Send ping...")
			if err := relay.Ping(); err != nil {
				return err
			}
		case events := <-chEvents:
			logger.Infof("Got relay events...")
			for _, event := range events {
				select {
				case <-wctx.Done():
					return errors.Wrapf(wctx.Err(), "relay failed")
				default:
					switch event.Type {
					case wsapi.ChannelType:
						if event.Channel != nil && event.Channel.ID != "" {
							logger.Debugf("Channel event %s", event.Channel.ID)
							if err := s.PullMessages(ctx, event.Channel.ID); err != nil {
								return err
							}
						}
					case wsapi.ChannelsType:
						logger.Debugf("Channels event")
						if err := s.updateChannels(ctx); err != nil {
							return err
						}
					}
				}
			}
			for _, event := range events {
				out := relayEventToRPC(event)
				if err := srv.Send(out); err != nil {
					return err
				}
			}
		}
	}
}

func relayEventToRPC(event *wsapi.Event) *RelayOutput {
	out := &RelayOutput{
		Type: string(event.Type),
	}
	if event.Channel != nil {
		out.Channel = &RelayOutput_Channel{
			ID:    event.Channel.ID.String(),
			Index: event.Channel.Index,
		}
	}
	return out
}

func (s *service) relayTokens(ctx context.Context) ([]string, error) {
	ks, err := s.keyring.KeysWithLabel("channel")
	if err != nil {
		return nil, err
	}
	tokens := dstore.NewStringSet()
	for _, k := range ks {
		if k.ExtString("token") != "" {
			tokens.Add(k.ExtString("token"))
		} else {
			logger.Warningf("Missing token for channel %s", k.ID)
		}
	}
	return tokens.Strings(), nil
}
