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

// AuthSetup (RPC) ...
func (s *service) AuthSetup(ctx context.Context, req *AuthSetupRequest) (*AuthSetupResponse, error) {
	s.unlockMtx.Lock()
	defer s.unlockMtx.Unlock()

	logger.Infof("Auth setup...")
	if !s.vault.NeedsSetup() {
		return nil, errors.Errorf("already setup")
	}

	logger.Infof("Setup (%s)", req.Type)
	switch req.Type {
	case PasswordAuth:
		if _, err := s.vault.SetupPassword(req.Secret); err != nil {
			return nil, authErr(err, req.Type, "failed to setup")
		}
	case PaperKeyAuth:
		// TODO: Implement
		return nil, errors.Errorf("setup with paper key not supported")
	case FIDO2HMACSecretAuth:
		_, err := s.vault.GenerateFIDO2HMACSecret(ctx, req.Secret, req.Device, s.env.AppName())
		if err != nil {
			return nil, authErr(err, req.Type, "failed to setup")
		}
	default:
		return nil, errors.Errorf("unsupported auth type")
	}

	return &AuthSetupResponse{}, nil
}

func (s *service) unlock(ctx context.Context, req *AuthUnlockRequest) (*[32]byte, error) {
	switch req.Type {
	case PasswordAuth:
		mk, err := s.vault.UnlockWithPassword(req.Secret)
		if err != nil {
			return nil, authErr(err, req.Type, "failed to unlock")
		}
		return mk, nil
	case PaperKeyAuth:
		// if _, err := s.vault.UnlockWithPaperKey(req.Secret); err != nil {
		// 	return nil, authErr(err, req.Type, "failed to unlock")
		// }
		return nil, errors.Errorf("unlock with paper key not supported")
	case FIDO2HMACSecretAuth:
		mk, err := s.vault.UnlockWithFIDO2HMACSecret(ctx, req.Secret)
		if err != nil {
			return nil, authErr(err, req.Type, "failed to unlock")
		}
		return mk, nil
	default:
		return nil, errors.Errorf("unsupported auth type")
	}
}

// AuthUnlock (RPC) ...
func (s *service) AuthUnlock(ctx context.Context, req *AuthUnlockRequest) (*AuthUnlockResponse, error) {
	s.unlockMtx.Lock()
	defer s.unlockMtx.Unlock()

	mk, err := s.unlock(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.openDB(ctx, mk); err != nil {
		return nil, err
	}

	logger.Infof("Unlocked (%s)", req.Type)
	token := s.authIr.registerToken(req.Client)

	s.startCheck()

	return &AuthUnlockResponse{
		AuthToken: token,
	}, nil
}

func (s *service) openDB(ctx context.Context, mk *[32]byte) error {
	path, err := s.env.AppPath("service.db", true)
	if err != nil {
		return err
	}
	dbk := keys.Bytes32(keys.HKDFSHA256(mk[:], 32, nil, []byte("keys.pub/service.db")))
	if err := s.db.OpenAtPath(ctx, path, dbk); err != nil {
		if err == sqlcipher.ErrAlreadyOpen {
			return nil
		}
		return err
	}
	return nil
}

// AuthLock (RPC) ...
func (s *service) AuthLock(ctx context.Context, req *AuthLockRequest) (*AuthLockResponse, error) {
	s.unlockMtx.Lock()
	defer s.unlockMtx.Unlock()

	s.stopCheck()
	s.db.Close()
	s.authIr.clearTokens()
	if err := s.vault.Lock(); err != nil {
		return nil, err
	}
	return &AuthLockResponse{}, nil
}
