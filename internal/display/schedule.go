package display

import (
	"fmt"
	"math"
	"time"
)

// Sleep bounds and the fallback when no wake times are configured, in
// seconds. The device clamps nothing itself; what the header says is what it
// sleeps.
const (
	MinSleepSeconds     = 60
	MaxSleepSeconds     = 86400
	DefaultSleepSeconds = 43200
)

// NextWake computes when the device should wake next: the first of
// s.WakeTimes after now, as a wall-clock time in s.TZ. Building each candidate
// with time.Date in the loaded location makes DST transitions correct by
// construction — a 07:15 that does not exist on the spring-forward morning
// normalises forward, and the sleep is the real elapsed time either way.
//
// sleep is the whole seconds until next, clamped to [MinSleepSeconds,
// MaxSleepSeconds]; when the clamp bites, next is moved to agree with it so
// the two headers never disagree. An empty wake list sleeps
// DefaultSleepSeconds. No jitter is added: the device's RC clock already
// drifts a few percent over twelve hours and the server re-anchors it every
// wake.
func NextWake(s Schedule, now time.Time) (next time.Time, sleep int, err error) {
	loc, err := time.LoadLocation(s.TZ)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("schedule.tz: %w", err)
	}
	if len(s.WakeTimes) == 0 {
		return now.Add(DefaultSleepSeconds * time.Second).In(loc), DefaultSleepSeconds, nil
	}

	local := now.In(loc)
	var best time.Time
	for dayOffset := 0; dayOffset <= 2 && best.IsZero(); dayOffset++ {
		day := local.AddDate(0, 0, dayOffset)
		for _, wt := range s.WakeTimes {
			h, m, err := parseHHMM(wt)
			if err != nil {
				return time.Time{}, 0, fmt.Errorf("schedule.wake_times: %w", err)
			}
			cand := time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, loc)
			if !cand.After(now) {
				continue
			}
			if best.IsZero() || cand.Before(best) {
				best = cand
			}
		}
	}
	if best.IsZero() {
		// Unreachable with a non-empty list: tomorrow always has a candidate.
		return now.Add(DefaultSleepSeconds * time.Second).In(loc), DefaultSleepSeconds, nil
	}

	sleep = int(math.Ceil(best.Sub(now).Seconds()))
	switch {
	case sleep < MinSleepSeconds:
		sleep = MinSleepSeconds
		best = now.Add(MinSleepSeconds * time.Second)
	case sleep > MaxSleepSeconds:
		sleep = MaxSleepSeconds
		best = now.Add(MaxSleepSeconds * time.Second)
	}
	return best.In(loc), sleep, nil
}
