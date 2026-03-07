package app

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func layoutDeviceStatusBadge(gtx layout.Context, state *AppState, mui *material.Theme) layout.Dimensions {
	if state.DeviceModel == "" {
		return layout.Dimensions{}
	}

	status := "disconnected"
	statusDot := color.NRGBA{R: 255, G: 116, B: 108, A: 255}
	if state.DeviceConnected {
		status = "connected"
		statusDot = color.NRGBA{R: 128, G: 239, B: 128, A: 255}
	}

	return layout.Inset{
		Bottom: unit.Dp(38),
		Left:   unit.Dp(38),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						size := gtx.Dp(unit.Dp(6))
						return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							defer clip.UniformRRect(image.Rect(0, 0, size, size), size/2).Push(gtx.Ops).Pop()
							paint.Fill(gtx.Ops, statusDot)
							return layout.Dimensions{Size: image.Pt(size, size)}
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.Body2(mui, state.DeviceModel+" "+status)
						label.Color = color.NRGBA{R: 227, G: 227, B: 227, A: 255}
						return label.Layout(gtx)
					}),
				)
			})
		})
	})
}
