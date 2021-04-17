package service

import (
	"context"

	"github.com/keys-pub/vault"
)

// Status (RPC) gets the current app status.
func (s *service) Status(ctx context.Context, req *StatusRequest) (*StatusResponse, error) {
	if s.vault.Status() != vault.Unlocked {
		return &StatusResponse{}, nil
	}

	var account *Account
	var org *Org
	a, err := s.account(false)
	if err != nil {
		return nil, err
	}
	if a != nil {
		account = &Account{
			KID:   a.ID.String(),
			Email: a.Email,
		}
	}
	o, err := s.org(false)
	if err != nil {
		return nil, err
	}
	if o != nil {
		org = &Org{
			ID:     o.ID.String(),
			Domain: o.Org,
		}
	}

	resp := StatusResponse{
		Account: account,
		Org:     org,
	}
	return &resp, nil
}
