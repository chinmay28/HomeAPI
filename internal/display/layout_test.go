package display

import (
	"flag"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden frame in testdata/")

// The fixture: the reference rendering's numbers at the reference instant.
var (
	fixtureLoc, _ = time.LoadLocation("America/Los_Angeles")
	fixtureNow    = time.Date(2026, 11, 3, 7, 15, 0, 0, fixtureLoc) // a Tuesday
	fixtureQuotes = `{"BTC-USD":"109432.12","IBIT":"61.30","STRC":"101.5","VTI":"301.20","GLD":"214.05","P":"58.41","ORCL":"190.77","timestamp":"11/03 07:10:00"}`
	fixturePrev   = `{"BTC-USD":"109000","VTI":"299.40","GLD":"216.65","P":"57.04","ORCL":"191.34","timestamp":"11/02 19:10:00"}`
)

func fixtureInputs() Inputs {
	return Inputs{
		Quotes:  ParseQuotes(fixtureQuotes, fixtureNow, fixtureLoc),
		Prev:    ParseQuotes(fixturePrev, fixtureNow, fixtureLoc),
		Sum:     "42",
		Wake:    183,
		HasWake: true,
	}
}

func mustRender(t *testing.T, cfg Config, in Inputs, now time.Time) *Frame {
	t.Helper()
	f, err := Render(cfg, in, now)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return f
}

// inkCount counts pixels of the given colour inside rect (exclusive max).
func inkCount(f *Frame, r image.Rectangle, black bool) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if f.Get(x, y) == black {
				n++
			}
		}
	}
	return n
}

// tileRect is the outer bounds of tile i (0–3), exclusive max.
func tileRect(i int) image.Rectangle {
	x0 := tileX0 + (i%2)*tileStrideX
	y0 := tileY0 + (i/2)*tileStrideY
	return image.Rect(x0, y0, x0+tileW+1, y0+tileH+1)
}

// priceBand is the part of tile i where its price is drawn, clear of the
// label, the pill and the rounded corners.
func priceBand(i int) image.Rectangle {
	r := tileRect(i)
	return image.Rect(r.Min.X+2, r.Min.Y+14, r.Max.X-1, r.Max.Y-5)
}

// pillArea is where tile i's change pill would be, clear of the outline.
func pillArea(i int) image.Rectangle {
	r := tileRect(i)
	return image.Rect(r.Min.X+40, r.Min.Y+2, r.Max.X-6, r.Min.Y+pillDY+pillH)
}

var (
	headerRect = image.Rect(0, 0, Width, headerBottom+1)
	footerRect = image.Rect(0, footerTop, Width, Height)
	stripRect  = image.Rect(0, 98, Width, footerTop)
)

// pixelTextAt reports whether text, in the pixel font and the given colour,
// sits exactly at (x, y) — every glyph pixel on and every background pixel of
// the glyph cells off.
func pixelTextAt(f *Frame, x, y int, text string, black, bold bool) bool {
	want := NewFrame()
	if !black {
		want.FillRect(0, 0, Width-1, Height-1, true)
	}
	want.DrawPixelText(x, y, text, black, bold)
	w := PixelTextWidth(text, bold)
	for yy := y; yy < y+pixelGlyphH; yy++ {
		for xx := x; xx < x+w; xx++ {
			if f.Get(xx, yy) != want.Get(xx, yy) {
				return false
			}
		}
	}
	return true
}

// findPixelText scans row y for text and returns its x, or -1.
func findPixelText(f *Frame, y int, text string, black, bold bool) int {
	w := PixelTextWidth(text, bold)
	for x := 0; x+w <= Width; x++ {
		if pixelTextAt(f, x, y, text, black, bold) {
			return x
		}
	}
	return -1
}

// expectOutline renders text alone at the given pen position and checks that
// rect in f carries exactly that ink (and nothing else).
func expectOutline(t *testing.T, f *Frame, rect image.Rectangle, size float64, x, baseline int, text string, black bool) {
	t.Helper()
	want := NewFrame()
	if !black {
		want.FillRect(0, 0, Width-1, Height-1, true)
	}
	if _, err := want.DrawOutlineText(size, x, baseline, text, black, rect); err != nil {
		t.Fatal(err)
	}
	for yy := rect.Min.Y; yy < rect.Max.Y; yy++ {
		for xx := rect.Min.X; xx < rect.Max.X; xx++ {
			if f.Get(xx, yy) != want.Get(xx, yy) {
				t.Fatalf("outline text %q: pixel (%d,%d) differs", text, xx, yy)
			}
		}
	}
	if inkCount(want, rect, black) == 0 {
		t.Fatalf("outline text %q drew nothing in %v", text, rect)
	}
}

