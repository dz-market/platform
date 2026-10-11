package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dz-market/platform/config"
)

type result struct {
	data map[string]any
	err  error
}

type Resolver struct {
	client *Client
	cache  map[string]result
}

func NewResolver(client *Client) *Resolver {
	return &Resolver{
		client: client,
		cache:  make(map[string]result),
	}
}

func (r *Resolver) Resolve(ctx context.Context, ref string) (string, error) {
	path, field, ok := strings.Cut(ref, "#")
	if !ok || path == "" || field == "" {
		return "", fmt.Errorf("%w: want path#field", config.ErrInvalidReference)
	}

	res := r.read(ctx, path)
	if errors.Is(res.err, ErrNotFound) {
		return "", config.ErrNotFound
	}

	if res.err != nil {
		return "", res.err
	}

	return lookup(res.data, field)
}

func (r *Resolver) read(ctx context.Context, path string) result {
	res, ok := r.cache[path]
	if !ok {
		data, err := r.client.Read(ctx, path)
		res = result{
			data: data,
			err:  err,
		}
		r.cache[path] = res
	}

	return res
}

func lookup(data map[string]any, field string) (string, error) {
	var value any = data

	for key := range strings.SplitSeq(field, ".") {
		node, ok := value.(map[string]any)
		if !ok {
			return "", config.ErrNotFound
		}

		if value, ok = node[key]; !ok {
			return "", config.ErrNotFound
		}
	}

	switch v := value.(type) {
	case nil:
		return "", config.ErrNotFound

	case string:
		return v, nil

	case json.Number, bool:
		return fmt.Sprint(v), nil

	default:
		return "", fmt.Errorf("%w: %s is not a value", config.ErrInvalidReference, field)
	}
}
