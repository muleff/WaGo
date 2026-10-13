package wa

import (
	"testing"
	"time"
)

func TestLastSeenAtMonthBoundaryAndInLocalTimezone(t *testing.T) {
	loc := time.FixedZone("Honduras", -6*3600)
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, loc)
	seen := time.Date(2026, 10, 31, 20, 30, 0, 0, loc)
	if got := lastSeen(seen, now); got != "yesterday at 20:30" {
		t.Fatalf("month boundary: %q", got)
	}
	seen = time.Date(2026, 11, 1, 2, 30, 0, 0, time.UTC)
	if got := lastSeen(seen, now); got != "yesterday at 20:30" {
		t.Fatalf("local timezone: %q", got)
	}
}
