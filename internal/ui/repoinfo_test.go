package ui

import (
	"testing"
	"time"
)

func TestShortAge(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{10 * time.Second, "now"},
		{5 * time.Minute, "5m"},
		{3 * time.Hour, "3h"},
		{12 * 24 * time.Hour, "12d"},
		{120 * 24 * time.Hour, "4mo"},
		{800 * 24 * time.Hour, "2y"},
	}
	for _, c := range cases {
		if got := shortAge(now.Add(-c.ago), now); got != c.want {
			t.Errorf("shortAge(%v ago) = %q, want %q", c.ago, got, c.want)
		}
	}
	if got := shortAge(time.Time{}, now); got != "" {
		t.Errorf("shortAge(zero) = %q, want empty", got)
	}
}
