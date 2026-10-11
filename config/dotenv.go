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

func envResolver(env Env) Resolver {
	return ResolverFunc(
		func(_ context.Context, name string) (string, error) {
			if name == "" {
				return "", fmt.Errorf("%w: empty variable name", ErrInvalidReference)
			}

			if v, ok := env(name); ok {
				return v, nil
			}

			return "", ErrNotFound
		},
	)
}
