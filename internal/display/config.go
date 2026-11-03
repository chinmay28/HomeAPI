package display

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Entry keys and categories the display reads and writes. Everything goes
// through the ordinary entries store; nothing new in the database.
const (
	ParamsKey     = "minion-params"       // the user's configuration (category "minion")
	QuotesKey     = "minion-quotes"       // symbol → price, written by the quote fetcher
	PrevQuotesKey = "minion3-prev-quotes" // previous quotes snapshot, written here
	LastSeenKey   = "minion3-last-seen"   // device telemetry, written here
	Category      = "minion3"             // category for the entries this feature writes
)

// Slot configures a region that shows one symbol: the hero, the aux and the
// four tiles. Decimals is a pointer so "not set" can fall back to the slot's
// own default rather than to zero.
type Slot struct {
	Key      string `json:"key,omitempty"`
	Morning  string `json:"morning,omitempty"` // aux only: symbol before 12:00 local
	Evening  string `json:"evening,omitempty"` // aux only: symbol from 12:00 local
	Label    string `json:"label,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Decimals *int   `json:"decimals,omitempty"`
}

// RatioSlot configures one position in the ratio strip: num ÷ den.
type RatioSlot struct {
	Num      string `json:"num"`
	Den      string `json:"den"`
	Label    string `json:"label,omitempty"`
	Decimals *int   `json:"decimals,omitempty"`
}

// SumSlot names the entry whose text fills the footer's free text slot.
type SumSlot struct {
	Key string `json:"key"`
}

// ClockSlot sets the footer clock's zone and Go time layout.
type ClockSlot struct {
	TZ     string `json:"tz"`
	Format string `json:"format"`
}

// WakeSlot sets the label in front of the device's wake counter.
type WakeSlot struct {
	Label string `json:"label"`
}

// Schedule drives X-Minion-Sleep-Seconds: local HH:MM wake times in TZ.
type Schedule struct {
	TZ        string   `json:"tz"`
	WakeTimes []string `json:"wake_times"`
}

// Pills sets the |Δ%| below which a tile shows no change pill.
type Pills struct {
	ThresholdPct float64 `json:"threshold_pct"`
}

// Stale sets how old quotes may be before the clock gives way to a stale
// marker. Zero disables the check.
type Stale struct {
	AfterMinutes int `json:"after_minutes"`
}

// Config is the parsed `minion-params` entry: one named parameter object per
// slot on the panel. Slot pointers are nil when the object is absent, and an
// absent slot renders empty. The non-slot objects (clock, wake, schedule,
// pills, stale) always carry a value; fields a configuration leaves out keep
// the built-in default.
type Config struct {
	Hero *Slot `json:"hero,omitempty"`
	Aux  *Slot `json:"aux,omitempty"`

	Tile1 *Slot `json:"tile1,omitempty"`
	Tile2 *Slot `json:"tile2,omitempty"`
	Tile3 *Slot `json:"tile3,omitempty"`
	Tile4 *Slot `json:"tile4,omitempty"`

	Ratio1 *RatioSlot `json:"ratio1,omitempty"`
	Ratio2 *RatioSlot `json:"ratio2,omitempty"`
	Ratio3 *RatioSlot `json:"ratio3,omitempty"`

	Sum *SumSlot `json:"sum,omitempty"`

	Clock    ClockSlot `json:"clock"`
	Wake     WakeSlot  `json:"wake"`
	Schedule Schedule  `json:"schedule"`
	Pills    Pills     `json:"pills"`
	Stale    Stale     `json:"stale"`
}

// Tiles returns the four tile slots in panel order (left to right, top to
// bottom). Missing slots are nil.
func (c *Config) Tiles() [4]*Slot {
	return [4]*Slot{c.Tile1, c.Tile2, c.Tile3, c.Tile4}
}

// Ratios returns the three ratio positions in order. Missing slots are nil.
func (c *Config) Ratios() [3]*RatioSlot {
	return [3]*RatioSlot{c.Ratio1, c.Ratio2, c.Ratio3}
}

// intp is a small helper for literal optional integers.
func intp(n int) *int { return &n }

// Defaults for the per-slot decimals when a slot leaves them out.
const (
	defaultHeroDecimals  = 0
	defaultTileDecimals  = 2
	defaultRatioDecimals = 2
	defaultThresholdPct  = 0.05
	defaultStaleMinutes  = 180
	defaultTZ            = "America/Los_Angeles"
	defaultClockFormat   = "Mon 01/02  15:04"
	defaultWakeLabel     = "Wake"
)

// DefaultConfig reproduces today's Minion content and is what a missing or
// invalid `minion-params` entry falls back to.
func DefaultConfig() Config {
	return Config{
		Hero: &Slot{Key: "BTC-USD", Label: "BTC", Prefix: "$", Decimals: intp(0)},
		Aux:  &Slot{Morning: "IBIT", Evening: "STRC", Prefix: "$", Decimals: intp(2)},

		Tile1: &Slot{Key: "VTI", Label: "VTI", Prefix: "$", Decimals: intp(2)},
		Tile2: &Slot{Key: "GLD", Label: "GLD", Prefix: "$", Decimals: intp(2)},
		Tile3: &Slot{Key: "P", Label: "PSTG", Prefix: "$", Decimals: intp(2)},
		Tile4: &Slot{Key: "ORCL", Label: "ORCL", Prefix: "$", Decimals: intp(2)},

		Ratio1: &RatioSlot{Num: "VTI", Den: "GLD", Decimals: intp(2)},
		Ratio2: &RatioSlot{Num: "P", Den: "VTI", Label: "PSTG/VTI", Decimals: intp(2)},
		Ratio3: &RatioSlot{Num: "ORCL", Den: "VTI", Decimals: intp(2)},

		Sum:   &SumSlot{Key: "minion-sum"},
		Clock: ClockSlot{TZ: defaultTZ, Format: defaultClockFormat},
		Wake:  WakeSlot{Label: defaultWakeLabel},

		Schedule: Schedule{TZ: defaultTZ, WakeTimes: []string{"07:15", "19:15"}},
		Pills:    Pills{ThresholdPct: defaultThresholdPct},
		Stale:    Stale{AfterMinutes: defaultStaleMinutes},
	}
}

// baseConfig is what a user configuration is decoded over: no slots, but every
// non-slot object at its default so a configuration that only names tiles
// still has a clock, a schedule and a wake label.
func baseConfig() Config {
	d := DefaultConfig()
	return Config{
		Clock:    d.Clock,
		Wake:     d.Wake,
		Schedule: Schedule{TZ: d.Schedule.TZ},
		Pills:    d.Pills,
		Stale:    d.Stale,
	}
}

// ParseConfig decodes a `minion-params` value. On any failure — malformed
// JSON, a zone that does not load, a wake time that is not HH:MM — it returns
// DefaultConfig together with the error, so a caller can always render and
// can tell that it is rendering the fallback.
func ParseConfig(data []byte) (Config, error) {
	cfg := baseConfig()
	if len(strings.TrimSpace(string(data))) == 0 {
		return DefaultConfig(), errors.New("minion-params is empty")
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("minion-params is not valid JSON: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return DefaultConfig(), fmt.Errorf("minion-params: %w", err)
	}
	return cfg, nil
}

// Validate checks everything a render would otherwise trip over: zones load,
// wake times parse, decimals and thresholds are sane. It does not check that
// symbols exist — a symbol missing from the quotes shows N/A by design.
func (c *Config) Validate() error {
	if c.Clock.Format == "" {
		return errors.New("clock.format must not be empty")
	}
	if _, err := time.LoadLocation(c.Clock.TZ); err != nil {
		return fmt.Errorf("clock.tz: %w", err)
	}
	if _, err := time.LoadLocation(c.Schedule.TZ); err != nil {
		return fmt.Errorf("schedule.tz: %w", err)
	}
	for _, wt := range c.Schedule.WakeTimes {
		if _, _, err := parseHHMM(wt); err != nil {
			return fmt.Errorf("schedule.wake_times: %w", err)
		}
	}
	if c.Pills.ThresholdPct < 0 {
		return errors.New("pills.threshold_pct must be >= 0")
	}
	if c.Stale.AfterMinutes < 0 {
		return errors.New("stale.after_minutes must be >= 0")
	}
	check := func(name string, d *int) error {
		if d != nil && (*d < 0 || *d > 8) {
			return fmt.Errorf("%s.decimals must be between 0 and 8", name)
		}
		return nil
	}
	slots := map[string]*Slot{"hero": c.Hero, "aux": c.Aux, "tile1": c.Tile1, "tile2": c.Tile2, "tile3": c.Tile3, "tile4": c.Tile4}
	for name, s := range slots {
		if s == nil {
			continue
		}
		if err := check(name, s.Decimals); err != nil {
			return err
		}
	}
	ratios := map[string]*RatioSlot{"ratio1": c.Ratio1, "ratio2": c.Ratio2, "ratio3": c.Ratio3}
	for name, r := range ratios {
		if r == nil {
			continue
		}
		if err := check(name, r.Decimals); err != nil {
			return err
		}
	}
	return nil
}

// parseHHMM parses a local wall-clock time of the form HH:MM.
func parseHHMM(s string) (hour, minute int, err error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("%q is not HH:MM", s)
	}
	hour, err1 := strconv.Atoi(parts[0])
	minute, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("%q is not HH:MM", s)
	}
	return hour, minute, nil
}

// ── Inputs ──────────────────────────────────────────────────────────────────

// Quotes is a decoded `minion-quotes` entry: symbol → price string, plus the
// fetcher's timestamp. TimestampText is the raw timestamp, which is what the
// previous-quotes snapshot is compared by.
type Quotes struct {
	Prices        map[string]string
	Timestamp     time.Time // zero when absent or unparseable
	TimestampText string
}

// Price returns the price string for a symbol and whether it is present.
func (q Quotes) Price(symbol string) (string, bool) {
	if q.Prices == nil {
		return "", false
	}
	p, ok := q.Prices[symbol]
	return p, ok
}

// quoteTimestampLayouts are the timestamp forms the fetcher is known to
// write. The first has no year: it is resolved against now (see ParseQuotes).
var quoteTimestampLayouts = []string{"01/02 15:04:05", "01/02 15:04", time.RFC3339}

// ParseQuotes decodes a `minion-quotes` value. It never fails: anything that
// is not a JSON object yields empty quotes, and a timestamp that does not
// parse yields a zero Timestamp (which the stale check treats as unknown
// age). A timestamp without a year is read in loc and assigned now's year,
// or the previous year when that would put it more than a day in the future.
func ParseQuotes(raw string, now time.Time, loc *time.Location) Quotes {
	q := Quotes{Prices: map[string]string{}}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return q
	}
	for k, v := range obj {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			var n json.Number
			if err := json.Unmarshal(v, &n); err != nil {
				continue // nested objects, bools and nulls are not prices
			}
			s = n.String()
		}
		if k == "timestamp" {
			q.TimestampText = s
			q.Timestamp = parseQuoteTimestamp(s, now, loc)
			continue
		}
		q.Prices[k] = s
	}
	return q
}

func parseQuoteTimestamp(s string, now time.Time, loc *time.Location) time.Time {
	s = strings.TrimSpace(s)
	if loc == nil {
		loc = time.UTC
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	for _, layout := range quoteTimestampLayouts[:2] {
		t, err := time.ParseInLocation(layout, s, loc)
		if err != nil {
			continue
		}
		local := now.In(loc)
		t = time.Date(local.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
		if t.Sub(now) > 24*time.Hour {
			t = t.AddDate(-1, 0, 0)
		}
		return t
	}
	return time.Time{}
}

// Inputs is everything besides the configuration and the clock that a render
// depends on. Every field may be left at its zero value; the picture degrades
// rather than the call failing.
type Inputs struct {
	Quotes Quotes // current quotes
	Prev   Quotes // snapshot from the previous wake, for the change pills
	Sum    string // the text of the entry named by sum.key, verbatim

	Wake    int  // the device's wake counter
	HasWake bool // false when the device did not report one

	// ConfigInvalid marks that cfg is the built-in fallback because the stored
	// `minion-params` was missing or unparseable; the footer then shows
	// "Cfg?" where the sum would be so the problem is visible on the panel.
	ConfigInvalid bool
}

// Staleness reports whether the quotes are older than cfg.Stale allows and
// the label the footer shows in place of the clock when they are. Quotes with
// no usable timestamp count as stale with unknown age, so a feed that has
// stopped is visible on the panel. A zero after_minutes disables the check.
func Staleness(cfg Config, q Quotes, now time.Time) (stale bool, label string) {
	if cfg.Stale.AfterMinutes <= 0 {
		return false, ""
	}
	if q.Timestamp.IsZero() {
		return true, "Stale"
	}
	age := now.Sub(q.Timestamp)
	if age <= time.Duration(cfg.Stale.AfterMinutes)*time.Minute {
		return false, ""
	}
	switch {
	case age < time.Hour:
		return true, fmt.Sprintf("Stale %dm", int(age.Minutes()))
	case age < 48*time.Hour:
		return true, fmt.Sprintf("Stale %dh", int(age.Hours()))
	default:
		return true, fmt.Sprintf("Stale %dd", int(age.Hours()/24))
	}
}
