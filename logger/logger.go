package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/dz-market/platform/logctx"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

type Options struct {
	Format         Format
	Output         io.Writer
	HandlerOptions slog.HandlerOptions
	Service        string
	Version        string
}

func New(opts Options) (*slog.Logger, error) {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}

	var handler slog.Handler

	switch opts.Format {
	case "", FormatJSON:
		handler = slog.NewJSONHandler(out, &opts.HandlerOptions)

	case FormatText:
		handler = slog.NewTextHandler(out, &opts.HandlerOptions)

	default:
		return nil, fmt.Errorf("unsupported log format %q", opts.Format)
	}

	log := slog.New(logctx.ContextHandler(handler)).With(
		slog.String("service", opts.Service),
		slog.String("version", opts.Version),
	)

	return log, nil
}
