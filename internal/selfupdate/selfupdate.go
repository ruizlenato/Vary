package selfupdate

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	updater "github.com/creativeprojects/go-selfupdate"
)

const (
	repositoryOwner = "ruizlenato"
	repositoryName  = "Vary"
	devVersion      = "dev"
	baseVersion     = "0.0.0"
)

type CheckResult struct {
	CurrentVersion  string
	CurrentIsDev    bool
	LatestVersion   string
	UpdateAvailable bool
	AssetName       string
	ReleaseURL      string
}

func CurrentVersion(buildVersion string) string {
	if version, ok := normalizeVersion(buildVersion); ok {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if version, ok := normalizeVersion(info.Main.Version); ok {
			return version
		}
	}

	return devVersion
}

func Check(ctx context.Context, buildVersion string) (*CheckResult, error) {
	current := CurrentVersion(buildVersion)
	up, err := newUpdater()
	if err != nil {
		return nil, err
	}

	release, found, err := up.DetectLatest(ctx, repositorySlug())
	if err != nil {
		return nil, fmt.Errorf("detect latest release: %w", err)
	}

	result := &CheckResult{
		CurrentVersion: current,
		CurrentIsDev:   current == devVersion,
	}

	if !found {
		return result, nil
	}

	result.LatestVersion = release.Version()
	result.AssetName = release.AssetName
	result.ReleaseURL = release.URL
	result.UpdateAvailable = !result.CurrentIsDev && release.GreaterThan(currentForCompare(current))

	return result, nil
}

func Apply(ctx context.Context, buildVersion string) (*CheckResult, error) {
	current := CurrentVersion(buildVersion)
	up, err := newUpdater()
	if err != nil {
		return nil, err
	}

	release, found, err := up.DetectLatest(ctx, repositorySlug())
	if err != nil {
		return nil, fmt.Errorf("detect latest release: %w", err)
	}
	if !found {
		return &CheckResult{
			CurrentVersion: current,
			CurrentIsDev:   current == devVersion,
		}, nil
	}

	result := &CheckResult{
		CurrentVersion: current,
		CurrentIsDev:   current == devVersion,
		LatestVersion:  release.Version(),
		AssetName:      release.AssetName,
		ReleaseURL:     release.URL,
	}

	if !result.CurrentIsDev && !release.GreaterThan(currentForCompare(current)) {
		return result, nil
	}

	if _, err := up.UpdateSelf(ctx, currentForCompare(current), repositorySlug()); err != nil {
		return nil, fmt.Errorf("apply self-update: %w", err)
	}

	result.UpdateAvailable = true
	return result, nil
}

func newUpdater() (*updater.Updater, error) {
	source, err := updater.NewGitHubSource(updater.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("create GitHub source: %w", err)
	}

	return updater.NewUpdater(updater.Config{
		Source: source,
		OS:     releaseOS(runtime.GOOS),
	})
}

func repositorySlug() updater.Repository {
	return updater.NewRepositorySlug(repositoryOwner, repositoryName)
}

func currentForCompare(version string) string {
	if version == devVersion {
		return baseVersion
	}
	return version
}

func releaseOS(goos string) string {
	switch goos {
	case "darwin":
		return "macos"
	default:
		return goos
	}
}

func normalizeVersion(version string) (string, bool) {
	version = strings.TrimSpace(version)
	if version == "" || version == "(devel)" {
		return "", false
	}

	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, ".")
	for i, r := range version {
		if (r < '0' || r > '9') && r != '.' {
			version = version[:i]
			break
		}
	}

	if version == "" {
		return "", false
	}

	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return "", false
	}

	return strings.Join(parts[:3], "."), true
}
