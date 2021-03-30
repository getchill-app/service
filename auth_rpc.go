package service

import (
	"context"
	"os"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys-ext/sqlcipher"
	"github.com/keys-pub/keys/api"
	"github.com/keys-pub/keys/tsutil"
	"github.com/keys-pub/vault"
	"github.com/keys-pub/vault/auth"
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
	if s.vault.Status() != vault.SetupNeeded {
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

	if err := s.setupAccount(); err != nil {
		return nil, err
	}

	logger.Infof("Unlocked (%s)", req.Type)
	token := s.authIr.registerToken(req.Client)

	s.startCheck()

	return &AuthUnlockResponse{
		AuthToken: token,
	}, nil
}

func (s *service) setupAccount() error {
	// TODO: Implement
	currentUser, err := s.currentUserID()
	if err != nil {
		return err
	}
	if currentUser == "" {
		user := api.NewKey(keys.GenerateEdX25519Key()).Created(s.clock.NowMillis())
		if err := s.vault.Keyring().Set(user); err != nil {
			return err
		}
		if err := s.setCurrentUser(user.ID); err != nil {
			return err
		}
	}
	return nil
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

func (s *service) AuthReset(ctx context.Context, req *AuthResetRequest) (*AuthResetResponse, error) {
	if s.vault.Status() != vault.Locked {
		return nil, errors.Wrapf(errors.Errorf("auth is not locked"), "failed to reset")
	}

	if req.AppName != s.env.AppName() {
		return nil, errors.Wrapf(errors.Errorf("invalid app name"), "failed to reset")
	}

	if err := s.vault.Reset(); err != nil {
		return nil, err
	}

	path, err := s.env.AppPath("service.db", false)
	if err != nil {
		return nil, err
	}
	if err := os.RemoveAll(path); err != nil {
		return nil, err
	}

	return &AuthResetResponse{}, nil
}

// AuthProvision (RPC) ...
func (s *service) AuthProvision(ctx context.Context, req *AuthProvisionRequest) (*AuthProvisionResponse, error) {
	var auth *auth.Auth
	var err error
	switch req.Type {
	case PasswordAuth:
		auth, err = s.vault.RegisterPassword(req.Secret)
	default:
		return nil, errors.Errorf("unsupported provision type")
	}
	if err != nil {
		return nil, err
	}
	return &AuthProvisionResponse{
		Provision: authToRPC(auth),
	}, nil
}

// AuthDeprovision (RPC) ...
func (s *service) AuthDeprovision(ctx context.Context, req *AuthDeprovisionRequest) (*AuthDeprovisionResponse, error) {
	// TODO: If FIDO2 resident key and supports credMgmt remove from the device also?
	return nil, errors.Errorf("no implemented")
}

// AuthProvisions (RPC) ...
func (s *service) AuthProvisions(ctx context.Context, req *AuthProvisionsRequest) (*AuthProvisionsResponse, error) {
	auths, err := s.vault.Auth().List()
	if err != nil {
		return nil, err
	}

	out := make([]*AuthProvision, 0, len(auths))
	for _, auth := range auths {
		out = append(out, authToRPC(auth))
	}

	return &AuthProvisionsResponse{
		Provisions: out,
	}, nil
}

// AuthPasswordChange (RPC) ...
func (s *service) AuthPasswordChange(ctx context.Context, req *AuthPasswordChangeRequest) (*AuthPasswordChangeResponse, error) {
	return nil, errors.Errorf("no implemented")
}

func authToRPC(auth *auth.Auth) *AuthProvision {
	return &AuthProvision{
		ID:        auth.ID,
		Type:      authTypeToRPC(auth.Type),
		AAGUID:    auth.AAGUID,
		NoPin:     auth.NoPin,
		CreatedAt: tsutil.Millis(auth.CreatedAt),
	}
}

func authTypeToRPC(t auth.Type) AuthType {
	switch t {
	case auth.PasswordType:
		return PasswordAuth
	case auth.PaperKeyType:
		return PaperKeyAuth
	case auth.FIDO2HMACSecretType:
		return FIDO2HMACSecretAuth
	default:
		return UnknownAuth
	}
}
