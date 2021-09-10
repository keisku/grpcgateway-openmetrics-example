package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	testpb "github.com/kskumgk63/grpcgateway-openmetrics-example/proto/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ testpb.TestServiceServer = (*server)(nil)

type server struct {
	gRPCAddr        string
	gRPCGatewayAddr string
	promAddr        string
	logger          *zap.Logger
}

func newServer(gRPCAddr, gRPCGWAddr, promAddr string, logger *zap.Logger) *server {
	return &server{
		gRPCAddr:        gRPCAddr,
		gRPCGatewayAddr: gRPCGWAddr,
		promAddr:        promAddr,
		logger:          logger,
	}
}

func (s *server) HealthCheck(ctx context.Context, req *testpb.HealthCheckRequest) (*testpb.HealthCheckResponse, error) {
	return &testpb.HealthCheckResponse{}, nil
}

func (s *server) Ping(ctx context.Context, req *testpb.PingRequest) (*testpb.PingResponse, error) {
	time.Sleep(time.Millisecond * time.Duration(req.GetSleepTimeMs()))

	if req.GetStatusCodeReturned() == code.Code_OK {
		return &testpb.PingResponse{Value: req.GetValue()}, nil
	}

	return nil, status.Error(codes.Code(req.GetStatusCodeReturned()), "returning requested error")
}

func (s *server) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	regi := prometheus.NewRegistry()
	grpcMetrics := grpc_prometheus.NewServerMetrics()
	if err := regi.Register(grpcMetrics); err != nil {
		return err
	}

	g.Go(func() error {
		return (&http.Server{
			Handler: promhttp.HandlerFor(regi, promhttp.HandlerOpts{}),
			Addr:    s.promAddr,
		}).ListenAndServe()
	})

	g.Go(func() error {
		lis, err := net.Listen("tcp", s.gRPCAddr)
		if err != nil {
			return err
		}
		defer func() { _ = lis.Close() }()

		grpcsvc := grpc.NewServer(
			grpc_middleware.WithUnaryServerChain(
				grpcMetrics.UnaryServerInterceptor(),
				grpc_zap.UnaryServerInterceptor(s.logger),
			),
		)
		testpb.RegisterTestServiceServer(grpcsvc, s)
		grpcMetrics.InitializeMetrics(grpcsvc)

		s.logger.Info("gRPC server starts")
		return grpcsvc.Serve(lis)
	})

	g.Go(func() error {
		conn, err := grpc.DialContext(
			ctx,
			s.gRPCAddr,
			grpc.WithBlock(),
			grpc.WithInsecure(),
			grpc.WithDisableHealthCheck(),
		)
		if err != nil {
			return fmt.Errorf("failed to dial gRPC server: %w", err)
		}
		defer func() { _ = conn.Close() }()

		gwmux := runtime.NewServeMux()
		if err := testpb.RegisterTestServiceHandler(ctx, gwmux, conn); err != nil {
			return fmt.Errorf("failed to register test service handler: %w", err)
		}

		s.logger.Info("gRPC-Gateway server starts")
		return (&http.Server{
			Addr:    s.gRPCGatewayAddr,
			Handler: gwmux,
		}).ListenAndServe()
	})

	return g.Wait()
}
