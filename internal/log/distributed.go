package log

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/zrma/proglog/internal/pb"
	"google.golang.org/protobuf/proto"
)

type DistributedLog struct {
	Config  Config
	log     *Log
	raftLog *logStore
	raft    *raft.Raft
}

func NewDistributedLog(dataDir string, config Config) (*DistributedLog, error) {
	l := &DistributedLog{
		Config: config,
	}

	if err := l.setupLog(dataDir); err != nil {
		return nil, err
	}

	if err := l.setupRaft(dataDir); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *DistributedLog) setupLog(dataDir string) error {
	logDir := filepath.Join(dataDir, "log")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	var err error
	l.log, err = NewLog(logDir, l.Config)
	return err
}

func (l *DistributedLog) setupRaft(dataDir string) error {
	fsm := &fsm{log: l.log}

	logDir := filepath.Join(dataDir, "raft", "log")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logConfig := l.Config
	logConfig.Segment.InitialOffset = 1

	var err error
	l.raftLog, err = newLogStore(logDir, logConfig)
	if err != nil {
		return err
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft", "stable"))
	if err != nil {
		return err
	}

	const retain = 1
	snapshotStore, err := raft.NewFileSnapshotStore(filepath.Join(dataDir, "raft"), retain, os.Stderr)
	if err != nil {
		return err
	}

	const (
		maxPool = 5
		timeout = 10 * time.Second
	)

	transport := raft.NewNetworkTransport(l.Config.Raft.StreamLayer, maxPool, timeout, os.Stderr)

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = l.Config.Raft.LocalID

	if l.Config.Raft.HeartbeatTimeout > 0 {
		raftConfig.HeartbeatTimeout = l.Config.Raft.HeartbeatTimeout
	}
	if l.Config.Raft.ElectionTimeout > 0 {
		raftConfig.ElectionTimeout = l.Config.Raft.ElectionTimeout
	}
	if l.Config.Raft.LeaderLeaseTimeout > 0 {
		raftConfig.LeaderLeaseTimeout = l.Config.Raft.LeaderLeaseTimeout
	}
	if l.Config.Raft.CommitTimeout > 0 {
		raftConfig.CommitTimeout = l.Config.Raft.CommitTimeout
	}

	l.raft, err = raft.NewRaft(raftConfig, fsm, l.raftLog, stableStore, snapshotStore, transport)
	if err != nil {
		return err
	}

	hasState, err := raft.HasExistingState(l.raftLog, stableStore, snapshotStore)
	if err != nil {
		return err
	}

	if l.Config.Raft.Bootstrap && !hasState {
		config := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raftConfig.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		err = l.raft.BootstrapCluster(config).Error()
	}

	return err
}

func (l *DistributedLog) Append(record *pb.Record) (uint64, error) {
	res, err := l.apply(AppendRequestType, &pb.ProduceRequest{Record: record})
	if err != nil {
		return 0, err
	}

	return res.(*pb.ProduceResponse).Offset, nil
}

func (l *DistributedLog) apply(reqType RequestType, req proto.Message) (any, error) {
	var buf bytes.Buffer
	_, err := buf.Write([]byte{byte(reqType)})
	if err != nil {
		return nil, err
	}
	b, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(b)
	if err != nil {
		return nil, err
	}

	const timeout = 10 * time.Second

	future := l.raft.Apply(buf.Bytes(), timeout)
	if future.Error() != nil {
		return nil, future.Error()
	}

	res := future.Response()
	if err, ok := res.(error); ok {
		return nil, err
	}

	return res, nil
}

func (l *DistributedLog) Read(offset uint64) (*pb.Record, error) {
	return l.log.Read(offset)
}

var _ raft.FSM = (*fsm)(nil)

type fsm struct {
	log *Log
}

type RequestType uint8

const (
	AppendRequestType RequestType = 0
)

func (f *fsm) Apply(record *raft.Log) any {
	buf := record.Data
	reqType := RequestType(buf[0])

	switch reqType {
	case AppendRequestType:
		return f.applyAppend(buf[1:])
	}

	return nil
}

func (f *fsm) applyAppend(b []byte) any {
	var req pb.ProduceRequest
	if err := proto.Unmarshal(b, &req); err != nil {
		return err
	}

	offset, err := f.log.Append(req.Record)
	if err != nil {
		return err
	}

	return &pb.ProduceResponse{Offset: offset}
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	r := f.log.Reader()
	return &snapshot{reader: r}, nil
}

func (f *fsm) Restore(r io.ReadCloser) error {
	b := make([]byte, lenWidth)
	var buf bytes.Buffer
	for i := 0; ; i++ {
		_, err := io.ReadFull(r, b)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		size := int64(enc.Uint64(b))
		if _, err := io.CopyN(&buf, r, size); err != nil {
			return err
		}

		var record pb.Record
		if err := proto.Unmarshal(buf.Bytes(), &record); err != nil {
			return err
		}

		if i == 0 {
			f.log.Config.Segment.InitialOffset = record.Offset
			if err := f.log.Reset(); err != nil {
				return err
			}
		}

		if _, err := f.log.Append(&record); err != nil {
			return err
		}

		buf.Reset()
	}
	return nil
}

var _ raft.FSMSnapshot = (*snapshot)(nil)

type snapshot struct {
	reader io.Reader
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := io.Copy(sink, s.reader); err != nil {
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *snapshot) Release() {}

var _ raft.LogStore = (*logStore)(nil)

type logStore struct {
	*Log
}

func newLogStore(dir string, config Config) (*logStore, error) {
	l, err := NewLog(dir, config)
	if err != nil {
		return nil, err
	}
	return &logStore{l}, nil
}

func (l *logStore) FirstIndex() (uint64, error) {
	return l.LowestOffset()
}

func (l *logStore) LastIndex() (uint64, error) {
	return l.HighestOffset()
}

func (l *logStore) GetLog(index uint64, out *raft.Log) error {
	in, err := l.Read(index)
	if err != nil {
		return err
	}

	out.Data = in.Value
	out.Index = in.Offset
	out.Type = raft.LogType(in.Type)
	out.Term = in.Term

	return nil
}

func (l *logStore) StoreLog(record *raft.Log) error {
	return l.StoreLogs([]*raft.Log{record})
}

func (l *logStore) StoreLogs(records []*raft.Log) error {
	for _, record := range records {
		if _, err := l.Append(&pb.Record{
			Value: record.Data,
			Term:  record.Term,
			Type:  uint32(record.Type),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (l *logStore) DeleteRange(_, max uint64) error {
	return l.Truncate(max)
}

var _ raft.StreamLayer = (*StreamLayer)(nil)

type StreamLayer struct {
	ln              net.Listener
	serverTLSConfig *tls.Config
	peerTLSConfig   *tls.Config
}

func NewStreamLayer(ln net.Listener, serverTLSConfig, peerTLSConfig *tls.Config,
) *StreamLayer {
	return &StreamLayer{
		ln:              ln,
		serverTLSConfig: serverTLSConfig,
		peerTLSConfig:   peerTLSConfig,
	}
}

const RaftRPC = 1

func (s *StreamLayer) Dial(addr raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", string(addr))
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write([]byte{RaftRPC}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if s.peerTLSConfig != nil {
		conn = tls.Client(conn, s.peerTLSConfig)
	}
	return conn, nil
}

func (s *StreamLayer) Accept() (net.Conn, error) {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return nil, err
		}
		b := make([]byte, 1)
		if _, err := io.ReadFull(conn, b); err != nil {
			_ = conn.Close()
			continue
		}
		if b[0] != RaftRPC {
			_ = conn.Close()
			continue
		}
		if s.serverTLSConfig != nil {
			conn = tls.Server(conn, s.serverTLSConfig)
		}
		return conn, nil
	}
}

func (s *StreamLayer) Close() error {
	return s.ln.Close()
}

func (s *StreamLayer) Addr() net.Addr {
	return s.ln.Addr()
}
