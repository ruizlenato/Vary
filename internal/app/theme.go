package app

import (
	"image/color"

	"gioui.org/unit"
)

type Theme struct {
	Background color.NRGBA
	Surface    color.NRGBA
	Primary    color.NRGBA
	Secondary  color.NRGBA
	Text       color.NRGBA
	TextMuted  color.NRGBA
	Border     color.NRGBA
	Success    color.NRGBA
	Error      color.NRGBA
	Warning    color.NRGBA

	TextSize unit.Sp
	Padding  unit.Dp
	Margin   unit.Dp
}

func NewTheme() *Theme {
	return &Theme{
		Background: color.NRGBA{R: 0, G: 0, B: 0, A: 255},
		Surface:    color.NRGBA{R: 20, G: 20, B: 20, A: 255},
		Primary:    color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		Secondary:  color.NRGBA{R: 180, G: 180, B: 180, A: 255},
		Text:       color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		TextMuted:  color.NRGBA{R: 128, G: 128, B: 128, A: 255},
		Border:     color.NRGBA{R: 64, G: 64, B: 64, A: 255},
		Success:    color.NRGBA{R: 100, G: 255, B: 100, A: 255},
		Error:      color.NRGBA{R: 255, G: 80, B: 80, A: 255},
		Warning:    color.NRGBA{R: 255, G: 200, B: 80, A: 255},
		TextSize:   14,
		Padding:    16,
		Margin:     8,
	}
}
