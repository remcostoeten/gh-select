package gh

import (
	"fmt"
	"strings"
	"time"
)

// RateBucket is one of the viewer's GitHub API budgets. Limits are per
// account, so other tools using the same token draw from them too.
type RateBucket struct {
	Name      string
	Limit     int
	Used      int
	Remaining int
	Reset     time.Time
	Window    time.Duration
}

// rateBuckets are the budgets gh-select draws from, in display order, with
// the length of each rolling window.
var rateBuckets = []struct {
	key    string
	window time.Duration
}{
	{"search", time.Minute},
	{"core", time.Hour},
	{"graphql", time.Hour},
	{"code_search", time.Minute},
}

// RateLimits fetches the viewer's current usage. The endpoint itself does not
// count against any limit.
func (c *Client) RateLimits() ([]RateBucket, error) {
	var resp struct {
		Resources map[string]struct {
			Limit     int   `json:"limit"`
			Used      int   `json:"used"`
			Remaining int   `json:"remaining"`
			Reset     int64 `json:"reset"`
		} `json:"resources"`
	}
	if err := c.rest.Get("rate_limit", &resp); err != nil {
		return nil, err
	}
	out := make([]RateBucket, 0, len(rateBuckets))
	for _, b := range rateBuckets {
		r, ok := resp.Resources[b.key]
		if !ok {
			continue
		}
		out = append(out, RateBucket{
			Name:      b.key,
			Limit:     r.Limit,
			Used:      r.Used,
			Remaining: r.Remaining,
			Reset:     time.Unix(r.Reset, 0),
			Window:    b.window,
		})
	}
	return out, nil
}

// Elapsed is how far into its current window the bucket is at now. A window
// starts at its first request, so an unused bucket reports zero.
func (b RateBucket) Elapsed(now time.Time) time.Duration {
	if b.Used == 0 {
		return 0
	}
	left := max(b.Reset.Sub(now), 0)
	return min(max(b.Window-left, 0), b.Window)
}

// Exhaustion estimates how long until the bucket runs dry at the pace used so
// far in this window. ok is false when the pace can't empty it before reset.
func (b RateBucket) Exhaustion(now time.Time) (time.Duration, bool) {
	if b.Remaining == 0 {
		return 0, true
	}
	elapsed := b.Elapsed(now)
	if b.Used == 0 || elapsed <= 0 {
		return 0, false
	}
	perReq := elapsed / time.Duration(b.Used)
	eta := perReq * time.Duration(b.Remaining)
	return eta, eta < max(b.Reset.Sub(now), 0)
}

// Pace summarises usage against time, e.g. "pace fine, resets in 12m" or
// "dry in ~4m at this pace, resets in 41m".
func (b RateBucket) Pace(now time.Time) string {
	if b.Remaining == 0 {
		return "exhausted, resets in " + ShortDuration(b.Reset.Sub(now))
	}
	if b.Used == 0 {
		return "unused"
	}
	resets := ", resets in " + ShortDuration(b.Reset.Sub(now))
	if eta, ok := b.Exhaustion(now); ok {
		return "dry in ~" + ShortDuration(eta) + " at this pace" + resets
	}
	return "pace fine" + resets
}

// ShortDuration renders d compactly: 42s, 7m, 7m05s, 1h, 1h05m.
func ShortDuration(d time.Duration) string {
	d = max(d, 0).Round(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour && d%time.Minute == 0:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	case d%time.Hour == 0:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

// UsageBar draws used/limit as a width-cell bar with a marker at how much of
// the window has passed, so usage running ahead of time stands out.
func (b RateBucket) UsageBar(width int, now time.Time) string {
	if width <= 0 || b.Limit <= 0 {
		return ""
	}
	filled := min(b.Used*width/b.Limit, width)
	if b.Used > 0 && filled == 0 {
		filled = 1
	}
	mark := -1
	if b.Used > 0 {
		mark = min(int(b.Elapsed(now)*time.Duration(width)/b.Window), width-1)
	}
	var s strings.Builder
	for i := range width {
		switch {
		case i == mark:
			s.WriteString("│")
		case i < filled:
			s.WriteString("█")
		default:
			s.WriteString("░")
		}
	}
	return s.String()
}
