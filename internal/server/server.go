package server

import (
	"context"

	"google.golang.org/grpc"

	"github.com/zrma/proglog/internal/pb"
)

type CommitLog interface {
	Append(*pb.Record) (uint64, error)
	Read(uint64) (*pb.Record, error)
}

func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	svc := grpc.NewServer(opts...)
	
	svr, err := newGrpcServer(config)
	if err != nil {
		return nil, err
	}

	pb.RegisterLogServer(svc, svr)
	return svc, nil
}

type Config struct {
	CommitLog CommitLog
}

var _ pb.LogServer = (*grpcServer)(nil)

type grpcServer struct {
	pb.UnimplementedLogServer
	*Config
}

func newGrpcServer(config *Config) (*grpcServer, error) {
	return &grpcServer{Config: config}, nil
}

func (s grpcServer) Produce(_ context.Context, req *pb.ProduceRequest) (*pb.ProduceResponse, error) {
	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}

	return &pb.ProduceResponse{Offset: offset}, nil
}

func (s grpcServer) Consume(_ context.Context, req *pb.ConsumeRequest) (*pb.ConsumeResponse, error) {
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
