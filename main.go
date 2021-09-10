package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"
)

var (
	gRPCServerAddr   = envOrDefaultString("GRPC_SERVER_ADDR", ":50050")
	gRPCGatewayAddr  = envOrDefaultString("GRPC_GATEWAY_ADDR", ":50051")
	promAddr         = envOrDefaultString("PROMETHEUS_ADDR", ":9092")
	enableLoopClient = envOrDefaultBool("ENABLE_LOOP_CLIENT", true)
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		logger.Warn("Server will shut down in 5s")
		time.Sleep(5 * time.Second)
		cancel()
	}()

	go func() {
		s := newServer(gRPCServerAddr, gRPCGatewayAddr, promAddr, logger)
		if err := s.Start(ctx); err != nil {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	go func() {
		if !enableLoopClient {
			return
		}
		c, closer, err := newLoopClient(ctx, gRPCServerAddr, logger)
		if err != nil {
			logger.Fatal("failed to initialize the loop client", zap.Error(err))
		}
		for {
			select {
			case <-ctx.Done():
				_ = closer()
				break
			default:
				_, _ = c.ping(ctx)
			}
		}
	}()

	<-ctx.Done()
}
