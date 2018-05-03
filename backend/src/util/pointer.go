package util

import (
	"fmt"
	"time"
)

func StringPointer(s string) *string {
	return &s
}

func BoolPointer(b bool) *bool {
	return &b
}

func TimePointer(t time.Time) *time.Time {
	return &t
}

func Int64Pointer(i int64) *int64 {
	return &i
}

func StringNilOrValue(s *string) string {
	if s == nil {
		return "<nil>"
	} else {
		return *s
	}
}

func Int64NilOrValue(i *int64) string {
	if i == nil {
		return "<nil>"
	} else {
		return fmt.Sprintf("%v", *i)
	}
}

func BoolNilOrValue(b *bool) string {
	if b == nil {
		return "<nil>"
	} else {
		return fmt.Sprintf("%v", *b)
	}
}
