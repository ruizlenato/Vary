package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vary/internal/storage"
)

type patchSelectionState struct {
	PackageName     string   `json:"packageName"`
	SelectedPatches []string `json:"selectedPatches"`
	UpdatedAt       string   `json:"updatedAt"`
}

func SavePatchSelection(packageName string, selected []string) error {
	stateDir, err := packagePatchStateDir(packageName)
	if err != nil {
		return err
	}
	if err := storage.EnsureDir(stateDir); err != nil {
		return err
	}

	unique := uniqueNonEmpty(selected)
	payload := patchSelectionState{
		PackageName:     packageName,
		SelectedPatches: unique,
		UpdatedAt:       time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(stateDir, "state.json"), data, 0644)
}

func LoadPatchSelection(packageName string) ([]string, error) {
	stateDir, err := packagePatchStateDir(packageName)
	if err != nil {
		return nil, err
	}

	statePath := filepath.Join(stateDir, "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var payload patchSelectionState
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	return uniqueNonEmpty(payload.SelectedPatches), nil
}

func packagePatchStateDir(packageName string) (string, error) {
	appDir, err := storage.AppDataDir("vary")
	if err != nil {
		return "", err
	}

	safePackage := sanitizePackageName(packageName)
	if safePackage == "" {
		safePackage = "unknown"
	}

	return filepath.Join(appDir, "patch-state", safePackage), nil
}

func sanitizePackageName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, string(filepath.Separator), "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	return name
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		result = append(result, v)
	}
	return result
}
