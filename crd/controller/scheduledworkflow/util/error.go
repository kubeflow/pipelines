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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsNotFound returns whether an error indicates that a resource was "not found".
func IsNotFound(err error) bool {
	return reasonForError(err) == metav1.StatusReasonNotFound
}

// ReasonForError returns the HTTP status for a particular error.
func reasonForError(err error) metav1.StatusReason {
	switch t := err.(type) {
	case errors.APIStatus:
		return t.Status().Reason
	case *errors.StatusError:
		return t.Status().Reason
	}
	return metav1.StatusReasonUnknown
}
