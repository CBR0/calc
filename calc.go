package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// calculator in expression-entry mode: the text is the literal expression
// being built ("12+3.5"), without grouping (canonical form); the display
// groups it by token. Nothing is computed until Enter/=, which evaluates
// with immediate semantics (Windows standard style, no precedence) and
// shows "expr=" on top and the result below.
type calculator struct {
	text      string // canonical expression being edited; "" = empty
	hist      string // canonical history after Enter, e.g. "2+2="; "" = none
	evaluated bool   // display shows a freshly computed result
	lastOp    string // last binary operation (to repeat "=")
	lastVal   float64
	err       bool
	errMsg    string
	sep       byte // decimal separator: '.' (US) or ',' (BR)
}

func newCalculator() *calculator {
	return &calculator{sep: '.'}
}

// SetSep switches the decimal separator, converting expression and history
// (both canonical, without grouping, so the switch is safe).
func (c *calculator) SetSep(sep byte) {
	if sep != '.' && sep != ',' {
		return
	}
	if sep == c.sep {
		return
	}
	old := string([]byte{c.sep})
	nw := string([]byte{sep})
	c.sep = sep
	c.text = strings.ReplaceAll(c.text, old, nw)
	c.hist = strings.ReplaceAll(c.hist, old, nw)
}

func (c *calculator) Display() string {
	if c.err {
		return c.errMsg
	}
	if c.text == "" {
		return ""
	}
	return groupExpression(c.text, c.sep)
}

func (c *calculator) Expr() string {
	if c.err {
		return ""
	}
	if c.hist == "" {
		return ""
	}
	// canonical hist ("2+2=") grouped for display.
	h := strings.TrimSuffix(c.hist, "=")
	g := groupExpression(h, c.sep)
	if strings.HasSuffix(c.hist, "=") {
		g += "="
	}
	return g
}

func isBinOpRune(r rune) bool {
	return r == '+' || r == '−' || r == '×' || r == '÷'
}

func isDigitByte(b byte) bool { return b >= '0' && b <= '9' }

// parseCanonNum reads a canonical number (no grouping, with sep).
// It returns the value, whether it had a trailing %, and whether it had digits.
func parseCanonNum(tok string, sep byte) (v float64, pct bool, ok bool) {
	t := strings.ReplaceAll(tok, "−", "-") // pretty unary → ASCII
	if strings.HasSuffix(t, "%") {
		pct = true
		t = strings.TrimSuffix(t, "%")
	}
	var b strings.Builder
	for i := 0; i < len(t); i++ {
		ch := t[i]
		switch {
		case isDigitByte(ch):
			b.WriteByte(ch)
		case ch == sep:
			b.WriteByte('.')
		case ch == 'e' || ch == 'E' || ch == '+' || ch == '-':
			b.WriteByte(ch)
		case ch == ',' || ch == '.':
			// other surviving separator: ignore (defensive).
		}
	}
	s := b.String()
	if !hasDigit(s) {
		return 0, pct, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, pct, false
	}
	return v, pct, true
}

// fmt formats a number with the configured separator, without grouping
// (the canonical form kept by the engine).
func (c *calculator) fmt(v float64) string {
	return formatSep(v, c.sep)
}

// fmtg formats a number for the history display, with grouping.
func (c *calculator) fmtg(v float64) string {
	return groupDigits(c.fmt(v), c.sep)
}

// groupDigits groups thousands for display: "6306.04879" → "6,306.04879"
// (sep '.'), "6306,04879" → "6.306,04879" (sep ','). Exponent, fraction,
// sign and "" pass through untouched.
func groupDigits(s string, sep byte) string {
	if s == "" || s == "-" {
		return s
	}
	mant, exp, hasExp := s, "", false
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mant, exp, hasExp = s[:i], s[i+1:], true
	}
	dec := string([]byte{sep})
	grp := ","
	if sep == ',' {
		grp = "."
	}
	intpart, frac, hasFrac := mant, "", false
	if i := strings.Index(mant, dec); i >= 0 {
		intpart, frac, hasFrac = mant[:i], mant[i+len(dec):], true
	}
	neg := false
	digits := intpart
	if strings.HasPrefix(digits, "-") || strings.HasPrefix(digits, "−") {
		neg = true
		digits = strings.TrimPrefix(strings.TrimPrefix(digits, "-"), "−")
	}
	rd := []rune(digits)
	if len(rd) > 3 && hasDigit(digits) {
		var b strings.Builder
		rem := len(rd) % 3
		if rem > 0 {
			b.WriteString(string(rd[:rem]))
		}
		for i := rem; i < len(rd); i += 3 {
			if b.Len() > 0 {
				b.WriteString(grp)
			}
			b.WriteString(string(rd[i : i+3]))
		}
		digits = b.String()
	}
	out := digits
	if neg {
		out = "−" + out
	}
	if hasFrac {
		out += dec + frac
	}
	if hasExp {
		out += "E" + exp
	}
	return out
}

