package util

import (
	"time"
)

type TimeInterface interface {
	Now() time.Time
}

type RealTime struct {
}

func NewRealTime() TimeInterface {
	return &RealTime{}
}

func (r *RealTime) Now() time.Time {
	return time.Now()
}

type FakeTime struct {
	now time.Time
}

func NewFakeTime(now time.Time) TimeInterface {
	return &FakeTime{
		now: now,
	}
}

func NewFakeTimeForEpoch() TimeInterface {
	return &FakeTime{
		now: time.Unix(0, 0),
	}
}

func (f *FakeTime) Now() time.Time {
	f.now = time.Unix(f.now.Unix() + 1, 0)
	return f.now
}