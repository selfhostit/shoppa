package app

import (
	"testing"
	"time"
)

func TestWaitFor(t *testing.T) {
	first := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		since, want time.Duration
	}{
		{0, 2 * time.Minute},               // the first item waits for more
		{2 * time.Minute, 2 * time.Minute}, // each one waits a quiet time again
		{4 * time.Minute, time.Minute},     // but five minutes is the most
		{6 * time.Minute, 0},
	} {
		if got := waitFor(first, first.Add(c.since)); got != c.want {
			t.Errorf("%v after the first: wait %v, want %v", c.since, got, c.want)
		}
	}
}
