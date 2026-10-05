package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// typeExpr simulates typing character by character (appending at the end,
// as the editor does when typing with the caret at the end).
func typeExpr(c *calculator, s string) {
	for _, r := range s {
		prev := c.Display()
		if prev == "" {
			// Empty display: build directly (avoid a grouping prefix).
			c.UserEdit(string(r), "")
			continue
		}
		c.UserEdit(prev+string(r), prev)
	}
}

func TestBasicOps(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "12+3")
	// Building: literal expression in the display, nothing on top.
	if c.Display() != "12+3" || c.Expr() != "" {
		t.Fatalf("building = %q expr %q, want 12+3 / ''", c.Display(), c.Expr())
	}
	c.Equals()
	if c.Display() != "15" {
		t.Fatalf("12+3 = %q, want 15", c.Display())
	}
	if c.Expr() != "12+3=" {
		t.Fatalf("hist = %q, want '12+3='", c.Expr())
	}
}

func TestChain(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "2+3×4")
	c.Equals()
	// Immediate semantics, Windows standard style: (2+3)×4.
	if c.Display() != "20" {
		t.Fatalf("2+3×4 = %q, want 20", c.Display())
	}
}

func TestRepeatEquals(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "2+3")
	c.Equals()
	c.Equals()
	if c.Display() != "8" {
		t.Fatalf("repeat = %q, want 8", c.Display())
	}
}

func TestDecimalComma(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "1,5+2,5")
	c.Equals()
	if c.Display() != "4" {
		t.Fatalf("1,5+2,5 = %q, want 4", c.Display())
	}
}

func TestDivByZero(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "5÷0")
	c.Equals()
	if !c.err {
		t.Fatalf("expected error, display %q", c.Display())
	}
	c.ClearAll()
	if c.Display() != "" {
		t.Fatalf("after C = %q, want empty", c.Display())
	}
}

func TestPercentPlus(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "200+10%")
	// % stays literal in the expression; nothing computed yet.
	if c.Display() != "200+10%" {
		t.Fatalf("building = %q, want 200+10%%", c.Display())
	}
	c.Equals()
	if c.Display() != "220" {
		t.Fatalf("200+10%% = %q, want 220", c.Display())
	}
	if c.Expr() != "200+10%=" {
		t.Fatalf("hist = %q, want '200+10%%='", c.Expr())
	}
}

func TestUnary(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "4")
	c.Square()
	if c.Display() != "16" {
		t.Fatalf("4² = %q, want 16", c.Display())
	}
	c.ClearAll()
	typeExpr(c, "9")
	c.Sqrt()
	if c.Display() != "3" {
		t.Fatalf("√9 = %q, want 3", c.Display())
	}
	c.ClearAll()
	typeExpr(c, "4")
	c.Inverse()
	if c.Display() != "0.25" {
		t.Fatalf("1/4 = %q, want 0.25", c.Display())
	}
	c.ClearAll()
	typeExpr(c, "12+9")
	c.Sqrt()
	if c.Display() != "12+3" {
		t.Fatalf("12+√9 = %q, want 12+3", c.Display())
	}
}

func TestBackspaceAndSign(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "123")
	c.Backspace()
	if c.Display() != "12" {
		t.Fatalf("backspace = %q, want 12", c.Display())
	}
	c.ToggleSign()
	if c.Display() != "−12" {
		t.Fatalf("sign = %q, want −12", c.Display())
	}
	c.ToggleSign()
	if c.Display() != "12" {
		t.Fatalf("sign2 = %q, want 12", c.Display())
	}
	// +/- inside an expression: flips the final number.
	c.ClearAll()
	typeExpr(c, "12+9")
	c.ToggleSign()
	if c.Display() != "12+−9" {
		t.Fatalf("sign in expr = %q, want 12+−9", c.Display())
	}
	c.Equals()
	if c.Display() != "3" {
		t.Fatalf("12+-9 = %q, want 3", c.Display())
	}
}

