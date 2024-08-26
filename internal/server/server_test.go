package server

import (
	"context"
	"errors"
	"flag"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opencensus.io/examples/exporter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/zrma/proglog/internal/auth"
	"github.com/zrma/proglog/internal/config"
	"github.com/zrma/proglog/internal/log"
	"github.com/zrma/proglog/internal/pb"
)

var debug = flag.Bool("debug", false, "Enable observability for debugging.")

func TestMain(m *testing.M) {
	flag.Parse()
	if *debug {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		zap.ReplaceGlobals(logger)
	}
	os.Exit(m.Run())
}

func TestGRPCServer_ProduceAndConsume(t *testing.T) {
	t.Run("OK/RootClient", func(t *testing.T) {
		f := newFixture(t, config.RootClientCertFile, config.RootClientKeyFile)

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
	})

	t.Run("Err/NobodyClient", func(t *testing.T) {
		f := newFixture(t, config.NobodyClientCertFile, config.NobodyClientKeyFile)

		ctx := context.Background()

		produce, err := f.client.Produce(ctx, &pb.ProduceRequest{Record: &pb.Record{Value: []byte("hello world")}})
		require.Error(t, err)
		require.Nil(t, produce)

		gotCode := status.Code(err)
		wantCode := codes.PermissionDenied
		require.Equal(t, wantCode, gotCode)

		consume, err := f.client.Consume(ctx, &pb.ConsumeRequest{Offset: 0})
		require.Error(t, err)
		require.Nil(t, consume)

		gotCode = status.Code(err)
		wantCode = codes.PermissionDenied
		require.Equal(t, wantCode, gotCode)
	})
}

func TestGRPCServer_ConsumePastBoundary(t *testing.T) {
	f := newFixture(t, config.RootClientCertFile, config.RootClientKeyFile)

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
	t.Run("OK/RootClient", func(t *testing.T) {
		f := newFixture(t, config.RootClientCertFile, config.RootClientKeyFile)

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
	})

	t.Run("Err/NobodyClient", func(t *testing.T) {
		f := newFixture(t, config.NobodyClientCertFile, config.NobodyClientKeyFile)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()

		{
			stream, err := f.client.ProduceStream(ctx)
			require.NoError(t, err)

			record := &pb.Record{Value: []byte("first message"), Offset: 0}

			err = stream.Send(&pb.ProduceRequest{Record: record})
			require.NoError(t, err)

			resp, err := stream.Recv()
			require.Error(t, err)
			require.Nil(t, resp)

			gotCode := status.Code(err)
			wantCode := codes.PermissionDenied
			require.Equal(t, wantCode, gotCode)
		}

		{
			stream, err := f.client.ConsumeStream(ctx, &pb.ConsumeRequest{Offset: 0})
			require.NoError(t, err)

			resp, err := stream.Recv()
			require.Error(t, err)
			require.Nil(t, resp)

			gotCode := status.Code(err)
			wantCode := codes.PermissionDenied
			require.Equal(t, wantCode, gotCode)
		}
	})
}

type fixture struct {
	client pb.LogClient
	cfg    *Config
}

func newFixture(t *testing.T, cliCert, cliKey string) *fixture {
	t.Helper()

	flushTelemetry := startTelemetryExporter(t)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	dir, err := os.MkdirTemp(os.TempDir(), "server-test")
	require.NoError(t, err)

	diskLog, err := log.NewLog(dir, log.Config{})
	require.NoError(t, err)

	authorizer, err := auth.New(config.ACLModelFile, config.ACLPolicyFile)
	require.NoError(t, err)

	cfg := &Config{
		CommitLog:  diskLog,
		Authorizer: authorizer,
	}

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

	t.Cleanup(func() {
		svr.GracefulStop()
		err := l.Close()
		require.Error(t, err)
		require.True(t, errors.Is(err, net.ErrClosed), "because svr.GracefulStop closed the listener")

		require.NoError(t, diskLog.Remove())
		flushTelemetry()
	})

	return &fixture{
		client: newClient(t, l.Addr().String(), cliCert, cliKey),
		cfg:    cfg,
	}
}

func startTelemetryExporter(t *testing.T) func() {
	if !*debug {
		return func() {}
	}

	metricsLogFile, err := os.CreateTemp(os.TempDir(), "metrics-*.log")
	require.NoError(t, err)
	t.Logf("metrics log file: %s", metricsLogFile.Name())

	tracesLogFile, err := os.CreateTemp(os.TempDir(), "traces-*.log")
	require.NoError(t, err)
	t.Logf("traces log file: %s", tracesLogFile.Name())

	telemetryExporter, err := exporter.NewLogExporter(exporter.Options{
		MetricsLogFile:    metricsLogFile.Name(),
		TracesLogFile:     tracesLogFile.Name(),
		ReportingInterval: time.Second,
	})
	require.NoError(t, err)

	require.NoError(t, telemetryExporter.Start())

	return func() {
		time.Sleep(1_500 * time.Millisecond)
		telemetryExporter.Stop()
		telemetryExporter.Close()
	}
}

func newClient(
	t *testing.T,
	addr string,
	certFile string,
	keyFile string,
) pb.LogClient {
	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   config.CAFile,
	})
	require.NoError(t, err)

	clientOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLSConfig)),
	}
	clientConn, err := grpc.NewClient(addr, clientOpts...)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, clientConn.Close())
	})

	return pb.NewLogClient(clientConn)
}
