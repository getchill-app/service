package service

import (
	"context"

	"github.com/keys-pub/keys"
	kapi "github.com/keys-pub/keys/api"
	"github.com/keys-pub/keys/http/client"
	"github.com/pkg/errors"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

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
	if err := s.vclient.AccountCreate(ctx, accountKey, req.Email); err != nil {
		if client.IsConflict(err) {
			return nil, status.Error(codes.AlreadyExists, "account already exists")
		}
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

func (s *service) AccountVerify(ctx context.Context, req *AccountVerifyRequest) (*AccountVerifyResponse, error) {
	return nil, nil
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