func TestViewSmoke(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 340, 580)
	for _, label := range []string{"7", "+", "8", "="} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("click %q: %v (texts %q)", label, err, tt.Texts())
		}
	}
	// The display is an input: its text is not in Texts(), validate on the engine.
	if a.calc.Display() != "15" || a.edit != "15" {
		t.Fatalf("display = %q buffer %q, want 15", a.calc.Display(), a.edit)
	}
	if !tt.HasText("7+8=") {
		t.Fatalf("expected expr in %q", tt.Texts())
	}
}

func TestFilterEntry(t *testing.T) {
	c := newCalculator() // US default: dot
	cases := map[string]string{
		"247.227915": "247.227915", // all digits, US default
		"247,227915": "247.227915", // comma becomes a dot in US mode
		"12a3":       "123",
		"1,2,3":      "1.23",
		".5":         "0.5",
		"007":        "7",
		"":           "",
		"-5":         "-5",
		"1,5E+3":     "1.5E+3",
		"abc":        "",
	}
	for in, want := range cases {
		if got := c.filterEntry(in); got != want {
			t.Errorf("filterEntry(%q) = %q, want %q", in, got, want)
		}
	}
	br := newCalculator()
	br.SetSep(',')
	for in, want := range map[string]string{
		"247.227915": "247,227915",
		"1.5":        "1,5",
	} {
		if got := br.filterEntry(in); got != want {
			t.Errorf("BR filterEntry(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUserEditTyping(t *testing.T) {
	c := newCalculator()
	// Builds literally: the operator stays where it is typed.
	c.UserEdit("1", "")
	c.UserEdit("12", "1")
	c.UserEdit("12+", "12")
	if c.Display() != "12+" || c.Expr() != "" {
		t.Fatalf("building = %q expr %q, want 12+ / ''", c.Display(), c.Expr())
	}
	c.UserEdit("12+3", "12+")
	if c.Display() != "12+3" {
		t.Fatalf("building2 = %q, want 12+3", c.Display())
	}
	c.Equals()
	if c.Display() != "15" {
		t.Fatalf("12+3 = %q, want 15", c.Display())
	}
}

func TestUserEditOps(t *testing.T) {
	c := newCalculator()
	c.UserEdit("12", "")
	// Changing the trailing operator: replaces it.
	c.UserEdit("12+", "12")
	if c.Display() != "12+" {
		t.Fatalf("op = %q, want 12+", c.Display())
	}
	c.UserEdit("12-", "12+")
	if c.Display() != "12−" {
		t.Fatalf("change op = %q, want 12−", c.Display())
	}
	c.UserEdit("12×", "12-")
	if c.Display() != "12×" {
		t.Fatalf("change op2 = %q, want 12×", c.Display())
	}
	// "-" after an operator becomes unary.
	c.UserEdit("12×-", "12×")
	if c.Display() != "12×−" {
		t.Fatalf("unary minus = %q, want 12×−", c.Display())
	}
	// Nothing computed until Enter.
	if c.Expr() != "" {
		t.Fatalf("expr = %q, want ''", c.Expr())
	}
}

func TestUserEditAfterEquals(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "2+3")
	c.Equals() // 5
	// Typed "2" over the result: starts a new calculation.
	c.UserEdit("52", "5")
	if c.Display() != "2" || c.Expr() != "" {
		t.Fatalf("new entry = %q expr %q, want 2 / ''", c.Display(), c.Expr())
	}
	// Operator over the result: continues from it.
	c2 := newCalculator()
	typeExpr(c2, "2+3")
	c2.Equals() // 5
	c2.UserEdit("5+", "5")
	if c2.Display() != "5+" || c2.Expr() != "" {
		t.Fatalf("continue = %q expr %q, want 5+ / ''", c2.Display(), c2.Expr())
	}
	c2.UserEdit("5+2", "5+")
	c2.Equals()
	if c2.Display() != "7" {
		t.Fatalf("5+2 = %q, want 7", c2.Display())
	}
}

func TestUserEditRepeatEquals(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "2+3")
	c.Equals() // 5
	c.UserEdit("5=", "5")
	if c.Display() != "8" {
		t.Fatalf("repeat = %q, want 8", c.Display())
	}
}

func TestUserEditPaste(t *testing.T) {
	c := newCalculator()
	c.UserEdit("247,227.915", "")
	if c.Display() != "247,227.915" {
		t.Fatalf("paste = %q, want 247,227.915", c.Display())
	}
}

func TestNormalizeGrouping(t *testing.T) {
	cases := map[string]string{
		"6,306.04879":  "6306.04879", // grouping + US decimal
		"6.306,04879":  "6306,04879", // grouping + BR decimal
		"1.000.000":    "1000000",    // repeated grouping
		"1,000,000":    "1000000",    // repeated grouping
		"1,000,000.5":  "1000000.5",  // grouping + decimal
		"1.5":          "1.5",        // a lone decimal does not change
		"1,5":          "1,5",        // a lone decimal does not change
		"":             "",           // empty does not change
		"1.000E+3":     "1.000E+3",   // a single separator is decimal, even with an exponent
		"1.000.000E+3": "1000000E+3", // repeated grouping with an exponent
		"247.227915":   "247.227915", // no grouping, does not change
		"12":           "12",         // no separator, does not change
	}
	for in, want := range cases {
		if got := normalizeGrouping(in); got != want {
			t.Errorf("normalizeGrouping(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUserEditGroupedPaste(t *testing.T) {
	c := newCalculator()
	c.UserEdit("6,306.04879", "")
	if c.Display() != "6,306.04879" {
		t.Fatalf("grouped paste US = %q, want 6,306.04879", c.Display())
	}
	br := newCalculator()
	br.SetSep(',')
	br.UserEdit("6,306.04879", "")
	if br.Display() != "6.306,04879" {
		t.Fatalf("grouped paste BR = %q, want 6.306,04879", br.Display())
	}
	// Typing a second separator: ignored, the number is kept.
	c2 := newCalculator()
	c2.UserEdit("12.5", "")
	c2.UserEdit("12.5,", "12.5")
	if c2.Display() != "12.5" {
		t.Fatalf("second sep typed = %q, want 12.5", c2.Display())
	}
}

func TestGroupedDisplayEdit(t *testing.T) {
	c := newCalculator()
	c.UserEdit("6,306.04879", "")
	if c.Display() != "6,306.04879" {
		t.Fatalf("display = %q, want 6,306.04879", c.Display())
	}
	// Typing at the end of the grouped display: the grouping comma drops out.
	c.UserEdit("6,306.048795", "6,306.04879")
	if c.Display() != "6,306.048795" {
		t.Fatalf("typed on grouped = %q, want 6,306.048795", c.Display())
	}
	// Erasing digits from the grouped display.
	c.UserEdit("6,306.04879", "6,306.048795")
	if c.Display() != "6,306.04879" {
		t.Fatalf("deleted on grouped = %q, want 6,306.04879", c.Display())
	}
	c.UserEdit("6,306", "6,306.04879")
	if c.Display() != "6,306" {
		t.Fatalf("deleted frac = %q, want 6,306", c.Display())
	}
}

func TestBuildExpr(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "2+2")
	// Building: literal expression in the display, nothing on top until Enter.
	if c.Display() != "2+2" || c.Expr() != "" {
		t.Fatalf("building = %q expr %q, want 2+2 / ''", c.Display(), c.Expr())
	}
	c.Equals()
	if c.Display() != "4" || c.Expr() != "2+2=" {
		t.Fatalf("after enter = %q expr %q, want 4 / '2+2='", c.Display(), c.Expr())
	}
	// CE erases the final number, keeping the operator.
	c2 := newCalculator()
	typeExpr(c2, "2+9")
	c2.ClearEntry()
	if c2.Display() != "2+" {
		t.Fatalf("after CE = %q, want '2+'", c2.Display())
	}
	c2.Equals() // no second operand: repeats the first
	if c2.Display() != "4" {
		t.Fatalf("2+ = %q, want 4", c2.Display())
	}
}

func TestOpThenEquals(t *testing.T) {
	c := newCalculator()
	typeExpr(c, "12+")
	c.Equals() // no second operand: repeats the first (12+12)
	if c.Display() != "24" {
		t.Fatalf("12+ = %q, want 24", c.Display())
	}
}

func TestEmptyStart(t *testing.T) {
	c := newCalculator()
	if c.Display() != "" || c.Expr() != "" {
		t.Fatalf("start display = %q expr %q, want empty", c.Display(), c.Expr())
	}
	// "=" with nothing: does nothing.
	c.Equals()
	if c.Display() != "" || c.Expr() != "" {
		t.Fatalf("empty equals = %q expr %q, want empty", c.Display(), c.Expr())
	}
	// Leading binary operator: ignored (only a unary "-" passes).
	c.UserEdit("+", "")
	if c.Display() != "" {
		t.Fatalf("leading + = %q, want empty", c.Display())
	}
	c.UserEdit("-", "")
	if c.Display() != "−" {
		t.Fatalf("leading - = %q, want −", c.Display())
	}
	c.ClearAll()
	typeExpr(c, "5")
	c.Equals()
	if c.Display() != "5" || c.Expr() != "5=" {
		t.Fatalf("bare 5 = %q expr %q, want 5 / '5='", c.Display(), c.Expr())
	}
	// Erasing everything goes back to empty.
	c.ClearAll()
	typeExpr(c, "7")
	c.Backspace()
	if c.Display() != "" {
		t.Fatalf("backspace-all = %q, want empty", c.Display())
	}
}

func TestSepToggle(t *testing.T) {
	c := newCalculator()
	if c.sep != '.' {
		t.Fatalf("default sep = %q, want '.'", c.sep)
	}
	typeExpr(c, "1.5")
	c.SetSep(',')
	if c.Display() != "1,5" {
		t.Fatalf("after SetSep display = %q, want 1,5", c.Display())
	}
	typeExpr(c, "2")
	if c.Display() != "1,52" {
		t.Fatalf("typing after sep = %q, want 1,52", c.Display())
	}
	c.SetSep('.')
	if c.Display() != "1.52" {
		t.Fatalf("back to dot = %q, want 1.52", c.Display())
	}
	// The expression being edited survives the switch (only the format changes).
	c2 := newCalculator()
	typeExpr(c2, "12+")
	c2.SetSep(',')
	if c2.Display() != "12+" {
		t.Fatalf("expr display = %q, want '12+'", c2.Display())
	}
	// 1/4 in BR mode uses a comma.
	c2.ClearAll()
	typeExpr(c2, "4")
	c2.Inverse()
	if c2.Display() != "0,25" {
		t.Fatalf("BR 1/4 = %q, want 0,25", c2.Display())
	}
}

// Typing 1..9 in sequence must not scramble (the caret stays at the end
// after each grouping rewrite — see ui.Box Key(editGen)).
func TestViewSequentialDigits(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 340, 580)
	if !tt.Focused("Result") {
		if err := tt.Click("Result"); err != nil {
			t.Fatalf("focus display: %v", err)
		}
	}
	tt.Command("selectAll")
	for _, d := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		tt.Type(d)
	}
	if a.calc.Display() != "123,456,789" || a.edit != "123,456,789" {
		t.Fatalf("sequential = %q buffer %q, want 123,456,789", a.calc.Display(), a.edit)
	}
}

func TestViewTyping(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 340, 580)
	if !tt.Focused("Result") {
		if err := tt.Click("Result"); err != nil {
			t.Fatalf("focus display: %v", err)
		}
	}
	tt.Command("selectAll")
	tt.Type("7")
	tt.Type("+")
	tt.Type("8")
	// Building: literal expression in the display, nothing on top until Enter.
	if a.calc.Display() != "7+8" || a.edit != "7+8" {
		t.Fatalf("building = %q buffer %q, want 7+8", a.calc.Display(), a.edit)
	}
	if a.calc.Expr() != "" {
		t.Fatalf("top = %q, want empty until Enter", a.calc.Expr())
	}
	tt.Key(0, ui.KeyEnter)
	if a.calc.Display() != "15" || a.edit != "15" {
		t.Fatalf("display = %q buffer %q, want 15", a.calc.Display(), a.edit)
	}
	// Esc clears everything.
	tt.Key(0, ui.KeyEscape)
	if a.calc.Display() != "" || a.edit != "" {
		t.Fatalf("after Esc display = %q buffer %q, want empty", a.calc.Display(), a.edit)
	}
}

