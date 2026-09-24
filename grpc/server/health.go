package server

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func RegisterHealth(srv grpc.ServiceRegistrar) *health.Server {
	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)

	return hs
}

func ReportHealth(hs *health.Server) func(healthy bool) {
	return func(healthy bool) {
		status := healthpb.HealthCheckResponse_NOT_SERVING

		if healthy {
			status = healthpb.HealthCheckResponse_SERVING
		}

		hs.SetServingStatus("", status)
	}
}
