package log

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/zrma/proglog/internal/pb"
)

type Replicator struct {
	DialOptions []grpc.DialOption
	LocalServer pb.LogClient

	logger *zap.Logger

	mu      sync.Mutex
	servers map[string]chan struct{}
	closed  bool
	close   chan struct{}
}

func (r *Replicator) Join(name, addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.init()

	if r.closed {
		return nil
	}

	if _, ok := r.servers[name]; ok {
		// already replicating to this server
		return nil
	}

	r.servers[name] = make(chan struct{})

	go r.replicateFrom(addr, r.servers[name])

	return nil
}

func (r *Replicator) replicateFrom(addr string, leave chan struct{}) {
	cc, err := grpc.NewClient(addr, r.DialOptions...)
	if err != nil {
		r.logError(err, "failed to dial", addr)
		return
	}
	defer cc.Close()

	client := pb.NewLogClient(cc)

	ctx := context.Background()
	stream, err := client.ConsumeStream(ctx, &pb.ConsumeRequest{Offset: 0})
	if err != nil {
		r.logError(err, "failed to consume", addr)
		return
	}

	records := make(chan *pb.Record)
	go func() {
		for {
			res, err := stream.Recv()
			if err != nil {
				r.logError(err, "failed to receive", addr)
				return
			}
			records <- res.Record
		}
	}()

	for {
		select {
		case <-r.close:
			return
		case <-leave:
			return
		case record := <-records:
			_, err := r.LocalServer.Produce(ctx, &pb.ProduceRequest{Record: record})
			if err != nil {
				r.logError(err, "failed to produce", addr)
				return
			}
		}
	}
}

func (r *Replicator) Leave(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.init()

	if ch, ok := r.servers[name]; ok {
		close(ch)
		delete(r.servers, name)
	}

	return nil
}

func (r *Replicator) init() {
	if r.logger == nil {
		r.logger = zap.L().Named("replicator")
	}
	if r.servers == nil {
		r.servers = make(map[string]chan struct{})
	}
	if r.close == nil {
		r.close = make(chan struct{})
	}
}

func (r *Replicator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.init()

	if r.closed {
		return nil
	}

	r.closed = true
	close(r.close)

	return nil
}

func (r *Replicator) logError(err error, msg string, addr string) {
	r.logger.Error(msg,
		zap.String("addr", addr),
		zap.Error(err),
	)
}
