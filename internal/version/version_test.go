package version

import (
	"regexp"
	"testing"
)

func TestStringFormat(t *testing.T) {
	// scripts/version.mjs builds the same string for the web bundle and must
	// stay byte-identical: `v` + year.month.patch, no suffixes.
	got := String()
	if !regexp.MustCompile(`^v\d+\.\d+\.\d+$`).MatchString(got) {
		t.Errorf("String() = %q, want vYEAR.MONTH.PATCH", got)
	}
}

func TestStringUsesStampedPatch(t *testing.T) {
	orig := Patch
	t.Cleanup(func() { Patch = orig })

	Patch = "311"
	if got, want := String(), "v2026.8.311"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestUnstampedBuildIsPatchZero(t *testing.T) {
	// A bare `go build` leaves Patch alone; patch 0 is the agreed marker for a
	// build made without git, never a release.
	if Patch != "0" {
		t.Skip("binary was built with a stamped patch number")
	}
	if got, want := String(), "v2026.8.0"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestMonthIsACalendarMonth(t *testing.T) {
	// The month goes into a version string that has to stay valid semver and
	// has to name a real month; scripts/version.mjs refuses to assemble one out
	// of anything else, so a typo should fail here first.
	if Month < 1 || Month > 12 {
		t.Errorf("Month = %d, want a calendar month (1-12)", Month)
	}
}
