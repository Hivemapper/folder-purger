// Package config reads folder size limits from the on-device config files, the
// same way map-ai's utils/DashcamConfig.py reads its own keys.
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	ConfigPath     = "/data/config/config.json"
	UserConfigPath = "/data/config/user_config.json"

	limitsKey = "FOLDER_PURGER_LIMITS"
)

type FolderLimit struct {
	Path       string `json:"path"`
	LimitBytes int64  `json:"limit_bytes"`
}

type configFile struct {
	Limits []FolderLimit `json:"FOLDER_PURGER_LIMITS"`
}

type userConfigFile struct {
	Overrides configFile `json:"overrides"`
}

// FolderLimits returns limits keyed by cleaned path, preferring the user
// config's overrides over the device config.
func FolderLimits() (map[string]int64, error) {
	return folderLimits(UserConfigPath, ConfigPath)
}

func folderLimits(userConfigPath, configPath string) (map[string]int64, error) {
	limits, err := readUserConfigLimits(userConfigPath)
	if err != nil {
		log.Printf("reading %s: %v", userConfigPath, err)
	} else if len(limits) > 0 {
		return limits, nil
	}

	return readConfigLimits(configPath)
}

func readUserConfigLimits(path string) (map[string]int64, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var parsed userConfigFile
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return limitsByPath(parsed.Overrides.Limits), nil
}

func readConfigLimits(path string) (map[string]int64, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var parsed configFile
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return limitsByPath(parsed.Limits), nil
}

func limitsByPath(entries []FolderLimit) map[string]int64 {
	limits := make(map[string]int64, len(entries))
	for _, entry := range entries {
		if entry.Path == "" {
			log.Printf("%s: skipping entry with empty path", limitsKey)
			continue
		}
		// A zero or negative limit would purge the folder on every sweep.
		if entry.LimitBytes <= 0 {
			log.Printf("%s: skipping %s, limit_bytes must be positive (got %d)", limitsKey, entry.Path, entry.LimitBytes)
			continue
		}
		limits[filepath.Clean(entry.Path)] = entry.LimitBytes
	}
	return limits
}
