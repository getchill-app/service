package service

import (
	"context"

	"github.com/keys-pub/keys"
	kapi "github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

func (s *service) TeamCreate(ctx context.Context, req *TeamCreateRequest) (*TeamCreateResponse, error) {
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}
	existing, err := s.team(false)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.Errorf("already in a team")
	}

	var teamKey *keys.EdX25519Key
	if req.TeamKey != "" {
		a, err := keys.NewEdX25519KeyFromPaperKey(req.TeamKey)
		if err != nil {
			return nil, err
		}
		teamKey = a
	} else {
		teamKey = keys.GenerateEdX25519Key()
	}
	if err := s.client.TeamCreate(ctx, teamKey, account.AsEdX25519()); err != nil {
		return nil, err
	}

	if err := s.saveTeam(teamKey); err != nil {
		return nil, err
	}

	return &TeamCreateResponse{}, nil
}

func (s *service) saveTeam(team *keys.EdX25519Key) error {
	logger.Debugf("Save team %s", team.ID())
	teamKey := kapi.NewKey(team).Created(s.clock.NowMillis()).WithLabels("team")
	if err := s.keyring.Set(teamKey); err != nil {
		return err
	}
	return nil
}

func (s *service) team(required bool) (*kapi.Key, error) {
	key, err := s.keyring.KeyWithLabel("team")
	if err != nil {
		return nil, err
	}
	if required && key == nil {
		return nil, errors.Errorf("no team")
	}
	return key, nil
}
