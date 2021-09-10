package main

import (
	"context"
	"math/rand"

	testpb "github.com/kskumgk63/grpcgateway-openmetrics-example/proto/test"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
)

type loopClient struct {
	testpb.TestServiceClient
	logger *zap.Logger
}

func newLoopClient(ctx context.Context, serverAddr string, logger *zap.Logger) (*loopClient, func() error, error) {
	conn, err := grpc.DialContext(
		ctx,
		serverAddr,
		grpc.WithInsecure(),
		grpc.WithDisableHealthCheck(),
	)
	if err != nil {
		return nil, nil, err
	}
	return &loopClient{testpb.NewTestServiceClient(conn), logger}, func() error { return conn.Close() }, nil
}

func (c *loopClient) ping(ctx context.Context) (*testpb.PingResponse, error) {
	return c.Ping(ctx, &testpb.PingRequest{
		Value:              randString(rand.Intn(10)),
		SleepTimeMs:        rand.Int31n(10000), // up to 10s
		StatusCodeReturned: code.Code(rand.Int31n(16)),
	})
}