func formatSep(v float64, sep byte) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	if v == 0 {
		return "0"
	}
	abs := math.Abs(v)
	var s string
	if (abs >= 1e12 || abs < 1e-9) && v != 0 {
		s = strconv.FormatFloat(v, 'E', 6, 64)
	} else {
		s = strconv.FormatFloat(v, 'G', 12, 64)
	}
	// 'G' sometimes uses 'e+'; normalize to 'E'
	if strings.Contains(s, "e+") || strings.Contains(s, "e-") {
		s = strings.ReplaceAll(s, "e+", "E+")
		s = strings.ReplaceAll(s, "e-", "E-")
		s = strings.ReplaceAll(s, "e", "E")
	}
	if sep == ',' {
		s = strings.ReplaceAll(s, ".", ",")
	}
	if s == "-0" || s == "-0,0" || s == "-0.0" {
		return "0"
	}
	return s
}

func compute(a float64, op string, b float64) (float64, bool) {
	switch op {
	case "+":
		return a + b, true
	case "−", "-":
		return a - b, true
	case "×", "*", "x":
		return a * b, true
	case "÷", "/":
		if b == 0 {
			return 0, false
		}
		return a / b, true
	}
	return b, true
}

// etok is an expression token: a number (raw canonical form) or an operator.
type etok struct {
	isOp bool
	text string // number or operator (rune string)
}

// tokenizeExpr splits the expression into numbers and operators.
// It expects text already normalized by sanitize (collapsed runs): an
// optional initial negative number, then pairs (binary op, number with an
// optional unary '-'). It is defensive about unexpected leftovers (skips
// them). Numbers may carry an E exponent and a trailing %.
func tokenizeExpr(s string) []etok {
	var toks []etok
	rs := []rune(s)
	i, n := 0, len(rs)
	isOp := func(r rune) bool {
		return r == '+' || r == '−' || r == '×' || r == '÷'
	}
	// Start: drop [+×÷], a net '-' becomes unary.
	neg := false
	for i < n && isOp(rs[i]) {
		if rs[i] == '−' {
			neg = !neg
		}
		i++
	}
	toks = append(toks, etok{text: netSign(scanNumber(rs, &i), neg)})
	for i < n {
		if !isOp(rs[i]) {
			i++ // defensive: skip unexpected char
			continue
		}
		bop := rs[i]
		i++
		neg = false
		if i < n && rs[i] == '−' {
			neg = true // unary after a binary op ("2+-3")
			i++
		}
		toks = append(toks, etok{isOp: true, text: string(bop)})
		toks = append(toks, etok{text: netSign(scanNumber(rs, &i), neg)})
	}
	return toks
}

// scanNumber consumes [digits sep]* ([eE][+-]?digits)? [%]? from *i.
func scanNumber(rs []rune, i *int) string {
	n := len(rs)
	start := *i
	for *i < n && (isDigitByte(byte(rs[*i])) || rs[*i] == '.' || rs[*i] == ',') {
		*i++
	}
	if *i < n && (rs[*i] == 'e' || rs[*i] == 'E') {
		j := *i + 1
		if j < n && (rs[j] == '+' || rs[j] == '-' || rs[j] == '−') {
			j++
		}
		if j < n && isDigitByte(byte(rs[j])) {
			for j < n && isDigitByte(byte(rs[j])) {
				j++
			}
			*i = j
		}
	}
	if *i < n && rs[*i] == '%' {
		*i++
	}
	return string(rs[start:*i])
}

// netSign applies a unary '-' to a number ("" becomes "−", canonical pretty).
func netSign(num string, neg bool) string {
	if !neg {
		return num
	}
	if strings.HasPrefix(num, "-") {
		return strings.TrimPrefix(num, "-")
	}
	if strings.HasPrefix(num, "−") {
		return strings.TrimPrefix(num, "−")
	}
	return "−" + num
}

