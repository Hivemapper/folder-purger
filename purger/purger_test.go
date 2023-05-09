package purger

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFileExist(t *testing.T) {
	e := fileExists("purger.go")
	require.Equal(t, true, e)

	e = fileExists("foo.bar")
	require.Equal(t, false, e)
}

func TestAll(t *testing.T) {

	err := os.RemoveAll("/tmp/dest")
	require.NoError(t, err)

	folders := map[string]*Folder{}
	folders["/tmp/dest"] = NewFolder("/tmp/dest", 1024*1024*100)

	p := NewPurger(folders)

	go func() {
		err := p.Purge()
		require.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)
	for i := 0; i < 101; i++ {
		err := generateFiles(t, fmt.Sprintf("/tmp/dest/%04d.jpg", i+1), 1024*1024)
		require.NoError(t, err)
	}

	time.Sleep(5 * time.Second)
}

func TestPreExistingFiles(t *testing.T) {
	err := os.RemoveAll("/tmp/dest")
	require.NoError(t, err)

	err = os.MkdirAll("/tmp/dest", 0755)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		err := generateFiles(t, fmt.Sprintf("/tmp/dest/%04d.jpg", i+1), 1024)
		require.NoError(t, err)
	}

	folders := map[string]*Folder{}
	folders["/tmp/dest"] = NewFolder("/tmp/dest", 1024*11)

	p := NewPurger(folders)

	go func() {
		err := p.Purge()
		require.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)

	files, err := os.ReadDir("/tmp/dest")
	require.NoError(t, err)

	require.Equal(t, 10, len(files))
}

func generateFiles(t *testing.T, file string, size int) error {
	t.Helper()
	err := os.WriteFile(file, make([]byte, size), 0644)
	if err != nil {
		return fmt.Errorf("generating file: %w", err)
	}
	return nil
}
