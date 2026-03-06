package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Linux: ~/.local/share/<appName>/
// Windows: %LOCALAPPDATA%\<appName>\
func AppDataDir(appName string) (string, error) {
	switch runtime.GOOS {
	case "linux":
		return linuxAppDir(appName)
	case "windows":
		return windowsAppDir(appName)
	default:
		return linuxAppDir(appName)
	}
}

func linuxAppDir(appName string) (string, error) {
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome != "" {
		return filepath.Join(xdgDataHome, appName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home dir: %w", err)
	}

	return filepath.Join(home, ".local", "share", appName), nil
}

func windowsAppDir(appName string) (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", fmt.Errorf("LOCALAPPDATA not set")
	}

	return filepath.Join(localAppData, appName), nil
}

func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