// groupExpression groups thousands per numeric token for display:
// "12+6306.04879" → "12+6,306.04879" (sep '.'). Operators pass through
// untouched; empty text/"-" pass through untouched.
func groupExpression(s string, sep byte) string {
	if s == "" {
		return ""
	}
	toks := tokenizeExpr(s)
	var b strings.Builder
	for _, t := range toks {
		if t.isOp {
			b.WriteString(t.text)
			continue
		}
		if t.text == "" {
			continue
		}
		pct := false
		num := t.text
		if strings.HasSuffix(num, "%") {
			pct = true
			num = strings.TrimSuffix(num, "%")
		}
		b.WriteString(groupDigits(num, sep))
		if pct {
			b.WriteString("%")
		}
	}
	return b.String()
}

// evalImmediate evaluates tokens with immediate semantics (Windows standard
// style): left to right, no precedence. % in an additive context becomes a
// fraction of the accumulator (200+10% → +20); otherwise it becomes /100.
// A trailing operator without an operand repeats the accumulator
// ("2+" → 2+2). It returns the result, the last (op, effective operand) to
// repeat "=", and an error message ("" if ok). With no numbers → ok=false.
func evalImmediate(toks []etok, sep byte) (acc float64, lastOp string, lastB float64, hasOp bool, errMsg string) {
	// Flatten: first number, then (op, number) pairs.
	if len(toks) == 0 {
		return 0, "", 0, false, ""
	}
	nums := []string{toks[0].text}
	var ops []rune
	for i := 1; i < len(toks); i++ {
		if toks[i].isOp {
			// defensive ASCII normalization
			op := []rune(toks[i].text)[0]
			if op == '-' {
				op = '−'
			}
			ops = append(ops, op)
		} else {
			nums = append(nums, toks[i].text)
		}
	}
	if len(nums) == 0 || (len(nums) == 1 && !hasDigit(nums[0])) {
		return 0, "", 0, false, ""
	}
	v0, pct0, ok0 := parseCanonNum(nums[0], sep)
	if !ok0 {
		v0 = 0
	}
	acc = v0
	if pct0 {
		acc = v0 / 100 // no left context: % = /100
	}
	for k, op := range ops {
		var b float64
		if k+1 < len(nums) && hasDigit(nums[k+1]) {
			v, pct, ok := parseCanonNum(nums[k+1], sep)
			if !ok {
				v = 0
			}
			b = v
			if pct {
				if op == '+' || op == '−' {
					b = acc * v / 100
				} else {
					b = v / 100
				}
			}
		} else {
			b = acc // trailing operator without operand: repeat the accumulator
		}
		var ok bool
		acc, ok = compute(acc, string(op), b)
		if !ok {
			return 0, "", 0, true, "Cannot divide by zero"
		}
		lastOp, lastB, hasOp = string(op), b, true
	}
	return acc, lastOp, lastB, hasOp, ""
}

func (c *calculator) setErr(msg string) {
	c.err = true
	c.errMsg = msg
	c.hist = ""
}

func (c *calculator) clearErr() {
	c.err = false
	c.errMsg = ""
}

// normalizeGrouping resolves grouping separators in pasted text:
// with both "." and ",", the LAST one is the decimal separator and the
// others are grouping ("6,306.04879" → "6306.04879"); the same separator
// twice or more is grouping ("1.000.000" → "1000000"). A single separator
// is decimal and does not change. The exponent ("1.000E+3") is preserved.
func normalizeGrouping(s string) string {
	mant, exp, hasExp := s, "", false
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mant, exp, hasExp = s[:i], s[i+1:], true
	}
	dots := strings.Count(mant, ".")
	commas := strings.Count(mant, ",")
	switch {
	case dots+commas <= 1:
		// Nothing or a single separator: already in decimal form.
	case dots > 0 && commas > 0:
		lastDot := strings.LastIndex(mant, ".")
		lastComma := strings.LastIndex(mant, ",")
		last := max(lastDot, lastComma)
		var b strings.Builder
		for i, r := range mant {
			if (r == '.' || r == ',') && i != last {
				continue
			}
			b.WriteRune(r)
		}
		mant = b.String()
	default:
		// Same separator repeated: grouping, remove all.
		mant = strings.ReplaceAll(mant, ".", "")
		mant = strings.ReplaceAll(mant, ",", "")
	}
	if !hasExp {
		return mant
	}
	return mant + "E" + exp
}

