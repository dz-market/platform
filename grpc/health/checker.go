package health

import (
	"context"
	"time"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type checkerState uint8

const (
	stateCreated checkerState = iota
	stateRunning
	stateStopped
)

type dependency struct {
	name    string
	ping    func(context.Context) error
	timeout time.Duration
}

func (c *Checker) runChecks(ctx context.Context) {
	defer func() {
		c.stop()
		close(c.checksDone)
	}()

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		if ctx.Err() != nil {
			return
		}

		status := healthpb.HealthCheckResponse_NOT_SERVING

		if c.check(ctx) {
			status = healthpb.HealthCheckResponse_SERVING
		}

		c.setStatus(status)

		timer.Reset(c.period)
	}
}

func (c *Checker) check(ctx context.Context) bool {
	results := make(chan error, len(c.dependencies))

	for _, dep := range c.dependencies {
		go func(dep dependency) {
			results <- ping(ctx, dep)
		}(dep)
	}

	healthy := true

	for range c.dependencies {
		if err := <-results; err != nil {
			healthy = false
		}
	}

	return healthy
}

func (c *Checker) setStatus(status healthpb.HealthCheckResponse_ServingStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != stateRunning {
		return
	}

	c.health.SetServingStatus("", status)
}

func (c *Checker) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == stateStopped {
		return
	}

	previous := c.state
	c.state = stateStopped

	c.health.Shutdown()

	if c.cancel != nil {
		c.cancel()
	}

	if previous == stateCreated {
		close(c.checksDone)
	}
}

func ping(ctx context.Context, dep dependency) error {
	checkCtx, cancel := context.WithTimeout(ctx, dep.timeout)
	defer cancel()

	if err := checkCtx.Err(); err != nil {
		return checkCtx.Err()
	}

	if err := dep.ping(checkCtx); err != nil {
		return err
	}

	return checkCtx.Err()
}
