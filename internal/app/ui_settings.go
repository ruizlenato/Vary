package app

import (
	"errors"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vary/internal/config"
	"vary/internal/storage"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
)

type SettingsScreen struct {
	releaseMode           widget.Enum
	autoUpdate            widget.Bool
	saveBtn               widget.Clickable
	keystoreBtn           widget.Clickable
	clearKeyBtn           widget.Clickable
	backBtn               widget.Clickable
	closeBtn              widget.Clickable
	mui                   *material.Theme
	backIcon              *widget.Icon
	closeIcon             *widget.Icon
	lastMode              config.Mode
	lastAutoUpdate        bool
	lastKeystorePath      string
	pendingKeystoreSource string
	pendingClearKeystore  bool
	explorer              *explorer.Explorer
	pickResults           chan filePickResult
}

func NewSettingsScreen(ex *explorer.Explorer) *SettingsScreen {
	return &SettingsScreen{
		backIcon:    mustIcon(backArrowIconVG),
		closeIcon:   mustIcon(closeIconVG),
		mui:         material.NewTheme(),
		explorer:    ex,
		pickResults: make(chan filePickResult, 1),
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
	if state.Config.AutoUpdate != s.lastAutoUpdate {
		s.lastAutoUpdate = state.Config.AutoUpdate
		s.autoUpdate.Value = state.Config.AutoUpdate
	}
	if state.Config.CustomKeystorePath != s.lastKeystorePath {
		s.lastKeystorePath = state.Config.CustomKeystorePath
		s.pendingKeystoreSource = ""
		s.pendingClearKeystore = false
	}

	originalConstraints := gtx.Constraints

	layout.Stack{}.Layout(gtx,

		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				contentWidth := min(gtx.Constraints.Max.X-gtx.Dp(unit.Dp(48)), gtx.Dp(unit.Dp(760)))
				if contentWidth < gtx.Dp(unit.Dp(280)) {
					contentWidth = gtx.Constraints.Max.X - gtx.Dp(unit.Dp(24))
				}
				narrow := contentWidth < gtx.Dp(unit.Dp(540))
				return layout.Inset{
					Top:    unit.Dp(28),
					Bottom: unit.Dp(28),
					Left:   unit.Dp(12),
					Right:  unit.Dp(12),
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
							return layout.Inset{Top: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return s.layoutSettingsCard(gtx, th, contentWidth, "Updates", "Choose how Vary checks and applies updates for Vary itself.", func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											label := material.Body1(s.mui, "Release channel")
											label.Color = th.Text
											return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, label.Layout)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return s.radioOption(gtx, th, contentWidth, radioKeyStable, "Morphe (Stable)")
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return s.radioOption(gtx, th, contentWidth, radioKeyDev, "Morphe Dev (Pre-release)")
											})
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return layout.Inset{Top: unit.Dp(20), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												check := material.CheckBox(s.mui, &s.autoUpdate, "Automatic updates")
												check.Color = th.Text
												check.IconColor = th.Primary
												return check.Layout(gtx)
											})
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											help := material.Body2(s.mui, "When enabled, Vary downloads and applies updates for Vary itself automatically when a new version is found.")
											help.Color = th.TextMuted
											return help.Layout(gtx)
										}),
									)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
							return layout.Inset{Top: unit.Dp(20), Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									width := min(gtx.Dp(unit.Dp(220)), contentWidth)
									if narrow {
										width = min(gtx.Dp(unit.Dp(220)), contentWidth-gtx.Dp(unit.Dp(24)))
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
			return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, subtitle.Layout)
		}),
	)
}

func (s *SettingsScreen) layoutSettingsCard(gtx layout.Context, th *Theme, maxWidth int, titleText, subtitleText string, content layout.Widget) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		width := min(maxWidth, gtx.Dp(unit.Dp(560)))
		if width > gtx.Constraints.Max.X {
			width = gtx.Constraints.Max.X
		}
		cardGtx := gtx
		cardGtx.Constraints.Min.X = width
		cardGtx.Constraints.Max.X = width
		return layoutMeasuredOutlinedSurface(cardGtx, unit.Dp(8), color.NRGBA{R: 78, G: 78, B: 78, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(18), Bottom: unit.Dp(18), Left: unit.Dp(18), Right: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.Body1(s.mui, titleText)
						title.Color = th.Text
						return title.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						subtitle := material.Body2(s.mui, subtitleText)
						subtitle.Color = th.TextMuted
						return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(18)}.Layout(gtx, subtitle.Layout)
					}),
					layout.Rigid(content),
				)
			})
		})
	})
}

