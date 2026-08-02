package version

import (
	"regexp"
	"testing"
)

func TestStringFormat(t *testing.T) {
	// scripts/version.mjs builds the same string for the web bundle and must
	// stay byte-identical: `v` + major.minor.patch, no suffixes.
	got := String()
	if !regexp.MustCompile(`^v\d+\.\d+\.\d+$`).MatchString(got) {
		t.Errorf("String() = %q, want vMAJOR.MINOR.PATCH", got)
	}
}

func TestStringUsesStampedPatch(t *testing.T) {
	orig := Patch
	t.Cleanup(func() { Patch = orig })

	Patch = "311"
	if got, want := String(), "v1.0.311"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestUnstampedBuildIsPatchZero(t *testing.T) {
	// A bare `go build` leaves Patch alone; patch 0 is the agreed marker for a
	// build made without git, never a release.
	if Patch != "0" {
		t.Skip("binary was built with a stamped patch number")
	}
	if got, want := String(), "v1.0.0"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
