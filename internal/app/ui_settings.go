package app

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vary/internal/config"
	"vary/internal/github"
	"vary/internal/storage"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
	"gioui.org/x/richtext"
)

type SettingsScreen struct {
	releaseMode           widget.Enum
	sourceMode            widget.Enum
	autoUpdate            widget.Bool
	saveBtn               widget.Clickable
	sourceMorpheBtn       widget.Clickable
	sourceCustomBtn       widget.Clickable
	releaseStableBtn      widget.Clickable
	releasePreBtn         widget.Clickable
	keystoreBtn           widget.Clickable
	clearKeyBtn           widget.Clickable
	backBtn               widget.Clickable
	closeBtn              widget.Clickable
	mui                   *material.Theme
	backIcon              *widget.Icon
	closeIcon             *widget.Icon
	lastMode              config.Mode
	lastAutoUpdate        bool
	lastCustomPatchesRepo string
	lastKeystorePath      string
	pendingKeystoreSource string
	pendingClearKeystore  bool
	customPatchesRepo     widget.Editor
	repoErrorOkBtn        widget.Clickable
	repoInfoOkBtn         widget.Clickable
	repoValidationError   string
	showRepoErrorModal    bool
	repoInfoMessage       string
	repoInfoRepo          string
	repoInfoPreferred     string
	repoInfoFallback      string
	repoInfoTextState     richtext.InteractiveText
	showRepoInfoModal     bool
	lastCustomRepoInput   string
	suppressRepoInputSync bool
	layoutScale           float32
	explorer              *explorer.Explorer
	pickResults           chan filePickResult
}

func NewSettingsScreen(ex *explorer.Explorer) *SettingsScreen {
	screen := &SettingsScreen{
		backIcon:    mustIcon(backArrowIconVG),
		closeIcon:   mustIcon(closeIconVG),
		mui:         material.NewTheme(),
		explorer:    ex,
		pickResults: make(chan filePickResult, 1),
	}
	screen.releaseMode.Value = radioKeyStable
	screen.sourceMode.Value = sourceKeyMorphe
	screen.customPatchesRepo.SingleLine = true
	return screen
}

const (
	radioKeyStable  = string(config.ModeStable)
	radioKeyDev     = string(config.ModeDev)
	sourceKeyMorphe = "morphe"
	sourceKeyCustom = "custom"
)

func (s *SettingsScreen) computeLayoutScale(gtx layout.Context) float32 {
	baseW := float32(gtx.Dp(unit.Dp(920)))
	baseH := float32(gtx.Dp(unit.Dp(980)))
	if baseW <= 0 || baseH <= 0 {
		return 1
	}

	scaleW := float32(gtx.Constraints.Max.X) / baseW
	scaleH := float32(gtx.Constraints.Max.Y) / baseH

	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}
	if scale > 1 {
		scale = 1
	}
	if scale < 0.68 {
		scale = 0.68
	}

	return scale
}

func (s *SettingsScreen) dp(v float32) unit.Dp {
	if s.layoutScale <= 0 {
		return unit.Dp(v)
	}
	return unit.Dp(v * s.layoutScale)
}

func (s *SettingsScreen) sp(v float32) unit.Sp {
	scaled := v
	if s.layoutScale > 0 {
		scaled = v * s.layoutScale
	}
	if scaled < 11 {
		scaled = 11
	}
	return unit.Sp(scaled)
}

