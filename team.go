package service

import (
	"context"

	hapi "github.com/getchill-app/http/api"
)

func (s *service) TeamInvites(ctx context.Context, req *TeamInvitesRequest) (*TeamInvitesResponse, error) {
	account, err := s.account(true)
	if err != nil {
		return nil, err
	}
	invites, err := s.client.TeamAccountInvites(ctx, account.AsEdX25519())
	if err != nil {
		return nil, err
	}
	return &TeamInvitesResponse{
		Invites: invitesToRPC(invites),
	}, nil
}

func invitesToRPC(invites []*hapi.TeamInvite) []*TeamInvite {
	out := make([]*TeamInvite, 0, len(invites))
	for _, invite := range invites {
		out = append(out, &TeamInvite{
			Team: &Team{
				ID:     invite.Team.String(),
				Domain: invite.Domain,
			},
			InvitedBy: invite.InvitedBy.String(),
		})
	}
	return out
}
