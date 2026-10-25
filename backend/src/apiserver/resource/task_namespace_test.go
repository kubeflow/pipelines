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

package resource

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestTaskNamespaceUsesRunOwner(t *testing.T) {
	for _, tc := range []struct {
		name, runNS, experimentNS, requested, pipeline, want   string
		reject, missingExperiment, legacyRef, imported, single bool
	}{
		{name: "omitted", runNS: "team-a", experimentNS: "team-b", want: "team-a"},
		{name: "matching", runNS: "team-a", experimentNS: "team-b", requested: "team-a", want: "team-a"},
		{name: "mismatch", runNS: "team-a", experimentNS: "team-b", requested: "team-b", reject: true},
		{name: "pipeline mismatch", runNS: "team-a", experimentNS: "team-a", pipeline: "namespace/team-b/pipeline/test", reject: true},
		{name: "no experiment", runNS: "team-a", missingExperiment: true, want: "team-a"},
		{name: "legacy empty", experimentNS: "team-a", want: "team-a"},
		{name: "legacy sentinel", runNS: "-", experimentNS: "team-a", want: "team-a"},
		{name: "legacy reference", legacyRef: true, want: "team-a"},
		{name: "unresolved multi-user", reject: true},
		{name: "single-user", single: true, want: ""},
		{name: "single-user supplied", single: true, requested: "legacy", want: "legacy"},
		{name: "imported", runNS: "team-a", experimentNS: "team-a", imported: true, reject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, manager, run := initWithOneTimeRun(t)
			defer store.Close()
			_, err := store.DB().Exec(`UPDATE run_details SET Namespace = ? WHERE UUID = ?`, tc.runNS, run.UUID)
			require.NoError(t, err)
			// ToV2 resolves legacy run references; remove the fixture's ns1
			// reference so this case exercises the stored namespace column.
			_, err = store.DB().Exec(`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`, run.UUID, model.RunResourceType, model.NamespaceResourceType)
			require.NoError(t, err)
			_, err = store.DB().Exec(`UPDATE experiments SET Namespace = ? WHERE UUID = ?`, tc.experimentNS, run.ExperimentId)
			require.NoError(t, err)
			_, err = store.DB().Exec(`DELETE FROM resource_references WHERE ResourceUUID = ? AND ResourceType = ? AND ReferenceType = ?`, run.ExperimentId, model.ExperimentResourceType, model.NamespaceResourceType)
			require.NoError(t, err)
			if tc.legacyRef {
				ref := model.ResourceReference{ResourceUUID: run.ExperimentId, ResourceType: model.ExperimentResourceType, ReferenceType: model.NamespaceResourceType, ReferenceUUID: "team-a", Relationship: model.OwnerRelationship}
				payload, err := json.Marshal(ref)
				require.NoError(t, err)
				_, err = store.DB().Exec(`INSERT INTO resource_references (ResourceUUID,ResourceType,ReferenceUUID,ReferenceName,ReferenceType,Relationship,Payload) VALUES (?,?,?,?,?,?,?)`, ref.ResourceUUID, ref.ResourceType, ref.ReferenceUUID, "", ref.ReferenceType, ref.Relationship, string(payload))
				require.NoError(t, err)
			}
			if tc.missingExperiment {
				_, err = store.DB().Exec(`UPDATE run_details SET ExperimentUUID = '' WHERE UUID = ?`, run.UUID)
				require.NoError(t, err)
			}
			if tc.imported {
				_, err = store.DB().Exec(`UPDATE run_details SET ImportedFrom = 'source' WHERE UUID = ?`, run.UUID)
				require.NoError(t, err)
			}
			viper.Set(common.MultiUserMode, !tc.single)
			t.Cleanup(func() { viper.Set(common.MultiUserMode, false) })
			pipeline := tc.pipeline
			if pipeline == "" {
				pipeline = "pipeline/test"
			}
			created, err := manager.CreateTask(&model.Task{RunID: run.UUID, Namespace: tc.requested, PipelineName: pipeline, MLMDExecutionID: "42", Fingerprint: "fp"})
			if tc.reject {
				require.Error(t, err)
				require.Nil(t, created)
				_, getErr := manager.taskStore.GetTask(DefaultFakeUUID)
				require.Error(t, getErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, created.Namespace)
			}
		})
	}
}
