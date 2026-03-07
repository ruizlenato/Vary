package app

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type PackagesScreen struct {
	list      widget.List
	mui       *material.Theme
	backBtn   widget.Clickable
	items     []PackageItem
	closeBtn  widget.Clickable
	backIcon  *widget.Icon
	closeIcon *widget.Icon
}

type PackageItem struct {
	label   string
	clicked widget.Clickable
}

func NewPackagesScreen() *PackagesScreen {
	backIcon := mustIcon(backArrowIconVG)
	closeIcon := mustIcon(closeIconVG)
	return &PackagesScreen{
		mui: material.NewTheme(),
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		backIcon:  backIcon,
		closeIcon: closeIcon,
	}
}

func (p *PackagesScreen) Layout(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	if len(p.items) != len(state.FilteredPackages) {
		p.items = make([]PackageItem, len(state.FilteredPackages))
		for i, pkg := range state.FilteredPackages {
			p.items[i].label = pkg
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
						return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							title := material.H5(p.mui, "Select Application")
							title.Color = th.Text
							return title.Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(300))
						return layout.Inset{
							Left:  unit.Dp(24),
							Right: unit.Dp(24),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							if len(p.items) == 0 {
								msg := material.Body2(p.mui, "No packages found")
								msg.Color = th.TextMuted
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return msg.Layout(gtx)
								})
							}
							return material.List(p.mui, &p.list).Layout(gtx, len(p.items), func(gtx layout.Context, index int) layout.Dimensions {
								return p.packageItem(gtx, th, &p.items[index])
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							statusText := state.StatusMessage
							if len(state.FilteredPackages) > 0 {
								statusText = fmt.Sprintf("%d apps | %s", len(state.FilteredPackages), statusText)
							}
							status := material.Body2(p.mui, statusText)
							status.Color = th.TextMuted
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return status.Layout(gtx)
							})
						})
					}),
				)
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return layout.Inset{
				Top:  unit.Dp(38),
				Left: unit.Dp(22),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
			return layout.Inset{
				Top:   unit.Dp(38),
				Right: unit.Dp(38),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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

func (p *PackagesScreen) packageItem(gtx layout.Context, th *Theme, item *PackageItem) layout.Dimensions {
	return layout.Inset{
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		itemHeight := gtx.Dp(unit.Dp(44))
		itemWidth := min(gtx.Constraints.Max.X-gtx.Dp(unit.Dp(20)), gtx.Dp(unit.Dp(560)))
		if itemWidth <= 0 {
			itemWidth = gtx.Constraints.Max.X
		}

		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(image.Pt(itemWidth, itemHeight))
			return item.clicked.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layoutOutlinedSurface(gtx, unit.Dp(4), color.NRGBA{R: 95, G: 95, B: 95, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.Body1(p.mui, item.label)
						label.Color = th.Text
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return label.Layout(gtx)
						})
					})
				})
			})
		})
	})
}

func (p *PackagesScreen) HandleInput(gtx layout.Context, state *AppState) {
	for p.closeBtn.Clicked(gtx) {
		os.Exit(0)
	}
	for p.backBtn.Clicked(gtx) {
		state.SetScreen(ScreenHome)
	}
	for i := range p.items {
		if p.items[i].clicked.Clicked(gtx) {
			state.SelectedPackage = p.items[i].label
			state.SetPatches(nil)
			state.PatchStatus = ""
			state.SetStatus("Selected: "+p.items[i].label, false)
			state.SetScreen(ScreenPatches)
		}
	}
}
