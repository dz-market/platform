package config

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

const (
	defaultConfigPath = "configs/config.yml"
	defaultEnvPath    = ".env"
)

type lookupFunc func(string) (string, bool)

type Option func(*loader)

type loader struct {
	configPath string
	envPath    string
	lookup     lookupFunc
}

func WithPath(path string) Option {
	return func(l *loader) {
		l.configPath = path
	}
}

func WithEnvPath(envPath string) Option {
	return func(l *loader) {
		l.envPath = envPath
	}
}

func withLookup(fn lookupFunc) Option {
	return func(l *loader) {
		l.lookup = fn
	}
}

func Load[T any](opts ...Option) (cfg T, err error) {
	cfg, err = load[T](opts...)
	if err != nil {
		return cfg, err
	}

	if err := validate(cfg); err != nil {
		return cfg, fmt.Errorf("%w: %w", ErrValidate, err)
	}

	return cfg, nil
}

func load[T any](opts ...Option) (cfg T, err error) {
	l := loader{
		configPath: defaultConfigPath,
		envPath:    defaultEnvPath,
	}

	for _, opt := range opts {
		opt(&l)
	}

	if l.lookup == nil {
		lookup, err := envLookup(l.envPath)
		if err != nil {
			return cfg, fmt.Errorf("load env: %w", err)
		}

		l.lookup = lookup
	}

	raw, err := os.ReadFile(l.configPath)
	if err != nil {
		return cfg, fmt.Errorf("%w: %w", ErrRead, err)
	}

	var tree map[string]any
	if err := yaml.Unmarshal(raw, &tree); err != nil {
		return cfg, fmt.Errorf("%w: %w", ErrParse, err)
	}

	var missing []string

	expandNode(tree, l.lookup, &missing)

	if len(missing) > 0 {
		slices.Sort(missing)
		missing = slices.Compact(missing)

		return cfg, fmt.Errorf("%w: %s", ErrUnresolved, strings.Join(missing, ", "))
	}

	return decode[T](tree)
}