func (s *SettingsScreen) Layout(gtx layout.Context, th *Theme, state *AppState) layout.Dimensions {
	if state.Config.Mode != s.lastMode {
		s.lastMode = state.Config.Mode
		s.releaseMode.Value = radioKeyStable
		if s.lastMode == config.ModeDev {
			s.releaseMode.Value = radioKeyDev
		}
	}
	if state.Config.AutoUpdate != s.lastAutoUpdate {
		s.lastAutoUpdate = state.Config.AutoUpdate
		s.autoUpdate.Value = state.Config.AutoUpdate
	}
	if state.Config.CustomPatchesRepo != s.lastCustomPatchesRepo {
		s.lastCustomPatchesRepo = state.Config.CustomPatchesRepo
		s.sourceMode.Value = sourceKeyMorphe
		if strings.TrimSpace(state.Config.CustomPatchesRepo) != "" {
			s.sourceMode.Value = sourceKeyCustom
		}
		if strings.TrimSpace(s.customPatchesRepo.Text()) != state.Config.CustomPatchesRepo {
			s.suppressRepoInputSync = true
			s.customPatchesRepo.SetText(state.Config.CustomPatchesRepo)
		}
	}
	if currentInput := s.customPatchesRepo.Text(); currentInput != s.lastCustomRepoInput {
		s.lastCustomRepoInput = currentInput
		if s.suppressRepoInputSync {
			s.suppressRepoInputSync = false
		} else {
			s.repoValidationError = ""
			s.showRepoErrorModal = false
			s.repoInfoMessage = ""
			s.repoInfoRepo = ""
			s.repoInfoPreferred = ""
			s.repoInfoFallback = ""
			s.showRepoInfoModal = false
		}
	}
	if s.sourceMode.Value != sourceKeyCustom && s.repoValidationError != "" {
		s.repoValidationError = ""
		s.showRepoErrorModal = false
	}
	if state.Config.CustomKeystorePath != s.lastKeystorePath {
		s.lastKeystorePath = state.Config.CustomKeystorePath
		s.pendingKeystoreSource = ""
		s.pendingClearKeystore = false
	}

	originalConstraints := gtx.Constraints
	s.layoutScale = s.computeLayoutScale(gtx)
	s.mui.TextSize = s.sp(16)

	layout.Stack{}.Layout(gtx,

		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				contentWidth := min(gtx.Constraints.Max.X-gtx.Dp(s.dp(48)), gtx.Dp(s.dp(760)))
				if contentWidth < gtx.Dp(s.dp(280)) {
					contentWidth = gtx.Constraints.Max.X - gtx.Dp(s.dp(24))
				}
				narrow := contentWidth < gtx.Dp(s.dp(540))
				return layout.Inset{
					Top:    s.dp(28),
					Bottom: s.dp(28),
					Left:   s.dp(12),
					Right:  s.dp(12),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = contentWidth
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return s.layoutHeader(gtx, th)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: s.dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return s.layoutSettingsCard(gtx, th, contentWidth, "Patches", "", func(gtx layout.Context) layout.Dimensions {
									hasRepoValidationError := s.repoValidationError != ""
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											label := material.Body1(s.mui, "Source")
											label.Color = th.Text
											return layout.Inset{Bottom: s.dp(12)}.Layout(gtx, label.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
												layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
													return s.channelOptionButton(gtx, th, "Morphe", &s.sourceMorpheBtn, s.sourceMode.Value == sourceKeyMorphe)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Spacer{Width: s.dp(10)}.Layout(gtx)
												}),
												layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
													return s.channelOptionButton(gtx, th, "Custom", &s.sourceCustomBtn, s.sourceMode.Value == sourceKeyCustom)
												}),
											)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											label := material.Body1(s.mui, "Release channel")
											label.Color = th.Text
											return layout.Inset{Top: s.dp(16), Bottom: s.dp(10)}.Layout(gtx, label.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
												layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
													return s.channelOptionButton(gtx, th, "Stable", &s.releaseStableBtn, s.releaseMode.Value == radioKeyStable)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Spacer{Width: s.dp(10)}.Layout(gtx)
												}),
												layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
													return s.channelOptionButton(gtx, th, "Pre Release", &s.releasePreBtn, s.releaseMode.Value == radioKeyDev)
												}),
											)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if s.sourceMode.Value != sourceKeyCustom {
												return layout.Dimensions{}
											}
											label := material.Body1(s.mui, "Custom repository")
											label.Color = th.Text
											return layout.Inset{Top: s.dp(14)}.Layout(gtx, label.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if s.sourceMode.Value != sourceKeyCustom {
												return layout.Dimensions{}
											}
											help := material.Body2(s.mui, "Use owner/repo or a GitHub URL. Example: RookieEnough/De-ReVanced")
											help.Color = th.TextMuted
											return layout.Inset{Top: s.dp(10), Bottom: s.dp(10)}.Layout(gtx, help.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if s.sourceMode.Value != sourceKeyCustom {
												return layout.Dimensions{}
											}
											inputHeight := gtx.Dp(s.dp(46))
											inputGtx := gtx
											inputGtx.Constraints.Min.Y = inputHeight
											inputGtx.Constraints.Max.Y = inputHeight
											return layoutOutlinedSurface(inputGtx, s.dp(6), color.NRGBA{R: 64, G: 64, B: 64, A: 255}, color.NRGBA{R: 10, G: 10, B: 10, A: 255}, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: s.dp(10), Bottom: s.dp(10), Left: s.dp(12), Right: s.dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													editor := material.Editor(s.mui, &s.customPatchesRepo, "owner/repo or GitHub URL")
													editor.Color = th.Text
													editor.HintColor = th.TextMuted
													return editor.Layout(gtx)
												})
											})
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if !hasRepoValidationError || s.sourceMode.Value != sourceKeyCustom {
												return layout.Dimensions{}
											}
											info := material.Body2(s.mui, "Invalid Repo")
											info.Color = color.NRGBA{R: 230, G: 120, B: 120, A: 255}
											return layout.Inset{Top: s.dp(10)}.Layout(gtx, info.Layout)
										}),
									)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: s.dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return s.layoutSettingsCard(gtx, th, contentWidth, "App updates", "", func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Bottom: s.dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												check := material.CheckBox(s.mui, &s.autoUpdate, "Automatic updates")
												check.Color = th.Text
												check.IconColor = th.Primary
												return check.Layout(gtx)
											})
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											help := material.Body2(s.mui, "When enabled, Vary downloads and applies updates for Vary automatically when a new version is found.")
											help.Color = th.TextMuted
											return help.Layout(gtx)
										}),
									)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: s.dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return s.layoutSettingsCard(gtx, th, contentWidth, "Signing", "Manage the keystore used when Morphe signs patched apps.", func(gtx layout.Context) layout.Dimensions {
									keystoreText := "No custom keystore selected"
									if s.pendingClearKeystore {
										keystoreText = "Will be cleared on Save"
									} else if s.pendingKeystoreSource != "" {
										keystoreText = filepath.Base(s.pendingKeystoreSource) + " (pending save)"
									} else if state.Config.CustomKeystorePath != "" {
										keystoreText = filepath.Base(state.Config.CustomKeystorePath)
									}

									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											label := material.Body1(s.mui, "Custom keystore")
											label.Color = th.Text
											return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, label.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											body := material.Body2(s.mui, keystoreText)
											body.Color = th.TextMuted
											return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, body.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											warning := material.Body2(s.mui, "If no custom keystore is selected, one will be generated automatically.")
											warning.Color = th.TextMuted
											return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, warning.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return s.layoutDualButtons(gtx, th, narrow)
										}),
									)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: s.dp(20), Bottom: s.dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									width := min(gtx.Dp(s.dp(220)), contentWidth)
									if narrow {
										width = min(gtx.Dp(s.dp(220)), contentWidth-gtx.Dp(s.dp(24)))
									}
									return s.actionButton(gtx, th, "Save settings", &s.saveBtn, width)
								})
							})
						}),
					)
				})
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return layout.Inset{
				Top:  s.dp(38),
				Left: s.dp(38),
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
				Top:   s.dp(38),
				Right: s.dp(38),
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
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if !s.showRepoErrorModal {
				return layout.Dimensions{}
			}
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return s.layoutRepoModal(gtx, th, "Invalid patches repository", s.repoValidationError, th.TextMuted, &s.repoErrorOkBtn)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if !s.showRepoInfoModal {
				return layout.Dimensions{}
			}
			gtx.Constraints = layout.Exact(originalConstraints.Max)
			return s.layoutRepoModal(gtx, th, "Channel fallback", s.repoInfoMessage, th.TextMuted, &s.repoInfoOkBtn)
		}),
	)

	return layout.Dimensions{Size: originalConstraints.Max}
}

