package app

import (
	"image/color"
	"os"

	"vary/internal/config"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type SettingsScreen struct {
	releaseMode widget.Enum
	saveBtn     widget.Clickable
	backBtn     widget.Clickable
	closeBtn    widget.Clickable
	mui         *material.Theme
	backIcon    *widget.Icon
	closeIcon   *widget.Icon
	lastMode    config.Mode
}

func NewSettingsScreen() *SettingsScreen {
	return &SettingsScreen{
		backIcon:  mustIcon(backArrowIconVG),
		closeIcon: mustIcon(closeIconVG),
		mui:       material.NewTheme(),
	}
}

const (
	radioKeyStable = string(config.ModeStable)
	radioKeyDev    = string(config.ModeDev)
)

func (s *SettingsScreen) Layout(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	if state.Config.Mode != s.lastMode {
		s.lastMode = state.Config.Mode
		s.releaseMode.Value = radioKeyStable
		if s.lastMode == config.ModeDev {
			s.releaseMode.Value = radioKeyDev
		}
	}

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
							title := material.H5(s.mui, "Settings")
							title.Color = th.Text
							return title.Layout(gtx)
						})
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis:      layout.Vertical,
							Alignment: layout.Middle,
							Spacing:   layout.SpaceEvenly,
						}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								label := material.Body1(s.mui, "Release Mode:")
								label.Color = th.Text
								return label.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return s.radioOption(gtx, th, radioKeyStable, "Morphe (Stable)")
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return s.radioOption(gtx, th, radioKeyDev, "Morphe Dev (Pre-release)")
								})
							}),
						)
					}),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis:      layout.Horizontal,
								Spacing:   layout.SpaceEvenly,
								Alignment: layout.Middle,
							}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return s.button(gtx, th, "Save", &s.saveBtn)
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
				Top:  unit.Dp(38),
				Left: unit.Dp(38),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
			return layout.Inset{
				Top:   unit.Dp(38),
				Right: unit.Dp(38),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
	)

	return layout.Dimensions{Size: originalConstraints.Max}
}

func (s *SettingsScreen) radioOption(gtx layout.Context, th *Theme, key, label string) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			radioBtn := material.RadioButton(s.mui, &s.releaseMode, key, label)
			radioBtn.Color = th.Text
			radioBtn.IconColor = th.Primary
			return radioBtn.Layout(gtx)
		}),
	)
}

func (s *SettingsScreen) button(gtx layout.Context, th *Theme, text string, btn *widget.Clickable) layout.Dimensions {
	btnStyle := material.Button(s.mui, btn, text)
	btnStyle.Background = th.Surface
	btnStyle.Color = th.Text
	return layout.Inset{
		Top:    unit.Dp(8),
		Bottom: unit.Dp(8),
		Left:   unit.Dp(16),
		Right:  unit.Dp(16),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return btnStyle.Layout(gtx)
	})
}

func (s *SettingsScreen) HandleInput(gtx layout.Context, state *AppState) {
	for s.closeBtn.Clicked(gtx) {
		os.Exit(0)
	}
	for s.backBtn.Clicked(gtx) {
		state.SetScreen(ScreenHome)
	}
	for s.saveBtn.Clicked(gtx) {
		if s.releaseMode.Value == radioKeyDev {
			state.Config.Mode = config.ModeDev
		} else {
			state.Config.Mode = config.ModeStable
		}
		if err := state.Config.Save(); err != nil {
			state.SetStatus("Error saving: "+err.Error(), true)
		} else {
			state.SetStatus("Settings saved", false)
		}
		state.SetScreen(ScreenHome)
	}
}
