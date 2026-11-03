package display

import (
	"bytes"
	"image/png"
	"testing"
)

func TestFrame_Native(t *testing.T) {
	tests := []struct {
		name string
		x, y int
	}{
		{"origin", 0, 0},
		{"far corner", Width - 1, Height - 1},
		{"top right", Width - 1, 0},
		{"bottom left", 0, Height - 1},
		{"interior", 37, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFrame()
			f.Set(tt.x, tt.y, true)
			body := f.Native()
			if len(body) != NativeSize {
				t.Fatalf("len = %d, want %d", len(body), NativeSize)
			}
			nx, ny := tt.y, Width-1-tt.x
			zeros := 0
			for i, b := range body {
				row, col := i/NativeStride, i%NativeStride
				for bit := 0; bit < 8; bit++ {
					if b&(0x80>>uint(bit)) != 0 {
						continue
					}
					zeros++
					if row != ny || col*8+bit != nx {
						t.Errorf("unexpected black bit at native (%d,%d), want only (%d,%d)", col*8+bit, row, nx, ny)
					}
				}
			}
			if zeros != 1 {
				t.Errorf("black bits = %d, want exactly 1", zeros)
			}
			if !f.Get(tt.x, tt.y) {
				t.Errorf("Get(%d,%d) = white, want black", tt.x, tt.y)
			}
		})
	}
}

func TestFrame_PaddingBitsStayWhite(t *testing.T) {
	f := NewFrame()
	f.FillRect(0, 0, Width-1, Height-1, true)
	body := f.Native()
	for row := 0; row < NativeRows; row++ {
		if got := body[row*NativeStride+15] & 0x3F; got != 0x3F {
			t.Fatalf("row %d padding bits = %06b, want all 1", row, got)
		}
	}
	// Every real column black.
	for row := 0; row < NativeRows; row++ {
		for col := 0; col < Height; col++ {
			if body[row*NativeStride+col/8]&(0x80>>uint(col%8)) != 0 {
				t.Fatalf("native (%d,%d) white, want black", col, row)
			}
		}
	}
}

func TestFrame_OutOfBoundsIsIgnored(t *testing.T) {
	f := NewFrame()
	f.Set(-1, 0, true)
	f.Set(Width, 0, true)
	f.Set(0, Height, true)
	if !bytes.Equal(f.Native(), NewFrame().Native()) {
		t.Error("out-of-bounds Set changed the frame")
	}
	if f.Get(-1, -1) {
		t.Error("out-of-bounds Get should read white")
	}
}

func TestFrame_RoundRectOutline(t *testing.T) {
	f := NewFrame()
	f.RoundRect(10, 10, 30, 20, 4, true, false)
	if f.Get(10, 10) {
		t.Error("corner pixel should be clipped by the radius")
	}
	for _, p := range [][2]int{{14, 10}, {26, 10}, {14, 20}, {10, 14}, {30, 16}} {
		if !f.Get(p[0], p[1]) {
			t.Errorf("edge pixel (%d,%d) should be black", p[0], p[1])
		}
	}
	if f.Get(20, 15) {
		t.Error("interior should stay white for an outline")
	}
	// Four-fold symmetry: the arc is drawn the same way in every corner.
	for dx := 0; dx <= 4; dx++ {
		for dy := 0; dy <= 4; dy++ {
			tl := f.Get(10+dx, 10+dy)
			if tr := f.Get(30-dx, 10+dy); tr != tl {
				t.Errorf("corner asymmetry at dx=%d dy=%d (top-right)", dx, dy)
			}
			if bl := f.Get(10+dx, 20-dy); bl != tl {
				t.Errorf("corner asymmetry at dx=%d dy=%d (bottom-left)", dx, dy)
			}
		}
	}
}

func TestFrame_RoundRectFilled(t *testing.T) {
	f := NewFrame()
	f.RoundRect(10, 10, 47, 21, 3, true, true)
	if !f.Get(20, 15) || !f.Get(11, 15) || !f.Get(46, 15) {
		t.Error("interior should be black for a filled pill")
	}
	if f.Get(10, 10) || f.Get(47, 21) {
		t.Error("corners should be rounded off")
	}
	// A filled shape is at least as inked as its own outline.
	o := NewFrame()
	o.RoundRect(10, 10, 47, 21, 3, true, false)
	for y := 10; y <= 21; y++ {
		for x := 10; x <= 47; x++ {
			if o.Get(x, y) && !f.Get(x, y) {
				t.Fatalf("outline pixel (%d,%d) missing from fill", x, y)
			}
		}
	}
}

func TestFrame_PBMRoundTrip(t *testing.T) {
	f := NewFrame()
	f.FillRect(5, 7, 60, 40, true)
	f.DrawPixelText(100, 100, "PBM", true, false)
	back, err := ParsePBM(f.PBM())
	if err != nil {
		t.Fatalf("ParsePBM: %v", err)
	}
	if n := f.Diff(back); n != 0 {
		t.Errorf("round trip differs in %d pixels", n)
	}
	if _, err := ParsePBM([]byte("P1\n1 1\n0")); err == nil {
		t.Error("expected an error for a foreign PBM")
	}
}

func TestFrame_PNGScale(t *testing.T) {
	f := NewFrame()
	f.Set(1, 2, true)
	for _, scale := range []int{1, 3, 8} {
		data, err := f.PNG(scale)
		if err != nil {
			t.Fatalf("PNG(%d): %v", scale, err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("decode PNG(%d): %v", scale, err)
		}
		b := img.Bounds()
		if b.Dx() != Width*scale || b.Dy() != Height*scale {
			t.Errorf("PNG(%d) = %dx%d, want %dx%d", scale, b.Dx(), b.Dy(), Width*scale, Height*scale)
		}
		r, _, _, _ := img.At(1*scale, 2*scale).RGBA()
		if r != 0 {
			t.Errorf("PNG(%d): black pixel not black at scaled position", scale)
		}
		r, _, _, _ = img.At(0, 0).RGBA()
		if r == 0 {
			t.Errorf("PNG(%d): white pixel rendered black", scale)
		}
	}
}
