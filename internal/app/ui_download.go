package app

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"vary/internal/adb"
	"vary/internal/downloader"
	"vary/internal/github"
	"vary/internal/morphe"
	"vary/internal/storage"
)

type DownloadScreen struct {
	cancelBtn  widget.Clickable
	progress   float32
	cancelFunc context.CancelFunc
	closeBtn   widget.Clickable
	closeIcon  *widget.Icon
}

func NewDownloadScreen() *DownloadScreen {
	return &DownloadScreen{closeIcon: mustIcon(closeIconVG)}
}

func (d *DownloadScreen) StartDownload(state *AppState) {
	if state.IsDownloading {
		return
	}
	state.IsDownloading = true
	d.progress = 0
	state.DownloadProgress = 0
	state.DownloadStatus = "Starting..."

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelFunc = cancel

	go func() {
		fail := func(prefix string, err error) {
			msg := prefix + err.Error()
			state.SetStatus(msg, true)
			state.DownloadStatus = msg
		}

		setStage := func(status string, p float32) {
			state.DownloadStatus = status
			d.progress = p
			state.DownloadProgress = float64(p)
		}
		advanceStage := func(status string, p float32) {
			setStage(status, p)
			time.Sleep(120 * time.Millisecond)
		}

		defer func() {
			state.IsDownloading = false
			d.cancelFunc = nil
		}()

		advanceStage("Checking releases...", 0.05)

		appDir, err := storage.AppDataDir("Vary")
		if err != nil {
			fail("Error: ", err)
			return
		}

		if err := storage.EnsureDir(appDir); err != nil {
			fail("Error: ", err)
			return
		}

		if !adb.IsAvailable() {
			state.DownloadStatus = "Downloading Android Platform Tools..."
			progressCb := func(downloaded, total int64) {
				if total > 0 {
					raw := float32(downloaded) / float32(total)
					d.progress = 0.06 + (raw * 0.12)
					state.DownloadProgress = float64(d.progress)
				}
			}
			if err := adb.EnsurePlatformTools(progressCb); err != nil {
				fail("Platform tools error: ", err)
				return
			}
			advanceStage("Android Platform Tools ready", 0.18)
		}

		client := github.NewClient()
		devMode := state.Config.IsDev()

		advanceStage("Fetching morphe-cli...", 0.22)
		cliRelease, err := client.GetCLIRelease(devMode)
		if err != nil {
			fail("GitHub CLI error: ", err)
			return
		}

		advanceStage("Fetching morphe-patches...", 0.30)
		patchesRelease, err := client.GetPatchesRelease(devMode)
		if err != nil {
			fail("GitHub Patches error: ", err)
			return
		}

		cliPath := filepath.Join(appDir, cliRelease.AssetName)
		cliStatePath := filepath.Join(appDir, "cli_state.json")
		cliState, _ := downloader.LoadState(cliStatePath)

		if cliState == nil || cliState.TagName != cliRelease.TagName {
			state.DownloadStatus = fmt.Sprintf("Downloading %s...", cliRelease.AssetName)
			progressCb := func(downloaded, total int64) {
				if total > 0 {
					raw := float32(downloaded) / float32(total)
					d.progress = 0.25 + (raw * 0.30)
					state.DownloadProgress = float64(d.progress)
				}
			}

			err := downloader.Download(cliRelease.AssetURL, cliPath, progressCb)
			if err != nil {
				fail("CLI download error: ", err)
				return
			}

			newState := &downloader.State{
				TagName:      cliRelease.TagName,
				AssetName:    cliRelease.AssetName,
				DownloadedAt: time.Now().Format(time.RFC3339),
			}
			downloader.SaveState(cliStatePath, newState)
		} else {
			advanceStage("morphe-cli up to date", 0.55)
		}

		patchesPath := filepath.Join(appDir, patchesRelease.AssetName)
		patchesStatePath := filepath.Join(appDir, "patches_state.json")
		patchesState, _ := downloader.LoadState(patchesStatePath)

		if patchesState == nil || patchesState.TagName != patchesRelease.TagName {
			state.DownloadStatus = fmt.Sprintf("Downloading %s...", patchesRelease.AssetName)
			progressCb := func(downloaded, total int64) {
				if total > 0 {
					raw := float32(downloaded) / float32(total)
					d.progress = 0.55 + (raw * 0.30)
					state.DownloadProgress = float64(d.progress)
				}
			}

			err := downloader.Download(patchesRelease.AssetURL, patchesPath, progressCb)
			if err != nil {
				fail("Patches download error: ", err)
				return
			}

			newState := &downloader.State{
				TagName:      patchesRelease.TagName,
				AssetName:    patchesRelease.AssetName,
				DownloadedAt: time.Now().Format(time.RFC3339),
			}
			downloader.SaveState(patchesStatePath, newState)
		} else {
			advanceStage("morphe-patches up to date", 0.85)
		}

		advanceStage("Listing packages...", 0.92)
		executor := morphe.NewExecutor(cliPath, patchesPath)
		packages, err := executor.ListPackages(ctx)
		if err != nil {
			fail("Error: ", err)
			return
		}

		advanceStage("Done", 1.0)
		state.SetPackages(packages)
		state.SetStatus(fmt.Sprintf("%d packages found", len(packages)), false)
		state.SetScreen(ScreenPackages)
	}()
}

func (d *DownloadScreen) Layout(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	originalConstraints := gtx.Constraints

	layout.Stack{}.Layout(gtx,

		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Vertical,
					Alignment: layout.Middle,
					Spacing:   layout.SpaceEvenly,
				}.Layout(gtx,

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							title := material.H5(material.NewTheme(), "Updating")
							title.Color = th.Text
							return title.Layout(gtx)
						})
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							status := material.Body1(material.NewTheme(), state.DownloadStatus)
							status.Color = th.Text
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return status.Layout(gtx)
							})
						})
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Left:  unit.Dp(40),
							Right: unit.Dp(40),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							progressBar := material.ProgressBar(material.NewTheme(), d.progress)
							progressBar.Color = th.Primary
							return progressBar.Layout(gtx)
						})
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(material.NewTheme(), &d.cancelBtn, "Cancel")
							btn.Background = th.Surface
							btn.Color = th.Text
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return btn.Layout(gtx)
							})
						})
					}),
				)
			})
		}),

		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return layout.Inset{
				Top:   unit.Dp(38),
				Right: unit.Dp(38),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return d.closeBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if d.closeIcon == nil {
								return layout.Dimensions{}
							}
							return d.closeIcon.Layout(gtx, color.NRGBA{R: 227, G: 227, B: 227, A: 255})
						})
					})
				})
			})
		}),
	)

	return layout.Dimensions{Size: originalConstraints.Max}
}

func (d *DownloadScreen) HandleInput(gtx layout.Context, state *AppState) {
	for d.closeBtn.Clicked(gtx) {
		os.Exit(0)
	}
	for d.cancelBtn.Clicked(gtx) {
		if state.IsDownloading {
			if d.cancelFunc != nil {
				d.cancelFunc()
			}
			state.DownloadStatus = "Canceled"
			d.progress = 0
			state.DownloadProgress = 0
			state.IsDownloading = false
		}
		state.SetScreen(ScreenHome)
	}
}
