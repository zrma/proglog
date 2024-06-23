package log

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zrma/proglog/pkg/pb"
)

func TestSegment(t *testing.T) {
	tempDir, err := os.MkdirTemp(os.TempDir(), "segment_test")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tempDir))
	}()

	want := &pb.Record{Value: []byte("foo bar baz")}

	c := Config{}
	c.Segment.MaxStoreBytes = 1024
	c.Segment.MaxIndexBytes = entWidth * 3

	s, err := newSegment(tempDir, 16, c)
	require.NoError(t, err)
	require.Equal(t, uint64(16), s.nextOffset)
	require.False(t, s.IsMaxed())

	for i := uint64(0); i < 3; i++ {
		off, err := s.Append(want)
		require.NoError(t, err)
		require.Equal(t, i+16, off)

		got, err := s.Read(off)
		require.NoError(t, err)
		require.Equal(t, want.GetValue(), got.GetValue())
	}

	_, err = s.Append(want)
	require.Equal(t, io.EOF, err)

	require.True(t, s.IsMaxed())

	c.Segment.MaxStoreBytes = uint64(len(want.GetValue())+lenWidth) * 4
	c.Segment.MaxIndexBytes = entWidth * 4

	s, err = newSegment(tempDir, 16, c)
	require.NoError(t, err)

	require.False(t, s.IsMaxed())

	c.Segment.MaxStoreBytes = uint64(len(want.GetValue())+lenWidth) * 3
	c.Segment.MaxIndexBytes = entWidth * 4

	s, err = newSegment(tempDir, 16, c)
	require.NoError(t, err)

	require.True(t, s.IsMaxed())

	err = s.Remove()
	require.NoError(t, err)

	s, err = newSegment(tempDir, 16, c)
	require.NoError(t, err)

	require.False(t, s.IsMaxed())
}
