package service

import (
	"context"
	"sort"
	"strings"

	"github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

// Keys (RPC) ...
func (s *service) Keys(ctx context.Context, req *KeysRequest) (*KeysResponse, error) {
	query := strings.TrimSpace(req.Query)
	sortField := req.SortField
	if sortField == "" {
		sortField = "user"
	}
	sortDirection := req.SortDirection

	vks, err := s.vault.Keyring().Keys()
	if err != nil {
		return nil, err
	}

	out, err := s.filterKeys(ctx, vks, true, query, req.Types, sortField, sortDirection)
	if err != nil {
		return nil, err
	}

	return &KeysResponse{
		Keys:          out,
		SortField:     sortField,
		SortDirection: sortDirection,
	}, nil
}

func hasType(k *api.Key, types []string) bool {
	for _, t := range types {
		if k.Type == t {
			return true
		}
	}
	return false
}

func (s *service) filterKeys(ctx context.Context, ks []*api.Key, saved bool, query string, types []string, sortField string, sortDirection SortDirection) ([]*Key, error) {
	keys := make([]*Key, 0, len(ks))
	for _, k := range ks {
		if len(types) != 0 && !hasType(k, types) {
			continue
		}
		key, err := s.keyToRPC(ctx, k, saved)
		if err != nil {
			return nil, err
		}
		if query == "" || (key.User != nil && strings.HasPrefix(key.User.ID, query)) || strings.HasPrefix(key.ID, query) {
			keys = append(keys, key)
		}
	}

	switch sortField {
	case "kid", "user", "type":
	default:
		return nil, errors.Errorf("invalid sort field")
	}

	sort.Slice(keys, func(i, j int) bool {
		return keysSort(keys, sortField, sortDirection, i, j)
	})
	return keys, nil
}

func keysSort(pks []*Key, sortField string, sortDirection SortDirection, i, j int) bool {
	switch sortField {
	case "type":
		if pks[i].Type == pks[j].Type {
			return keysSort(pks, "user", sortDirection, i, j)
		}
		if sortDirection == SortDesc {
			return pks[i].Type < pks[j].Type
		}
		return pks[i].Type > pks[j].Type

	case "user":
		if pks[i].User == nil && pks[j].User == nil {
			return keysSort(pks, "kid", sortDirection, i, j)
		} else if pks[i].User == nil {
			return false
		} else if pks[j].User == nil {
			return true
		}
		if sortDirection == SortDesc {
			return pks[i].User.Name > pks[j].User.Name
		}
		return pks[i].User.Name <= pks[j].User.Name
	default:
		if sortDirection == SortDesc {
			return pks[i].ID > pks[j].ID
		}
		return pks[i].ID <= pks[j].ID
	}
}