// isPasteInsert reports whether the edit inserted 2+ characters at once (a
// paste) instead of a single one (typing). Only pasting goes through
// normalizeGrouping: while typing, a second separator is ignored instead of
// reinterpreting the number ("12.5" + "," stays "12.5").
func isPasteInsert(num, oldShown string) bool {
	return len(num) > len(oldShown)+1
}

// isOpRune reports whether the rune is an operator typeable in the display.
func isOpRune(r rune) bool {
	switch r {
	case '+', '-', '*', '/', '%', '=', 'x', 'X', '÷', '×', '−':
		return true
	}
	return false
}

// xpart is a loose piece of expression: a raw number or a run of operators.
type xpart struct {
	isOp bool
	text string
}

// splitLoose separates numbers (digits/sep/eE/% stuck together, with
// E[+-]?digits) and runs of operators, preserving order. Used by sanitize.
func splitLoose(s string) []xpart {
	var parts []xpart
	rs := []rune(s)
	i, n := 0, len(rs)
	isOp := func(r rune) bool {
		return r == '+' || r == '−' || r == '×' || r == '÷'
	}
	isNum := func(r rune) bool {
		return (r >= '0' && r <= '9') || r == '.' || r == ',' ||
			r == 'e' || r == 'E' || r == '%'
	}
	for i < n {
		if isOp(rs[i]) {
			j := i
			for j < n && isOp(rs[j]) {
				j++
			}
			parts = append(parts, xpart{isOp: true, text: string(rs[i:j])})
			i = j
			continue
		}
		if isNum(rs[i]) {
			j := i
			for j < n {
				r := rs[j]
				if (r >= '0' && r <= '9') || r == '.' || r == ',' || r == '%' {
					j++
					continue
				}
				if r == 'e' || r == 'E' {
					k := j + 1
					if k < n && (rs[k] == '+' || rs[k] == '-' || rs[k] == '−') {
						k++
					}
					if k < n && rs[k] >= '0' && rs[k] <= '9' {
						for k < n && rs[k] >= '0' && rs[k] <= '9' {
							k++
						}
						j = k
						continue
					}
					j++ // stray 'e': include it (the filter cleans it later)
					continue
				}
				break
			}
			parts = append(parts, xpart{text: string(rs[i:j])})
			i = j
			continue
		}
		i++ // defensive: should not happen after the charset filter
	}
	return parts
}

// collapseOpRun reduces a run of operators in the middle/end: with ×÷ the
// last one wins; with only +- : "+", "−", "+−" pass, the rest becomes the
// last char ("++"→"+").
func collapseOpRun(run string) string {
	rs := []rune(run)
	last := rs[len(rs)-1]
	if last == '×' || last == '÷' {
		return string(last)
	}
	if last == '+' {
		return "+"
	}
	// last == '−': keep it if it is unary after ×÷/+ ("×−", "+−"), else swap.
	if len(rs) == 1 {
		return "−"
	}
	prev := rs[len(rs)-2]
	if prev == '×' || prev == '÷' || prev == '+' {
		return string([]rune{prev, '−'})
	}
	return "−"
}

// collapseLeadingRun: the start of an expression only accepts a net unary '-'.
func collapseLeadingRun(run string) string {
	n := 0
	for _, r := range run {
		if r == '−' {
			n++
		}
	}
	if n%2 == 1 {
		return "−"
	}
	return ""
}

const maxExprRunes = 200