func TestViewSelectAllDelete(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 340, 580)
	if !tt.Focused("Result") {
		if err := tt.Click("Result"); err != nil {
			t.Fatalf("focus display: %v", err)
		}
	}
	tt.Command("selectAll")
	tt.Type("123")
	// Ctrl+A + Backspace erases everything.
	tt.Command("selectAll")
	tt.Key(0, ui.KeyBackspace)
	if a.calc.Display() != "" || a.edit != "" {
		t.Fatalf("select-all+backspace = %q buffer %q, want empty", a.calc.Display(), a.edit)
	}
	// Also after "=" (fresh): select all and erase.
	tt.Type("2")
	tt.Type("+")
	tt.Type("3")
	tt.Key(0, ui.KeyEnter)
	if a.calc.Display() != "5" {
		t.Fatalf("2+3 = %q, want 5", a.calc.Display())
	}
	tt.Command("selectAll")
	tt.Key(0, ui.KeyBackspace)
	if a.calc.Display() != "" || a.edit != "" {
		t.Fatalf("after = select-all+backspace = %q buffer %q, want empty", a.calc.Display(), a.edit)
	}
}

func TestViewEnterNoNewline(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 340, 580)
	if !tt.Focused("Result") {
		if err := tt.Click("Result"); err != nil {
			t.Fatalf("focus display: %v", err)
		}
	}
	tt.Command("selectAll")
	tt.Type("2")
	tt.Type("+")
	tt.Type("3")
	tt.Key(0, ui.KeyEnter)
	if strings.Contains(a.edit, "\n") {
		t.Fatalf("enter inserted newline: %q", a.edit)
	}
	if a.calc.Display() != "5" {
		t.Fatalf("2+3 enter = %q, want 5", a.calc.Display())
	}
}

