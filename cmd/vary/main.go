package main

import (
	"context"
	"os"

	giouiApp "gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/explorer"

	"vary/internal/adb"
	varyApp "vary/internal/app"
	"vary/internal/config"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}

	state := varyApp.NewAppState(cfg)

	customTheme := varyApp.NewTheme()

	var fileExplorer *explorer.Explorer

	homeScreen := varyApp.NewHomeScreen()
	settingsScreen := varyApp.NewSettingsScreen(fileExplorer)
	downloadScreen := varyApp.NewDownloadScreen()
	packagesScreen := varyApp.NewPackagesScreen()
	patchesScreen := varyApp.NewPatchesScreen()
	patchLogsScreen := varyApp.NewPatchLogsScreen()

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var w giouiApp.Window
		w.Option(
			giouiApp.Title("Vary"),
			giouiApp.Decorated(false),
			giouiApp.MaxSize(unit.Dp(1280), unit.Dp(1024)),
			giouiApp.MinSize(unit.Dp(800), unit.Dp(600)),
		)

		w.Option(giouiApp.Size(unit.Dp(1100), unit.Dp(700)))

		fileExplorer = explorer.NewExplorer(&w)
		settingsScreen = varyApp.NewSettingsScreen(fileExplorer)
		selectFileScreen := varyApp.NewSelectFileScreen(fileExplorer)

		go adb.WatchFirstDeviceModel(ctx, func(model string) {
			if model != "" {
				state.DeviceModel = model
				state.DeviceConnected = true
			} else {
				state.DeviceConnected = false
			}
			w.Invalidate()
		})

		var ops op.Ops
		for {
			ev := w.Event()
			selectFileScreen.ListenEvent(ev)

			switch e := ev.(type) {
			case giouiApp.DestroyEvent:
				os.Exit(0)
			case giouiApp.FrameEvent:
				gtx := giouiApp.NewContext(&ops, e)
				prevScreen := state.CurrentScreen

				dragArea := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
				system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
				dragArea.Pop()

				paint.Fill(gtx.Ops, customTheme.Background)

				layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						switch state.CurrentScreen {
						case varyApp.ScreenHome:
							homeScreen.HandleInput(gtx, state)
							dims := homeScreen.Layout(gtx, customTheme, state)
							return dims
						case varyApp.ScreenSettings:
							settingsScreen.HandleInput(gtx, state)
							dims := settingsScreen.Layout(gtx, customTheme, state)
							return dims
						case varyApp.ScreenDownload:
							downloadScreen.HandleInput(gtx, state)
							dims := downloadScreen.Layout(gtx, customTheme, state)
							return dims
						case varyApp.ScreenPackages:
							packagesScreen.HandleInput(gtx, state)
							dims := packagesScreen.Layout(gtx, customTheme, state)
							return dims
						case varyApp.ScreenPatches:
							patchesScreen.HandleInput(gtx, state)
							dims := patchesScreen.Layout(gtx, customTheme, state)
							return dims
						case varyApp.ScreenSelectFile:
							selectFileScreen.HandleInput(gtx, state)
							dims := selectFileScreen.Layout(gtx, customTheme, state)
							return dims
						case varyApp.ScreenPatchLogs:
							patchLogsScreen.HandleInput(gtx, state)
							dims := patchLogsScreen.Layout(gtx, customTheme, state)
							return dims
						default:
							homeScreen.HandleInput(gtx, state)
							dims := homeScreen.Layout(gtx, customTheme, state)
							return dims
						}
					}),
				)

				if state.CurrentScreen == varyApp.ScreenDownload && prevScreen != varyApp.ScreenDownload && !state.IsDownloading {
					downloadScreen.StartDownload(state)
				}
				if state.CurrentScreen == varyApp.ScreenPatches && prevScreen != varyApp.ScreenPatches && !state.IsLoadingPatches {
					patchesScreen.StartLoadPatches(state)
				}
				if state.CurrentScreen == varyApp.ScreenSelectFile && prevScreen != varyApp.ScreenSelectFile && !state.IsLoadingVersions {
					selectFileScreen.StartLoadVersions(state)
				}
				if state.CurrentScreen == varyApp.ScreenPatchLogs && prevScreen != varyApp.ScreenPatchLogs && state.StartPatchRequested {
					patchLogsScreen.StartPatch(state)
				}

				if state.CurrentScreen != prevScreen {
					w.Invalidate()
				}
				if state.CurrentScreen == varyApp.ScreenDownload {
					w.Invalidate()
				}
				if state.CurrentScreen == varyApp.ScreenPatches {
					w.Invalidate()
				}
				if state.CurrentScreen == varyApp.ScreenSelectFile && state.IsLoadingVersions {
					w.Invalidate()
				}
				if state.CurrentScreen == varyApp.ScreenPatchLogs {
					w.Invalidate()
				}

				e.Frame(gtx.Ops)
			}
		}
	}()

	giouiApp.Main()
}
