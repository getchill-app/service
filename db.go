package service

import (
	"context"
	"unicode/utf8"

	"github.com/davecgh/go-spew/spew"
	"github.com/keys-pub/keys/dstore"
	"github.com/keys-pub/keys/tsutil"
	"github.com/pkg/errors"
)

// Collections (RPC) ...
func (s *service) Collections(ctx context.Context, req *CollectionsRequest) (*CollectionsResponse, error) {
	switch req.DB {
	case "service":
		return s.serviceCollections(ctx, req.Parent)
	case "vault":
		return s.vaultCollections(ctx, req.Parent)
	default:
		return nil, errors.Errorf("invalid db %s", req.DB)
	}
}

func (s *service) serviceCollections(ctx context.Context, parent string) (*CollectionsResponse, error) {
	cols, err := s.db.Collections(ctx, parent)
	if err != nil {
		return nil, err
	}
	return &CollectionsResponse{Collections: collectionsToRPC(cols)}, nil
}

func (s *service) vaultCollections(ctx context.Context, parent string) (*CollectionsResponse, error) {
	out := []*Collection{
		{Path: "/config"},
		{Path: "/keys"},
		{Path: "/pull"},
		{Path: "/push"},
	}
	return &CollectionsResponse{Collections: out}, nil
}

func collectionsToRPC(cols []*dstore.Collection) []*Collection {
	out := make([]*Collection, 0, len(cols))
	for _, c := range cols {
		out = append(out, &Collection{Path: c.Path})
	}
	return out
}

// Documents (RPC) lists document from db or vault.
func (s *service) Documents(ctx context.Context, req *DocumentsRequest) (*DocumentsResponse, error) {
	var out []*Document
	var err error
	switch req.DB {
	case "service":
		out, err = s.dbService(ctx, req.Path)
	case "vault":
		logger.Debugf("Vault path %s", req.Path)
		switch req.Path {
		case "/keys":
			out, err = s.dbKeys()
		}
	default:
		return nil, errors.Errorf("unrecognized db")
	}

	if err != nil {
		return nil, err
	}

	return &DocumentsResponse{
		Documents: out,
	}, nil
}

func (s *service) dbService(ctx context.Context, path string) ([]*Document, error) {
	docs, err := s.db.Documents(ctx, path)
	if err != nil {
		return nil, err
	}

	dataToString := func(b []byte) string {
		var val string
		if !utf8.Valid(b) {
			val = spew.Sdump(b)
		} else {
			val = string(b)
		}
		return val
	}

	out := make([]*Document, 0, len(docs))
	for _, doc := range docs {
		out = append(out, &Document{
			Path:      doc.Path,
			Value:     dataToString(doc.Data()),
			CreatedAt: tsutil.Millis(doc.CreatedAt),
			UpdatedAt: tsutil.Millis(doc.UpdatedAt),
		})
	}
	return out, nil
}

func (s *service) dbKeys() ([]*Document, error) {
	keys, err := s.vault.Keyring().Keys()
	if err != nil {
		return nil, err
	}
	out := make([]*Document, 0, len(keys))
	for _, k := range keys {
		out = append(out, &Document{
			Path:      dstore.Path(k.ID),
			Value:     spew.Sdump(k),
			CreatedAt: k.CreatedAt,
			UpdatedAt: k.UpdatedAt,
		})
	}
	return out, nil
}
