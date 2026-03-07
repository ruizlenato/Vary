package app

import (
	"context"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"vary/internal/adb"
	"vary/internal/morphe"
	"vary/internal/storage"
)

type PatchLogsScreen struct {
	mui          *material.Theme
	list         widget.List
	backBtn      widget.Clickable
	closeBtn     widget.Clickable
	copyBtn      widget.Clickable
	openDirBtn   widget.Clickable
	installBtn   widget.Clickable
	backIcon     *widget.Icon
	closeIcon    *widget.Icon
	lastLogCount int
}

func NewPatchLogsScreen() *PatchLogsScreen {
	return &PatchLogsScreen{
		mui:       material.NewTheme(),
		list:      widget.List{List: layout.List{Axis: layout.Vertical}},
		backIcon:  mustIcon(backArrowIconVG),
		closeIcon: mustIcon(closeIconVG),
	}
}

func (p *PatchLogsScreen) StartPatch(state *AppState) {
	if state.IsApplyingPatches {
		return
	}
	state.StartPatchRequested = false
	state.IsApplyingPatches = true
	state.PatchLogs = state.PatchLogs[:0]
	p.lastLogCount = 0
	state.PatchedOutputFile = ""
	state.AppendPatchLog("Starting patch process...")
	state.PatchStatus = "Patching app..."

	inputFile := state.SelectedInputFile
	startTime := time.Now()
	selected := make([]string, len(state.SelectedPatches))
	copy(selected, state.SelectedPatches)

	go func() {
		executor := morphe.NewExecutor(state.CLIPath, state.PatchesPath)
		err := executor.PatchAppWithLogs(context.Background(), inputFile, selected, state.Config.CustomKeystorePath, func(line string, isErr bool) {
			if isErr {
				state.AppendPatchLog("[ERR] " + line)
				return
			}
			state.AppendPatchLog(line)
		})

		state.IsApplyingPatches = false
		if err != nil {
			state.PatchStatus = "Patch failed"
			state.AppendPatchLog("[ERR] " + err.Error())
			state.SetStatus("Patch error: "+err.Error(), true)
			return
		}

		if out := detectPatchedOutput(inputFile, startTime); out != "" {
			state.PatchedOutputFile = out
			state.AppendPatchLog("Output file: " + out)
		}
		state.PatchStatus = "Patch completed"
		state.AppendPatchLog("Patch completed successfully")
		state.SetStatus("Patch completed", false)
	}()
}

func (p *PatchLogsScreen) Layout(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	if len(state.PatchLogs) > p.lastLogCount {
		if len(state.PatchLogs) > 0 {
			p.list.Position.First = len(state.PatchLogs) - 1
			p.list.Position.Offset = 0
		}
		p.lastLogCount = len(state.PatchLogs)
	}

	originalConstraints := gtx.Constraints

	layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H5(p.mui, "Patching Application")
						title.Color = th.Text
						return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, title.Layout)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						status := material.Body2(p.mui, state.PatchStatus)
						status.Color = th.TextMuted
						return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, status.Layout)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						cardWidth := min(gtx.Constraints.Max.X-gtx.Dp(unit.Dp(80)), gtx.Dp(unit.Dp(840)))
						cardHeight := min(gtx.Constraints.Max.Y-gtx.Dp(unit.Dp(260)), gtx.Dp(unit.Dp(460)))
						if cardWidth < gtx.Dp(unit.Dp(360)) {
							cardWidth = gtx.Constraints.Max.X - gtx.Dp(unit.Dp(24))
						}
						if cardHeight < gtx.Dp(unit.Dp(260)) {
							cardHeight = gtx.Dp(unit.Dp(260))
						}

						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints = layout.Exact(image.Pt(cardWidth, cardHeight))
							return layoutOutlinedSurface(gtx, unit.Dp(8), color.NRGBA{R: 78, G: 78, B: 78, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									if len(state.PatchLogs) == 0 {
										msg := material.Body2(p.mui, "Waiting for logs...")
										msg.Color = th.TextMuted
										return layout.Center.Layout(gtx, msg.Layout)
									}

									return material.List(p.mui, &p.list).Layout(gtx, len(state.PatchLogs), func(gtx layout.Context, index int) layout.Dimensions {
										line := material.Body2(p.mui, state.PatchLogs[index])
										line.Color = th.TextMuted
										return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, line.Layout)
									})
								})
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if state.IsApplyingPatches {
							return layout.Dimensions{}
						}
						return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							actionWidth := min(gtx.Constraints.Max.X-gtx.Dp(unit.Dp(80)), gtx.Dp(unit.Dp(840)))
							if actionWidth < gtx.Dp(unit.Dp(360)) {
								actionWidth = gtx.Constraints.Max.X - gtx.Dp(unit.Dp(24))
							}
							if actionWidth < gtx.Dp(unit.Dp(260)) {
								actionWidth = gtx.Dp(unit.Dp(260))
							}

							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints = layout.Exact(image.Pt(actionWidth, gtx.Dp(unit.Dp(44))))
								return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return p.actionButton(gtx, th, &p.copyBtn, "Copy logs")
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return layout.Spacer{Width: unit.Dp(10)}.Layout(gtx)
									}),
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return p.actionButton(gtx, th, &p.openDirBtn, "Open output folder")
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										if !state.DeviceConnected {
											return layout.Dimensions{}
										}
										return layout.Spacer{Width: unit.Dp(10)}.Layout(gtx)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										if !state.DeviceConnected {
											return layout.Dimensions{}
										}
										btnWidth := min(gtx.Dp(unit.Dp(220)), gtx.Constraints.Max.X)
										gtx.Constraints = layout.Exact(image.Pt(btnWidth, gtx.Dp(unit.Dp(44))))
										return p.actionButton(gtx, th, &p.installBtn, "Install on device")
									}),
								)
							})
						})
					}),
				)
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return layout.Inset{Top: unit.Dp(38), Left: unit.Dp(22)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if state.IsApplyingPatches {
							return layout.Dimensions{}
						}
						return p.backBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if p.backIcon == nil {
								return layout.Dimensions{}
							}
							return p.backIcon.Layout(gtx, color.NRGBA{R: 227, G: 227, B: 227, A: 255})
						})
					})
				})
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return layout.Inset{Top: unit.Dp(38), Right: unit.Dp(38)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return p.closeBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if p.closeIcon == nil {
								return layout.Dimensions{}
							}
							return p.closeIcon.Layout(gtx, color.NRGBA{R: 227, G: 227, B: 227, A: 255})
						})
					})
				})
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return layoutDeviceStatusBadge(gtx, state, p.mui)
		}),
	)

	return layout.Dimensions{Size: originalConstraints.Max}
}

