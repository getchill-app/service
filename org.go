package service

import (
	"context"

	hapi "github.com/getchill-app/http/api"
	"github.com/getchill-app/http/client"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

func (s *service) OrgKey(ctx context.Context, req *OrgKeyRequest) (*OrgKeyResponse, error) {
	key, err := s.vault.Keyring().KeyWithLabel(req.Domain)
	if err != nil {
		return nil, err
	}

	created := false
	if key == nil {
		key = api.NewKey(keys.GenerateEdX25519Key()).WithLabels("org", req.Domain).Created(s.clock.NowMillis())
		if err := s.vault.Keyring().Save(key); err != nil {
			return nil, err
		}
		created = true
	}

	if !key.HasLabel("org") {
		return nil, errors.Errorf("key missing org label")
	}
	verified := false
	if key.HasLabel("verified") {
		verified = true
	}

	return &OrgKeyResponse{KID: key.ID.String(), Created: created, Verified: verified}, nil
}

func (s *service) OrgSign(ctx context.Context, req *OrgSignRequest) (*OrgSignResponse, error) {
	key, err := s.vault.Keyring().KeyWithLabel(req.Domain)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, errors.Errorf("no org key setup")
	}
	if !key.IsEdX25519() {
		return nil, errors.Errorf("invalid key")
	}

	sig, err := s.client.OrgSign(key.AsEdX25519(), req.Domain, s.clock.Now())
	if err != nil {
		return nil, err
	}
	return &OrgSignResponse{Sig: sig}, nil
}

func (s *service) OrgCreate(ctx context.Context, req *OrgCreateRequest) (*OrgCreateResponse, error) {
	key, err := s.vault.Keyring().KeyWithLabel(req.Domain)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, errors.Errorf("no org key setup")
	}
	if !key.IsEdX25519() {
		return nil, errors.Errorf("invalid key")
	}
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}
	if err := s.client.OrgCreate(ctx, key.AsEdX25519(), req.Domain, account.AsEdX25519()); err != nil {
		return nil, err
	}

	if err := s.vault.Keyring().Save(key); err != nil {
		return nil, err
	}

	return &OrgCreateResponse{}, nil
}

func (s *service) OrgInvites(ctx context.Context, req *OrgInvitesRequest) (*OrgInvitesResponse, error) {
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}
	invites, err := s.client.OrgAccountInvites(ctx, account.AsEdX25519())
	if err != nil {
		return nil, err
	}
	return &OrgInvitesResponse{
		Invites: invitesToRPC(invites),
	}, nil
}

func invitesToRPC(invites []*hapi.OrgInvite) []*OrgInvite {
	out := make([]*OrgInvite, 0, len(invites))
	for _, invite := range invites {
		out = append(out, &OrgInvite{
			Org: &Org{
				ID:     invite.Org.String(),
				Domain: invite.Domain,
			},
			InvitedBy: invite.InvitedBy.String(),
		})
	}
	return out
}

func (s *service) OrgInviteAccept(ctx context.Context, req *OrgInviteAcceptRequest) (*OrgInviteAcceptResponse, error) {
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}

	existing, err := s.org(false)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.Errorf("org already set")
	}

	oid, err := keys.ParseID(req.ID)
	if err != nil {
		return nil, err
	}

	invite, err := s.client.OrgAccountInvite(ctx, account.AsEdX25519(), oid)
	if err != nil {
		return nil, err
	}
	if invite == nil {
		return nil, errors.Errorf("invite not found")
	}

	orgKey, err := client.DecryptKey(invite.EncryptedKey, account.AsEdX25519())
	if err != nil {
		return nil, err
	}

	key := api.NewKey(orgKey).WithLabels("org", invite.Domain).Created(s.clock.NowMillis())
	if err := s.vault.Keyring().Save(key); err != nil {
		return nil, err
	}

	if err := s.client.OrgInviteAccept(ctx, account.AsEdX25519(), orgKey); err != nil {
		return nil, err
	}
	return &OrgInviteAcceptResponse{}, nil
}
