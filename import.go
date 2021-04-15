package service

import (
	"context"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
)

// KeyImport (RPC) imports a key.
func (s *service) KeyImport(ctx context.Context, req *KeyImportRequest) (*KeyImportResponse, error) {
	key, err := api.ParseKey(req.In, req.Password)
	if err != nil {
		return nil, err
	}
	now := s.clock.NowMillis()
	if key.CreatedAt == 0 {
		key.CreatedAt = now
	}
	if key.UpdatedAt == 0 {
		key.UpdatedAt = now
	}
	if err := s.vault.Keyring().Save(key); err != nil {
		return nil, err
	}

	if req.Update {
		// TODO: Update UI to option to update key on import
		if _, err := s.updateUser(ctx, key.ID, false); err != nil {
			return nil, err
		}
	} else {
		if err := s.scs.Index(key.ID); err != nil {
			return nil, err
		}
	}

	return &KeyImportResponse{
		KID: key.ID.String(),
	}, nil
}

func (s *service) importID(id keys.ID) error {
	// Check if key already exists and skip if so.
	key, err := s.vault.Keyring().Get(id)
	if err != nil {
		return err
	}
	if key != nil {
		return nil
	}
	vk := api.NewKey(id)
	now := s.clock.NowMillis()
	vk.CreatedAt = now
	vk.UpdatedAt = now
	if err := s.vault.Keyring().Save(vk); err != nil {
		return err
	}
	if err := s.scs.Index(id); err != nil {
		return err
	}
	return nil
}
