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

// MediaDir returns the cache directory for downloaded media, creating it.
func MediaDir() (string, error) {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".cache")
	}
	dir := filepath.Join(base, "yaawp", "media")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// AvatarDir returns the cache directory for profile pictures, creating it.
func AvatarDir() (string, error) {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".cache")
	}
	dir := filepath.Join(base, "yaawp", "avatars")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
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

// LockPath returns the single-instance lock file path, alongside the socket.
func LockPath() (string, error) {
	sock, err := SocketPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(sock), "daemon.lock"), nil
}

// StateDir returns the XDG state directory for yaawp, creating it if needed.
// It holds logs and other data that should persist but is not user content.
func StateDir() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	dir := filepath.Join(base, "yaawp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// LogPath returns the daemon log file path. The GUI launches the daemon
// detached, so its stdout is otherwise lost; writing here keeps a record for
// diagnosing connection and session problems.
func LogPath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.log"), nil
}
