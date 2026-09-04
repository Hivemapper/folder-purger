package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0644))
	return path
}

func TestFolderLimitsFromDeviceConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config.json", `{
		"isUnity": true,
		"FOLDER_PURGER_LIMITS": [
			{"path": "/data/recording/unprocessed_framekm", "limit_bytes": 8000000000},
			{"path": "/data/video/", "limit_bytes": 30000000000}
		]
	}`)

	limits, err := folderLimits(filepath.Join(dir, "missing.json"), cfg)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{
		"/data/recording/unprocessed_framekm": 8000000000,
		"/data/video":                         30000000000,
	}, limits)
}

func TestFolderLimitsUserOverridesWin(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config.json",
		`{"FOLDER_PURGER_LIMITS": [{"path": "/data/video", "limit_bytes": 1}]}`)
	userCfg := writeFile(t, dir, "user_config.json",
		`{"overrides": {"FOLDER_PURGER_LIMITS": [{"path": "/data/video", "limit_bytes": 2}]}}`)

	limits, err := folderLimits(userCfg, cfg)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"/data/video": 2}, limits)
}

func TestFolderLimitsFallsThroughWhenOverrideAbsent(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config.json",
		`{"FOLDER_PURGER_LIMITS": [{"path": "/data/video", "limit_bytes": 1}]}`)
	userCfg := writeFile(t, dir, "user_config.json", `{"overrides": {"other": true}}`)

	limits, err := folderLimits(userCfg, cfg)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"/data/video": 1}, limits)
}

func TestFolderLimitsFallsThroughWhenUserConfigMalformed(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config.json",
		`{"FOLDER_PURGER_LIMITS": [{"path": "/data/video", "limit_bytes": 1}]}`)
	userCfg := writeFile(t, dir, "user_config.json", `not json`)

	limits, err := folderLimits(userCfg, cfg)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"/data/video": 1}, limits)
}

func TestFolderLimitsKeyAbsent(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config.json", `{"isUnity": true}`)

	limits, err := folderLimits(filepath.Join(dir, "missing.json"), cfg)
	require.NoError(t, err)
	require.Empty(t, limits)
}

func TestFolderLimitsNoFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := folderLimits(filepath.Join(dir, "a.json"), filepath.Join(dir, "b.json"))
	require.Error(t, err)
}

func TestFolderLimitsMalformedDeviceConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config.json", `not json`)

	_, err := folderLimits(filepath.Join(dir, "missing.json"), cfg)
	require.ErrorContains(t, err, "parsing")
}

func TestFolderLimitsSkipsInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config.json", `{"FOLDER_PURGER_LIMITS": [
		{"path": "/data/video", "limit_bytes": 0},
		{"path": "/data/other", "limit_bytes": -1},
		{"path": "", "limit_bytes": 100},
		{"path": "/data/keep", "limit_bytes": 100}
	]}`)

	limits, err := folderLimits(filepath.Join(dir, "missing.json"), cfg)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"/data/keep": 100}, limits)
}
