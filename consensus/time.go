package consensus

import "time"

type TimerDriver interface {
	Now() time.Time
}

type Timer struct {
	TimerDriver
}

type FakeTimer struct {
	t time.Time
	TimerDriver
}

func (d *Timer) Now() time.Time {
	return time.Now();
}

func (d *FakeTimer) Now() time.Time {
	return d.t;
}

func (d *FakeTimer) SetTime(t time.Time) {
	d.t = t;
}