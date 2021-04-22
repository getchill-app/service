package service

import (
	"context"

	"github.com/getchill-app/http/api"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/dstore"
)

func (s *service) userName(ctx context.Context, kid keys.ID) (string, error) {
	path := dstore.Path("ausers", kid)
	doc, err := s.db.Get(ctx, path)
	if err != nil {
		return "", err
	}
	if doc == nil {
		account, err := s.account(true)
		if err != nil {
			return "", err
		}
		usr, err := s.client.UserLookup(ctx, "kid", kid.String(), account.AsEdX25519())
		if err != nil {
			return "", nil
		}
		if usr != nil {
			if err := s.db.Set(ctx, path, dstore.From(usr)); err != nil {
				return "", nil
			}
		}
		return usr.Username, nil
	}
	var usr api.User
	if err := doc.To(&usr); err != nil {
		return "", err
	}
	return usr.Username, nil
}