func TestViewSepToggle(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 340, 580)
	if a.calc.sep != '.' {
		t.Fatalf("default sep = %q, want '.'", a.calc.sep)
	}
	if err := tt.Click("1,5"); err != nil {
		t.Fatalf("click sep BR: %v", err)
	}
	if a.calc.sep != ',' {
		t.Fatalf("sep = %q, want ','", a.calc.sep)
	}
	if err := tt.Click("1.5"); err != nil {
		t.Fatalf("click sep US: %v", err)
	}
	if a.calc.sep != '.' {
		t.Fatalf("sep = %q, want '.'", a.calc.sep)
	}
}

func TestViewCopyPaste(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 340, 580)
	if !tt.Focused("Result") {
		if err := tt.Click("Result"); err != nil {
			t.Fatalf("focus display: %v", err)
		}
	}
	tt.Command("selectAll")
	tt.Type("247,227.915")
	if a.calc.Display() != "247,227.915" || a.edit != "247,227.915" {
		t.Fatalf("typed = %q buffer %q, want 247,227.915", a.calc.Display(), a.edit)
	}
	// Manual Ctrl+C via the edit menu: select and copy.
	tt.Command("selectAll")
	tt.Command("copy")
	if tt.Clipboard() != "247,227.915" {
		t.Fatalf("clipboard = %q, want 247,227.915", tt.Clipboard())
	}
	// Paste back after clearing.
	tt.Key(0, ui.KeyEscape)
	tt.Command("selectAll")
	tt.Command("paste")
	if a.calc.Display() != "247,227.915" || a.edit != "247,227.915" {
		t.Fatalf("pasted = %q buffer %q, want 247,227.915", a.calc.Display(), a.edit)
	}
	// Paste from outside with a dot: all digits show up.
	tt.SetClipboard("247,227.915")
	tt.Key(0, ui.KeyEscape)
	tt.Command("selectAll")
	tt.Command("paste")
	if a.calc.Display() != "247,227.915" || a.edit != "247,227.915" {
		t.Fatalf("external paste = %q buffer %q, want 247,227.915", a.calc.Display(), a.edit)
	}
	// Paste with grouping + decimal ("6,306.04879" → 6306.04879).
	tt.SetClipboard("6,306.04879")
	tt.Key(0, ui.KeyEscape)
	tt.Command("selectAll")
	tt.Command("paste")
	if a.calc.Display() != "6,306.04879" || a.edit != "6,306.04879" {
		t.Fatalf("grouped paste = %q buffer %q, want 6,306.04879", a.calc.Display(), a.edit)
	}
}