func TestRender_Golden(t *testing.T) {
	f := mustRender(t, DefaultConfig(), fixtureInputs(), fixtureNow)
	path := filepath.Join("testdata", "golden.pbm")
	if *update {
		if err := os.WriteFile(path, f.PBM(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if got := f.PBM(); string(got) != string(want) {
		golden, perr := ParsePBM(want)
		if perr != nil {
			t.Fatalf("golden file unreadable: %v", perr)
		}
		t.Errorf("frame differs from golden in %d pixels; inspect with -update and a visual diff", f.Diff(golden))
	}
}

func TestRender_Deterministic(t *testing.T) {
	a := mustRender(t, DefaultConfig(), fixtureInputs(), fixtureNow)
	b := mustRender(t, DefaultConfig(), fixtureInputs(), fixtureNow)
	if n := a.Diff(b); n != 0 {
		t.Errorf("two renders of the same inputs differ in %d pixels", n)
	}
	if len(a.Native()) != NativeSize {
		t.Errorf("native body is %d bytes", len(a.Native()))
	}
}

func TestRender_MatchesReferenceLayout(t *testing.T) {
	// The fixed furniture of the design, as measured on the reference
	// rendering (docs/minion3/minion3_final_1x.png).
	f := mustRender(t, DefaultConfig(), fixtureInputs(), fixtureNow)

	if x := findPixelText(f, 5, "BTC", false, false); x != 6 {
		t.Errorf("hero label at x=%d, want 6", x)
	}
	if x := findPixelText(f, 5, "IBIT", false, false); x != 221 {
		t.Errorf("aux symbol at x=%d, want 221 (right-aligned to 243)", x)
	}
	if x := findPixelText(f, 15, "$61.30", false, true); x != 203 {
		t.Errorf("aux price at x=%d, want 203 (right-aligned to 243)", x)
	}
	for i, label := range []string{"VTI", "GLD", "PSTG", "ORCL"} {
		r := tileRect(i)
		if !pixelTextAt(f, r.Min.X+tileLabelDX, r.Min.Y+tileLabelDY, label, true, false) {
			t.Errorf("tile %d label %q not at (%d,%d)", i+1, label, r.Min.X+tileLabelDX, r.Min.Y+tileLabelDY)
		}
		// Outline and corners.
		if !f.Get(r.Min.X, r.Min.Y+10) || !f.Get(r.Max.X-1, r.Min.Y+10) || !f.Get(r.Min.X+10, r.Min.Y) || !f.Get(r.Min.X+10, r.Max.Y-1) {
			t.Errorf("tile %d outline missing", i+1)
		}
		if f.Get(r.Min.X, r.Min.Y) {
			t.Errorf("tile %d corner should be rounded", i+1)
		}
	}
	// Ratio strip: three groups, 4-px margins, equal gaps.
	if x := findPixelText(f, ratioY, "VTI/GLD", true, false); x != 4 {
		t.Errorf("ratio1 label at x=%d, want 4", x)
	}
	if x := findPixelText(f, ratioY, "1.41", true, true); x != 48 {
		t.Errorf("ratio1 value at x=%d, want 48", x)
	}
	if x := findPixelText(f, ratioY, "PSTG/VTI", true, false); x != 83 {
		t.Errorf("ratio2 label at x=%d, want 83", x)
	}
	if x := findPixelText(f, ratioY, "ORCL/VTI", true, false); x != 168 {
		t.Errorf("ratio3 label at x=%d, want 168", x)
	}
	if x := findPixelText(f, ratioY, "0.63", true, true); x != 218 {
		t.Errorf("ratio3 value at x=%d, want 218 (right margin 5)", x)
	}
	// Footer.
	if x := findPixelText(f, footerTextY, "42", false, false); x != 6 {
		t.Errorf("sum at x=%d, want 6", x)
	}
	if x := findPixelText(f, footerTextY, "Tue 11/03  07:15", false, false); x != 59 {
		t.Errorf("clock at x=%d, want 59 (centred in the free span)", x)
	}
	if x := findPixelText(f, footerTextY, "Wake 183", false, false); x != 197 {
		t.Errorf("wake at x=%d, want 197", x)
	}
	// Bands.
	if n := inkCount(f, image.Rect(0, 0, Width, 5), false); n != 0 {
		t.Errorf("header band top rows should be solid black, %d white pixels", n)
	}
	if n := inkCount(f, image.Rect(0, Height-2, Width, Height), false); n != 0 {
		t.Errorf("footer band bottom rows should be solid black, %d white pixels", n)
	}
}

func TestRender_Degraded(t *testing.T) {
	t.Run("missing symbol shows N/A and no pill", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tile1.Key = "NOPE"
		f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		r := tileRect(0)
		expectOutline(t, f, priceBand(0), tilePriceSz, r.Min.X+tilePriceDX, r.Min.Y+tilePriceDY, NA, true)
		if n := inkCount(f, pillArea(0), true); n != 0 {
			t.Errorf("pill area should be empty for a missing symbol, has %d black pixels", n)
		}
	})

	t.Run("hero missing symbol", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Hero.Key = "NOPE"
		f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		band := image.Rect(24, 0, 190, headerBottom+1)
		expectOutline(t, f, band, heroPriceSize, 6+PixelTextWidth("BTC", false)+heroGap, heroBaseline, NA, false)
	})

	t.Run("stale quotes replace the clock", func(t *testing.T) {
		in := fixtureInputs()
		f := mustRender(t, DefaultConfig(), in, fixtureNow.Add(4*time.Hour))
		if findPixelText(f, footerTextY, "Stale 4h", false, false) < 0 {
			t.Error("footer should read \"Stale 4h\"")
		}
		if findPixelText(f, footerTextY, "11/03", false, false) >= 0 {
			t.Error("the clock should give way to the stale marker")
		}
	})

	t.Run("missing params shows Cfg?", func(t *testing.T) {
		in := fixtureInputs()
		in.ConfigInvalid = true
		f := mustRender(t, DefaultConfig(), in, fixtureNow)
		if !pixelTextAt(f, sideMargin, footerTextY, "Cfg?", false, false) {
			t.Error("footer left should read \"Cfg?\"")
		}
		if findPixelText(f, footerTextY, "42", false, false) >= 0 {
			t.Error("the sum must not be drawn when the config is invalid")
		}
	})

	t.Run("empty slot renders an empty tile", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tile2 = nil
		f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		r := tileRect(1)
		if n := inkCount(f, r.Inset(4), true); n != 0 { // inset past the corner arcs
			t.Errorf("empty tile interior has %d black pixels", n)
		}
		if !f.Get(r.Min.X, r.Min.Y+10) {
			t.Error("empty tile should keep its outline")
		}
	})

	t.Run("empty ratio position", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Ratio2 = nil
		f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		if findPixelText(f, ratioY, "PSTG/VTI", true, false) >= 0 {
			t.Error("removed ratio still drawn")
		}
		if findPixelText(f, ratioY, "VTI/GLD", true, false) != 4 {
			t.Error("first ratio should keep the left margin")
		}
		// Two groups share the slack that three split in two; the right
		// margin is 4 or 5 depending on the remainder.
		if x := findPixelText(f, ratioY, "0.63", true, true); x < 218 || x > 219 {
			t.Errorf("last ratio value at x=%d, want it against the right margin", x)
		}
	})

	t.Run("ratio with zero denominator", func(t *testing.T) {
		cfg := DefaultConfig()
		in := fixtureInputs()
		in.Quotes.Prices["GLD"] = "0"
		f := mustRender(t, cfg, in, fixtureNow)
		if findPixelText(f, ratioY, NA, true, true) < 0 {
			t.Error("ratio with a zero denominator should read N/A")
		}
	})

	t.Run("no data at all never fails", func(t *testing.T) {
		f := mustRender(t, DefaultConfig(), Inputs{}, time.Time{})
		for i := 0; i < 4; i++ {
			r := tileRect(i)
			if !f.Get(r.Min.X, r.Min.Y+10) {
				t.Errorf("tile %d outline missing", i+1)
			}
		}
		if findPixelText(f, footerTextY, "Stale", false, false) < 0 {
			t.Error("quotes with no timestamp should be marked stale")
		}
		if findPixelText(f, footerTextY, "Wake —", false, false) < 0 {
			t.Error("no wake counter should read \"Wake —\"")
		}
		mustRender(t, Config{}, Inputs{}, time.Time{})
	})

	t.Run("unparseable price passes through", func(t *testing.T) {
		in := fixtureInputs()
		in.Quotes.Prices["VTI"] = "halted"
		f := mustRender(t, DefaultConfig(), in, fixtureNow)
		r := tileRect(0)
		expectOutline(t, f, priceBand(0), tilePriceSz, r.Min.X+tilePriceDX, r.Min.Y+tilePriceDY, "$halted", true)
	})
}

