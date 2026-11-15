package vault

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/dz-market/platform/config"
	pvault "github.com/dz-market/platform/vault"
)

var _ config.Resolver = (*Resolver)(nil)

type Resolver struct {
	client *pvault.Client
}

func New(client *pvault.Client) *Resolver {
	return &Resolver{
		client: client,
	}
}

func (r Resolver) Resolve(ctx context.Context, ref string) (any, error) {
	path, fields, err := parseReference(ref)
	if err != nil {
		return nil, err
	}

	data, err := r.client.Read(ctx, path)
	if errors.Is(err, pvault.ErrPathNotFound) {
		return nil, fmt.Errorf("%w: vault path %q", config.ErrNotFound, path)
	}

	if err != nil {
		return nil, err
	}

	return lookup(data, fields)
}

func parseReference(ref string) (string, []string, error) {
	path, field, ok := strings.Cut(ref, "#")
	if !ok || path == "" || field == "" || strings.Contains(field, "#") {
		return "", nil, fmt.Errorf("%w: expected path#field", config.ErrInvalidReference)
	}

	fields := strings.Split(field, ".")

	if slices.Contains(fields, "") {
		return "", nil, fmt.Errorf("%w: empty field segment", config.ErrInvalidReference)
	}

	return path, fields, nil
}

func lookup(data map[string]any, fields []string) (any, error) {
	var value any = data

	for _, key := range fields {
		node, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: field parent is not an object", config.ErrInvalidReference)
		}

		value, ok = node[key]
		if !ok {
			return nil, fmt.Errorf("%w: field %q", config.ErrNotFound, strings.Join(fields, "."))
		}
	}

	return value, nil
}