func (s *SettingsScreen) layoutHeader(gtx layout.Context, th *Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			title := material.H5(s.mui, "Settings")
			title.Color = th.Text
			return title.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			subtitle := material.Body2(s.mui, "Tune update behavior and signing options before you patch.")
			subtitle.Color = th.TextMuted
			return layout.Inset{Top: s.dp(6)}.Layout(gtx, subtitle.Layout)
		}),
	)
}

func (s *SettingsScreen) layoutSettingsCard(gtx layout.Context, th *Theme, maxWidth int, titleText, subtitleText string, content layout.Widget) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		width := min(maxWidth, gtx.Dp(s.dp(560)))
		if width > gtx.Constraints.Max.X {
			width = gtx.Constraints.Max.X
		}
		cardGtx := gtx
		cardGtx.Constraints.Min.X = width
		cardGtx.Constraints.Max.X = width
		return layoutMeasuredOutlinedSurface(cardGtx, s.dp(8), color.NRGBA{R: 78, G: 78, B: 78, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: s.dp(18), Bottom: s.dp(18), Left: s.dp(18), Right: s.dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				children := make([]layout.FlexChild, 0, 3)
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					title := material.Body1(s.mui, titleText)
					title.Color = th.Text
					return title.Layout(gtx)
				}))
				if strings.TrimSpace(subtitleText) != "" {
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						subtitle := material.Body2(s.mui, subtitleText)
						subtitle.Color = th.TextMuted
						return layout.Inset{Top: s.dp(6), Bottom: s.dp(18)}.Layout(gtx, subtitle.Layout)
					}))
				} else {
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: s.dp(8)}.Layout(gtx)
					}))
				}
				children = append(children, layout.Rigid(content))

				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		})
	})
}

