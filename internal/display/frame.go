// Package display renders the Minion 3 e-paper dashboard: a 250×122 1-bit
// frame built from a handful of HomeAPI entries. It is pure rendering — no
// HTTP, no database — so a frame is a deterministic function of its inputs
// and the clock, which is what makes golden tests and the `?at=` preview
// parameter possible.
package display

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// Panel geometry. The design is landscape (250 wide × 122 tall, FPC on the
// right); the panel itself is portrait and wants 250 rows of 122 columns.
const (
	Width  = 250 // landscape width
	Height = 122 // landscape height

	// NativeRows is the panel's own row count, NativeStride the bytes per
	// native row (122 columns padded to 128 bits) and NativeSize the body
	// length of a format-1 frame.
	NativeRows   = Width
	NativeStride = 16
	NativeSize   = NativeRows * NativeStride // 4000
)

// Frame is a 1-bit bitmap in the panel's native packing: NativeRows rows of
// NativeStride bytes, MSB = leftmost column, bit 1 = white, 0 = black, with
// the six padding bits of every row held at 1. All drawing happens in
// landscape coordinates; the native mapping is native_x = y, native_y = 249 − x,
// which matches FB_ROTATE 0 in the device's framebuf, so the body can be
// copied straight into epd_display().
//
// The zero Frame is not usable; call NewFrame.
type Frame struct {
	buf [NativeSize]byte
}

// NewFrame returns an all-white frame.
func NewFrame() *Frame {
	f := &Frame{}
	for i := range f.buf {
		f.buf[i] = 0xFF
	}
	return f
}

// inBounds reports whether (x, y) is a landscape pixel of the frame.
func inBounds(x, y int) bool {
	return x >= 0 && x < Width && y >= 0 && y < Height
}

// native maps a landscape pixel to its byte index and bit mask.
func native(x, y int) (idx int, mask byte) {
	nx, ny := y, Width-1-x
	return ny*NativeStride + nx/8, 0x80 >> uint(nx%8)
}

// Set paints the landscape pixel (x, y) black or white. Pixels outside the
// frame are ignored, so callers may draw partially clipped shapes freely.
func (f *Frame) Set(x, y int, black bool) {
	if !inBounds(x, y) {
		return
	}
	idx, mask := native(x, y)
	if black {
		f.buf[idx] &^= mask
	} else {
		f.buf[idx] |= mask
	}
}

// Get reports whether the landscape pixel (x, y) is black. Pixels outside the
// frame read as white.
func (f *Frame) Get(x, y int) bool {
	if !inBounds(x, y) {
		return false
	}
	idx, mask := native(x, y)
	return f.buf[idx]&mask == 0
}

// Native returns a copy of the 4,000-byte panel-native body (format 1).
func (f *Frame) Native() []byte {
	out := make([]byte, NativeSize)
	copy(out, f.buf[:])
	return out
}

// FillRect paints the landscape rectangle [x0, x1] × [y0, y1] (inclusive).
func (f *Frame) FillRect(x0, y0, x1, y1 int, black bool) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			f.Set(x, y, black)
		}
	}
}

// HLine draws a horizontal run from x0 to x1 inclusive on row y.
func (f *Frame) HLine(x0, x1, y int, black bool) {
	for x := x0; x <= x1; x++ {
		f.Set(x, y, black)
	}
}

// VLine draws a vertical run from y0 to y1 inclusive in column x.
func (f *Frame) VLine(x, y0, y1 int, black bool) {
	for y := y0; y <= y1; y++ {
		f.Set(x, y, black)
	}
}

