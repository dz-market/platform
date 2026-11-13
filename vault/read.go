package vault

import (
	"context"
	"fmt"
)

func (c *Client) Read(ctx context.Context, path string) (map[string]any, error) {
	secret, err := c.api.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	if secret == nil {
		return nil, fmt.Errorf("read %q: %w", path, ErrPathNotFound)
	}

	if secret.Data == nil {
		return nil, fmt.Errorf("read %q: %w", path, ErrNoData)
	}

	return secret.Data, nil
}
