package main

import (
	"log"
	"os"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type app struct {
	calc    *calculator
	edit    string // display buffer (input); synchronized with Display()
	editGen int    // display generation: new editor (caret at the end) on each rewrite of ours
	// sepIdx is the decimal separator in the UI: 0 = "1,5" (BR), 1 = "1.5" (US).
	sepIdx int
	// settingsOpen shows the settings modal (decimal separator, project link).
	settingsOpen bool
}

func newApp() *app {
	return &app{calc: newCalculator(), sepIdx: 1}
}

func openCalculatorWindow() {
	a := newApp()
	mygo.NewWindow(mygo.WindowOptions{
		Title:     "Calc",
		Width:     340,
		Height:    580,
		MinWidth:  300,
		MinHeight: 480,
		StateKey:  "calc",
		// CALC_FULLSCREEN=1 opens full screen without decorations: the
		// PortMaster port for the R36S runs the window under Weston's
		// kiosk shell, where a title bar would only waste the 640×480
		// screen — and GTK's own title bar looks up window-control icons
		// a minimal runtime may not have. Unset (the desktop default),
		// nothing changes.
		FullScreen: os.Getenv("CALC_FULLSCREEN") == "1",
		Frameless:  os.Getenv("CALC_FULLSCREEN") == "1",
		Content:    ui.View(a.view),
	})
}

// handleKeys handles the keys the text field does not consume on its own.
// Digits, operators and pasting go through the display (input) via
// Changed(), which is independent of the keyboard layout (ABNT2, US,
// numeric): the engine reacts to the character, not to the key. Enter (=)
// arrives via Submitted().
func (a *app) handleKeys(c *ui.Context) {
	// While the settings modal is open, Escape dismisses it (the modal
	// handles that itself) instead of clearing the calculation.
	if !a.settingsOpen && c.Shortcut(0, ui.KeyEscape) {
		a.calc.ClearAll()
	}
	// New window: Ctrl+N
	if c.Shortcut(ui.Cmd, ui.KeyN) {
		openCalculatorWindow()
	}
}

type btnKind int

const (
	btnDigit btnKind = iota
	btnOp
	btnAccent
)

// calcButton draws a Windows 11 Calculator-style button.
// It uses Box (not focusable) so Enter always means "=" and the physical
// keyboard keeps working after a click.
func calcButton(c *ui.Context, label string, kind btnKind, fontSize float32, bold bool) bool {
	t := c.Theme()

	b := ui.Box(c).Center().Grow(1).Radius(t.Radius).Cursor(ui.CursorPointer).Padding(6).MinHeight(44)
	b.AlignSelf(ui.Stretch)

	var bg, hov, pr ui.Color
	fg := t.Text

	switch kind {
	case btnAccent:
		bg, hov, pr = t.Accent, t.AccentHover, t.AccentPressed
		fg = t.AccentText
	case btnDigit:
		if t.Dark {
			bg = ui.Hex("#2b2b2b")
			hov = ui.Hex("#323232")
			pr = ui.Hex("#3a3a3a")
		} else {
			bg = ui.Hex("#ffffff")
			hov = ui.Hex("#f9f9f9")
			pr = ui.Hex("#f3f3f3")
		}
		fg = t.Text
	default: // btnOp
		bg, hov, pr = t.Surface, t.SurfaceHover, t.SurfacePressed
		fg = t.Text
	}

	pressed := b.Pressed()
	hovered := b.Hovered()
	cur := bg
	if pressed {
		cur = pr
	} else if hovered {
		cur = hov
	}
	b.Background(cur).TextColor(fg)
	if kind == btnDigit && !t.Dark {
		b.Border(1, t.Border)
	}

	b.Children(func() {
		if windowsIcons && label == "⌫" {
			drawBackspace(c, fg)
			return
		}
		tx := ui.Text(c, label).FontSize(fontSize).SingleLine()
		if bold {
			tx.Bold()
		}
	})
	return b.Clicked()
}

// windowsIcons draws the icons with shapes on Windows: Segoe UI has no
// glyphs for "⌫" (U+232B) and "⧉" (U+29C9), and the system font fallback
// may not find them — they showed up as boxes. The handheld port sets
// CALC_DRAWN_ICONS=1 for the same reason: its minimal font set (DejaVu)
// has ⌫ but not ⧉. On other systems the plain glyph is used.
var windowsIcons = runtime.GOOS == "windows" || os.Getenv("CALC_DRAWN_ICONS") == "1"

// githubURL is the project page linked from the settings modal.
const githubURL = "https://github.com/CBR0/calc"

// gearIcon opens the settings; githubIcon links to the project page.
// They are SVGs (not font glyphs) so they render on Windows and on minimal
// handheld font sets alike — see windowsIcons above.
var gearIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`))

var githubIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>`))

