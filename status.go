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
	currentAccount, err := s.currentAccount()
	if err != nil {
		return nil, err
	}
	if currentAccount != nil {
		account = &Account{
			KID:   currentAccount.ID.String(),
			Email: currentAccount.Email,
		}
	}
	currentOrg, err := s.currentOrg()
	if err != nil {
		return nil, err
	}
	if currentOrg != nil {
		org = &Org{
			KID:    currentOrg.ID.String(),
			Domain: currentOrg.Org,
		}
	}

	resp := StatusResponse{
		Account: account,
		Org:     org,
	}
	logger.Infof("Status, %s", resp.String())
	return &resp, nil
}