// sanitizeExpr normalizes typed/pasted text into a canonical expression
// (no grouping, with sep): charset, collapse of operator runs, decimals
// and grouping per numeric token. oldShown (the previous shown text, with
// grouping) aligns tokens to tell typing (contextual filtering) from
// pasting (normalize+filter). Above the limit, it keeps the current text.
func (c *calculator) sanitizeExpr(raw, oldShown string) string {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == ',':
			b.WriteRune(r)
		case r == '+':
			b.WriteRune(r)
		case r == '-':
			b.WriteRune('−')
		case r == '*' || r == 'x' || r == 'X':
			b.WriteRune('×')
		case r == '/':
			b.WriteRune('÷')
		case r == '−' || r == '×' || r == '÷' || r == '%':
			b.WriteRune(r)
		case r == 'e' || r == 'E':
			b.WriteRune(r)
		}
	}
	parts := splitLoose(b.String())
	var oldNums []string
	for _, p := range splitLoose(oldShown) {
		if !p.isOp {
			oldNums = append(oldNums, p.text)
		}
	}
	var out []xpart
	ni := 0
	for _, p := range parts {
		if p.isOp {
			out = append(out, p)
			continue
		}
		old := ""
		if ni < len(oldNums) {
			old = oldNums[ni]
		}
		ni++
		// % stuck to the number: split it to filter the number, and reattach
		// it when valid.
		core, pct := p.text, false
		if strings.HasSuffix(core, "%") {
			core = strings.TrimRight(core, "%")
			pct = true
		}
		var clean string
		if isPasteInsert(core, old) {
			clean = c.filterEntry(normalizeGrouping(core))
		} else {
			clean = c.filterTyped(core, old)
		}
		if clean == "" {
			continue // an empty number disappears (a stray % goes with it)
		}
		if pct && hasDigit(clean) {
			clean += "%"
		}
		out = append(out, xpart{text: clean})
	}
	var rb strings.Builder
	emitted := false
	for _, p := range out {
		if !p.isOp {
			rb.WriteString(p.text)
			emitted = true
			continue
		}
		var run string
		if !emitted {
			run = collapseLeadingRun(p.text)
		} else {
			run = collapseOpRun(p.text)
		}
		if run == "" {
			continue
		}
		rb.WriteString(run)
		emitted = true
	}
	res := rb.String()
	if len([]rune(res)) > maxExprRunes {
		return c.text
	}
	return res
}

// splitTrailingNum separates the final number (digits/sep/E, with a unary
// '-' and an optional '%') from the rest. With no number → num "". Used by
// CE, +/-, 1/x, x², √ (which ignore a trailing %).
func splitTrailingNum(s string) (head, num string, hasPct bool) {
	rs := []rune(s)
	i := len(rs)
	if i > 0 && rs[i-1] == '%' {
		hasPct = true
		i--
	}
	for i > 0 && (isDigitByte(byte(rs[i-1])) || rs[i-1] == '.' || rs[i-1] == ',') {
		i--
	}
	// exponent "E[+-]?": the sign above may belong to it.
	if i > 0 && (rs[i-1] == '+' || rs[i-1] == '-' || rs[i-1] == '−') && i >= 2 &&
		(rs[i-2] == 'e' || rs[i-2] == 'E') && i >= 3 &&
		(isDigitByte(byte(rs[i-3])) || rs[i-3] == '.' || rs[i-3] == ',') {
		i -= 2
		for i > 0 && (isDigitByte(byte(rs[i-1])) || rs[i-1] == '.' || rs[i-1] == ',') {
			i--
		}
	}
	// unary '-': preceded by an operator or at the start.
	if i > 0 && rs[i-1] == '−' && (i-1 == 0 || isBinOpRune(rs[i-2])) {
		i--
	}
	end := len(rs)
	if hasPct {
		end--
	}
	num = string(rs[i:end])
	if !hasDigit(num) {
		return s, "", false
	}
	return string(rs[:i]), num, hasPct
}

// splitTrailingOp separates a trailing operator ("12+" → "+", "12").
// It lets the app treat what was typed/pasted into the display apart from
// the keyboard layout (ABNT2, US, numeric).
func splitTrailingOp(raw string) (op string, num string) {
	if raw == "" {
		return "", ""
	}
	r, size := utf8.DecodeLastRuneInString(raw)
	if isOpRune(r) {
		return string(r), raw[:len(raw)-size]
	}
	return "", raw
}

func hasDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// finishMantissa finishes a filtered mantissa: "" and a pending "-" pass,
// leading zeros collapse ("007"→"7") and a lone separator becomes "0.".
func finishMantissa(out string, D byte) string {
	if out == "" {
		return ""
	}
	if out == "-" {
		return "-"
	}
	sep := string([]byte{D})
	neg := strings.HasPrefix(out, "-")
	core := strings.TrimPrefix(out, "-")
	if i := strings.Index(core, sep); i < 0 {
		core = strings.TrimLeft(core, "0")
		if core == "" {
			core = "0"
		}
	} else {
		intpart, frac := core[:i], core[i+len(sep):]
		intpart = strings.TrimLeft(intpart, "0")
		if intpart == "" {
			intpart = "0"
		}
		core = intpart + sep + frac
	}
	if neg && core != "0" && core != "0"+sep {
		out = "-" + core
	} else {
		out = core
	}
	if out == sep {
		out = "0" + sep
	}
	return out
}

