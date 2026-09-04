package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFoldersForReplacesArgs(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	require.NoError(t, os.MkdirAll(a, 0755))

	folders := foldersFor(map[string]int64{b: 2, a: 1})

	require.Len(t, folders, 2)
	require.Equal(t, a, folders[0].Path, "sorted by path")
	require.Equal(t, int64(1), folders[0].MaxSize)
	require.Equal(t, b, folders[1].Path)
	require.Equal(t, int64(2), folders[1].MaxSize)
	require.DirExists(t, b, "missing config folders are created")
}

func TestFoldersForNoLimitsFallsBackToArgs(t *testing.T) {
	require.Nil(t, foldersFor(nil))
}

func TestArgFolders(t *testing.T) {
	dir := t.TempDir()
	folders := argFolders([]string{dir + "/", "4096"})

	require.Len(t, folders, 1)
	require.Equal(t, dir, folders[0].Path, "trailing slash stripped")
	require.Equal(t, int64(4096), folders[0].MaxSize)
}

func TestArgFoldersCreatesMissingFolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new")
	folders := argFolders([]string{path, "4096"})

	require.Len(t, folders, 1)
	require.DirExists(t, path)
}

func TestArgFoldersPanicsOnBadSize(t *testing.T) {
	require.PanicsWithValue(t,
		fmt.Sprintf("Failed to parse destination max size: %s", "not-a-size"),
		func() { argFolders([]string{t.TempDir(), "not-a-size"}) })
}
