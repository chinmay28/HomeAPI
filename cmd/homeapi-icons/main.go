// Command homeapi-icons derives the Android app's icons from the one piece of
// artwork HomeAPI has, frontend/public/icon.svg.
//
// The outputs are committed; change the artwork, run `make icons`, commit
// what falls out, and never edit an output by hand. `make icons-check` (and
// `go test ./cmd/homeapi-icons`, which `make test` runs) fails when a
// committed copy has drifted from the artwork.
//
// Usage:
//
//	homeapi-icons [-root DIR]          # write every output
//	homeapi-icons [-root DIR] -check   # exit 1 if any output is stale
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

func main() {
	root := flag.String("root", ".", "repository root")
	check := flag.Bool("check", false, "report stale outputs instead of writing them")
	flag.Parse()

	if err := run(*root, *check); err != nil {
		fmt.Fprintln(os.Stderr, "homeapi-icons:", err)
		os.Exit(1)
	}
}

func run(root string, check bool) error {
	files, err := plan(root)
	if err != nil {
		return err
	}
	if check {
		stale, err := staleFiles(root, files)
		if err != nil {
			return err
		}
		for _, f := range stale {
			fmt.Fprintf(os.Stderr, "stale: %s\n", f)
		}
		if len(stale) > 0 {
			return fmt.Errorf("%d icon file(s) out of date — run: make icons", len(stale))
		}
		fmt.Println("icons match", artPath)
		return nil
	}
	for _, f := range files {
		p := filepath.Join(root, f.path)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, f.data, 0o644); err != nil {
			return err
		}
		fmt.Printf("%-72s %6.1f KB\n", f.path, float64(len(f.data))/1024)
	}
	return nil
}

// output is one derived file: where it goes and what it holds.
type output struct {
	path string
	data []byte
	// png outputs are compared by their pixels, not their bytes, so a change
	// in the encoder's compression alone never reads as a stale icon.
	png bool
}

func loadArt(root string) (Art, error) {
	src, err := os.ReadFile(filepath.Join(root, artPath))
	if err != nil {
		return Art{}, err
	}
	a, err := ParseArt(src)
	if err != nil {
		return Art{}, fmt.Errorf("%s: %w", artPath, err)
	}
	return a, nil
}

// plan renders every output without writing anything.
func plan(root string) ([]output, error) {
	art, err := loadArt(root)
	if err != nil {
		return nil, err
	}
	var out []output
	for _, ic := range icons() {
		data, err := encodePNG(ic.draw(art))
		if err != nil {
			return nil, err
		}
		out = append(out, output{path: ic.path, data: data, png: true})
	}
	for _, t := range texts() {
		out = append(out, output{path: t.path, data: t.data(art)})
	}
	return out, nil
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// staleFiles lists the outputs whose committed file is missing or differs.
func staleFiles(root string, files []output) ([]string, error) {
	var stale []string
	for _, f := range files {
		have, err := os.ReadFile(filepath.Join(root, f.path))
		if errors.Is(err, fs.ErrNotExist) {
			stale = append(stale, f.path)
			continue
		}
		if err != nil {
			return nil, err
		}
		same := bytes.Equal(have, f.data)
		if !same && f.png {
			if same, err = samePixels(have, f.data); err != nil {
				return nil, fmt.Errorf("%s: %w", f.path, err)
			}
		}
		if !same {
			stale = append(stale, f.path)
		}
	}
	return stale, nil
}

// samePixels reports whether two PNGs decode to the same image. A committed
// file that will not decode is simply stale.
func samePixels(have, want []byte) (bool, error) {
	w, err := png.Decode(bytes.NewReader(want))
	if err != nil {
		return false, err
	}
	h, err := png.Decode(bytes.NewReader(have))
	if err != nil {
		return false, nil
	}
	hn, wn := toNRGBA(h), toNRGBA(w)
	return hn.Rect == wn.Rect && slices.Equal(hn.Pix, wn.Pix), nil
}

func toNRGBA(img image.Image) *image.NRGBA {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Rect, img, b.Min, draw.Src)
	return out
}
