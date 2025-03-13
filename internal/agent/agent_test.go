package agent_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
	"github.com/zrma/proglog/internal/agent"
	"github.com/zrma/proglog/internal/config"
	"github.com/zrma/proglog/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestAgent(t *testing.T) {
	t.Skip("Need to implement consensus")

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		Server:        true,
		ServerAddress: "127.0.0.1",
	})
	require.NoError(t, err)

	peerTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.RootClientCertFile,
		KeyFile:       config.RootClientKeyFile,
		CAFile:        config.CAFile,
		Server:        false,
		ServerAddress: "127.0.0.1",
	})
	require.NoError(t, err)

	var agents []*agent.Agent
	for i := range 3 {
		ports := dynaport.Get(2)
		membershipPort := ports[0]
		rpcPort := ports[1]

		bindAddr := fmt.Sprintf("127.0.0.1:%d", membershipPort)

		dataDir, err := os.MkdirTemp("", "agent-test-log")
		require.NoError(t, err)

		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(startJoinAddrs, agents[0].Config.BindAddr)
		}

		agent, err := agent.New(agent.Config{
			NodeName:        fmt.Sprintf("agent-%d", i),
			StartJoinPeers:  startJoinAddrs,
			BindAddr:        bindAddr, // membership port
			RPCPort:         rpcPort,  // gRPC port
			DataDir:         dataDir,
			ACLModelFile:    config.ACLModelFile,
			ACLPolicyFile:   config.ACLPolicyFile,
			ServerTLSConfig: serverTLSConfig,
			PeerTLSConfig:   peerTLSConfig,
		})
		require.NoError(t, err)

		agents = append(agents, agent)
	}

	defer func() {
		for _, agent := range agents {
			err := agent.Shutdown()
			require.NoError(t, err)
			require.NoError(t, os.RemoveAll(agent.Config.DataDir))
		}
	}()

	time.Sleep(3 * time.Second)

	leaderConn, leaderClient := client(t, agents[0], peerTLSConfig)
	defer leaderConn.Close()

	produceResponse, err := leaderClient.Produce(
		context.Background(),
		&pb.ProduceRequest{
			Record: &pb.Record{
				Value: []byte("foo"),
			},
		},
	)
	require.NoError(t, err)

	consumeResponse, err := leaderClient.Consume(
		context.Background(),
		&pb.ConsumeRequest{
			Offset: produceResponse.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, []byte("foo"), consumeResponse.Record.Value)

	time.Sleep(3 * time.Second)

	followerConn, followerClient := client(t, agents[1], peerTLSConfig)
	defer followerConn.Close()

	consumeResponse, err = followerClient.Consume(
		context.Background(),
		&pb.ConsumeRequest{
			Offset: produceResponse.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, []byte("foo"), consumeResponse.Record.Value)

	consumeResponse, err = leaderClient.Consume(
		context.Background(),
		&pb.ConsumeRequest{
			Offset: produceResponse.Offset + 1,
		},
	)
	require.Nil(t, consumeResponse)
	require.Error(t, err)

	got := status.Code(err)
	want := status.Code(pb.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	require.Equal(t, want, got)
}

func client(t *testing.T, agent *agent.Agent, tlsConfig *tls.Config) (*grpc.ClientConn, pb.LogClient) {
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
	rpcAddr, err := agent.Config.RPCAddr()
	require.NoError(t, err)

	conn, err := grpc.NewClient(rpcAddr, opts...)
	require.NoError(t, err)

	return conn, pb.NewLogClient(conn)
}
