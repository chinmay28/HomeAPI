package display

import (
	"testing"
	"time"
)

func TestNextWake_DST(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	twice := Schedule{TZ: "America/Los_Angeles", WakeTimes: []string{"07:15", "19:15"}}
	once := Schedule{TZ: "America/Los_Angeles", WakeTimes: []string{"07:15"}}

	tests := []struct {
		name      string
		s         Schedule
		now       time.Time
		wantNext  time.Time
		wantSleep int
	}{
		{
			name:      "ordinary evening to morning",
			s:         twice,
			now:       time.Date(2026, 11, 3, 19, 15, 0, 0, la),
			wantNext:  time.Date(2026, 11, 4, 7, 15, 0, 0, la),
			wantSleep: 12 * 3600,
		},
		{
			name:      "ordinary morning to evening",
			s:         twice,
			now:       time.Date(2026, 11, 3, 7, 15, 0, 0, la),
			wantNext:  time.Date(2026, 11, 3, 19, 15, 0, 0, la),
			wantSleep: 12 * 3600,
		},
		{
			name:      "spring forward night is an hour short",
			s:         twice,
			now:       time.Date(2026, 3, 7, 19, 15, 0, 0, la), // PST
			wantNext:  time.Date(2026, 3, 8, 7, 15, 0, 0, la),  // PDT
			wantSleep: 11 * 3600,
		},
		{
			name:      "fall back night is an hour long",
			s:         twice,
			now:       time.Date(2026, 10, 31, 19, 15, 0, 0, la), // PDT
			wantNext:  time.Date(2026, 11, 1, 7, 15, 0, 0, la),   // PST
			wantSleep: 13 * 3600,
		},
		{
			name:      "unordered list still picks the soonest",
			s:         Schedule{TZ: "America/Los_Angeles", WakeTimes: []string{"19:15", "07:15", "12:00"}},
			now:       time.Date(2026, 11, 3, 8, 0, 0, 0, la),
			wantNext:  time.Date(2026, 11, 3, 12, 0, 0, 0, la),
			wantSleep: 4 * 3600,
		},
		{
			name:      "fractional seconds round up",
			s:         twice,
			now:       time.Date(2026, 11, 3, 7, 14, 0, 500_000_000, la),
			wantNext:  time.Date(2026, 11, 3, 7, 15, 0, 0, la),
			wantSleep: 60,
		},
		{
			name:      "clamped to the minimum",
			s:         twice,
			now:       time.Date(2026, 11, 3, 7, 14, 59, 0, la),
			wantNext:  time.Date(2026, 11, 3, 7, 15, 59, 0, la),
			wantSleep: MinSleepSeconds,
		},
		{
			name:      "clamped to the maximum across fall back",
			s:         once,
			now:       time.Date(2026, 10, 31, 7, 15, 0, 0, la), // next 07:15 is 25 h away
			wantNext:  time.Date(2026, 11, 1, 6, 15, 0, 0, la),
			wantSleep: MaxSleepSeconds,
		},
		{
			name:      "now in another zone is converted",
			s:         twice,
			now:       time.Date(2026, 11, 4, 3, 15, 0, 0, time.UTC), // 19:15 PST the day before
			wantNext:  time.Date(2026, 11, 4, 7, 15, 0, 0, la),
			wantSleep: 12 * 3600,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, sleep, err := NextWake(tt.s, tt.now)
			if err != nil {
				t.Fatalf("NextWake: %v", err)
			}
			if !next.Equal(tt.wantNext) {
				t.Errorf("next = %v, want %v", next, tt.wantNext)
			}
			if sleep != tt.wantSleep {
				t.Errorf("sleep = %d, want %d", sleep, tt.wantSleep)
			}
			if next.Location().String() != la.String() {
				t.Errorf("next is in %v, want the schedule's zone", next.Location())
			}
		})
	}
}

func TestNextWake_EmptyList(t *testing.T) {
	now := time.Date(2026, 11, 3, 7, 15, 0, 0, time.UTC)
	next, sleep, err := NextWake(Schedule{TZ: "UTC"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if sleep != DefaultSleepSeconds {
		t.Errorf("sleep = %d, want %d", sleep, DefaultSleepSeconds)
	}
	if !next.Equal(now.Add(12 * time.Hour)) {
		t.Errorf("next = %v, want now+12h", next)
	}
}

func TestNextWake_Errors(t *testing.T) {
	now := time.Now()
	if _, _, err := NextWake(Schedule{TZ: "Mars/Olympus", WakeTimes: []string{"07:15"}}, now); err == nil {
		t.Error("expected an error for an unknown zone")
	}
	if _, _, err := NextWake(Schedule{TZ: "UTC", WakeTimes: []string{"noon"}}, now); err == nil {
		t.Error("expected an error for a malformed wake time")
	}
}
