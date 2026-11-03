package display

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseConfig_Default(t *testing.T) {
	// The default must survive its own JSON round trip unchanged and be valid.
	d := DefaultConfig()
	if err := d.Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("ParseConfig(default): %v", err)
	}
	again, _ := json.Marshal(back)
	if string(again) != string(data) {
		t.Errorf("round trip changed the default:\n%s\n%s", data, again)
	}
}

func TestParseConfig_FallsBack(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string // substring of the error
	}{
		{"empty", "", "empty"},
		{"whitespace", "  \n", "empty"},
		{"not json", "{nope", "not valid JSON"},
		{"bad clock tz", `{"clock":{"tz":"Mars/Olympus"}}`, "clock.tz"},
		{"bad schedule tz", `{"schedule":{"tz":"Nowhere"}}`, "schedule.tz"},
		{"bad wake time", `{"schedule":{"wake_times":["7am"]}}`, "wake_times"},
		{"wake hour out of range", `{"schedule":{"wake_times":["25:00"]}}`, "wake_times"},
		{"negative threshold", `{"pills":{"threshold_pct":-1}}`, "threshold"},
		{"negative stale", `{"stale":{"after_minutes":-5}}`, "after_minutes"},
		{"decimals too large", `{"hero":{"key":"X","decimals":9}}`, "decimals"},
		{"ratio decimals negative", `{"ratio1":{"num":"A","den":"B","decimals":-1}}`, "decimals"},
		{"empty clock format", `{"clock":{"format":""}}`, "clock.format"},
	}
	def, _ := json.Marshal(DefaultConfig())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig([]byte(tt.in))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want one containing %q", err, tt.want)
			}
			got, _ := json.Marshal(cfg)
			if string(got) != string(def) {
				t.Errorf("fallback is not the default config")
			}
		})
	}
}

func TestParseConfig_PartialKeepsDefaultsForNonSlots(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{"tile1":{"key":"VTI"},"clock":{"tz":"UTC"}}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Hero != nil || cfg.Aux != nil || cfg.Tile2 != nil || cfg.Ratio1 != nil || cfg.Sum != nil {
		t.Error("slots not named in the config must be nil")
	}
	if cfg.Tile1 == nil || cfg.Tile1.Key != "VTI" || cfg.Tile1.Decimals != nil {
		t.Errorf("tile1 = %+v", cfg.Tile1)
	}
	if cfg.Clock.TZ != "UTC" || cfg.Clock.Format != defaultClockFormat {
		t.Errorf("clock = %+v, want tz overridden and format defaulted", cfg.Clock)
	}
	if cfg.Wake.Label != defaultWakeLabel {
		t.Errorf("wake label = %q", cfg.Wake.Label)
	}
	if cfg.Pills.ThresholdPct != defaultThresholdPct || cfg.Stale.AfterMinutes != defaultStaleMinutes {
		t.Errorf("pills/stale not defaulted: %+v %+v", cfg.Pills, cfg.Stale)
	}
	if cfg.Schedule.TZ != defaultTZ || len(cfg.Schedule.WakeTimes) != 0 {
		t.Errorf("schedule = %+v, want default tz and no wake times", cfg.Schedule)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("partial config should validate: %v", err)
	}
}

func TestParseConfig_UnknownKeysIgnored(t *testing.T) {
	if _, err := ParseConfig([]byte(`{"banner":{"text":"hi"},"hero":{"key":"X"}}`)); err != nil {
		t.Errorf("unknown top-level key should be ignored, got %v", err)
	}
}

func TestParseQuotes(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2026, 11, 3, 7, 15, 0, 0, loc)

	t.Run("strings and numbers", func(t *testing.T) {
		q := ParseQuotes(`{"VTI":"301.20","GLD":214.05,"nested":{"a":1},"flag":true,"timestamp":"11/03 07:10:00"}`, now, loc)
		if got, _ := q.Price("VTI"); got != "301.20" {
			t.Errorf("VTI = %q", got)
		}
		if got, _ := q.Price("GLD"); got != "214.05" {
			t.Errorf("GLD = %q (numbers should keep their text)", got)
		}
		if _, ok := q.Price("nested"); ok {
			t.Error("objects are not prices")
		}
		if _, ok := q.Price("timestamp"); ok {
			t.Error("timestamp is not a price")
		}
		if q.TimestampText != "11/03 07:10:00" {
			t.Errorf("TimestampText = %q", q.TimestampText)
		}
		want := time.Date(2026, 11, 3, 7, 10, 0, 0, loc)
		if !q.Timestamp.Equal(want) {
			t.Errorf("Timestamp = %v, want %v", q.Timestamp, want)
		}
	})

	t.Run("year boundary", func(t *testing.T) {
		jan := time.Date(2027, 1, 1, 0, 30, 0, 0, loc)
		q := ParseQuotes(`{"timestamp":"12/31 23:50:00"}`, jan, loc)
		if q.Timestamp.Year() != 2026 {
			t.Errorf("a December timestamp read on New Year's Day belongs to last year, got %v", q.Timestamp)
		}
	})

	t.Run("rfc3339 timestamp", func(t *testing.T) {
		q := ParseQuotes(`{"timestamp":"2026-11-03T07:10:00-08:00"}`, now, loc)
		if !q.Timestamp.Equal(time.Date(2026, 11, 3, 7, 10, 0, 0, loc)) {
			t.Errorf("Timestamp = %v", q.Timestamp)
		}
	})

	t.Run("garbage", func(t *testing.T) {
		for _, raw := range []string{"", "not json", "[1,2]", `{"timestamp":"yesterday"}`} {
			q := ParseQuotes(raw, now, loc)
			if q.Prices == nil {
				t.Errorf("%q: Prices must never be nil", raw)
			}
			if !q.Timestamp.IsZero() {
				t.Errorf("%q: Timestamp should be zero", raw)
			}
		}
	})
}

func TestStaleness(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2026, 11, 3, 7, 15, 0, 0, loc)
	cfg := DefaultConfig() // 180 minutes
	at := func(age time.Duration) Quotes { return Quotes{Timestamp: now.Add(-age)} }

	tests := []struct {
		name  string
		cfg   Config
		q     Quotes
		stale bool
		label string
	}{
		{"fresh", cfg, at(5 * time.Minute), false, ""},
		{"exactly at the limit is fresh", cfg, at(180 * time.Minute), false, ""},
		{"three hours", cfg, at(3*time.Hour + time.Minute), true, "Stale 3h"},
		{"forty-seven hours", cfg, at(47 * time.Hour), true, "Stale 47h"},
		{"days", cfg, at(49 * time.Hour), true, "Stale 2d"},
		{"no timestamp", cfg, Quotes{}, true, "Stale"},
		{"disabled", Config{Stale: Stale{AfterMinutes: 0}}, Quotes{}, false, ""},
		{"minutes", Config{Stale: Stale{AfterMinutes: 10}}, at(45 * time.Minute), true, "Stale 45m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stale, label := Staleness(tt.cfg, tt.q, now)
			if stale != tt.stale || label != tt.label {
				t.Errorf("Staleness = (%v, %q), want (%v, %q)", stale, label, tt.stale, tt.label)
			}
		})
	}
}
