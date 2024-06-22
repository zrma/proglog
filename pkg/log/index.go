package log

import (
	"errors"
	"io"
	"os"

	"github.com/edsrzf/mmap-go"
)

const (
	offWidth = 4
	posWidth = 8
	entWidth = offWidth + posWidth
)

type index struct {
	file *os.File
	mmap mmap.MMap
	size uint64
}

func newIndex(f *os.File, cfg Config) (*index, error) {
	idx := &index{
		file: f,
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	idx.size = uint64(fi.Size())
	if idx.size > cfg.Segment.MaxIndexBytes {
		return nil, ErrIndexMaxSizeExceeded
	}

	if err := os.Truncate(f.Name(), int64(cfg.Segment.MaxIndexBytes)); err != nil {
		return nil, err
	}

	idx.mmap, err = mmap.MapRegion(f, -1, mmap.RDWR, 0, 0)
	if err != nil {
		return nil, err
	}

	return idx, nil
}

var ErrIndexMaxSizeExceeded = errors.New("index file size exceeded maximum limit")

func (i *index) Close() error {
	if err := i.mmap.Flush(); err != nil {
		return err
	}

	if err := i.mmap.Unmap(); err != nil {
		return err
	}

	if err := i.file.Sync(); err != nil {
		return err
	}

	if err := i.file.Truncate(int64(i.size)); err != nil {
		return err
	}

	return i.file.Close()
}

func (i *index) Read(in int64) (out uint32, pos uint64, err error) {
	if i.size == 0 {
		return 0, 0, io.EOF
	}

	if in < 0 {
		out = uint32((i.size / entWidth) - 1)
	} else {
		out = uint32(in)
	}

	pos = uint64(out) * entWidth

	if i.size < pos+entWidth {
		return 0, 0, io.EOF
	}

	out = enc.Uint32(i.mmap[pos : pos+offWidth])
	pos = enc.Uint64(i.mmap[pos+offWidth : pos+entWidth])

	return out, pos, nil
}

func (i *index) Write(off uint32, pos uint64) error {
	if uint64(len(i.mmap)) < i.size+entWidth {
		return io.EOF
	}

	enc.PutUint32(i.mmap[i.size:i.size+offWidth], off)
	enc.PutUint64(i.mmap[i.size+offWidth:i.size+entWidth], pos)

	i.size += entWidth

	return nil
}

func (i *index) Name() string {
	return i.file.Name()
}
