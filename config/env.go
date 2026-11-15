package config

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

const schemeEnv = "env"

var _ Resolver = Env(nil)

type Env func(name string) (string, bool)

func LoadEnv(path string) (Env, error) {
	vars, err := godotenv.Read(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return func(name string) (string, bool) {
		if v, ok := os.LookupEnv(name); ok {
			return v, true
		}

		v, ok := vars[name]

		return v, ok
	}, nil
}

func (env Env) Required(name string) (string, error) {
	value, ok := env(name)
	if !ok || value == "" {
		return "", fmt.Errorf("%w: environment variable %q is required", ErrNotFound, name)
	}

	return value, nil
}

func (env Env) Resolve(_ context.Context, name string) (any, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: empty variable name", ErrInvalidReference)
	}

	if value, ok := env(name); ok {
		return value, nil
	}

	return nil, ErrNotFound
}
