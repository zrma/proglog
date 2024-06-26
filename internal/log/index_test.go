package log

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIndex(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "index_test")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(f.Name()))
	}()

	cfg := Config{}
	cfg.Segment.MaxIndexBytes = 1024

	idx, err := newIndex(f, cfg)
	require.NoError(t, err)

	_, _, err = idx.Read(-1)
	require.Error(t, err)
	require.Equal(t, io.EOF, err)
	require.Equal(t, f.Name(), idx.Name())

	tt := []struct {
		off uint32
		pos uint64
	}{
		{0, 0},
		{1, 10},
	}

	for _, want := range tt {
		err := idx.Write(want.off, want.pos)
		require.NoError(t, err)

		_, pos, err := idx.Read(int64(want.off))
		require.NoError(t, err)
		require.Equal(t, want.pos, pos)
	}

	_, _, err = idx.Read(int64(len(tt)))
	require.Error(t, err)
	require.Equal(t, io.EOF, err, "범위를 벗어난 오프셋을 읽을 때 에러가 발생")

	require.NoError(t, idx.Close())

	f, err = os.OpenFile(f.Name(), os.O_RDWR, 0600)
	require.NoError(t, err)

	cfg.Segment.MaxIndexBytes = entWidth * 3
	idx, err = newIndex(f, cfg)
	require.NoError(t, err)

	off, pos, err := idx.Read(-1)
	require.NoError(t, err)
	require.Equal(t, uint32(1), off)
	require.Equal(t, tt[1].pos, pos)

	cfg.Segment.MaxIndexBytes = entWidth * 2
	idx, err = newIndex(f, cfg)
	require.Error(t, err)
	require.Equal(t, ErrIndexMaxSizeExceeded, err)
}
