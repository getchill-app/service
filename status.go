package service

import (
	"context"

	"github.com/keys-pub/vault"
)

// Status (RPC) gets the current app status.
// This call is NOT AUTHENTICATED.
func (s *service) Status(ctx context.Context, req *StatusRequest) (*StatusResponse, error) {
	exe, exeErr := executablePath()
	if exeErr != nil {
		logger.Errorf("Failed to get current executable path: %s", exeErr)
	}
	status := s.vault.Status()

	resp := StatusResponse{
		Version:    s.build.Version,
		AppName:    s.env.AppName(),
		Exe:        exe,
		AuthStatus: vaultStatusToRPC(status),
		FIDO2:      s.vault.FIDO2Plugin() != nil,
	}
	logger.Infof("Status, %s", resp.String())
	return &resp, nil
}

func vaultStatusToRPC(st vault.Status) AuthStatus {
	switch st {
	case vault.Locked:
		return AuthLocked
	case vault.Unlocked:
		return AuthUnlocked
	case vault.SetupNeeded:
		return AuthSetupNeeded
	default:
		return AuthUnknown
	}
}
