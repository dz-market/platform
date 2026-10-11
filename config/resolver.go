package config

import "context"

type Resolver interface {
	Resolve(ctx context.Context, ref string) (string, error)
}

type ResolverFunc func(ctx context.Context, ref string) (string, error)

func (f ResolverFunc) Resolve(ctx context.Context, ref string) (string, error) {
	return f(ctx, ref)
}