// drawBackspace draws the ⌫ key as a rounded box with "×"
// (U+00D7, present in Segoe UI).
func drawBackspace(c *ui.Context, col ui.Color) {
	ui.Box(c).Width(22).Height(15).Radius(3).Border(1, col).Center().Children(func() {
		ui.Text(c, "×").FontSize(12).Bold().TextColor(col).SingleLine()
	})
}

// drawNewWindow draws ⧉ as two overlapping squares.
func drawNewWindow(c *ui.Context, col ui.Color) {
	ui.Box(c).Width(15).Height(15).Children(func() {
		ui.Box(c).Absolute().Top(0).Right(0).Width(10).Height(10).Radius(2).Border(1, col)
		ui.Box(c).Absolute().Bottom(0).Left(0).Width(10).Height(10).Radius(2).Border(1, col)
	})
}

// fitFont picks a discrete font step so the text fits the width.
// Fixed steps (no continuous values) avoid jitter on every keystroke.
func fitFont(c *ui.Context, s string, maxW float32) float32 {
	steps := []float32{42, 36, 32, 28, 24, 20, 18, 16, 14}
	if s == "" || maxW <= 0 {
		return steps[0]
	}
	span := ui.Span{Text: s, Size: steps[0], Weight: 700, Features: "tnum"}
	w, _ := c.MeasureText(0, span)
	if w <= maxW {
		return steps[0]
	}
	ideal := maxW / w * steps[0]
	for _, st := range steps[1:] {
		if st <= ideal {
			return st
		}
	}
	return steps[len(steps)-1]
}

// syncDisplay mirrors the engine into the field, creating a new editor
// (caret at the end) only when the text really changed — no rewrite, no
// caret jump.
func (a *app) syncDisplay(c *ui.Context) {
	if d := a.calc.Display(); a.edit != d {
		a.edit = d
		a.editGen++
		c.Invalidate()
	}
}

