package app

import (
	"vary/internal/config"
	"vary/internal/morphe"
)

type Screen int

const (
	ScreenHome Screen = iota
	ScreenSettings
	ScreenDownload
	ScreenPackages
	ScreenPatches
)

type AppState struct {
	CurrentScreen   Screen
	Config          *config.Config
	DeviceModel     string
	DeviceConnected bool

	IsDownloading    bool
	DownloadProgress float64
	DownloadStatus   string

	Packages         []string
	FilteredPackages []string
	SearchQuery      string
	Patches          []morphe.Patch
	SelectedPackage  string
	CLIPath          string
	PatchesPath      string
	IsLoadingPatches bool
	PatchStatus      string

	StatusMessage string
	StatusError   bool
}

func NewAppState(cfg *config.Config) *AppState {
	return &AppState{
		CurrentScreen:    ScreenHome,
		Config:           cfg,
		Packages:         make([]string, 0),
		FilteredPackages: make([]string, 0),
		Patches:          make([]morphe.Patch, 0),
		StatusMessage:    "Ready",
	}
}

func (s *AppState) SetScreen(screen Screen) {
	s.CurrentScreen = screen
}

func (s *AppState) SetStatus(msg string, isError bool) {
	s.StatusMessage = msg
	s.StatusError = isError
}

func (s *AppState) SetPackages(packages []string) {
	s.Packages = packages
	s.FilterPackages(s.SearchQuery)
}

func (s *AppState) SetPatches(patches []morphe.Patch) {
	s.Patches = patches
}

func (s *AppState) FilterPackages(query string) {
	s.SearchQuery = query
	if query == "" {
		s.FilteredPackages = make([]string, len(s.Packages))
		copy(s.FilteredPackages, s.Packages)
		return
	}

	filtered := make([]string, 0)
	lowerQuery := toLower(query)
	for _, pkg := range s.Packages {
		if contains(toLower(pkg), lowerQuery) {
			filtered = append(filtered, pkg)
		}
	}
	s.FilteredPackages = filtered
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
