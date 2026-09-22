// Copyright 2026 The Kubeflow Authors
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

func TestRunToV2_UsesConditionsWhenStateIsUnspecified(t *testing.T) {
	run := &Run{
		RunDetails: RunDetails{
			State:      RuntimeStateUnspecified,
			Conditions: string(RuntimeStateRunningV1),
		},
	}

	converted := run.ToV2()
	if converted.State != RuntimeStateRunning {
		t.Fatalf("expected state %q, got %q", RuntimeStateRunning, converted.State)
	}
}

func TestStoredOwnershipReferencesOnlyFillMissingV2Fields(t *testing.T) {
	for _, direct := range []string{"", "v2-owner"} {
		run := (&Run{ExperimentId: direct, Namespace: direct, RecurringRunId: direct,
			ResourceReferences: []*ResourceReference{
				{ReferenceType: ExperimentResourceType, ReferenceUUID: "historical-owner"},
				{ReferenceType: NamespaceResourceType, ReferenceUUID: "historical-owner"},
				{ReferenceType: JobResourceType, ReferenceUUID: "historical-owner"},
			},
		}).ToV2()
		want := direct
		if want == "" {
			want = "historical-owner"
		}
		if run.ExperimentId != want || run.Namespace != want || run.RecurringRunId != want || run.ResourceReferences != nil {
			t.Fatalf("unexpected normalized run ownership: %+v", run)
		}
		job := (&Job{ExperimentId: direct, Namespace: direct,
			PipelineSpec: PipelineSpec{PipelineId: direct, PipelineVersionId: direct},
			ResourceReferences: []*ResourceReference{
				{ReferenceType: ExperimentResourceType, ReferenceUUID: "historical-owner"},
				{ReferenceType: NamespaceResourceType, ReferenceUUID: "historical-owner"},
				{ReferenceType: PipelineResourceType, ReferenceUUID: "historical-owner"},
				{ReferenceType: PipelineVersionResourceType, ReferenceUUID: "historical-owner"},
			},
		}).ToV2()
		if job.ExperimentId != want || job.Namespace != want || job.PipelineId != want || job.PipelineVersionId != want || job.ResourceReferences != nil {
			t.Fatalf("unexpected normalized job ownership: %+v", job)
		}
	}
}
