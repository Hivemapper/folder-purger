package purger

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPurge(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < 10; i++ {
		sub := filepath.Join(dir, fmt.Sprintf("fkm_%04d", i))
		require.NoError(t, os.MkdirAll(sub, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(sub, "image.jpg"), make([]byte, 1024), 0644))
	}

	f := &Folder{Path: dir, MaxSize: 1024 * 5}
	require.NoError(t, f.CheckAndPurge())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	// 80% of 10 = 8 removed, 2 remain
	require.Equal(t, 2, len(entries))
	require.Equal(t, "fkm_0008", entries[0].Name())
	require.Equal(t, "fkm_0009", entries[1].Name())
}

func TestNoPurge(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < 5; i++ {
		sub := filepath.Join(dir, fmt.Sprintf("fkm_%04d", i))
		require.NoError(t, os.MkdirAll(sub, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(sub, "image.jpg"), make([]byte, 1024), 0644))
	}

	f := &Folder{Path: dir, MaxSize: 1024 * 1024}
	require.NoError(t, f.CheckAndPurge())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))
}

func TestEmptyDir(t *testing.T) {
	dir := t.TempDir()

	f := &Folder{Path: dir, MaxSize: 1024}
	require.NoError(t, f.CheckAndPurge())
}

func TestPurgeWithNestedFiles(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < 4; i++ {
		sub := filepath.Join(dir, fmt.Sprintf("fkm_%04d", i))
		require.NoError(t, os.MkdirAll(sub, 0755))
		for j := 0; j < 3; j++ {
			require.NoError(t, os.WriteFile(
				filepath.Join(sub, fmt.Sprintf("img_%d.jpg", j)),
				make([]byte, 1024),
				0644,
			))
		}
	}

	// 4 dirs * 3 files * 1024 = 12288 bytes, maxSize = 6000 triggers purge
	f := &Folder{Path: dir, MaxSize: 6000}
	require.NoError(t, f.CheckAndPurge())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	// 80% of 4 = 3.2 → 3 removed, 1 remains
	require.Equal(t, 1, len(entries))
	require.Equal(t, "fkm_0003", entries[0].Name())
}
