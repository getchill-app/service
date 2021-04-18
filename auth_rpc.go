package service

import (
	"context"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys-ext/sqlcipher"
	"github.com/keys-pub/vault"
	"github.com/pkg/errors"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// ErrInvalidPassword if invalid password.
var ErrInvalidPassword = status.Error(codes.Unauthenticated, "invalid password")

// ErrInvalidAuth if invalid auth.
var ErrInvalidAuth = status.Error(codes.Unauthenticated, "invalid auth")

func authErr(err error, typ AuthType, wrap string) error {
	if errors.Cause(err) == vault.ErrInvalidAuth {
		switch typ {
		case PasswordAuth:
			return ErrInvalidPassword
		default:
			return ErrInvalidAuth
		}

	}
	return errors.Wrapf(err, wrap)
}

func (s *service) AuthUnlock(ctx context.Context, req *AuthUnlockRequest) (*AuthUnlockResponse, error) {
	token, _, err := s.authUnlock(ctx, req.Secret, req.Type, req.Client)
	if err != nil {
		return nil, err
	}
	return &AuthUnlockResponse{AuthToken: token}, nil
}

func (s *service) authUnlock(ctx context.Context, secret string, typ AuthType, client string) (string, *[32]byte, error) {
	s.unlockMtx.Lock()
	defer s.unlockMtx.Unlock()

	mk, err := s.unlock(ctx, secret, typ)
	if err != nil {
		return "", nil, err
	}
	if err := s.openDB(ctx, mk); err != nil {
		return "", nil, err
	}

	logger.Infof("Unlocked (%s)", typ)
	token := s.authIr.registerToken(client)

	s.startCheck()

	return token, mk, nil
}

func (s *service) unlock(ctx context.Context, secret string, typ AuthType) (*[32]byte, error) {
	switch typ {
	case PasswordAuth:
		mk, err := s.vault.UnlockWithPassword(secret)
		if err != nil {
			return nil, authErr(err, typ, "failed to unlock")
		}
		return mk, nil
	case PaperKeyAuth:
		mk, err := s.vault.UnlockWithPaperKey(secret)
		if err != nil {
			return nil, authErr(err, typ, "failed to unlock")
		}
		return mk, nil
	case FIDO2HMACSecretAuth:
		mk, err := s.vault.UnlockWithFIDO2HMACSecret(ctx, secret)
		if err != nil {
			return nil, authErr(err, typ, "failed to unlock")
		}
		return mk, nil
	default:
		return nil, errors.Errorf("unsupported auth type")
	}
}

func (s *service) openDB(ctx context.Context, mk *[32]byte) error {
	path, err := s.env.AppPath("service.db", true)
	if err != nil {
		return err
	}
	dbk := keys.Bytes32(keys.HKDFSHA256(mk[:], 32, nil, []byte("getchill.app/service.db")))
	if err := s.db.OpenAtPath(ctx, path, dbk); err != nil {
		if err == sqlcipher.ErrAlreadyOpen {
			return nil
		}
		return err
	}
	return nil
}

func (s *service) AuthLock(ctx context.Context, req *AuthLockRequest) (*AuthLockResponse, error) {
	if err := s.authLock(ctx); err != nil {
		return nil, err
	}
	return &AuthLockResponse{}, nil
}

func (s *service) authLock(ctx context.Context) error {
	s.unlockMtx.Lock()
	defer s.unlockMtx.Unlock()
	logger.Infof("Locking...")

	s.stopCheck()
	s.db.Close()
	s.authIr.clearTokens()
	if err := s.vault.Lock(); err != nil {
		return err
	}
	return nil
}
