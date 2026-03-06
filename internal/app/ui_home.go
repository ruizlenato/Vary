package app

import (
	"image"
	"image/color"
	"os"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type HomeScreen struct {
	startBtn          widget.Clickable
	settingsBtn       widget.Clickable
	closeBtn          widget.Clickable
	closeIcon         *widget.Icon
	OnStartClicked    func()
	OnSettingsClicked func()
}

func NewHomeScreen() *HomeScreen {
	return &HomeScreen{closeIcon: mustIcon(closeIconVG)}
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
						title := material.H2(material.NewTheme(), "Vary")
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
									return h.button(gtx, th, "Start", &h.startBtn)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return h.button(gtx, th, "Settings", &h.settingsBtn)
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
	)

	return layout.Dimensions{Size: originalConstraints.Max}
}

func (h *HomeScreen) button(gtx layout.Context, th *Theme, text string, btn *widget.Clickable) layout.Dimensions {
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
				label := material.Label(material.NewTheme(), unit.Sp(15), text)
				label.Color = borderColor
				return label.Layout(gtx)
			})
		})
	})

	return dims
}

func (h *HomeScreen) HandleInput(gtx layout.Context, state *AppState) {
	for h.closeBtn.Clicked(gtx) {
		os.Exit(0)
	}
	for h.startBtn.Clicked(gtx) {
		state.SetScreen(ScreenDownload)
	}
	for h.settingsBtn.Clicked(gtx) {
		state.SetScreen(ScreenSettings)
	}
}
