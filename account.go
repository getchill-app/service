package service

import (
	"context"

	"github.com/keys-pub/keys"
	kapi "github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

func (s *service) AccountCreate(ctx context.Context, req *AccountCreateRequest) (*AccountCreateResponse, error) {
	logger.Debugf("Creating account...")

	if req.Password == "" {
		return nil, errors.Errorf("empty password")
	}

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

	logger.Debugf("Registering password...")
	if _, err := s.vault.RegisterPassword(mk, req.Password); err != nil {
		return nil, err
	}

	return &AccountCreateResponse{
		AuthToken: token,
	}, nil
}

func (s *service) currentAccount() (*kapi.Key, error) {
	return s.vault.Keyring().KeyWithLabel("account")
}

func (s *service) currentOrg() (*kapi.Key, error) {
	return s.vault.Keyring().KeyWithLabel("org")
}
