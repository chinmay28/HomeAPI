package main

import (
	"bytes"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

const repoRoot = "../.."

var planned = sync.OnceValues(func() ([]output, error) { return plan(repoRoot) })

// TestCommittedIconsMatchTheArtwork fails when icon.svg changes without a
// `make icons`, or when someone hand-edits a derived icon.
func TestCommittedIconsMatchTheArtwork(t *testing.T) {
	files, err := planned()
	if err != nil {
		t.Fatal(err)
	}
	stale, err := staleFiles(repoRoot, files)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range stale {
		t.Errorf("%s does not match %s — run: make icons", f, artPath)
	}
}

func TestEveryOutputHasOneOwnerAndTheRightSize(t *testing.T) {
	files, err := planned()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range files {
		if seen[f.path] {
			t.Errorf("%s is written twice", f.path)
		}
		seen[f.path] = true
		if !f.png {
			continue
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(f.data))
		if err != nil {
			t.Fatalf("%s: %v", f.path, err)
		}
		// Launcher mipmaps must be exactly their density's size, or Android
		// scales them again.
		for _, d := range densities {
			if !strings.Contains(f.path, "/mipmap-"+d.name+"/") {
				continue
			}
			dp := 48
			if strings.HasSuffix(f.path, "_foreground.png") {
				dp = 108
			}
			if want := dp * d.scale / 4; cfg.Width != want || cfg.Height != want {
				t.Errorf("%s is %dx%d, want %d", f.path, cfg.Width, cfg.Height, want)
			}
		}
	}
}

// The renderer draws what the SVG says: the plate's corners are cut away,
// the card is near-white, the key squares are the accent, and the adaptive
// foreground leaves the plate to the background layer.
func TestRenderMatchesTheArtwork(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(repoRoot, artPath))
	if err != nil {
		t.Fatal(err)
	}
	a, err := ParseArt(src)
	if err != nil {
		t.Fatal(err)
	}
	if p := a.Plate(); p.R != 0x1f || p.G != 0x29 || p.B != 0x33 {
		t.Errorf("plate = %v, want #1f2933", p)
	}
	img := a.Render(512, 1, true) // one pixel per viewBox unit
	at := func(img *image.NRGBA, x, y int) [4]uint8 {
		o := img.PixOffset(x, y)
		return [4]uint8{img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3]}
	}
	for _, c := range []struct {
		name string
		x, y int
		want [4]uint8
	}{
		{"cut corner", 2, 2, [4]uint8{}},
		{"plate", 40, 256, [4]uint8{0x1f, 0x29, 0x33, 255}},
		{"card", 256, 400, [4]uint8{0xf5, 0xf7, 0xf7, 255}},
		{"key", 185, 293, [4]uint8{0x4e, 0xcd, 0xc4, 255}},
		{"value", 280, 293, [4]uint8{0xc2, 0xd0, 0xcf, 255}},
		{"roof", 256, 110, [4]uint8{0x4e, 0xcd, 0xc4, 255}},
	} {
		if got := at(img, c.x, c.y); got != c.want {
			t.Errorf("%s at (%d,%d) = %v, want %v", c.name, c.x, c.y, got, c.want)
		}
	}
	fg := a.Render(108, 72.0/108, false)
	if got := at(fg, 3, 54); got[3] != 0 {
		t.Errorf("foreground has the plate in it: %v", got)
	}
	if got := at(fg, 54, 54); got[3] != 255 {
		t.Errorf("foreground centre is empty: %v", got)
	}
}

// What the renderer cannot draw is refused, never approximated.
func TestUnsupportedArtworkIsRefused(t *testing.T) {
	plate := `<rect width="10" height="10" fill="#000000"/>`
	for name, body := range map[string]string{
		"curve":      plate + `<path d="M1 1 C 2 2 3 3 4 4" fill="none" stroke="#ffffff" stroke-linecap="round"/>`,
		"circle":     plate + `<circle cx="5" cy="5" r="2"/>`,
		"transform":  plate + `<rect width="2" height="2" fill="#ffffff" transform="rotate(4)"/>`,
		"square cap": plate + `<line x1="1" y1="1" x2="2" y2="2" stroke="#ffffff" stroke-linecap="butt"/>`,
		"named fill": plate + `<rect width="2" height="2" fill="red"/>`,
		"no plate":   `<rect x="1" width="2" height="2" fill="#ffffff"/>`,
	} {
		svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">` + body + `</svg>`
		if _, err := ParseArt([]byte(svg)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestAndroidAppUsesTheAdaptiveIcon(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join(repoRoot, res, "../AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), `android:icon="@mipmap/ic_launcher"`) {
		t.Error("AndroidManifest.xml does not use @mipmap/ic_launcher")
	}
}

// TestAndroidIconReferencesResolve fails when the app refers to a drawable or
// mipmap that no longer exists, which only a Gradle build would otherwise
// notice. References into the platform are skipped.
func TestAndroidIconReferencesResolve(t *testing.T) {
	ref := regexp.MustCompile(`(?:@|\bR\.)(drawable|mipmap)[/.]([a-z0-9_]+)`)
	dir := filepath.Join(repoRoot, "apps/android/app/src/main")
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if ext := filepath.Ext(p); ext != ".kt" && ext != ".xml" {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range ref.FindAllSubmatchIndex(src, -1) {
			if m[0] >= 8 && string(src[m[0]-8:m[0]]) == "android." {
				continue
			}
			kind, name := string(src[m[2]:m[3]]), string(src[m[4]:m[5]])
			found, _ := filepath.Glob(filepath.Join(dir, "res", kind+"*", name+".*"))
			if len(found) == 0 {
				rel, _ := filepath.Rel(repoRoot, p)
				t.Errorf("%s refers to %s/%s, which no res/%s* directory has", rel, kind, name, kind)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStaleness(t *testing.T) {
	root := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	data, err := encodePNG(img)
	if err != nil {
		t.Fatal(err)
	}
	files := []output{
		{path: "a.png", data: data, png: true},
		{path: "b.xml", data: []byte("<b/>")},
		{path: "c.png", data: data, png: true},
	}
	if stale, _ := staleFiles(root, files); len(stale) != 3 {
		t.Fatalf("nothing written, but stale = %v", stale)
	}
	write := func(p string, b []byte) {
		if err := os.WriteFile(filepath.Join(root, p), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.NoCompression}).Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	write("a.png", buf.Bytes())
	write("b.xml", []byte("<b>edited</b>"))
	write("c.png", []byte("not a png"))
	stale, err := staleFiles(root, files)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(stale, ",") != "b.xml,c.png" {
		t.Errorf("stale = %v, want b.xml and c.png", stale)
	}
}
