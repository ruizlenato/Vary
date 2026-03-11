package app

import (
	"context"
	"image"
	"image/color"
	"os"
	"os/exec"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"vary/internal/selfupdate"
)

type HomeScreen struct {
	startBtn          widget.Clickable
	settingsBtn       widget.Clickable
	closeBtn          widget.Clickable
	updateLinkBtn     widget.Clickable
	restartLinkBtn    widget.Clickable
	updateLaterBtn    widget.Clickable
	updateYesBtn      widget.Clickable
	closeIcon         *widget.Icon
	mui               *material.Theme
	OnStartClicked    func()
	OnSettingsClicked func()
	RequestRedraw     func()
	showUpdateModal   bool
	updateResults     chan appUpdateResult
}

type appUpdateResult struct {
	result *selfupdate.CheckResult
	err    error
}

func NewHomeScreen() *HomeScreen {
	return &HomeScreen{
		closeIcon:     mustIcon(closeIconVG),
		mui:           material.NewTheme(),
		updateResults: make(chan appUpdateResult, 1),
	}
}

func (h *HomeScreen) Layout(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
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
						title := material.H2(h.mui, "Vary")
						title.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
						return layout.Inset{Bottom: unit.Dp(32)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return title.Layout(gtx)
						})
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis:      layout.Horizontal,
								Alignment: layout.Middle,
								Spacing:   layout.SpaceEvenly,
							}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return h.button(gtx, "Start", &h.startBtn)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return h.button(gtx, "Settings", &h.settingsBtn)
								}),
							)
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
						return h.closeBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if h.closeIcon == nil {
								return layout.Dimensions{}
							}
							return h.closeIcon.Layout(gtx, color.NRGBA{R: 227, G: 227, B: 227, A: 255})
						})
					})
				})
			})
		}),

		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return layoutDeviceStatusBadge(gtx, state, h.mui)
		}),

		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return h.layoutAppVersionBadge(gtx, th, state)
		}),

		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if !h.showUpdateModal {
				return layout.Dimensions{}
			}
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return h.layoutUpdateModal(gtx, th, state)
		}),
	)

	return layout.Dimensions{Size: originalConstraints.Max}
}

func (h *HomeScreen) button(gtx layout.Context, text string, btn *widget.Clickable) layout.Dimensions {
	buttonWidth := gtx.Dp(unit.Dp(150))
	buttonHeight := gtx.Dp(unit.Dp(50))
	borderColor := color.NRGBA{R: 200, G: 200, B: 200, A: 255}

	dims := layout.Inset{
		Top:    unit.Dp(8),
		Bottom: unit.Dp(8),
		Left:   unit.Dp(15),
		Right:  unit.Dp(15),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Exact(image.Pt(buttonWidth, buttonHeight))
		return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			borderInset := gtx.Dp(unit.Dp(2))
			cornerRadius := gtx.Dp(unit.Dp(0))
			outerRRect := clip.UniformRRect(image.Rect(0, 0, buttonWidth, buttonHeight), cornerRadius)
			paint.FillShape(gtx.Ops, borderColor, outerRRect.Op(gtx.Ops))
			innerRadius := max(cornerRadius-borderInset, 0)
			innerRect := image.Rect(borderInset, borderInset, buttonWidth-borderInset, buttonHeight-borderInset)
			innerRRect := clip.UniformRRect(innerRect, innerRadius)
			paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, innerRRect.Op(gtx.Ops))
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Label(h.mui, unit.Sp(15), text)
				label.Color = borderColor
				return label.Layout(gtx)
			})
		})
	})

	return dims
}

func (h *HomeScreen) HandleInput(gtx layout.Context, state *AppState) {
	if state.AppPromptForUpdate && state.AppUpdateAvailable && !state.AppIsUpdating {
		h.startUpdate(state)
		state.AppPromptForUpdate = false
	}

	select {
	case update := <-h.updateResults:
		state.AppIsUpdating = false
		state.AppPromptForUpdate = false
		h.showUpdateModal = false
		if update.err != nil {
			state.SetStatus("Failed to update app: "+update.err.Error(), true)
			if h.RequestRedraw != nil {
				h.RequestRedraw()
			}
			return
		}
		if update.result != nil {
			state.AppUpdateAvailable = false
		}
		state.AppRestartRequired = true
		state.SetStatus("Restart Vary to use the new version.", false)
		if h.RequestRedraw != nil {
			h.RequestRedraw()
		}
	default:
	}

	for h.closeBtn.Clicked(gtx) {
		os.Exit(0)
	}
	for h.startBtn.Clicked(gtx) {
		state.SetScreen(ScreenDownload)
	}
	for h.settingsBtn.Clicked(gtx) {
		state.SetScreen(ScreenSettings)
	}
	for h.updateLinkBtn.Clicked(gtx) {
		if state.AppUpdateAvailable && !state.AppIsUpdating {
			h.showUpdateModal = true
		}
	}
	for h.restartLinkBtn.Clicked(gtx) {
		if err := restartApp(); err != nil {
			state.SetStatus("Failed to restart app: "+err.Error(), true)
			continue
		}
		os.Exit(0)
	}
	for h.updateLaterBtn.Clicked(gtx) {
		h.showUpdateModal = false
	}
	for h.updateYesBtn.Clicked(gtx) {
		h.showUpdateModal = false
		h.startUpdate(state)
	}
}

