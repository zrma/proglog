package server

import (
	"errors"
	"sync"

	"github.com/zrma/proglog/pkg/pb"
)

type Log struct {
	mu      sync.Mutex
	records []*pb.Record
}

func NewLog() *Log {
	return &Log{}
}

func (c *Log) Append(record *pb.Record) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	record.Offset = uint64(len(c.records))
	c.records = append(c.records, record)

	return record.GetOffset(), nil
}

func (c *Log) Read(offset uint64) (*pb.Record, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if offset >= uint64(len(c.records)) {
		return nil, ErrRecordNotFound
	}

	return c.records[offset], nil
}

var ErrRecordNotFound = errors.New("record not found at the given offset")
