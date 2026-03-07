package app

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gioui.org/io/event"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"

	"vary/internal/adb"
	"vary/internal/morphe"
)

type SelectFileScreen struct {
	mui           *material.Theme
	list          widget.List
	backBtn       widget.Clickable
	closeBtn      widget.Clickable
	localBtn      widget.Clickable
	helpBtn       widget.Clickable
	backIcon      *widget.Icon
	closeIcon     *widget.Icon
	explorer      *explorer.Explorer
	pickResults   chan filePickResult
	isPicking     bool
	showHelpModal bool
	popupQuery    string
	popupOpenBtn  widget.Clickable
	popupCloseBtn widget.Clickable
	abiResults    chan string
}

type filePickResult struct {
	path   string
	err    error
	noPath bool
}

func NewSelectFileScreen(ex *explorer.Explorer) *SelectFileScreen {
	return &SelectFileScreen{
		mui:         material.NewTheme(),
		list:        widget.List{List: layout.List{Axis: layout.Vertical}},
		backIcon:    mustIcon(backArrowIconVG),
		closeIcon:   mustIcon(closeIconVG),
		explorer:    ex,
		pickResults: make(chan filePickResult, 1),
		abiResults:  make(chan string, 1),
	}
}

func (s *SelectFileScreen) ListenEvent(ev event.Event) {
	if s.explorer != nil {
		s.explorer.ListenEvents(ev)
	}
}

func (s *SelectFileScreen) StartLoadVersions(state *AppState) {
	if state.IsLoadingVersions {
		return
	}
	state.IsLoadingVersions = true
	state.VersionStatus = "Loading versions..."

	go func() {
		defer func() { state.IsLoadingVersions = false }()

		if state.CLIPath == "" || state.PatchesPath == "" {
			state.VersionStatus = "Missing morphe assets"
			state.SetCompatibleVersions(nil)
			state.SetStatus(state.VersionStatus, true)
			return
		}
		if state.SelectedPackage == "" {
			state.VersionStatus = "No package selected"
			state.SetCompatibleVersions(nil)
			state.SetStatus(state.VersionStatus, true)
			return
		}

		executor := morphe.NewExecutor(state.CLIPath, state.PatchesPath)
		versions, err := executor.ListCompatibleVersions(context.Background(), state.SelectedPackage)
		if err != nil {
			state.VersionStatus = "Failed to load versions"
			state.SetCompatibleVersions(nil)
			state.SetStatus("Version list error: "+err.Error(), true)
			return
		}

		state.SetCompatibleVersions(versions)
		state.VersionStatus = fmt.Sprintf("%d available versions", len(versions))
		state.SetStatus(state.VersionStatus, false)
	}()
}

func (s *SelectFileScreen) Layout(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	originalConstraints := gtx.Constraints

	layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H4(s.mui, "Select File")
						title.Color = th.Text
						return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, title.Layout)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						sub := material.Body2(s.mui, state.SelectedPackage)
						sub.Color = th.TextMuted
						return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, sub.Layout)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						selected := "No file selected"
						if state.SelectedInputFile != "" {
							selected = "Selected file: " + filepath.Base(state.SelectedInputFile)
						}
						label := material.Body2(s.mui, selected)
						label.Color = th.TextMuted
						return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, label.Layout)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						cardWidth := min(gtx.Constraints.Max.X-gtx.Dp(unit.Dp(80)), gtx.Dp(unit.Dp(760)))
						cardHeight := min(gtx.Constraints.Max.Y-gtx.Dp(unit.Dp(240)), gtx.Dp(unit.Dp(440)))
						if cardWidth < gtx.Dp(unit.Dp(340)) {
							cardWidth = gtx.Constraints.Max.X - gtx.Dp(unit.Dp(24))
						}
						if cardHeight < gtx.Dp(unit.Dp(240)) {
							cardHeight = gtx.Dp(unit.Dp(240))
						}

						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints = layout.Exact(image.Pt(cardWidth, cardHeight))
							return layoutOutlinedSurface(gtx, unit.Dp(8), color.NRGBA{R: 78, G: 78, B: 78, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(14), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											head := material.Body1(s.mui, "Available versions")
											head.Color = th.Text
											return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, head.Layout)
										}),
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											if state.IsLoadingVersions {
												msg := material.Body2(s.mui, "Loading versions...")
												msg.Color = th.TextMuted
												return layout.Center.Layout(gtx, msg.Layout)
											}
											if len(state.CompatibleVersions) == 0 {
												msg := material.Body2(s.mui, "No versions found")
												msg.Color = th.TextMuted
												return layout.Center.Layout(gtx, msg.Layout)
											}
											return material.List(s.mui, &s.list).Layout(gtx, len(state.CompatibleVersions), func(gtx layout.Context, i int) layout.Dimensions {
												row := material.Body2(s.mui, "- "+state.CompatibleVersions[i])
												row.Color = th.TextMuted
												return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, row.Layout)
											})
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											status := material.Body2(s.mui, state.VersionStatus)
											status.Color = th.TextMuted
											return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(10)}.Layout(gtx, status.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly, Alignment: layout.Middle}.Layout(gtx,
												layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
													return s.actionButton(gtx, th, &s.localBtn, "Select local file")
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Spacer{Width: unit.Dp(10)}.Layout(gtx)
												}),
												layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
													return s.actionButton(gtx, th, &s.helpBtn, "Help me download")
												}),
											)
										}),
									)
								})
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
						return s.backBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if s.backIcon == nil {
								return layout.Dimensions{}
							}
							return s.backIcon.Layout(gtx, color.NRGBA{R: 227, G: 227, B: 227, A: 255})
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
						return s.closeBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if s.closeIcon == nil {
								return layout.Dimensions{}
							}
							return s.closeIcon.Layout(gtx, color.NRGBA{R: 227, G: 227, B: 227, A: 255})
						})
					})
				})
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if !s.showHelpModal {
				return layout.Dimensions{}
			}
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				width := min(gtx.Constraints.Max.X-gtx.Dp(unit.Dp(60)), gtx.Dp(unit.Dp(700)))
				height := gtx.Dp(unit.Dp(230))
				if width < gtx.Dp(unit.Dp(320)) {
					width = gtx.Constraints.Max.X - gtx.Dp(unit.Dp(24))
				}
				gtx.Constraints = layout.Exact(image.Pt(width, height))
				return layoutOutlinedSurface(gtx, unit.Dp(8), color.NRGBA{R: 95, G: 95, B: 95, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(14), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								title := material.Body1(s.mui, "Search APKMirror")
								title.Color = th.Text
								return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, title.Layout)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								query := material.Body2(s.mui, s.popupQuery)
								query.Color = th.TextMuted
								return query.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return s.actionButton(gtx, th, &s.popupOpenBtn, "Search on DuckDuckGo")
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return layout.Spacer{Width: unit.Dp(10)}.Layout(gtx)
									}),
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return s.actionButton(gtx, th, &s.popupCloseBtn, "Close")
									}),
								)
							}),
						)
					})
				})
			})
		}),
	)

	return layout.Dimensions{Size: originalConstraints.Max}
}

