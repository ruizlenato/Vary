package app

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"vary/internal/morphe"
)

type PatchesScreen struct {
	list        widget.List
	search      widget.Editor
	mui         *material.Theme
	backBtn     widget.Clickable
	closeBtn    widget.Clickable
	continueBtn widget.Clickable
	selectAll   widget.Clickable
	deselectAll widget.Clickable
	backIcon    *widget.Icon
	closeIcon   *widget.Icon
	items       []PatchItem
	filtered    []int
	lastSearch  string
}

type PatchItem struct {
	patch    morphe.Patch
	selected widget.Bool
}

func NewPatchesScreen() *PatchesScreen {
	search := widget.Editor{SingleLine: true, Submit: false}
	return &PatchesScreen{
		search:    search,
		mui:       material.NewTheme(),
		list:      widget.List{List: layout.List{Axis: layout.Vertical}},
		backIcon:  mustIcon(backArrowIconVG),
		closeIcon: mustIcon(closeIconVG),
	}
}

func (p *PatchesScreen) StartLoadPatches(state *AppState) {
	if state.IsLoadingPatches {
		return
	}
	state.IsLoadingPatches = true
	state.PatchStatus = "Loading patches..."

	go func() {
		defer func() { state.IsLoadingPatches = false }()

		if state.CLIPath == "" || state.PatchesPath == "" {
			state.SetStatus("Missing morphe assets. Run update again.", true)
			state.PatchStatus = "Missing morphe assets"
			state.SetPatches(nil)
			return
		}
		if state.SelectedPackage == "" {
			state.SetStatus("No application selected", true)
			state.PatchStatus = "No application selected"
			state.SetPatches(nil)
			return
		}

		executor := morphe.NewExecutor(state.CLIPath, state.PatchesPath)
		patches, err := executor.ListPatches(context.Background(), state.SelectedPackage)
		if err != nil {
			state.SetStatus("Patch list error: "+err.Error(), true)
			state.PatchStatus = "Failed to load patches"
			state.SetPatches(nil)
			return
		}

		state.SetPatches(patches)
		state.PatchStatus = fmt.Sprintf("%d patches found", len(patches))
		state.SetStatus(state.PatchStatus, false)
	}()
}

func (p *PatchesScreen) syncItems(state *AppState) {
	if len(p.items) == len(state.Patches) {
		same := true
		for i := range p.items {
			if p.items[i].patch.Name != state.Patches[i].Name {
				same = false
				break
			}
		}
		if same {
			return
		}
	}

	previous := make(map[string]bool, len(p.items))
	for i := range p.items {
		previous[p.items[i].patch.Name] = p.items[i].selected.Value
	}

	p.items = make([]PatchItem, len(state.Patches))
	for i, patch := range state.Patches {
		p.items[i].patch = patch
		if selected, ok := previous[patch.Name]; ok {
			p.items[i].selected.Value = selected
		} else {
			p.items[i].selected.Value = patch.Enabled
		}
	}
	p.applyFilter()
}

func (p *PatchesScreen) applyFilter() {
	query := strings.TrimSpace(strings.ToLower(p.search.Text()))
	p.filtered = p.filtered[:0]
	for i := range p.items {
		if query == "" {
			p.filtered = append(p.filtered, i)
			continue
		}
		name := strings.ToLower(p.items[i].patch.Name)
		desc := strings.ToLower(p.items[i].patch.Description)
		if strings.Contains(name, query) || strings.Contains(desc, query) {
			p.filtered = append(p.filtered, i)
		}
	}
}