func (s *SettingsScreen) layoutDualButtons(gtx layout.Context, th *Theme, stacked bool) layout.Dimensions {
	buttonWidth := min(gtx.Dp(s.dp(210)), (gtx.Constraints.Max.X-gtx.Dp(s.dp(10)))/2)
	if stacked || buttonWidth < gtx.Dp(s.dp(120)) {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					width := min(gtx.Dp(s.dp(220)), gtx.Constraints.Max.X-gtx.Dp(s.dp(12)))
					return s.actionButton(gtx, th, "Select keystore", &s.keystoreBtn, width)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: s.dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						width := min(gtx.Dp(s.dp(220)), gtx.Constraints.Max.X-gtx.Dp(s.dp(12)))
						return s.actionButton(gtx, th, "Clear", &s.clearKeyBtn, width)
					})
				})
			}),
		)
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.actionButton(gtx, th, "Select keystore", &s.keystoreBtn, buttonWidth)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Width: s.dp(10)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.actionButton(gtx, th, "Clear", &s.clearKeyBtn, buttonWidth)
			}),
		)
	})
}

func (s *SettingsScreen) radioOption(gtx layout.Context, th *Theme, maxWidth int, mode *widget.Enum, key, label string) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		width := min(gtx.Dp(s.dp(320)), maxWidth-gtx.Dp(s.dp(24)))
		if width > gtx.Constraints.Max.X {
			width = gtx.Constraints.Max.X
		}
		if width < gtx.Dp(s.dp(180)) {
			width = gtx.Constraints.Max.X
		}
		optionGtx := gtx
		optionGtx.Constraints = layout.Exact(image.Pt(width, gtx.Dp(s.dp(52))))
		return layoutOutlinedSurface(optionGtx, s.dp(6), color.NRGBA{R: 64, G: 64, B: 64, A: 255}, color.NRGBA{R: 10, G: 10, B: 10, A: 255}, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: s.dp(12), Right: s.dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				radioBtn := material.RadioButton(s.mui, mode, key, label)
				radioBtn.Color = th.Text
				radioBtn.IconColor = th.Primary
				return layout.Center.Layout(gtx, radioBtn.Layout)
			})
		})
	})
}

