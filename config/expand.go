package config

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

var refPattern = regexp.MustCompile(`\$\$|\$\{[^{}]*\}`)

type result struct {
	value string
	err   error
}

type expander struct {
	resolvers map[string]Resolver
	cache     map[string]result
	errs      map[string]error
}

func newExpander(resolvers map[string]Resolver) *expander {
	return &expander{
		resolvers: resolvers,
		cache:     make(map[string]result),
		errs:      make(map[string]error),
	}
}

func (e *expander) node(ctx context.Context, node any) any {
	switch v := node.(type) {
	case string:
		return e.string(ctx, v)

	case map[string]any:
		for key, child := range v {
			v[key] = e.node(ctx, child)
		}

		return v

	case []any:
		for i, child := range v {
			v[i] = e.node(ctx, child)
		}

		return v

	default:
		return node
	}
}

func (e *expander) string(ctx context.Context, s string) string {
	return refPattern.ReplaceAllStringFunc(
		s, func(match string) string {
			if match == "$$" {
				return "$"
			}

			return e.ref(ctx, match)
		},
	)
}

func (e *expander) ref(ctx context.Context, match string) string {
	inner := match[2 : len(match)-1]

	scheme, rest, ok := strings.Cut(inner, ":")
	if !ok || scheme == "" {
		e.errs[match] = fmt.Errorf("%w: %s: missing scheme, use ${%s:%s}", ErrInvalidReference, match, schemeEnv, inner)

		return ""
	}

	r, ok := e.resolvers[scheme]
	if !ok {
		e.errs[match] = fmt.Errorf("%w: %s: unknown scheme %q", ErrInvalidReference, match, scheme)

		return ""
	}

	ref, def, hasDefault := strings.Cut(rest, ":-")
	key := scheme + ":" + ref

	res, ok := e.cache[key]
	if !ok {
		value, err := r.Resolve(ctx, ref)
		res = result{
			value: value,
			err:   err,
		}
		e.cache[key] = res
	}

	switch {
	case res.err == nil:
		return res.value

	case errors.Is(res.err, ErrNotFound) && hasDefault:
		return def

	case errors.Is(res.err, ErrNotFound):
		e.errs[match] = fmt.Errorf("%w: %s", ErrMissingReference, match)

	case errors.Is(res.err, ErrInvalidReference):
		e.errs[match] = fmt.Errorf(
			"%w: %s: %s",
			ErrInvalidReference, match, strings.TrimPrefix(res.err.Error(), ErrInvalidReference.Error()+": "),
		)

	default:
		e.errs[match] = fmt.Errorf("%w: %s: %w", ErrResolverFailed, match, res.err)
	}

	return ""
}

func (e *expander) err() error {
	errs := slices.SortedFunc(
		maps.Values(e.errs), func(a, b error) int {
			return strings.Compare(a.Error(), b.Error())
		},
	)

	return errors.Join(errs...)
}
