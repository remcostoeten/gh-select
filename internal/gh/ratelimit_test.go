package gh

import (
	"testing"
	"time"
)

func TestRateBucketPace(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	bucket := func(used, limit int, resetIn time.Duration) RateBucket {
		return RateBucket{Limit: limit, Used: used, Remaining: limit - used, Reset: now.Add(resetIn), Window: time.Minute}
	}

	if got := bucket(0, 30, time.Minute).Pace(now); got != "unused" {
		t.Errorf("unused pace = %q", got)
	}
	if got := bucket(30, 30, 20*time.Second).Pace(now); got != "exhausted, resets in 20s" {
		t.Errorf("exhausted pace = %q", got)
	}

	burning := bucket(20, 30, 50*time.Second)
	if eta, soon := burning.Exhaustion(now); !soon || eta != 5*time.Second {
		t.Errorf("20 used in 10s: eta %v soon %v, want 5s true", eta, soon)
	}
	if got := burning.Pace(now); got != "dry in ~5s at this pace, resets in 50s" {
		t.Errorf("burning pace = %q", got)
	}

	steady := bucket(5, 30, 10*time.Second)
	if _, soon := steady.Exhaustion(now); soon {
		t.Error("5 used in 50s should not run dry before reset")
	}
	if got := steady.Elapsed(now); got != 50*time.Second {
		t.Errorf("elapsed = %v, want 50s", got)
	}
}

func TestUsageBar(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	b := RateBucket{Limit: 10, Used: 5, Remaining: 5, Reset: now.Add(15 * time.Second), Window: time.Minute}
	if got := b.UsageBar(10, now); got != "█████░░│░░" {
		t.Errorf("UsageBar = %q", got)
	}
	if got := (RateBucket{Limit: 10, Window: time.Minute}).UsageBar(4, now); got != "░░░░" {
		t.Errorf("unused bar = %q", got)
	}
}

func TestShortDuration(t *testing.T) {
	cases := map[time.Duration]string{
		-time.Second:                    "0s",
		42 * time.Second:                "42s",
		time.Minute:                     "1m",
		7*time.Minute + 5*time.Second:   "7m05s",
		time.Hour:                       "1h",
		time.Hour + 5*time.Minute + 9e9: "1h05m",
	}
	for d, want := range cases {
		if got := ShortDuration(d); got != want {
			t.Errorf("ShortDuration(%v) = %q, want %q", d, got, want)
		}
	}
}
