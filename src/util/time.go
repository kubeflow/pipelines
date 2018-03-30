package util

import (
	"time"

	"github.com/golang/glog"
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
	return time.Now().UTC()
}

type FakeTime struct {
	now time.Time
}

func NewFakeTime(now time.Time) TimeInterface {
	return &FakeTime{
		now: now.UTC(),
	}
}

func NewFakeTimeForEpoch() TimeInterface {
	return &FakeTime{
		now: time.Unix(0, 0).UTC(),
	}
}

func (f *FakeTime) Now() time.Time {
	f.now = time.Unix(f.now.Unix()+1, 0).UTC()
	return f.now
}

func ParseTimeOrFatal(value string) time.Time {
	result, err := time.Parse(time.RFC3339, value)
	if err != nil {
		glog.Fatalf("Could not parse time: %+v", err)
	}
	return result.UTC()
}
