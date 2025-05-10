package config

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

// GetShareDir returns the data directory path for ASC.
// It follows the XDG Base Directory Specification:
// - Uses XDG_DATA_HOME if set
// - Falls back to $HOME/.local/share
func GetShareDir() (string, error) {
	// Try XDG_DATA_HOME first
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		dir := filepath.Join(xdgDataHome, "asc")
		log.Debug("Using XDG_DATA_HOME directory", "path", dir)
		return dir, nil
	}

	// Fall back to $HOME/.local/share
	home, err := os.UserHomeDir()
	if err != nil {
		log.Error("Failed to get user home directory", "error", err)
		return "", err
	}

	dir := filepath.Join(home, ".local", "share", "asc")
	log.Debug("Using default data directory", "path", dir)
	return dir, nil
}

// EnsureShareDir creates the data directory if it doesn't exist.
func EnsureShareDir() error {
	dir, err := GetShareDir()
	if err != nil {
		log.Error("Failed to get share directory", "error", err)
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Error("Failed to create share directory", "path", dir, "error", err)
		return err
	}

	log.Debug("Share directory ensured", "path", dir)
	return nil
}
