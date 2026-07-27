package client

import (
	"testing"
	"time"
)

func TestIsTimeWithinRangeInclusive(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	finish := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	if !isTimeWithinRangeInclusive(start, start, finish) {
		t.Fatal("expected start boundary to be included")
	}
	if !isTimeWithinRangeInclusive(finish, start, finish) {
		t.Fatal("expected finish boundary to be included")
	}
	if isTimeWithinRangeInclusive(start.Add(-time.Nanosecond), start, finish) {
		t.Fatal("expected time before start to be excluded")
	}
	if isTimeWithinRangeInclusive(finish.Add(time.Nanosecond), start, finish) {
		t.Fatal("expected time after finish to be excluded")
	}
}
