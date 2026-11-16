package health

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type Options struct {
	Period  time.Duration
	Timeout time.Duration
}

type DependencyOptions struct {
	Timeout time.Duration
}

type Checker struct {
	health  *health.Server
	period  time.Duration
	timeout time.Duration

	mu           sync.Mutex
	state        checkerState
	dependencies []dependency
	cancel       context.CancelFunc
	checksDone   chan struct{}
}

func New(registrar grpc.ServiceRegistrar, opts Options) (*Checker, error) {
	if registrar == nil {
		return nil, errors.New("registrar must not be nil")
	}

	if opts.Period <= 0 {
		return nil, fmt.Errorf("period must be positive, got %s", opts.Period)
	}

	if opts.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive, got %s", opts.Timeout)
	}

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	healthpb.RegisterHealthServer(registrar, healthSrv)

	return &Checker{
		health:     healthSrv,
		period:     opts.Period,
		timeout:    opts.Timeout,
		checksDone: make(chan struct{}),
	}, nil
}

func (c *Checker) Register(name string, opts DependencyOptions, ping func(context.Context) error) error {
	if name == "" {
		return errors.New("dependency name must not be empty")
	}

	if ping == nil {
		return errors.New("dependency ping function must not be nil")
	}

	if opts.Timeout < 0 {
		return fmt.Errorf("timeout must be positive, got %s", opts.Timeout)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != stateCreated {
		return errors.New("dependencies can only be registered before run")
	}

	for _, dep := range c.dependencies {
		if dep.name == name {
			return fmt.Errorf("dependency %q is already registered", name)
		}
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = c.timeout
	}

	c.dependencies = append(
		c.dependencies, dependency{
			name:    name,
			ping:    ping,
			timeout: timeout,
		},
	)

	return nil
}

func (c *Checker) Run(ctx context.Context) error {
	c.mu.Lock()

	if c.state != stateCreated {
		c.mu.Unlock()

		return errors.New("checker can only be run once")
	}

	checkCtx, cancel := context.WithCancel(ctx)

	c.cancel = cancel
	c.state = stateRunning

	c.mu.Unlock()

	go c.runChecks(checkCtx)

	select {
	case <-c.checksDone:
	case <-checkCtx.Done():
	}

	c.stop()

	return nil
}

func (c *Checker) Shutdown(ctx context.Context) error {
	c.stop()

	select {
	case <-c.checksDone:
		return nil
	default:
	}

	select {
	case <-c.checksDone:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}
