package logger

import (
	"context"
	"log/slog"
	"slices"
)

type attrsKey struct{}

func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	all := slices.Concat(attrsFrom(ctx), attrs)

	return context.WithValue(ctx, attrsKey{}, all)
}

type contextHandler struct {
	slog.Handler
}

func ContextHandler(h slog.Handler) slog.Handler {
	return contextHandler{
		Handler: h,
	}
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	if attrs := attrsFrom(ctx); len(attrs) > 0 {
		record = record.Clone()
		record.AddAttrs(attrs...)
	}

	return h.Handler.Handle(ctx, record)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{
		Handler: h.Handler.WithGroup(name),
	}
}

func attrsFrom(ctx context.Context) []slog.Attr {
	attrs, ok := ctx.Value(attrsKey{}).([]slog.Attr)
	if !ok {
		return nil
	}

	return attrs
}
