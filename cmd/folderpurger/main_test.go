package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"folder_purger/config"
	"folder_purger/purger"

	"github.com/stretchr/testify/require"
)

func limitsServer(t *testing.T, limits []config.FolderLimit) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(map[string]any{"FOLDER_PURGER_LIMITS": limits})
	require.NoError(t, err)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
}

func configuratorFoldersFrom(t *testing.T, srv *httptest.Server) []*purger.Folder {
	t.Helper()
	limits, err := config.FetchFolderLimits(srv.URL)
	require.NoError(t, err)
	return foldersFor(limits)
}

func TestConfiguratorFoldersReplacesArgs(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	require.NoError(t, os.MkdirAll(a, 0755))

	srv := limitsServer(t, []config.FolderLimit{
		{Path: b, LimitBytes: 2},
		{Path: a, LimitBytes: 1},
	})
	defer srv.Close()

	folders := configuratorFoldersFrom(t, srv)

	require.Len(t, folders, 2)
	require.Equal(t, a, folders[0].Path, "sorted by path")
	require.Equal(t, int64(1), folders[0].MaxSize)
	require.Equal(t, b, folders[1].Path)
	require.Equal(t, int64(2), folders[1].MaxSize)
	require.DirExists(t, b, "missing config folders are created")
}

func TestConfiguratorFoldersEmptyFallsBackToArgs(t *testing.T) {
	srv := limitsServer(t, nil)
	defer srv.Close()

	require.Nil(t, configuratorFoldersFrom(t, srv))
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
