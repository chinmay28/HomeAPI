package display

import (
	"image"
	"testing"
)

// glyphRows renders the 5×7 glyph for r as seven strings of '#' and '.'.
func glyphRows(f *Frame, x, y int, bold bool) []string {
	w := pixelGlyphW
	if bold {
		w = pixelBoldW
	}
	rows := make([]string, pixelGlyphH)
	for row := 0; row < pixelGlyphH; row++ {
		b := make([]byte, w)
		for col := 0; col < w; col++ {
			b[col] = '.'
			if f.Get(x+col, y+row) {
				b[col] = '#'
			}
		}
		rows[row] = string(b)
	}
	return rows
}

func TestPixelFont_GlyphShapes(t *testing.T) {
	tests := []struct {
		r    rune
		bold bool
		want []string
	}{
		{'1', false, []string{
			"..#..",
			".##..",
			"..#..",
			"..#..",
			"..#..",
			"..#..",
			".###.",
		}},
		{'G', false, []string{
			".###.",
			"#...#",
			"#....",
			"#.###",
			"#...#",
			"#...#",
			".####",
		}},
		{'%', false, []string{
			"##...",
			"##..#",
			"...#.",
			"..#..",
			".#...",
			"#..##",
			"...##",
		}},
		{'1', true, []string{
			"..##..",
			".###..",
			"..##..",
			"..##..",
			"..##..",
			"..##..",
			".####.",
		}},
		{'—', false, []string{
			".....",
			".....",
			".....",
			"#####",
			".....",
			".....",
			".....",
		}},
	}
	for _, tt := range tests {
		t.Run(string(tt.r), func(t *testing.T) {
			f := NewFrame()
			f.DrawPixelText(10, 10, string(tt.r), true, tt.bold)
			got := glyphRows(f, 10, 10, tt.bold)
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("row %d = %q, want %q\nfull:\n%v", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}

func TestPixelFont_Advance(t *testing.T) {
	f := NewFrame()
	end := f.DrawPixelText(0, 0, "AB", true, false)
	if end != 2*pixelAdvance {
		t.Errorf("regular advance = %d, want %d", end, 2*pixelAdvance)
	}
	end = f.DrawPixelText(0, 20, "AB", true, true)
	if end != 2*pixelBoldAdv {
		t.Errorf("bold advance = %d, want %d", end, 2*pixelBoldAdv)
	}
	// Column 5 is the spacing column and must stay empty in regular weight.
	for row := 0; row < pixelGlyphH; row++ {
		if f.Get(5, row) {
			t.Errorf("spacing column inked at row %d", row)
		}
	}
}

func TestPixelTextWidth(t *testing.T) {
	tests := []struct {
		s    string
		bold bool
		want int
	}{
		{"", false, 0},
		{"A", false, 5},
		{"ABC", false, 17},
		{"A", true, 6},
		{"1.41", true, 27},
		{"Tue 11/03  07:15", false, 95},
		{"Wake 183", false, 47},
	}
	for _, tt := range tests {
		if got := PixelTextWidth(tt.s, tt.bold); got != tt.want {
			t.Errorf("PixelTextWidth(%q, bold=%v) = %d, want %d", tt.s, tt.bold, got, tt.want)
		}
	}
}

func TestPixelFont_UnknownRuneIsVisible(t *testing.T) {
	f := NewFrame()
	f.DrawPixelText(0, 0, "é", true, false)
	if !f.Get(0, 0) || !f.Get(4, 6) {
		t.Error("an unknown rune should render as a box, not vanish")
	}
}

func TestOutlineText_DrawsWithinClip(t *testing.T) {
	f := NewFrame()
	clip := image.Rect(10, 10, 60, 40)
	end, err := f.DrawOutlineText(17, 12, 34, "301.20", true, clip)
	if err != nil {
		t.Fatalf("DrawOutlineText: %v", err)
	}
	if end <= 12 {
		t.Errorf("pen did not advance: %d", end)
	}
	ink := 0
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			if !f.Get(x, y) {
				continue
			}
			ink++
			if !image.Pt(x, y).In(clip) {
				t.Fatalf("ink at (%d,%d) outside clip %v", x, y, clip)
			}
		}
	}
	if ink == 0 {
		t.Error("no ink drawn")
	}
	// Digits sit on the baseline: the row just above it is inked, the row
	// at it is not (no descenders in the string).
	rowInk := func(y int) bool {
		for x := clip.Min.X; x < clip.Max.X; x++ {
			if f.Get(x, y) {
				return true
			}
		}
		return false
	}
	if !rowInk(33) {
		t.Error("row above baseline should carry ink")
	}
	if rowInk(34) {
		t.Error("baseline row should be clear for digits")
	}
}

func TestOutlineText_Width(t *testing.T) {
	w17, err := OutlineTextWidth(17, "$301.20")
	if err != nil {
		t.Fatal(err)
	}
	w20, err := OutlineTextWidth(20, "$301.20")
	if err != nil {
		t.Fatal(err)
	}
	if w17 <= 0 || w20 <= w17 {
		t.Errorf("widths: 17px=%d 20px=%d; want 0 < w17 < w20", w17, w20)
	}
	if w17 > tileW-tilePriceDX {
		t.Errorf("a typical tile price (%d px) must fit a tile", w17)
	}
}