func (s *SettingsScreen) channelOptionButton(gtx layout.Context, th *Theme, label string, btn *widget.Clickable, selected bool) layout.Dimensions {
	width := gtx.Constraints.Max.X
	height := gtx.Dp(s.dp(46))
	if height < gtx.Dp(s.dp(40)) {
		height = gtx.Dp(s.dp(40))
	}
	gtx.Constraints = layout.Exact(image.Pt(width, height))

	border := color.NRGBA{R: 64, G: 64, B: 64, A: 255}
	if selected {
		border = th.Primary
	}

	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layoutOutlinedSurface(gtx, s.dp(6), border, color.NRGBA{R: 10, G: 10, B: 10, A: 255}, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				text := material.Body1(s.mui, label)
				text.Color = th.Text
				return text.Layout(gtx)
			})
		})
	})
}

func (s *SettingsScreen) actionButton(gtx layout.Context, th *Theme, text string, btn *widget.Clickable, width int) layout.Dimensions {
	height := gtx.Dp(s.dp(44))
	if width <= 0 {
		width = gtx.Dp(s.dp(180))
	}
	return layout.Inset{}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Exact(image.Pt(width, height))
		return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			borderInset := gtx.Dp(s.dp(1))
			cornerRadius := gtx.Dp(s.dp(6))
			outer := clip.UniformRRect(image.Rect(0, 0, width, height), cornerRadius)
			paint.FillShape(gtx.Ops, color.NRGBA{R: 120, G: 120, B: 120, A: 255}, outer.Op(gtx.Ops))

			innerRadius := cornerRadius - borderInset
			if innerRadius < 0 {
				innerRadius = 0
			}
			innerRect := image.Rect(borderInset, borderInset, width-borderInset, height-borderInset)
			inner := clip.UniformRRect(innerRect, innerRadius)
			paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, inner.Op(gtx.Ops))

			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Label(s.mui, s.sp(14), text)
				label.Color = th.Text
				return label.Layout(gtx)
			})
		})
	})
}

func (s *SettingsScreen) layoutRepoModal(gtx layout.Context, th *Theme, titleText, messageText string, messageColor color.NRGBA, okBtn *widget.Clickable) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		width := min(gtx.Constraints.Max.X-gtx.Dp(s.dp(60)), gtx.Dp(s.dp(460)))
		height := gtx.Dp(s.dp(210))
		if width < gtx.Dp(s.dp(280)) {
			width = gtx.Constraints.Max.X - gtx.Dp(s.dp(24))
		}
		gtx.Constraints = layout.Exact(image.Pt(width, height))
		return layoutOutlinedSurface(gtx, s.dp(8), color.NRGBA{R: 95, G: 95, B: 95, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: s.dp(16), Bottom: s.dp(16), Left: s.dp(16), Right: s.dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.Body1(s.mui, titleText)
						title.Color = th.Text
						return layout.Inset{Bottom: s.dp(10)}.Layout(gtx, title.Layout)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if titleText == "Channel fallback" && s.repoInfoRepo != "" && s.repoInfoFallback != "" {
							styles := []richtext.SpanStyle{
								{Content: "Repository ", Size: s.sp(14), Color: th.TextMuted},
								{Content: s.repoInfoRepo, Size: s.sp(14), Color: th.Text},
								{Content: " has no " + s.repoInfoPreferred + " channel. ", Size: s.sp(14), Color: th.TextMuted},
								{Content: s.repoInfoFallback + " channel will be used instead.", Size: s.sp(14), Color: th.Text},
							}
							text := richtext.Text(&s.repoInfoTextState, s.mui.Shaper, styles...)
							return text.Layout(gtx)
						}

						msg := messageText
						if msg == "" {
							msg = "Unable to validate repository releases."
						}
						body := material.Body2(s.mui, msg)
						body.Color = messageColor
						return body.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return s.actionButton(gtx, th, "OK", okBtn, gtx.Dp(s.dp(140)))
						})
					}),
				)
			})
		})
	})
}

