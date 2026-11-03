package display

import (
	"fmt"
	"image"
	"math"
	"strings"
	"time"
)

// Fixed layout, in landscape pixels. Every element sits on the integer grid;
// nothing is anti-aliased or dithered.
const (
	headerBottom = 26 // header band rows 0–26, black
	footerTop    = 111
	sideMargin   = 6 // text inset in the header and footer bands

	heroLabelY    = 5
	heroPriceSize = 20.0
	heroBaseline  = 22
	heroGap       = 6 // between the hero label and its price

	auxLabelY = 5
	auxPriceY = 15

	tileX0      = 3
	tileY0      = 30
	tileW       = 120 // outer edge spans x0 .. x0+tileW
	tileH       = 32
	tileStrideX = 124
	tileStrideY = 35
	tileRadius  = 4
	tileLabelDX = 6
	tileLabelDY = 5
	tilePriceDX = 5
	tilePriceDY = 27 // baseline
	tilePriceSz = 17.0

	pillH       = 12
	pillRadius  = 3
	pillDY      = 2  // below the tile's top edge
	pillMarginR = 5  // from the tile's right edge
	pillPad     = 15 // outline + padding + triangle + gap, beyond the text

	ratioY      = 101
	ratioMargin = 4 // left and right of the strip
	ratioGap    = 3 // between a ratio's label and its value

	footerTextY = 113
	sumMaxRunes = 10
)

// Render draws the dashboard for the given configuration, inputs and
// instant. It is deterministic for a given (cfg, in, now) and never fails on
// bad data; the only error is a font that will not load, which is a build
// problem, not a data one.
func Render(cfg Config, in Inputs, now time.Time) (*Frame, error) {
	f := NewFrame()
	loc, err := time.LoadLocation(cfg.Clock.TZ)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)

	if err := drawHeader(f, cfg, in, local); err != nil {
		return nil, err
	}
	if err := drawTiles(f, cfg, in); err != nil {
		return nil, err
	}
	drawRatios(f, cfg, in)
	drawFooter(f, cfg, in, local)
	return f, nil
}

// ── Header ──────────────────────────────────────────────────────────────────

func drawHeader(f *Frame, cfg Config, in Inputs, local time.Time) error {
	f.FillRect(0, 0, Width-1, headerBottom, true)
	clip := image.Rect(0, 0, Width, headerBottom+1)

	if h := cfg.Hero; h != nil {
		x := sideMargin
		if label := slotLabel(h); label != "" {
			x = f.DrawPixelText(x, heroLabelY, label, false, false) - 1 + heroGap
		}
		text := slotPrice(h, in.Quotes, defaultHeroDecimals, true)
		if _, err := f.DrawOutlineText(heroPriceSize, x, heroBaseline, text, false, clip); err != nil {
			return err
		}
	}

	if a := cfg.Aux; a != nil {
		slot := *a
		if slot.Key == "" {
			if local.Hour() < 12 {
				slot.Key = slot.Morning
			} else {
				slot.Key = slot.Evening
			}
		}
		label := slotLabel(&slot)
		right := Width - 1 - sideMargin
		f.DrawPixelText(right-PixelTextWidth(label, false)+1, auxLabelY, label, false, false)
		price := slotPrice(&slot, in.Quotes, defaultTileDecimals, false)
		f.DrawPixelText(right-PixelTextWidth(price, true)+1, auxPriceY, price, false, true)
	}
	return nil
}

// slotLabel is what a slot prints as its name: the label, else the key.
func slotLabel(s *Slot) string {
	if s.Label != "" {
		return s.Label
	}
	return s.Key
}

// slotPrice formats a slot's number, or N/A when its symbol is not quoted.
func slotPrice(s *Slot, q Quotes, defaultDecimals int, thousands bool) string {
	src, ok := q.Price(s.Key)
	if !ok || s.Key == "" {
		return NA
	}
	d := s.Decimals
	if d == nil {
		d = intp(defaultDecimals)
	}
	return FormatPrice(src, d, s.Prefix, thousands)
}

// ── Tiles ───────────────────────────────────────────────────────────────────

func drawTiles(f *Frame, cfg Config, in Inputs) error {
	for i, slot := range cfg.Tiles() {
		x0 := tileX0 + (i%2)*tileStrideX
		y0 := tileY0 + (i/2)*tileStrideY
		x1, y1 := x0+tileW, y0+tileH
		f.RoundRect(x0, y0, x1, y1, tileRadius, true, false)
		if slot == nil {
			continue
		}
		clip := image.Rect(x0+1, y0+1, x1, y1)
		f.DrawPixelText(x0+tileLabelDX, y0+tileLabelDY, slotLabel(slot), true, false)
		price := slotPrice(slot, in.Quotes, defaultTileDecimals, false)
		if _, err := f.DrawOutlineText(tilePriceSz, x0+tilePriceDX, y0+tilePriceDY, price, true, clip); err != nil {
			return err
		}
		if delta, ok := Change(slot.Key, in.Quotes, in.Prev); ok {
			drawPill(f, x1-pillMarginR, y0+pillDY, delta, cfg.Pills.ThresholdPct)
		}
	}
	return nil
}

// Change returns the percentage move of symbol between prev and cur, and
// false when either side is missing or not a number, or prev is zero.
func Change(symbol string, cur, prev Quotes) (pct float64, ok bool) {
	cs, ok1 := cur.Price(symbol)
	ps, ok2 := prev.Price(symbol)
	if !ok1 || !ok2 {
		return 0, false
	}
	c, _, okc := parsePrice(cs)
	p, _, okp := parsePrice(ps)
	if !okc || !okp || p == 0 {
		return 0, false
	}
	return (c - p) / p * 100, true
}

