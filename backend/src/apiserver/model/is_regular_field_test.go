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

package model

import "testing"

// IsRegularField reports whether a string is a real column on the model
// (a value in APIToModelFieldMap), as opposed to a string that merely looks
// like a SQL identifier. Each type's implementation is a one-line delegation
// to its own field map, so a single known-field/unknown-field/metric-alias
// case per type is enough to pin down the behavior.
func TestIsRegularField(t *testing.T) {
	tests := []struct {
		name      string
		isRegular func(string) bool
		knownGood string
	}{
		{"Run", (&Run{}).IsRegularField, "CreatedAtInSec"},
		{"Pipeline", (&Pipeline{}).IsRegularField, "Name"},
		{"Job", (&Job{}).IsRegularField, "DisplayName"},
		{"Experiment", (&Experiment{}).IsRegularField, "Name"},
		{"PipelineVersion", (&PipelineVersion{}).IsRegularField, "DisplayName"},
		{"Task", (Task{}).IsRegularField, "Name"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.isRegular(tc.knownGood) {
				t.Errorf("IsRegularField(%q) = false, want true", tc.knownGood)
			}
			if tc.isRegular("not_a_real_field") {
				t.Errorf("IsRegularField(%q) = true, want false", "not_a_real_field")
			}
			if tc.isRegular(MetricSortSQLAlias) {
				t.Errorf("IsRegularField(%q) = true, want false (alias is never a real field on any model)", MetricSortSQLAlias)
			}
			if tc.isRegular("") {
				t.Errorf("IsRegularField(%q) = true, want false", "")
			}
		})
	}
}
