package morphe

import (
	"os"
	"path/filepath"
)

func FindLatestCLI(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var latest string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".jar" && len(name) > len(latest) {
			latest = name
		}
	}

	if latest == "" {
		return "", os.ErrNotExist
	}

	return filepath.Join(dir, latest), nil
}

func FindLatestPatches(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var latest string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".mpp" && len(name) > len(latest) {
			latest = name
		}
	}

	if latest == "" {
		return "", os.ErrNotExist
	}

	return filepath.Join(dir, latest), nil
}