func (p *PatchesScreen) Layout(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	p.syncItems(state)
	currentSearch := p.search.Text()
	if currentSearch != p.lastSearch {
		p.lastSearch = currentSearch
		p.applyFilter()
	}
	originalConstraints := gtx.Constraints

	layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H4(p.mui, "Select patches to include")
						title.Color = th.Text
						return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, title.Layout)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						sub := material.Body2(p.mui, "Choose patches to apply")
						sub.Color = th.TextMuted
						return layout.Inset{Bottom: unit.Dp(18)}.Layout(gtx, sub.Layout)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						cardWidth := min(gtx.Constraints.Max.X-gtx.Dp(unit.Dp(64)), gtx.Dp(unit.Dp(760)))
						cardHeight := min(gtx.Constraints.Max.Y-gtx.Dp(unit.Dp(220)), gtx.Dp(unit.Dp(620)))
						if cardWidth < gtx.Dp(unit.Dp(320)) {
							cardWidth = gtx.Constraints.Max.X - gtx.Dp(unit.Dp(24))
						}
						if cardHeight < gtx.Dp(unit.Dp(260)) {
							cardHeight = gtx.Dp(unit.Dp(260))
						}

						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints = layout.Exact(image.Pt(cardWidth, cardHeight))
							outer := clip.UniformRRect(image.Rect(0, 0, cardWidth, cardHeight), gtx.Dp(unit.Dp(8)))
							paint.FillShape(gtx.Ops, color.NRGBA{R: 78, G: 78, B: 78, A: 255}, outer.Op(gtx.Ops))

							inner := image.Rect(1, 1, cardWidth-1, cardHeight-1)
							innerRRect := clip.UniformRRect(inner, gtx.Dp(unit.Dp(8)))
							paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, innerRRect.Op(gtx.Ops))

							return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
											layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												height := gtx.Dp(unit.Dp(40))
												gtx.Constraints.Min.Y = height
												gtx.Constraints.Max.Y = height
												editor := material.Editor(p.mui, &p.search, "Search patches")
												editor.Color = th.Text
												editor.HintColor = th.TextMuted
												return p.outlinedField(gtx, func(gtx layout.Context) layout.Dimensions {
													return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10), Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, editor.Layout)
												})
											}),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(unit.Dp(108)), gtx.Dp(unit.Dp(40))))
													btn := material.Button(p.mui, &p.selectAll, "Select all")
													btn.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
													btn.Color = th.Text
													return p.outlinedButton(gtx, btn.Layout)
												})
											}),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(unit.Dp(126)), gtx.Dp(unit.Dp(40))))
													btn := material.Button(p.mui, &p.deselectAll, "Deselect all")
													btn.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
													btn.Color = th.Text
													return p.outlinedButton(gtx, btn.Layout)
												})
											}),
										)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										selected := 0
										for i := range p.items {
											if p.items[i].selected.Value {
												selected++
											}
										}
										meta := material.Body2(p.mui, fmt.Sprintf("%d selected • %d shown", selected, len(p.filtered)))
										meta.Color = th.TextMuted
										return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(2)}.Layout(gtx, meta.Layout)
									}),
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										if state.IsLoadingPatches {
											msg := material.Body1(p.mui, "Loading patches...")
											msg.Color = th.TextMuted
											return layout.Center.Layout(gtx, msg.Layout)
										}
										if len(p.filtered) == 0 {
											msg := material.Body1(p.mui, "No patches found")
											msg.Color = th.TextMuted
											return layout.Center.Layout(gtx, msg.Layout)
										}
										return material.List(p.mui, &p.list).Layout(gtx, len(p.filtered), func(gtx layout.Context, index int) layout.Dimensions {
											itemIndex := p.filtered[index]
											return p.patchItem(gtx, th, &p.items[itemIndex])
										})
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											btnWidth := gtx.Constraints.Max.X
											btnHeight := gtx.Dp(unit.Dp(46))
											gtx.Constraints = layout.Exact(image.Pt(btnWidth, btnHeight))
											btn := material.Button(p.mui, &p.continueBtn, "Continue")
											btn.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
											btn.Color = th.Text
											return p.outlinedButton(gtx, func(gtx layout.Context) layout.Dimensions {
												return btn.Layout(gtx)
											})
										})
									}),
								)
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
			return layout.Inset{Top: unit.Dp(38), Right: unit.Dp(38)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
	)

	return layout.Dimensions{Size: originalConstraints.Max}
}

func (p *PatchesScreen) patchItem(gtx layout.Context, th *Theme, item *PatchItem) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Start}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(2), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						cb := material.CheckBox(p.mui, &item.selected, "")
						cb.Color = color.NRGBA{R: 227, G: 227, B: 227, A: 255}
						cb.IconColor = color.NRGBA{R: 227, G: 227, B: 227, A: 255}
						return cb.Layout(gtx)
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							name := material.Label(p.mui, unit.Sp(16), item.patch.Name)
							name.Color = th.Text
							return name.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							desc := material.Body2(p.mui, item.patch.Description)
							desc.Color = th.TextMuted
							return layout.Inset{Top: unit.Dp(2)}.Layout(gtx, desc.Layout)
						}),
					)
				}),
			)
		})
	})
}

func (p *PatchesScreen) outlinedButton(gtx layout.Context, content layout.Widget) layout.Dimensions {
	border := gtx.Dp(unit.Dp(1))
	max := gtx.Constraints.Max
	if max.X <= 0 || max.Y <= 0 {
		return content(gtx)
	}

	outer := clip.UniformRRect(image.Rect(0, 0, max.X, max.Y), gtx.Dp(unit.Dp(6)))
	paint.FillShape(gtx.Ops, color.NRGBA{R: 120, G: 120, B: 120, A: 255}, outer.Op(gtx.Ops))
	innerRect := image.Rect(border, border, max.X-border, max.Y-border)
	inner := clip.UniformRRect(innerRect, gtx.Dp(unit.Dp(6)))
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, inner.Op(gtx.Ops))

	return layout.Inset{
		Top:    unit.Dp(1),
		Bottom: unit.Dp(1),
		Left:   unit.Dp(1),
		Right:  unit.Dp(1),
	}.Layout(gtx, content)
}

func (p *PatchesScreen) outlinedField(gtx layout.Context, content layout.Widget) layout.Dimensions {
	max := gtx.Constraints.Max
	if max.X <= 0 || max.Y <= 0 {
		return content(gtx)
	}

	outer := clip.UniformRRect(image.Rect(0, 0, max.X, max.Y), gtx.Dp(unit.Dp(6)))
	paint.FillShape(gtx.Ops, color.NRGBA{R: 90, G: 90, B: 90, A: 255}, outer.Op(gtx.Ops))
	innerRect := image.Rect(1, 1, max.X-1, max.Y-1)
	inner := clip.UniformRRect(innerRect, gtx.Dp(unit.Dp(6)))
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, inner.Op(gtx.Ops))

	return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(1), Right: unit.Dp(1)}.Layout(gtx, content)
}

func (p *PatchesScreen) HandleInput(gtx layout.Context, state *AppState) {
	for p.closeBtn.Clicked(gtx) {
		os.Exit(0)
	}
	for p.backBtn.Clicked(gtx) {
		state.SetScreen(ScreenPackages)
	}
	for p.selectAll.Clicked(gtx) {
		for i := range p.items {
			p.items[i].selected.Value = true
		}
	}
	for p.deselectAll.Clicked(gtx) {
		for i := range p.items {
			p.items[i].selected.Value = false
		}
	}
	for p.continueBtn.Clicked(gtx) {
		selected := 0
		for i := range p.items {
			if p.items[i].selected.Value {
				selected++
			}
		}
		state.SetStatus(fmt.Sprintf("%d patches selected", selected), false)
	}
}
