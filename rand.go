package service

import (
	"context"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/encoding"
	"github.com/pkg/errors"
)

// Rand (RPC) ...
func (s *service) Rand(ctx context.Context, req *RandRequest) (*RandResponse, error) {
	b := keys.RandBytes(int(req.NumBytes))

	enc, err := encodingFromRPC(req.Encoding)
	if err != nil {
		return nil, err
	}

	opts := []encoding.EncodeOption{}
	if req.NoPadding {
		opts = append(opts, encoding.NoPadding())
	}
	if req.Lowercase {
		opts = append(opts, encoding.Lowercase())
	}

	out, err := encoding.Encode(b, enc, opts...)
	if err != nil {
		return nil, err
	}

	return &RandResponse{
		Data: out,
	}, nil
}

func (s *service) RandPassword(ctx context.Context, req *RandPasswordRequest) (*RandPasswordResponse, error) {
	length := int(req.Length)
	if length < 16 {
		return nil, errors.Errorf("invalid length")
	}
	password := keys.RandPassword(length)
	return &RandPasswordResponse{
		Password: password,
	}, nil
}
