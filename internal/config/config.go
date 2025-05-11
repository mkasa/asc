package config

import (
	"os"
	"path/filepath"
)

// GetDataDir returns the path to the data directory
func GetDataDir() (string, error) {
	shareDir, err := GetShareDir()
	if err != nil {
		return "", err
	}
	dataDir := filepath.Join(shareDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}
	return dataDir, nil
}
