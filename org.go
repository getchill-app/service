package service

import (
	"context"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

func (s *service) OrgKey(ctx context.Context, req *OrgKeyRequest) (*OrgKeyResponse, error) {
	key, err := s.vault.Keyring().KeyWithLabel(req.Domain)
	if err != nil {
		return nil, err
	}

	created := false
	if key == nil {
		key = api.NewKey(keys.GenerateEdX25519Key()).WithLabels("org", req.Domain).Created(s.clock.NowMillis())
		if err := s.vault.Keyring().Save(key); err != nil {
			return nil, err
		}
		created = true
	}

	if !key.HasLabel("org") {
		return nil, errors.Errorf("key missing org label")
	}
	verified := false
	if key.HasLabel("verified") {
		verified = true
	}

	return &OrgKeyResponse{KID: key.ID.String(), Created: created, Verified: verified}, nil
}

func (s *service) OrgSign(ctx context.Context, req *OrgSignRequest) (*OrgSignResponse, error) {
	key, err := s.vault.Keyring().KeyWithLabel(req.Domain)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, errors.Errorf("no org key setup")
	}
	if !key.IsEdX25519() {
		return nil, errors.Errorf("invalid key")
	}

	sig, err := s.client.OrgSign(key.AsEdX25519(), req.Domain, s.clock.Now())
	if err != nil {
		return nil, err
	}
	return &OrgSignResponse{Sig: sig}, nil
}

func (s *service) OrgCreate(ctx context.Context, req *OrgCreateRequest) (*OrgCreateResponse, error) {
	key, err := s.vault.Keyring().KeyWithLabel(req.Domain)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, errors.Errorf("no org key setup")
	}
	if !key.IsEdX25519() {
		return nil, errors.Errorf("invalid key")
	}
	account, err := s.currentAccount()
	if err != nil {
		return nil, err
	}
	if err := s.client.OrgCreate(ctx, key.AsEdX25519(), req.Domain, account.AsEdX25519()); err != nil {
		return nil, err
	}

	if err := s.vault.Keyring().Save(key); err != nil {
		return nil, err
	}

	return &OrgCreateResponse{}, nil
}
