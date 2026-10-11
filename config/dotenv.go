package config

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const schemeEnv = "env"

func envLookup(envPath string) (lookupFunc, error) {
	vars, err := godotenv.Read(envPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", envPath, err)
	}

	return func(name string) (string, bool) {
		if v, ok := os.LookupEnv(name); ok {
			return v, true
		}

		v, ok := vars[name]

		return v, ok
	}, nil
}

func envResolver(lookup lookupFunc) Resolver {
	return ResolverFunc(
		func(_ context.Context, ref string) (string, error) {
			name, def, hasDefault := strings.Cut(ref, ":-")

			if name == "" {
				return "", fmt.Errorf("%w: empty variable name", ErrInvalidReference)
			}

			if v, ok := lookup(name); ok {
				return v, nil
			}

			if hasDefault {
				return def, nil
			}

			return "", ErrNotFound
		},
	)
}
