package log

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zrma/proglog/internal/pb"
)

func TestLog(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		f := newFixture(t)

		want := &pb.Record{
			Value: []byte("hello world"),
		}

		off, err := f.log.Append(want)
		require.NoError(t, err)
		require.Equal(t, uint64(0), off)

		got, err := f.log.Read(off)
		require.NoError(t, err)
		require.Equal(t, want.GetValue(), got.GetValue())
	})

	t.Run("Err/OffsetOutOfRange", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.log.Read(1)
		require.Error(t, err)
		require.Nil(t, got)

		var err0 pb.ErrOffsetOutOfRange
		require.True(t, errors.As(err, &err0))
		require.Equal(t, uint64(1), err0.Offset)
	})

	t.Run("OK/InitWithExisting", func(t *testing.T) {
		f := newFixture(t)

		want := &pb.Record{
			Value: []byte("hello world"),
		}

		for i := 0; i < 3; i++ {
			_, err := f.log.Append(want)
			require.NoError(t, err)
		}
		require.NoError(t, f.log.Close())

		lowest, err := f.log.LowestOffset()
		require.NoError(t, err)
		require.Equal(t, uint64(0), lowest)

		highest, err := f.log.HighestOffset()
		require.NoError(t, err)
		require.Equal(t, uint64(2), highest)

		log, err := NewLog(f.log.Dir, f.log.Config)
		require.NoError(t, err)

		lowest, err = log.LowestOffset()
		require.NoError(t, err)
		require.Equal(t, uint64(0), lowest)

		highest, err = log.HighestOffset()
		require.NoError(t, err)
		require.Equal(t, uint64(2), highest)
	})

	t.Run("OK/Reader", func(t *testing.T) {
		f := newFixture(t)

		want := &pb.Record{
			Value: []byte("hello world"),
		}

		off, err := f.log.Append(want)
		require.NoError(t, err)
		require.Equal(t, uint64(0), off)

		reader := f.log.Reader()
		b, err := io.ReadAll(reader)
		require.NoError(t, err)

		rawStr, err := proto.Marshal(want)
		require.NoError(t, err)

		size := enc.Uint64(b[:lenWidth])
		require.Equal(t, uint64(len(rawStr)), size)
		require.Len(t, b[lenWidth:], int(size))
		require.Equal(t, rawStr, b[lenWidth:])

		got := &pb.Record{}
		err = proto.Unmarshal(b[lenWidth:], got)
		require.NoError(t, err)
		require.Equal(t, want.GetValue(), got.GetValue())
	})

	t.Run("OK/Truncate", func(t *testing.T) {
		f := newFixture(t)

		want := &pb.Record{
			Value: []byte("hello world"),
		}

		for i := 0; i < 3; i++ {
			_, err := f.log.Append(want)
			require.NoError(t, err)
		}

		_, err := f.log.Read(0)
		require.NoError(t, err)

		lowest, err := f.log.LowestOffset()
		require.NoError(t, err)
		require.Equal(t, uint64(0), lowest)

		highest, err := f.log.HighestOffset()
		require.NoError(t, err)
		require.Equal(t, uint64(2), highest)

		err = f.log.Truncate(1)
		require.NoError(t, err)

		_, err = f.log.Read(0)
		require.Error(t, err)

		var err0 pb.ErrOffsetOutOfRange
		require.True(t, errors.As(err, &err0))
		require.Equal(t, uint64(0), err0.Offset)

		lowest, err = f.log.LowestOffset()
		require.NoError(t, err)
		require.Equal(t, uint64(2), lowest)

		highest, err = f.log.HighestOffset()
		require.NoError(t, err)
		require.Equal(t, uint64(2), highest)
	})
}

type fixture struct {
	log *Log
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	dir, err := os.MkdirTemp(os.TempDir(), "log-test")
	require.NoError(t, err)

	cfg := Config{}
	cfg.Segment.MaxStoreBytes = 32

	log, err := NewLog(dir, cfg)
	require.NoError(t, err)

	return &fixture{
		log: log,
	}
}