// filterTyped sanitizes typing/erasing a single character over the shown
// text (which has grouping). oldShown gives the context: separators already
// in it are grouping and drop out; a separator typed with no decimal
// present becomes the decimal one ("12" + "," → "12.").
func (c *calculator) filterTyped(num, oldShown string) string {
	D := c.sep
	G := byte(',')
	if c.sep == ',' {
		G = '.'
	}
	prevG := strings.Count(oldShown, string([]byte{G}))
	newG := strings.Count(num, string([]byte{G}))
	hasD := strings.IndexByte(num, D) >= 0
	// Keep one G as the decimal only if there is no decimal and some G is new.
	keepG := -1
	if !hasD && newG > prevG {
		keepG = strings.LastIndex(num, string([]byte{G}))
	}
	var b strings.Builder
	digits := 0
	gotDec := false
	for i := 0; i < len(num); i++ {
		r := num[i]
		switch {
		case r >= '0' && r <= '9':
			if digits >= 32 {
				continue
			}
			digits++
			b.WriteByte(r)
		case r == D:
			if !gotDec {
				gotDec = true
				b.WriteByte(D)
			}
		case r == G:
			if i == keepG && !gotDec {
				gotDec = true
				b.WriteByte(D)
			}
		case r == '-' && i == 0:
			b.WriteByte(r)
		}
	}
	return finishMantissa(b.String(), D)
}

