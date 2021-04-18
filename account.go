package service

import (
	"context"

	"github.com/keys-pub/keys"
	kapi "github.com/keys-pub/keys/api"
	"github.com/keys-pub/vault"
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

	logger.Debugf("Saving paper key...")
	paperKey := accountKey.PaperKey()
	if _, err := s.vault.SetupPaperKey(paperKey, ck); err != nil {
		return nil, err
	}

	token, mk, err := s.authUnlock(ctx, paperKey, PaperKeyAuth, "")
	if err != nil {
		return nil, err
	}

	logger.Debugf("Saving account key...")
	ak := kapi.NewKey(accountKey).WithLabels("account").Created(s.clock.NowMillis())
	ak.Email = req.Email
	if err := s.vault.Keyring().Save(ak); err != nil {
		return nil, err
	}

	if req.Password != "" {
		logger.Debugf("Registering password...")
		if _, err := s.vault.RegisterPassword(mk, req.Password); err != nil {
			return nil, err
		}
	}

	return &AccountCreateResponse{
		AuthToken: token,
	}, nil
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

func (s *service) org(required bool) (*kapi.Key, error) {
	key, err := s.vault.Keyring().KeyWithLabel("org")
	if err != nil {
		return nil, err
	}
	if required && key == nil {
		return nil, errors.Errorf("no org")
	}
	return key, nil
}

func (s *service) AccountStatus(ctx context.Context, req *AccountStatusRequest) (*AccountStatusResponse, error) {
	switch s.vault.Status() {
	case vault.SetupNeeded:
		return &AccountStatusResponse{Status: AccountSetupNeeded}, nil
	case vault.Locked:
		return &AccountStatusResponse{Status: AccountLocked}, nil
	}

	if s.vault.Status() != vault.Unlocked {
		return nil, errors.Errorf("invalid account status")
	}

	account, err := s.account(false)
	if err != nil {
		return nil, err
	}
	if account == nil {
		// If we are setup we should have an account
		return &AccountStatusResponse{Status: AccountUnknown}, nil
	}

	org, err := s.org(false)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return &AccountStatusResponse{Status: AccountOrgNeeded}, nil
	}

	return &AccountStatusResponse{Status: AccountRegistered}, nil
}
