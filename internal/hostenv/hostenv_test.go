package hostenv

import (
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func keepLocal(t *testing.T) {
	t.Helper()
	loc := time.Local
	t.Cleanup(func() { time.Local = loc })
}

func TestUnsetChangesNothing(t *testing.T) {
	keepLocal(t)
	before := time.Local
	msg, err := Apply(env(nil))
	if err != nil || msg != "" {
		t.Fatalf("Apply() = %q, %v; want nothing", msg, err)
	}
	if time.Local != before {
		t.Error("time.Local changed with HOMEAPI_TZ unset")
	}
}

func TestZoneBecomesLocal(t *testing.T) {
	keepLocal(t)
	msg, err := Apply(env(map[string]string{EnvTZ: " Asia/Kolkata "}))
	if err != nil {
		t.Fatal(err)
	}
	if time.Local.String() != "Asia/Kolkata" || msg != "time zone Asia/Kolkata" {
		t.Errorf("time.Local = %v, msg %q", time.Local, msg)
	}
	// 12:00 UTC is 17:30 in India, which is what the display would draw.
	noon := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	if got := noon.Local().Format("15:04"); got != "17:30" {
		t.Errorf("noon UTC in local time = %s, want 17:30", got)
	}
}

func TestBadZoneIsAnErrorAndChangesNothing(t *testing.T) {
	keepLocal(t)
	before := time.Local
	if _, err := Apply(env(map[string]string{EnvTZ: "Mars/Olympus_Mons"})); err == nil {
		t.Error("an unknown zone was accepted")
	}
	if time.Local != before {
		t.Error("time.Local changed on a bad zone")
	}
}
