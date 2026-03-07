package app

import (
	"errors"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vary/internal/config"
	"vary/internal/storage"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
)

type SettingsScreen struct {
	releaseMode           widget.Enum
	saveBtn               widget.Clickable
	keystoreBtn           widget.Clickable
	clearKeyBtn           widget.Clickable
	backBtn               widget.Clickable
	closeBtn              widget.Clickable
	mui                   *material.Theme
	backIcon              *widget.Icon
	closeIcon             *widget.Icon
	lastMode              config.Mode
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
	if state.Config.CustomKeystorePath != s.lastKeystorePath {
		s.lastKeystorePath = state.Config.CustomKeystorePath
		s.pendingKeystoreSource = ""
		s.pendingClearKeystore = false
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
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(24), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									label := material.Body1(s.mui, "Custom keystore")
									label.Color = th.Text
									return label.Layout(gtx)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								keystoreText := "No custom keystore selected"
								if s.pendingClearKeystore {
									keystoreText = "Will be cleared on Save"
								} else if s.pendingKeystoreSource != "" {
									keystoreText = filepath.Base(s.pendingKeystoreSource) + " (pending save)"
								} else if state.Config.CustomKeystorePath != "" {
									keystoreText = filepath.Base(state.Config.CustomKeystorePath)
								}
								body := material.Body2(s.mui, keystoreText)
								body.Color = th.TextMuted
								return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, body.Layout)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								warning := material.Body2(s.mui, "If no custom keystore is selected, one will be generated automatically.")
								warning.Color = th.TextMuted
								return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, warning.Layout)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return s.button(gtx, th, "Select keystore", &s.keystoreBtn)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return s.button(gtx, th, "Clear", &s.clearKeyBtn)
									}),
								)
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
	appDir, err := storage.AppDataDir("Vary")
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

	appDir, err := storage.AppDataDir("Vary")
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
