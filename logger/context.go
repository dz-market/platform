package logger

import (
	"context"
	"log/slog"
)

type attrsKey struct{}

func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing := attrsFrom(ctx)

	all := make([]slog.Attr, 0, len(existing)+len(attrs))
	all = append(all, existing...)
	all = append(all, attrs...)

	return context.WithValue(ctx, attrsKey{}, all)
}

type contextHandler struct {
	slog.Handler
}

func ContextHandler(h slog.Handler) slog.Handler {
	return contextHandler{h}
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(attrsFrom(ctx)...)

	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}

func attrsFrom(ctx context.Context) []slog.Attr {
	attrs, ok := ctx.Value(attrsKey{}).([]slog.Attr)
	if !ok {
		return nil
	}

	return attrs
}
