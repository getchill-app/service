package service

import (
	"context"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/user"
	"github.com/keys-pub/keys/user/services"
	"github.com/keys-pub/keys/users"
	"github.com/pkg/errors"
)

func (s *service) publicKeys() ([]keys.ID, error) {
	pks, err := s.vault.Keyring().KeysByType(string(keys.EdX25519))
	if err != nil {
		return nil, err
	}
	out := []keys.ID{}
	for _, pk := range pks {
		if len(pk.Private) > 0 {
			continue
		}
		out = append(out, pk.ID)
	}
	return out, nil
}

func (s *service) checkKeys(ctx context.Context) error {
	logger.Infof("Checking keys...")
	pks, err := s.publicKeys()
	if err != nil {
		return errors.Wrapf(err, "failed to list public keys")
	}
	for _, pk := range pks {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.checkForExpiredKey(ctx, pk); err != nil {
			return err
		}
	}
	return nil
}

// Check if expired, and then update.
// If we don't have a local result, we don't update.
func (s *service) checkForExpiredKey(ctx context.Context, kid keys.ID) error {
	res, err := s.users.Get(ctx, kid)
	if err != nil {
		return err
	}
	if res != nil {
		// If not OK, check every "userCheckFailureExpire", otherwise check every "userCheckExpire".
		now := s.clock.Now()
		if (res.Status != user.StatusOK && res.IsTimestampExpired(now, userCheckFailureExpire)) ||
			res.IsTimestampExpired(now, userCheckExpire) {
			_, err := s.updateUser(ctx, kid, true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) updateUser(ctx context.Context, kid keys.ID, allowProxyCache bool) (*user.Result, error) {
	logger.Infof("Update user %s", kid)

	// TODO: Only get new sigchain entries.
	resp, err := s.client.Sigchain(ctx, kid)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		// TODO: Check that our existing statements haven't changed or disappeared
		logger.Infof("Received sigchain %s, len=%d", kid, len(resp.Statements))

		sc := keys.NewSigchain(kid)
		if err := sc.AddAll(resp.Statements); err != nil {
			return nil, err
		}
		if err := s.scs.Save(sc); err != nil {
			return nil, err
		}
	} else {
		logger.Infof("No sigchain for %s", kid)
	}

	if err := s.scs.Index(kid); err != nil {
		return nil, err
	}

	service := func(usr *user.User) services.Service {
		switch usr.Service {
		case "twitter":
			if allowProxyCache {
				return services.KeysPub
			}
			return services.Proxy
		}
		return nil
	}

	res, err := s.users.Update(ctx, kid, users.UseService(service))
	if err != nil {
		return nil, err
	}

	return res, nil
}

func twitterProxy(usr *user.User) services.Service {
	switch usr.Service {
	case "twitter":
		return services.Proxy
	}
	return nil
}
