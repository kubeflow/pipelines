// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/golang/protobuf/ptypes/timestamp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TimestampPointer(t timestamp.Timestamp) *timestamp.Timestamp {
	return &t
}

func DateTimePointer(t strfmt.DateTime) *strfmt.DateTime {
	return &t
}

func MetaV1TimePointer(t metav1.Time) *metav1.Time {
	return &t
}

func Int64Pointer(i int64) *int64 {
	return &i
}

func UInt32Pointer(i uint32) *uint32 {
	return &i
}

func Int32Pointer(i int32) *int32 {
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

// BooleanPointer converts a bool to a bool pointer.
func BooleanPointer(b bool) *bool {
	return &b
}

// Metav1TimePointer converts a metav1.Time to a pointer.
func Metav1TimePointer(t metav1.Time) *metav1.Time {
	return &t
}

func ToInt64Pointer(t *metav1.Time) *int64 {
	if t == nil {
		return nil
	} else {
		return Int64Pointer(t.Unix())
	}
}