// RoundRect draws a rounded rectangle whose outer edge spans [x0, x1] × [y0, y1]
// inclusive, with the given corner radius. When fill is true the interior is
// painted too; otherwise only the 1-px outline.
func (f *Frame) RoundRect(x0, y0, x1, y1, r int, black, fill bool) {
	if r < 0 {
		r = 0
	}
	if 2*r > x1-x0 {
		r = (x1 - x0) / 2
	}
	if 2*r > y1-y0 {
		r = (y1 - y0) / 2
	}
	if fill {
		f.FillRect(x0+r, y0, x1-r, y1, black)
		f.FillRect(x0, y0+r, x0+r, y1-r, black)
		f.FillRect(x1-r, y0+r, x1, y1-r, black)
	} else {
		f.HLine(x0+r, x1-r, y0, black)
		f.HLine(x0+r, x1-r, y1, black)
		f.VLine(x0, y0+r, y1-r, black)
		f.VLine(x1, y0+r, y1-r, black)
	}
	if r == 0 {
		return
	}
	// Quarter arcs by the midpoint circle algorithm; they meet the straight
	// edges exactly because the edge ends are at the arc's own end points.
	cx0, cy0, cx1, cy1 := x0+r, y0+r, x1-r, y1-r
	x, y, d := r, 0, 1-r
	for x >= y {
		if fill {
			f.HLine(cx0-x, cx1+x, cy0-y, black)
			f.HLine(cx0-x, cx1+x, cy1+y, black)
			f.HLine(cx0-y, cx1+y, cy0-x, black)
			f.HLine(cx0-y, cx1+y, cy1+x, black)
		} else {
			f.Set(cx1+x, cy0-y, black)
			f.Set(cx1+y, cy0-x, black)
			f.Set(cx0-x, cy0-y, black)
			f.Set(cx0-y, cy0-x, black)
			f.Set(cx1+x, cy1+y, black)
			f.Set(cx1+y, cy1+x, black)
			f.Set(cx0-x, cy1+y, black)
			f.Set(cx0-y, cy1+x, black)
		}
		y++
		if d < 0 {
			d += 2*y + 1
		} else {
			x--
			d += 2*(y-x) + 1
		}
	}
}

// Image returns the frame as an 8-bit grayscale image, upsampled by scale
// (nearest-neighbour, so pixels stay square). Scale must be ≥ 1.
func (f *Frame) Image(scale int) *image.Gray {
	if scale < 1 {
		scale = 1
	}
	img := image.NewGray(image.Rect(0, 0, Width*scale, Height*scale))
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			c := color.Gray{Y: 0xFF}
			if f.Get(x, y) {
				c = color.Gray{Y: 0}
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.SetGray(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}
	return img
}

// PNG encodes the frame as a PNG at the given integer scale.
func (f *Frame) PNG(scale int) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, f.Image(scale)); err != nil {
		return nil, fmt.Errorf("encoding frame png: %w", err)
	}
	return buf.Bytes(), nil
}

// PBM encodes the frame as a binary PBM (P4), the golden-file format: one bit
// per pixel in landscape row-major order, 1 = black, which any image viewer
// opens for a visual diff.
func (f *Frame) PBM() []byte {
	stride := (Width + 7) / 8
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "P4\n%d %d\n", Width, Height)
	row := make([]byte, stride)
	for y := 0; y < Height; y++ {
		for i := range row {
			row[i] = 0
		}
		for x := 0; x < Width; x++ {
			if f.Get(x, y) {
				row[x/8] |= 0x80 >> uint(x%8)
			}
		}
		buf.Write(row)
	}
	return buf.Bytes()
}

// ParsePBM decodes a binary PBM written by PBM back into a Frame.
func ParsePBM(data []byte) (*Frame, error) {
	header := fmt.Sprintf("P4\n%d %d\n", Width, Height)
	if !bytes.HasPrefix(data, []byte(header)) {
		return nil, fmt.Errorf("pbm: unexpected header (want %q)", header)
	}
	stride := (Width + 7) / 8
	body := data[len(header):]
	if len(body) != stride*Height {
		return nil, fmt.Errorf("pbm: body is %d bytes, want %d", len(body), stride*Height)
	}
	f := NewFrame()
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			if body[y*stride+x/8]&(0x80>>uint(x%8)) != 0 {
				f.Set(x, y, true)
			}
		}
	}
	return f, nil
}

// Diff returns the number of landscape pixels that differ between two frames.
func (f *Frame) Diff(o *Frame) int {
	n := 0
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			if f.Get(x, y) != o.Get(x, y) {
				n++
			}
		}
	}
	return n
}
