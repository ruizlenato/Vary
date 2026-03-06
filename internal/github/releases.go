package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	CLITRepo    = "MorpheApp/morphe-cli"
	PatchesRepo = "MorpheApp/morphe-patches"
)

type Release struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	PreRelease  bool    `json:"prerelease"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type ReleaseInfo struct {
	TagName   string
	AssetName string
	AssetURL  string
	AssetSize int64
	IsDev     bool
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) GetLatestRelease(repo string, devMode bool) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return nil, fmt.Errorf("GitHub API rate limit exceeded. Please try again later")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}

	if devMode {

		for _, r := range releases {
			if r.PreRelease {
				return &r, nil
			}
		}

	}

	for _, r := range releases {
		if !r.PreRelease {
			return &r, nil
		}
	}

	return &releases[0], nil
}

func (c *Client) GetCLIRelease(devMode bool) (*ReleaseInfo, error) {
	release, err := c.GetLatestRelease(CLITRepo, devMode)
	if err != nil {
		return nil, err
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, "-all.jar") {
			return &ReleaseInfo{
				TagName:   release.TagName,
				AssetName: asset.Name,
				AssetURL:  asset.BrowserDownloadURL,
				AssetSize: asset.Size,
				IsDev:     release.PreRelease,
			}, nil
		}
	}

	return nil, fmt.Errorf("no -all.jar asset found in release %s", release.TagName)
}

func (c *Client) GetPatchesRelease(devMode bool) (*ReleaseInfo, error) {
	release, err := c.GetLatestRelease(PatchesRepo, devMode)
	if err != nil {
		return nil, err
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".mpp") {
			return &ReleaseInfo{
				TagName:   release.TagName,
				AssetName: asset.Name,
				AssetURL:  asset.BrowserDownloadURL,
				AssetSize: asset.Size,
				IsDev:     release.PreRelease,
			}, nil
		}
	}

	return nil, fmt.Errorf("no .mpp asset found in release %s", release.TagName)
}
