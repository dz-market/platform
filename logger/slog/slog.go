package slog

import (
	"io"
	"log/slog"
	"os"

	"github.com/dz-market/platform/logger"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

type Options struct {
	Level   slog.Level
	Format  Format
	Service string
	Version string
	Output  io.Writer
}

func New(opts Options) *slog.Logger {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}

	handlerOpts := &slog.HandlerOptions{Level: opts.Level}

	var handler slog.Handler

	if opts.Format == FormatText {
		handler = slog.NewTextHandler(out, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(out, handlerOpts)
	}

	return slog.New(logger.ContextHandler(handler)).With(
		slog.String("service", opts.Service),
		slog.String("version", opts.Version),
	)
}
