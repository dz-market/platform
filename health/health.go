package health

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"
)

const (
	defaultPeriod  = 5 * time.Second
	defaultTimeout = 2 * time.Second
)

type Probe func(ctx context.Context) error

type Options struct {
	Period  time.Duration
	Timeout time.Duration
}

type check struct {
	name  string
	probe Probe
	up    bool
}

type Checker struct {
	period  time.Duration
	timeout time.Duration
	log     *slog.Logger

	mu      sync.Mutex
	checks  []*check
	healthy bool
}

func New(opts Options, log *slog.Logger) *Checker {
	return &Checker{
		period:  cmp.Or(opts.Period, defaultPeriod),
		timeout: cmp.Or(opts.Timeout, defaultTimeout),
		log:     log,
		healthy: true,
	}
}

func (c *Checker) Register(name string, probe Probe) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checks = append(c.checks, &check{name: name, probe: probe, up: true})
}

func (c *Checker) Run(ctx context.Context, onChange func(healthy bool)) error {
	ticker := time.NewTicker(c.period)
	defer ticker.Stop()

	c.evaluate(ctx, onChange)

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			c.evaluate(ctx, onChange)
		}
	}
}

func (c *Checker) Healthy() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.healthy
}

func (c *Checker) evaluate(ctx context.Context, onChange func(bool)) {
	c.mu.Lock()
	checks := slices.Clone(c.checks)
	c.mu.Unlock()

	healthy := true

	for _, check := range checks {
		err := c.probe(ctx, check.probe)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			healthy = false
		}

		c.record(ctx, check, err)
	}

	c.mu.Lock()
	changed := healthy != c.healthy
	c.healthy = healthy
	c.mu.Unlock()

	if changed {
		onChange(healthy)
	}
}

func (c *Checker) record(ctx context.Context, check *check, err error) {
	up := err == nil

	if up == check.up {
		c.log.DebugContext(
			ctx, "health check completed",
			slog.String("dependency", check.name),
		)

		return
	}

	check.up = up

	if up {
		c.log.InfoContext(
			ctx, "dependency restored",
			slog.String("dependency", check.name),
		)

		return
	}

	c.log.ErrorContext(
		ctx, "dependency unavailable",
		slog.String("dependency", check.name),
		slog.Any("err", err),
	)
}

func (c *Checker) probe(ctx context.Context, probe Probe) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return probe(ctx)
}
