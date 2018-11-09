// Package mclock is a wrapper for a monotonic clock source
package mclock

import (
	"time"

	"github.com/aristanetworks/goarista/monotime"
)

// AbsTime represents absolute monotonic time.
type AbsTime time.Duration

// Now returns the current absolute monotonic time.
func Now() AbsTime {
	return AbsTime(monotime.Now())
}

// Add returns t + d.
func (t AbsTime) Add(d time.Duration) AbsTime {
	return t + AbsTime(d)
}

// Clock interface makes it possible to replace the monotonic system clock with
// a simulated clock.
type Clock interface {
	Now() AbsTime
	Sleep(time.Duration)
	After(time.Duration) <-chan time.Time
}

// System implements Clock using the system clock.
type System struct{}

// Now implements Clock.
func (System) Now() AbsTime {
	return AbsTime(monotime.Now())
}

// Sleep implements Clock.
func (System) Sleep(d time.Duration) {
	time.Sleep(d)
}

// After implements Clock.
func (System) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