func (s *SelectFileScreen) actionButton(gtx layout.Context, th *Theme, btn *widget.Clickable, label string) layout.Dimensions {
	height := gtx.Dp(unit.Dp(44))
	gtx.Constraints.Min.Y = height
	gtx.Constraints.Max.Y = height
	style := material.Button(s.mui, btn, label)
	style.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	style.Color = th.Text
	return layoutOutlinedSurface(gtx, unit.Dp(6), color.NRGBA{R: 120, G: 120, B: 120, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, style.Layout)
}

func (s *SelectFileScreen) HandleInput(gtx layout.Context, state *AppState) {
	select {
	case abi := <-s.abiResults:
		if s.showHelpModal {
			s.popupQuery = s.buildSearchQuery(state, abi)
		}
	default:
	}

	select {
	case result := <-s.pickResults:
		s.isPicking = false
		if result.err != nil {
			if errors.Is(result.err, explorer.ErrUserDecline) {
				state.SetStatus("File selection canceled", false)
				return
			}
			state.SetStatus("File picker error: "+result.err.Error(), true)
			return
		}
		if result.noPath || result.path == "" {
			state.SetStatus("File selected, but path is unavailable on this platform", true)
			return
		}

		state.SelectedInputFile = result.path
		state.SetStatus("Selected file: "+filepath.Base(result.path), false)
		state.SetScreen(ScreenPatches)
	default:
	}

	for s.closeBtn.Clicked(gtx) {
		os.Exit(0)
	}
	for s.backBtn.Clicked(gtx) {
		state.SetScreen(ScreenPackages)
	}
	for s.localBtn.Clicked(gtx) {
		if s.explorer == nil {
			state.SetStatus("File picker is unavailable", true)
			continue
		}
		if s.isPicking {
			continue
		}
		s.isPicking = true
		state.SetStatus("Opening file picker...", false)
		go func() {
			rc, err := s.explorer.ChooseFile(".apk", ".apkm", ".xapk")
			if err != nil {
				s.pickResults <- filePickResult{err: err}
				return
			}
			defer rc.Close()

			if f, ok := rc.(*os.File); ok {
				s.pickResults <- filePickResult{path: f.Name()}
				return
			}

			s.pickResults <- filePickResult{noPath: true}
		}()
	}
	for s.helpBtn.Clicked(gtx) {
		s.popupQuery = s.buildSearchQuery(state, "")
		s.showHelpModal = true
		go func() {
			abi, err := adb.FirstConnectedABI()
			if err != nil || abi == "" {
				return
			}
			s.abiResults <- abi
		}()
	}
	for s.popupCloseBtn.Clicked(gtx) {
		s.showHelpModal = false
	}
	for s.popupOpenBtn.Clicked(gtx) {
		if s.popupQuery == "" {
			continue
		}
		if err := openURL("https://www.google.com/search?q=" + url.QueryEscape(s.popupQuery)); err != nil {
			state.SetStatus("Failed to open browser: "+err.Error(), true)
		} else {
			state.SetStatus("Opened browser search", false)
			s.showHelpModal = false
		}
	}
}

func (s *SelectFileScreen) buildSearchQuery(state *AppState, abi string) string {
	parts := []string{"apkmirror"}
	if state.SelectedPackage != "" {
		parts = append(parts, state.SelectedPackage)
	}
	if len(state.CompatibleVersions) > 0 {
		parts = append(parts, state.CompatibleVersions[0])
	}
	parts = append(parts, "nodpi")
	if abi != "" {
		parts = append(parts, abi)
	}
	return strings.Join(parts, " ")
}

func openURL(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}