func TestParams_Slots(t *testing.T) {
	baseline := mustRender(t, DefaultConfig(), fixtureInputs(), fixtureNow)

	// Each slot's parameters land in its own region and nowhere else.
	t.Run("regions", func(t *testing.T) {
		tests := []struct {
			name    string
			mutate  func(*Config)
			regions []image.Rectangle
		}{
			{"hero", func(c *Config) { c.Hero.Label = "BITCOIN" }, []image.Rectangle{{Min: image.Pt(0, 0), Max: image.Pt(200, headerBottom+1)}}},
			{"aux", func(c *Config) { c.Aux.Key = "GLD" }, []image.Rectangle{{Min: image.Pt(195, 0), Max: image.Pt(Width, headerBottom+1)}}},
			// A tile's label also names it in the ratio strip, by design.
			{"tile1 label", func(c *Config) { c.Tile1.Label = "TOTAL" }, []image.Rectangle{tileRect(0), stripRect}},
			{"tile1", func(c *Config) { c.Tile1.Decimals = intp(0) }, []image.Rectangle{tileRect(0)}},
			{"tile2", func(c *Config) { c.Tile2.Prefix = "" }, []image.Rectangle{tileRect(1)}},
			{"tile3", func(c *Config) { c.Tile3.Decimals = intp(0) }, []image.Rectangle{tileRect(2)}},
			{"tile4", func(c *Config) { c.Tile4.Key = "GLD" }, []image.Rectangle{tileRect(3)}},
			{"ratio1", func(c *Config) { c.Ratio1.Label = "V/G" }, []image.Rectangle{stripRect}},
			{"ratio3", func(c *Config) { c.Ratio3.Decimals = intp(3) }, []image.Rectangle{stripRect}},
			{"sum", func(c *Config) { c.Sum = nil }, []image.Rectangle{footerRect}},
			{"clock", func(c *Config) { c.Clock.Format = "15:04" }, []image.Rectangle{footerRect}},
			{"wake", func(c *Config) { c.Wake.Label = "Up" }, []image.Rectangle{footerRect}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := DefaultConfig()
				tt.mutate(&cfg)
				f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
				changed := 0
				for y := 0; y < Height; y++ {
					for x := 0; x < Width; x++ {
						if f.Get(x, y) == baseline.Get(x, y) {
							continue
						}
						changed++
						inside := false
						for _, r := range tt.regions {
							inside = inside || image.Pt(x, y).In(r)
						}
						if !inside {
							t.Fatalf("pixel (%d,%d) changed outside the %s regions %v", x, y, tt.name, tt.regions)
						}
					}
				}
				if changed == 0 {
					t.Errorf("changing %s had no visible effect", tt.name)
				}
			})
		}
	})

	t.Run("hero label defaults to key", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Hero.Label = ""
		f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		if !pixelTextAt(f, sideMargin, heroLabelY, "BTC-USD", false, false) {
			t.Error("hero should print its key when it has no label")
		}
	})

	t.Run("hero prefix and decimals", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Hero.Prefix = ""
		cfg.Hero.Decimals = intp(2)
		f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		band := image.Rect(24, 0, 195, headerBottom+1)
		expectOutline(t, f, band, heroPriceSize, 6+PixelTextWidth("BTC", false)+heroGap, heroBaseline, "109,432.12", false)
	})

	t.Run("aux morning and evening", func(t *testing.T) {
		cfg := DefaultConfig()
		morning := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		if findPixelText(morning, auxLabelY, "IBIT", false, false) != 221 {
			t.Error("before noon the aux slot should show the morning symbol")
		}
		evening := mustRender(t, cfg, fixtureInputs(), fixtureNow.Add(12*time.Hour))
		if findPixelText(evening, auxLabelY, "STRC", false, false) != 221 {
			t.Error("after noon the aux slot should show the evening symbol")
		}
		if findPixelText(evening, auxPriceY, "$101.5", false, true) < 0 {
			t.Error("evening price should be the evening symbol's, trimmed not padded")
		}
		cfg.Aux.Key = "GLD"
		fixed := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		if findPixelText(fixed, auxLabelY, "GLD", false, false) < 0 {
			t.Error("an explicit key wins over the morning/evening pair")
		}
		cfg.Aux.Label = "GOLD"
		labelled := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		if findPixelText(labelled, auxLabelY, "GOLD", false, false) < 0 {
			t.Error("aux label should be printed when set")
		}
	})

	t.Run("tile label and prefix", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tile3.Label = ""
		cfg.Tile1.Prefix = ""
		f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		r := tileRect(2)
		if !pixelTextAt(f, r.Min.X+tileLabelDX, r.Min.Y+tileLabelDY, "P", true, false) {
			t.Error("tile label should default to the key")
		}
		r = tileRect(0)
		expectOutline(t, f, priceBand(0), tilePriceSz, r.Min.X+tilePriceDX, r.Min.Y+tilePriceDY, "301.20", true)
	})

	t.Run("ratio default label uses tile labels", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Ratio2.Label = ""
		groups := ratioGroups(cfg, fixtureInputs().Quotes)
		want := []ratioGroup{{"VTI/GLD", "1.41"}, {"PSTG/VTI", "0.19"}, {"ORCL/VTI", "0.63"}}
		if len(groups) != len(want) {
			t.Fatalf("groups = %v", groups)
		}
		for i := range want {
			if groups[i] != want[i] {
				t.Errorf("group %d = %v, want %v", i, groups[i], want[i])
			}
		}
		cfg.Tile3 = nil // nothing labels "P" any more
		groups = ratioGroups(cfg, fixtureInputs().Quotes)
		if groups[1].label != "P/VTI" {
			t.Errorf("without a tile label the raw key is used, got %q", groups[1].label)
		}
	})

	t.Run("sum is verbatim, first line, ten characters", func(t *testing.T) {
		in := fixtureInputs()
		in.Sum = "  1234567890123\nsecond line"
		f := mustRender(t, DefaultConfig(), in, fixtureNow)
		if !pixelTextAt(f, sideMargin, footerTextY, "1234567890", false, false) {
			t.Error("sum should be cut at ten characters, no separators added")
		}
		if findPixelText(f, footerTextY, "second", false, false) >= 0 {
			t.Error("only the first line of the sum is shown")
		}
		cfg := DefaultConfig()
		cfg.Sum = nil
		f = mustRender(t, cfg, in, fixtureNow)
		if n := inkCount(f, image.Rect(0, footerTop, 40, Height), false); n != 0 {
			t.Error("no sum slot, no sum text")
		}
	})

	t.Run("wake label", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Wake.Label = ""
		f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		if x := findPixelText(f, footerTextY, "183", false, false); x != Width-1-sideMargin-PixelTextWidth("183", false)+1 {
			t.Errorf("bare wake counter at x=%d, want right-aligned to 243", x)
		}
		in := fixtureInputs()
		in.HasWake = false
		f = mustRender(t, DefaultConfig(), in, fixtureNow)
		if findPixelText(f, footerTextY, "Wake —", false, false) < 0 {
			t.Error("missing counter should read \"Wake —\"")
		}
	})

	t.Run("clock format and zone", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Clock = ClockSlot{TZ: "UTC", Format: "2006-01-02 15:04"}
		f := mustRender(t, cfg, fixtureInputs(), fixtureNow)
		if findPixelText(f, footerTextY, "2026-11-03 15:15", false, false) < 0 {
			t.Error("clock should honour the configured zone and layout")
		}
	})
}

