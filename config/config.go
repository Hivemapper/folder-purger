// Package config reads folder size limits from the on-device configurator.
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

const (
	DefaultBaseURL = "http://127.0.0.1:8091"
	limitsKey      = "FOLDER_PURGER_LIMITS"

	fetchTimeout  = 3 * time.Second
	fetchAttempts = 5
	retryDelay    = 10 * time.Second
)

type FolderLimit struct {
	Path       string `json:"path"`
	LimitBytes int64  `json:"limit_bytes"`
}

type configSnapshot struct {
	FolderPurgerLimits []FolderLimit `json:"FOLDER_PURGER_LIMITS"`
}

// FetchFolderLimits returns limits keyed by cleaned path. The configurator is
// not up yet on a cold boot, so failures are retried before giving up.
func FetchFolderLimits(baseURL string) (map[string]int64, error) {
	var lastErr error
	for attempt := 1; attempt <= fetchAttempts; attempt++ {
		limits, err := fetchOnce(baseURL)
		if err == nil {
			return limits, nil
		}
		lastErr = err
		log.Printf("config fetch attempt %d/%d failed: %v", attempt, fetchAttempts, err)
		if attempt < fetchAttempts {
			time.Sleep(retryDelay)
		}
	}
	return nil, lastErr
}

func fetchOnce(baseURL string) (map[string]int64, error) {
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(baseURL + "/config")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("configurator returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return ParseFolderLimits(body)
}

func ParseFolderLimits(body []byte) (map[string]int64, error) {
	var cfg configSnapshot
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	limits := make(map[string]int64, len(cfg.FolderPurgerLimits))
	for _, entry := range cfg.FolderPurgerLimits {
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

	return limits, nil
}
