package app

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

func layoutOutlinedSurface(gtx layout.Context, radius unit.Dp, borderColor, fillColor color.NRGBA, content layout.Widget) layout.Dimensions {
	max := gtx.Constraints.Max
	if max.X <= 0 || max.Y <= 0 {
		return content(gtx)
	}

	border := gtx.Dp(unit.Dp(1))
	if border < 1 {
		border = 1
	}

	r := gtx.Dp(radius)
	outer := clip.UniformRRect(image.Rect(0, 0, max.X, max.Y), r)
	paint.FillShape(gtx.Ops, borderColor, outer.Op(gtx.Ops))

	innerRect := image.Rect(border, border, max.X-border, max.Y-border)
	if innerRect.Dx() <= 0 || innerRect.Dy() <= 0 {
		return content(gtx)
	}
	innerRadius := r - border
	if innerRadius < 0 {
		innerRadius = 0
	}
	inner := clip.UniformRRect(innerRect, innerRadius)
	paint.FillShape(gtx.Ops, fillColor, inner.Op(gtx.Ops))

	return layout.Inset{
		Top:    unit.Dp(1),
		Bottom: unit.Dp(1),
		Left:   unit.Dp(1),
		Right:  unit.Dp(1),
	}.Layout(gtx, content)
}

func layoutMeasuredOutlinedSurface(gtx layout.Context, radius unit.Dp, borderColor, fillColor color.NRGBA, content layout.Widget) layout.Dimensions {
	rec := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top:    unit.Dp(1),
		Bottom: unit.Dp(1),
		Left:   unit.Dp(1),
		Right:  unit.Dp(1),
	}.Layout(gtx, content)
	call := rec.Stop()

	if dims.Size.X <= 0 || dims.Size.Y <= 0 {
		call.Add(gtx.Ops)
		return dims
	}

	border := gtx.Dp(unit.Dp(1))
	if border < 1 {
		border = 1
	}

	r := gtx.Dp(radius)
	outer := clip.UniformRRect(image.Rect(0, 0, dims.Size.X, dims.Size.Y), r)
	paint.FillShape(gtx.Ops, borderColor, outer.Op(gtx.Ops))

	innerRect := image.Rect(border, border, dims.Size.X-border, dims.Size.Y-border)
	if innerRect.Dx() > 0 && innerRect.Dy() > 0 {
		innerRadius := r - border
		if innerRadius < 0 {
			innerRadius = 0
		}
		inner := clip.UniformRRect(innerRect, innerRadius)
		paint.FillShape(gtx.Ops, fillColor, inner.Op(gtx.Ops))
	}

	call.Add(gtx.Ops)
	return dims
}
