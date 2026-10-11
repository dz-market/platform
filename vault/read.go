package vault

import (
	"context"
	"fmt"
)

func (c *Client) Read(ctx context.Context, path string) (map[string]any, error) {
	secret, err := c.api.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("read %s: %w", path, ErrNotFound)
	}

	return secret.Data, nil
}
