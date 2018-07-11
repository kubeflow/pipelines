// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StringPointer converts a string to a string pointer.
func StringPointer(s string) *string {
	return &s
}

// BooleanPointer converts a bool to a bool pointer.
func BooleanPointer(b bool) *bool {
	return &b
}

// Metav1TimePointer converts a metav1.Time to a pointer.
func Metav1TimePointer(t metav1.Time) *metav1.Time {
	return &t
}

// Int64Pointer converts an int64 to a pointer.
func Int64Pointer(i int64) *int64 {
	return &i
}

func toInt64Pointer(t *metav1.Time) *int64 {
	if t == nil {
		return nil
	} else {
		return Int64Pointer(t.Unix())
	}
}
