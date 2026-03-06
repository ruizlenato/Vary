package downloader

import (
	"encoding/json"
	"fmt"
	"os"
)

type State struct {
	TagName      string `json:"tag_name"`
	AssetName    string `json:"asset_name"`
	DownloadedAt string `json:"downloaded_at"`
}

func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func SaveState(path string, state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
