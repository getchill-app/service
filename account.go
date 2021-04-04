package service

import (
	"context"

	"github.com/keys-pub/keys"
	kapi "github.com/keys-pub/keys/api"
	"github.com/keys-pub/vault"
	"github.com/pkg/errors"
)

func (s *service) AccountCreate(ctx context.Context, req *AccountCreateRequest) (*AccountCreateResponse, error) {
	logger.Infof("Auth setup...")
	if s.vault.Status() != vault.SetupNeeded {
		return nil, errors.Errorf("already setup")
	}

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
	if err := s.vclient.AccountCreate(ctx, accountKey, req.Email); err != nil {
		return nil, err
	}

	logger.Debugf("Registering client key...")
	var clientKey *keys.EdX25519Key
	if req.AccountKey != "" {
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

	unlock, mk, err := s.authUnlock(ctx, &AuthUnlockRequest{Secret: paperKey, Type: PaperKeyAuth})
	if err != nil {
		return nil, err
	}

	logger.Debugf("Saving account key...")
	ak := kapi.NewKey(accountKey).WithLabels("account").Created(s.clock.NowMillis())
	if err := s.vault.Keyring().Set(ak); err != nil {
		return nil, err
	}

	if req.Password != "" {
		logger.Debugf("Registering password...")
		if _, err := s.vault.RegisterPassword(mk, req.Password); err != nil {
			return nil, err
		}
	}

	return &AccountCreateResponse{
		AuthToken: unlock.AuthToken,
	}, nil
}

func (s *service) currentUser() (*kapi.Key, error) {
	accountKeys, err := s.vault.Keyring().KeysWithLabel("account")
	if err != nil {
		return nil, err
	}
	if len(accountKeys) == 0 {
		return nil, errors.Errorf("no current user")
	}
	return accountKeys[0], nil
}