// filterEntry sanitizes text typed or pasted into the display into a valid
// number with the configured separator: only digits, one separator ("." and
// "," both become the configured one), a leading "-" and exponent notation
// ("1.5E+3"). "" stays "" (the display starts empty).
func (c *calculator) filterEntry(s string) string {
	sep := rune(c.sep)
	mant := s
	exp := ""
	hasExp := false
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mant, exp, hasExp = s[:i], s[i+1:], true
	}
	var b strings.Builder
	digits := 0
	gotSep := false
	for i, r := range mant {
		switch {
		case r >= '0' && r <= '9':
			if digits >= 32 {
				continue
			}
			digits++
			b.WriteRune(r)
		case r == '.' || r == ',':
			if !gotSep {
				gotSep = true
				b.WriteRune(sep)
			}
		case r == '-' && i == 0:
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = finishMantissa(out, byte(sep))
	if !hasExp {
		return out
	}
	var e strings.Builder
	edigits := 0
	for i, r := range exp {
		switch {
		case r >= '0' && r <= '9':
			if edigits >= 4 {
				continue
			}
			edigits++
			e.WriteRune(r)
		case (r == '+' || r == '-' || r == '−') && i == 0:
			e.WriteRune(r)
		}
	}
	es := e.String()
	if !hasDigit(es) {
		return out
	}
	return out + "E" + es
}

// editedFresh starts a fresh edit over a result: clears history/repeat.
func (c *calculator) editedFresh() {
	c.hist = ""
	c.evaluated = false
	c.lastOp = ""
}

// Backspace erases the last character (⌫).
func (c *calculator) Backspace() {
	if c.err || c.text == "" {
		return
	}
	if c.evaluated {
		c.editedFresh()
	}
	rs := []rune(c.text)
	c.text = string(rs[:len(rs)-1])
}

// ClearEntry erases the final number (keeping the operator). After "=" it
// clears the display.
func (c *calculator) ClearEntry() {
	if c.err {
		c.clearErr()
	}
	if c.evaluated {
		c.text = ""
		c.evaluated = false
		return // hist stays (as in Windows)
	}
	head, num, _ := splitTrailingNum(c.text)
	if num == "" {
		return
	}
	c.text = head
}

// ClearAll clears everything (C).
func (c *calculator) ClearAll() {
	c.clearErr()
	c.text = ""
	c.hist = ""
	c.evaluated = false
	c.lastOp = ""
	c.lastVal = 0
}

// ToggleSign flips the sign of the final number (+/-).
func (c *calculator) ToggleSign() {
	if c.err || c.text == "" {
		return
	}
	if c.evaluated {
		c.editedFresh()
	}
	head, num, hasPct := splitTrailingNum(c.text)
	if num == "" || hasPct {
		return
	}
	bare := strings.TrimPrefix(strings.TrimPrefix(num, "-"), "−")
	if bare == "0" || bare == "0." || bare == "0," {
		return
	}
	if strings.HasPrefix(num, "-") {
		c.text = head + strings.TrimPrefix(num, "-")
	} else if strings.HasPrefix(num, "−") {
		c.text = head + strings.TrimPrefix(num, "−")
	} else {
		c.text = head + "−" + num
	}
}

// transformTrailing applies fn to the final number (for 1/x, x², √).
func (c *calculator) transformTrailing(fn func(float64) (float64, string)) {
	if c.err {
		return
	}
	if c.evaluated {
		c.editedFresh()
	}
	head, num, hasPct := splitTrailingNum(c.text)
	if num == "" || hasPct || !hasDigit(num) {
		return
	}
	v, _, ok := parseCanonNum(num, c.sep)
	if !ok {
		return
	}
	r, errMsg := fn(v)
	if errMsg != "" {
		c.setErr(errMsg)
		return
	}
	c.text = head + c.fmt(r)
}

// Inverse computes 1/(final number).
func (c *calculator) Inverse() {
	c.transformTrailing(func(v float64) (float64, string) {
		if v == 0 {
			return 0, "Cannot divide by zero"
		}
		return 1 / v, ""
	})
}

// Square raises the final number to the square.
func (c *calculator) Square() {
	c.transformTrailing(func(v float64) (float64, string) {
		return v * v, ""
	})
}

// Sqrt computes the root of the final number.
func (c *calculator) Sqrt() {
	c.transformTrailing(func(v float64) (float64, string) {
		if v < 0 {
			return 0, "Invalid input"
		}
		return math.Sqrt(v), ""
	})
}

// Equals evaluates the expression (Enter/=) or repeats the last operation.
func (c *calculator) Equals() {
	if c.err {
		return
	}
	if c.evaluated {
		if c.lastOp == "" {
			return
		}
		a, _, ok := parseCanonNum(c.text, c.sep)
		if !ok {
			return
		}
		r, ok2 := compute(a, c.lastOp, c.lastVal)
		if !ok2 {
			c.setErr("Cannot divide by zero")
			return
		}
		c.hist = fmt.Sprintf("%s %s %s =",
			groupExpression(c.text, c.sep), c.lastOp, c.fmtg(c.lastVal))
		c.text = c.fmt(r)
		return
	}
	if !hasDigit(c.text) {
		return // empty → nothing
	}
	toks := tokenizeExpr(c.text)
	acc, lastOp, lastB, hasOp, errMsg := evalImmediate(toks, c.sep)
	if errMsg != "" {
		c.setErr(errMsg)
		return
	}
	shown := groupExpression(c.text, c.sep)
	if shown == "" {
		return
	}
	c.hist = shown + "="
	c.text = c.fmt(acc)
	c.evaluated = true
	c.lastOp, c.lastVal = lastOp, lastB
	_ = hasOp
}

// UserEdit applies to the engine the display text changed by the user
// (typing, pasting, erasing). The expression is built literally ("2+2"
// appears where it is typed); it only evaluates on a trailing '=', Enter or
// the = button. It works with any keyboard layout because it reacts to the
// character.
func (c *calculator) UserEdit(raw, oldShown string) {
	if c.err {
		c.ClearAll()
	}
	doEval := false
	if r, _ := utf8.DecodeLastRuneInString(raw); r == '=' {
		doEval = true
		raw = strings.TrimRight(raw, "=")
	}
	if doEval && c.evaluated {
		c.Equals() // repeat (or nothing, without lastOp)
		return
	}
	entry := c.sanitizeExpr(raw, oldShown)
	if c.evaluated {
		if entry == c.text {
			return // no change: keep hist/repeat
		}
		if c.text != "" && strings.HasPrefix(entry, c.text) && len(entry) > len(c.text) {
			rest := strings.TrimPrefix(entry, c.text)
			if fr, _ := utf8.DecodeRuneInString(rest); fr == '%' || isBinOpRune(fr) {
				// continue from the result ("4+")
				c.text = entry
			} else {
				// start over ("45" → "5")
				c.text = rest
				c.lastOp = ""
			}
		} else {
			c.text = entry
			c.lastOp = ""
		}
		c.hist = ""
		c.evaluated = false
		return
	}
	c.text = entry
	if doEval {
		c.Equals()
	}
}