func (p *PatchLogsScreen) HandleInput(gtx layout.Context, state *AppState) {
	for p.closeBtn.Clicked(gtx) {
		os.Exit(0)
	}
	for p.backBtn.Clicked(gtx) {
		if state.IsApplyingPatches {
			continue
		}
		state.SetScreen(ScreenPatches)
	}
	for p.copyBtn.Clicked(gtx) {
		if len(state.PatchLogs) == 0 {
			state.SetStatus("No logs to copy", true)
			continue
		}
		if err := copyTextToClipboard(strings.Join(state.PatchLogs, "\n")); err != nil {
			state.SetStatus("Failed to copy logs: "+err.Error(), true)
		} else {
			state.SetStatus("Logs copied to clipboard", false)
		}
	}
	for p.openDirBtn.Clicked(gtx) {
		base := state.PatchedOutputFile
		if base == "" {
			base = state.SelectedInputFile
		}
		if base == "" {
			state.SetStatus("No output folder available", true)
			continue
		}
		dir := filepath.Dir(base)
		if err := openPath(dir); err != nil {
			state.SetStatus("Failed to open folder: "+err.Error(), true)
		} else {
			state.SetStatus("Opened output folder", false)
		}
	}
	for p.installBtn.Clicked(gtx) {
		if !state.DeviceConnected {
			state.SetStatus("No connected adb device", true)
			continue
		}
		apkPath := state.PatchedOutputFile
		if apkPath == "" {
			apkPath = state.SelectedInputFile
		}
		if apkPath == "" {
			state.SetStatus("No APK file available to install", true)
			continue
		}
		state.SetStatus("Installing on device...", false)
		go func(path string) {
			out, err := adb.InstallAPKOnFirstDevice(path)
			if err != nil {
				state.AppendPatchLog("[ERR] install: " + err.Error())
				state.SetStatus("Install failed: "+err.Error(), true)
				return
			}
			if out != "" {
				state.AppendPatchLog("install: " + out)
			}
			state.SetStatus("Installed on device", false)
		}(apkPath)
	}
}

func (p *PatchLogsScreen) actionButton(gtx layout.Context, th *Theme, btn *widget.Clickable, label string) layout.Dimensions {
	height := gtx.Dp(unit.Dp(44))
	gtx.Constraints.Min.Y = height
	gtx.Constraints.Max.Y = height
	style := material.Button(p.mui, btn, label)
	style.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	style.Color = th.Text
	return layoutOutlinedSurface(gtx, unit.Dp(6), color.NRGBA{R: 120, G: 120, B: 120, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, style.Layout)
}

func detectPatchedOutput(inputFile string, since time.Time) string {
	searchDirs := make([]string, 0, 2)
	if inputFile != "" {
		searchDirs = append(searchDirs, filepath.Dir(inputFile))
	}
	if appDir, err := storage.AppDataDir("Vary"); err == nil {
		searchDirs = append(searchDirs, appDir)
	}

	inputBase := filepath.Base(inputFile)
	best := ""
	bestTime := time.Time{}

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".apk") {
				continue
			}
			if name == inputBase {
				continue
			}
			full := filepath.Join(dir, name)
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(since.Add(-2 * time.Second)) {
				continue
			}
			if best == "" || info.ModTime().After(bestTime) {
				best = full
				bestTime = info.ModTime()
			}
		}
	}

	return best
}

func openPath(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}

func copyTextToClipboard(text string) error {
	if text == "" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "clip")
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		cmd = exec.Command("sh", "-c", "(command -v wl-copy >/dev/null 2>&1 && wl-copy) || (command -v xclip >/dev/null 2>&1 && xclip -selection clipboard) || (command -v xsel >/dev/null 2>&1 && xsel --clipboard --input)")
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}
	_, writeErr := stdin.Write([]byte(text))
	closeErr := stdin.Close()
	waitErr := cmd.Wait()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}
