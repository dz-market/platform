package git

import (
	"context"
	"fmt"
	"strings"
)

func Tags(ctx context.Context, root string) ([]string, error) {
	output, err := Run(ctx, root, "for-each-ref", "--format=%(refname:strip=2)", "refs/tags/")
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	return strings.Fields(output), nil
}

func Tag(ctx context.Context, root, name, commit string) error {
	if _, err := Run(ctx, root, "tag", "--", name, commit); err != nil {
		return fmt.Errorf("create tag %q: %w", name, err)
	}

	return nil
}

func PushTags(ctx context.Context, root string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	args := []string{"push", "--atomic", "origin"}

	for _, tag := range tags {
		ref := "refs/tags/" + tag
		args = append(args, ref+":"+ref)
	}

	if _, err := Run(ctx, root, args...); err != nil {
		return fmt.Errorf("push tags: %w", err)
	}

	return nil
}
