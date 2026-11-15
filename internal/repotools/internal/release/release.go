package release

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dz-market/platform/internal/repotools/internal/git"
)

func Run(ctx context.Context, root, ref string) error {
	if ref == "" {
		return errors.New("release ref is required")
	}

	root, err := git.Run(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}

	commit, err := git.Run(ctx, root, "rev-parse", "--verify", "--end-of-options", fmt.Sprintf("%s^{commit}", ref))
	if err != nil {
		return fmt.Errorf("resolve release ref: %w", err)
	}

	return releaseCommit(ctx, root, commit)
}

func releaseCommit(ctx context.Context, root, commit string) error {
	data, err := git.Run(ctx, root, "show", commit+":versions.yml")
	if err != nil {
		return fmt.Errorf("read versions.yml: %w", err)
	}

	versions, err := readManifest(data)
	if err != nil {
		return err
	}

	tags, err := moduleTags(ctx, root, commit, versions)
	if err != nil {
		return err
	}

	return publishMissingTags(ctx, root, commit, tags)
}

func publishMissingTags(ctx context.Context, root, commit string, tags []string) error {
	pending, err := missingTags(ctx, root, tags)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		slog.Info("No new module tags to create")

		return nil
	}

	if err := createTags(ctx, root, commit, pending); err != nil {
		return err
	}

	if err := git.PushTags(ctx, root, pending); err != nil {
		return err
	}

	slog.Info("Published module tags", "count", len(pending))

	return nil
}

func missingTags(ctx context.Context, root string, tags []string) ([]string, error) {
	existingTags, err := git.Tags(ctx, root)
	if err != nil {
		return nil, err
	}

	existing := make(map[string]bool, len(existingTags))

	for _, tag := range existingTags {
		existing[tag] = true
	}

	return filterMissingTags(tags, existing), nil
}

func filterMissingTags(tags []string, existing map[string]bool) []string {
	pending := make([]string, 0, len(tags))

	for _, tag := range tags {
		if existing[tag] {
			slog.Info("Skip existing tag", "tag", tag)

			continue
		}

		pending = append(pending, tag)
	}

	return pending
}

func createTags(ctx context.Context, root, commit string, tags []string) error {
	for _, tag := range tags {
		if err := git.Tag(ctx, root, tag, commit); err != nil {
			return err
		}

		slog.Info("Created module tag", "tag", tag, "commit", commit)
	}

	return nil
}