func TestPill_Thresholds(t *testing.T) {
	tests := []struct {
		name      string
		pct       float64
		threshold float64
		want      PillKind
	}{
		{"just under up", 0.04, 0.05, PillNone},
		{"at threshold up", 0.05, 0.05, PillUp},
		{"well up", 2.4, 0.05, PillUp},
		{"just under down", -0.04, 0.05, PillNone},
		{"at threshold down", -0.05, 0.05, PillDown},
		{"flat", 0, 0.05, PillNone},
		{"zero threshold still hides flat", 0, 0, PillNone},
		{"zero threshold shows any move", 0.001, 0, PillUp},
		{"zero threshold shows any drop", -0.001, 0, PillDown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyChange(tt.pct, tt.threshold); got != tt.want {
				t.Errorf("ClassifyChange(%v, %v) = %v, want %v", tt.pct, tt.threshold, got, tt.want)
			}
		})
	}

	t.Run("change", func(t *testing.T) {
		cur := Quotes{Prices: map[string]string{"A": "101", "B": "x", "C": "5", "Z": "1"}}
		prev := Quotes{Prices: map[string]string{"A": "100", "B": "1", "C": "0"}}
		if pct, ok := Change("A", cur, prev); !ok || pct < 0.999 || pct > 1.001 {
			t.Errorf("A: (%v, %v)", pct, ok)
		}
		for _, sym := range []string{"B", "C", "Z", "missing"} {
			if _, ok := Change(sym, cur, prev); ok {
				t.Errorf("%s: expected no change", sym)
			}
		}
	})

	t.Run("rendered pills", func(t *testing.T) {
		f := mustRender(t, DefaultConfig(), fixtureInputs(), fixtureNow)
		// VTI +0.60 % → filled pill, white text; GLD −1.20 % → outlined, black text.
		r := tileRect(0)
		if x := findPixelText(f, r.Min.Y+pillDY+3, "0.6%", false, false); x != r.Max.X-1-pillMarginR-PixelTextWidth("0.6%", false)-3 {
			t.Errorf("up pill text at x=%d", x)
		}
		r = tileRect(1)
		if x := findPixelText(f, r.Min.Y+pillDY+3, "1.2%", true, false); x != r.Max.X-1-pillMarginR-PixelTextWidth("1.2%", false)-3 {
			t.Errorf("down pill text at x=%d", x)
		}
		// Up pill is filled: a pixel between the triangle and the text is black.
		pillRight := tileRect(0).Max.X - 1 - pillMarginR
		pillLeft := pillRight - (PixelTextWidth("0.6%", false) + pillPad) + 1
		if !f.Get(pillLeft+10, tileRect(0).Min.Y+pillDY+6) {
			t.Error("up pill should be filled")
		}
		pillRight = tileRect(1).Max.X - 1 - pillMarginR
		pillLeft = pillRight - (PixelTextWidth("1.2%", false) + pillPad) + 1
		if f.Get(pillLeft+10, tileRect(1).Min.Y+pillDY+6) {
			t.Error("down pill should be outlined")
		}
		// Triangles: ▲ apex is a single pixel on the first row; ▼ is five wide.
		y := tileRect(0).Min.Y + pillDY + 5
		x := tileRect(0).Max.X - 1 - pillMarginR - (PixelTextWidth("0.6%", false) + pillPad) + 1 + 4
		if f.Get(x+2, y) || !f.Get(x+1, y) || !f.Get(x+3, y) { // white apex in a black pill
			t.Error("up triangle apex misplaced")
		}
		y = tileRect(1).Min.Y + pillDY + 5
		x = tileRect(1).Max.X - 1 - pillMarginR - (PixelTextWidth("1.2%", false) + pillPad) + 1 + 4
		for dx := 0; dx < 5; dx++ {
			if !f.Get(x+dx, y) {
				t.Errorf("down triangle top row missing at dx=%d", dx)
			}
		}
	})

	t.Run("no previous snapshot means no pills", func(t *testing.T) {
		in := fixtureInputs()
		in.Prev = Quotes{}
		f := mustRender(t, DefaultConfig(), in, fixtureNow)
		for i := 0; i < 4; i++ {
			if n := inkCount(f, pillArea(i), true); n != 0 {
				t.Errorf("tile %d has a pill without a previous snapshot", i+1)
			}
		}
	})

	t.Run("below threshold means no pill", func(t *testing.T) {
		in := fixtureInputs()
		in.Prev.Prices["VTI"] = "301.10" // +0.03 %
		f := mustRender(t, DefaultConfig(), in, fixtureNow)
		if n := inkCount(f, pillArea(0), true); n != 0 {
			t.Errorf("tile 1 shows a pill for a sub-threshold move")
		}
	})
}
