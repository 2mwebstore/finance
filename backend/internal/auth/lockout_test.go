package auth

import (
	"testing"
	"time"
)

func TestLockoutAfterGrowsAndCaps(t *testing.T) {
	cases := map[int]time.Duration{
		1: 0, 4: 0, 6: 0, 9: 0,
		5:   15 * time.Minute,
		10:  30 * time.Minute,
		15:  time.Hour,
		20:  2 * time.Hour,
		35:  16 * time.Hour,
		40:  24 * time.Hour,
		500: 24 * time.Hour,
	}
	for failures, want := range cases {
		if got := lockoutAfter(failures); got != want {
			t.Errorf("lockoutAfter(%d) = %v, want %v", failures, got, want)
		}
	}
}
