package service

import (
	"context"
	"time"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/user"
	"github.com/pkg/errors"
)

func (s *service) startCheck() {
	s.checkMtx.Lock()
	defer s.checkMtx.Unlock()

	if s.checking {
		return
	}
	logger.Debugf("Start check...")
	ticker := time.NewTicker(time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	s.checkCancelFn = cancel
	s.checking = true

	go func() {
		s.tryCheck(ctx)
		for {
			select {
			case <-ticker.C:
				s.tryCheck(ctx)
			case <-ctx.Done():
				logger.Debugf("Check canceled")
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *service) stopCheck() {
	s.checkMtx.Lock()
	defer s.checkMtx.Unlock()

	logger.Debugf("Stop check...")
	s.checking = false
	s.checkCancelFn()
	// We should give it little bit of time to finish checking after the cancel
	// otherwise it might error trying to write to a closed database.
	// This wait isn't strictly required but we do it to be nice.
	// TODO: Use a WaitGroup with a timeout or channel
	for i := 0; i < 100; i++ {
		if !s.checking {
			logger.Debugf("Check stopped")
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
	logger.Debugf("Timed out waiting for stop check")
}

func (s *service) tryCheck(ctx context.Context) {
	s.checkMtx.Lock()
	defer s.checkMtx.Unlock()

	if err := s.checkKeys(ctx); err != nil {
		logger.Warningf("Failed to check keys: %v", err)
	}
}

func (s *service) edxPublicKeys() ([]keys.ID, error) {
	pks, err := s.vault.Keyring().KeysByType(string(keys.EdX25519))
	if err != nil {
		return nil, err
	}
	out := []keys.ID{}
	for _, pk := range pks {
		out = append(out, pk.ID)
	}
	return out, nil
}

func (s *service) checkKeys(ctx context.Context) error {
	logger.Infof("Checking keys...")
	pks, err := s.edxPublicKeys()
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
