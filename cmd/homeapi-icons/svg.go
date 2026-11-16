package main

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// The artwork is frontend/public/icon.svg, which the web GUI already shows. It
// is drawn from three shapes — rounded rectangles, round-capped lines and a
// round-joined polyline — so rather than take on an SVG library, this file
// reads exactly that subset and renders it by signed distance, supersampled.
// Anything outside the subset (a curve, a gradient, a transform) is an error,
// not a guess: if the artwork grows past what this draws, the generator says
// so instead of quietly drawing something else.

// shape is one filled region of the artwork, in its viewBox units.
type shape struct {
	fill color.NRGBA
	// dist is the signed distance from (x, y) to the shape's edge: negative
	// inside.
	dist func(x, y float64) float64
}

// Art is the parsed artwork.
type Art struct {
	size   float64 // the viewBox is square: 0 0 size size
	plate  color.NRGBA
	radius float64 // the plate's corner radius
	shapes []shape // painted in order; shapes[0] is the plate
}

// Plate is the colour of the rounded square everything sits on.
func (a Art) Plate() color.NRGBA { return a.plate }

type svgNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []svgNode  `xml:",any"`
}

func (n svgNode) attr(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// ParseArt reads the supported subset of SVG.
func ParseArt(src []byte) (Art, error) {
	var root svgNode
	if err := xml.Unmarshal(src, &root); err != nil {
		return Art{}, err
	}
	if root.XMLName.Local != "svg" {
		return Art{}, fmt.Errorf("root element is <%s>, want <svg>", root.XMLName.Local)
	}
	vb, _ := root.attr("viewBox")
	f := strings.Fields(vb)
	if len(f) != 4 || f[0] != "0" || f[1] != "0" || f[2] != f[3] {
		return Art{}, fmt.Errorf("viewBox %q: want a square \"0 0 N N\"", vb)
	}
	size, err := strconv.ParseFloat(f[2], 64)
	if err != nil || size <= 0 {
		return Art{}, fmt.Errorf("viewBox %q: %v", vb, err)
	}
	a := Art{size: size}
	if err := a.walk(root.Children); err != nil {
		return Art{}, err
	}
	if len(a.shapes) == 0 {
		return Art{}, fmt.Errorf("the artwork draws nothing")
	}
	return a, nil
}

func (a *Art) walk(nodes []svgNode) error {
	for _, n := range nodes {
		var err error
		switch n.XMLName.Local {
		case "g":
			if len(n.Attrs) > 0 {
				return fmt.Errorf("<g> with attributes is not supported")
			}
			err = a.walk(n.Children)
		case "rect":
			err = a.rect(n)
		case "line":
			err = a.line(n)
		case "path":
			err = a.path(n)
		default:
			err = fmt.Errorf("<%s> is not supported", n.XMLName.Local)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// allowed rejects attributes the renderer would otherwise silently ignore.
func allowed(n svgNode, names ...string) error {
	for _, at := range n.Attrs {
		ok := false
		for _, name := range names {
			ok = ok || at.Name.Local == name
		}
		if !ok {
			return fmt.Errorf("<%s %s> is not supported", n.XMLName.Local, at.Name.Local)
		}
	}
	return nil
}

func num(n svgNode, name string, def float64) (float64, error) {
	v, ok := n.attr(name)
	if !ok {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("<%s %s=%q>: %v", n.XMLName.Local, name, v, err)
	}
	return f, nil
}

func paint(n svgNode, name string) (color.NRGBA, error) {
	v, _ := n.attr(name)
	var c color.NRGBA
	if _, err := fmt.Sscanf(strings.ToLower(v), "#%02x%02x%02x", &c.R, &c.G, &c.B); err != nil || len(v) != 7 {
		return c, fmt.Errorf("<%s %s=%q>: want #rrggbb", n.XMLName.Local, name, v)
	}
	c.A = 255
	return c, nil
}

func (a *Art) rect(n svgNode) error {
	if err := allowed(n, "x", "y", "width", "height", "rx", "fill"); err != nil {
		return err
	}
	var v [5]float64
	for i, name := range []string{"x", "y", "width", "height", "rx"} {
		f, err := num(n, name, 0)
		if err != nil {
			return err
		}
		v[i] = f
	}
	fill, err := paint(n, "fill")
	if err != nil {
		return err
	}
	x, y, w, h, r := v[0], v[1], v[2], v[3], math.Min(v[4], math.Min(v[2], v[3])/2)
	if len(a.shapes) == 0 {
		if x != 0 || y != 0 || w != a.size || h != a.size {
			return fmt.Errorf("the first shape must be the plate: a rect covering the whole viewBox")
		}
		a.plate, a.radius = fill, r
	}
	cx, cy, hx, hy := x+w/2, y+h/2, w/2-r, h/2-r
	a.shapes = append(a.shapes, shape{fill, func(px, py float64) float64 {
		qx, qy := math.Abs(px-cx)-hx, math.Abs(py-cy)-hy
		return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
	}})
	return nil
}

// stroke checks a stroked shape's attributes and returns its colour and half
// width. Only round caps and joins are drawn, so only those are accepted.
func stroke(n svgNode) (color.NRGBA, float64, error) {
	if v, _ := n.attr("stroke-linecap"); v != "round" {
		return color.NRGBA{}, 0, fmt.Errorf("<%s>: only stroke-linecap=\"round\" is supported", n.XMLName.Local)
	}
	if v, ok := n.attr("stroke-linejoin"); ok && v != "round" {
		return color.NRGBA{}, 0, fmt.Errorf("<%s>: only stroke-linejoin=\"round\" is supported", n.XMLName.Local)
	}
	c, err := paint(n, "stroke")
	if err != nil {
		return c, 0, err
	}
	w, err := num(n, "stroke-width", 1)
	return c, w / 2, err
}

// capsules is the distance to a polyline stroked with round caps and joins:
// the nearest of its segments, each a capsule.
func capsules(pts [][2]float64, half float64) func(x, y float64) float64 {
	return func(px, py float64) float64 {
		best := math.Inf(1)
		for i := 0; i+1 < len(pts); i++ {
			ax, ay, bx, by := pts[i][0], pts[i][1], pts[i+1][0], pts[i+1][1]
			dx, dy := bx-ax, by-ay
			t := 0.0
			if l := dx*dx + dy*dy; l > 0 {
				t = math.Max(0, math.Min(1, ((px-ax)*dx+(py-ay)*dy)/l))
			}
			best = math.Min(best, math.Hypot(px-(ax+t*dx), py-(ay+t*dy)))
		}
		return best - half
	}
}

func (a *Art) line(n svgNode) error {
	if err := allowed(n, "x1", "y1", "x2", "y2", "stroke", "stroke-width", "stroke-linecap"); err != nil {
		return err
	}
	if len(a.shapes) == 0 {
		return fmt.Errorf("the first shape must be the plate rect")
	}
	var v [4]float64
	for i, name := range []string{"x1", "y1", "x2", "y2"} {
		f, err := num(n, name, 0)
		if err != nil {
			return err
		}
		v[i] = f
	}
	c, half, err := stroke(n)
	if err != nil {
		return err
	}
	a.shapes = append(a.shapes, shape{c, capsules([][2]float64{{v[0], v[1]}, {v[2], v[3]}}, half)})
	return nil
}

// pathTokens splits path data into commands and numbers ("M92 242" is M, 92, 242).
var pathTokens = regexp.MustCompile(`[A-Za-z]|-?(?:\d+\.?\d*|\.\d+)`)

func (a *Art) path(n svgNode) error {
	if err := allowed(n, "d", "fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin"); err != nil {
		return err
	}
	if len(a.shapes) == 0 {
		return fmt.Errorf("the first shape must be the plate rect")
	}
	if v, _ := n.attr("fill"); v != "none" {
		return fmt.Errorf("<path>: only stroked paths (fill=\"none\") are supported")
	}
	d, _ := n.attr("d")
	f := pathTokens.FindAllString(d, -1)
	if strings.Join(f, "") != strings.Join(strings.Fields(strings.ReplaceAll(d, ",", " ")), "") {
		return fmt.Errorf("<path d=%q>: only \"M x y L x y …\" is supported", d)
	}
	var pts [][2]float64
	for i := 0; i < len(f); i += 3 {
		if i+2 >= len(f) || (f[i] != "M" && f[i] != "L") || (f[i] == "M") != (i == 0) {
			return fmt.Errorf("<path d=%q>: only \"M x y L x y …\" is supported", d)
		}
		x, err1 := strconv.ParseFloat(f[i+1], 64)
		y, err2 := strconv.ParseFloat(f[i+2], 64)
		if err1 != nil || err2 != nil {
			return fmt.Errorf("<path d=%q>: bad number", d)
		}
		pts = append(pts, [2]float64{x, y})
	}
	if len(pts) < 2 {
		return fmt.Errorf("<path d=%q>: want at least two points", d)
	}
	c, half, err := stroke(n)
	if err != nil {
		return err
	}
	a.shapes = append(a.shapes, shape{c, capsules(pts, half)})
	return nil
}

// samples is the supersampling grid per pixel side: 6×6 = 36 samples, which
// is what keeps a 48px icon's diagonals smooth.
const samples = 6

// Render draws the artwork into a px×px image. The artwork is scaled to
// scale×px and centred; withPlate false leaves the plate out (an adaptive
// icon's foreground, which goes over a background of the plate colour).
func (a Art) Render(px int, scale float64, withPlate bool) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, px, px))
	first := 0
	if !withPlate {
		first = 1
	}
	// Output pixel space → viewBox units.
	unit := a.size / (scale * float64(px))
	off := (float64(px) - scale*float64(px)) / 2
	for y := 0; y < px; y++ {
		for x := 0; x < px; x++ {
			var r, g, b, al float64 // premultiplied sums
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					ux := float64((float64(x) + (float64(sx)+0.5)/samples - off) * unit)
					uy := float64((float64(y) + (float64(sy)+0.5)/samples - off) * unit)
					// Painter's order: the last opaque shape under the
					// sample is the colour it sees.
					for i := len(a.shapes) - 1; i >= first; i-- {
						if a.shapes[i].dist(ux, uy) <= 0 {
							c := a.shapes[i].fill
							r, g, b, al = r+float64(c.R), g+float64(c.G), b+float64(c.B), al+1
							break
						}
					}
				}
			}
			if al == 0 {
				continue
			}
			o := img.PixOffset(x, y)
			img.Pix[o+0] = uint8(math.Round(r / al))
			img.Pix[o+1] = uint8(math.Round(g / al))
			img.Pix[o+2] = uint8(math.Round(b / al))
			img.Pix[o+3] = uint8(math.Round(255 * al / (samples * samples)))
		}
	}
	return img
}