func (s *SettingsScreen) layoutDualButtons(gtx layout.Context, th *Theme, stacked bool) layout.Dimensions {
	buttonWidth := min(gtx.Dp(unit.Dp(210)), (gtx.Constraints.Max.X-gtx.Dp(unit.Dp(10)))/2)
	if stacked || buttonWidth < gtx.Dp(unit.Dp(120)) {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					width := min(gtx.Dp(unit.Dp(220)), gtx.Constraints.Max.X-gtx.Dp(unit.Dp(12)))
					return s.actionButton(gtx, th, "Select keystore", &s.keystoreBtn, width)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						width := min(gtx.Dp(unit.Dp(220)), gtx.Constraints.Max.X-gtx.Dp(unit.Dp(12)))
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
				return layout.Spacer{Width: unit.Dp(10)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.actionButton(gtx, th, "Clear", &s.clearKeyBtn, buttonWidth)
			}),
		)
	})
}

func (s *SettingsScreen) radioOption(gtx layout.Context, th *Theme, maxWidth int, key, label string) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		width := min(gtx.Dp(unit.Dp(320)), maxWidth-gtx.Dp(unit.Dp(24)))
		if width > gtx.Constraints.Max.X {
			width = gtx.Constraints.Max.X
		}
		if width < gtx.Dp(unit.Dp(180)) {
			width = gtx.Constraints.Max.X
		}
		optionGtx := gtx
		optionGtx.Constraints = layout.Exact(image.Pt(width, gtx.Dp(unit.Dp(52))))
		return layoutOutlinedSurface(optionGtx, unit.Dp(6), color.NRGBA{R: 64, G: 64, B: 64, A: 255}, color.NRGBA{R: 10, G: 10, B: 10, A: 255}, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				radioBtn := material.RadioButton(s.mui, &s.releaseMode, key, label)
				radioBtn.Color = th.Text
				radioBtn.IconColor = th.Primary
				return layout.Center.Layout(gtx, radioBtn.Layout)
			})
		})
	})
}

func (s *SettingsScreen) actionButton(gtx layout.Context, th *Theme, text string, btn *widget.Clickable, width int) layout.Dimensions {
	height := gtx.Dp(unit.Dp(44))
	if width <= 0 {
		width = gtx.Dp(unit.Dp(180))
	}
	return layout.Inset{}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Exact(image.Pt(width, height))
		return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			borderInset := gtx.Dp(unit.Dp(1))
			cornerRadius := gtx.Dp(unit.Dp(6))
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
				label := material.Label(s.mui, unit.Sp(14), text)
				label.Color = th.Text
				return label.Layout(gtx)
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
	for s.saveBtn.Clicked(gtx) {
		newMode := config.ModeStable
		if s.releaseMode.Value == radioKeyDev {
			newMode = config.ModeDev
		}

		changed := false
		if state.Config.Mode != newMode {
			state.Config.Mode = newMode
			changed = true
		}
		if state.Config.AutoUpdate != s.autoUpdate.Value {
			state.Config.AutoUpdate = s.autoUpdate.Value
			state.AppPromptForUpdate = state.Config.AutoUpdate && state.AppUpdateAvailable
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
			state.SetScreen(ScreenHome)
			return
		}

		if err := state.Config.Save(); err != nil {
			state.SetStatus("Error saving: "+err.Error(), true)
		} else {
			state.SetStatus("Settings saved", false)
			s.lastMode = state.Config.Mode
			s.lastAutoUpdate = state.Config.AutoUpdate
			s.lastKeystorePath = state.Config.CustomKeystorePath
			s.pendingKeystoreSource = ""
			s.pendingClearKeystore = false
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
