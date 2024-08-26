package server

import (
	"context"
	"time"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcCtxTags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"

	"github.com/zrma/proglog/internal/pb"
)

type CommitLog interface {
	Append(*pb.Record) (uint64, error)
	Read(uint64) (*pb.Record, error)
}

type Authorizer interface {
	Authorize(subject, object, action string) error
}

func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	logger := zap.L().Named("Server")
	zapOpts := []grpcZap.Option{
		grpcZap.WithDurationField(
			func(duration time.Duration) zapcore.Field {
				return zap.Int64("grpc.time_ns", duration.Nanoseconds())
			},
		),
	}

	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	//trace.ApplyConfig(trace.Config{
	//	DefaultSampler: func(p trace.SamplingParameters) trace.SamplingDecision {
	//		if strings.Contains(p.Name, "Produce") {
	//			return trace.SamplingDecision{Sample: true}
	//		}
	//		return halfSampler(p)
	//	},
	//})

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		return nil, err
	}

	opts = append(
		opts,
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpcCtxTags.StreamServerInterceptor(),
				grpcZap.StreamServerInterceptor(logger, zapOpts...),
				grpc_auth.StreamServerInterceptor(authenticate),
			),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpcCtxTags.UnaryServerInterceptor(),
				grpcZap.UnaryServerInterceptor(logger, zapOpts...),
				grpc_auth.UnaryServerInterceptor(authenticate),
			),
		),
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	)

	svc := grpc.NewServer(opts...)

	svr, err := newGrpcServer(config)
	if err != nil {
		return nil, err
	}

	pb.RegisterLogServer(svc, svr)
	return svc, nil
}

type Config struct {
	CommitLog  CommitLog
	Authorizer Authorizer
}

const (
	objectWildcard = "*"
	produceAction  = "produce"
	consumeAction  = "consume"
)

var _ pb.LogServer = (*grpcServer)(nil)

type grpcServer struct {
	pb.UnimplementedLogServer
	*Config
}

func newGrpcServer(config *Config) (*grpcServer, error) {
	return &grpcServer{Config: config}, nil
}

func (s grpcServer) Produce(ctx context.Context, req *pb.ProduceRequest) (*pb.ProduceResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		produceAction,
	); err != nil {
		return nil, err
	}

	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}

	return &pb.ProduceResponse{Offset: offset}, nil
}

func (s grpcServer) Consume(ctx context.Context, req *pb.ConsumeRequest) (*pb.ConsumeResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		consumeAction,
	); err != nil {
		return nil, err
	}

	record, err := s.CommitLog.Read(req.GetOffset())
	if err != nil {
		return nil, err
	}

	return &pb.ConsumeResponse{Record: record}, nil
}

func (s grpcServer) ProduceStream(stream pb.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}

		if err := stream.Send(res); err != nil {
			return err
		}
	}
}

func (s grpcServer) ConsumeStream(req *pb.ConsumeRequest, stream pb.Log_ConsumeStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
			case pb.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}

			if err := stream.Send(res); err != nil {
				return err
			}

			req.Offset++
		}
	}
}

func authenticate(ctx context.Context) (context.Context, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(codes.Unknown, "couldn't find peer info").Err()
	}

	if p.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}

	tlsInfo := p.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName

	return context.WithValue(ctx, subjectContextKey{}, subject), nil
}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}
