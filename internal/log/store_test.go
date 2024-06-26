package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

var (
	write = []byte("foo bar baz")
	width = uint64(len(write)) + lenWidth
)

func TestStore_AppendRead(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "store_append_read_test")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.Remove(f.Name()))
	}()

	{
		s, err := newStore(f)
		require.NoError(t, err)

		testAppend(t, s)
		testRead(t, s)
		testReadAt(t, s)
	}

	{
		s, err := newStore(f)
		require.NoError(t, err)
		testRead(t, s)
	}
}

func testAppend(t *testing.T, s *store) {
	t.Helper()

	for i := uint64(1); i < 4; i++ {
		n, pos, err := s.Append(write)
		require.NoError(t, err)
		require.Equal(t, width, n)
		require.Equal(t, (i-1)*width, pos)
		require.Equal(t, width*i, pos+n)
	}
}

func testRead(t *testing.T, s *store) {
	t.Helper()

	var pos uint64
	for i := uint64(1); i < 4; i++ {
		p, err := s.Read(pos)
		require.NoError(t, err)
		require.Equal(t, write, p)
		pos += width
	}
}

func testReadAt(t *testing.T, s *store) {
	t.Helper()

	for i, off := 1, int64(0); i < 4; i++ {
		b := make([]byte, lenWidth)
		n, err := s.ReadAt(b, off)
		require.NoError(t, err)
		require.Equal(t, lenWidth, n)
		off += int64(n)

		size := enc.Uint64(b)
		b = make([]byte, size)
		n, err = s.ReadAt(b, off)
		require.NoError(t, err)
		require.Equal(t, write, b)
		require.Equal(t, int(size), n)
		require.Equal(t, len(write), n)
		off += int64(n)
	}
}

func TestStore_Close(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "store_close_test")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.Remove(f.Name()))
	}()

	s, err := newStore(f)
	require.NoError(t, err)

	_, _, err = s.Append(write)
	require.NoError(t, err)

	_, beforeSize, err := openFile(f.Name())
	require.NoError(t, err)

	require.NoError(t, s.Close())

	_, afterSize, err := openFile(f.Name())
	require.NoError(t, err)
	require.Greater(t, afterSize, beforeSize)

	t.Logf("beforeSize: %d, afterSize: %d", beforeSize, afterSize)
}

func openFile(name string) (file *os.File, size uint64, err error) {
	f, err := os.OpenFile(
		name,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		err = multierr.Append(err, f.Close())
	}()

	fi, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	return f, uint64(fi.Size()), nil
}
