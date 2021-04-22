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

	logger.Debugf("Registering client key...")
	var clientKey *keys.EdX25519Key
	if req.ClientKey != "" {
		c, err := keys.NewEdX25519KeyFromPaperKey(req.ClientKey)
		if err != nil {
			return nil, err
		}
		clientKey = c
	} else {
		clientKey = keys.GenerateEdX25519Key()
	}

	ck, err := s.vclient.Register(ctx, clientKey, accountKey)
	if err != nil {
		return nil, err
	}
	if err := s.vault.SetClientKey(ck); err != nil {
		return nil, err
	}

	// TODO: Register paper key for backup
	// logger.Debugf("Saving paper key...")
	// paperKey := accountKey.PaperKey()
	// if _, err := s.vault.RegisterPaperKey(paperKey); err != nil {
	// 	return nil, err
	// }

	logger.Debugf("Saving account key...")
	ak := kapi.NewKey(accountKey).WithLabels("account").Created(s.clock.NowMillis())
	ak.SetExtString("email", req.Email)
	if err := s.vault.Keyring().Save(ak); err != nil {
		return nil, err
	}

	return &AccountCreateResponse{}, nil
}

func (s *service) AccountSetUsername(ctx context.Context, req *AccountSetUsernameRequest) (*AccountSetUsernameResponse, error) {
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}

	if err := s.client.AccountSetUsername(ctx, account.AsEdX25519(), req.Username); err != nil {
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

	remote, err := s.client.Account(ctx, account.AsEdX25519())
	if err != nil {
		return nil, err
	}
	if remote.Username == "" {
		return &AccountStatusResponse{Status: AccountStatusUsername}, nil
	}

	team, err := s.team(false)
	if err != nil {
		return nil, err
	}
	if team == nil {
		o, err := s.checkInvites(ctx, account.AsEdX25519())
		if err != nil {
			return nil, err
		}
		team = o
	}
	if team == nil {
		return &AccountStatusResponse{Status: AccountStatusAcceptance}, nil
	}

	// TODO: Move to app init
	if err := s.importTeamChannels(ctx); err != nil {
		return nil, err
	}

	return &AccountStatusResponse{Status: AccountStatusComplete}, nil
}

func (s *service) account(required bool) (*kapi.Key, error) {
	key, err := s.vault.Keyring().KeyWithLabel("account")
	if err != nil {
		return nil, err
	}
	if required && key == nil {
		return nil, errors.Errorf("no account")
	}
	return key, nil
}

func (s *service) team(required bool) (*kapi.Key, error) {
	key, err := s.vault.Keyring().KeyWithLabel("team")
	if err != nil {
		return nil, err
	}
	if required && key == nil {
		return nil, errors.Errorf("no team")
	}
	return key, nil
}

func (s *service) checkInvites(ctx context.Context, account *keys.EdX25519Key) (*kapi.Key, error) {
	invites, err := s.client.TeamAccountInvites(ctx, account)
	if err != nil {
		return nil, err
	}
	for _, invite := range invites {
		team, err := invite.DecryptKey(account)
		if err != nil {
			logger.Errorf("Unable to decrypt invite key %v", err)
			continue
		}
		key := kapi.NewKey(team).Created(s.clock.NowMillis()).WithLabels("team")
		if err := s.vault.Keyring().Save(key); err != nil {
			return nil, err
		}
		return key, nil
	}
	return nil, nil
}
