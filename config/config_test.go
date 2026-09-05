package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFolderLimits(t *testing.T) {
	body := []byte(`{
		"VERSION": 42,
		"FOLDER_PURGER_LIMITS": [
			{"path": "/data/recording/unprocessed_framekm", "limit_bytes": 8000000000},
			{"path": "/data/video/", "limit_bytes": 30000000000}
		]
	}`)

	limits, err := ParseFolderLimits(body)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{
		"/data/recording/unprocessed_framekm": 8000000000,
		"/data/video":                         30000000000,
	}, limits)
}

func TestParseFolderLimitsKeyAbsent(t *testing.T) {
	limits, err := ParseFolderLimits([]byte(`{"VERSION": 42}`))
	require.NoError(t, err)
	require.Empty(t, limits)
}

func TestParseFolderLimitsSkipsInvalidEntries(t *testing.T) {
	body := []byte(`{"FOLDER_PURGER_LIMITS": [
		{"path": "/data/video", "limit_bytes": 0},
		{"path": "/data/other", "limit_bytes": -1},
		{"path": "", "limit_bytes": 100},
		{"path": "/data/keep", "limit_bytes": 100}
	]}`)

	limits, err := ParseFolderLimits(body)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"/data/keep": 100}, limits)
}

func TestParseFolderLimitsMalformed(t *testing.T) {
	_, err := ParseFolderLimits([]byte(`not json`))
	require.Error(t, err)
}

func TestFetchFolderLimits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/config", r.URL.Path)
		w.Write([]byte(`{"FOLDER_PURGER_LIMITS": [{"path": "/data/video", "limit_bytes": 123}]}`))
	}))
	defer srv.Close()

	limits, err := FetchFolderLimits(srv.URL)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"/data/video": 123}, limits)
}

func TestFetchFolderLimitsErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "config not yet available", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := fetchOnce(srv.URL)
	require.ErrorContains(t, err, "503")
}