func (a *app) view(c *ui.Context) {
	a.handleKeys(c)
	t := c.Theme()
	c0 := a.calc

	ui.Column(c).Fill().Padding(12).Gap(6).Children(func() {
		// Top bar: settings + new window (the decimal separator lives in
		// the settings modal now).
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			settingsBtn := ui.Box(c).Center().Radius(t.Radius).Cursor(ui.CursorPointer).Padding(6, 10).Tooltip("Settings")
			settingsBtn.Label("Settings")
			if settingsBtn.Hovered() {
				settingsBtn.Background(t.SurfaceHover)
			} else {
				settingsBtn.Background(t.Surface)
			}
			settingsBtn.Children(func() {
				ui.Icon(c, gearIcon).Size(15, 15).TextColor(t.Text)
			})
			if settingsBtn.Clicked() {
				a.settingsOpen = true
			}
			ui.Spacer(c)
			newWin := ui.Box(c).Center().Radius(t.Radius).Cursor(ui.CursorPointer).Padding(6, 10).Tooltip("New window (Ctrl+N)")
			newWin.Label("New window")
			if newWin.Hovered() {
				newWin.Background(t.SurfaceHover)
			} else {
				newWin.Background(t.Surface)
			}
			newWin.Children(func() {
				if windowsIcons {
					drawNewWindow(c, t.Text)
					return
				}
				ui.Text(c, "⧉").FontSize(15).TextColor(t.Text)
			})
			if newWin.Clicked() {
				openCalculatorWindow()
			}
		})

		// Display: small expression + large input aligned to the right.
		// The display IS an input: one can select part of the result,
		// copy (Ctrl+C), paste (Ctrl+V, e.g. 247.227915) and type into it.
		expr := c0.Expr()
		if expr == "" {
			expr = " "
		}
		ui.Text(c, expr).FontSize(14).TextColor(t.TextMuted).TextAlign(ui.End).SingleLine().MinHeight(20)

		prev := a.edit
		ww, _ := c.Size()
		fs := fitFont(c, prev, ww-40)
		// Wrapper with Key: each rewrite of OURS (editGen++) recreates the
		// editor with the caret at the end — without it the caret lags on
		// grouping and the digits come out of order ("12345"→"12354").
		// Plain typing does not rewrite.
		ui.Box(c.Key(a.editGen)).FillWidth().Children(func() {
			disp := ui.TextAreaBase(c, &a.edit).Label("Result").AutoFocus().NoWrap()
			// Focus always on the display so the physical keyboard lands
			// here — except while the settings modal is open, so its
			// controls keep the focus.
			if !a.settingsOpen {
				disp.Focus()
			}
			// Enter in the display = "=" (consumes it before the editor, which
			// would break the line).
			disp.HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEnter {
					a.calc.Equals()
					a.edit = a.calc.Display()
					a.editGen++
					return true
				}
				return false
			})
			// Height(84): the TextArea (multiline) keeps the caret in view by
			// scrolling the content when the line does not fit the content
			// box. At the largest font (42px) the line is ~58px; with 10+10
			// vertical padding, 84 leaves 64px — so reveal does not scroll and
			// the text does not move up/down on every keystroke. Do not shrink
			// it without checking this.
			disp.Padding(10, 8).Height(84).Radius(t.Radius)
			disp.FontSize(fs).Bold().TextAlign(ui.End)
			disp.FontFeatures("tnum")
			disp.Tooltip("Type, select and copy (Ctrl+C / Ctrl+V)")
			if c0.err {
				disp.TextColor(t.Danger)
			}
			if disp.Changed() {
				// Typing/pasting/erasing: build the literal expression
				// ("2+2" appears where it is typed); it only evaluates on
				// Enter/=.
				c0.UserEdit(a.edit, prev)
				a.syncDisplay(c)
			} else {
				a.syncDisplay(c)
			}
		})
		ui.Box(c).Height(8)

		// 4x6 button grid, Windows style
		ui.Column(c).Grow(1).Gap(6).Children(func() {
			// Row 1: % CE C ⌫
			ui.Row(c).Grow(1).Gap(6).AlignItems(ui.Stretch).Children(func() {
				if calcButton(c, "%", btnOp, 17, false) {
					c0.UserEdit(a.edit+"%", a.edit)
				}
				if calcButton(c, "CE", btnOp, 15, false) {
					c0.ClearEntry()
				}
				if calcButton(c, "C", btnOp, 15, false) {
					c0.ClearAll()
				}
				if calcButton(c, "⌫", btnOp, 18, false) {
					c0.Backspace()
				}
			})
			// Row 2: 1/x x² √ ÷
			ui.Row(c).Grow(1).Gap(6).AlignItems(ui.Stretch).Children(func() {
				if calcButton(c, "1/x", btnOp, 16, false) {
					c0.Inverse()
				}
				if calcButton(c, "x²", btnOp, 16, false) {
					c0.Square()
				}
				if calcButton(c, "√", btnOp, 18, false) {
					c0.Sqrt()
				}
				if calcButton(c, "÷", btnOp, 20, false) {
					c0.UserEdit(a.edit+"÷", a.edit)
				}
			})
			// Row 3: 7 8 9 ×
			ui.Row(c).Grow(1).Gap(6).AlignItems(ui.Stretch).Children(func() {
				if calcButton(c, "7", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"7", a.edit)
				}
				if calcButton(c, "8", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"8", a.edit)
				}
				if calcButton(c, "9", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"9", a.edit)
				}
				if calcButton(c, "×", btnOp, 20, false) {
					c0.UserEdit(a.edit+"×", a.edit)
				}
			})
			// Row 4: 4 5 6 −
			ui.Row(c).Grow(1).Gap(6).AlignItems(ui.Stretch).Children(func() {
				if calcButton(c, "4", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"4", a.edit)
				}
				if calcButton(c, "5", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"5", a.edit)
				}
				if calcButton(c, "6", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"6", a.edit)
				}
				if calcButton(c, "−", btnOp, 20, false) {
					c0.UserEdit(a.edit+"−", a.edit)
				}
			})
			// Row 5: 1 2 3 +
			ui.Row(c).Grow(1).Gap(6).AlignItems(ui.Stretch).Children(func() {
				if calcButton(c, "1", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"1", a.edit)
				}
				if calcButton(c, "2", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"2", a.edit)
				}
				if calcButton(c, "3", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"3", a.edit)
				}
				if calcButton(c, "+", btnOp, 20, false) {
					c0.UserEdit(a.edit+"+", a.edit)
				}
			})
			// Row 6: +/- 0 sep =
			ui.Row(c).Grow(1).Gap(6).AlignItems(ui.Stretch).Children(func() {
				if calcButton(c, "+/-", btnDigit, 16, true) {
					c0.ToggleSign()
				}
				if calcButton(c, "0", btnDigit, 18, true) {
					c0.UserEdit(a.edit+"0", a.edit)
				}
				decLabel := string([]byte{c0.sep})
				if calcButton(c, decLabel, btnDigit, 18, true) {
					c0.UserEdit(a.edit+string([]byte{c0.sep}), a.edit)
				}
				if calcButton(c, "=", btnAccent, 20, true) {
					c0.Equals()
				}
			})
		})
	})

	a.settingsModal(c)
}

