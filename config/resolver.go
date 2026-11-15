package config

import "context"

type Resolver interface {
	Resolve(ctx context.Context, ref string) (any, error)
}

type ResolverFunc func(ctx context.Context, ref string) (any, error)

func (f ResolverFunc) Resolve(ctx context.Context, ref string) (any, error) {
	return f(ctx, ref)
}
