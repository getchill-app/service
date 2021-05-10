package service

import (
	"context"

	"github.com/keys-pub/keys"
	kapi "github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

func (s *service) AccountRegister(ctx context.Context, req *AccountRegisterRequest) (*AccountRegisterResponse, error) {
	if err := s.client.AccountRegister(ctx, req.Email); err != nil {
		return nil, err
	}
	return &AccountRegisterResponse{}, nil
}

func (s *service) AccountCreate(ctx context.Context, req *AccountCreateRequest) (*AccountCreateResponse, error) {
	logger.Debugf("Creating account...")

	var accountKey *keys.EdX25519Key
	if req.AccountKey != "" {
		a, err := keys.NewEdX25519KeyFromPaperKey(req.AccountKey)
		if err != nil {
			return nil, err
		}
		accountKey = a
	} else {
		accountKey = keys.GenerateEdX25519Key()
	}
	if err := s.client.AccountCreate(ctx, accountKey, req.Email, req.Code); err != nil {
		return nil, err
	}

	logger.Debugf("Saving account key...")
	ak := kapi.NewKey(accountKey).WithLabels("account").Created(s.clock.NowMillis())
	ak.SetExtString("email", req.Email)
	if err := s.keyring.Set(ak); err != nil {
		return nil, err
	}

	return &AccountCreateResponse{}, nil
}

func (s *service) AccountSetUsername(ctx context.Context, req *AccountSetUsernameRequest) (*AccountSetUsernameResponse, error) {
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}

	if err := s.client.AccountSetUsername(ctx, req.Username, account.AsEdX25519()); err != nil {
		return nil, err
	}

	return &AccountSetUsernameResponse{}, nil
}

func (s *service) AccountStatus(ctx context.Context, req *AccountStatusRequest) (*AccountStatusResponse, error) {
	account, err := s.account(false)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return &AccountStatusResponse{Status: AccountStatusCreate}, nil
	}

	team, err := s.team(false)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return &AccountStatusResponse{Status: AccountStatusInviteCode}, nil
	}

	remote, err := s.client.Account(ctx, account.AsEdX25519())
	if err != nil {
		return nil, err
	}
	if remote.Username == "" {
		return &AccountStatusResponse{Status: AccountStatusUsername}, nil
	}

	return &AccountStatusResponse{Status: AccountStatusComplete}, nil
}

func (s *service) account(required bool) (*kapi.Key, error) {
	key, err := s.keyring.KeyWithLabel("account")
	if err != nil {
		return nil, err
	}
	if required && key == nil {
		return nil, errors.Errorf("no account")
	}
	return key, nil
}

func (s *service) AccountInvite(ctx context.Context, req *AccountInviteRequest) (*AccountInviteResponse, error) {
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}
	team, err := s.team(false)
	if err != nil {
		return nil, err
	}
	inviteCode, err := s.client.TeamInvite(ctx, team.AsEdX25519(), req.Email, account.AsEdX25519())
	if err != nil {
		return nil, err
	}
	return &AccountInviteResponse{InviteCode: inviteCode}, nil
}

func (s *service) AccountInviteAccept(ctx context.Context, req *AccountInviteAcceptRequest) (*AccountInviteAcceptResponse, error) {
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}
	existing, err := s.team(false)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.Errorf("already in team")
	}
	logger.Debugf("Team invite...")
	team, err := s.client.TeamInviteOpen(ctx, req.Code, account.AsEdX25519())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open team invite")
	}
	if err := s.saveTeam(team); err != nil {
		return nil, err
	}
	if err := s.importTeamChannels(ctx); err != nil {
		return nil, err
	}

	return &AccountInviteAcceptResponse{}, nil
}
