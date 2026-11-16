// Package hostenv adapts the process to a host that does not look like a Linux
// distribution — today, the HomeAPI Android app, which runs this same static
// binary on a phone.
//
// Go on Android always starts in UTC: there is no /etc/localtime to read. The
// Minion 3 display renders the time of day and schedules its sleep by the
// local clock, so on a phone it would be hours out. HOMEAPI_TZ names the zone
// (the app passes the phone's), and Apply installs it as time.Local.
//
// Unset, it does nothing, so a server anywhere else behaves exactly as before.
// HomeAPI makes no outbound connections, so unlike SAND Vault's hostenv there
// is no DNS piece: nothing here would ever look a name up.
//
// A zone name needs a zone database. A phone has none Go can read, so the
// Android build links Go's own in (`-tags timetzdata`).
package hostenv

import (
	"fmt"
	"strings"
	"time"
)

// EnvTZ is an IANA time zone name, e.g. "Asia/Kolkata".
const EnvTZ = "HOMEAPI_TZ"

// Apply reads the host settings from getenv (os.Getenv in production) and
// installs them. It returns a description of what it changed, for a log line
// ("" when nothing was set), and an error for a value it could not use — in
// which case time.Local is left as it was.
func Apply(getenv func(string) string) (string, error) {
	name := strings.TrimSpace(getenv(EnvTZ))
	if name == "" {
		return "", nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return "", fmt.Errorf("%s=%q: %w", EnvTZ, name, err)
	}
	time.Local = loc
	return "time zone " + loc.String(), nil
}
