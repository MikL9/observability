package tracing

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func GRPCServer() grpc.ServerOption {
	return grpc.StatsHandler(otelgrpc.NewServerHandler())
}

func GRPCClient() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler())
}
