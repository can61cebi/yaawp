// Package store resolves data and socket paths following the XDG base
// directory specification.
package store

import (
	"os"
	"path/filepath"
)

// DataDir returns the XDG data directory for yaawp, creating it if needed.
func DataDir() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(base, "yaawp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// DatabasePath returns the path to the whatsmeow SQLite session store.
func DatabasePath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.db"), nil
}

// SocketPath returns the Unix domain socket path used for IPC.
func SocketPath() (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "yaawp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.sock"), nil
}
