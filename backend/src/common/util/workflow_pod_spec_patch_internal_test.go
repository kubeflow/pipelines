// Copyright 2026 The Kubeflow Authors
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
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestServiceAccountsInPodSpecPatchRejectsNoncanonicalFields(t *testing.T) {
	tests := []struct {
		name, patch string
	}{
		{name: "lowercase JSON", patch: `{"serviceaccountname":"privileged-sa"}`},
		{name: "mixed case YAML", patch: `serviceAccountname: privileged-sa`},
		{name: "uppercase with replacement", patch: "$patch: replace\nServiceAccountName: privileged-sa"},
		{name: "conflicting spellings", patch: `{"serviceAccountName":"allowed-sa","serviceaccountname":"privileged-sa"}`},
		{name: "Unicode case folding", patch: `{"ſerviceAccountName":"privileged-sa"}`},
		{name: "deprecated alias", patch: `{"ServiceAccount":"privileged-sa"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := serviceAccountsInPodSpecPatch(tt.patch, false)
			require.ErrorContains(t, err, "canonical")
		})
	}
}

func TestWorkflow_ServiceAccountsRejectsAccountHiddenUntilTemplatePatch(t *testing.T) {
	workflowPatch := `{"ServiceAccountName":"privileged-sa"}`
	templatePatch := `serviceAccountName: null`
	// Argo merges both patches before decoding: deleting the canonical field
	// exposes ServiceAccountName, which its decoder accepts as privileged-sa.
	w := NewWorkflow(&workflowapi.Workflow{Spec: workflowapi.WorkflowSpec{
		ServiceAccountName: "allowed-sa",
		PodSpecPatch:       workflowPatch,
		Templates:          []workflowapi.Template{{PodSpecPatch: templatePatch}},
	}})
	_, err := w.ServiceAccounts(false)
	require.ErrorContains(t, err, "canonical")
}

func TestServiceAccountsInPodSpecPatchPreservesArgoJSONValidation(t *testing.T) {
	// Argo validates the original JSON before merging, including overwritten fields.
	patch := `{"serviceAccountName":[],"serviceAccountName":"allowed-sa"}`
	_, err := serviceAccountsInPodSpecPatch(patch, false)
	require.ErrorContains(t, err, "valid Kubernetes PodSpec patch")
}