func (h *HomeScreen) layoutAppVersionBadge(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	if state.AppVersion == "" {
		return layout.Dimensions{}
	}

	versionText := state.AppVersion
	if state.AppVersion != "dev" && !strings.HasPrefix(state.AppVersion, "v") {
		versionText = "v" + state.AppVersion
	}

	return layout.Inset{
		Bottom: unit.Dp(38),
		Right:  unit.Dp(38),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.Body2(h.mui, versionText)
						label.Color = color.NRGBA{R: 227, G: 227, B: 227, A: 255}
						return label.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !state.AppUpdateAvailable && !state.AppRestartRequired && !state.AppIsUpdating {
							return layout.Dimensions{}
						}
						return layout.Inset{Left: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if state.AppIsUpdating {
								return h.layoutUpdatingText(gtx, th)
							}
							if state.AppRestartRequired {
								return h.layoutRestartLink(gtx, th)
							}
							return h.layoutUpdateLink(gtx, th, state)
						})
					}),
				)
			})
		})
	})
}

func (h *HomeScreen) layoutUpdateLink(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			prefix := material.Body2(h.mui, "- ")
			prefix.Color = th.Text
			return prefix.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return h.updateLinkBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Body2(h.mui, "update available")
				label.Color = th.Text

				var labelDims layout.Dimensions
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						labelDims = label.Layout(gtx)
						return labelDims
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lineHeight := gtx.Dp(unit.Dp(1))
						if lineHeight < 1 {
							lineHeight = 1
						}
						size := image.Pt(labelDims.Size.X, lineHeight)
						defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, th.Text)
						return layout.Dimensions{Size: size}
					}),
				)
			})
		}),
	)
}

func (h *HomeScreen) layoutRestartLink(gtx layout.Context, th *Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			prefix := material.Body2(h.mui, "- ")
			prefix.Color = th.Text
			return prefix.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return h.restartLinkBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Body2(h.mui, "restart to apply update")
				label.Color = th.Text

				var labelDims layout.Dimensions
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						labelDims = label.Layout(gtx)
						return labelDims
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lineHeight := gtx.Dp(unit.Dp(1))
						if lineHeight < 1 {
							lineHeight = 1
						}
						size := image.Pt(labelDims.Size.X, lineHeight)
						defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, th.Text)
						return layout.Dimensions{Size: size}
					}),
				)
			})
		}),
	)
}

func (h *HomeScreen) layoutUpdatingText(gtx layout.Context, th *Theme) layout.Dimensions {
	label := material.Body2(h.mui, "- updating")
	label.Color = th.Text
	return label.Layout(gtx)
}

func (h *HomeScreen) startUpdate(state *AppState) {
	if state.AppIsUpdating {
		return
	}

	state.AppIsUpdating = true
	state.SetStatus("Updating Vary...", false)
	go func(buildVersion string) {
		result, err := selfupdate.Apply(context.Background(), buildVersion)
		h.updateResults <- appUpdateResult{result: result, err: err}
	}(state.AppBuildVersion)
}

func restartApp() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	return cmd.Start()
}

func (h *HomeScreen) layoutUpdateModal(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		width := min(gtx.Constraints.Max.X-gtx.Dp(unit.Dp(60)), gtx.Dp(unit.Dp(420)))
		height := gtx.Dp(unit.Dp(190))
		if width < gtx.Dp(unit.Dp(280)) {
			width = gtx.Constraints.Max.X - gtx.Dp(unit.Dp(24))
		}
		gtx.Constraints = layout.Exact(image.Pt(width, height))
		return layoutOutlinedSurface(gtx, unit.Dp(8), color.NRGBA{R: 95, G: 95, B: 95, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				message := "Do you want to update now?"
				if state.AppIsUpdating {
					message = "Updating Vary..."
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.Body1(h.mui, "Update available")
						title.Color = th.Text
						return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, title.Layout)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						body := material.Body2(h.mui, message)
						body.Color = th.TextMuted
						return body.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return h.modalButton(gtx, th, &h.updateLaterBtn, "Later")
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Width: unit.Dp(10)}.Layout(gtx)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return h.modalButton(gtx, th, &h.updateYesBtn, "Yes")
							}),
						)
					}),
				)
			})
		})
	})
}

func (h *HomeScreen) modalButton(gtx layout.Context, th *Theme, btn *widget.Clickable, label string) layout.Dimensions {
	height := gtx.Dp(unit.Dp(44))
	gtx.Constraints.Min.Y = height
	gtx.Constraints.Max.Y = height
	style := material.Button(h.mui, btn, label)
	style.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	style.Color = th.Text
	return layoutOutlinedSurface(gtx, unit.Dp(6), color.NRGBA{R: 120, G: 120, B: 120, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, style.Layout)
}
