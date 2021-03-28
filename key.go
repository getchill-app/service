package service

import (
	"context"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/keys-pub/keys/tsutil"
	"github.com/pkg/errors"
)

// Key (RPC) ...
func (s *service) Key(ctx context.Context, req *KeyRequest) (*KeyResponse, error) {
	kid, err := s.lookup(ctx, req.Key, &lookupOpts{SearchRemote: req.Search})
	if err != nil {
		return nil, err
	}

	if req.Update {
		if _, err := s.updateUser(ctx, kid, false); err != nil {
			return nil, err
		}
	} else {
		if err := s.checkForExpiredKey(ctx, kid); err != nil {
			return nil, err
		}
	}

	key, err := s.key(ctx, kid)
	if err != nil {
		return nil, err
	}

	return &KeyResponse{
		Key: key,
	}, nil
}

func (s *service) keyToRPC(ctx context.Context, key *api.Key, saved bool) (*Key, error) {
	if key == nil {
		return nil, nil
	}
	out := &Key{
		ID:        key.ID.String(),
		Type:      key.Type,
		Saved:     saved,
		IsPrivate: len(key.Private) > 0,
	}

	if err := s.fillKey(ctx, key.ID, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *service) key(ctx context.Context, kid keys.ID) (*Key, error) {
	key, err := s.vault.Keyring().Key(kid)
	if err != nil {
		return nil, err
	}
	if key != nil {
		return s.keyToRPC(ctx, key, true)
	}
	return s.keyToRPC(ctx, api.NewKey(kid), false)
}

func (s *service) fillKey(ctx context.Context, kid keys.ID, key *Key) error {
	res, err := s.users.Get(ctx, kid)
	if err != nil {
		return err
	}
	key.User = userResultToRPC(res)

	// Sigchain info
	sc, err := s.scs.Sigchain(kid)
	if err != nil {
		return err
	}
	if sc != nil {
		key.SigchainLength = int32(sc.Length())
		last := sc.Last()
		if last != nil {
			key.SigchainUpdatedAt = tsutil.Millis(last.Timestamp)
		}
	}
	return nil
}

// KeyRemove (RPC) removes a key.
func (s *service) KeyRemove(ctx context.Context, req *KeyRemoveRequest) (*KeyRemoveResponse, error) {
	if req.KID == "" {
		return nil, errors.Errorf("kid not specified")
	}
	kid, err := keys.ParseID(req.KID)
	if err != nil {
		return nil, err
	}
	key, err := s.vault.Keyring().Key(kid)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, keys.NewErrNotFound(kid.String())
	}

	if err := s.vault.Keyring().Remove(kid); err != nil {
		return nil, err
	}

	if kid.IsEdX25519() {
		_, err = s.scs.Delete(kid)
		if err != nil {
			return nil, err
		}
		if _, err := s.users.Update(ctx, kid); err != nil {
			return nil, err
		}
	}

	return &KeyRemoveResponse{}, nil
}

// KeyGenerate (RPC) creates a key.
func (s *service) KeyGenerate(ctx context.Context, req *KeyGenerateRequest) (*KeyGenerateResponse, error) {
	if req.Type == "" {
		return nil, errors.Errorf("no key type specified")
	}
	var key keys.Key
	switch req.Type {
	case string(keys.EdX25519):
		key = keys.GenerateEdX25519Key()
	case string(keys.X25519):
		key = keys.GenerateX25519Key()
	default:
		return nil, errors.Errorf("unknown key type %s", req.Type)
	}
	vk := api.NewKey(key)
	now := s.clock.NowMillis()
	vk.CreatedAt = now
	vk.UpdatedAt = now
	if err := s.vault.Keyring().Set(vk); err != nil {
		return nil, err
	}
	if err := s.scs.Index(vk.ID); err != nil {
		return nil, err
	}

	return &KeyGenerateResponse{
		KID: vk.ID.String(),
	}, nil
}

// KeySearch (RPC) ...
func (s *service) KeySearch(ctx context.Context, req *KeySearchRequest) (*KeySearchResponse, error) {
	res, err := s.searchUsersRemote(ctx, req.Query, 0)
	if err != nil {
		return nil, err
	}
	keys := make([]*Key, 0, len(res))
	for _, u := range res {
		kid := u.KID
		typ := string(kid.Type())
		key := &Key{
			ID:   kid.String(),
			User: apiUserToRPC(u),
			Type: typ,
		}
		keys = append(keys, key)
	}

	return &KeySearchResponse{
		Keys: keys,
	}, nil
}
