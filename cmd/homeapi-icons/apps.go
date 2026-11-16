package main

import (
	"fmt"
	"image"
	"path"
)

// artPath is the artwork every icon is derived from: the mark the web GUI
// already shows in its header and as its favicon.
const artPath = "frontend/public/icon.svg"

// res is the Android app's resource directory.
const res = "apps/android/app/src/main/res"

// icon is one derived PNG.
type icon struct {
	path string
	draw func(Art) *image.NRGBA
}

// text is one derived text file.
type text struct {
	path string
	data func(Art) []byte
}

// densities are Android's launcher-icon buckets, as multiples of mdpi.
var densities = []struct {
	name  string
	scale int // in quarters: mdpi is 4
}{{"mdpi", 4}, {"hdpi", 6}, {"xhdpi", 8}, {"xxhdpi", 12}, {"xxxhdpi", 16}}

// icons is everything drawn from the artwork:
//
//   - mipmap-*/ic_launcher.png, the 48dp legacy launcher icon, rounded plate
//     on transparency, for launchers before Android 8;
//   - mipmap-*/ic_launcher_foreground.png, the 108dp foreground layer of the
//     adaptive icon: the artwork without its plate, in the 72dp a launcher
//     shows, over a background of the plate colour;
//   - drawable-nodpi/homeapi_logo.png, the mark on the app's own screens
//     (setup, the status panel, the widget), 72dp at most.
func icons() []icon {
	var out []icon
	for _, d := range densities {
		px := func(dp int) int { return dp * d.scale / 4 }
		dir := path.Join(res, "mipmap-"+d.name)
		out = append(out,
			icon{path.Join(dir, "ic_launcher.png"), func(a Art) *image.NRGBA { return a.Render(px(48), 1, true) }},
			icon{path.Join(dir, "ic_launcher_foreground.png"), func(a Art) *image.NRGBA { return a.Render(px(108), 72.0/108, false) }},
		)
	}
	return append(out, icon{path.Join(res, "drawable-nodpi/homeapi_logo.png"), func(a Art) *image.NRGBA { return a.Render(288, 1, true) }})
}

// texts is the adaptive icon that puts the foreground over the plate colour,
// with the hand-drawn silhouette (drawable/ic_launcher_monochrome.xml, the
// notification icon inset to the 72dp safe zone) for Android 13's themed
// icons.
func texts() []text {
	return []text{
		// -v26 although every Android this app runs on (29+) has adaptive
		// icons: in mipmap-anydpi, lint objects to the legacy PNGs beside it,
		// which launchers that ignore the adaptive icon still want.
		{path.Join(res, "mipmap-anydpi-v26/ic_launcher.xml"), func(Art) []byte {
			return []byte(`<?xml version="1.0" encoding="utf-8"?>
<!-- Generated from ` + artPath + ` by ` + "`make icons`" + `; edit that, not this. -->
<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">
    <background android:drawable="@color/ic_launcher_background" />
    <foreground android:drawable="@mipmap/ic_launcher_foreground" />
    <monochrome android:drawable="@drawable/ic_launcher_monochrome" />
</adaptive-icon>
`)
		}},
		{path.Join(res, "values/ic_launcher_background.xml"), func(a Art) []byte {
			p := a.Plate()
			return fmt.Appendf(nil, `<?xml version="1.0" encoding="utf-8"?>
<!-- The icon artwork's plate colour. Generated from %s by `+"`make icons`"+`. -->
<resources>
    <color name="ic_launcher_background">#%02X%02X%02X</color>
</resources>
`, artPath, p.R, p.G, p.B)
		}},
	}
}
