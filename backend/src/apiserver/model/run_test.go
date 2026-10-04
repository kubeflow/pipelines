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

func TestHistoricalRuntimeStateMappings(t *testing.T) {
	for _, test := range []struct {
		stored RuntimeState
		v2     RuntimeState
		phase  RuntimeState
	}{
		{"Pending", RuntimeStatePending, RuntimeStatePendingV1},
		{"Running", RuntimeStateRunning, RuntimeStateRunningV1},
		{"Succeeded", RuntimeStateSucceeded, RuntimeStateSucceededV1},
		{"Skipped", RuntimeStateSkipped, RuntimeStateSkippedV1},
		{"Failed", RuntimeStateFailed, RuntimeStateFailedV1},
		{"Terminating", RuntimeStateCancelling, RuntimeStateTerminatingV1},
		{"Ready", RuntimeStateRunning, RuntimeStateRunningV1},
		{"Done", RuntimeStateSucceeded, RuntimeStateSucceededV1},
		{"Error", RuntimeStateFailed, RuntimeStateFailedV1},
		{"Unknown", RuntimeStateUnspecified, RuntimeStateUnknownV1},
		{"NO_STATUS", RuntimeStateUnspecified, RuntimeStateUnknownV1},
		{"Enabled", RuntimeStateRunning, RuntimeStateRunningV1},
		{"Disabled", RuntimeStateCanceled, RuntimeStateFailedV1},
		{"", RuntimeStateUnspecified, RuntimeStateUnknownV1},
		{"invalid", RuntimeStateUnspecified, RuntimeStateUnknownV1},
	} {
		t.Run(string(test.stored), func(t *testing.T) {
			if got := test.stored.ToV2(); got != test.v2 {
				t.Errorf("ToV2() = %q, want %q", got, test.v2)
			}
			if got := test.stored.ToExecutionPhase(); got != test.phase {
				t.Errorf("ToExecutionPhase() = %q, want %q", got, test.phase)
			}
			if got := test.stored.ToV2().ToExecutionPhase(); got != test.phase {
				t.Errorf("normalized execution phase = %q, want %q", got, test.phase)
			}
			run := (&Run{RunDetails: RunDetails{State: RuntimeStateUnspecified, Conditions: string(test.stored)}}).ToV2()
			if run.State != test.v2 {
				t.Errorf("historical Conditions normalized to %q, want %q", run.State, test.v2)
			}
		})
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
