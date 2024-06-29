package server

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/zrma/proglog/internal/config"
	"github.com/zrma/proglog/internal/log"
	"github.com/zrma/proglog/internal/pb"
)

func TestGRPCServer_ProduceAndConsume(t *testing.T) {
	f := newFixture(t)

	ctx := context.Background()

	want := &pb.Record{
		Value: []byte("hello world"),
	}

	produce, err := f.client.Produce(ctx, &pb.ProduceRequest{Record: want})
	require.NoError(t, err)

	consume, err := f.client.Consume(ctx, &pb.ConsumeRequest{Offset: produce.GetOffset()})
	require.NoError(t, err)

	require.Equal(t, want.GetValue(), consume.GetRecord().GetValue())
	require.Equal(t, produce.GetOffset(), consume.GetRecord().GetOffset())
}

func TestGRPCServer_ConsumePastBoundary(t *testing.T) {
	f := newFixture(t)

	ctx := context.Background()

	produce, err := f.client.Produce(ctx, &pb.ProduceRequest{Record: &pb.Record{Value: []byte("hello world")}})
	require.NoError(t, err)

	consume, err := f.client.Consume(ctx, &pb.ConsumeRequest{Offset: produce.GetOffset() + 1})
	require.Error(t, err)
	require.Nil(t, consume)

	got := status.Code(err)
	want := status.Code(pb.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	require.Equal(t, want, got)
}

func TestGrpcServer_Stream_ProduceAndConsume(t *testing.T) {
	f := newFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	records := []*pb.Record{
		{Value: []byte("first message"), Offset: 0},
		{Value: []byte("second message"), Offset: 1},
	}

	{
		stream, err := f.client.ProduceStream(ctx)
		require.NoError(t, err)

		for offset, record := range records {
			err := stream.Send(&pb.ProduceRequest{Record: record})
			require.NoError(t, err)

			resp, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, uint64(offset), resp.GetOffset())
		}
	}

	{
		stream, err := f.client.ConsumeStream(ctx, &pb.ConsumeRequest{Offset: 0})
		require.NoError(t, err)

		for offset, record := range records {
			resp, err := stream.Recv()
			require.NoError(t, err)

			got := resp.GetRecord()
			require.Equal(t, &pb.Record{
				Value:  record.GetValue(),
				Offset: record.GetOffset(),
			}, got)
			require.Equal(t, uint64(offset), got.GetOffset())
		}
	}
}

type fixture struct {
	client pb.LogClient
	cfg    *Config
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: config.ClientCertFile,
		KeyFile:  config.ClientKeyFile,
		CAFile:   config.CAFile,
	})
	require.NoError(t, err)

	clientOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLSConfig)),
	}
	clientConn, err := grpc.NewClient(l.Addr().String(), clientOpts...)
	require.NoError(t, err)

	dir, err := os.MkdirTemp(os.TempDir(), "server-test")
	require.NoError(t, err)

	diskLog, err := log.NewLog(dir, log.Config{})
	require.NoError(t, err)

	cfg := &Config{CommitLog: diskLog}

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
		Server:        true,
	})
	require.NoError(t, err)

	serverOpts := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(serverTLSConfig)),
	}
	svr, err := NewGRPCServer(cfg, serverOpts...)
	require.NoError(t, err)

	go func() {
		require.NoError(t, svr.Serve(l))
	}()

	client := pb.NewLogClient(clientConn)

	t.Cleanup(func() {
		svr.GracefulStop()
		err := l.Close()
		require.Error(t, err)
		require.True(t, errors.Is(err, net.ErrClosed), "because svr.GracefulStop closed the listener")

		require.NoError(t, clientConn.Close())
		require.NoError(t, diskLog.Remove())
	})

	return &fixture{
		client: client,
		cfg:    cfg,
	}
}