// PillKind is what a change pill looks like for a given move.
type PillKind int

const (
	PillNone PillKind = iota // |Δ| under the threshold: no pill
	PillUp                   // filled, white on black, ▲
	PillDown                 // outlined, ▼
)

// ClassifyChange maps a percentage move to a pill kind. Direction is carried
// by the fill, not the glyph alone, so it reads from across a room.
func ClassifyChange(pct, thresholdPct float64) PillKind {
	if thresholdPct <= 0 {
		thresholdPct = math.SmallestNonzeroFloat64 // any move at all shows a pill
	}
	switch {
	case pct >= thresholdPct:
		return PillUp
	case pct <= -thresholdPct:
		return PillDown
	}
	return PillNone
}

// drawPill draws a change pill whose right edge is at right and top at top.
func drawPill(f *Frame, right, top int, pct, threshold float64) {
	kind := ClassifyChange(pct, threshold)
	if kind == PillNone {
		return
	}
	text := fmt.Sprintf("%.1f%%", math.Abs(pct))
	w := PixelTextWidth(text, false) + pillPad
	x0, y0 := right-w+1, top
	x1, y1 := right, top+pillH-1
	up := kind == PillUp
	f.RoundRect(x0, y0, x1, y1, pillRadius, true, up)
	ink := !up // white on a filled pill, black on an outlined one
	// 5×3 triangle, 1 px in from the padding, then a 2-px gap to the text.
	tx, ty := x0+4, y0+5
	if up {
		f.HLine(tx+2, tx+2, ty, ink)
		f.HLine(tx+1, tx+3, ty+1, ink)
		f.HLine(tx, tx+4, ty+2, ink)
	} else {
		f.HLine(tx, tx+4, ty, ink)
		f.HLine(tx+1, tx+3, ty+1, ink)
		f.HLine(tx+2, tx+2, ty+2, ink)
	}
	f.DrawPixelText(x0+11, y0+3, text, ink, false)
}

// ── Ratio strip ─────────────────────────────────────────────────────────────

type ratioGroup struct {
	label, value string
}

// ratioGroups resolves the three ratio positions to label/value pairs,
// skipping positions with no configuration.
func ratioGroups(cfg Config, q Quotes) []ratioGroup {
	var out []ratioGroup
	for _, r := range cfg.Ratios() {
		if r == nil {
			continue
		}
		label := r.Label
		if label == "" {
			label = symbolLabel(cfg, r.Num) + "/" + symbolLabel(cfg, r.Den)
		}
		d := defaultRatioDecimals
		if r.Decimals != nil {
			d = *r.Decimals
		}
		value := NA
		ns, okn := q.Price(r.Num)
		ds, okd := q.Price(r.Den)
		if okn && okd {
			n, _, okn := parsePrice(ns)
			dv, _, okd := parsePrice(ds)
			if okn && okd && dv != 0 {
				value = FormatComputed(n/dv, d)
			}
		}
		out = append(out, ratioGroup{label: label, value: value})
	}
	return out
}

// symbolLabel is the label of the tile that shows symbol, else the symbol.
func symbolLabel(cfg Config, symbol string) string {
	for _, t := range cfg.Tiles() {
		if t != nil && t.Key == symbol && t.Label != "" {
			return t.Label
		}
	}
	return symbol
}

func drawRatios(f *Frame, cfg Config, in Inputs) {
	groups := ratioGroups(cfg, in.Quotes)
	if len(groups) == 0 {
		return
	}
	total := 0
	for _, g := range groups {
		total += PixelTextWidth(g.label, false) + ratioGap + PixelTextWidth(g.value, true)
	}
	gap := 0
	if len(groups) > 1 {
		gap = (Width - 2*ratioMargin - total) / (len(groups) - 1)
		if gap < 2 {
			gap = 2
		}
	}
	x := ratioMargin
	for _, g := range groups {
		x = f.DrawPixelText(x, ratioY, g.label, true, false) - 1 + ratioGap
		x = f.DrawPixelText(x, ratioY, g.value, true, true) - 1 + gap
	}
}

// ── Footer ──────────────────────────────────────────────────────────────────

func drawFooter(f *Frame, cfg Config, in Inputs, local time.Time) {
	f.FillRect(0, footerTop, Width-1, Height-1, true)

	// Left: the sum (or the config warning), bare text.
	sum := ""
	switch {
	case in.ConfigInvalid:
		sum = "Cfg?"
	case cfg.Sum != nil:
		sum = firstLine(in.Sum, sumMaxRunes)
	}
	spanStart := sideMargin
	if sum != "" {
		spanStart = f.DrawPixelText(sideMargin, footerTextY, sum, false, false) - 1
	}

	// Right: the wake counter.
	wake := "—"
	if in.HasWake {
		wake = fmt.Sprintf("%d", in.Wake)
	}
	if cfg.Wake.Label != "" {
		wake = cfg.Wake.Label + " " + wake
	}
	spanEnd := Width - 1 - sideMargin - PixelTextWidth(wake, false) + 1
	f.DrawPixelText(spanEnd, footerTextY, wake, false, false)

	// Centre: the clock, or the stale marker in its place.
	clock := local.Format(cfg.Clock.Format)
	if stale, label := Staleness(cfg, in.Quotes, local); stale {
		clock = label
	}
	clock = strings.TrimSpace(clock)
	x := (spanStart + spanEnd - PixelTextWidth(clock, false)) / 2
	f.DrawPixelText(x, footerTextY, clock, false, false)
}