// settingsModal shows the app settings over a dimmed window: the decimal
// separator switch and a link to the project on GitHub. Clicking outside
// it or pressing Escape dismisses it.
func (a *app) settingsModal(c *ui.Context) {
	t := c.Theme()
	c0 := a.calc
	ui.Modal(c, &a.settingsOpen, func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Settings").Bold().FontSize(17)
			ui.Spacer(c)
			closeBtn := ui.Box(c).Center().Radius(t.Radius).Cursor(ui.CursorPointer).Padding(4, 10).Tooltip("Close")
			closeBtn.Label("Close")
			if closeBtn.Hovered() {
				closeBtn.Background(t.SurfaceHover)
			} else {
				closeBtn.Background(t.Surface)
			}
			closeBtn.Children(func() {
				ui.Text(c, "×").FontSize(15).TextColor(t.Text)
			})
			if closeBtn.Clicked() {
				a.settingsOpen = false
			}
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Decimal separator").FontSize(14)
			ui.Spacer(c)
			seg := ui.Segmented(c, &a.sepIdx, "1,5", "1.5")
			seg.Label("Decimal separator").Tooltip("Dot (US) or comma (BR)")
			if seg.Changed() {
				if a.sepIdx == 0 {
					c0.SetSep(',')
				} else {
					c0.SetSep('.')
				}
				a.syncDisplay(c)
			}
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Project on GitHub").FontSize(14)
			ui.Spacer(c)
			gh := ui.Box(c).Center().Radius(t.Radius).Cursor(ui.CursorPointer).Padding(6, 10).Tooltip("Open the project on GitHub")
			gh.Label("GitHub")
			if gh.Hovered() {
				gh.Background(t.SurfaceHover)
			} else {
				gh.Background(t.Surface)
			}
			gh.Children(func() {
				ui.Icon(c, githubIcon).Size(16, 16).TextColor(t.Text)
			})
			if gh.Clicked() {
				c.OpenURL(githubURL)
			}
		})
	})
}

func main() {
	// No RequestSingleInstanceLock on purpose: every time the launcher
	// opens "Calc", a fresh process with its own window is born. Inside the
	// app, the ⧉ button / Ctrl+N opens more windows in the same process.
	mygo.App.WhenReady(openCalculatorWindow)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
