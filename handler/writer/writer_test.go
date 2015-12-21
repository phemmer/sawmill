package writer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/phemmer/sawmill/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppend(t *testing.T) {
	td, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(td)

	fp := filepath.Join(td, "TestAppend")
	wh, err := Append(fp, 0600, "")
	require.NoError(t, err)

	f, err := os.Open(fp)
	require.NoError(t, err)

	stat, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), stat.Mode())

	e := event.New(0, event.Info, "TestAppend message 1", nil, false)
	err = wh.Event(e)
	require.NoError(t, err)

	buf := make([]byte, 128)
	_, err = f.Read(buf)
	assert.NoError(t, err)
	assert.Contains(t, string(buf), "TestAppend message 1")

	// create a new appender and make sure it appends

	wh, err = Append(fp, 0600, "")
	require.NoError(t, err)

	_, err = f.Seek(0, 0)
	require.NoError(t, err)

	e = event.New(0, event.Info, "TestAppend message 2", nil, false)
	err = wh.Event(e)
	require.NoError(t, err)

	_, err = f.Read(buf)
	assert.NoError(t, err)
	assert.Contains(t, string(buf), "TestAppend message 1")
	assert.Contains(t, string(buf), "TestAppend message 2")
}
