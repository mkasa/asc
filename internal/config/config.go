package config

import (
	"os"
	"path/filepath"
)

// GetDataDir returns the path to the data directory
func GetDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dataDir := filepath.Join(homeDir, ".asc", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}
	return dataDir, nil
}
