package release

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"

	"github.com/dz-market/platform/internal/repotools/internal/git"
)

var moduleDirPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*(/[a-z][a-z0-9_-]*)*$`)

type manifest struct {
	Modules map[string]string `yaml:"modules"`
}

func readManifest(data string) (manifest, error) {
	var result manifest

	if err := yaml.UnmarshalWithOptions([]byte(data), &result, yaml.Strict()); err != nil {
		return manifest{}, fmt.Errorf("decode versions.yml: %w", err)
	}

	if len(result.Modules) == 0 {
		return manifest{}, fmt.Errorf("versions.yml: modules must not be empty")
	}

	return result, nil
}

func moduleTags(ctx context.Context, root, commit string, versions manifest) ([]string, error) {
	dirs := slices.Sorted(maps.Keys(versions.Modules))
	tags := make([]string, 0, len(dirs))

	for _, dir := range dirs {
		tag, err := moduleTag(ctx, root, commit, dir, versions.Modules[dir])
		if err != nil {
			return nil, err
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

func moduleTag(ctx context.Context, root, commit, dir, version string) (string, error) {
	if !moduleDirPattern.MatchString(dir) {
		return "", fmt.Errorf("invalid module directory %q", dir)
	}

	if !semver.IsValid(version) || semver.Canonical(version) != version {
		return "", fmt.Errorf("invalid canonical version for %s: %q", dir, version)
	}

	data, err := git.Run(ctx, root, "show", fmt.Sprintf("%s:%s/go.mod", commit, dir))
	if err != nil {
		return "", fmt.Errorf("read %s/go.mod: %w", dir, err)
	}

	if err := module.Check(modfile.ModulePath([]byte(data)), version); err != nil {
		return "", fmt.Errorf("validate %s at %s: %w", dir, version, err)
	}

	return moduleTagPrefix(dir, version) + "/" + version, nil
}

func moduleTagPrefix(dir, version string) string {
	major := semver.Major(version)

	if major != "v0" && major != "v1" {
		return strings.TrimSuffix(dir, "/"+major)
	}

	return dir
}
