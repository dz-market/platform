package config

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/goccy/go-yaml"
)

const (
	defaultConfigPath = "configs/config.yml"
	defaultEnvPath    = ".env"
)

var schemePattern = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

type lookupFunc func(string) (string, bool)

type Option func(*loader)

type loader struct {
	configPath string
	envPath    string
	resolvers  map[string]Resolver
	err        error
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

func WithResolver(scheme string, r Resolver) Option {
	return func(l *loader) {
		if !schemePattern.MatchString(scheme) {
			l.err = fmt.Errorf("resolver scheme %q: must match %s", scheme, schemePattern)

			return
		}

		l.resolvers[scheme] = r
	}
}

func Load[T any](ctx context.Context, opts ...Option) (cfg T, err error) {
	cfg, err = load[T](ctx, opts...)
	if err != nil {
		return cfg, err
	}

	if err := validate(cfg); err != nil {
		return cfg, fmt.Errorf("%w: %w", ErrValidation, err)
	}

	return cfg, nil
}

func load[T any](ctx context.Context, opts ...Option) (cfg T, err error) {
	l := loader{
		configPath: defaultConfigPath,
		envPath:    defaultEnvPath,
		resolvers:  make(map[string]Resolver),
	}

	for _, opt := range opts {
		opt(&l)
	}

	if l.err != nil {
		return cfg, l.err
	}

	if _, ok := l.resolvers[schemeEnv]; !ok {
		lookup, err := envLookup(l.envPath)
		if err != nil {
			return cfg, fmt.Errorf("load env: %w", err)
		}

		l.resolvers[schemeEnv] = envResolver(lookup)
	}

	raw, err := os.ReadFile(l.configPath)
	if err != nil {
		return cfg, fmt.Errorf("%w: %w", ErrReadFile, err)
	}

	var tree map[string]any
	if err := yaml.Unmarshal(raw, &tree); err != nil {
		return cfg, fmt.Errorf("%w: %w", ErrParseYAML, err)
	}

	e := newExpander(l.resolvers)
	e.node(ctx, tree)

	if err := e.err(); err != nil {
		return cfg, err
	}

	return decode[T](tree)
}
