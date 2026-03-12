package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	giouiApp "gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
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
	"vary/internal/selfupdate"
)

var version = "dev"

type windowDragRestoreState struct {
	pressed   bool
	restored  bool
	pointerID pointer.ID
	startPos  f32.Point
}

func main() {
	if handled, err := handleCLIFlags(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else if handled {
		return
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}

	state := varyApp.NewAppState(cfg)
	state.AppBuildVersion = version
	state.AppVersion = selfupdate.CurrentVersion(version)

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
			giouiApp.MinSize(unit.Dp(800), unit.Dp(800)),
		)

		w.Option(giouiApp.Size(unit.Dp(1100), unit.Dp(700)))

		fileExplorer = explorer.NewExplorer(&w)
		homeScreen.RequestRedraw = w.Invalidate
		settingsScreen = varyApp.NewSettingsScreen(fileExplorer)
		selectFileScreen := varyApp.NewSelectFileScreen(fileExplorer)

		go func() {
			result, err := selfupdate.Check(ctx, version)
			if err != nil {
				return
			}
			state.AppUpdateAvailable = result.UpdateAvailable
			state.AppPromptForUpdate = state.Config.AutoUpdate && result.UpdateAvailable
			if result.CurrentVersion != "" {
				state.AppVersion = result.CurrentVersion
			}
			w.Invalidate()
		}()

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
		var dragRestore windowDragRestoreState
		var dragRestoreTag struct{}
		windowMode := giouiApp.Windowed
		for {
			ev := w.Event()
			selectFileScreen.ListenEvent(ev)

			switch e := ev.(type) {
			case giouiApp.DestroyEvent:
				os.Exit(0)
			case giouiApp.ConfigEvent:
				windowMode = e.Config.Mode
			case giouiApp.FrameEvent:
				gtx := giouiApp.NewContext(&ops, e)
				prevScreen := state.CurrentScreen

				dragArea := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
				pass := pointer.PassOp{}.Push(gtx.Ops)
				event.Op(gtx.Ops, &dragRestoreTag)
				system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
				pass.Pop()
				dragArea.Pop()
				handleWindowDragRestore(gtx, &w, &dragRestoreTag, windowMode, &dragRestore)

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

func handleWindowDragRestore(gtx layout.Context, w *giouiApp.Window, tag event.Tag, mode giouiApp.WindowMode, state *windowDragRestoreState) {
	const restoreThreshold = 12

	for {
		ev, ok := gtx.Source.Event(pointer.Filter{
			Target: tag,
			Kinds:  pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel,
		})
		if !ok {
			return
		}

		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Kind {
		case pointer.Press:
			state.pressed = true
			state.restored = false
			state.pointerID = e.PointerID
			state.startPos = e.Position
		case pointer.Drag:
			if mode != giouiApp.Maximized || !state.pressed || state.restored || e.PointerID != state.pointerID {
				continue
			}
			if e.Position.Y-state.startPos.Y < float32(gtx.Dp(unit.Dp(restoreThreshold))) {
				continue
			}
			state.restored = true
			w.Perform(system.ActionUnmaximize | system.ActionMove)
		case pointer.Release, pointer.Cancel:
			if e.PointerID == state.pointerID {
				state.pressed = false
				state.restored = false
			}
		}
	}
}

func handleCLIFlags() (bool, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	checkUpdates := fs.Bool("check-updates", false, "check GitHub releases for a newer Vary binary and exit")
	selfUpdateFlag := fs.Bool("self-update", false, "download and replace this binary with the latest GitHub release, then exit")
	showVersion := fs.Bool("version", false, "print the current Vary version and exit")

	if err := fs.Parse(filteredArgs(os.Args[1:])); err != nil {
		return false, err
	}

	switch {
	case *showVersion:
		fmt.Println(selfupdate.CurrentVersion(version))
		return true, nil
	case *checkUpdates:
		result, err := selfupdate.Check(context.Background(), version)
		if err != nil {
			return true, err
		}

		fmt.Printf("Current version: %s\n", result.CurrentVersion)
		if result.LatestVersion == "" {
			fmt.Println("No compatible release asset was found for this platform.")
			return true, nil
		}

		fmt.Printf("Latest version: %s\n", result.LatestVersion)
		fmt.Printf("Asset: %s\n", result.AssetName)
		if result.ReleaseURL != "" {
			fmt.Printf("Release: %s\n", result.ReleaseURL)
		}
		if result.UpdateAvailable {
			fmt.Println("Update available.")
		} else {
			fmt.Println("Already up to date.")
		}
		return true, nil
	case *selfUpdateFlag:
		result, err := selfupdate.Apply(context.Background(), version)
		if err != nil {
			return true, err
		}
		if result.LatestVersion == "" {
			fmt.Println("No compatible release asset was found for this platform.")
			return true, nil
		}
		if !result.CurrentIsDev && !result.UpdateAvailable {
			fmt.Printf("Already up to date at %s.\n", result.CurrentVersion)
			return true, nil
		}

		fmt.Printf("Updated Vary from %s to %s using %s.\n", result.CurrentVersion, result.LatestVersion, result.AssetName)
		return true, nil
	default:
		return false, nil
	}
}

func filteredArgs(args []string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-psn_") {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}