func (s *SettingsScreen) HandleInput(gtx layout.Context, state *AppState) {
	select {
	case result := <-s.pickResults:
		if result.err != nil {
			if errors.Is(result.err, explorer.ErrUserDecline) {
				state.SetStatus("Keystore selection canceled", false)
				return
			}
			state.SetStatus("Keystore picker error: "+result.err.Error(), true)
			return
		}
		if result.noPath || result.path == "" {
			state.SetStatus("Selected keystore path is unavailable on this platform", true)
			return
		}
		s.pendingKeystoreSource = result.path
		s.pendingClearKeystore = false
		state.SetStatus("Keystore selected (pending Save)", false)
	default:
	}

	for s.closeBtn.Clicked(gtx) {
		os.Exit(0)
	}
	for s.backBtn.Clicked(gtx) {
		s.pendingKeystoreSource = ""
		s.pendingClearKeystore = false
		state.SetScreen(ScreenHome)
	}
	for s.repoErrorOkBtn.Clicked(gtx) {
		s.showRepoErrorModal = false
	}
	for s.repoInfoOkBtn.Clicked(gtx) {
		s.showRepoInfoModal = false
		state.SetScreen(ScreenHome)
	}
	for s.sourceMorpheBtn.Clicked(gtx) {
		s.sourceMode.Value = sourceKeyMorphe
	}
	for s.sourceCustomBtn.Clicked(gtx) {
		s.sourceMode.Value = sourceKeyCustom
	}
	for s.releaseStableBtn.Clicked(gtx) {
		s.releaseMode.Value = radioKeyStable
	}
	for s.releasePreBtn.Clicked(gtx) {
		s.releaseMode.Value = radioKeyDev
	}
	for s.saveBtn.Clicked(gtx) {
		s.repoInfoMessage = ""
		s.repoInfoRepo = ""
		s.repoInfoPreferred = ""
		s.repoInfoFallback = ""
		s.showRepoInfoModal = false
		newMode := config.ModeStable
		if s.releaseMode.Value == radioKeyDev {
			newMode = config.ModeDev
		}
		newAutoUpdate := s.autoUpdate.Value

		newCustomPatchesRepo := ""
		if s.sourceMode.Value == sourceKeyCustom {
			newCustomPatchesRepo = strings.TrimSpace(s.customPatchesRepo.Text())
			if newCustomPatchesRepo == "" {
				s.repoValidationError = "You need to enter a GitHub repository (owner/repo) or a GitHub URL."
				s.showRepoErrorModal = true
				s.repoErrorOkBtn = widget.Clickable{}
				state.SetStatus("Invalid patches repo", true)
				return
			}

			normalizedRepo, err := github.NormalizeRepo(newCustomPatchesRepo)
			if err != nil {
				s.repoValidationError = err.Error()
				s.showRepoErrorModal = true
				s.repoErrorOkBtn = widget.Clickable{}
				state.SetStatus("Invalid patches repo", true)
				return
			}

			client := github.NewClient()
			preferredDev := s.releaseMode.Value == radioKeyDev
			patchesRelease, err := client.GetPatchesReleaseFromRepo(normalizedRepo, preferredDev)
			if err != nil {
				s.repoValidationError = err.Error()
				s.showRepoErrorModal = true
				s.repoErrorOkBtn = widget.Clickable{}
				state.SetStatus("Invalid patches repo", true)
				return
			}

			if patchesRelease.IsDev != preferredDev {
				preferredText := "stable"
				fallbackText := "pre-release"
				if preferredDev {
					preferredText = "pre-release"
					fallbackText = "stable"
				}
				s.repoInfoMessage = fmt.Sprintf("Repository %s has no %s channel. %s channel will be used instead.", normalizedRepo, preferredText, fallbackText)
				s.repoInfoRepo = normalizedRepo
				s.repoInfoPreferred = preferredText
				s.repoInfoFallback = fallbackText
				s.showRepoInfoModal = true
				s.repoInfoOkBtn = widget.Clickable{}
			}

			newCustomPatchesRepo = normalizedRepo
		}

		s.repoValidationError = ""
		s.showRepoErrorModal = false

		changed := false
		if state.Config.Mode != newMode {
			state.Config.Mode = newMode
			changed = true
		}
		if state.Config.AutoUpdate != newAutoUpdate {
			state.Config.AutoUpdate = newAutoUpdate
			state.AppPromptForUpdate = state.Config.AutoUpdate && state.AppUpdateAvailable
			changed = true
		}

		if state.Config.CustomPatchesRepo != newCustomPatchesRepo {
			state.Config.CustomPatchesRepo = newCustomPatchesRepo
			changed = true
		}

		if s.pendingClearKeystore {
			if err := removeImportedKeystoreFromAppData(state.Config.CustomKeystorePath); err != nil {
				state.SetStatus("Failed to remove imported keystore: "+err.Error(), true)
				return
			}
			if state.Config.CustomKeystorePath != "" {
				changed = true
			}
			state.Config.CustomKeystorePath = ""
		}

		if s.pendingKeystoreSource != "" {
			copiedPath, err := copyKeystoreToAppData(s.pendingKeystoreSource)
			if err != nil {
				state.SetStatus("Failed to import keystore: "+err.Error(), true)
				return
			}
			if state.Config.CustomKeystorePath != "" && state.Config.CustomKeystorePath != copiedPath {
				_ = removeImportedKeystoreFromAppData(state.Config.CustomKeystorePath)
			}
			if state.Config.CustomKeystorePath != copiedPath {
				changed = true
			}
			state.Config.CustomKeystorePath = copiedPath
		}

		if !changed {
			state.SetStatus("No changes to save", false)
			s.pendingKeystoreSource = ""
			s.pendingClearKeystore = false
			if s.showRepoInfoModal {
				return
			}
			state.SetScreen(ScreenHome)
			return
		}

		if err := state.Config.Save(); err != nil {
			state.SetStatus("Error saving: "+err.Error(), true)
		} else {
			state.SetStatus("Settings saved", false)
			s.lastMode = state.Config.Mode
			s.lastAutoUpdate = state.Config.AutoUpdate
			s.lastCustomPatchesRepo = state.Config.CustomPatchesRepo
			s.lastKeystorePath = state.Config.CustomKeystorePath
			s.pendingKeystoreSource = ""
			s.pendingClearKeystore = false
			s.suppressRepoInputSync = true
			s.customPatchesRepo.SetText(state.Config.CustomPatchesRepo)
		}
		if s.showRepoInfoModal {
			return
		}
		state.SetScreen(ScreenHome)
	}
	for s.keystoreBtn.Clicked(gtx) {
		if s.explorer == nil {
			state.SetStatus("Keystore picker is unavailable", true)
			continue
		}
		go func() {
			rc, err := s.explorer.ChooseFile(".keystore", ".jks")
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
	for s.clearKeyBtn.Clicked(gtx) {
		s.pendingKeystoreSource = ""
		s.pendingClearKeystore = true
		state.SetStatus("Keystore will be cleared on Save", false)
	}
}

func copyKeystoreToAppData(sourcePath string) (string, error) {
	appDir, err := storage.AppDataDir("vary")
	if err != nil {
		return "", err
	}
	if err := storage.EnsureDir(appDir); err != nil {
		return "", err
	}

	name := filepath.Base(sourcePath)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "custom.keystore"
	}
	if strings.EqualFold(name, "vary.keystore") {
		name = "custom-vary.keystore"
	}
	destinationPath := filepath.Join(appDir, name)

	if filepath.Clean(sourcePath) == filepath.Clean(destinationPath) {
		return destinationPath, nil
	}

	src, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(destinationPath)
	if err != nil {
		return "", err
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}

	return destinationPath, nil
}

func removeImportedKeystoreFromAppData(path string) error {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil
	}

	appDir, err := storage.AppDataDir("vary")
	if err != nil {
		return err
	}
	appDir = filepath.Clean(appDir)

	rel, err := filepath.Rel(appDir, path)
	if err != nil {
		return nil
	}
	if rel == "." || strings.HasPrefix(rel, "..") {
		return nil
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
